package composition

import (
	"context"
	"time"

	"infradesktop/internal/adapters/importfile"
	"infradesktop/internal/adapters/ipc"
	"infradesktop/internal/adapters/routerssh"
	"infradesktop/internal/adapters/storage"
	"infradesktop/internal/adapters/vault"
	"infradesktop/internal/modules/access"
	"infradesktop/internal/modules/equipment"
	"infradesktop/internal/modules/inventory"
	"infradesktop/internal/modules/monitoring"
	"infradesktop/internal/modules/pppoe"
	"infradesktop/internal/modules/pulse"
	"infradesktop/internal/modules/radio"
)

func composeEquipment(
	service *access.Service, store *storage.Store, provider vault.Provider, collector *pulse.Collector, reader *pulse.ReadCollector,
	server ...*ipc.Server,
) *equipment.Service {
	mon := monitoring.New(service, store)
	inv := inventory.New(service, store, vault.New(provider), routerssh.Adapter{}, equipmentIDs{}, time.Now)
	inv.Pulse = collector
	rad := radio.New(inv)
	if len(server) > 0 && reader == nil {
		reader = composeLiveReader(server[0], liveInventory{access: service, store: store, follow: rad.FollowEnlaceSessions},
			vault.New(provider), collector, inv)
	}
	inv.Live = reader
	inv.Parser, inv.Digester = importfile.Parser{}, importfile.Parser{}
	if len(server) > 0 {
		composeFirstContacts(server[0], service, inv, collector)
		server[0].StartBackground(func(ctx context.Context) {
			rad.RunEnlace(ctx, func() (string, bool) { return service.ActiveProfile(ctx) })
		})
	}
	return &equipment.Service{Handler: radio.NewComposite(inv, pppoe.New(inv, mon), rad, mon)}
}
