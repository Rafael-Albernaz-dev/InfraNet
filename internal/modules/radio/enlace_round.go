package radio

import (
	"context"
	"infradesktop/internal/modules/equipment"
	"strings"
	"sync"
	"time"
)

type enlaceAPCache struct {
	seen  APSeen
	stamp string
	at    time.Time
}
type enlaceWork struct {
	session equipment.EnlaceSession
	seen    []APSeen
	link    *RadioLink
}

func enlaceKey(profile, concentrator, login string) string {
	return profile + "|" + concentrator + "|" + login
}

// RunEnlace is daemon-only; IPC cannot supply the active-company guard.
func (s *Service) RunEnlace(ctx context.Context, active func() (string, bool)) {
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			profile, ok := active()
			if !ok {
				continue
			}
			guard := func() bool { current, ok := active(); return ok && current == profile && ctx.Err() == nil }
			s.EnlaceRound(ctx, profile, guard)
		}
	}
}
func (s *Service) EnlaceRound(ctx context.Context, profile string, active func() bool) {
	store, ok := s.Store.(equipment.EnlaceStore)
	if !ok || !active() {
		return
	}
	bindings, code := store.InterfaceAPs(ctx, profile)
	if code != OK {
		return
	}
	sessions, code := store.EnlaceSessions(ctx, profile)
	if code != OK {
		return
	}
	s.pruneEnlaceViews(profile, sessions)
	if code = s.FollowEnlaceSessions(ctx, profile, sessions); code != OK {
		return
	}
	if code = s.resolveEnlaceSessions(ctx, profile, sessions); code != OK {
		return
	}
	work := s.enlaceWork(ctx, profile, sessions, bindings, active)
	var mu sync.Mutex
	var workers sync.WaitGroup
	for i := 0; i < 4; i++ {
		workers.Add(1)
		go func() {
			defer workers.Done()
			for active() {
				mu.Lock()
				next, ok := s.nextEnlaceWork(profile, &work)
				mu.Unlock()
				if !ok {
					return
				}
				s.mountEnlace(ctx, profile, next, active)
			}
		}()
	}
	workers.Wait()
}
func (s *Service) nextEnlaceWork(profile string, work *[]enlaceWork) (enlaceWork, bool) {
	if len(*work) == 0 {
		return enlaceWork{}, false
	}
	s.enlaceMu.Lock()
	priority := s.enlacePriority
	s.enlaceMu.Unlock()
	index := 0
	for i, w := range *work {
		if enlaceKey(profile, w.session.Concentrator, w.session.Row.Login) == priority {
			index = i
			break
		}
	}
	next := (*work)[index]
	*work = append((*work)[:index], (*work)[index+1:]...)
	return next, true
}
func (s *Service) enlaceWork(ctx context.Context, profile string, sessions []equipment.EnlaceSession,
	bindings []equipment.InterfaceAPs, active func() bool) []enlaceWork {
	by := map[string][]string{}
	for _, b := range bindings {
		by[b.Concentrator+"|"+b.Interface] = b.APs
	}
	links, code := s.enlaceLinks(ctx, profile, sessions)
	if code != OK {
		return nil
	}
	state, code := s.Store.ReadEquipment(ctx, profile)
	if code != OK {
		return nil
	}
	devices := map[string]Device{}
	for _, d := range state.Devices {
		devices[d.ID] = d
	}
	revisions := map[string]uint64{}
	for _, cr := range state.Credentials {
		revisions[cr.Metadata.ID] = cr.Metadata.Revision
	}
	cache := map[string]APSeen{}
	work := []enlaceWork{}
	for _, session := range sessions {
		if session.Row.State != "ONLINE" || session.Row.Duplicates > 1 {
			continue
		}
		aps := by[session.Concentrator+"|"+enlaceInterface(session.Row)]
		link, exists := links[enlaceKey(profile, session.Concentrator, session.Row.Login)]
		if exists && currentEnlaceAssociation(link, session.Row, aps) {
			if s.enlaceRondaDeferred(profile, session, link, devices[link.CPE], revisions) {
				continue
			}
			saved := link
			work = append(work, enlaceWork{session: session, link: &saved})
			continue
		}
		if len(aps) == 0 {
			continue
		}
		w, ready := s.prepareEnlaceWork(ctx, profile, session, aps, cache, active)
		if !ready {
			return nil
		}

		work = append(work, w)
	}
	return work
}

