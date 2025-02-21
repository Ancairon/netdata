package main

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"log"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/gosnmp/gosnmp"
	"gopkg.in/yaml.v3"
)



type Metadata struct {
	Device DeviceMetadata `yaml:"device"`
}

type DeviceMetadata struct {
	Fields map[string]Symbol `yaml:"fields"`
}

type Metric struct {
	//this is a superset of the other 3 sub types of metrics
	Name    string `yaml:"Name"`
	OID     string `yaml:"OID"`
	Tags    []MetricTag  `yaml:"metric_tags,omitempty"`
	Type    string `yaml:"metric_type,omitempty"`
	Options map[string]string
	MIB     string   `yaml:"MIB"`
	Symbol  *Symbol  `yaml:"symbol,omitempty"`
	Table   *Symbol   `yaml:"table,omitempty"`
	Symbols []Symbol `yaml:"symbols,omitempty"`
}

// MetricDefinition represents a metric with a type and a value.
type MetricDefinition struct {
	Type  string  `json:"type"`
	Value float64 `json:"value"`
}

type Symbol struct {
	OID          string `yaml:"OID"`
	Name         string `yaml:"name"`
	MatchPattern string `yaml:"match_pattern,omitempty"`
	MatchValue   string `yaml:"match_value,omitempty"`
	ExtractValue string `yaml:"extract_value,omitempty"`
}

type ParsedMetric struct {
	Name                string
	Tags                []MetricTag
	Type                string
	EnforceScalar       bool
	Options             map[string]string
	ExtractValuePattern *regexp.Regexp
	// Table specific
	IndexTags  []IndexTag
	ColumnTags []ColumnTag
}

type IndexTag struct {
	ParsedMetricTag ParsedMetricTag
	Index           int
}

type ColumnTag struct {
	DEDUPParsedMetricTag ParsedMetricTag
	Column          string
	IndexSlices     []IndexSlice
}

type MetricTag struct {
	Tag string `yaml:"tag"`
	// index specific
	Index int
	Mapping map[int]string
	// column specific
	MIB string
	Column Symbol
	Table string
	IndexTransform []IndexSlice


	Symbol string `yaml:"symbol"`
	OID string
	Match string
	Tags map[string]string
}


// type Tag struct {
// 	Tag    string 
// 	Symbol Symbol 
// }
type ParsedTableMetric struct {
	Name string
	IndexTags []IndexTag
	ColumnTags []ColumnTag
	Type string
	Options map[string]string
	ExtractValuePattern *regexp.Regexp
}
type ParsedTableMetricTag struct {
	OIDsToResolve map[string]string
	TableBatches  TableBatches
	ColumnTags    []ColumnTag
	// index specific
	IndexTags     []IndexTag
	IndexMappings map[int]map[string]string
}

type ParsedMetricTag struct {
	Name string
	// Match (regex) metric specific
	Tags    map[string]string
	Pattern *regexp.Regexp
}

type IndexMapping struct {
	Tag     string
	Index   int
	Mapping map[string]string
}

type TableBatchKey struct {
	MIB   string
	Table string
}

type TableBatch struct {
	TableOID string
	OIDs     []string
}

type TableBatches map[TableBatchKey]TableBatch

type MetricParseResult struct {
	OIDsToFetch   []string
	OIDsToResolve map[string]string
	ParsedMetrics []ParsedMetric
	TableBatches  TableBatches
	IndexMappings []IndexMapping
}

type ParsedSymbol struct {
	Name                string
	OID                 string
	ExtractValuePattern *regexp.Regexp
	OIDsToResolve       map[string]string
}

// Profile represents the structure of a Datadog SNMP profile.
type Profile struct {
	Extends     []string     `yaml:"extends"`
	SysObjectID SysObjectIDs `yaml:"sysobjectid"`
	Metadata    Metadata     `yaml:"metadata"`
	Metrics     []Metric     `yaml:"metrics"`
}

// SysObjectIDs allows both a string and list of strings for sysobjectid.
type SysObjectIDs []string

func parseMetrics(metrics []Metric) {
	for _, metric := range metrics {
		// fmt.Print(metric)
		// if len(metric.Type) > 0 {
		// 	print("metric_type")
		// 	metric["Forced_type"] := Metric.Type
		// } else {
		// 	print("empty metric_type")
		// }
		// maybe not needed

		result := parseMetric(metric)

		fmt.Print(result)
	}
}

