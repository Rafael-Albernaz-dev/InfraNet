package presentation

import (
	"encoding/csv"
	"github.com/wailsapp/wails/v2/pkg/runtime"
	"infradesktop/internal/modules/equipment"
	"io"
	"strconv"
	"time"
)

func (a *App) ExportSessionHistory(p, act, device, login, reason string, days int) string {
	a.mu.Lock()
	if a.ppExporting {
		a.mu.Unlock()
		return "BUSY"
	}
	a.ppExporting = true
	generation := a.ppGeneration
	picker, ctx := a.saveDialog, a.ctx
	a.mu.Unlock()
	defer func() { a.mu.Lock(); a.ppExporting = false; a.mu.Unlock() }()
	in := equipment.PPInput{DeviceID: device, Login: login, HistoryReason: reason, HistoryDays: days}
	check := func() string {
		a.mu.Lock()
		defer a.mu.Unlock()
		if generation != a.ppGeneration {
			return "PROFILE_CHANGED"
		}
		probe := in
		probe.Cursor = ""
		return a.ppCallLocked("pppoe.session-history", p, act, probe).Code
	}
	if code := check(); code != "OK" {
		return code
	}
	path,
		err := picker(ctx,
		runtime.SaveDialogOptions{Title: "Exportar histórico — dados identificáveis de assinantes",
			DefaultFilename: "historico-sessao.csv",
			Filters:         []runtime.FileFilter{{DisplayName: "Histórico CSV", Pattern: "*.csv"}}})
	if err != nil {
		return "EXPORT_UNAVAILABLE"
	}
	if path == "" {
		return "EXPORT_CANCELLED"
	}
	produce := func(w io.Writer) string { return a.writeSessionHistory(w, p, act, in, generation) }
	publish := func(fn func() string) string {
		if code := check(); code != "OK" {
			return code
		}
		a.mu.Lock()
		defer a.mu.Unlock()
		if generation != a.ppGeneration {
			return "PROFILE_CHANGED"
		}
		return fn()
	}
	return writePPExportFile(path, ".csv", produce, publish, a.ppExportFS)
}

func (a *App) writeSessionHistory(w io.Writer, p, act string, in equipment.PPInput, generation uint64) string {
	out := csv.NewWriter(w)
	if out.Write([]string{"Conectou",
		"Desconectou",
		"Duração (s)",
		"Motivo",
		"IP",
		"IPv6 prefixo",
		"IPv6 PD",
		"MAC",
		"Serviço",
		"Interface",
		"Concentrador"}) != nil {
		return "EXPORT_UNAVAILABLE"
	}
	for {
		a.mu.Lock()
		if generation != a.ppGeneration {
			a.mu.Unlock()
			return "PROFILE_CHANGED"
		}
		r := a.ppCallLocked("pppoe.session-history", p, act, in)
		a.mu.Unlock()
		if r.Code != "OK" {
			return r.Code
		}
		if r.Data == nil {
			return "EXPORT_UNAVAILABLE"
		}
		for _, row := range r.Data.History {
			values := historyCSVValues(row)
			for i := range values {
				values[i] = ppCSVCell(values[i])
			}
			if out.Write(values) != nil {
				return "EXPORT_UNAVAILABLE"
			}
		}
		if r.Data.Next == "" {
			break
		}
		if in.Cursor == r.Data.Next {
			return "EXPORT_UNAVAILABLE"
		}
		in.Cursor = r.Data.Next
	}
	out.Flush()
	if out.Error() != nil {
		return "EXPORT_UNAVAILABLE"
	}
	return "OK"
}

func historyCSVValues(row equipment.PPSessionHistory) []string {
	duration := ""
	connected, e1 := time.Parse(time.RFC3339Nano, row.Connected)
	disconnected, e2 := time.Parse(time.RFC3339Nano, row.Disconnected)
	if e1 == nil && e2 == nil {
		duration = strconv.FormatInt(int64(disconnected.Sub(connected)/time.Second), 10)
	}
	values := []string{row.Connected, row.Disconnected, duration, row.Reason, row.Row.Address, "", "", row.Row.Caller, "", "", row.DeviceID}
	if c := row.Row.Context; c != nil {
		values[6], values[8], values[9] = c.DelegatedPrefix, c.Service, c.Interface
	}
	return values
}

type SessionHistoryQuery struct {
	Login  string `json:"login"`
	Reason string `json:"reason"`
	Days   int    `json:"days"`
	Cursor string `json:"cursor"`
}

func (a *App) PPPoESessionHistory(p, act, device string, q SessionHistoryQuery) PPPoEResult {
	return a.ppCall("pppoe.session-history",
		p,
		act,
		equipment.PPInput{DeviceID: device,
			Login:         q.Login,
			HistoryReason: q.Reason,
			HistoryDays:   q.Days,
			Cursor:        q.Cursor})
}
