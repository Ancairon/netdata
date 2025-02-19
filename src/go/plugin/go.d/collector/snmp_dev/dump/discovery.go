// discovery.go
// (C) Datadog, Inc. 2020-present (translated)
// All rights reserved
//
// This file contains the discovery routine that loops over a subnet to discover devices.
// It’s intended to run in its own goroutine.

package snmp_dev

// // Logger is a placeholder interface representing logging functionality.
// type Logger interface {
// 	Debugf(format string, args ...interface{})
// 	Warningf(format string, args ...interface{})
// }

// // writePersistentCache is a stub for writing persistent cache data.
// // You can implement this to write to a file or other persistent store.
// func writePersistentCache(key, value string) {
// 	// Implement persistent cache write here.
// 	// For now, this is a no-op.
// }

// // getKeys extracts the keys from a map[string]*InstanceConfig.
// func getKeys(m map[string]*InstanceConfig) []string {
// 	keys := make([]string, 0, len(m))
// 	for k := range m {
// 		keys = append(keys, k)
// 	}
// 	return keys
// }

// // DiscoverInstances continuously scans network hosts to discover devices.
// // - config: the instance configuration (from which to get the subnet hosts).
// // - interval: the time between scans.
// // - checkRef: a function that returns a pointer to SnmpCheck (or nil when no longer valid).
// func DiscoverInstances(config *InstanceConfig, interval time.Duration, checkRef func() *SnmpCheck) {
// 	for {
// 		startTime := time.Now()

// 		hosts, err := config.NetworkHosts()
// 		if err != nil {
// 			// If you cannot obtain hosts, wait and try again.
// 			time.Sleep(interval)
// 			continue
// 		}

// 		for _, host := range hosts {
// 			check := checkRef()
// 			if check == nil || !check.running {
// 				return
// 			}

// 			// Build a host-specific configuration.
// 			hostConfig, err := check.buildAutodiscoveryConfig(config.Instance, host)
// 			if err != nil {
// 				check.log.Debugf("Error building autodiscovery config for host %s: %v", host, err)
// 				continue
// 			}

// 			// Fetch sysObjectID (or similar identifier).
// 			sysObjectOid, err := check.fetchSysobjectOid(hostConfig)
// 			if err != nil {
// 				check.log.Debugf("Error scanning host %s: %v", host, err)
// 				continue
// 			}

// 			// Determine profile based on sysObjectOid.
// 			profile, err := check.profileForSysobjectOid(sysObjectOid)
// 			if err != nil {
// 				// If a configuration error occurred and no OIDs are defined, log a warning and skip.
// 				if !hostConfig.OidConfig.HasOids() {
// 					check.log.Warningf("Host %s didn't match a profile for sysObjectID %s", host, sysObjectOid)
// 					continue
// 				}
// 			} else {
// 				// Refresh the host configuration with profile-specific settings.
// 				if err := hostConfig.RefreshWithProfile(check.profiles[profile]); err != nil {
// 					check.log.Debugf("Error refreshing profile for host %s: %v", host, err)
// 					continue
// 				}
// 				hostConfig.AddProfileTag(profile)
// 			}

// 			// Add the discovered host configuration.
// 			config.DiscoveredInstances[host] = hostConfig

// 			// Write the discovered hosts to persistent cache.
// 			keys := getKeys(config.DiscoveredInstances)
// 			if keysJSON, err := json.Marshal(keys); err == nil {
// 				writePersistentCache(check.checkID, string(keysJSON))
// 			}
// 		}

// 		// At the end of the loop, write the discovered instances again.
// 		check := checkRef()
// 		if check == nil {
// 			return
// 		}
// 		keys := getKeys(config.DiscoveredInstances)
// 		if keysJSON, err := json.Marshal(keys); err == nil {
// 			writePersistentCache(check.checkID, string(keysJSON))
// 		}

// 		// Sleep for the remaining interval (if any) before the next scan.
// 		if sleepDuration := interval - time.Since(startTime); sleepDuration > 0 {
// 			time.Sleep(sleepDuration)
// 		}
// 	}
// }
