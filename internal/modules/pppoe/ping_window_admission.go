package pppoe

import (
	"context"
	"infradesktop/internal/modules/equipment"
	"infradesktop/internal/modules/inventory"
	"time"
)

// An explicit terminal replaces only this caller's passive live reading of the
// same device. Waiting happens outside the account lock so publication can end.
func (s *Service) releaseLiveForPing(ctx context.Context, q Command, in PPInput) Code {
	ticker := time.NewTicker(10 * time.Millisecond)
	defer ticker.Stop()
	for {
		waiting, code := s.cancelOwnLiveForPing(ctx, q, in)
		if code != OK {
			return code
		}
		if !waiting {
			return OK
		}
		select {
		case <-ctx.Done():
			return Cancelled
		case <-ticker.C:
		}
	}
}
func (s *Service) cancelOwnLiveForPing(ctx context.Context, q Command, in PPInput) (bool, Code) {
	waiting := false
	code := s.inv.Authorized(ctx, q, func(uint64) Code {
		target, code := s.admitStartPPRead(ctx, q, in)
		if code != OK {
			return code
		}
		in.RowID = ""
		if len(in.RowIDs) == 1 {
			in.RowID = in.RowIDs[0]
		}
		// ignored: preflight needs validation only; admission builds the execution target again.
		if _, code = s.preparePPPing(ctx, q, in, target); code != OK {
			return code
		}
		s.inv.Mu.Lock()
		defer s.inv.Mu.Unlock()
		for _, j := range s.inv.Jobs {
			if ownLiveReading(j, q, in) {
				j.Cancel()
				waiting = true
			}
		}
		return OK
	})
	return waiting, code
}
func ownLiveReading(j *RunningJob, q Command, in PPInput) bool {
	return j.Command.Op == "pppoe.read-start" && j.PPInput.Scope == "live" &&
		j.Device.ID == in.DeviceID && inventory.SameScope(q, j.Command) && !j.Released
}
func anotherPingWindow(j *RunningJob, q Command, in PPInput) bool {
	if in.Scope != equipment.PingWindowScope || j.PPInput.Scope != equipment.PingWindowScope || j.Released {
		return false
	}
	return !inventory.SameScope(q, j.Command) || j.Command.Operation != in.Operation
}
