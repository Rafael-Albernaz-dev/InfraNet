package inventory

import (
	"context"
	"fmt"
	"strings"
)

func groupExists(groups []Group, id string) bool {
	if id == "" {
		return true
	}
	for _, group := range groups {
		if group.ID == id {
			return true
		}
	}
	return false
}

func (s *Service) handleGroups(ctx context.Context, q Command, v *View) Code {
	if q.Op == "inventory.groups" {
		state, code := s.Store.ReadEquipment(ctx, q.Scope.Profile)
		v.Groups = state.Groups
		return code
	}
	if !ValidID(q.Operation) || !ValidID(q.ID) {
		return Invalid
	}
	stamp := fmt.Sprintf("%s:%q:%q:%q", q.Op, q.ID, q.ParentID, q.Name)
	return s.Store.UpdateEquipment(ctx, q.Scope.Profile, func(state *State) Code {
		for _, operation := range state.Operations {
			if operation.ID == q.Operation {
				if operation.Stamp != stamp {
					return Revision
				}
				v.Groups = state.Groups
				return OK
			}
		}
		if len(state.Operations) >= MaxOperations {
			return Limit
		}
		if code := mutateGroup(state, q); code != OK {
			return code
		}
		state.Operations = append(state.Operations, Operation{ID: q.Operation, Stamp: stamp, Entity: q.ID})
		v.Groups = state.Groups
		return OK
	})
}

func mutateGroup(state *State, q Command) Code {
	index := -1
	for i, group := range state.Groups {
		if group.ID == q.ID {
			index = i
			break
		}
	}
	switch q.Op {
	case "inventory.group-create":
		if index >= 0 {
			return Revision
		}
		state.Groups = append(state.Groups, Group{ID: q.ID, Parent: q.ParentID, Name: q.Name})
	case "inventory.group-rename":
		if index < 0 {
			return Changed
		}
		state.Groups[index].Name = q.Name
	case "inventory.group-delete":
		if index < 0 {
			return Changed
		}
		for _, group := range state.Groups {
			if group.Parent == q.ID {
				return Invalid
			}
		}
		for _, device := range state.Devices {
			if device.GroupID == q.ID {
				return Invalid
			}
		}
		state.Groups = append(state.Groups[:index], state.Groups[index+1:]...)
	default:
		return Invalid
	}
	if !ValidGroups(state.Groups) {
		return Invalid
	}
	return OK
}

func (s *Service) ensureGroupPath(state *State, path string) (string, Code) {
	parts, valid := SplitGroupPath(path)
	if !valid {
		return "", Invalid
	}
	parent := ""
	for _, part := range parts {
		id := ""
		for _, group := range state.Groups {
			if group.Parent == parent && strings.EqualFold(group.Name, part) {
				id = group.ID
				break
			}
		}
		if id == "" {
			var code Code
			id, code = s.IDs.ID()
			if code != OK {
				return "", code
			}
			state.Groups = append(state.Groups, Group{ID: id, Parent: parent, Name: part})
		}
		parent = id
	}
	if !ValidGroups(state.Groups) {
		return "", Invalid
	}
	return parent, OK
}
