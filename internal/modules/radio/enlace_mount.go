package radio

import (
	"context"
	"infradesktop/internal/modules/equipment"
	"infradesktop/internal/modules/inventory"
	"time"
)

func (s *Service) mountEnlace(ctx context.Context, profile string, work enlaceWork, active func() bool) {
	session, seen := work.session, work.seen
	row := session.Row
	v := equipment.EnlaceView{State: "mounting", Stage: 2, MAC: row.Caller,
		Interface: enlaceInterface(row), Login: row.Login, Address: row.Address}
	for _, ap := range seen {
		v.APs = append(v.APs, ap.AP)
	}

	key := enlaceKey(profile, session.Concentrator, row.Login)
	link := work.link
	if !active() {
		return
	}
	if link == nil {
		var code Code
		link, code = s.mountNewEnlace(ctx, profile, work, &v)
		if code != OK {
			s.setEnlaceView(key, v)
			return
		}
	}
	v.Link, v.Stage = link, 3
	s.setEnlaceView(key, v)
	s.readEnlaceRonda(ctx, profile, work, &v, active)
	if work.link == nil && v.State == "ready" {
		initial := work
		initial.link = link
		if code := s.readInitialEnlace(ctx, profile, initial, active); code != OK {
			v.State, v.Reason = "failed", string(code)
		}
	}
	s.setEnlaceView(key, v)
}

func chosenEnlaceAP(result EnlaceResult, selected string) (EnlaceChoice, bool) {
	if result.State == EnlaceFound {
		return result.Choices[0], true
	}
	if result.State == EnlaceTwo && selected != "" {
		for _, choice := range result.Choices {
			if choice.AP == selected {
				return choice, true
			}
		}
	}
	return EnlaceChoice{}, false
}
func (s *Service) setEnlaceView(key string, v equipment.EnlaceView) {
	s.enlaceMu.Lock()
	defer s.enlaceMu.Unlock()
	s.enlaceViews[key] = v
}
func (s *Service) enlaceSessionOnline(ctx context.Context, profile string, session equipment.EnlaceSession, d Device) bool {
	store, ok := s.Store.(PPStore)
	if !ok {
		return false
	}
	current, code := store.PPCurrent(ctx, profile, session.Concentrator)
	if code != OK || current.Current == nil {
		return false
	}
	detail, code := store.PPSession(ctx, profile, PPInput{
		DeviceID: session.Concentrator, ObservationID: current.Current.ID, RowID: session.Row.ID})
	return code == OK && len(detail.Rows) == 1 && detail.Rows[0].Login == session.Row.Login &&
		detail.Rows[0].State == "ONLINE" && detail.Rows[0].Address == d.Address
}

func (s *Service) mountNewEnlace(ctx context.Context, profile string, work enlaceWork, v *equipment.EnlaceView) (*RadioLink, Code) {
	session, seen := work.session, work.seen
	row := session.Row
	key := enlaceKey(profile, session.Concentrator, row.Login)
	match := EnlaceMatch(row.Caller, seen)
	v.Failed = match.Failed
	s.enlaceMu.Lock()
	selected := s.enlaceChoice[enlaceChoiceKey(key, row.Caller)]
	s.enlaceMu.Unlock()
	for _, choice := range match.Choices {
		v.Choices = append(v.Choices, equipment.EnlaceChoice{AP: choice.AP, Name: choice.AP, Signal: choice.Signal})
	}
	choice, valid := chosenEnlaceAP(match, selected)
	if !valid {
		v.State = string(match.State)
		return nil, RadioCodeLinkUnconfirmed
	}
	link, code := s.createEnlaceCPE(ctx, profile, session, choice)
	if code != OK {
		v.State, v.Reason = "failed", string(code)
	}
	return link, code
}
func (s *Service) readEnlaceRonda(ctx context.Context, profile string, work enlaceWork, v *equipment.EnlaceView, active func() bool) {
	session, link := work.session, v.Link
	key := enlaceKey(profile, session.Concentrator, session.Row.Login)
	state, code := s.Store.ReadEquipment(ctx, profile)
	if code != OK {
		return
	}
	d := device(&state, link.CPE)
	if d == nil {
		return
	}
	cr := credential(&state, d.CredentialID)
	if cr == nil {
		v.State = "failed"
		v.Reason = string(MissingCredential)
		s.setEnlaceView(key, *v)
		return
	}
	last := time.Time{}
	if d.LastAttempt != nil {
		// ignored: an invalid timestamp makes the CPE due immediately.
		last, _ = time.Parse(time.RFC3339Nano, d.LastAttempt.At)
	}
	if enlaceContactDue(*d, cr.Metadata.Revision, last, s.Now()) {
		guard := func() bool { return active() && s.enlaceSessionOnline(ctx, profile, session, *d) }
		v.Stage = 4
		s.setEnlaceView(key, *v)
		code = s.BasicContact(ctx, profile, *d, guard)
	}
	if code != OK {
		v.State = "failed"
		v.Reason = string(code)
	} else {
		v.State, v.Stage = "ready", 5
		state, code = s.Store.ReadEquipment(ctx, profile)
		if code == OK {
			if current := device(&state, link.CPE); current != nil && current.LastAttempt != nil && current.LastAttempt.Code != OK {
				v.State, v.Reason = "failed", string(current.LastAttempt.Code)
			}
		}
	}
}

func enlaceContactDue(d Device, credential uint64, last, now time.Time) bool {
	if !inventory.BasicContactDue(d, credential, now) {
		return false
	}
	if d.LastAttempt == nil || d.LastAttempt.CredentialRevision != credential || d.LastAttempt.DeviceRevision != d.Revision {
		return true
	}
	return RondaDue(d.ID, last, now)
}
