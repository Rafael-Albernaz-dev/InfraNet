// Package contactprobe provides a deterministic network boundary for import fixtures.
package contactprobe

import (
	"context"
	"infradesktop/internal/modules/equipment"
	"sync/atomic"
	"time"
)

type Adapter struct {
	equipment.Remote
	Calls atomic.Int64
}

func (a *Adapter) FirstContact(ctx context.Context,
	d equipment.Device,
	_ string,
	_ string,
	guard func() equipment.Code) (*equipment.ContactResult,
	equipment.Code) {
	if ctx.Err() != nil {
		return nil, equipment.Cancelled
	}
	if code := guard(); code != equipment.OK {
		return nil, code
	}
	a.Calls.Add(1)
	return &equipment.ContactResult{Fingerprint: "SHA256:synthetic-import", Observation: &equipment.Observation{
		Identity: "Fixture " + d.Address, Board: "Synthetic", Version: "fixture", At: equipment.PPTime(time.Now())}}, equipment.OK
}
