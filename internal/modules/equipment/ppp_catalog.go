package equipment

import "context"

const PPPCatalogScope = "ppp-catalog"
const PPPMaxSecrets = 4096
const PPPCatalogBytes = 8 * 1024 * 1024

// PPPEntry is a local scalar projection; no password or raw comment is retained.
type PPPEntry struct {
	At             string   `json:"at,omitempty"`
	APs            []string `json:"aps,omitempty"`
	Peak           *int     `json:"peak,omitempty"`
	Drop           *float64 `json:"drop,omitempty"`
	ID             string   `json:"id"`
	Kind           string   `json:"kind"`
	DeviceID       string   `json:"device_id"`
	Concentrator   string   `json:"concentrator"`
	ObservationID  string   `json:"observation_id,omitempty"`
	Name           string   `json:"name"`
	Client         string   `json:"client,omitempty"`
	Service        string   `json:"service,omitempty"`
	Profile        string   `json:"profile,omitempty"`
	Disabled       *bool    `json:"disabled,omitempty"`
	Invalid        *bool    `json:"invalid,omitempty"`
	Interface      string   `json:"interface,omitempty"`
	VLAN           *int     `json:"vlan,omitempty"`
	MTU            string   `json:"mtu,omitempty"`
	MRU            string   `json:"mru,omitempty"`
	Authentication string   `json:"authentication,omitempty"`
	Rate           string   `json:"rate,omitempty"`
	LocalAddress   string   `json:"local_address,omitempty"`
	RemoteAddress  string   `json:"remote_address,omitempty"`
	OnlyOne        string   `json:"only_one,omitempty"`
	Caller         string   `json:"caller_id,omitempty"`
	LastOut        string   `json:"last_logged_out,omitempty"`
	LastCaller     string   `json:"last_caller_id,omitempty"`
	LastReason     string   `json:"last_disconnect_reason,omitempty"`
	State          string   `json:"state,omitempty"`
	Online         int      `json:"online"`
	Registrations  int      `json:"registrations"`
	Session        *PPRow   `json:"session,omitempty"`
}

type PPPCatalog struct {
	At      string     `json:"at"`
	Limited bool       `json:"limited"`
	Entries []PPPEntry `json:"entries"`
}

type PPPQuery struct {
	Screen   string            `json:"screen"`
	Search   string            `json:"search,omitempty"`
	Sort     string            `json:"sort,omitempty"`
	Desc     bool              `json:"desc,omitempty"`
	Filters  map[string]string `json:"filters,omitempty"`
	PageSize int               `json:"page_size,omitempty"`
}

type PPPSummary struct {
	Online          int `json:"online"`
	Offline         int `json:"offline"`
	DropsToday      int `json:"drops_today"`
	Plans           int `json:"plans"`
	Servers         int `json:"servers"`
	FallingServers  int `json:"falling_servers"`
	UnlinkedServers int `json:"unlinked_servers"`
	InvalidServers  int `json:"invalid_servers"`
}

type PPPCatalogStore interface {
	PPPRadioLinks(context.Context, string, string) ([]RadioLink, Code)
	SavePPPCatalog(context.Context, string, Device, uint64, *PPReadResult) Code
	ReadPPPCatalog(context.Context, string, string) ([]PPPEntry, map[string]string, bool, Code)
	PPPDropsToday(context.Context, string, string, string) (int, Code)
}
