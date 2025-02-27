package main

import "regexp"

type Metadata struct {
	Device DeviceMetadata `yaml:"device"`
}

type processedMetric struct {
	// adapt netdata specific keys here
	OID         string
	name        string
	Value       string
	metric_type string
}

type DeviceMetadata struct {
	Fields map[string]Symbol `yaml:"fields"`
}

type parseResult struct {
	oids           []string
	next_oids      []string
	bulk_oids      []string
	parsed_metrics []ParsedMetric
}
// MetricDefinition represents a metric with a type and a value.
type MetricDefinition struct {
	Type  string  `json:"type"`
	Value float64 `json:"value"`
}

type Symbol struct {
	OID  string `yaml:"OID,omitempty"`
	Name string `yaml:"name,omitempty"`
	MatchPattern string `yaml:"match_pattern,omitempty"`
	MatchValue   string `yaml:"match_value,omitempty"`
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

//TODO check for s["Tag"] if and when making these lowecase
type TableMetricTag struct {
	Index   int
	Mapping map[int]string

	Tag string

	MIB            string
	Symbol         Symbol
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
	Name       string        `yaml:"name,omitempty"`
	OID        string        `yaml:"OID,omitempty"`
	//TODO check for only name existing in metric tag, as there is some case for that
	MetricTags []interface{} `yaml:"metric_tags,omitempty"`
	MetricType string        `yaml:"metric_type,omitempty"`
	Options    map[string]string

	MIB    string      `yaml:"MIB,omitempty"`
	Symbol Symbol `yaml:"symbol,omitempty"` //can be either string or Symbol

	Table   interface{} `yaml:"table,omitempty"` // can be either a string or Symbol
	Symbols []Symbol `yaml:"symbols,omitempty"`
}

// type SymbolOrString struct {
//     Symbol Symbol
// }



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
	baseoid string //TODO change this to OID, it will not have nested OIDs as it is a symbol
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
	Symbol Symbol `yaml:"symbol"`
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
