// Package vault implements authenticated envelope encryption. It never logs keys.
package vault

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"infradesktop/internal/modules/equipment"
	"io"
)

type Provider func() ([]byte, error)
type Vault struct {
	Key    Provider
	Random io.Reader
}
type envelope struct {
	Version    int    `json:"version"`
	Algorithm  string `json:"algorithm"`
	KeyID      string `json:"key_id"`
	Nonce      []byte `json:"nonce"`
	Ciphertext []byte `json:"ciphertext"`
}

func New(key Provider) *Vault { return &Vault{Key: key, Random: rand.Reader} }
func (v *Vault) master() ([]byte, error) {
	if v.Key == nil {
		return nil, errors.New("key unavailable")
	}
	b, e := v.Key()
	if e != nil || len(b) != 32 {
		return nil, errors.New("key unavailable")
	}
	return b, nil
}
func (v *Vault) Available() bool { b, e := v.master(); clear(b); return e == nil }
func (v *Vault) Ready(installation, company string, wrapped []byte) bool {
	if len(wrapped) == 0 {
		return v.Available()
	}
	master, e := v.master()
	if e != nil {
		return false
	}
	defer clear(master)
	key, _, e := v.dataKey(installation, company, wrapped, master)
	clear(key)
	return e == nil
}
func aad(purpose, installation, company, id string, revision uint64) []byte {
	b, _ := json.Marshal([]string{"infradesktop-vault", "1", purpose, installation, company, id, fmt.Sprint(revision)})
	return b
}
func (v *Vault) seal(key, plain, aad []byte, id string) ([]byte, error) {
	block, e := aes.NewCipher(key)
	if e != nil {
		return nil, e
	}
	g, e := cipher.NewGCM(block)
	if e != nil {
		return nil, e
	}
	nonce := make([]byte, g.NonceSize())
	if _, e = io.ReadFull(v.Random, nonce); e != nil {
		return nil, e
	}
	return json.Marshal(envelope{1, "AES-256-GCM", id, nonce, g.Seal(nil, nonce, plain, aad)})
}
func open(key []byte, e envelope, aad []byte) ([]byte, error) {
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}
	g, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}
	return g.Open(nil, e.Nonce, e.Ciphertext, aad)
}
func (v *Vault) dataKey(installation, company string, wrapped, master []byte) ([]byte, string, error) {
	e, err := decode(wrapped)
	if err != nil {
		return nil, "", err
	}
	b, err := open(master, e, aad("company-key", installation, company, e.KeyID, 1))
	if err != nil || len(b) != 32 {
		clear(b)
		return nil, "", errors.New("key unavailable")
	}
	return b, e.KeyID, nil
}
func (v *Vault) Seal(installation, company, id string, revision uint64, secret string, wrapped []byte) ([]byte, []byte, equipment.Code) {
	return v.sealPurpose(binding{installation, company, id, "read-credential", revision, 1024}, secret, wrapped)
}
func (v *Vault) Open(installation, company, id string, revision uint64, b, wrapped []byte) (string, equipment.Code) {
	return v.openPurpose(binding{installation, company, id, "read-credential", revision, 1024}, b, wrapped)
}
func (v *Vault) Stamp(input string) (string, equipment.Code) {
	key, e := v.master()
	if e != nil {
		return "", equipment.VaultUnavailable
	}
	defer clear(key)
	h := hmac.New(sha256.New, key)
	h.Write([]byte("idempotency-v1\x00"))
	h.Write([]byte(input))
	return hex.EncodeToString(h.Sum(nil)), equipment.OK
}
