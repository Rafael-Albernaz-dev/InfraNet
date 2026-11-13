package presentation

import "infradesktop/internal/modules/equipment"

func (a *App) ListPPP(p, act, device string, query equipment.PPPQuery, cursor string) PPPoEResult {
	return a.ppCall("pppoe.ppp-list", p, act, equipment.PPInput{DeviceID: device, PPPQuery: &query, Cursor: cursor})
}
func (a *App) GetPPPSummary(p, act, device string) PPPoEResult {
	return a.ppCall("pppoe.summary", p, act, equipment.PPInput{DeviceID: device})
}
