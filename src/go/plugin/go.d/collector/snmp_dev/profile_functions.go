// profiles.go
package snmp_dev

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"gopkg.in/yaml.v3"
)

func getConfig(key string) string {
	// Stub: return an environment variable.
	return os.Getenv(key)
}

func getProfilesConfdUserRoot() string {
	confd := getConfig("confd_path")
	return filepath.Join(confd, "snmp.d", "profiles")
}

func getProfilesConfdDefaultRoot() string {
	confd := getConfig("confd_path")
	return filepath.Join(confd, "snmp.d", "default_profiles")
}

func getProfilesSiteRoot() string {
	_, currentFile, _, _ := runtime.Caller(0)
	return filepath.Join(filepath.Dir(currentFile), "data", "default_profiles")
}

func resolveDefinitionFile(definitionFile string) string {
	if filepath.IsAbs(definitionFile) {
		return definitionFile
	}
	defPath := filepath.Join(getProfilesConfdUserRoot(), definitionFile)
	if fileExists(defPath) {
		return defPath
	}
	defPath = filepath.Join(getProfilesConfdDefaultRoot(), definitionFile)
	if fileExists(defPath) {
		return defPath
	}
	return filepath.Join(getProfilesSiteRoot(), definitionFile)
}

func fileExists(path string) bool {
	info, err := os.Stat(path)
	return err == nil && !info.IsDir()
}

func readProfileDefinition(definitionFile string) (map[string]interface{}, error) {
	path := resolveDefinitionFile(definitionFile)
	data, err := ioutil.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var def map[string]interface{}
	err = yaml.Unmarshal(data, &def)
	return def, err
}

func recursivelyExpandBaseProfiles(definition map[string]interface{}) error {
	extendsRaw, ok := definition["extends"]
	if !ok {
		return nil
	}
	extends, ok := extendsRaw.([]interface{})
	if !ok {
		return ConfigurationError{Msg: "'extends' field is not a list"}
	}
	for _, filenameRaw := range extends {
		filename, ok := filenameRaw.(string)
		if !ok {
			return ConfigurationError{Msg: fmt.Sprintf("invalid extends filename: %v", filenameRaw)}
		}
		baseDef, err := readProfileDefinition(filename)
		if err != nil {
			return err
		}
		if err := recursivelyExpandBaseProfiles(baseDef); err != nil {
			return err
		}
		baseMetrics, _ := baseDef["metrics"].([]interface{})
		existingMetrics, _ := definition["metrics"].([]interface{})
		definition["metrics"] = append(baseMetrics, existingMetrics...)
		currentTags, _ := definition["metric_tags"].([]interface{})
		baseTags, _ := baseDef["metric_tags"].([]interface{})
		definition["metric_tags"] = append(currentTags, baseTags...)
	}
	return nil
}

func iterDefaultProfileFilePaths() ([]string, error) {
	paths := []string{getProfilesConfdUserRoot(), getProfilesConfdDefaultRoot(), getProfilesSiteRoot()}
	var files []string
	for _, p := range paths {
		info, err := os.Stat(p)
		if err != nil || !info.IsDir() {
			continue
		}
		entries, err := ioutil.ReadDir(p)
		if err != nil {
			continue
		}
		for _, entry := range entries {
			if entry.IsDir() || filepath.Ext(entry.Name()) != ".yaml" {
				continue
			}
			files = append(files, filepath.Join(p, entry.Name()))
		}
	}
	return files, nil
}

func getProfileName(path string) string {
	base := filepath.Base(path)
	return strings.TrimSuffix(base, filepath.Ext(base))
}

func isAbstractProfile(name string) bool {
	return strings.HasPrefix(name, "_")
}

func loadDefaultProfiles() (map[string]interface{}, error) {
	profiles := make(map[string]interface{})
	paths, err := iterDefaultProfileFilePaths()
	if err != nil {
		return nil, err
	}
	for _, path := range paths {
		name := getProfileName(path)
		if _, exists := profiles[name]; exists {
			continue
		}
		if isAbstractProfile(name) {
			continue
		}
		def, err := readProfileDefinition(path)
		if err != nil {
			continue
		}
		if err := recursivelyExpandBaseProfiles(def); err != nil {
			continue
		}
		profiles[name] = map[string]interface{}{"definition": def}
	}
	return profiles, nil
}

var defaultProfiles map[string]interface{}

func init() {
	var err error
	defaultProfiles, err = loadDefaultProfiles()
	if err != nil {
		defaultProfiles = make(map[string]interface{})
	}
}

func GetDefaultProfiles() map[string]interface{} {
	return defaultProfiles
}
