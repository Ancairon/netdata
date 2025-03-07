package main

import (
	"fmt"
	"log"
	"time"

	"github.com/davecgh/go-spew/spew"
	"github.com/gosnmp/gosnmp"
)

func main() {
	profileDir := "profiles/"
	// profileDir := "../snmp_profiles_dump/"
	subnet := "20.20.21.0/24" // CHANGE THIS TO YOUR SUBNET
	community := "public"
	timeout := 2 * time.Second

	// Load all profiles
	profiles, err := LoadAllProfiles(profileDir)
	if err != nil {
		log.Fatalf("Failed to load profiles: %v", err)
	}

	devices := ScanSubnet(subnet, community, timeout)

	if len(devices) == 0 {
		log.Fatal("No active SNMP devices found in subnet")
	}

	// Iterate over discovered devices
	for _, deviceIP := range devices {
		snmp := &gosnmp.GoSNMP{
			Target:    deviceIP,
			Port:      161,
			Community: "public",
			Version:   gosnmp.Version2c,
			Timeout:   time.Duration(5) * time.Second,
			Retries:   3,
		}

		err = snmp.Connect()
		if err != nil {
			log.Fatalf("SNMP Connection failed: %v", err)
		}
		defer snmp.Conn.Close()

		// fmt.Println("Fetching sysObjectID...")
		sysObjectID, err := GetSysObjectID(snmp)
		if err != nil {
			log.Printf("Failed to get sysObjectID for %s: %v\n", deviceIP, err)
			continue
		}
		fmt.Printf("Device sysObjectID: %s\n", sysObjectID)

		matchingProfiles := FindMatchingProfiles(profiles, sysObjectID)
		if len(matchingProfiles) == 0 {
			log.Printf("No matching profile found for sysObjectID: %s", sysObjectID)
		}

		metricMap := map[string]processedMetric{}

		for _, profile := range matchingProfiles {
			// fmt.Println("Profile:", name)
			// fmt.Println("Profile Metrics")
			// spew.Dump(profile.Metrics)
			// fmt.Print("\n\n\n")

			results := parseMetrics(profile.Metrics)
			// fmt.Println(parseMetrics(profile.Metrics))

			for _, oid := range results.oids {
				// fmt.Println("OID:", oid)

				response, err := SNMPGet(deviceIP, oid, "public")
				if err != nil {
					log.Fatalf("SNMP Exec failed: %v", err)
				}

				if (response != snmpPDU{}) {

					// fmt.Println(response)

					for _, metric := range results.parsed_metrics {
						switch s := metric.(type) {
						case parsedSymbolMetric:
							// fmt.Println("parsedsymbolmetric")

							if s.baseoid == oid {
								fmt.Println("FOUND MATCH", s, response)

								metricName := s.name
								metricType := response.metric_type
								metricValue := response.value

								fmt.Printf("METRIC: %s | %s | %s\n", metricName, metricType, metricValue)

								metricMap[oid] = processedMetric{oid: oid, name: metricName, value: metricValue, metric_type: metricType}
							}
						}
					}

				}
			}

			for _, oid := range results.next_oids {

				spew.Dump(oid)

				if len(oid) < 1 {
					fmt.Println("empty OID, skipping", oid)
					continue
				}
				if tableRows, err := walkOIDTree(deviceIP, community, oid); err != nil {
					log.Fatalf("Error walking OID tree: %v, oid %s", err, oid)
				} else {
					for _, metric := range results.parsed_metrics {
						switch s := metric.(type) {
						case parsedTableMetric:
							if s.rowOID == oid {
								// fmt.Println("FOUND MATCH", s)


								fmt.Println(tableRows)

								for key, value := range tableRows {
									value.name = s.name
									value.tableName = s.tableName
									// fmt.Println(value)
									tableRows[key] = value

								}


								metricMap = mergeProcessedMetricMaps(metricMap, tableRows)
							}
							// case parsedSymbolMetric:
							// 	fmt.Println(s,oid)
							// 	if s.baseoid == oid {
							// 	}
						}
					}
				}

			}

			for _, value := range metricMap {
				// fmt.Println(value)
				if value.tableName == "" {

					spew.Dump(value)
				}
			}
		}
	}
}