func parseMetric(metric Metric) string {
	// Can either be:

	// * An OID metric:

	// ```
	// metrics:
	//   - OID: 1.3.6.1.2.1.2.2.1.14
	//     name: ifInErrors
	// ```

	// * A symbol metric:

	// ```
	// metrics:
	//   - MIB: IF-MIB
	//     symbol: ifInErrors
	//     # OR:
	//     symbol:
	//       OID: 1.3.6.1.2.1.2.2.1.14
	//       name: ifInErrors
	// ```

	// * A table metric (see parsing for table metrics for all possible options):

	// ```
	// metrics:
	//   - MIB: IF-MIB
	//     table: ifTable
	//     symbols:
	//       - OID: 1.3.6.1.2.1.2.2.1.14
	//         name: ifInErrors
	// ```
	if len(metric.OID) > 0 {
		fmt.Println(parseOIDMetric(metric))
		return "oid metric parsed"
	} else if len(metric.MIB) == 0 {
		fmt.Errorf("Unsupported metric {%v}", metric)
	} else if metric.Symbol != nil {
		fmt.Println(parseSymbolMetric(metric))
		return "symbol metric parsed"

		//TODO, left here
		// return parseSymbolMetric(metric)

	} else if metric.Table != nil {

		if metric.Symbols == nil {
			fmt.Errorf("When specifying a table, you must specify a list of symbols %s", metric)
		}

		fmt.Println(parseTableMetric(metric))


		// return parseTableMetric(metric)
		return ""
	}
	return ""

}
// TODO error outs on functions
func parseOIDMetric(metric Metric) MetricParseResult {
	// Parse a fully resolved OID/name metric.

	//     Note: This `OID/name` syntax is deprecated in favour of `symbol` syntax.

	//     Example:

	//     ```
	//     metrics:
	//       - OID: 1.3.6.1.2.1.2.1
	//         name: ifNumber
	//     ```

	name := metric.Name
	oid := metric.OID

	parsed_symbol_metric := ParsedMetric{
		Name:                name,
		Tags:                metric.Tags,
		Type:                metric.Type,
		EnforceScalar:       true,
		Options:             metric.Options,
		ExtractValuePattern: nil,
		IndexTags:           nil,
		ColumnTags:          nil,
	}

	return MetricParseResult{
		OIDsToFetch:[]string{oid},
		OIDsToResolve:map[string]string{name:oid},
		ParsedMetrics:[]ParsedMetric{parsed_symbol_metric},
		TableBatches: nil,
		IndexMappings: nil,
	}
	// return ParsedSymbol {
	// 	name,
	// 	oid,
	// 	nil,
	// 	map[string]string{name:oid},
	// }

}

func parseTableMetric(metric Metric) MetricParseResult {
	mib:= metric.MIB

	parsed_table := parseSymbol(mib, metric.Table)
	oids_to_resolve := parsed_table.OIDsToResolve

	var index_tags []string
	var column_tags []ColumnTag
	var index_mappings []string
	var table_batches map[TableBatchKey]TableBatches

	if metric.Tags != nil {
		for _, metric_tag := range metric.Tags {
			parsed_table_metric_tag := parseTableMetricTag(mib, parsed_table, metric_tag)
			
			if reflect.DeepEqual(parsed_table_metric_tag, ParsedTableMetricTag{}) {
				oids_to_resolve = mergeMaps(oids_to_resolve, parsed_table_metric_tag.OIDsToResolve)

				column_tags = append(column_tags, parsed_table_metric_tag.ColumnTags...)

				table_batches = mergeTableBatches(table_batches,parsed_table_metric_tag.TableBatches)
			} else{
				index_tags = append(index_tags, parsed_table_metric_tag.IndexTags)

				for index, mapping := range parsed_table_metric_tag.index_mappings.items(){
					for symbol := range metric.Symbols{
						index_mappings = append(index_mappings, IndexMapping{Tag:symbol.Name, Index: index, Mapping: mapping})
					}

					for tag := range metric.Tags{
						if tag.Column != "" {
							tag = TableMetricTag{
								Tag: tag.Tag,
								Column: tag.Symbol,
							}
							index_mappings = append(index_mappings, IndexMapping{
								Tag: tag.Column.Name,
								Index: index,
								Mapping: mapping,
							})
						}
					}
				}
			}
		}
	}

	table_oids := []string
	parsed_metrics := []ParsedMetric


	for symbol := range metric.Symbols{
		switch s := symbol.(type) {
		case string:
			continue
		// case Symbol:
		parsed_symbol := parseSymbol(mib,symbol)
		oids_to_resolve= append(oids_to_resolve, parsed_symbol.oids_to_resolve)
		
		table_oids = append(table_oids, parsed_symbol.OID)

		parsed_table_metric := ParsedTableMetric{
			Name: parsed_symbol.Name,
			IndexTags: index_tags,
			ColumnTags: column_tags,
			Type: metric.Type,
			Options: metric.Options,
			ExtractValuePattern: parsed_symbol.extractValuePattern,
		}

		parsed_metrics := append(parsed_metrics,parsed_table_metric)
	}

	table_batches := mergeTableBatches(table_batches, map[TableBatchKey]TableBatch{TableBatchKey{MIB:mib, Table:parsed_table.Name}: TableBatch{OID: parsed_table.OID, OIDs: table_oids}})

	return MetricParseResult{
		OIDsToFetch: []string{},
		OIDsToResolve: oids_to_resolve,
		TableBatches: table_batches,
		IndexMappings: index_mappings,
		ParsedMetrics: parsed_metrics,
	}

}}

