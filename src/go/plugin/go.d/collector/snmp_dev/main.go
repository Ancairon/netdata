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

func main() {
	profileDir := "../snmp_profiles/default_profiles/"
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
		fmt.Printf("found %v profile(s)\n", len(matchingProfiles))

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

		metricMap := map[string]processedMetric{}

		for name, profile := range matchingProfiles {
			fmt.Println("Profile:", name)

			results := parseMetrics(profile.Metrics)
			// fmt.Println(parseMetrics(profile.Metrics))
			// walk OID subtree
			for _, oid := range results.next_oids {
				if tableRows, err := walkOIDTree(deviceIP, community, oid); err != nil {
					log.Fatalf("Error walking OID tree: %v", err)
				} else {
					fmt.Println(tableRows)
					
					for _, metric := range results.parsed_metrics {
						switch s := metric.(type) {
						case ParsedSymbolMetric:
							// fmt.Println("parsedsymbolmetric")
						case ParsedTableMetric:
							// fmt.Println("parsedtablemetric")
							if s.baseoid == oid {
								fmt.Println("FOUND MATCH", s)
								
								for key,value := range tableRows{
									value.name = s.Name
									fmt.Println(value)
									tableRows[key] = value
								}

								
								// fmt.Println(tableRows)
								metricMap = mergeProcessedMetricMaps(metricMap, tableRows)
								// os.Exit(16)
							}
						default:
							fmt.Println("NONE OF THE TWO",s)
						}
					}

					// response, err := walkOIDTree(deviceIP, oid, "public")
					// if err != nil {
					// 	log.Fatalf("SNMP Exec failed: %v", err)
					// }

					// if len(response) > 0 {}
				}
			}

			for _, value := range metricMap {
				fmt.Println(value)
			}
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
}

type Metadata struct {
	Device DeviceMetadata `yaml:"device"`
}

type DeviceMetadata struct {
	Fields map[string]Symbol `yaml:"fields"`
}

// MetricDefinition represents a metric with a type and a value.
type MetricDefinition struct {
	Type  string  `json:"type"`
	Value float64 `json:"value"`
}

type Symbol struct {
	OID  string `yaml:"OID"`
	Name string `yaml:"name"`
	// MatchPattern string `yaml:"match_pattern,omitempty"`
	// MatchValue   string `yaml:"match_value,omitempty"`
	ExtractValue string `yaml:"extract_value,omitempty"`
}

type IndexTableMetricTag struct {
	Index   int
	Mapping map[int]string
	Tag     string
}

type ColumnTableMetricTag struct {
	MIB            string
	Column         Symbol
	Table          string
	Tag            string
	IndexTransform []IndexSlice
}

// In python this is used as a union of the above types, here we will implement it and then check the type with reflect
type TableMetricTag struct {
	Index   int
	Mapping map[int]string

	Tag string

	MIB            string
	Column         Symbol
	Table          string
	IndexTransform []IndexSlice
}

type OIDMetric struct {
	Name       string   `yaml:"Name"`
	OID        string   `yaml:"OID"`
	MetricTags []string `yaml:"metric_tags,omitempty"`
	ForcedType string   `yaml:"metric_type,omitempty"`
	Options    map[string]string
}

type SymbolMetric struct {
	MIB        string      `yaml:"MIB"`
	Symbol     interface{} `yaml:"symbol,omitempty"` //can be either string or Symbol
	ForcedType string      `yaml:"metric_type,omitempty"`
	MetricTags []string
	Options    map[string]string
}

type TableMetric struct {
	MIB        string
	Table      interface{} // can be either a string or Symbol
	Symbols    []Symbol
	ForcedType string
	MetricTags []TableMetricTag
	Options    map[string]string
}

// superset of OIDMetric, SymbolMetric and TableMetric
type Metric struct {
	Name       string        `yaml:"Name"`
	OID        string        `yaml:"OID"`
	MetricTags []interface{} `yaml:"metric_tags,omitempty"`
	MetricType string        `yaml:"metric_type,omitempty"`
	Options    map[string]string

	MIB    string      `yaml:"MIB"`
	Symbol interface{} `yaml:"symbol,omitempty"` //can be either string or Symbol

	Table   interface{} `yaml:"table,omitempty"` // can be either a string or Symbol
	Symbols []Symbol
}

type IndexMapping struct {
	Tag     string
	Index   int
	Mapping map[int]string
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

type ParsedSymbol struct {
	Name                string
	OID                 string
	ExtractValuePattern *regexp.Regexp
	OIDsToResolve       map[string]string
}

type IndexTag struct {
	ParsedMetricTag ParsedMetricTag
	Index           int
}

type ColumnTag struct {
	DEDUPParsedMetricTag ParsedMetricTag
	Column               string
	IndexSlices          []IndexSlice
}

type ParsedColumnMetricTag struct {
	OIDsToResolve map[string]string
	TableBatches  TableBatches
	ColumnTags    []ColumnTag
}
type ParsedIndexMetricTag struct {
	IndexTags     []IndexTag
	IndexMappings map[int]map[string]string
}

type ParsedTableMetricTag struct {
	OIDsToResolve map[string]string
	TableBatches  TableBatches
	ColumnTags    []ColumnTag
	IndexTags     []IndexTag
	IndexMappings map[int]map[int]string
}

type ParsedSymbolMetric struct {
	Name                string
	Tags                []string
	ForcedType          string
	EnforceScalar       bool
	Options             map[string]string
	ExtractValuePattern *regexp.Regexp
	baseoid string
}

type ParsedTableMetric struct {
	Name                string
	IndexTags           []IndexTag
	ColumnTags          []ColumnTag
	ForcedType          string
	Options             map[string]string
	ExtractValuePattern *regexp.Regexp
	baseoid string
}

// union of two above
type ParsedMetric interface{}

type ParsedSimpleMetricTag struct {
	Name string
}

type ParsedMatchMetricTag struct {
	tags    []string
	symbol  Symbol
	pattern *regexp.Regexp
}

type ParsedMetricTag struct {
	Name string

	tags    []string
	symbol  Symbol
	pattern *regexp.Regexp
}

type SymbolTag struct {
	parsedMetricTag ParsedMetricTag
	symbol          string
}

type ParsedSymbolTagsResult struct {
	oids             []string
	parsedSymbolTags []SymbolTag
}

type MetricParseResult struct {
	oidsToFetch   []string
	oidsToResolve map[string]string
	indexMappings []IndexMapping
	tableBatches  TableBatches
	parsedMetrics []ParsedMetric
}

type MetricTag struct {
	OID    string
	MIB    string
	Symbol string `yaml:"symbol"`
	// simple tag
	Tag string `yaml:"tag"`
	// regex matching
	Match string
	Tags  []string
}

type IndexSlice struct {
	Start int
	End   int
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

func parseMetrics(metrics []Metric) parseResult {
	oids := []string{}
	next_oids := []string{}
	bulk_oids := []string{}
	parsed_metrics := []ParsedMetric{}
	oids_to_resolve := []map[string]string{}
	indexes_to_resolve := []IndexMapping{}
	bulk_threshold := 0
	for _, metric := range metrics {

		result := parseMetric(metric)

		oids = append(oids, result.oidsToFetch...)

		for name, oid := range result.oidsToResolve {
			// here in the python implementation a registration happens to their OIDResolver. I will not support this atm
			oids_to_resolve = append(oids_to_resolve, map[string]string{name: oid})
		}

		for _, index_mapping := range result.indexMappings {
			// same as above
			indexes_to_resolve = append(indexes_to_resolve, index_mapping)
		}

		for _, batch := range result.tableBatches {
			should_query_in_bulk := bulk_threshold > 0 && len(batch.OIDs) > bulk_threshold
			if should_query_in_bulk {
				bulk_oids = append(bulk_oids, batch.TableOID)
			} else {
				next_oids = append(next_oids, batch.OIDs...)
			}
		}

		parsed_metrics = append(parsed_metrics, result.parsedMetrics...)

	}
	return parseResult{oids: oids,
		next_oids: next_oids, bulk_oids: bulk_oids, parsed_metrics: parsed_metrics}
}

type parseResult struct {
	oids           []string
	next_oids      []string
	bulk_oids      []string
	parsed_metrics []ParsedMetric
}

func parseMetric(metric Metric) MetricParseResult {
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
	castedStringMetricTags := []string{}
	castedMetricTags := []TableMetricTag{}
	if len(metric.MetricTags) > 0 {

		switch metric.MetricTags[0].(type) {
		case string:
			castedStringMetricTags = sliceToStrings(metric.MetricTags)

		case MetricTag:
			castedMetricTags = sliceToTableMetricTags(metric.MetricTags)

		}
	}
	if len(metric.OID) > 0 {
		fmt.Printf("parseMetric/Parsing metric: %s\n", metric.Name)
		return (parseOIDMetric(OIDMetric{Name: metric.Name, OID: metric.OID, MetricTags: castedStringMetricTags, ForcedType: metric.MetricType, Options: metric.Options}))
	} else if len(metric.MIB) == 0 {
		fmt.Errorf("Unsupported metric {%v}", metric)
	} else if metric.Symbol != nil {
		return (parseSymbolMetric(SymbolMetric{MIB: metric.MIB, Symbol: metric.Symbol, ForcedType: metric.MetricType, MetricTags: castedStringMetricTags, Options: metric.Options}))
	} else if metric.Table != nil {
		fmt.Printf("parseMetric/Parsing Table: %s\n", metric.Table)

		if metric.Symbols == nil {
			fmt.Errorf("When specifying a table, you must specify a list of symbols %s", metric)
		}

		return (parseTableMetric(TableMetric{MIB: metric.MIB, Table: metric.Table, Symbols: metric.Symbols, ForcedType: metric.MetricType, MetricTags: castedMetricTags, Options: metric.Options}))

	}
	return MetricParseResult{}

}

// TODO error outs on functions
// done translating
func parseOIDMetric(metric OIDMetric) MetricParseResult {
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

	parsed_symbol_metric := ParsedSymbolMetric{
		Name:          name,
		Tags:          metric.MetricTags,
		ForcedType:    metric.ForcedType,
		EnforceScalar: true,
		Options:       metric.Options,
		baseoid: oid,
	}

	return MetricParseResult{
		oidsToFetch:   []string{oid},
		oidsToResolve: map[string]string{name: oid},
		parsedMetrics: []ParsedMetric{parsed_symbol_metric},
		tableBatches:  nil,
		indexMappings: nil,
	}
}

// TODO error outs on functions
// done translating
func parseSymbolMetric(metric SymbolMetric) MetricParseResult {
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

	parsed_symbol_metric := ParsedSymbolMetric{
		Name:                parsed_symbol.Name,
		Tags:                metric.MetricTags,
		ForcedType:          metric.ForcedType,
		EnforceScalar:       false,
		Options:             metric.Options,
		ExtractValuePattern: parsed_symbol.ExtractValuePattern,
		// baseoid: oid,
	}

	return MetricParseResult{
		oidsToFetch:   []string{parsed_symbol.OID},
		oidsToResolve: parsed_symbol.OIDsToResolve,
		parsedMetrics: []ParsedMetric{parsed_symbol_metric},
		tableBatches:  nil,
		indexMappings: nil,
	}
}

// TODO error outs on functions
// done translating
func parseTableMetric(metric TableMetric) MetricParseResult {

	mib := metric.MIB
	fmt.Printf("attempting to parse table with parseSymbol %s\n", metric.Table)
	parsed_table := parseSymbol(mib, metric.Table)
	fmt.Printf("Parsed_table: %s\n", parsed_table)

	oids_to_resolve := parsed_table.OIDsToResolve

	var index_tags []IndexTag
	var column_tags []ColumnTag
	var index_mappings []IndexMapping
	var table_batches map[TableBatchKey]TableBatch

	if metric.MetricTags != nil {
		for _, metric_tag := range metric.MetricTags {
			parsed_table_metric_tag := parseTableMetricTag(mib, parsed_table, metric_tag)

			if parsed_table_metric_tag.OIDsToResolve != nil {
				oids_to_resolve = mergeStringMaps(oids_to_resolve, parsed_table_metric_tag.OIDsToResolve)

				column_tags = append(column_tags, parsed_table_metric_tag.ColumnTags...)

				table_batches = mergeTableBatches(table_batches, parsed_table_metric_tag.TableBatches)
			} else {

				index_tags = append(index_tags, parsed_table_metric_tag.IndexTags...)

				for index, mapping := range parsed_table_metric_tag.IndexMappings {
					for _, symbol := range metric.Symbols {
						index_mappings = append(index_mappings, IndexMapping{Tag: symbol.Name, Index: index, Mapping: mapping})
					}

					for _, tag := range metric.MetricTags {
						if reflect.DeepEqual(tag.Column, Symbol{}) {
							tag = TableMetricTag{
								Tag:    tag.Tag,
								Column: tag.Column,
							}
							index_mappings = append(index_mappings, IndexMapping{
								Tag:     tag.Column.Name,
								Index:   index,
								Mapping: mapping,
							})
						}
					}
				}
			}
		}
	}

	table_oids := []string{}
	parsed_metrics := []ParsedMetric{}

	for _, symbol := range metric.Symbols {
		parsed_symbol := parseSymbol(mib, symbol)

		fmt.Printf("PARSED SYMBOL %s\n", parsed_symbol)

		for key, value := range parsed_symbol.OIDsToResolve {
			oids_to_resolve[key] = value
		}

		table_oids = append(table_oids, parsed_symbol.OID)

		parsed_table_metric := ParsedTableMetric{
			Name:                parsed_symbol.Name,
			IndexTags:           index_tags,
			ColumnTags:          column_tags,
			ForcedType:          metric.ForcedType,
			Options:             metric.Options,
			ExtractValuePattern: parsed_symbol.ExtractValuePattern,
			baseoid: parsed_symbol.OID,
		}

		fmt.Printf("PARSED TABLE METRIC %s\n", parsed_table_metric)

		parsed_metrics = append(parsed_metrics, parsed_table_metric)
	}

	table_batches = mergeTableBatches(table_batches, map[TableBatchKey]TableBatch{TableBatchKey{MIB: mib, Table: parsed_table.Name}: TableBatch{TableOID: parsed_table.OID, OIDs: table_oids}})

	return MetricParseResult{
		oidsToFetch:   []string{},
		oidsToResolve: oids_to_resolve,
		tableBatches:  table_batches,
		indexMappings: index_mappings,
		parsedMetrics: parsed_metrics,
	}

}

func sliceToStrings(items []interface{}) []string {
	var strs []string
	for _, v := range items {
		s, ok := v.(string)
		if !ok {
			// Handle error if an element is not a string.
			continue
		}
		strs = append(strs, s)
	}
	return strs
}

func sliceToTableMetricTags(items []interface{}) []TableMetricTag {
	var metricTag []TableMetricTag
	for _, v := range items {
		s, ok := v.(TableMetricTag)
		if !ok {
			// Handle error if an element is not a string.
			continue
		}
		metricTag = append(metricTag, s)
	}
	return metricTag
}

func mergeTableBatches(target TableBatches, source TableBatches) TableBatches {
	merged := TableBatches{}

	// Extend batches in `target` with OIDs from `source` that share the same key.
	for key, batch := range target {

		if srcBatch, ok := source[key]; ok {
			mergedOids := append(batch.OIDs, srcBatch.OIDs...)
			merged[key] = TableBatch{
				TableOID: batch.TableOID,
				OIDs:     mergedOids,
			}
		}
	}

	for key := range source {
		if _, ok := target[key]; !ok {
			merged[key] = source[key]
		}
	}

	return merged
}

/*
Parse an item of the `metric_tags` section of a table metric.

Items can be:

* A reference to a column in the same table.

Example using entPhySensorTable in ENTITY-SENSOR-MIB:

```
metric_tags:
  - tag: sensor_type
    column: entPhySensorType
    # OR
    column:
    OID: 1.3.6.1.2.1.99.1.1.1.1
    name: entPhySensorType

```

* A reference to a column in a different table.

Example:

```
metric_tags:
  - tag: adapter
    table: genericAdaptersAttrTable
    column: adapterName
    # OR
    column:
    OID: 1.3.6.1.4.1.343.2.7.2.2.1.1.1.2
    name: adapterName

```

* A reference to an OID by its index in the table entry.

An optional `mapping` can be used to map index values to human-readable strings.

Example using ipIfStatsTable in IP-MIB:

```
metric_tags:
  - # ipIfStatsIPVersion (1.3.6.1.2.1.4.21.3.1.1)
    tag: ip_version
    index: 1
    mapping:
    0: unknown
    1: ipv4
    2: ipv6
    3: ipv4z
    4: ipv6z
    16: dns
  - # ipIfStatsIfIndex (1.3.6.1.2.1.4.21.3.1.2)
    tag: interface
    index: 2
    ```
*/
func parseTableMetricTag(mib string, parsed_table ParsedSymbol, metric_tag TableMetricTag) ParsedTableMetricTag {
	if metric_tag.Column != (Symbol{}) {
		metric_tag_mib := metric_tag.MIB

		if metric_tag.Table != "" {
			return parseOtherTableColumnMetricTag(metric_tag_mib, metric_tag.Table, metric_tag)
		}

		if mib != metric_tag_mib {
			fmt.Errorf("When tagging from a different MIB, the table must be specified")
		}

		return parseColumnMetricTag(mib, parsed_table, metric_tag)
	}

	if &metric_tag.Index != nil {
		return parseIndexMetricTag(metric_tag)
	}

	return ParsedTableMetricTag{}
}

func parseIndexMetricTag(metric_tag TableMetricTag) ParsedTableMetricTag {
	index_tags := []IndexTag{IndexTag{
		ParsedMetricTag: parseMetricTag(MetricTag{
			Tag: metric_tag.Tag,
		}), Index: metric_tag.Index,
	}}

	index_mappings := map[int]map[int]string{}

	if metric_tag.Mapping != nil {
		index_mappings = map[int]map[int]string{metric_tag.Index: metric_tag.Mapping}
	}

	return ParsedTableMetricTag{
		IndexTags:     index_tags,
		IndexMappings: index_mappings,
	}
}

func parseOtherTableColumnMetricTag(mib string, table string, metric_tag TableMetricTag) ParsedTableMetricTag {
	parsed_table := parseSymbol(mib, &table)
	parsed_metric_tag := parseColumnMetricTag(mib, parsed_table, metric_tag)

	oids_to_resolve := parsed_metric_tag.OIDsToResolve
	oids_to_resolve = mergeStringMaps(oids_to_resolve, parsed_table.OIDsToResolve)

	return ParsedTableMetricTag{
		OIDsToResolve: oids_to_resolve,
		TableBatches:  parsed_metric_tag.TableBatches,
		ColumnTags:    parsed_metric_tag.ColumnTags,
	}
}

func parseColumnMetricTag(mib string, parsed_table ParsedSymbol, metric_tag TableMetricTag) ParsedTableMetricTag {
	parsed_column := parseSymbol(mib, &metric_tag.Column)

	batches := map[TableBatchKey]TableBatch{TableBatchKey{MIB: mib, Table: parsed_table.Name}: TableBatch{TableOID: parsed_table.OID, OIDs: []string{parsed_column.OID}}}

	return ParsedTableMetricTag{
		OIDsToResolve: parsed_column.OIDsToResolve,
		ColumnTags: []ColumnTag{ColumnTag{
			DEDUPParsedMetricTag: parseMetricTag(MetricTag{MIB: metric_tag.MIB, OID: "", Tag: metric_tag.Tag}),
			Column:               parsed_column.Name,
			IndexSlices:          parseIndexSlices(metric_tag),
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

// ```yaml
// index_transform:
//   - start: 1
//   - end: 3
//
// ```
func parseIndexSlices(metric_tag TableMetricTag) []IndexSlice {
	raw_index_slices := metric_tag.IndexTransform
	index_slices := []IndexSlice{}

	if raw_index_slices != nil {
		for _, rule := range raw_index_slices {
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

			index_slices = append(index_slices, IndexSlice{start, end + 1})

		}

	}
	return index_slices
}

func parseMetricTag(metric_tag MetricTag) ParsedMetricTag {
	parsed_metric_tag := ParsedMetricTag{}
	if metric_tag.Tag != "" {
		parsed_metric_tag = parseSimpleMetricTag(metric_tag)
	} else if metric_tag.Match != "" && metric_tag.Tags != nil {
		parsed_metric_tag = parseRegexMetricTag(metric_tag)
	} else {
		fmt.Errorf("A metric tag must specify either a tag, or a mapping of tags and a regular expression %v", metric_tag)
	}
	return parsed_metric_tag
}

func parseRegexMetricTag(metric_tag MetricTag) ParsedMetricTag {
	// Extract the "match" value.

	match := metric_tag.Match
	tags := metric_tag.Tags

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
	return ParsedMetricTag{tags: tags, pattern: pattern}

}

func parseSimpleMetricTag(metric_tag MetricTag) ParsedMetricTag {
	return ParsedMetricTag{Name: metric_tag.Tag}
}

func mergeStringMaps(m1 map[string]string, m2 map[string]string) map[string]string {
	merged := make(map[string]string)
	for k, v := range m1 {
		merged[k] = v
	}
	for key, value := range m2 {
		merged[key] = value
	}
	return merged
}
func mergeProcessedMetricMaps(m1 map[string]processedMetric, m2 map[string]processedMetric) map[string]processedMetric {
	merged := make(map[string]processedMetric)
	for k, v := range m1 {
		merged[k] = v
	}
	for key, value := range m2 {
		merged[key] = value
	}
	return merged
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

	fmt.Printf("ParsingSymbol %s for MIB %s with type %s\n", symbol, mib, reflect.TypeOf(symbol))

	switch s := symbol.(type) {
	case Symbol:
		fmt.Printf("Symbol found\n")
		oid := s.OID
		name := s.Name
		if s.ExtractValue != "" {
			extractValuePattern, err := regexp.Compile(s.ExtractValue)
			if err != nil {

				return ParsedSymbol{}
				// return "", fmt.Errorf("Failed to compile regular expression %q: %v", symbol.ExtractValue, err)
			}
			fmt.Printf("Returning regexcase %s", ParsedSymbol{
				name,
				oid,
				extractValuePattern,
				map[string]string{name: oid},
			})
			return ParsedSymbol{
				name,
				oid,
				extractValuePattern,
				map[string]string{name: oid},
			}
		} else {
			fmt.Printf("Returning %s\n", ParsedSymbol{
				name,
				oid,
				nil,
				map[string]string{name: oid},
			})
			return ParsedSymbol{
				name,
				oid,
				nil,
				map[string]string{name: oid},
			}
		}
	case string:
		fmt.Printf("string, can't support yet\n")
		return ParsedSymbol{}
	// case interface{}:
	// 	v:=s.(Symbol)
	// 	oid := v.OID
	// 	name := v.Name
	// 	fmt.Printf("interface found\n", )

	// 	fmt.Printf("Returning %s\n",ParsedSymbol{
	// 		name,
	// 		oid,
	// 		nil,
	// 		map[string]string{name: oid},
	// 	})
	// 	return ParsedSymbol{
	// 		name,
	// 		oid,
	// 		nil,
	// 		map[string]string{name: oid},
	// 	}
	case map[string]interface{}:
		oid, okOID := s["OID"].(string)
		name, okName := s["name"].(string)

		if !okOID || !okName {
			fmt.Errorf("invalid symbol format: %+v", s)
			return ParsedSymbol{}
		}

		return ParsedSymbol{
			Name:                name,
			OID:                 oid,
			ExtractValuePattern: nil,
			OIDsToResolve:       map[string]string{name: oid},
		}

	default:
		fmt.Errorf("unsupported symbol type: %T", symbol)
		return ParsedSymbol{}
	}

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

type processedMetric struct {
	// adapt netdata specific keys here
	OID         string
	name        string
	value       string
	metric_type string
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

			fmt.Println(processedMetric{OID: nextOID, value: parts[3], metric_type: parts[2]})

			tableRows[nextOID]= processedMetric{OID: nextOID, value: parts[3], metric_type: strings.Replace(parts[2],":","",1)}

			currentOID = nextOID
		} else {
			fmt.Println("empty response, returning")
			return tableRows, nil
		}
	}
}
