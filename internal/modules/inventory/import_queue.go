package inventory

import (
	"context"
	"infradesktop/internal/modules/equipment"
	"sync"
)

type importQueue struct {
	command   Command
	stamp     string
	items     []equipment.ImportItem
	cancelled bool
	done      bool
	created   int
	failure   Code
	cancel    context.CancelFunc
}

func (s *Service) startImportQueue(ctx context.Context, q Command, v *View) Code {
	if !validQueueInput(q) || s.Digester == nil {
		return Invalid
	}
	input := *q.Import
	input.Rows = append([]equipment.ImportRow(nil), input.Rows...)
	for i := range input.Rows {
		input.Rows[i].Name = ""
	}
	q.Import = &input
	stamp, code := importStamp(s.Digester, q.Import.Rows)
	if code != OK {
		return code
	}
	state, code := s.Store.ReadEquipment(ctx, q.Scope.Profile)
	if code != OK {
		return code
	}
	if replayed, code := replayImport(&state, q.Operation, stamp, v); replayed {
		return code
	}
	items, code := readyImportItems(&state, q.Import.Rows)
	if code != OK {
		return code
	}
	s.Mu.Lock()
	defer s.Mu.Unlock()
	if s.Closed {
		return Unavailable
	}
	if old := s.Imports[q.Scope.Profile]; old != nil && !old.done {
		if old.command.Operation != q.Operation || old.stamp != stamp || !SameScope(old.command, q) {
			return equipment.Code("BUSY")
		}
		s.importViewLocked(old, v)
		return OK
	}
	runctx, cancel := context.WithCancel(context.Background())
	queue := &importQueue{command: q, stamp: stamp, items: items, cancel: cancel}
	s.Imports[q.Scope.Profile] = queue
	s.WG.Add(1)
	go s.runImportQueue(runctx, queue)
	s.importViewLocked(queue, v)
	return OK
}

func (s *Service) importViewLocked(queue *importQueue, v *View) {
	v.ImportItems = append([]equipment.ImportItem(nil), queue.items...)
	v.OperationState = "RUNNING"
	if queue.done {
		v.OperationState = "COMPLETED"
	}
	if queue.failure != "" {
		v.OperationState = "FAILED"
		v.ContactCode = queue.failure
	}
	receipt := &equipment.ImportReceipt{Created: queue.created}
	for _, item := range queue.items {
		if item.Status == importIgnored || item.Status == "failed" || item.Status == "cancelled" {
			receipt.Ignored++
		}
	}
	if queue.done {
		v.ImportReceipt = receipt
	}
}

func (s *Service) importQueueStatus(_ context.Context, q Command, v *View) Code {
	s.Mu.Lock()
	defer s.Mu.Unlock()
	queue := s.Imports[q.Scope.Profile]
	if queue == nil {
		return OK
	}
	if !SameScope(queue.command, q) {
		return Changed
	}
	if q.Op == "inventory.import-cancel" {
		queue.cancelled = true
	}
	s.importViewLocked(queue, v)
	return OK
}

func (s *Service) nextImportLine(queue *importQueue) (int, equipment.ImportItem) {
	s.Mu.Lock()
	defer s.Mu.Unlock()
	for i := range queue.items {
		if queue.items[i].Status != "queued" {
			continue
		}
		if queue.cancelled {
			queue.items[i].Status, queue.items[i].Reason = "cancelled", "Cancelada"
			continue
		}
		queue.items[i].Status = "connecting"
		return i, queue.items[i]
	}
	return -1, equipment.ImportItem{}
}

func (s *Service) runImportQueue(ctx context.Context, queue *importQueue) {
	defer s.WG.Done()
	defer queue.cancel()
	var workers sync.WaitGroup
	for n := 0; n < 2; n++ {
		workers.Add(1)
		go func() {
			defer workers.Done()
			for {
				index, item := s.nextImportLine(queue)
				if index < 0 {
					return
				}
				identity, code := s.connectImportLine(ctx, queue.command, item)
				s.Mu.Lock()
				if code == OK {
					queue.items[index].Status = "registered"
					queue.created++
				} else {
					queue.items[index].Status, queue.items[index].Reason = "failed", string(code)
				}
				queue.items[index].Identity = identity
				s.Mu.Unlock()
			}
		}()
	}
	workers.Wait()
	s.Mu.Lock()
	queue.done = true
	view := View{}
	s.importViewLocked(queue, &view)
	queue.done = false
	s.Mu.Unlock()
	// The final receipt is durable; completed rows were committed independently.
	code := s.Authorized(ctx, queue.command, func(uint64) Code {
		return s.Store.UpdateEquipment(ctx, queue.command.Scope.Profile, func(state *State) Code {
			if len(state.Operations) >= MaxOperations {
				return Limit
			}
			state.Operations = append(state.Operations, Operation{ID: queue.command.Operation,
				Stamp: queue.stamp, ImportReceipt: view.ImportReceipt})
			return OK
		})
	})
	s.Mu.Lock()
	queue.done = true
	if code != OK {
		queue.failure = code
	}
	s.Mu.Unlock()
}

func readyImportItems(state *State, rows []equipment.ImportRow) ([]equipment.ImportItem, Code) {
	items := reviewImport(state, rows)
	ready := 0
	for i := range items {
		if items[i].Status == importReady {
			items[i].Status = "queued"
			ready++
		} else if items[i].Status != importIgnored {
			return nil, Revision
		}
	}
	if ready == 0 {
		return nil, Invalid
	}
	return items, OK
}

func validQueueInput(q Command) bool {
	if q.Import == nil || !ValidID(q.Operation) {
		return false
	}
	input := q.Import
	return len(input.Rows) > 0 && len(input.Rows) <= MaxDevices && len(input.Data) == 0 && input.Filename == "" &&
		input.DefaultGroup == "" && input.DefaultBlock == "" && input.DefaultCredential == ""
}
