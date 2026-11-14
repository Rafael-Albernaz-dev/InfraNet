package cfgbackup

import (
	"context"
	"infradesktop/internal/modules/equipment"
)

func (s *Service) compare(ctx context.Context, st equipment.State, d equipment.Device, id string, out *View) equipment.Code {
	versions, code := s.Store.ConfigVersions(ctx, st.Company, d.ID)
	if code != equipment.OK {
		return code
	}
	for i, v := range versions {
		if v.ID != id || i+1 >= len(versions) {
			continue
		}
		before, c := s.open(ctx, st, versions[i+1])
		if c != equipment.OK {
			return c
		}
		after, c := s.open(ctx, st, v)
		if c != equipment.OK {
			return c
		}
		change := Diff(before, after)
		change.From, change.To = versions[i+1].CreatedAt, v.CreatedAt
		out.Change = &change
		return equipment.OK
	}
	return equipment.Invalid
}
func (s *Service) download(ctx context.Context, q equipment.Command, st equipment.State, d equipment.Device, in Input) equipment.Code {
	if !in.Confirmed || in.Path == "" || s.Files == nil {
		return equipment.Invalid
	}
	versions, code := s.Store.ConfigVersions(ctx, st.Company, d.ID)
	if code != equipment.OK {
		return code
	}
	for _, v := range versions {
		if v.ID != in.VersionID {
			continue
		}
		text, c := s.open(ctx, st, v)
		if c != equipment.OK {
			return c
		}
		// Handle already holds the authorization lock. Publication stays inside that
		// same short transaction, so logout/company selection cannot interleave.
		return s.Files.WriteConfig(in.Path, v.Format, text, func(publish func() equipment.Code) equipment.Code {
			if ctx.Err() != nil {
				return equipment.Cancelled
			}
			return publish()
		})
	}
	return equipment.Invalid
}
