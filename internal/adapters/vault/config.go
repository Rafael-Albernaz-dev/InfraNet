package vault

import (
	"infradesktop/internal/modules/cfgbackup"
	"infradesktop/internal/modules/equipment"
)

func (v *Vault) SealConfig(installation, company, id, text string, wrapped []byte) ([]byte, equipment.Code) {
	if len(wrapped) == 0 || len(text) > cfgbackup.MaxBytes {
		return nil, equipment.VaultUnavailable
	}
	// ignored: only the result code or validation outcome is needed here.
	body, _, code := v.sealPurpose(binding{installation, company, id, "config-backup", 1, cfgbackup.MaxBytes}, text, wrapped)
	return body, code
}
func (v *Vault) OpenConfig(installation, company, id string, body, wrapped []byte) (string, equipment.Code) {
	return v.openPurpose(binding{installation, company, id, "config-backup", 1, cfgbackup.MaxBytes}, body, wrapped)
}
