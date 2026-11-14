package radio

import (
	"context"
	"fmt"
)

func (s *Service) readEnlaceAP(ctx context.Context, profile string, d Device, active func() bool) APSeen {
	failed := APSeen{AP: d.ID, Failed: true}
	state, code := s.Store.ReadEquipment(ctx, profile)
	if code != OK || !s.VaultReady(&state) {
		return failed
	}
	cr := credential(&state, d.CredentialID)
	if cr == nil {
		return failed
	}
	guard := func() Code { return s.guardEnlaceAP(ctx, profile, d, cr.Metadata.Revision, active) }
	if guard() != OK {
		return failed
	}
	secret, code := s.Vault.Open(state.Installation, state.Company, cr.Metadata.ID, cr.Metadata.Revision, cr.Envelope, state.WrappedKey)
	if code != OK {
		return failed
	}
	remote, ok := s.Remote.(RadioIndexRemote)
	if !ok {
		return failed
	}
	readCtx, cancel := context.WithTimeout(ctx, QueryBudget)
	defer cancel()
	entries, code := remote.ReadRadioIndex(readCtx, d, cr.Metadata.User, secret, guard)
	secret = ""
	if guard() != OK {
		return failed
	}
	seen := APSeen{AP: d.ID, Failed: code != OK}
	for _, e := range entries {
		signal := ""
		if e.Signal != nil {
			signal = fmt.Sprintf("%g dBm", *e.Signal)
		}
		seen.Clients = append(seen.Clients, APClient{e.MAC, signal, e.Interface, e.BSSID})
	}
	return seen
}
func (s *Service) guardEnlaceAP(ctx context.Context, profile string, d Device, revision uint64, active func() bool) Code {
	if !active() {
		return Changed
	}
	current, code := s.Store.ReadEquipment(ctx, profile)
	if code != OK {
		return code
	}
	cr := credential(&current, d.CredentialID)
	if !sameRadioTarget(device(&current, d.ID), d) || cr == nil || cr.Metadata.Revision != revision {
		return Revision
	}
	return OK
}
