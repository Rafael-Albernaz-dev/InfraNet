package cfgbackup

import (
	"context"
	"infradesktop/internal/modules/equipment"
	"strings"
	"time"
)

type guard func(func() equipment.Code) equipment.Code

func (s *Service) capture(ctx context.Context, profile string, d equipment.Device, origin Origin, allowed guard) {
	network, cancel := context.WithTimeout(ctx, equipment.QueryBudget)
	defer cancel()
	var st equipment.State
	var cr equipment.SealedCredential
	code := allowed(func() equipment.Code {
		var code equipment.Code
		st, cr, code = s.source(ctx, profile, d)
		return code
	})
	if code != equipment.OK {
		return
	}
	check := func() equipment.Code {
		if origin == Auto && !Due(s.Now(), "") {
			return equipment.Cancelled
		}
		return allowed(func() equipment.Code {
			current, credential, code := s.source(ctx, profile, d)
			if code == equipment.OK && (current.Company != st.Company || credential.Metadata.Revision != cr.Metadata.Revision) {
				return equipment.Changed
			}
			return code
		})
	}
	prior, c := s.Store.ConfigStatus(ctx, profile, d.ID)
	if c != equipment.OK {
		return
	}
	started := Status{At: s.Now().Format(time.RFC3339Nano), Code: equipment.Code("RUNNING"),
		CredentialRevision: cr.Metadata.Revision, Automatic: origin == Auto, Unchanged: prior.Unchanged}
	if code = allowed(func() equipment.Code { return s.Store.SaveConfig(ctx, profile, d, nil, started) }); code != equipment.OK {
		return
	}
	text, code := s.read(network, st, d, cr, check)
	status := Status{At: s.Now().Format(time.RFC3339Nano), Code: code, CredentialRevision: cr.Metadata.Revision,
		Automatic: origin == Auto, Unchanged: prior.Unchanged}
	// Never persist a result from a revoked session, replaced credential or company.
	result := allowed(func() equipment.Code {
		// ignored: only the result code or validation outcome is needed here.
		_, current, c := s.source(ctx, profile, d)
		if c != equipment.OK {
			return c
		}
		if current.Metadata.Revision != cr.Metadata.Revision {
			return equipment.Changed
		}
		return s.retain(ctx, st, d, origin, text, status)
	})
	if result != equipment.OK && ctx.Err() == nil {
		status.Code = result
		// ignored: the active-company guard may correctly refuse this diagnostic too.
		_ = allowed(func() equipment.Code { return s.Store.SaveConfig(ctx, profile, d, nil, status) })
	}
}
func (s *Service) source(ctx context.Context, profile string, d equipment.Device,
) (equipment.State, equipment.SealedCredential, equipment.Code) {
	st, code := s.Store.ReadEquipment(ctx, profile)
	if code != equipment.OK {
		return st, equipment.SealedCredential{}, code
	}
	current, ok := findDevice(st, d.ID)
	if !ok || current.Revision != d.Revision || current.Address != d.Address || current.Fingerprint != d.Fingerprint {
		return st, equipment.SealedCredential{}, equipment.Changed
	}
	cr, code := credential(st, current)
	return st, cr, code
}
func (s *Service) read(ctx context.Context, st equipment.State, d equipment.Device, cr equipment.SealedCredential,
	check func() equipment.Code) (string, equipment.Code) {
	secret, code := s.Vault.Open(st.Installation, st.Company, cr.Metadata.ID, cr.Metadata.Revision, cr.Envelope, st.WrappedKey)
	if code != equipment.OK {
		return "", equipment.VaultUnavailable
	}
	text, code := s.Remote.ReadConfig(ctx, d, cr.Metadata.User, secret, check)
	secret = ""
	if code != equipment.OK {
		return "", code
	}
	if len(text) > MaxBytes {
		return "", equipment.Limit
	}
	if strings.TrimSpace(text) == "" {
		return "", equipment.Partial
	}
	return text, equipment.OK
}
func (s *Service) retain(ctx context.Context, st equipment.State, d equipment.Device,
	origin Origin, text string, status Status) equipment.Code {
	if status.Code != equipment.OK {
		return s.Store.SaveConfig(ctx, st.Company, d, nil, status)
	}
	stamp, code := s.Vault.Stamp(Normalize(text))
	if code != equipment.OK {
		return equipment.VaultUnavailable
	}
	versions, code := s.Store.ConfigVersions(ctx, st.Company, d.ID)
	if code != equipment.OK {
		return code
	}
	if len(versions) > 0 && versions[0].Stamp == stamp {
		status.Unchanged = true
		return s.Store.SaveConfig(ctx, st.Company, d, nil, status)
	}
	status.Unchanged = false
	id, code := s.IDs.ID()
	if code != equipment.OK {
		return code
	}
	body, code := s.Vault.SealConfig(st.Installation, st.Company, id, text, st.WrappedKey)
	if code != equipment.OK {
		return equipment.VaultUnavailable
	}
	version := Stored{Version: Version{ID: id, DeviceID: d.ID, CreatedAt: status.At, Origin: origin,
		Format: FormatFor(d), Size: len(text), First: len(versions) == 0}, Stamp: stamp, Body: body}
	if len(versions) == 0 {
		version.Added = -1
	} else {
		previous, c := s.open(ctx, st, versions[0])
		if c != equipment.OK {
			return c
		}
		change := Diff(previous, text)
		version.Added, version.Removed = change.Added, change.Removed
	}
	return s.Store.SaveConfig(ctx, st.Company, d, &version, status)
}
func (s *Service) open(ctx context.Context, st equipment.State, v Stored) (string, equipment.Code) {
	stored, code := s.Store.ConfigVersion(ctx, st.Company, v.DeviceID, v.ID)
	if code != equipment.OK {
		return "", code
	}
	v.Body = stored.Body
	return s.Vault.OpenConfig(st.Installation, st.Company, v.ID, v.Body, st.WrappedKey)
}
