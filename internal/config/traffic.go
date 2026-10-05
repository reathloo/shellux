package config

// migrateTraffic folds the former standalone metrics into one entry. Loading
// only changes the in-memory config; saving persists the new component keys.
func migrateTraffic(visible, saved map[string]bool) {
	legacyFound, legacyEnabled, componentsConfigured := false, false, false
	parts := DisplayParts("network-traffic")
	for _, part := range parts {
		value, exists := saved[part]
		legacyFound = legacyFound || exists
		legacyEnabled = legacyEnabled || value
		_, configured := saved["network-traffic."+part]
		componentsConfigured = componentsConfigured || configured
	}
	if !legacyFound {
		return
	}
	trafficEnabled, configured := saved["network-traffic"]
	if !configured {
		trafficEnabled = true
	}
	for _, part := range parts {
		key := "network-traffic." + part
		if _, configured := saved[key]; !configured {
			value := part != "latency"
			if part == "latency" || (!trafficEnabled && legacyEnabled) {
				value = saved[part]
			}
			visible[key] = value
		}
		delete(visible, part)
	}
	if legacyEnabled && !componentsConfigured {
		visible["network-traffic"] = true
	}
}
