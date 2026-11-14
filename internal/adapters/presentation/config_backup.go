package presentation

import (
	"github.com/wailsapp/wails/v2/pkg/runtime"
	"infradesktop/internal/adapters/ipc"
	"infradesktop/internal/modules/cfgbackup"
	"infradesktop/internal/modules/equipment"
	"io"
)

type ConfigResult struct {
	Code string          `json:"code"`
	Data *cfgbackup.View `json:"data,omitempty"`
}

func (a *App) configCall(op, p, act string, in cfgbackup.Input) ConfigResult {
	a.mu.Lock()
	defer a.mu.Unlock()
	return a.configCallLocked(op, p, act, in)
}
func (a *App) configCallLocked(op, p, act string, in cfgbackup.Input) ConfigResult {
	r := a.client.Call(a.ctx, ipc.Request{Op: op, ExpectedProfile: p, ExpectedActivation: act, Config: &in})
	out := ConfigResult{Code: string(r.Code)}
	if r.Data != nil {
		out.Data = r.Data.Config
	}
	return out
}
func (a *App) ListConfig(p, act, device string) ConfigResult {
	return a.configCall("config.list", p, act, cfgbackup.Input{DeviceID: device})
}
func (a *App) SaveConfig(p, act, device string) ConfigResult {
	return a.configCall("config.save", p, act, cfgbackup.Input{DeviceID: device})
}
func (a *App) CompareConfig(p, act, device, version string) ConfigResult {
	return a.configCall("config.compare", p, act, cfgbackup.Input{DeviceID: device, VersionID: version})
}
func (a *App) DownloadConfig(p, act, device, version string, confirmed bool) ConfigResult {
	if !confirmed {
		return ConfigResult{Code: string(equipment.Invalid)}
	}
	a.mu.Lock()
	generation := a.ppGeneration
	picker, ctx := a.saveDialog, a.ctx
	list := a.configCallLocked("config.list", p, act, cfgbackup.Input{DeviceID: device})
	a.mu.Unlock()
	if list.Code != "OK" || list.Data == nil {
		return list
	}
	var format cfgbackup.Format
	for _, v := range list.Data.Versions {
		if v.ID == version {
			format = v.Format
		}
	}
	if format == "" {
		return ConfigResult{Code: string(equipment.Invalid)}
	}
	path, err := picker(ctx, runtime.SaveDialogOptions{Title: "Este arquivo contém senhas do equipamento",
		DefaultFilename: "configuracao." + string(format), Filters: []runtime.FileFilter{{DisplayName: "Configuração com senhas",
			Pattern: "*." + string(format)}}})
	if err != nil {
		return ConfigResult{Code: "EXPORT_UNAVAILABLE"}
	}
	if path == "" {
		return ConfigResult{Code: "EXPORT_CANCELLED"}
	}
	a.mu.Lock()
	defer a.mu.Unlock()
	if generation != a.ppGeneration {
		return ConfigResult{Code: "PROFILE_CHANGED"}
	}
	return a.configCallLocked("config.download", p, act, cfgbackup.Input{
		DeviceID: device, VersionID: version, Path: path, Confirmed: true})
}

// ConfigFiles is composed only in the daemon. Plaintext never returns through IPC.
type ConfigFiles struct{}

func (ConfigFiles) WriteConfig(path string, format cfgbackup.Format, text string,
	publish func(func() equipment.Code) equipment.Code) equipment.Code {
	produce := func(w io.Writer) string {
		// ignored: only the result code or validation outcome is needed here.
		if _, err := io.WriteString(w, text); err != nil {
			return "EXPORT_UNAVAILABLE"
		}
		return "OK"
	}
	commit := func(fn func() string) string {
		return string(publish(func() equipment.Code { return equipment.Code(fn()) }))
	}
	return equipment.Code(writePPExportFile(path, "."+string(format), produce, commit, ppFileOps{maximum: cfgbackup.MaxBytes}))
}
