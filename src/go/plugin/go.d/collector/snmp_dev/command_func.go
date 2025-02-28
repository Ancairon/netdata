package main

import (
	"bytes"
	"fmt"
	"os/exec"
	"strings"
)

// Execute SNMPWalk via shell command
func SNMPWalkExec(target string, oid string, community string) (map[string][2]string, error) {

	// fmt.Printf("Walking for %s\n", oid)

	results := make(map[string][2]string)

	// Construct the snmpwalk command
	cmd := exec.Command("snmpwalk", "-v2c", "-c", community, target, oid) // Walk entire SNMP tree

	// Capture the output
	var out bytes.Buffer
	cmd.Stdout = &out

	// Run the command
	err := cmd.Run()
	if err != nil {
		return nil, fmt.Errorf("snmpwalk failed: %v", err)
	}

	// Parse output
	lines := strings.Split(out.String(), "\n")
	for _, line := range lines {
		parts := strings.SplitN(line, " = ", 2)
		if len(parts) == 2 {
			nested_oid := strings.Replace(strings.TrimSpace(parts[0]), "iso", "1", -1)

			couple := strings.Split(strings.TrimSpace(parts[1]), ": ")

			if len(couple) > 1 {

				metric_type := couple[0]
				metric_value := couple[1]
				value := [2]string{metric_type, metric_value}
				// val := [2]string{value}

				if !strings.Contains(value[1], "No Such") || !strings.Contains(value[1], "at this OID") {
					results[nested_oid] = value
					// fmt.Print(walked_oid, value)
				} else {
					fmt.Print("skipping empty OID\n")
					continue
				}
			}

		}
	}

	fmt.Print("Parsing done\n")

	return results, nil
}

// Execute SNMPGet via shell command
func SNMPGetExec(target string, oid string, community string) (string, error) {
	// result := make(map[string]string)
	fmt.Println("snmpget", "-v2c", "-c", community, target, oid)
	// Construct the snmpwalk command
	cmd := exec.Command("snmpget", "-v2c", "-c", community, target, oid) // Walk entire SNMP tree

	// Capture the output
	var out bytes.Buffer
	cmd.Stdout = &out

	// Run the command
	err := cmd.Run()
	if err != nil {
		return "error", fmt.Errorf("snmpget failed: %v", err)
	}

	// // Parse output
	// lines := strings.Split(out.String(), "\n")
	// for _, line := range lines {
	// 	parts := strings.SplitN(line, " = ", 2)
	// 	if len(parts) == 2 {
	// 		oid := strings.Replace(strings.TrimSpace(parts[0]), "iso", "1", -1)
	// 		value := strings.TrimSpace(parts[1])
	// 		results[oid] = value
	// 	}
	// }
	if !strings.Contains(out.String(), "No Such") || !strings.Contains(out.String(), "at this OID") {
		return out.String(), nil
	} else {
		return "", nil
	}

}

func runSNMPGetNext(target string, oid string, community string) (string, error) {
	cmd := exec.Command("snmpgetnext", "-v2c", "-c", community, target, oid)
	var out bytes.Buffer
	var stderr bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &stderr
	err := cmd.Run()
	if err != nil {
		return "", fmt.Errorf("snmpgetnext failed: %v - %s", err, stderr.String())
	}
	return out.String(), nil
}

func walkOIDTree(target, community, baseOID string) (map[string]processedMetric, error) {
	currentOID := baseOID

	tableRows := map[string]processedMetric{}

	for {
		if len(currentOID) > 0 {
			fmt.Println("doing snmpgetnext for", currentOID)
			output, err := runSNMPGetNext(target, currentOID, community)
			if err != nil {
				return tableRows, err
			}
			// Assume output is in the format: "<nextOID> = <value>"
			parts := strings.Fields(output)
			if len(parts) < 1 {
				fmt.Println("No OID returned, ending walk.")
				return tableRows, nil
			}

			nextOID := strings.Replace(parts[0], "iso", "1", 1)
			// If the next OID does not start with the base OID, we've reached the end of the subtree.
			if !strings.HasPrefix(nextOID, baseOID) {
				fmt.Println("Not same prefix", baseOID, nextOID)
				// fmt.Printf("OID: %s, Value: %s\n", nextOID, strings.Join(parts[2:], " "))

				fmt.Println("Reached end of subtree.")
				return tableRows, nil
			}
			fmt.Println("OID:", nextOID, "Value:", parts[2:])

			fmt.Println(processedMetric{oid: nextOID, Value: parts[3], metric_type: parts[2]})

			tableRows[nextOID] = processedMetric{oid: nextOID, Value: parts[3], metric_type: strings.Replace(parts[2], ":", "", 1)}

			currentOID = nextOID
		} else {
			fmt.Println("empty response, returning")
			return tableRows, nil
		}
	}
}
