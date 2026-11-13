package presentation

import "infradesktop/internal/adapters/ipc"

func (a *App) EquipmentGroups(profile, activation string) EquipmentResult {
	return a.equipmentCall("inventory.groups", profile, activation, ipc.EquipmentInput{})
}
func (a *App) CreateEquipmentGroup(profile, activation, operation, id, parent, name string) EquipmentResult {
	return a.equipmentCall("inventory.group-create", profile, activation,
		ipc.EquipmentInput{Operation: operation, ID: id, ParentID: parent, Name: name})
}
func (a *App) RenameEquipmentGroup(profile, activation, operation, id, name string) EquipmentResult {
	return a.equipmentCall("inventory.group-rename", profile, activation, ipc.EquipmentInput{Operation: operation, ID: id, Name: name})
}
func (a *App) DeleteEquipmentGroup(profile, activation, operation, id string) EquipmentResult {
	return a.equipmentCall("inventory.group-delete", profile, activation, ipc.EquipmentInput{Operation: operation, ID: id})
}
