package radio

import (
	"context"
	"infradesktop/internal/modules/access"
	"infradesktop/internal/modules/equipment"
	"strings"
)

func (s *Service) handleEnlace(ctx context.Context, q Command, in RadioInput, v *RadioView) Code {
	store, ok := s.Store.(equipment.EnlaceStore)
	if !ok {
		return Unavailable
	}
	if q.Op == "radio.interface-aps" {
		var code Code
		v.InterfaceAPs, code = store.InterfaceAPs(ctx, q.Scope.Profile)
		return code
	}
	if q.Op == "radio.set-interface-aps" {
		return s.setInterfaceAPs(ctx, q, in, v)
	}
	row, code := s.enlaceSessionRow(ctx, q, in)
	if code != OK {
		return code
	}
	key := enlaceKey(q.Scope.Profile, in.DeviceID, row.Login)
	s.enlaceMu.Lock()
	saved := s.enlaceViews[key]
	s.enlacePriority = key
	if q.Op == "radio.enlace-confirm" {
		if !validEnlaceConfirmation(row, saved, in.ConfirmAP) {
			s.enlaceMu.Unlock()
			return RadioCodeLinkUnconfirmed
		}
		s.enlaceChoice[enlaceChoiceKey(key, row.Caller)] = in.ConfirmAP
	}
	s.enlaceMu.Unlock()
	bindings, code := store.InterfaceAPs(ctx, q.Scope.Profile)
	if code != OK {
		return code
	}
	view := currentEnlaceView(row, in.DeviceID, bindings, saved)
	s.nameEnlaceChoices(ctx, q.Scope.Profile, &view)
	v.Enlace, v.InterfaceAPs, v.Link = &view, bindings, view.Link
	return OK
}
func (s *Service) setInterfaceAPs(ctx context.Context, q Command, in RadioInput, v *RadioView) Code {
	if !access.ValidCompanyID(in.DeviceID) || !ValidText(in.Interface, 80) || len(in.APs) > MaxDevices {
		return Invalid
	}
	if code := s.validEnlaceInterface(ctx, q.Scope.Profile, in); code != OK {
		return code
	}
	store, ok := s.Store.(equipment.EnlaceStore)
	if !ok {
		return Unavailable
	}
	if code := store.SetInterfaceAPs(ctx, q.Scope.Profile, equipment.InterfaceAPs{
		Concentrator: in.DeviceID, Interface: in.Interface, APs: in.APs}); code != OK {
		return code
	}
	s.enlaceMu.Lock()
	s.enlaceViews = map[string]equipment.EnlaceView{}
	s.enlaceChoice = map[string]string{}
	s.enlaceMu.Unlock()
	var code Code
	v.InterfaceAPs, code = store.InterfaceAPs(ctx, q.Scope.Profile)
	return code
}
func (s *Service) enlaceSessionRow(ctx context.Context, q Command, in RadioInput) (PPRow, Code) {
	if !validRadioSourceIDs(in) {
		return PPRow{}, Invalid
	}
	pp, ok := s.Store.(PPStore)
	if !ok {
		return PPRow{}, Unavailable
	}
	current, code := pp.PPCurrent(ctx, q.Scope.Profile, in.DeviceID)
	if code != OK {
		return PPRow{}, code
	}
	if current.Current == nil || current.Current.ID != in.ObservationID {
		return PPRow{}, ObservationMissing
	}
	detail, code := pp.PPSession(ctx, q.Scope.Profile, PPInput{DeviceID: in.DeviceID, ObservationID: in.ObservationID, RowID: in.RowID})
	if code != OK {
		return PPRow{}, code
	}
	if len(detail.Rows) != 1 {
		return PPRow{}, ObservationMissing
	}
	sessions := []equipment.EnlaceSession{{Concentrator: in.DeviceID, Row: detail.Rows[0]}}
	if code := s.resolveEnlaceSessions(ctx, q.Scope.Profile, sessions); code != OK {
		return PPRow{}, code
	}
	return sessions[0].Row, OK
}
func validEnlaceChoice(view equipment.EnlaceView, ap string) bool {
	if view.State != "two" {
		return false
	}
	for _, choice := range view.Choices {
		if choice.AP == ap {
			return true
		}
	}
	return false
}
func currentEnlaceView(row PPRow, concentrator string, bindings []equipment.InterfaceAPs, saved equipment.EnlaceView) equipment.EnlaceView {
	view := equipment.EnlaceView{State: "unlinked", Interface: enlaceInterface(row), Login: row.Login, Address: row.Address}
	for _, b := range bindings {
		if b.Concentrator == concentrator && b.Interface == view.Interface {
			view.APs = b.APs
		}
	}
	if len(view.APs) > 0 {
		view.State = "mounting"
	}
	if saved.Login == row.Login && saved.Address == row.Address && saved.Interface == view.Interface &&
		strings.EqualFold(saved.MAC, row.Caller) {
		view = saved
	}
	if row.State == "OFFLINE" {
		view.State = "offline"
	}
	return view
}
func (s *Service) nameEnlaceChoices(ctx context.Context, profile string, v *equipment.EnlaceView) {
	state, code := s.Store.ReadEquipment(ctx, profile)
	if code != OK {
		return
	}
	v.Choices = append([]equipment.EnlaceChoice{}, v.Choices...)
	for i := range v.Choices {
		if d := device(&state, v.Choices[i].AP); d != nil {
			v.Choices[i].Name = d.Name
		}
	}
}

func validEnlaceConfirmation(row PPRow, view equipment.EnlaceView, ap string) bool {
	return validEnlaceChoice(view, ap) && strings.EqualFold(view.MAC, row.Caller) && view.Address == row.Address &&
		view.Interface == enlaceInterface(row)
}
