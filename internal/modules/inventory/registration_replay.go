package inventory

// registrationReplay retains the original probe receipt; it neither opens a
// credential nor publishes data. Authorization is checked by Handle first.
func (s *Service) registrationReplay(q Command, v *View) (bool, Code) {
	s.Mu.Lock()
	var original *Command
	for _, job := range s.Jobs {
		if job.Command.Op == q.Op && SameScope(job.Command, q) && job.Command.Operation == q.Operation {
			copy := job.Command
			original = &copy
			v.Job = CopyJob(job)
			break
		}
	}
	s.Mu.Unlock()
	if original == nil {
		return false, OK
	}
	if original.PendingCredential != nil {
		if q.CredentialID != "" {
			return true, Revision
		}
		stamp, code := s.Vault.Stamp(q.Secret)
		if code != OK {
			return true, code
		}
		q.ContactSecretStamp = stamp
		q.CredentialID = q.ID
	} else if q.CredentialID == "" {
		q.CredentialID = original.CredentialID
	}
	if buildMutationStamp(*original, "") != buildMutationStamp(q, "") {
		return true, Revision
	}
	return true, OK
}
