package storage

import (
	"encoding/json"
	"infradesktop/internal/modules/equipment"
	"strconv"
	"strings"
)

// Search follows the safe row projection shown by the sessions table, including
// hidden columns. No credential or remote command is involved.
func sessionTableSearch(body string) string {
	var row equipment.PPRow
	if json.Unmarshal([]byte(body), &row) != nil {
		return ""
	}
	if row.State == "" {
		row.State = "ONLINE"
	}
	parts := []string{row.Login, row.Address, row.Caller, row.SessionID, row.State}
	if row.State != "OFFLINE" && row.Uptime.Known {
		parts = append(parts, tableDuration(row.Uptime.Min))
	}
	if row.Context != nil {
		c := row.Context
		parts = append(parts, c.Interface, c.Service, c.DownloadLimit, c.UploadLimit)
		if c.SecretState == "active" || c.SecretState == "disabled" {
			parts = append(parts, c.Profile)
		}
		down, downUnit := tableRate(c.DownloadLimit)
		up, upUnit := tableRate(c.UploadLimit)
		parts = append(parts, down+" "+downUnit, up+" "+upUnit)
		if downUnit == upUnit {
			parts = append(parts, down+" / "+up+" "+downUnit)
		}
		if c.VLAN != nil {
			parts = append(parts, strconv.Itoa(*c.VLAN))
		}
	}
	return strings.Join(parts, " ")
}
func tableDuration(ns int64) string {
	seconds := ns / 1e9
	var compact, spaced []string
	for _, unit := range []struct {
		name string
		size int64
	}{{"d", 86400}, {"h", 3600}, {"min", 60}, {"s", 1}} {
		count := seconds / unit.size
		seconds %= unit.size
		if count > 0 || unit.size == 1 && len(compact) == 0 {
			value := strconv.FormatInt(count, 10)
			compact = append(compact, value+unit.name)
			spaced = append(spaced, value+" "+unit.name)
		}
		if len(compact) == 2 {
			break
		}
	}
	return strings.Join(compact, " ") + " " + strings.Join(spaced, " ")
}
func tableRate(raw string) (string, string) {
	value, err := strconv.ParseFloat(raw, 64)
	if err != nil {
		return "Não informado nesta leitura", ""
	}
	units := []string{"bps", "Kbps", "Mbps", "Gbps", "Tbps"}
	index := 0
	for value >= 1000 && index < len(units)-1 {
		value /= 1000
		index++
	}
	digits := 0
	if value < 10 {
		digits = 2
	} else if value < 100 {
		digits = 1
	}
	number := strconv.FormatFloat(value, 'f', digits, 64)
	if strings.Contains(number, ".") {
		number = strings.TrimRight(strings.TrimRight(number, "0"), ".")
	}
	return strings.ReplaceAll(number, ".", ","), units[index]
}
