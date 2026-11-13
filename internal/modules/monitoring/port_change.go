package monitoring

import "strconv"

func PortStatusChanged(before, after *DevicePort) bool {
	if PortNegotiationChanged(before, after) {
		return true
	}
	if before == nil || after == nil || before.LinkDowns == nil || after.LinkDowns == nil {
		return false
	}
	a, ea := strconv.ParseUint(*before.LinkDowns, 10, 64)
	b, eb := strconv.ParseUint(*after.LinkDowns, 10, 64)
	return ea == nil && eb == nil && b > a
}