func mergeTableBatches(target TableBatches, source TableBatches) TableBatches {
	merged := TableBatches{}

	// Extend batches in `target` with OIDs from `source` that share the same key.
	for key, batch := range target {
		if srcBatch, ok := source[key]; ok {
			// Create a new slice with OIDs from both batches.
			newOids := append([]OID{}, batch.Oids...)
			newOids = append(newOids, srcBatch.Oids...)
			merged[key] = TableBatch{
				TableOid: batch.TableOid,
				Oids:     newOids,
			}
		} else {
			merged[key] = batch
		}
	}

	// Add the rest of batches in `source`.make(map[TableBatchKey]TableBatch)

	return merged
}


// Parse an item of the `metric_tags` section of a table metric.

// Items can be:

// * A reference to a column in the same table.

// Example using entPhySensorTable in ENTITY-SENSOR-MIB:

// ```
// metric_tags:
//   - tag: sensor_type
//     column: entPhySensorType
//     # OR
//     column:
//     OID: 1.3.6.1.2.1.99.1.1.1.1
//     name: entPhySensorType
// ```

// * A reference to a column in a different table.

// Example:

// ```
// metric_tags:
//   - tag: adapter
//     table: genericAdaptersAttrTable
//     column: adapterName
//     # OR
//     column:
//       OID: 1.3.6.1.4.1.343.2.7.2.2.1.1.1.2
//       name: adapterName
// ```

// * A reference to an OID by its index in the table entry.

// An optional `mapping` can be used to map index values to human-readable strings.

// Example using ipIfStatsTable in IP-MIB:

// ```
// metric_tags:
//   - # ipIfStatsIPVersion (1.3.6.1.2.1.4.21.3.1.1)
//     tag: ip_version
//     index: 1
//     mapping:
//       0: unknown
//       1: ipv4
//       2: ipv6
//       3: ipv4z
//       4: ipv6z
//       16: dns
//   - # ipIfStatsIfIndex (1.3.6.1.2.1.4.21.3.1.2)
//     tag: interface
//     index: 2
// ```
func parseTableMetricTag(mib string, parsed_table ParsedSymbol, metric_tag TableMetricTag)ParsedTableMetricTag{

	if metric_tag.Column != (Symbol{}) {
		metric_tag_mib := metric_tag.MIB

		if metric_tag.Table != "" {
			return parseOtherTableColumnMetricTag(metric_tag_mib, metric_tag.Table,metric_tag)
		}
		
		if mib != metric_tag_mib {
			fmt.Errorf("When tagging from a different MIB, the table must be specified")
		}

		return parseColumnMetricTag(mib, parsed_table, metric_tag)
	}

	if &metric_tag.Index != nil{
		return parseIndexMetricTag(metricTag)
	}
}

func parseIndexMetricTag(metric_tag TableMetricTag) ParsedTableMetricTag{
	index_tags := []IndexTag{IndexTag{
		ParsedMetricTag: parsed_metric_tag(MetricTag{
			Tag: metric_tag.Tag,
		}),Index: metric_tag.Index,
	}}

	index_mappings := map[int]map[string]string{}

	if metric_tag.Mapping != nil {
		index_mappings:=map[int]map[string]string{metric_tag.Index:metric_tag.Mapping}
	}

	return ParsedTableMetricTag{
		IndexTags: index_tags,
		IndexMappings: index_mappings,
	}
}

