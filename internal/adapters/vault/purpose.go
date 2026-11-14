package vault

import (
	"encoding/hex"
	"infradesktop/internal/modules/equipment"
	"io"
)

type binding struct {
	installation, company, id, purpose string
	revision                           uint64
	maximum                            int
}

func (b binding) aad() []byte { return aad(b.purpose, b.installation, b.company, b.id, b.revision) }
func (v *Vault) sealPurpose(scope binding, secret string, wrapped []byte) ([]byte, []byte, equipment.Code) {
	master, err := v.master()
	if err != nil {
		return nil, nil, equipment.VaultUnavailable
	}
	defer clear(master)
	var key []byte
	var keyID string
	if len(wrapped) == 0 {
		key = make([]byte, 32)
		// ignored: io.ReadFull only requires its error here.
		if _, err = io.ReadFull(v.Random, key); err != nil {
			return nil, nil, equipment.VaultUnavailable
		}
		defer clear(key)
		idBytes := make([]byte, 16)
		// ignored: io.ReadFull only requires its error here.
		if _, err = io.ReadFull(v.Random, idBytes); err != nil {
			return nil, nil, equipment.VaultUnavailable
		}
		keyID = hex.EncodeToString(idBytes)
		wrapped, err = v.seal(master, key, aad("company-key", scope.installation, scope.company, keyID, 1), keyID)
	} else {
		key, keyID, err = v.dataKey(scope.installation, scope.company, wrapped, master)
		defer clear(key)
	}
	if err != nil {
		return nil, nil, equipment.VaultUnavailable
	}
	plain := []byte(secret)
	defer clear(plain)
	b, err := v.seal(key, plain, scope.aad(), keyID)
	if err != nil {
		return nil, nil, equipment.VaultUnavailable
	}
	return b, wrapped, equipment.OK
}
func (v *Vault) openPurpose(scope binding, b, wrapped []byte) (string, equipment.Code) {
	master, err := v.master()
	if err != nil {
		return "", equipment.VaultUnavailable
	}
	defer clear(master)
	key, keyID, err := v.dataKey(scope.installation, scope.company, wrapped, master)
	if err != nil {
		return "", equipment.VaultUnavailable
	}
	defer clear(key)
	e, err := decodeLimit(b, scope.maximum)
	if err != nil || e.KeyID != keyID {
		return "", equipment.VaultUnavailable
	}
	plain, err := open(key, e, scope.aad())
	if err != nil {
		return "", equipment.VaultUnavailable
	}
	defer clear(plain)
	return string(plain), equipment.OK
}