func (s *Service) enlaceRondaDeferred(profile string, session equipment.EnlaceSession, link RadioLink,
	d Device, revisions map[string]uint64,
) bool {
	last := time.Time{}
	if d.LastAttempt != nil {
		// ignored: an invalid date is due immediately.
		last, _ = time.Parse(time.RFC3339Nano, d.LastAttempt.At)
	}
	if d.CPELogin != nil && enlaceContactDue(d, revisions[d.CredentialID], last, s.Now()) {
		return false
	}
	view := equipment.EnlaceView{State: "ready", Stage: 5, MAC: session.Row.Caller,
		Interface: enlaceInterface(session.Row), Login: session.Row.Login,
		Address: session.Row.Address, Link: &link, APs: []string{link.AP}}
	if d.LastAttempt != nil && d.LastAttempt.Code != OK {
		view.State, view.Reason = "failed", string(d.LastAttempt.Code)
	}
	s.setEnlaceView(enlaceKey(profile, session.Concentrator, session.Row.Login), view)
	return true
}

func currentEnlaceAssociation(link RadioLink, row PPRow, aps []string) bool {
	if link.Origin != "enlace" {
		return true
	}
	mac, valid := RadioMAC(row.Caller)
	if !valid || mac != link.MAC {
		return false
	}
	for _, ap := range aps {
		if ap == link.AP {
			return true
		}
	}
	return false
}
func enlaceChoiceKey(key, caller string) string {
	mac, valid := RadioMAC(caller)
	if !valid {
		return key + "|invalid"
	}
	return key + "|" + mac
}

func (s *Service) pruneEnlaceViews(profile string, sessions []equipment.EnlaceSession) {
	present := map[string]bool{}
	for _, session := range sessions {
		present[enlaceKey(profile, session.Concentrator, session.Row.Login)] = true
	}
	s.enlaceMu.Lock()
	defer s.enlaceMu.Unlock()
	for key := range s.enlaceViews {
		if !present[key] {
			delete(s.enlaceViews, key)
		}
	}
	for key := range s.enlaceCache {
		if !strings.HasPrefix(key, profile+"|") {
			delete(s.enlaceCache, key)
		}
	}
}

func (s *Service) prepareEnlaceWork(ctx context.Context, profile string, session equipment.EnlaceSession,
	aps []string, cache map[string]APSeen, active func() bool,
) (enlaceWork, bool) {
	w := enlaceWork{session: session}
	s.setEnlaceView(enlaceKey(profile, session.Concentrator, session.Row.Login), equipment.EnlaceView{
		State: "mounting", Stage: 0, MAC: session.Row.Caller,
		Interface: enlaceInterface(session.Row), Login: session.Row.Login, Address: session.Row.Address, APs: aps})
	for _, ap := range aps {
		if !active() {
			return enlaceWork{}, false
		}
		// ignored: only membership is used.
		if _, exists := cache[ap]; !exists {
			reading := func() {
				s.setEnlaceView(enlaceKey(profile, session.Concentrator, session.Row.Login), equipment.EnlaceView{
					State: "mounting", Stage: 1, MAC: session.Row.Caller, Interface: enlaceInterface(session.Row),
					Login: session.Row.Login, Address: session.Row.Address, APs: aps})
			}
			cache[ap] = s.enlaceAP(ctx, profile, ap, active, reading)
		}
		w.seen = append(w.seen, cache[ap])
	}
	return w, true
}
