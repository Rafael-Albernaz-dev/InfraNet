package composition

import (
	"context"
	"infradesktop/internal/adapters/ipc"
	"infradesktop/internal/adapters/presentation"
	"infradesktop/internal/adapters/routerssh"
	"infradesktop/internal/adapters/storage"
	"infradesktop/internal/adapters/vault"
	"infradesktop/internal/modules/access"
	"infradesktop/internal/modules/cfgbackup"
)

func composeConfig(server *ipc.Server, auth *access.Service, store *storage.Store, provider vault.Provider) {
	service := cfgbackup.New(auth, store, vault.New(provider), routerssh.Adapter{}, presentation.ConfigFiles{}, equipmentIDs{})
	server.AttachConfig(service)
	server.StartBackground(func(ctx context.Context) {
		service.Run(ctx, func() (string, bool) { return auth.ActiveProfile(ctx) })
	})
}
