package main

import (
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	"github.com/davecgh/go-spew/spew"
	"github.com/gosnmp/gosnmp"
)

func main() {
	profileDir := "../snmp_profiles/default_profiles/"
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

		fmt.Println("Fetching sysObjectID...")
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
			fmt.Println("Profile Metrics")
			spew.Dump(profile.Metrics)
			fmt.Print("\n\n\n")

			results := parseMetrics(profile.Metrics)
			// fmt.Println(parseMetrics(profile.Metrics))
			
			for _, oid := range results.oids {
				fmt.Println("OID:",oid)


				response, err := SNMPGetExec(deviceIP, oid, "public")
				if err != nil {
					log.Fatalf("SNMP Exec failed: %v", err)
				}

				if len(response) > 0 {
					fmt.Println(response)

					for _, metric := range results.parsed_metrics {
						switch s := metric.(type) {
						case ParsedSymbolMetric:
							// fmt.Println("parsedsymbolmetric")
							
							if s.baseoid == oid {
								fmt.Println("FOUND MATCH", s)

								metricName := s.Name
								metricSplit := strings.SplitN(strings.SplitN(response, " = ", 2)[1], ": ", 2)
								if len(metricSplit) < 2 {
									fmt.Print(metricSplit)
									os.Exit(-9324)
								}
								metricType := metricSplit[0]
								metricValue := metricSplit[1]

								fmt.Printf("METRIC: %s | %s | %s\n", metricName, metricType, metricValue)
								
								metricMap[oid] = processedMetric{OID: oid,name: metricName, Value: metricValue, metric_type: metricType}
							}
						}}
					
			}

			// walk OID subtree
			for _, oid := range results.next_oids {
				fmt.Println("IN THE LOOP")
				// tables
				if tableRows, err := walkOIDTree(deviceIP, community, oid); err != nil {
					log.Fatalf("Error walking OID tree: %v", err)
				} else {
					fmt.Println(tableRows)

					for _, metric := range results.parsed_metrics {
						switch s := metric.(type) {
						// case ParsedSymbolMetric:
						// 	// fmt.Println("parsedsymbolmetric")
						case ParsedTableMetric:
							// fmt.Println("parsedtablemetric")
							if s.baseoid == oid {
								fmt.Println("FOUND MATCH", s)

								for key, value := range tableRows {
									value.name = s.Name
									fmt.Println(value)
									tableRows[key] = value
								}

								// fmt.Println(tableRows)
								metricMap = mergeProcessedMetricMaps(metricMap, tableRows)
								// os.Exit(16)
							}
						default:
							fmt.Println("NONE OF THE TWO", s)
						}
					}
				}
			}

			for _, value := range metricMap {
				fmt.Println(value)
			}
			// 	if metric.Symbol != nil {
			// 		// we have a symbol metric

			// 		// parseSymbolMetric()

			// 		// continue
			// 		
			// 		// }
			// 	} else if metric.Table != nil {
			// 		for _, symbol := range metric.Symbols {
			// 			if len(symbol.OID) > 1 {
			// 				// if it is a table we do a walk instead of a get
			// 				response, err := SNMPWalkExec(deviceIP, symbol.OID, "public")
			// 				if err != nil {
			// 					log.Fatalf("SNMP Exec failed: %v", err)
			// 				}

			// 				if len(response) > 0 {
			// 					// iterate through the response

			// 					for _, response := range response {

			// 						metric_type := response[0]
			// 						metric_value := response[1]

			// 						fmt.Print(metric_type, metric_value, "\n")

			// 						fmt.Printf("METRIC: %s | %s | %s\n", symbol.Name, metric_type, metric_value)

			// 					}
			// 					os.Exit(129)
			// 				}
			// 			}
			// 		}
			// 		// fmt.Print("something")
			// 	}
			// if metric.Symbol != nil {
			// 	val, ok := deviceData[metric.Symbol.OID]
			// 	// If the key exists
			// 	if ok {
			// 		// Do something
			// 		// you have the symbol found here

			// 		metricName := metric.Symbol.Name
			// 		metricSplit := strings.SplitN(val, ": ", 2)
			// 		metricType := metricSplit[0]
			// 		metricValue := metricSplit[1]

			// 		fmt.Printf("METRIC: %s | %s | %s\n", metricName, metricType, metricValue)

			// 	}
			// 	// for key := range deviceData {
			// 	// 	if strings.Contains(key, metric.Symbol.OID) {
			// 	// 		fmt.Println("Found:", key)
			// 	// 	}
			// } else if metric.Table != nil {
			// 	fmt.Print("TABLE FOUND\n")
			// 	metricName := metric.Symbol.Name
			// 	metricSplit := strings.SplitN(val, ": ", 2)
			// 	metricType := metricSplit[0]
			// 	metricValue := metricSplit[1]

			// 	fmt.Printf("METRIC: %s | %s | %s\n", metricName, metricType, metricValue)
			// 	// os.Exit(19)

			// }

			// WalkOID(snmp, metric.Symbol.OID, metric.Symbol.Name, results)
			// // 			// deviceDict["0"]["1"] = 1
			// // 		} else if metric.Table != nil {
			// // 			for _, sym := range metric.Symbols {
			// // 				fmt.Printf("HERE %s\n", sym.OID)
			// // 				// os.Exit(1)
			// // 				// WalkOID(snmp, sym.OID, sym.Name, results)

			// // 				// todo build a savable index here, and call the func to update it. The dict would be metric_name and value. it can be MIB.name and if it is a table an index I guess
			// // 			}
			// // 		}
			// }
			// }
		}
	}
}
}