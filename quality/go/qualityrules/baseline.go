package qualityrules

import "strings"

// Limits every authored Go/TS source meets once it carries no debt.
const (
	FunctionLimit   = 100
	ComplexityLimit = 15
)

// FileLimit is shared by the baseline exception list and changed-file validation.
func FileLimit(path string) int {
	if strings.HasSuffix(path, ".tsx") {
		return 600
	}
	return 500
}
