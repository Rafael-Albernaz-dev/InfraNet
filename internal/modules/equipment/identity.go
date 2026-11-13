package equipment

func FirstContactKey(saved, presented string) (string, Code) {
	if presented == "" {
		return saved, TrustRequired
	}
	if saved != "" && saved != presented {
		return saved, TrustChanged
	}
	return presented, OK
}

func DisplayNames(devices []Device) map[string]string {
	counts := map[string]int{}
	for _, device := range devices {
		if device.Identity != "" {
			counts[device.Identity]++
		}
	}
	names := map[string]string{}
	for _, device := range devices {
		name := device.Identity
		if name == "" {
			name = device.Address
		} else if counts[name] > 1 {
			name += " (" + device.Address + ")"
		}
		names[device.ID] = name
	}
	return names
}
