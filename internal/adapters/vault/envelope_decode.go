package vault

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
)

func decode(b []byte) (envelope, error) { return decodeLimit(b, 1024) }
func decodeLimit(b []byte, maximum int) (envelope, error) {
	var e envelope
	encodedMaximum := 4096
	if maximum > 1024 {
		encodedMaximum = maximum*2 + 4096
	}
	if len(b) < 1 || len(b) > encodedMaximum || !uniqueEnvelopeFields(b) {
		return e, errors.New("envelope unavailable")
	}
	d := json.NewDecoder(bytes.NewReader(b))
	d.DisallowUnknownFields()
	if d.Decode(&e) != nil || d.Decode(new(any)) != io.EOF || !validEnvelope(e, maximum) {
		return e, errors.New("envelope unavailable")
	}
	return e, nil
}
func validEnvelope(e envelope, maximum int) bool {
	return e.Version == 1 && e.Algorithm == "AES-256-GCM" && len(e.KeyID) == 32 && len(e.Nonce) == 12 &&
		len(e.Ciphertext) >= 16 && len(e.Ciphertext) <= maximum+16
}
func uniqueEnvelopeFields(b []byte) bool {
	tokens := json.NewDecoder(bytes.NewReader(b))
	first, err := tokens.Token()
	if err != nil || first != json.Delim('{') {
		return false
	}
	seen := map[string]bool{}
	for tokens.More() {
		if !nextEnvelopeField(tokens, seen) {
			return false
		}
	}
	return len(seen) == 5
}
func nextEnvelopeField(tokens *json.Decoder, seen map[string]bool) bool {
	name, err := tokens.Token()
	if err != nil {
		return false
	}
	key, ok := name.(string)
	if !ok || seen[key] {
		return false
	}
	seen[key] = true
	var raw json.RawMessage
	return tokens.Decode(&raw) == nil && !bytes.Equal(raw, []byte("null"))
}
