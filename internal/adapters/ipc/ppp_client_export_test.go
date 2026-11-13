package ipc

import (
	"encoding/json"
	"infradesktop/internal/modules/equipment"
	"strings"
	"testing"
)

func TestSanitizedSessionExportNeverIncludesLocalClientName(t *testing.T) {
	row := equipment.PPRow{ID: "row", Login: "ana", Client: "Ana Maria", Address: "192.0.2.1", Caller: "AA:BB:CC:DD:EE:FF"}
	safe := sanitizedSessionRow([]byte(strings.Repeat("k", 32)), "observation", row)
	body, err := json.Marshal(safe)
	if err != nil || strings.Contains(string(body), "Ana Maria") || strings.Contains(string(body), "192.0.2.1") {
		t.Fatal("identifiable subscriber data in sanitized JSON", err, string(body))
	}
}
