package routerssh

import (
	"context"
	"infradesktop/internal/adapters/routerssh/pppoeparse"
	"infradesktop/internal/modules/equipment"
	"time"
)

func readPPPCatalog(ctx context.Context, options ppReadOptions, check func() equipment.Code) (*equipment.PPReadResult, equipment.Code) {
	start := time.Now()
	connection, code := openPPConnection(ctx, options.device, options.user, options.secret, check, equipment.QueryBudget)
	options.secret = ""
	if code != equipment.OK {
		return nil, code
	}
	defer connection.close()
	connection.catalogBudget()
	text, code := connection.run(pppoeparse.PPPCatalogCommand)
	if code != equipment.OK {
		return nil, code
	}
	rows, limited, code := pppoeparse.ParsePPPCatalog(text)
	if code != equipment.OK {
		return nil, code
	}
	if code = check(); code != equipment.OK {
		return nil, code
	}
	return ppMeasureRead(&equipment.PPReadResult{Scope: equipment.PPPCatalogScope, Started: equipment.PPTime(start),
		Ended: equipment.PPTime(time.Now()), Rows: rows, CatalogLimited: limited})
}
