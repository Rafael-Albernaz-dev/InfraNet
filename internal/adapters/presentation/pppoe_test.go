package presentation

import (
	"context"

	"encoding/json"
	"errors"
	"github.com/google/uuid"
	"github.com/wailsapp/wails/v2/pkg/runtime"
	"golang.org/x/sys/unix"

	"infradesktop/internal/modules/equipment"
	"io"
	"os"
	"path/filepath"
	"strings"

	"testing"
	"time"
)

func TestPPPoEExportFilesystemFailures(t *testing.T) {
	producer := func(w io.Writer) string {
		_, e := io.WriteString(w, `{"profile":"pppoe-sanitized-v1","rows":[]}`)
		if e != nil {
			return "EXPORT_UNAVAILABLE"
		}
		return "OK"
	}
	publish := func(fn func() string) string { return fn() }
	for _, mode := range []string{"ok", "existing", "symlink", "ancestor-symlink", "file-sync", "dir-sync", "dir-sync-unlink-fails", "close", "rename", "budget", "context"} {
		t.Run(mode, func(t *testing.T) {
			root := t.TempDir()
			path := filepath.Join(root, "out.json")
			ops := ppFileOps{}
			prod := producer
			pub := publish
			want := "EXPORT_UNAVAILABLE"
			switch mode {
			case "ok":
				want = "OK"
			case "existing":
				os.WriteFile(path, []byte("preserve"), 0600)
				want = "EXPORT_EXISTS"
			case "symlink":
				os.Symlink("missing", path)
				want = "EXPORT_EXISTS"
			case "ancestor-symlink":
				os.Symlink(root, filepath.Join(root, "link"))
				path = filepath.Join(root, "link", "out.json")
			case "file-sync":
				ops.sync = func(int) error { return errors.New("synthetic") }
			case "dir-sync", "dir-sync-unlink-fails":
				calls := 0
				ops.sync = func(fd int) error {
					calls++
					if calls == 1 {
						return unix.Fsync(fd)
					}
					return errors.New("synthetic")
				}
				if mode == "dir-sync-unlink-fails" {
					want = "EXPORT_DURABILITY_UNCERTAIN"
					ops.unlink = func(int, string, int) error { return errors.New("synthetic") }
				}
			case "close":
				ops.close = func(f *os.File) error { f.Close(); return errors.New("synthetic") }
			case "rename":
				ops.rename = func(int, string, int, string, uint) error { return errors.New("synthetic") }
			case "budget":
				prod = func(w io.Writer) string {
					_, e := io.WriteString(w, strings.Repeat("x", ppExportBytes+1))
					if e != nil {
						return "EXPORT_UNAVAILABLE"
					}
					return "OK"
				}
			case "context":
				want = "PROFILE_CHANGED"
				pub = func(func() string) string { return "PROFILE_CHANGED" }
			}
			if c := writePPExport(path, prod, pub, ops); c != want {
				t.Fatalf("got %s want %s", c, want)
			}
			raw, e := os.ReadFile(path)
			switch mode {
			case "ok", "dir-sync-unlink-fails":
				if e != nil || !json.Valid(raw) {
					t.Fatal("complete output missing", e)
				}
			case "existing":
				if string(raw) != "preserve" {
					t.Fatal("overwritten")
				}
			case "symlink":
				if _, e := os.Lstat(path); e != nil {
					t.Fatal("symlink removed")
				}
			default:
				if e == nil {
					t.Fatal("unexpected final output")
				}
			}
		})
	}
}

func TestPPPoEReviewExportInitialAndFinalInputs(t *testing.T) {
	rig := newOperationalRig(t)
	rig.enrollDevice()
	rig.acceptTrustWithoutPassword()
	app, profile, activation, d, root := rig.app, rig.profile, rig.activation, rig.device, rig.root
	collect := func() *equipment.PPObservation { return collectExportFixture(t, app, profile, activation, d.ID) }
	a := collect()
	refID := uuid.NewString()
	if r := app.SavePPPoEReference(profile, activation, d.ID, a.ID, refID, uuid.NewString()); r.Code != "OK" {
		t.Fatal(r.Code)
	}
	b := collect()
	t.Run("invalid-before-picker", func(t *testing.T) {
		called := false
		app.saveDialog = func(context.Context, runtime.SaveDialogOptions) (string, error) { called = true; return "", nil }
		code := app.ExportPPPoE(uuid.NewString(), activation, d.ID, b.ID, a.ID)
		if called || code != "PROFILE_CHANGED" {
			t.Fatal("invalid context opened picker", called, code)
		}
	})
	t.Run("base-deleted-at-file-sync", func(t *testing.T) {
		dest := filepath.Join(root, "deleted-base.json")
		app.saveDialog = func(context.Context, runtime.SaveDialogOptions) (string, error) { return dest, nil }
		calls := 0
		app.ppExportFS = ppFileOps{sync: func(fd int) error {
			calls++
			if calls == 1 {
				r := app.DeletePPPoEReference(profile, activation, d.ID, refID, uuid.NewString())
				if r.Code != "OK" {
					t.Fatal(r.Code)
				}
			}
			return unix.Fsync(fd)
		}}
		code := app.ExportPPPoE(profile, activation, d.ID, b.ID, a.ID)
		if code != "OBSERVATION_UNAVAILABLE" {
			t.Fatal("deleted base published", code)
		}
		if _, e := os.Stat(dest); !os.IsNotExist(e) {
			t.Fatal("final output exists", e)
		}
	})
}

func TestCSVFormulaNeutralization(t *testing.T) {
	for _, value := range []string{"=1+1", " +SUM(A1:A2)", "-1", "@SUM(1)", "\tformula", "\rformula", "\nformula"} {
		if cell := ppCSVCell(value); !strings.HasPrefix(cell, "'") || cell[1:] != value {
			t.Fatal("active formula/control cell", value)
		}
	}
	for _, value := range []string{"ordinary", "a,b", "a\"b", "198.51.100.1", "02:00:00:00:00:01", "10000000"} {
		if ppCSVCell(value) != value {
			t.Fatal("ordinary cell changed", value)
		}
	}
}

func collectExportFixture(t *testing.T, app *App, profile, activation, device string) *equipment.PPObservation {
	r := app.StartPPPoECollection(profile, activation, device, uuid.NewString(), 2)
	if r.Code != "OK" {
		t.Fatal(r.Code)
	}
	end := time.Now().Add(3 * time.Second)
	for time.Now().Before(end) {
		x := app.PPPoECollectionStatus(profile, activation, r.Data.Job.ID)
		if x.Code != "OK" {
			t.Fatal(x.Code)
		}
		if x.Data.Job.State != "RUNNING" {
			if x.Data.Job.Code != equipment.OK {
				t.Fatal(x.Data.Job)
			}
			return app.GetPPPoECurrent(profile, activation, device).Data.Current
		}
		time.Sleep(time.Millisecond)
	}
	t.Fatal("collection deadline")
	return nil
}
