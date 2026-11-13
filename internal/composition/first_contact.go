package composition

import (
	"context"
	"infradesktop/internal/adapters/ipc"
	"infradesktop/internal/modules/access"
	"infradesktop/internal/modules/equipment"
	"infradesktop/internal/modules/inventory"
	"infradesktop/internal/modules/pulse"
	"sync"
	"time"
)

func composeFirstContacts(server *ipc.Server, auth *access.Service, inv *inventory.Service, ping *pulse.Collector) {
	server.StartBackground(func(ctx context.Context) {
		ticker := time.NewTicker(5 * time.Second)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case now := <-ticker.C:
				firstContactRound(ctx, auth, inv, ping, now)
			}
		}
	})
}

func firstContactRound(ctx context.Context, auth *access.Service, inv *inventory.Service, ping *pulse.Collector, now time.Time) {
	profile, ok := auth.ActiveProfile(ctx)
	if !ok {
		return
	}
	snapshot, ok := ping.Snapshot(profile, now)
	if !ok {
		return
	}
	state, code := inv.Store.ReadEquipment(ctx, profile)
	if code != equipment.OK {
		return
	}
	revisions := map[string]uint64{}
	for _, cr := range state.Credentials {
		revisions[cr.Metadata.ID] = cr.Metadata.Revision
	}
	var workers sync.WaitGroup
	slots := make(chan struct{}, 2)
	for _, d := range state.Devices {
		revision := revisions[d.CredentialID]
		if revision == 0 || snapshot.Devices[d.ID].State != pulse.Online || !inv.AutomaticContactDue(profile, d, revision, now) {
			continue
		}
		select {
		case slots <- struct{}{}:
		case <-ctx.Done():
			workers.Wait()
			return
		}
		workers.Add(1)
		go func(d equipment.Device) {
			defer workers.Done()
			defer func() { <-slots }()
			active := func() bool {
				current, ok := auth.ActiveProfile(ctx)
				if !ok || current != profile {
					return false
				}
				snapshot, ok := ping.Snapshot(profile, time.Now())
				return ok && snapshot.Devices[d.ID].State == pulse.Online
			}
			// ignored: the durable attempt contains the network outcome; a changed company abandons this round.
			_ = inv.AutomaticContact(ctx, profile, d, active)
		}(d)
	}
	workers.Wait()
}
