package inventory

func blockUpdateKeptRevision(q Command, v *View) bool {
	return q.Op == "inventory.update" && v.Device != nil && v.Device.Revision == q.Revision
}

func deviceUpdateRevision(next, old Device, operation string) uint64 {
	if operation == "inventory.update" && next.GroupID != old.GroupID && sameDeviceConfiguration(next, old) {
		return old.Revision
	}
	return old.Revision + 1
}

func sameDeviceConfiguration(a, b Device) bool {
	return a.Role == b.Role && a.Address == b.Address &&
		a.Port == b.Port && a.Transport == b.Transport && a.CredentialID == b.CredentialID
}
