package inventory

import (
	"context"
	"infradesktop/internal/modules/equipment"
	"time"
)

// ProbeFirstContact runs outside the store transaction. The adapter must check
// authority before authentication and each command, including after inspection.
func (s *Service) ProbeFirstContact(ctx context.Context, d Device, credential SealedCredential,
	state State, guard func() Code) (*equipment.ContactResult, Code) {
	select {
	case s.ContactSlots <- struct{}{}:
	case <-ctx.Done():
		return nil, Cancelled
	}
	defer func() { <-s.ContactSlots }()
	if code := guard(); code != OK {
		return nil, code
	}
	if !s.VaultReady(&state) {
		return nil, VaultUnavailable
	}
	secret, code := s.Vault.Open(state.Installation, state.Company, credential.Metadata.ID,
		credential.Metadata.Revision, credential.Envelope, state.WrappedKey)
	if code != OK {
		return nil, code
	}
	probe, ok := s.Remote.(equipment.ContactRemote)
	if !ok {
		return nil, Unavailable
	}
	ctx, cancel := context.WithTimeout(ctx, QueryBudget)
	defer cancel()
	result, code := probe.FirstContact(ctx, d, credential.Metadata.User, secret, guard)
	secret = ""
	if code != OK {
		return nil, code
	}
	if code := guard(); code != OK {
		return nil, code
	}
	if result == nil || result.Observation == nil {
		return nil, Partial
	}
	fingerprint, code := FirstContactKey(d.Fingerprint, result.Fingerprint)
	if code != OK {
		return nil, code
	}
	result.Fingerprint = fingerprint
	return result, OK
}

func acceptContact(d *Device, result *equipment.ContactResult, profile string, uid uint32, at time.Time) {
	d.Fingerprint = result.Fingerprint
	d.Identity = result.Observation.Identity
	d.Result = result.Observation
	d.TrustOrigin = "automatic-tofu"
	d.TrustAcceptance = &TrustAcceptance{Mode: "automatic-tofu", At: PPTime(at), UID: uid,
		Company: profile, DeviceID: d.ID, Address: d.Address, Port: d.Port, Revision: d.Revision, Fingerprint: d.Fingerprint}
}

func (s *Service) register(ctx context.Context, q Command, v *View) Code {
	if code := canonicalizeMutation(&q); code != OK {
		return code
	}
	q.Op = "inventory.create"
	state, code := s.Store.ReadEquipment(ctx, q.Scope.Profile)
	if code != OK {
		return code
	}
	if replayed, code := checkMutationReplay(&state, q, buildMutationStamp(q, ""), v); replayed {
		return code
	}
	d := Device{ID: q.ID, Name: q.Address, GroupID: q.GroupID, Address: q.Address, Port: q.Port,
		Role: q.Role, Transport: q.Transport, CredentialID: q.CredentialID, Active: true, Revision: 1}
	if code := prepareDeviceCreate(&state, DeviceByID(&state, q.ID), q, &d); code != OK {
		return code
	}
	if code := ValidateDevice(d); code != OK {
		return code
	}
	if q.PendingCredential != nil {
		state.Credentials = append(state.Credentials, *q.PendingCredential)
		state.WrappedKey = q.PendingWrapped
	}
	cr := CredentialByID(&state, d.CredentialID)
	if cr == nil || cr.Metadata.Revision != q.ContactCredentialRevision {
		return MissingCredential
	}
	q.CredentialID = d.CredentialID
	guard := func() Code { return s.guardNewContact(ctx, q, *cr) }
	result, probeCode := s.ProbeFirstContact(ctx, d, *cr, state, guard)
	if probeCode != OK {
		v.ContactCode = probeCode
		return OK
	}
	if code := guard(); code != OK {
		return code
	}
	q.Contact, q.ContactCode = result, probeCode
	return s.Authorized(ctx, q, func(uint64) Code { return s.mutate(ctx, q, v) })
}

func (s *Service) guardNewContact(ctx context.Context, q Command, credential SealedCredential) Code {
	return s.Authorized(ctx, q, func(uint64) Code {
		state, code := s.Store.ReadEquipment(ctx, q.Scope.Profile)
		if code != OK {
			return code
		}
		if q.PendingCredential != nil {
			if DeviceByID(&state, q.ID) != nil || CredentialByID(&state, q.ID) != nil {
				return Revision
			}
			return OK
		}
		current := CredentialByID(&state, credential.Metadata.ID)
		if DeviceByID(&state, q.ID) != nil || current == nil || current.Metadata.Revision != credential.Metadata.Revision {
			return Revision
		}
		return OK
	})
}
