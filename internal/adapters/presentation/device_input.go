package presentation

import (
	"infradesktop/internal/adapters/ipc"
	"infradesktop/internal/modules/equipment"
)

type DeviceInput struct {
	GroupID      string `json:"group_id"`
	User         string `json:"user"`
	Secret       string `json:"secret"`
	ID           string `json:"id"`
	Operation    string `json:"operation_id"`
	Revision     uint64 `json:"revision"`
	Role         string `json:"role,omitempty"`
	Name         string `json:"name"`
	Address      string `json:"address"`
	Port         int    `json:"port"`
	Transport    string `json:"transport"`
	CredentialID string `json:"credential_id"`
	Block        string `json:"block"`
}

func makeDeviceInput(d DeviceInput) ipc.EquipmentInput {
	return ipc.EquipmentInput{ID: d.ID, Operation: d.Operation, Revision: d.Revision, Role: equipment.Role(d.Role),
		Name: d.Name, GroupID: d.GroupID, Block: d.Block, Address: d.Address, Port: d.Port, Transport: equipment.Transport(d.Transport),
		CredentialID: d.CredentialID, User: d.User, Secret: d.Secret}
}

func deviceOperation(d DeviceInput) string {
	if d.Revision != 0 {
		return "inventory.update"
	}

	return "inventory.register"
}
