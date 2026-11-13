package ipc

import (
	"bytes"
	"encoding/json"
	"io"
)

// Preferences contain arbitrary filter keys and nullable optional fields. Walk
// every object so duplicate nested keys cannot silently choose another value.
func unambiguousTablePreference(body []byte) bool {
	decoder := json.NewDecoder(bytes.NewReader(body))
	if !unambiguousPreferenceValue(decoder) {
		return false
	}
	// ignored: only the EOF error is needed after the complete value has been validated.
	_, err := decoder.Token()
	return err == io.EOF
}
func unambiguousPreferenceValue(decoder *json.Decoder) bool {
	token, err := decoder.Token()
	if err != nil {
		return false
	}
	delimiter, compound := token.(json.Delim)
	if !compound {
		return true
	}
	seen := map[string]bool{}
	for decoder.More() {
		if delimiter == '{' {
			key, err := decoder.Token()
			name, ok := key.(string)
			if err != nil || !ok || seen[name] {
				return false
			}
			seen[name] = true
		}
		if !unambiguousPreferenceValue(decoder) {
			return false
		}
	}
	end, err := decoder.Token()
	return err == nil && (delimiter == '{' && end == json.Delim('}') || delimiter == '[' && end == json.Delim(']'))
}
