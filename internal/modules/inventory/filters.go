package inventory

import (
	"sort"
	"strings"

	"infradesktop/internal/modules/equipment"
)

func validateInventoryFilters(q Command) Code {
	if !validPulseFilter(q.State) {
		return Invalid
	}
	if q.Role != "" && !equipment.ValidInventoryRole(q.Role) ||
		q.GroupID != "" && !ValidID(q.GroupID) || q.GroupEmpty && q.GroupID != "" || len(q.Model) > 160 {
		return Invalid
	}
	return OK
}

func inventoryModel(d Device) string {
	if d.Result != nil && d.Result.Board != "" {
		return d.Result.Board
	}
	if d.RadioSummary != nil {
		return d.RadioSummary.Model
	}
	return ""
}

func inventoryModels(devices []Device) []string {
	models := map[string]bool{}
	for _, d := range devices {
		if model := inventoryModel(d); model != "" {
			models[model] = true
		}
	}
	return sortedOptions(models)
}

func sortedOptions(values map[string]bool) []string {
	result := make([]string, 0, len(values))
	for value := range values {
		result = append(result, value)
	}
	sort.Strings(result)
	return result
}

func filterDevices(devices []Device, q Command) []Device {
	filtered := []Device{}
	needle := strings.ToLower(q.Search)
	for _, d := range devices {
		if !inventoryMatches(d, q, needle) || !pulseMatches(d, q.State) {
			continue
		}
		if d.Result != nil {
			summary := *d.Result
			summary.Interfaces = nil
			d.Result = &summary
		}
		filtered = append(filtered, d)
	}
	return filtered
}

func inventoryMatches(d Device, q Command, needle string) bool {
	if q.Inactive && d.Active || !q.Inactive && !q.IncludeInactive && !d.Active {
		return false
	}
	if q.Role != "" && q.Role != d.Role || q.Model != "" && q.Model != inventoryModel(d) {
		return false
	}
	return strings.Contains(strings.ToLower(d.Name+" "+d.Address), needle)
}

func filterGroupDevices(devices []Device, groups []Group, q Command) []Device {
	if q.GroupID == "" && !q.GroupEmpty {
		return devices
	}
	filtered := []Device{}
	for _, d := range devices {
		if q.GroupEmpty && d.GroupID == "" || q.GroupID != "" && GroupIncludes(groups, q.GroupID, d.GroupID) {
			filtered = append(filtered, d)
		}
	}
	return filtered
}
