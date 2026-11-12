// quality measures authored sources and checks touched files against limits and static baseline exceptions.
package main

import (
	"encoding/json"
	"fmt"
	"infradesktop/quality/go/qualitymetrics"
	"infradesktop/quality/go/qualityrules"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

type metric = qualitymetrics.Metric
type report = qualitymetrics.Report
type span = qualityrules.Span

// Slice codes are not behavior: a touched test named after one must be renamed.
var forbiddenTestFunction = regexp.MustCompile(`(?i:L0[0-9]|BF0[0-9]|Avulso|Live2|Piloto|CPU90)|Focal`)
var forbidden = regexp.MustCompile(`(?i)(^|_)(j|live2|piloto|cpu90|avulso|l06c|l06d|bf03|bf04)(_|$)`)

func main() {
	measured, err := qualitymetrics.MeasureSources()
	fatal(err)
	raw, err := os.ReadFile("quality/baselines/quality-baseline.json")
	fatal(err)
	var previous report
	fatal(json.Unmarshal(raw, &previous))
	changed, err := qualityrules.ChangedSources()
	fatal(err)
	failures := validate(measured, previous, changed)
	for _, failure := range failures {
		fmt.Println(failure)
	}
	if len(failures) > 0 {
		fmt.Printf("quality: BLOCK (%d findings)\n", len(failures))
		os.Exit(1)
	}
	fmt.Println("quality: PASS")
}

func fatal(err error) {
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(2)
	}
}

func overlaps(start, end int, changed []span) bool { return qualityrules.Overlaps(start, end, changed) }

func validate(current, previous report, changed map[string][]span) []string {
	var failures []string
	for path, m := range current {
		if old, exists := previous[path]; exists {
			failures = append(failures, validateBaseline(path, m, old)...)
		}
		if spans, touched := changed[path]; touched {
			failures = append(failures, validateChanged(path, m, spans)...)
		}
	}
	return failures
}

func validateBaseline(path string, current, previous metric) []string {
	within := func(now, before, limit int) bool { return now <= before || now <= limit }
	if within(current.Lines, previous.Lines, qualityrules.FileLimit(path)) && current.LongLines <= previous.LongLines &&
		within(current.MaxFunction, previous.MaxFunction, qualityrules.FunctionLimit) &&
		within(current.MaxComplexity, previous.MaxComplexity, qualityrules.ComplexityLimit) {
		return nil
	}
	return []string{fmt.Sprintf("%s: baseline ceiling exceeded: lines %d→%d long %d→%d function %d→%d complexity %d→%d",
		path, previous.Lines, current.Lines, previous.LongLines, current.LongLines,
		previous.MaxFunction, current.MaxFunction, previous.MaxComplexity, current.MaxComplexity)}
}

func validateChanged(path string, m metric, spans []span) []string {
	var failures []string
	if strings.HasSuffix(path, ".go") {
		content, err := os.ReadFile(path)
		if err != nil {
			return []string{err.Error()}
		}
		failures = append(failures, qualityrules.GoLimits(path, content, spans)...)
	}
	if legacyExceptions[path] {
		spans = radioFunctionSpans(m, spans)
	} else {
		limit := qualityrules.FileLimit(path)
		if (strings.HasSuffix(path, ".go") || strings.HasSuffix(path, ".tsx")) && m.Lines > limit {
			failures = append(failures, fmt.Sprintf("%s: QG2 file %d > %d", path, m.Lines, limit))
		}
	}
	if len(spans) == 0 {
		return failures
	}
	failures = append(failures, validateFunctions(path, m, spans)...)
	return append(failures, validateText(path, spans)...)
}

// Explicit static exception: only these legacy files keep their original ceiling.
// Every radio file and named radio helper stays strict.
var legacyExceptions = map[string]bool{
	"frontend/src/app/pages/equipment.tsx":              true,
	"frontend/src/app/pages/equipment.test.tsx":         true,
	"internal/modules/equipment/service.go":             true,
	"internal/adapters/storage/equipment.go":            true,
	"internal/adapters/storage/units.go":                true,
	"internal/adapters/storage/device_insights_test.go": true,
	"internal/testsupport/sshfixture/server.go":         true,
	"internal/adapters/presentation/equipment.go":       true,
}

func radioFunctionSpans(m metric, changed []span) []span {
	var strict []span
	for _, f := range m.Functions {
		if strings.Contains(strings.ToLower(f.Name), "radio") && overlaps(f.Start, f.End, changed) {
			strict = append(strict, span{Start: f.Start, End: f.End})
		}
	}
	return strict
}

func validateFunctions(path string, m metric, spans []span) []string {
	failures := []string{}
	if strings.HasSuffix(path, "_test.go") && forbidden.MatchString(strings.TrimSuffix(filepath.Base(path), ".go")) {
		failures = append(failures, path+": QG5 dispatch name")
	}
	for _, f := range m.Functions {
		if !overlaps(f.Start, f.End, spans) {
			continue
		}
		if f.Lines > qualityrules.FunctionLimit || f.Complexity > qualityrules.ComplexityLimit {
			failures = append(failures, fmt.Sprintf("%s:%d: QG2/QG3 %s lines=%d complexity=%d", path, f.Start, f.Name, f.Lines, f.Complexity))
		}
		if strings.HasPrefix(f.Name, "Test") && forbiddenTestFunction.MatchString(f.Name) {
			failures = append(failures, fmt.Sprintf("%s:%d: QG5 %s", path, f.Start, f.Name))
		}
	}
	return failures
}

func validateText(path string, spans []span) []string { return qualityrules.Text(path, spans) }