func parseOtherTableColumnMetricTag(mib string, table string, metric_tag TableMetricTag) ParsedTableMetricTag{
	parsed_table := parseSymbol(mib, &table)
	parsed_metric_tag := parseColumnMetricTag(mib, parsed_table, metric_tag)

	oids_to_resolve := parsed_metric_tag.OIDsToResolve
	oids_to_resolve = mergeMaps(oids_to_resolve, parsed_table.OIDsToResolve)

	return ParsedTableMetricTag{
		OIDsToResolve: oids_to_resolve,
		TableBatches: parsed_metric_tag.TableBatches,
		ColumnTags: parsed_metric_tag.ColumnTags,
	}
}

func parseColumnMetricTag(mib string,parsed_table ParsedSymbol,metric_tag TableMetricTag) ParsedTableMetricTag{
	parsed_column:= parseSymbol(mib, &metric_tag.Column)
	
	batches := map[TableBatchKey]TableBatch{TableBatchKey{MIB:mib, Table: parsed_table.Name}: TableBatch{TableOID: parsed_table.OID, OIDs: []string{parsed_column.OID}}}

	return ParsedTableMetricTag{
		OIDsToResolve: parsed_column.OIDsToResolve, 
		ColumnTags: []ColumnTag{ColumnTag{
			DEDUPParsedMetricTag:parseMetricTag(MetricTag{MIB: metric_tag.MIB, OID: "", Tag: metric_tag.Tag}),
			Column: parsed_column.Name,
			IndexSlices: parseIndexSlices(metric_tag),
		},
		},
		TableBatches: batches,
	}
}

//     Transform index_transform into list of index slices.

//     `index_transform` is needed to support tagging using another table with different indexes.

//     Example: TableB have two indexes indexX (1 digit) and indexY (3 digits).
//         We want to tag by an external TableA that have indexY (3 digits).

//         For example TableB has a row with full index `1.2.3.4`, indexX is `1` and indexY is `2.3.4`.
//         TableA has a row with full index `2.3.4`, indexY is `2.3.4` (matches indexY of TableB).

//         SNMP integration doesn't know how to compare the full indexes from TableB and TableA.
//         We need to extract a subset of the full index of TableB to match with TableA full index.

//         Using the below `index_transform` we provide enough info to extract a subset of index that
//         will be used to match TableA's full index.

//         ```yaml
//         index_transform:
//           - start: 1
//           - end: 3
//         ```
func parseIndexSlices(metric_tag TableMetricTag) []IndexSlice {
	raw_index_slices := metric_tag.IndexTransform
	index_slices := []IndexSlice{}

	if raw_index_slices != nil{
		for _,rule := range raw_index_slices{
			// if not

			start, end := rule.Start, rule.End
			// check that they are int 
			// if not
			if start > end {
				fmt.Errorf("start bigger than end")
			}
			if start < 0 {
				fmt.Errorf("start is negative")
			}

			index_slices = append(index_slices, IndexSlice{start,end+1})

		}

	}
	return index_slices
}

type IndexSlice struct {
	Start int
	End int
}



func parseMetricTag(metric_tag MetricTag) ParsedMetricTag{
	parsed_metric_tag := ParsedMetricTag{}
	if metric_tag.Tag != ""{
		parsed_metric_tag = parseSimpleMetricTag(metric_tag)
	} else if metric_tag.Match != "" && metric_tag.Tags != nil {
		parsed_metric_tag = parseRegexMetricTag(metric_tag)
	} else {
		fmt.Errorf("A metric tag must specify either a tag, or a mapping of tags and a regular expression %v", metric_tag)
	}
	return parsed_metric_tag
}

func parseRegexMetricTag(metric_tag MetricTag)ParsedMetricTag{
	// Extract the "match" value.
	
	match := metric_tag.Match
	tags:=metric_tag.Tags


	if reflect.TypeOf(tags) != reflect.TypeOf(map[string]string{}) {
		fmt.Errorf("line 209, problem")
	}

	// Compile the regex.
	pattern, err := regexp.Compile(match)
	if err != nil {
		fmt.Errorf("Failed to compile regular expression")
		// return proper error
	}


	// Create and return a new ParsedMatchMetricTag.
	return ParsedMetricTag{Tags: tags,Pattern: pattern}

}



