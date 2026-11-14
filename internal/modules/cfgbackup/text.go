// Package cfgbackup owns configuration history, redaction and daily scheduling.
package cfgbackup

import (
	"strings"
	"time"
)

var secretNames = []string{"password", "secret", "psk", "pre-shared-key", "passphrase", "private-key", "authentication-key"}

func Normalize(text string) string {
	end := strings.IndexByte(text, '\n')
	if end < 0 {
		return text
	}
	fields := strings.Fields(text[:end])
	if len(fields) != 6 || fields[0] != "#" || fields[3] != "by" || fields[4] != "RouterOS" {
		return text
	}
	// ignored: only the result code or validation outcome is needed here.
	if _, err := time.Parse("15:04:05", fields[2]); err != nil {
		return text
	}
	valid := false
	for _, layout := range []string{"2006-01-02", "Jan/2/2006"} {
		// ignored: only the result code or validation outcome is needed here.
		if _, err := time.Parse(layout, fields[1]); err == nil {
			valid = true
		}
	}
	if valid {
		return text[end+1:]
	}
	return text
}

func secretKey(key string) bool {
	for _, name := range secretNames {
		if strings.Contains(strings.ToLower(key), name) {
			return true
		}
	}
	return false
}

// Mask parses values instead of matching up to the next space: quoted secrets
// may contain escaped quotes, newlines and RouterOS line continuations.
func Mask(text string) string {
	var out strings.Builder
	for len(text) > 0 {
		loc := assignmentLocation(text)
		if loc == nil {
			out.WriteString(text)
			break
		}
		end := loc[1]
		out.WriteString(text[:end])
		key := text[loc[0] : loc[1]-1]
		text = text[end:]
		n := valueEnd(text)
		if secretKey(key) && strings.Contains(key, ".") {
			n = airValueEnd(text)
		}
		if secretKey(key) {
			out.WriteString("********")
		} else {
			out.WriteString(text[:n])
		}
		text = text[n:]
	}
	return out.String()
}
func valueEnd(text string) int {
	quote := byte(0)
	if len(text) > 0 && (text[0] == '"' || text[0] == '\'') {
		quote = text[0]
	}
	for i := 0; i < len(text); i++ {
		if text[i] == '\\' && i+1 < len(text) {
			i = escapedValueEnd(text, i)
			continue
		}
		if quote != 0 {
			if i > 0 && text[i] == quote {
				return i + 1
			}
			continue
		}
		if strings.ContainsRune(" \t\r\n", rune(text[i])) {
			return i
		}
	}
	return len(text)
}

type DiffLine struct {
	Kind string `json:"kind"`
	Text string `json:"text"`
}
type Change struct {
	From      string     `json:"from,omitempty"`
	To        string     `json:"to,omitempty"`
	Lines     []DiffLine `json:"lines"`
	Added     int        `json:"added"`
	Removed   int        `json:"removed"`
	Truncated bool       `json:"truncated"`
}

func lines(text string) []string {
	if text == "" {
		return nil
	}
	return strings.Split(strings.TrimSuffix(text, "\n"), "\n")
}

// The comparison has a bounded LCS workspace. Large, wholly rewritten exports
// use a complete replacement hunk instead of allocating quadratic memory.
func Diff(before, after string) Change {
	a, b := lines(Normalize(before)), lines(Normalize(after))
	raw := diffLines(a, b)
	c := Change{Lines: []DiffLine{}}
	// Mask the complete hunk, including continuation lines and quoted newlines.
	texts := make([]string, len(raw))
	for i, l := range raw {
		texts[i] = l.Text
	}
	masked := lines(Mask(strings.Join(texts, "\n")))
	// A multiline value can collapse lines. Mask those entire hunks conservatively.
	collapse := len(masked) != len(raw)
	bytes := 0
	for i, l := range raw {
		if l.Kind == "add" {
			c.Added++
		}
		if l.Kind == "remove" {
			c.Removed++
		}
		if collapse {
			l.Text = "********"
		} else {
			l.Text = masked[i]
		}
		if c.Truncated || len(c.Lines) >= 2000 || bytes+len(l.Text) > 256*1024 {
			c.Truncated = true
			continue
		}
		bytes += len(l.Text)
		c.Lines = append(c.Lines, l)
	}
	return c
}
func diffLines(a, b []string) []DiffLine {
	out := []DiffLine{}
	for len(a) > 0 && len(b) > 0 && a[0] == b[0] {
		out = append(out, DiffLine{"same", a[0]})
		a, b = a[1:], b[1:]
	}
	suffix := 0
	for suffix < len(a) && suffix < len(b) && a[len(a)-1-suffix] == b[len(b)-1-suffix] {
		suffix++
	}
	tail := a[len(a)-suffix:]
	a, b = a[:len(a)-suffix], b[:len(b)-suffix]
	out = append(out, diffMiddle(a, b)...)
	for _, l := range tail {
		out = append(out, DiffLine{"same", l})
	}
	return out
}
func diffMiddle(a, b []string) []DiffLine {
	out := []DiffLine{}
	if len(a)*len(b) > 1000000 {
		for _, l := range a {
			out = append(out, DiffLine{"remove", l})
		}
		for _, l := range b {
			out = append(out, DiffLine{"add", l})
		}
		return out
	}
	width := len(b) + 1
	dp := make([]int, (len(a)+1)*width)
	for i := len(a) - 1; i >= 0; i-- {
		for j := len(b) - 1; j >= 0; j-- {
			if a[i] == b[j] {
				dp[i*width+j] = 1 + dp[(i+1)*width+j+1]
			} else {
				dp[i*width+j] = max(dp[(i+1)*width+j], dp[i*width+j+1])
			}
		}
	}
	i, j := 0, 0
	for i < len(a) || j < len(b) {
		switch {
		case i < len(a) && j < len(b) && a[i] == b[j]:
			out = append(out, DiffLine{"same", a[i]})
			i++
			j++
		case i < len(a) && (j == len(b) || dp[(i+1)*width+j] >= dp[i*width+j+1]):
			out = append(out, DiffLine{"remove", a[i]})
			i++
		default:
			out = append(out, DiffLine{"add", b[j]})
			j++
		}
	}
	return out
}

func assignmentLocation(text string) []int {
	end := strings.IndexByte(text, '=')
	if end < 0 {
		return nil
	}
	start := end
	for start > 0 && !strings.ContainsRune(" \t\r\n", rune(text[start-1])) {
		start--
	}
	return []int{start, end + 1}
}

func airValueEnd(text string) int {
	end := strings.IndexByte(text, '\n')
	if end < 0 {
		return len(text)
	}
	return end
}
func escapedValueEnd(text string, index int) int {
	index++
	if text[index] == '\r' && index+1 < len(text) && text[index+1] == '\n' {
		index++
	}
	if text[index] == '\n' {
		for index+1 < len(text) && (text[index+1] == ' ' || text[index+1] == '\t') {
			index++
		}
	}
	return index
}
