package inventory

func (s *Service) prepareInlineContact(state *State, q *Command) Code {
	if !s.VaultReady(state) {
		return VaultUnavailable
	}
	if q.CredentialID != "" || DeviceByID(state, q.ID) != nil || CredentialByID(state, q.ID) != nil {
		return Revision
	}
	if len(state.Credentials) >= MaxCredentials {
		return Limit
	}
	if code := ValidateCredential(q.Address, q.User, q.Secret); code != OK {
		return code
	}
	stamp, code := s.Vault.Stamp(q.Secret)
	if code != OK {
		return code
	}
	q.ContactSecretStamp = stamp
	envelope, wrapped, code := s.Vault.Seal(state.Installation, state.Company, q.ID, 1, q.Secret, state.WrappedKey)
	q.Secret = ""
	if code != OK {
		return code
	}
	q.PendingCredential = &SealedCredential{Metadata: Credential{ID: q.ID, Alias: q.Address, User: q.User, Revision: 1}, Envelope: envelope}
	q.PendingWrapped, q.CredentialID = wrapped, q.ID
	return OK
}

func commitInlineCredential(state *State, q Command) Code {
	if CredentialByID(state, q.ID) != nil || DeviceByID(state, q.ID) != nil {
		return Revision
	}
	if len(state.Credentials) >= MaxCredentials {
		return Limit
	}
	if len(state.WrappedKey) != 0 && string(state.WrappedKey) != string(q.PendingWrapped) {
		return Revision
	}
	state.WrappedKey = q.PendingWrapped
	state.Credentials = append(state.Credentials, *q.PendingCredential)
	return OK
}