func parseSimpleMetricTag(metric_tag MetricTag) ParsedMetricTag{
	return ParsedMetricTag{Name: metric_tag.Tag}
}

func mergeMaps(m1 map[string]string, m2 map[string]string) map[string]string {
	merged := make(map[string]string)
	for k, v := range m1 {
		merged[k] = v
	}
	for key, value := range m2 {
		merged[key] = value
	}
	return merged
}

func parseSymbolMetric(metric Metric) MetricParseResult {
	//     Parse a symbol metric (= an OID in a MIB).
	//     Example:

	//     ```
	//     metrics:
	//       - MIB: IF-MIB
	//         symbol: <string or OID/name object>
	//       - MIB: IF-MIB
	//         symbol:                     # MIB-less syntax
	//           OID: 1.3.6.1.2.1.6.5.0
	//           name: tcpActiveOpens
	//       - MIB: IF-MIB
	//         symbol: tcpActiveOpens      # require MIB syntax
	//     ```
	mib := metric.MIB
	symbol := metric.Symbol

	parsed_symbol := parseSymbol(mib, symbol)

	parsed_symbol_metric := ParsedMetric{
		Name:                parsed_symbol.Name,
		Tags:                metric.Tags,
		Type:                metric.Type,
		EnforceScalar:       false,
		Options:             metric.Options,
		ExtractValuePattern: parsed_symbol.ExtractValuePattern,
		IndexTags:           nil,
		ColumnTags:          nil,
	}

	return MetricParseResult{
		OIDsToFetch:[]string{parsed_symbol.OID},
		OIDsToResolve:parsed_symbol.OIDsToResolve,
		ParsedMetrics:[]ParsedMetric{parsed_symbol_metric},
		TableBatches: nil,
		IndexMappings: nil,
	}
}


func parseSymbol(mib string, symbol interface{}) ParsedSymbol {
	// Parse an OID symbol.

	// This can either be the unresolved name of a symbol:

	// ```
	// symbol: ifNumber
	// ```

	// Or a resolved OID/name object:

	// ```
	// symbol:
	//     OID: 1.3.6.1.2.1.2.1
	//     name: ifNumber
	// ```

	// if reflect.TypeOf(symbol) == reflect.TypeOf(string) {
	// 	// TODO, here they use ObjectIdentity(mib,symbol) to resolve the symbol. this is not straightfowrard in Go, it is a pysnmp function.
	// 	// oid:= 

	// 	// return ParsedSymbol
	// }
	
	switch s := symbol.(type) {
	case Symbol:
		oid := s.OID
		name := s.Name
		if s.ExtractValue != "" {
			extractValuePattern, err := regexp.Compile(s.ExtractValue)
			if err != nil {

				return ParsedSymbol{}
				// return "", fmt.Errorf("Failed to compile regular expression %q: %v", symbol.ExtractValue, err)
			}
			return ParsedSymbol{
				name,
				oid,
				extractValuePattern,
				map[string]string{name: oid},
			}
		}
	case string:
		return ParsedSymbol{}
	}
	return ParsedSymbol{}
}

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
func (s *Symbol) UnmarshalYAML(value *yaml.Node) error {
	// Case 1: symbol: ifNumber (string)
	if value.Kind == yaml.ScalarNode {
		s.Name = value.Value
		return nil
	}

	// Case 2: symbol: { OID: "...", name: "..." } (map)
	var temp struct {
		OID  string `yaml:"OID"`
		Name string `yaml:"name"`
	}
	if err := value.Decode(&temp); err != nil {
		return err
	}
	s.OID = temp.OID
	s.Name = temp.Name
	return nil
}

// type Table struct {
// 	OID  string `yaml:"OID"`
// 	Name string `yaml:"name"`
// }




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

