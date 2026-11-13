package presentation

import (
	"bytes"
	"context"
	"encoding/csv"
	"encoding/json"
	"github.com/wailsapp/wails/v2/pkg/runtime"
	"infradesktop/internal/modules/equipment"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// The same export journey exercises native-picker publication for local names
// and verifies that the aggregate JSON reuses the sanitized projection.
func (s *ppExportJourney) assertGlobalExports(t *testing.T, login string) {
	t.Helper()
	picker := s.app.saveDialog
	defer func() { s.app.saveDialog = picker }()
	target, code := s.store.ReadPPTarget(context.Background(), s.profile, s.d.ID, true)
	if code != equipment.OK {
		t.Fatal(code)
	}
	result := &equipment.PPReadResult{Scope: equipment.PPPCatalogScope, Ended: s.b.Provenance.Ended,
		Rows: []equipment.PPReadRow{{Kind: "secret", Entry: &equipment.PPPEntry{
			ID: "local", Kind: "secret", Name: login, Client: "CLIENT_LOCAL_CANARY",
		}}}}
	if code := s.store.SavePPPCatalog(
		context.Background(), s.profile, target.Device, target.Credential.Metadata.Revision, result,
	); code != equipment.OK {
		t.Fatal(code)
	}
	for _, format := range []string{"csv", "json"} {
		path := filepath.Join(s.root, "aggregate."+format)
		s.app.saveDialog = func(_ context.Context, options runtime.SaveDialogOptions) (string, error) {
			if format == "csv" && !strings.Contains(options.Title, "dados sensíveis de assinantes") {
				t.Fatal("missing CSV disclosure")
			}
			return path, nil
		}
		if code := s.app.ExportPPP(s.profile, s.activation, format); code != "OK" {
			t.Fatal("aggregate export", format, code)
		}
		body, err := os.ReadFile(path)
		if err != nil {
			t.Fatal(err)
		}
		if format == "csv" {
			assertGlobalCSV(t, body, s.b.Count)
		} else {
			assertGlobalJSON(t, body, s.b.Count)
		}
	}
}
func assertGlobalCSV(t *testing.T, body []byte, count int) {
	t.Helper()
	rows, err := csv.NewReader(bytes.NewReader(body)).ReadAll()
	if err != nil || len(rows) != count+1 || rows[0][len(rows[0])-1] != "concentrador" {
		t.Fatal("aggregate CSV shape", err, len(rows))
	}
	if !bytes.Contains(body, []byte("CLIENT_LOCAL_CANARY")) {
		t.Fatal("CSV lost the disclosed local client")
	}
}
func assertGlobalJSON(t *testing.T, body []byte, count int) {
	t.Helper()
	var groups []struct {
		Rows []equipment.PPExportRow `json:"rows"`
	}
	if json.Unmarshal(body, &groups) != nil || len(groups) != 1 || len(groups[0].Rows) != count {
		t.Fatal("aggregate JSON shape")
	}
	for _, canary := range []string{"CLIENT_LOCAL_CANARY", "synthetic-", "198.51.", "02:00:"} {
		if bytes.Contains(body, []byte(canary)) {
			t.Fatal("aggregate JSON leaked local identity", canary)
		}
	}
}
