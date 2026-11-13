package inventory

import (
	"context"
	"infradesktop/internal/modules/equipment"
)

func (s *Service) connectImportLine(ctx context.Context, parent Command, item equipment.ImportItem) (string, Code) {
	id, code := s.IDs.ID()
	if code != OK {
		return "", code
	}
	operation, code := s.IDs.ID()
	if code != OK {
		return "", code
	}
	d, reason := importDevice(item.Row, id, item.CredentialID)
	if reason != "" {
		return "", Invalid
	}
	state, code := s.Store.ReadEquipment(ctx, parent.Scope.Profile)
	if code != OK {
		return "", code
	}
	cr := CredentialByID(&state, d.CredentialID)
	if cr == nil {
		return "", MissingCredential
	}
	q := parent
	q.Op, q.Operation, q.ID, q.Name, q.Address = "inventory.create", operation, id, d.Address, d.Address
	q.Role, q.Transport, q.Port, q.CredentialID = d.Role, d.Transport, d.Port, d.CredentialID
	guard := func() Code { return s.guardNewContact(ctx, q, *cr) }
	result, code := s.ProbeFirstContact(ctx, d, *cr, state, guard)
	if code != OK {
		return "", code
	}
	code = s.Authorized(ctx, q, func(uint64) Code {
		return s.Store.UpdateEquipment(ctx, q.Scope.Profile, func(current *State) Code {
			credential := CredentialByID(current, d.CredentialID)
			if credential == nil || credential.Metadata.Revision != cr.Metadata.Revision {
				return Revision
			}
			for _, existing := range current.Devices {
				if existing.Address == d.Address {
					return Revision
				}
			}
			group, code := s.ensureGroupPath(current, item.Row.Group)
			if code != OK {
				return code
			}
			q.GroupID, q.Contact = group, result
			q.ContactCredentialRevision = cr.Metadata.Revision
			view := View{}
			stamp := buildMutationStamp(q, "")
			mutated := false
			return s.applyMutationState(current, q, &stamp, &view, &mutated)
		})
	})
	return result.Observation.Identity, code
}