// Discover all SNMP-enabled devices in the subnet
func ScanSubnet(subnet string, community string, timeout time.Duration) []string {
	ips := []string{}
	ip, ipNet, err := net.ParseCIDR(subnet)
	if err != nil {
		log.Fatalf("Invalid subnet format: %v", err)
	}

	var wg sync.WaitGroup
	ipMutex := &sync.Mutex{}

	for ip := ip.Mask(ipNet.Mask); ipNet.Contains(ip); inc(ip) {
		targetIP := ip.String()
		wg.Add(1)

		go func(ip string) {
			defer wg.Done()
			if isSNMPDevice(ip, community, timeout) {
				ipMutex.Lock()
				ips = append(ips, ip)
				ipMutex.Unlock()
				fmt.Printf("SNMP Device Found: %s\n", ip)
			}
		}(targetIP)
	}

	wg.Wait()
	return ips
}

// Check if an IP is an SNMP-enabled device
func isSNMPDevice(ip, community string, timeout time.Duration) bool {
	snmp := &gosnmp.GoSNMP{
		Target:    ip,
		Port:      161,
		Community: community,
		Version:   gosnmp.Version2c,
		Timeout:   timeout,
		Retries:   1,
	}

	err := snmp.Connect()
	if err != nil {
		return false
	}
	defer snmp.Conn.Close()

	// Check sysObjectID to verify SNMP response
	oid := "1.3.6.1.2.1.1.2.0" // sysObjectID
	result, err := snmp.Get([]string{oid})
	if err != nil || len(result.Variables) == 0 {
		return false
	}
	return true
}

// Increment IP address
func inc(ip net.IP) {
	for j := len(ip) - 1; j >= 0; j-- {
		ip[j]++
		if ip[j] > 0 {
			break
		}
	}
}

// Get sysObjectID dynamically from SNMP
func GetSysObjectID(snmp *gosnmp.GoSNMP) (string, error) {
	oid := "1.3.6.1.2.1.1.2.0" // Standard sysObjectID OID
	result, err := snmp.Get([]string{oid})
	if err != nil {
		return "", err
	}

	if len(result.Variables) == 0 {
		return "", fmt.Errorf("no sysObjectID found")
	}

	return strings.SplitN(fmt.Sprintf("%v", result.Variables[0].Value), ".", 2)[1], nil
}

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

// Execute SNMPWalk via shell command
func SNMPGetExec(target string, oid string, community string) (string, error) {
	// result := make(map[string]string)

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

func main() {
	profileDir := "../snmp_profiles/default_profiles/"
	// profileDir := "./integrations-core/snmp/datadog_checks/snmp/data/default_profiles/"
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

	// deviceDict := make(map[string]map[string]int)

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

		// deviceData, err := SNMPWalkExec(deviceIP, community)
		// if err != nil {
		// 	log.Fatalf("SNMP Walk failed: %v", err)
		// }

		// Print results
		// for oid, value := range deviceData {
		// 	fmt.Printf("%s = %s\n", oid, value)
		// }

		fmt.Println("Fetching sysObjectID...")

		// Get sysObjectID of the device
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
		fmt.Printf("found %s profiles", len(matchingProfiles))

		// // **🌟 Fetch all SNMP data once**
		// fmt.Println("Performing SNMP Walk for the entire device...")
		// deviceData, err := WalkDevice(snmp)
		// if err != nil {
		// 	log.Fatalf("SNMP Walk failed: %v", err)
		// }

		// fmt.Print(deviceData)

		// // // Store unique results
		// // results := make(map[string]string)

		// Walk through the SNMP device using all matched profiles
		for name, profile := range matchingProfiles {
			fmt.Print("Profile:", name)
			parseMetrics(profile.Metrics)
		}
		// for _, metric := range profile.Metrics {

		// 	if metric.Symbol != nil {
		// 		// we have a symbol metric

		// 		// parseSymbolMetric()

		// 		// continue
		// 		// response, err := SNMPGetExec(deviceIP, metric.Symbol.OID, "public")
		// 		// if err != nil {
		// 		// 	log.Fatalf("SNMP Exec failed: %v", err)
		// 		// }

		// 		// if len(response) > 0 {

		// 		// 	metricName := metric.Symbol.Name
		// 		// 	metricSplit := strings.SplitN(strings.SplitN(response, " = ", 2)[1], ": ", 2)
		// 		// 	if len(metricSplit) < 2 {
		// 		// 		fmt.Print(metricSplit)
		// 		// 		os.Exit(-9324)
		// 		// 	}
		// 		// 	metricType := metricSplit[0]
		// 		// 	metricValue := metricSplit[1]

		// 		// 	fmt.Printf("METRIC: %s | %s | %s\n", metricName, metricType, metricValue)
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
