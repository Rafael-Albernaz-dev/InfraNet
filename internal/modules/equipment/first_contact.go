package equipment

import "context"

type ContactResult struct {
	Fingerprint string
	Observation *Observation
}

type ContactRemote interface {
	FirstContact(context.Context, Device, string, string, func() Code) (*ContactResult, Code)
}
