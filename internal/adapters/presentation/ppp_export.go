package presentation

import (
	"encoding/csv"
	"github.com/wailsapp/wails/v2/pkg/runtime"
	"infradesktop/internal/modules/equipment"
	"io"
	"sort"
)

type pppExportTarget struct{ device, observation, name string }
type pppExport struct {
	app                 *App
	profile, activation string
	generation          uint64
	targets             []pppExportTarget
}

// ExportPPP freezes local observations before opening the native picker. JS
// chooses only the format; filenames, bytes and sanitization stay in this adapter.
func (a *App) ExportPPP(profile, activation, format string) string {
	if format != "csv" && format != "json" {
		return "INVALID_REQUEST"
	}
	a.mu.Lock()
	if a.ppExporting {
		a.mu.Unlock()
		return "BUSY"
	}
	a.ppExporting = true
	session := pppExport{app: a, profile: profile, activation: activation, generation: a.ppGeneration}
	picker, ctx := a.saveDialog, a.ctx
	a.mu.Unlock()
	defer func() { a.mu.Lock(); a.ppExporting = false; a.mu.Unlock() }()
	if code := session.freeze(); code != "OK" {
		return code
	}
	title := "Exportar PPP — JSON sanitizado"
	if format == "csv" {
		title = "Exportar PPP — dados sensíveis de assinantes (CSV)"
	}
	path, err := picker(ctx, runtime.SaveDialogOptions{Title: title, DefaultFilename: "ppp." + format,
		Filters: []runtime.FileFilter{{DisplayName: format, Pattern: "*." + format}}})
	if err != nil {
		return "EXPORT_UNAVAILABLE"
	}
	if path == "" {
		return "EXPORT_CANCELLED"
	}
	produce := session.json
	if format == "csv" {
		produce = session.csv
	}
	return writePPExportFile(path, "."+format, produce, session.publish, a.ppExportFS)
}
func (s *pppExport) freeze() string {
	in := equipment.PPInput{PPPQuery: &equipment.PPPQuery{Screen: "sessions", PageSize: 200}}
	targets := map[string]pppExportTarget{}
	for pages := 0; pages <= equipment.MaxDevices*equipment.PPMaxRows/200; pages++ {
		s.app.mu.Lock()
		if s.app.ppGeneration != s.generation {
			s.app.mu.Unlock()
			return "PROFILE_CHANGED"
		}
		r := s.app.ppCallLocked("pppoe.ppp-list", s.profile, s.activation, in)
		s.app.mu.Unlock()
		if r.Code != "OK" {
			return r.Code
		}
		if r.Data == nil {
			return "EXPORT_UNAVAILABLE"
		}
		for _, e := range r.Data.Entries {
			targets[e.DeviceID] = pppExportTarget{e.DeviceID, e.ObservationID, e.Concentrator}
		}
		if r.Data.Next == "" {
			for _, target := range targets {
				s.targets = append(s.targets, target)
			}
			sort.Slice(s.targets, func(i, j int) bool { return s.targets[i].device < s.targets[j].device })
			return "OK"
		}
		if r.Data.Next == in.Cursor {
			return "EXPORT_UNAVAILABLE"
		}
		in.Cursor = r.Data.Next
	}
	return "EXPORT_UNAVAILABLE"
}
func (s *pppExport) check() string {
	s.app.mu.Lock()
	defer s.app.mu.Unlock()
	if s.app.ppGeneration != s.generation {
		return "PROFILE_CHANGED"
	}
	if len(s.targets) == 0 {
		return s.app.ppCallLocked("pppoe.ppp-list", s.profile, s.activation,
			equipment.PPInput{PPPQuery: &equipment.PPPQuery{Screen: "sessions", PageSize: 25}}).Code
	}
	for _, target := range s.targets {
		if code := s.app.checkPPExportLocked(s.profile, s.activation, target.device, target.observation, ""); code != "OK" {
			return code
		}
	}
	return "OK"
}
func (s *pppExport) publish(write func() string) string {
	if code := s.check(); code != "OK" {
		return code
	}
	s.app.mu.Lock()
	defer s.app.mu.Unlock()
	if s.app.ppGeneration != s.generation {
		return "PROFILE_CHANGED"
	}
	return write()
}
func (s *pppExport) json(w io.Writer) string {
	if code := s.check(); code != "OK" {
		return code
	}
	// ignored: Only the write error is needed; a partial write aborts publication.
	if _, err := io.WriteString(w, "[\n"); err != nil {
		return "EXPORT_UNAVAILABLE"
	}
	for i, target := range s.targets {
		if i > 0 {
			// ignored: Only the write error is needed; a partial write aborts publication.
			if _, err := io.WriteString(w, ",\n"); err != nil {
				return "EXPORT_UNAVAILABLE"
			}
		}
		local := ppExportSession{app: s.app, profile: s.profile, activation: s.activation, device: target.device, target: target.observation,
			generation: s.generation, aliases: map[string]string{}}
		if code := local.initCrypto(); code != "OK" {
			return code
		}
		if code := local.produce(w); code != "OK" {
			return code
		}
	}
	// ignored: Only the write error is needed; a partial write aborts publication.
	if _, err := io.WriteString(w, "\n]\n"); err != nil {
		return "EXPORT_UNAVAILABLE"
	}
	return "OK"
}
func (s *pppExport) csv(w io.Writer) string {
	if code := s.check(); code != "OK" {
		return code
	}
	if len(s.targets) == 0 {
		out := csv.NewWriter(w)
		if out.Write(append(operationalCSVHeader(), "concentrador")) != nil {
			return "EXPORT_UNAVAILABLE"
		}
		out.Flush()
		if out.Error() != nil {
			return "EXPORT_UNAVAILABLE"
		}
		return "OK"
	}
	for i, target := range s.targets {
		request := operationalCSVRequest{profile: s.profile, activation: s.activation, device: target.device,
			observation: target.observation, generation: s.generation, concentrator: target.name, skipHeader: i > 0}
		if code := s.app.produceOperationalCSV(w, request, s.check); code != "OK" {
			return code
		}
	}
	return "OK"
}
