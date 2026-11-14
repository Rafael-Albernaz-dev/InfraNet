package presentation

import "infradesktop/internal/modules/equipment"

func (a *App) GetInterfaceAPs(p, act string) RadioResult {
	return a.radioCall("radio.interface-aps", p, act, equipment.RadioInput{})
}
func (a *App) SetInterfaceAPs(p, act string, in equipment.RadioInput) RadioResult {
	return a.radioCall("radio.set-interface-aps", p, act, in)
}
func (a *App) GetEnlace(p, act string, in equipment.RadioInput) RadioResult {
	return a.radioCall("radio.enlace", p, act, in)
}
func (a *App) ConfirmEnlaceAP(p, act string, in equipment.RadioInput) RadioResult {
	return a.radioCall("radio.enlace-confirm", p, act, in)
}
