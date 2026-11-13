package presentation

import (
	"bytes"
	"encoding/csv"
	"github.com/wailsapp/wails/v2/pkg/runtime"
	"infradesktop/internal/adapters/ipc"
	"os"
)

func (a *App) SaveDiscardedImport(profile, activation string) string {
	result := a.equipmentCall("inventory.import-status", profile, activation, ipc.EquipmentInput{})
	if result.Code != "OK" || result.Data == nil {
		return "EXPORT_UNAVAILABLE"
	}
	var body bytes.Buffer
	writer := csv.NewWriter(&body)
	if err := writer.Write([]string{"IP", "Porta", "Papel", "Grupo", "Credencial", "Motivo"}); err != nil {
		return "EXPORT_UNAVAILABLE"
	}
	for _, item := range result.Data.ImportItems {
		if item.Status != "failed" && item.Status != "cancelled" {
			continue
		}
		row := item.Row
		row.Role, row.Group, row.Credential = spreadsheetCell(row.Role), spreadsheetCell(row.Group), spreadsheetCell(row.Credential)
		if err := writer.Write([]string{row.Address, row.Port, row.Role, row.Group, row.Credential, item.Reason}); err != nil {
			return "EXPORT_UNAVAILABLE"
		}
	}
	writer.Flush()
	if writer.Error() != nil {
		return "EXPORT_UNAVAILABLE"
	}
	a.mu.Lock()
	picker, ctx := a.saveDialog, a.ctx
	a.mu.Unlock()
	if picker == nil {
		return "EXPORT_UNAVAILABLE"
	}
	path, err := picker(ctx, runtime.SaveDialogOptions{Title: "Salvar linhas descartadas", DefaultFilename: "linhas-descartadas.csv",
		Filters: []runtime.FileFilter{{DisplayName: "CSV", Pattern: "*.csv"}}})
	if err != nil {
		return "EXPORT_UNAVAILABLE"
	}
	if path == "" {
		return "EXPORT_CANCELLED"
	}
	file, err := os.OpenFile(path, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0600)
	if err != nil {
		return "EXPORT_UNAVAILABLE"
	}
	// ignored: affected count is not needed; the statement error is checked.
	_, writeErr := file.Write(body.Bytes())
	closeErr := file.Close()
	if writeErr != nil || closeErr != nil {
		return "EXPORT_UNAVAILABLE"
	}
	return "OK"
}

// spreadsheetCell prevents CSV formulas while leaving plain addresses unchanged.
func spreadsheetCell(value string) string {
	if len(value) > 0 && (value[0] == '=' || value[0] == '+' || value[0] == '-' || value[0] == '@') {
		return "'" + value
	}
	return value
}
