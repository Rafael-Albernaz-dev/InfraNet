package cfgbackup

import (
	"context"
	"infradesktop/internal/modules/equipment"
	"sync"
	"time"
)

func (s *Service) Run(ctx context.Context, active func() (string, bool)) {
	defer s.Close()
	ticker := time.NewTicker(time.Minute)
	defer ticker.Stop()
	for {
		s.Round(ctx, active)
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
		}
	}
}
func (s *Service) Round(ctx context.Context, active func() (string, bool)) {
	if !Due(s.Now(), "") {
		return
	}
	profile, ok := active()
	if !ok {
		return
	}
	st, code := s.Store.ReadEquipment(ctx, profile)
	if code != equipment.OK {
		return
	}
	var workers sync.WaitGroup
	for _, d := range st.Devices {
		if ctx.Err() != nil {
			break
		}
		// ignored: only the result code or validation outcome is needed here.
		if _, code := credential(st, d); code != equipment.OK {
			continue
		}
		status, c := s.Store.ConfigStatus(ctx, profile, d.ID)
		if c != equipment.OK || !Due(s.Now(), status.AutoAt) {
			continue
		}
		if !s.claim(profile, d.ID) {
			workers.Wait()
			if !s.claim(profile, d.ID) {
				continue
			}
		}
		workers.Add(1)
		go func(d equipment.Device) {
			defer workers.Done()
			defer s.release(profile, d.ID)
			allowed := func(fn func() equipment.Code) equipment.Code {
				current, ok := active()
				if !ok || current != profile || ctx.Err() != nil {
					return equipment.Changed
				}
				return fn()
			}
			s.capture(ctx, profile, d, Auto, allowed)
		}(d)
	}
	workers.Wait()
}
