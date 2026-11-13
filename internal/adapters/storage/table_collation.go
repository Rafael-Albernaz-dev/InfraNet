package storage

import (
	"database/sql/driver"
	"golang.org/x/text/collate"
	"golang.org/x/text/language"
	"modernc.org/sqlite"
	"strings"
	"sync"
)

func init() {
	// Collators reuse buffers; independent SQLite connections may call concurrently.
	var mu sync.Mutex
	order := collate.New(language.Portuguese, collate.IgnoreCase)
	sqlite.MustRegisterCollationUtf8("table_portuguese", func(a, b string) int {
		mu.Lock()
		defer mu.Unlock()
		return order.CompareString(a, b)
	})
	sqlite.MustRegisterDeterministicScalarFunction("table_lower", 1,
		func(_ *sqlite.FunctionContext, args []driver.Value) (driver.Value, error) {
			value, ok := args[0].(string)
			if !ok {
				return "", nil
			}
			return strings.ToLower(value), nil
		})
	sqlite.MustRegisterDeterministicScalarFunction("table_session_search", 1,
		func(_ *sqlite.FunctionContext, args []driver.Value) (driver.Value, error) {
			value, ok := args[0].(string)
			if !ok {
				return "", nil
			}
			return sessionTableSearch(value), nil
		})

}
