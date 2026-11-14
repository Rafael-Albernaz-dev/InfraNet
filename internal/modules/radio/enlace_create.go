package radio

import (
	"context"
	"infradesktop/internal/modules/equipment"
)

func (s *Service) createEnlaceCPE(ctx context.Context, profile string,
	session equipment.EnlaceSession, choice EnlaceChoice,
) (*RadioLink, Code) {
	state, code := s.Store.ReadEquipment(ctx, profile)
	if code != OK {
		return nil, code
	}
	row := session.Row
	link := &RadioLink{Concentrator: session.Concentrator, Scope: PPScope, Login: row.Login, AP: choice.AP, PPPInterface: enlaceInterface(row),
		MAC: choice.MAC, Interface: choice.Interface, BSSID: choice.BSSID, Origin: "enlace", At: PPTime(s.Now()),
		Evidence: "MAC nos APs ligados à interface PPPoE"}
	var valid bool
	link.BSSID, valid = RadioMAC(link.BSSID)
	if !valid || link.Interface == "" {
		return nil, RadioCodeLinkUnconfirmed
	}
	link.CPE, code = adoptEnlaceCPE(state.Devices, row)
	if code != OK {
		return nil, code
	}
	operation, code := s.IDs.ID()
	if code != OK {
		return nil, code
	}
	if link.CPE == "" {
		if len(state.Devices) >= MaxDevices {
			return nil, Limit
		}
		cr := state.CredentialDefaults[string(RoleCPE)]
		if credential(&state, cr) == nil {
			return nil, MissingCredential
		}
		link.CPE = operation
		link.NewCPE = &Device{ID: operation, Name: "CPE " + row.Login, Address: row.Address, Role: RoleCPE, Port: 22,
			Transport: TransportSSH, CredentialID: cr, Revision: 1, Active: true}
	}
	store, ok := s.Store.(equipment.RadioStore)
	if !ok {
		return nil, Unavailable
	}
	code = store.SaveRadioLink(ctx, profile, *link, operation, PPInput{
		DeviceID: session.Concentrator, ObservationID: session.Observation, RowID: row.ID})
	link.NewCPE = nil
	return link, code
}

func adoptEnlaceCPE(devices []Device, row PPRow) (string, Code) {
	id := ""
	for _, d := range devices {
		if !d.Active || d.Role != RoleCPE || d.Address != row.Address {
			continue
		}
		if id != "" || d.CPELogin != nil && d.CPELogin.Login != row.Login {
			return "", RadioCodeLinkUnconfirmed
		}
		id = d.ID
	}
	return id, OK
}
