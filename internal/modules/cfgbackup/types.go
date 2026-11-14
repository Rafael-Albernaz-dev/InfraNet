package cfgbackup

import (
	"context"
	"infradesktop/internal/modules/equipment"
	"time"
)

const MaxBytes = 4 * 1024 * 1024
const MaxVersions = 30

type Format string

const (
	RSC Format = "rsc"
	CFG Format = "cfg"
)

type Origin = equipment.ConfigOrigin

const (
	Auto   = equipment.ConfigAuto
	Manual = equipment.ConfigManual
)
const Unsupported equipment.Code = "CONFIG_UNSUPPORTED"

type Version struct {
	ID        string `json:"id"`
	DeviceID  string `json:"device_id"`
	CreatedAt string `json:"created_at"`
	Origin    Origin `json:"origin"`
	Format    Format `json:"format"`
	Size      int    `json:"size"`
	Added     int    `json:"added"`
	Removed   int    `json:"removed"`
	First     bool   `json:"first"`
}

// Stored never crosses a presentation boundary.
type Stored struct {
	Version
	Stamp string
	Body  []byte
}
type Status struct {
	AutoAt             string         `json:"-"`
	Automatic          bool           `json:"-"`
	CredentialRevision uint64         `json:"-"`
	At                 string         `json:"at"`
	Code               equipment.Code `json:"code"`
	Unchanged          bool           `json:"unchanged"`
}
type View struct {
	Versions  []Version `json:"versions"`
	Format    Format    `json:"format"`
	Supported bool      `json:"supported"`
	Busy      bool      `json:"busy"`
	Status    Status    `json:"status"`
	Change    *Change   `json:"change,omitempty"`
}
type Input struct {
	DeviceID  string `json:"device_id"`
	VersionID string `json:"version_id,omitempty"`
	Path      string `json:"path,omitempty"`
	Confirmed bool   `json:"confirmed,omitempty"`
}
type Store interface {
	equipment.Store
	ConfigVersions(context.Context, string, string) ([]Stored, equipment.Code)
	ConfigStatus(context.Context, string, string) (Status, equipment.Code)
	ConfigVersion(context.Context, string, string, string) (Stored, equipment.Code)
	SaveConfig(context.Context, string, equipment.Device, *Stored, Status) equipment.Code
}
type Vault interface {
	equipment.Vault
	SealConfig(string, string, string, string, []byte) ([]byte, equipment.Code)
	OpenConfig(string, string, string, []byte, []byte) (string, equipment.Code)
}
type Remote interface {
	ReadConfig(context.Context, equipment.Device, string, string, func() equipment.Code) (string, equipment.Code)
}
type FileWriter interface {
	WriteConfig(string, Format, string, func(func() equipment.Code) equipment.Code) equipment.Code
}

// FormatFor requires a family/version already observed by the app.
func FormatFor(d equipment.Device) Format {
	version := ""
	if d.Result != nil && d.Result.Family == equipment.RouterFamilyV6 {
		version = d.Result.Version
	}
	if d.RadioSummary != nil {
		if d.RadioSummary.Family == equipment.RadioAirOSV6 {
			return CFG
		}
		if d.RadioSummary.Family != equipment.RadioRouterOSWirelessV6 {
			return ""
		}
		version = d.RadioSummary.Version
	}
	if len(version) > 0 && (version[0] == '6' || version[0] == '7') {
		return RSC
	}

	return ""
}
func Due(now time.Time, last string) bool {
	if now.Hour() < 3 || now.Hour() >= 6 {
		return false
	}
	at, err := time.Parse(time.RFC3339Nano, last)
	return err != nil || at.In(now.Location()).Format("2006-01-02") != now.Format("2006-01-02")
}

func ResultCode(code equipment.Code) bool {
	switch code {
	case Unsupported, "EXPORT_UNAVAILABLE", "EXPORT_INVALID_DESTINATION", "EXPORT_EXISTS", "EXPORT_DURABILITY_UNCERTAIN":
		return true
	default:
		return false
	}
}
