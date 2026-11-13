package equipment

import (
	"infradesktop/internal/modules/access"
	"strings"
)

type Group struct {
	ID     string `json:"id"`
	Parent string `json:"parent_id"`
	Name   string `json:"name"`
}

func ValidGroupName(name string) bool {
	return ValidText(name, 80) && !strings.ContainsAny(name, "›>")
}

func ValidGroups(groups []Group) bool {
	if len(groups) > 500 {
		return false
	}
	byID := map[string]Group{}
	for index, group := range groups {
		if !access.ValidCompanyID(group.ID) || !ValidGroupName(group.Name) || duplicateSibling(groups, index) {
			return false
		}
		// ignored: only the validation outcome is needed here.
		if _, exists := byID[group.ID]; exists {
			return false
		}
		byID[group.ID] = group
	}
	for _, group := range groups {
		depth := 1
		for parent := group.Parent; parent != ""; depth++ {
			ancestor, exists := byID[parent]
			if !exists || depth >= 4 {
				return false
			}
			parent = ancestor.Parent
		}
	}
	return true
}

func GroupPath(groups []Group, id string) string {
	parts := []string{}
	for depth := 0; id != "" && depth < 4; depth++ {
		found := false
		for _, group := range groups {
			if group.ID == id {
				parts = append([]string{group.Name}, parts...)
				id, found = group.Parent, true
				break
			}
		}
		if !found {
			return ""
		}
	}
	return strings.Join(parts, " › ")
}

func GroupIncludes(groups []Group, selected, candidate string) bool {
	if selected == "" {
		return candidate == ""
	}
	for depth := 0; depth < 4; depth++ {
		if selected == candidate {
			return true
		}
		if candidate == "" {
			return false
		}
		found := false
		for _, group := range groups {
			if group.ID == candidate {
				candidate, found = group.Parent, true
				break
			}
		}
		if !found {
			return false
		}
	}
	return false
}

func SplitGroupPath(path string) ([]string, bool) {
	if strings.TrimSpace(path) == "" {
		return nil, true
	}
	parts := strings.Split(strings.ReplaceAll(path, ">", "›"), "›")
	if len(parts) > 4 {
		return nil, false
	}
	for i := range parts {
		parts[i] = strings.TrimSpace(parts[i])
		if !ValidGroupName(parts[i]) {
			return nil, false
		}
	}
	return parts, true
}

func duplicateSibling(groups []Group, index int) bool {
	current := groups[index]
	for _, other := range groups[:index] {
		if other.Parent == current.Parent && strings.EqualFold(other.Name, current.Name) {
			return true
		}
	}
	return false
}
