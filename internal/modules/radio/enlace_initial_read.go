package radio

import "context"

// Only the first assembled link reads radio details. Later automatic rounds use
// BasicContact; opening the client keeps the existing full/live use case.
func (s *Service) readInitialEnlace(ctx context.Context, profile string, work enlaceWork, active func() bool) Code {
	state, code := s.Store.ReadEquipment(ctx, profile)
	if code != OK {
		return code
	}
	d := device(&state, work.link.CPE)
	if d == nil {
		return Changed
	}
	cr := credential(&state, d.CredentialID)
	if cr == nil {
		return MissingCredential
	}
	guard := func() Code {
		if !active() || !s.enlaceSessionOnline(ctx, profile, work.session, *d) {
			return Changed
		}
		return s.guardEnlaceAP(ctx, profile, *d, cr.Metadata.Revision, active)
	}
	if code := guard(); code != OK {
		return code
	}
	if !s.VaultReady(&state) {
		return VaultUnavailable
	}
	secret, code := s.Vault.Open(state.Installation, state.Company, cr.Metadata.ID, cr.Metadata.Revision, cr.Envelope, state.WrappedKey)
	if code != OK {
		return code
	}
	remote, ok := s.Remote.(RadioRemote)
	if !ok {
		return Unavailable
	}
	readCtx, cancel := context.WithTimeout(ctx, QueryBudget)
	defer cancel()
	reading, outcome := remote.ReadRadio(readCtx, *d, cr.Metadata.User, secret, RadioStandalone, guard)
	secret = ""
	if code := guard(); code != OK {
		return code
	}
	if outcome != OK {
		return s.recordEnlaceReadFailure(ctx, profile, *d, cr.Metadata.Revision, outcome)
	}
	if reading == nil {
		return Partial
	}
	reading.ID, code = s.IDs.ID()
	if code != OK {
		return code
	}
	StampRadio(reading, *d, cr.Metadata.Revision, s.Now())
	store, ok := s.Store.(RadioStore)
	if !ok {
		return Unavailable
	}
	return store.SaveRadio(ctx, profile, *d, cr.Metadata.Revision, reading)
}
func (s *Service) recordEnlaceReadFailure(ctx context.Context, profile string, d Device, revision uint64, outcome Code) Code {
	code := s.Store.UpdateEquipment(ctx, profile, func(state *State) Code {
		current := device(state, d.ID)
		cr := credential(state, d.CredentialID)
		if !sameRadioTarget(current, d) || cr == nil || cr.Metadata.Revision != revision {
			return Revision
		}
		current.LastAttempt = &Attempt{Code: outcome, At: PPTime(s.Now()), DeviceRevision: d.Revision, CredentialRevision: revision}
		return OK
	})
	if code != OK {
		return code
	}
	return outcome
}
