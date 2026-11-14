package equipment

import "context"

type InterfaceAPs struct {
	ConcentratorName string   `json:"concentrator_name,omitempty"`
	Service          string   `json:"service,omitempty"`
	Concentrator     string   `json:"concentrator_id"`
	Interface        string   `json:"interface"`
	APs              []string `json:"ap_ids"`
}
type CPELogin struct {
	Session   *CPESessionPresence `json:"-"`
	Login     string              `json:"login"`
	Origin    string              `json:"origin"`
	CreatedAt string              `json:"created_at"`
}

// CPESessionPresence is persisted session evidence, never a probe result.
type CPESessionPresence struct {
	State, Since, ReadAt, LastOnline string
}
type EnlaceChoice struct {
	AP     string `json:"ap_id"`
	Name   string `json:"name"`
	Signal string `json:"signal"`
}
type EnlaceView struct {
	MAC       string         `json:"mac,omitempty"`
	State     string         `json:"state"`
	Stage     int            `json:"stage"`
	Interface string         `json:"interface"`
	Login     string         `json:"login"`
	Address   string         `json:"address"`
	APs       []string       `json:"ap_ids"`
	Choices   []EnlaceChoice `json:"choices,omitempty"`
	Failed    []string       `json:"failed,omitempty"`
	Reason    string         `json:"reason,omitempty"`
	Link      *RadioLink     `json:"link,omitempty"`
}
type EnlaceSession struct {
	Concentrator string
	Observation  string
	Row          PPRow
}

type EnlaceStore interface {
	EnlaceSessions(context.Context, string) ([]EnlaceSession, Code)
	InterfaceAPs(context.Context, string) ([]InterfaceAPs, Code)
	SetInterfaceAPs(context.Context, string, InterfaceAPs) Code
}
