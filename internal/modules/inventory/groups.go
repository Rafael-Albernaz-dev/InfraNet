package inventory

import "infradesktop/internal/modules/equipment"

type Group = equipment.Group

func ValidGroups(groups []Group) bool            { return equipment.ValidGroups(groups) }
func GroupPath(groups []Group, id string) string { return equipment.GroupPath(groups, id) }
func GroupIncludes(groups []Group, selected, candidate string) bool {
	return equipment.GroupIncludes(groups, selected, candidate)
}
func SplitGroupPath(path string) ([]string, bool) { return equipment.SplitGroupPath(path) }
func FirstContactKey(saved, presented string) (string, Code) {
	return equipment.FirstContactKey(saved, presented)
}
func DisplayNames(devices []Device) map[string]string { return equipment.DisplayNames(devices) }

func projectDeviceNames(state *State) {
	names := DisplayNames(state.Devices)
	for i := range state.Devices {
		state.Devices[i].Name = names[state.Devices[i].ID]
		state.Devices[i].GroupPath = GroupPath(state.Groups, state.Devices[i].GroupID)
	}
}
