package main

import (
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"strings"

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

func (s *SymbolOrString) UnmarshalYAML(value *yaml.Node) error {
	// If it's a scalar, treat it as the Name.
	if value.Kind == yaml.ScalarNode {
		sym := Symbol{Name: value.Value}
		fmt.Printf("Decoding as a symbol with only name: %+v\n", sym)
		s.Symbol = sym
		return nil
	}

	// Decode into a temporary Symbol once.
	var sym Symbol
	err := value.Decode(&sym)
	if err != nil {
		return err
	}
	// Log the decoded value.
	fmt.Printf("Decoding as a symbol: %+v\n", sym)
	s.Symbol = sym
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
