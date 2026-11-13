package presentation

import (
	"infradesktop/internal/adapters/ipc"
	"infradesktop/internal/modules/equipment"
)

func (a *App) GetTablePreference(profile, activation, table string) EquipmentResult {
	return a.equipmentCall("inventory.table-preference-get", profile, activation, ipc.EquipmentInput{Table: table})
}
func (a *App) SetTablePreference(profile, activation, table string, p equipment.TablePreference) EquipmentResult {
	return a.equipmentCall("inventory.table-preference-set", profile, activation, ipc.EquipmentInput{Table: table, TablePreference: &p})
}
