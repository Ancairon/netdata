package snmp

import (
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/gosnmp/gosnmp"
	"gopkg.in/yaml.v3"
)

func (s *SysObjectIDs) UnmarshalYAML(unmarshal func(interface{}) error) error {
	var single string
	if err := unmarshal(&single); err == nil {
		*s = []string{single}
		return nil
	}

	var multiple []string
	if err := unmarshal(&multiple); err == nil {
		*s = multiple
		return nil
	}

	return fmt.Errorf("invalid sysobjectid format")
}

func (c *Collector) parseMetricsFromProfiles(matchingProfiles []*Profile) (map[string]processedMetric, error) {
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

			response, err := c.snmpClient.Get([]string{oid})

			// response, err := SNMPGet(deviceIP, oid, "public")
			if err != nil {
				return nil, err
			}

			if (response != &gosnmp.SnmpPacket{}) {

				// fmt.Println(response)

				for _, metric := range results.parsed_metrics {
					switch s := metric.(type) {
					case parsedSymbolMetric:
						// fmt.Println("parsedsymbolmetric")

						if s.baseoid == oid {
							// fmt.Println("FOUND MATCH", s, response)

							metricName := s.name
							metricType := response.Variables[0].Type
							metricValue := response.Variables[0].Value

							// fmt.Printf("METRIC: %s | %s | %s\n", metricName, metricType, metricValue)

							metricMap[oid] = processedMetric{oid: oid, name: metricName, value: metricValue, metric_type: metricType}
						}
					}
				}

			}
		}

		for _, oid := range results.next_oids {

			// spew.Dump(oid)

			if len(oid) < 1 {
				// fmt.Println("empty OID, skipping", oid)
				continue
			}
			if tableRows, err := c.walkOIDTree(oid); err != nil {
				log.Fatalf("Error walking OID tree: %v, oid %s", err, oid)
			} else {
				for _, metric := range results.parsed_metrics {
					switch s := metric.(type) {
					case parsedTableMetric:
						if s.rowOID == oid {
							// fmt.Println("FOUND MATCH", s)

							// fmt.Println(tableRows)

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

	}
	return metricMap, nil
}

// UnmarshalYAML custom unmarshaller for Symbol to handle both string and object cases.
// func (s *Symbol) UnmarshalYAML(value *yaml.Node) error {
// 	if value.Kind == yaml.MappingNode && len(value.Content) == 2 {
// 		keyNode := value.Content[0]
// 		if keyNode.Value == "symbol" || keyNode.Value == "value" {
// 			value = value.Content[1]
// 		}
// 	}

// 	// If it's a scalar, treat it as the Name.
// 	if value.Kind == yaml.ScalarNode {
// 		s.Name = value.Value
// 		return nil
// 	}

// 	// Otherwise, decode into a temporary struct.
// 	var temp struct {
// 		OID          string `yaml:"OID"`
// 		Name         string `yaml:"name"`
// 		MatchPattern string `yaml:"match_pattern"`
// 		MatchValue   string `yaml:"match_value"`
// 		ExtractValue string `yaml:"extract_value"`
// 	}
// 	if err := value.Decode(&temp); err != nil {
// 		return err
// 	}
// 	s.OID = temp.OID
// 	s.Name = temp.Name
// 	s.MatchPattern = temp.MatchPattern
// 	s.MatchValue = temp.MatchValue
// 	s.ExtractValue = temp.ExtractValue

// 	fmt.Printf("unmarshal of symbol: %+v\n", s)
// 	return nil
// }

// func (s *SymbolOrString) UnmarshalYAML(value *yaml.Node) error {
// 	// If it's a scalar, treat it as the Name.
// 	if value.Kind == yaml.ScalarNode {
// 		sym := Symbol{Name: value.Value}
// 		fmt.Printf("Decoding as a symbol with only name: %+v\n", sym)
// 		s.Symbol = sym
// 		return nil
// 	}

// 	// Decode into a temporary Symbol once.
// 	var sym Symbol
// 	err := value.Decode(&sym)
// 	if err != nil {
// 		return err
// 	}
// 	// Log the decoded value.
// 	fmt.Printf("Decoding as a symbol: %+v\n", sym)
// 	s.Symbol = sym
// 	return nil
// }

func (s *Symbol) UnmarshalYAML(node *yaml.Node) error {
	// If it's a scalar node, assume the value is the name.
	if node.Kind == yaml.ScalarNode {
		s.Name = node.Value
		return nil
	}

	// Otherwise, decode normally into a temporary type to avoid recursion.
	type plainSymbol Symbol
	var ps plainSymbol
	if err := node.Decode(&ps); err != nil {
		return err
	}
	*s = Symbol(ps)
	return nil
}

// Load all profiles from the directory
func LoadAllProfiles(profileDir string) (map[string]*Profile, error) {
	profiles := make(map[string]*Profile)

	err := filepath.Walk(profileDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if strings.HasSuffix(info.Name(), ".yaml") {
			profile, err := LoadYAML(path, profileDir)
			if err == nil {
				profiles[path] = profile
			} else {
				log.Printf("Skipping invalid YAML: %s (%v)\n", path, err)
			}
		}
		return nil
	})

	if err != nil {
		return nil, err
	}

	return profiles, nil
}

// Load a single YAML profile
func LoadYAML(filename string, basePath string) (*Profile, error) {
	data, err := ioutil.ReadFile(filename)
	if err != nil {
		return nil, err
	}

	var profile Profile
	err = yaml.Unmarshal(data, &profile)
	if err != nil {
		return nil, err
	}

	// If the profile extends other files, load and merge them
	for _, parentFile := range profile.Extends {
		parentProfile, err := LoadYAML(filepath.Join(basePath, parentFile), basePath)
		if err != nil {
			return nil, err
		}
		MergeProfiles(&profile, parentProfile)
	}

	return &profile, nil
}

// Merge two profiles, giving priority to the child profile
func MergeProfiles(child, parent *Profile) {
	// Initialize child metadata fields if nil
	if child.Metadata.Device.Fields == nil {
		child.Metadata.Device.Fields = make(map[string]Symbol)
	}

	// Merge metadata
	for key, value := range parent.Metadata.Device.Fields {
		if _, exists := child.Metadata.Device.Fields[key]; !exists {
			child.Metadata.Device.Fields[key] = value
		}
	}

	// Merge metrics (append new ones)
	child.Metrics = append(parent.Metrics, child.Metrics...)
}

// Find the matching profile based on sysObjectID
func FindMatchingProfiles(profiles map[string]*Profile, deviceOID string) []*Profile {
	var matchedProfiles []*Profile

	for _, profile := range profiles {
		for _, oidPattern := range profile.SysObjectID {
			if strings.HasPrefix(deviceOID, strings.Split(oidPattern, "*")[0]) {
				matchedProfiles = append(matchedProfiles, profile)
				break
			}
		}
	}

	return matchedProfiles
}
