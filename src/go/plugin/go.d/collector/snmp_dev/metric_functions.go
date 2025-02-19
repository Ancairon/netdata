package snmp_dev

import (
	"fmt"
	"log"
	"regexp"
	"strconv"
)

func NewParsedSimpleMetricTag(name string) *ParsedSimpleMetricTag {
	return &ParsedSimpleMetricTag{Name: name}
}

func (p *ParsedSimpleMetricTag) MatchedTags(value interface{}) []string {
	return []string{fmt.Sprintf("%s:%v", p.Name, value)}
}

// NewDevice creates a new Device.
func NewDevice(ip string, port int, target string) *Device {
	return &Device{IP: ip, Port: port, Target: target}
}

func NewParsedMatchMetricTag(tags map[string]string, pattern *regexp.Regexp) *ParsedMatchMetricTag {
	return &ParsedMatchMetricTag{
		Tags:    tags,
		Pattern: pattern,
	}
}

func (p *ParsedMatchMetricTag) MatchedTags(value interface{}) []string {
	strVal := fmt.Sprintf("%v", value)
	matches := p.Pattern.FindStringSubmatch(strVal)
	if matches == nil {
		return nil
	}
	var result []string
	for tagName, tmpl := range p.Tags {
		expanded := p.Pattern.ReplaceAllString(strVal, tmpl)
		result = append(result, fmt.Sprintf("%s:%s", tagName, expanded))
	}
	return result
}

func NewParsedTableMetric(name string, indexTags, columnTags []any, forcedType string, options map[string]interface{}, extractValuePattern *regexp.Regexp) *ParsedTableMetric {
	if options == nil {
		options = make(map[string]interface{})
	}
	return &ParsedTableMetric{
		Name:                name,
		IndexTags:           indexTags,
		ColumnTags:          columnTags,
		ForcedType:          forcedType,
		Options:             options,
		ExtractValuePattern: extractValuePattern,
	}
}

func NewParsedSymbolMetric(name string, tags []string, forcedType string, enforceScalar bool, options map[string]interface{}, extractValuePattern *regexp.Regexp) *ParsedSymbolMetric {
	if tags == nil {
		tags = []string{}
	}
	if options == nil {
		options = make(map[string]interface{})
	}
	return &ParsedSymbolMetric{
		Name:                name,
		Tags:                tags,
		ForcedType:          forcedType,
		EnforceScalar:       enforceScalar,
		Options:             options,
		ExtractValuePattern: extractValuePattern,
	}
}

func ParseMetrics(metrics []map[string]interface{}, resolver *OIDResolver, logger *log.Logger, bulkThreshold int) (ParseMetricsResult, error) {
	var oids []OID
	var nextOids []OID
	var bulkOids []OID
	var parsedMetrics []ParsedMetric

	// Iterate over each metric config.
	for _, metric := range metrics {
		// Backward compatibility: if "metric_type" is present, set "forced_type"
		if mt, ok := metric["metric_type"]; ok {
			metric["forced_type"] = mt
		}

		result, err := parseMetric(metric, logger)
		if err != nil {
			return ParseMetricsResult{}, err
		}

		oids = append(oids, result.OidsToFetch...)
		for name, oid := range result.OidsToResolve {
			resolver.Register(&oid, name)
		}
		for _, im := range result.IndexMappings {
			resolver.RegisterIndex(im.Tag, im.Index, convertMapping(im.Mapping))
		}
		for _, batch := range result.TableBatches {
			shouldQueryInBulk := bulkThreshold > 0 && len(batch.Oids) > bulkThreshold
			if shouldQueryInBulk {
				bulkOids = append(bulkOids, batch.TableOid)
			} else {
				nextOids = append(nextOids, batch.Oids...)
			}
		}
		parsedMetrics = append(parsedMetrics, result.ParsedMetrics...)
	}

	return ParseMetricsResult{
		Oids:          oids,
		NextOids:      nextOids,
		BulkOids:      bulkOids,
		ParsedMetrics: parsedMetrics,
	}, nil
}

func parseMetric(metric map[string]interface{}, logger *log.Logger) (MetricParseResult, error) {
	if _, ok := metric["OID"]; ok {
		return parseOidMetric(metric)
	}

	if _, ok := metric["MIB"]; !ok {
		return MetricParseResult{}, fmt.Errorf("Unsupported metric in config file: %v", metric)
	}

	if _, ok := metric["symbol"]; ok {
		return parseSymbolMetric(metric)
	}

	if _, ok := metric["table"]; ok {
		if _, ok := metric["symbols"]; !ok {
			return MetricParseResult{}, fmt.Errorf("When specifying a table, you must specify a list of symbols")
		}
		return parseTableMetric(metric, logger)
	}

	return MetricParseResult{}, fmt.Errorf("When specifying a MIB, you must specify either a table or a symbol")
	// return fmt.Errorf("generic error line 142")
}

func parseOidMetric(metric map[string]interface{}) (MetricParseResult, error) {
	name, ok := metric["name"].(string)
	if !ok {
		return MetricParseResult{}, fmt.Errorf("OID metric missing name")
	}
	oidStr, ok := metric["OID"].(string)
	if !ok {
		return MetricParseResult{}, fmt.Errorf("OID metric missing OID")
	}
	oid := (oidStr)
	parsedSymbolMetric := NewParsedSymbolMetric(name, getOptionalStringSlice(metric, "metric_tags"), getOptionalString(metric, "forced_type"), false, getOptionalMap(metric, "options"))
	return MetricParseResult{
		OidsToFetch:   []OID{oid},
		OidsToResolve: map[string]OID{name: oid},
		IndexMappings: []IndexMapping{},
		TableBatches:  make(TableBatches),
		ParsedMetrics: []ParsedMetric{parsedSymbolMetric},
	}, nil
}

// AsMetricWithInferredType attempts to derive a metric definition from the SNMP value.
func AsMetricWithInferredType(value interface{}) *MetricDefinition {
	// If the value is a counter, return a rate metric.
	if IsCounter(value) {
		if intVal, err := ToInt(value); err == nil {
			return &MetricDefinition{
				Type:  "rate",
				Value: float64(intVal),
			}
		}
		if floatVal, err := VarbindValueToFloat(value); err == nil {
			return &MetricDefinition{
				Type:  "rate",
				Value: floatVal,
			}
		}
	}

	// If the value is a gauge, return a gauge metric.
	if IsGauge(value) {
		if intVal, err := ToInt(value); err == nil {
			return &MetricDefinition{
				Type:  "gauge",
				Value: float64(intVal),
			}
		}
		if floatVal, err := VarbindValueToFloat(value); err == nil {
			return &MetricDefinition{
				Type:  "gauge",
				Value: floatVal,
			}
		}
	}

	// If the value is opaque, try decoding it (e.g. ASN.1 encoded).
	if IsOpaque(value) {
		var data []byte
		switch v := value.(type) {
		case []byte:
			data = v
		case string:
			data = []byte(v)
		case OctetString:
			data = []byte(v)
		default:
			// Unsupported type for opaque decoding.
			break
		}
		if data != nil {
			decoded, err := PyAsn1Decode(data)
			if err == nil {
				return &MetricDefinition{
					Type:  "gauge",
					Value: decoded,
				}
			}
		}
	}

	// Fallback: try to convert to float.
	if number, err := VarbindValueToFloat(value); err == nil {
		return &MetricDefinition{
			Type:  "gauge",
			Value: number,
		}
	}
	return nil
}

func parseSymbolMetric(metric map[string]interface{}) (MetricParseResult, error) {
	mib, ok := metric["MIB"].(string)
	if !ok {
		return MetricParseResult{}, fmt.Errorf("Symbol metric missing MIB")
	}
	symbol := metric["symbol"]
	parsedSymbol, err := parseSymbol(mib, symbol)
	if err != nil {
		return MetricParseResult{}, err
	}
	// TODO, static enforceScalar here
	parsedSymbolMetric := NewParsedSymbolMetric(parsedSymbol.name, getOptionalStringSlice(metric, "metric_tags"), getOptionalString(metric, "forced_type"), false, getOptionalMap(metric, "options"), parsedSymbol.extractValuePattern)
	return MetricParseResult{
		OidsToFetch:   []OID{parsedSymbol.oid},
		OidsToResolve: parsedSymbol.oidsToResolve,
		IndexMappings: []IndexMapping{},
		TableBatches:  make(TableBatches),
		ParsedMetrics: []ParsedMetric{parsedSymbolMetric},
	}, nil
}

func parseSymbol(mib string, symbol interface{}) (ParsedSymbol, error) {
	switch s := symbol.(type) {
	case string:
		oi := NewObjectIdentity(mib, s)
		oid := NewOIDFromObjectIdentity(oi)
		return ParsedSymbol{
			name:                s,
			oid:                 oid,
			extractValuePattern: nil,
			oidsToResolve:       map[string]OID{},
		}, nil
	case map[string]interface{}:
		oidStr, ok := s["OID"].(string)
		if !ok {
			return ParsedSymbol{}, fmt.Errorf("Symbol metric map missing OID")
		}
		oid := NewOID(oidStr)
		name, ok := s["name"].(string)
		if !ok {
			return ParsedSymbol{}, fmt.Errorf("Symbol metric map missing name")
		}
		var extractValuePattern *regexp.Regexp
		if ev, ok := s["extract_value"]; ok {
			patternStr, ok := ev.(string)
			if !ok {
				return ParsedSymbol{}, fmt.Errorf("extract_value must be a string")
			}
			pat, err := regexp.Compile(patternStr)
			if err != nil {
				return ParsedSymbol{}, fmt.Errorf(fmt.Sprintf("Failed to compile regex %q: %v", patternStr, err))
			}
			extractValuePattern = pat
		}
		return ParsedSymbol{
			name:                name,
			oid:                 oid,
			extractValuePattern: extractValuePattern,
			oidsToResolve:       map[string]OID{name: oid},
		}, nil
	default:
		return ParsedSymbol{}, fmt.Errorf(fmt.Sprintf("Invalid symbol type: %T", symbol))
	}
}

func parseTableMetric(metric map[string]interface{}, logger *log.Logger) (MetricParseResult, error) {
	mib, ok := metric["MIB"].(string)
	if !ok {
		return MetricParseResult{}, fmt.Errorf("Table metric missing MIB")
	}
	parsedTable, err := parseSymbol(mib, metric["table"])
	if err != nil {
		return MetricParseResult{}, err
	}
	oidsToResolve := parsedTable.oidsToResolve

	var indexTags []IndexMapping
	var columnTags []ColumnTag
	var indexMappings []IndexMapping
	tableBatches := make(TableBatches)

	// If metric_tags is provided.
	if mt, ok := metric["metric_tags"]; ok {
		metricTags := mt.([]interface{})
		for _, item := range metricTags {
			// _parse_table_metric_tag returns either a ParsedColumnMetricTag or ParsedIndexMetricTag.
			ptmt, err := parseTableMetricTag(mib, parsedTable, item.(map[string]interface{}))
			if err != nil {
				return MetricParseResult{}, err
			}
			switch t := ptmt.(type) {
			case ParsedColumnMetricTag:
				for name, oid := range t.OidsToResolve {
					oidsToResolve[name] = oid
				}
				columnTags = append(columnTags, t.ColumnTags...)
				tableBatches = mergeTableBatches(tableBatches, t.TableBatches)
			case ParsedIndexMetricTag:
				indexTags = append(indexTags, t.IndexTags...)
				for idx, mapping := range t.IndexMappings {
					// For each symbol in metric["symbols"]
					for _, sym := range metric["symbols"].([]interface{}) {
						symMap := sym.(map[string]interface{})
						indexMappings = append(indexMappings, IndexMapping{Tag: symMap["name"].(string), Index: idx, Mapping: mapping})
					}
					for _, tagItem := range metric["metric_tags"].([]interface{}) {
						tagMap := tagItem.(map[string]interface{})
						if _, ok := tagMap["column"]; ok {
							ct := tagMap.(map[string]interface{})
							// Assume column is a map with key "name"
							indexMappings = append(indexMappings, IndexMapping{Tag: ct["column"].(map[string]interface{})["name"].(string), Index: idx, Mapping: mapping})
						}
					}
				}
			}
		}
	} else if logger != nil {
		logger.Printf("%s table doesn't have a 'metric_tags' section, all its metrics will use the same tags.", metric["table"].(string))
	}

	var tableOids []OID
	var parsedMetrics []ParsedMetric

	for _, sym := range metric["symbols"].([]interface{}) {
		// Skip symbols with constant_value_one.
		if symMap, ok := sym.(map[string]interface{}); ok {
			if constant, exists := symMap["constant_value_one"]; exists && constant.(bool) {
				if logger != nil {
					logger.Printf("`constant_value_one` is only available with the core SNMP integration")
				}
				continue
			}
		}
		parsedSymbol, err := parseSymbol(mib, sym)
		if err != nil {
			return MetricParseResult{}, err
		}
		for name, oid := range parsedSymbol.oidsToResolve {
			oidsToResolve[name] = oid
		}
		tableOids = append(tableOids, parsedSymbol.oid)
		parsedTableMetric := NewParsedTableMetric(parsedSymbol.name, indexTags, columnTags, getOptionalString(metric, "forced_type"), getOptionalMap(metric, "options"), parsedSymbol.extractValuePattern)
		parsedMetrics = append(parsedMetrics, parsedTableMetric)
	}

	tbKey := NewTableBatchKey(mib, parsedTable.name)
	tableBatches = mergeTableBatches(tableBatches, map[TableBatchKey]TableBatch{
		tbKey: NewTableBatch(parsedTable.oid, tableOids),
	})

	return NewMetricParseResult([]OID{}, oidsToResolve, indexMappings, tableBatches, parsedMetrics), nil
}

// mergeTableBatches merges two TableBatches maps.
func mergeTableBatches(target, source map[TableBatchKey]TableBatch) map[TableBatchKey]TableBatch {
	merged := make(map[TableBatchKey]TableBatch)
	for key, batch := range target {
		if srcBatch, exists := source[key]; exists {
			merged[key] = NewTableBatch(batch.TableOid, append(batch.Oids, srcBatch.Oids...))
		} else {
			merged[key] = batch
		}
	}
	for key, batch := range source {
		if _, exists := target[key]; !exists {
			merged[key] = batch
		}
	}
	return merged
}

// getOptionalString extracts an optional string from a map.
func getOptionalString(m map[string]interface{}, key string) string {
	if v, ok := m[key]; ok {
		return fmt.Sprintf("%v", v)
	}
	return ""
}

// getOptionalStringSlice extracts an optional slice of strings from a map.
func getOptionalStringSlice(m map[string]interface{}, key string) []string {
	if v, ok := m[key]; ok {
		switch s := v.(type) {
		case []interface{}:
			var result []string
			for _, item := range s {
				result = append(result, fmt.Sprintf("%v", item))
			}
			return result
		case []string:
			return s
		}
	}
	return []string{}
}

// getOptionalMap extracts an optional map[string]interface{} from a map.
func getOptionalMap(m map[string]interface{}, key string) map[string]interface{} {
	if v, ok := m[key]; ok {
		if mp, ok := v.(map[string]interface{}); ok {
			return mp
		}
	}
	return map[string]interface{}{}
}

func parseTableMetricTag(mib string, parsedTable ParsedSymbol, metricTag map[string]interface{}) (ParsedTableMetricTag, error) {
	// For backward compatibility: if 'symbol' is present, rename it to 'column'
	if _, ok := metricTag["symbol"]; ok {
		metricTag["column"] = metricTag["symbol"]
		delete(metricTag, "symbol")
	}
	if _, ok := metricTag["column"]; ok {
		// Check if a "table" key exists in metricTag (different table column)
		if tbl, ok := metricTag["table"]; ok {
			return parseOtherTableColumnMetricTag(mib, tbl.(string), metricTag)
		}
		if mib != getOptionalString(metricTag, "MIB") && getOptionalString(metricTag, "MIB") != "" {
			return nil, fmt.Errorf("When tagging from a different MIB, the table must be specified")
		}
		return parseColumnMetricTag(mib, parsedTable, metricTag)
	}
	if _, ok := metricTag["index"]; ok {
		return parseIndexMetricTag(metricTag)
	}
	return nil, fmt.Errorf("When specifying metric tags, you must specify either an index or a column")
}

func parseColumnMetricTag(mib string, parsedTable ParsedSymbol, metricTag map[string]interface{}) (ParsedColumnMetricTag, error) {
	parsedColumn, err := parseSymbol(mib, metricTag["column"])
	if err != nil {
		return ParsedColumnMetricTag{}, err
	}
	batches := map[TableBatchKey]TableBatch{
		NewTableBatchKey(mib, parsedTable.name): NewTableBatch(parsedTable.oid, []OID{parsedColumn.oid}),
	}
	return ParsedColumnMetricTag{
		OidsToResolve: parsedColumn.oidsToResolve,
		TableBatches:  batches,
		ColumnTags: []ColumnTag{
			{
				ParsedMetricTag: NewParsedSimpleMetricTag(getOptionalString(metricTag, "tag")),
				Column:          parsedColumn.name,
				IndexSlices:     parseIndexSlices(metricTag),
			},
		},
	}, nil
}

func parseOtherTableColumnMetricTag(mib string, table string, metricTag map[string]interface{}) (ParsedTableMetricTag, error) {
	parsedTable := func() ParsedSymbol {
		// Parse the table symbol using the provided table name.
		ps, _ := parseSymbol(mib, table)
		return ps
	}()
	parsedMetricTag, err := parseColumnMetricTag(mib, parsedTable, metricTag)
	if err != nil {
		return nil, err
	}
	// Merge oids_to_resolve from parsedTable.
	for name, oid := range parsedTable.oidsToResolve {
		parsedMetricTag.OidsToResolve[name] = oid
	}
	return parsedMetricTag, nil
}

func parseIndexMetricTag(metricTag map[string]interface{}) (ParsedIndexMetricTag, error) {
	indexVal, ok := metricTag["index"]
	if !ok {
		return ParsedIndexMetricTag{}, fmt.Errorf("Index metric tag missing index")
	}
	index, ok := indexVal.(int)
	if !ok {
		return ParsedIndexMetricTag{}, fmt.Errorf("Index metric tag index must be an integer")
	}
	var mapping map[string]interface{}
	if m, ok := metricTag["mapping"]; ok {
		mapping, _ = m.(map[string]interface{})
	}
	indexTags := []IndexTag{
		{
			ParsedMetricTag: NewParsedSimpleMetricTag(getOptionalString(metricTag, "tag")),
			Index:           index,
		},
	}
	indexMappings := map[int]map[string]interface{}{}
	if mapping != nil {
		indexMappings[index] = mapping
	}
	return ParsedIndexMetricTag{
		IndexTags:     indexTags,
		IndexMappings: indexMappings,
	}, nil
}

// parseIndexSlices transforms the index_transform rules into a slice of IndexSlice.
func parseIndexSlices(metricTag map[string]interface{}) []IndexSlice {
	raw, ok := metricTag["index_transform"]
	if !ok {
		return nil
	}
	rawSlices, ok := raw.([]interface{})
	if !ok {
		// Invalid format.
		panic(fmt.Sprintf("Transform rule must be a list, got: %v", raw))
	}
	var slices []IndexSlice
	for _, rule := range rawSlices {
		ruleMap, ok := rule.(map[string]interface{})
		if !ok || len(ruleMap) != 2 {
			panic(fmt.Sprintf("Transform rule must contain start and end. Invalid rule: %v", rule))
		}
		startVal, ok1 := ruleMap["start"].(int)
		endVal, ok2 := ruleMap["end"].(int)
		if !ok1 || !ok2 {
			panic(fmt.Sprintf("Transform rule start and end must be integers. Invalid rule: %v", rule))
		}
		if startVal > endVal {
			panic(fmt.Sprintf("Transform rule end should be greater than start. Invalid rule: %v", rule))
		}
		if startVal < 0 {
			panic(fmt.Sprintf("Transform rule start must be greater than 0. Invalid rule: %v", rule))
		}
		// In Python, end is inclusive so we add 1.
		slices = append(slices, IndexSlice{Start: startVal, Stop: endVal + 1})
	}
	return slices
}

// --------------------------
// Helper Functions for Merging and Constructing Parsed Results
// --------------------------

// NewMetricParseResult constructs a new MetricParseResult.
func NewMetricParseResult(oidsToFetch []OID, oidsToResolve map[string]OID, indexMappings []IndexMapping, tableBatches TableBatches, parsedMetrics []ParsedMetric) MetricParseResult {
	return MetricParseResult{
		OidsToFetch:   oidsToFetch,
		OidsToResolve: oidsToResolve,
		IndexMappings: indexMappings,
		TableBatches:  tableBatches,
		ParsedMetrics: parsedMetrics,
	}
}

func parseSymbolMetricTag(metricTag MetricTag) (metricTagParseResult, error) {
	oidsToResolve := make(map[string]OID)
	var oid OID
	if metricTag.MIB != "" {
		oi := NewObjectIdentity(metricTag.MIB, metricTag.Symbol)
		oid = NewOIDFromObjectIdentity(oi)
	} else if metricTag.OID != "" {
		oid = NewOIDFromString(metricTag.OID)
		oidsToResolve[metricTag.Symbol] = oid
	} else {
		return metricTagParseResult{}, ConfigurationError{Msg: fmt.Sprintf("A metric tag must specify an OID or a MIB: %+v", metricTag)}
	}
	parsedMetricTag, err := parseMetricTag(metricTag)
	if err != nil {
		return metricTagParseResult{}, err
	}
	symbolTag := SymbolTag{
		ParsedMetricTag: parsedMetricTag,
		Symbol:          metricTag.Symbol,
	}
	return metricTagParseResult{
		Oid:           oid,
		SymbolTag:     symbolTag,
		OidsToResolve: oidsToResolve,
	}, nil
}

// NewTableBatchKey constructs a new TableBatchKey.
func NewTableBatchKey(mib, table string) TableBatchKey {
	return TableBatchKey{Mib: mib, Table: table}
} // NewTableBatch constructs a new TableBatch.
func NewTableBatch(tableOid OID, oids []OID) TableBatch {
	return TableBatch{TableOid: tableOid, Oids: oids}
}

// convertMapping converts a map with string keys and interface{} values to a map with int keys and string values.
func convertMapping(m map[string]interface{}) map[int]string {
	result := make(map[int]string)
	for key, value := range m {
		// Convert key from string to int.
		intKey, err := strconv.Atoi(key)
		if err != nil {
			// You can choose to handle the error differently.
			// For now, we'll skip keys that cannot be converted.
			continue
		}
		// Convert value to string.
		result[intKey] = fmt.Sprintf("%v", value)
	}
	return result
}
func ParseSymbolMetricTags(metricTags []MetricTag, resolver *OIDResolver) (struct {
	Oids             []OID
	ParsedSymbolTags []SymbolTag
}, error) {
	var oids []OID
	var parsedSymbolTags []SymbolTag

	for _, mt := range metricTags {
		if mt.Symbol == "" {
			return struct {
					Oids             []OID
					ParsedSymbolTags []SymbolTag
				}{},
				// ConfigurationError{Msg: fmt.Sprintf("A metric tag must specify a symbol: %+v", mt)}
				fmt.Errorf("error metrics_tags_parsing line 47")
		}

		result, err := parseSymbolMetricTag(mt)
		if err != nil {
			return struct {
				Oids             []OID
				ParsedSymbolTags []SymbolTag
			}{}, err
		}

		for name, oid := range result.OidsToResolve {
			resolver.Register(oid, name)
		}
		oids = append(oids, result.Oid)
		parsedSymbolTags = append(parsedSymbolTags, result.SymbolTag)
	}

	return struct {
		Oids             []OID
		ParsedSymbolTags []SymbolTag
	}{
		Oids:             oids,
		ParsedSymbolTags: parsedSymbolTags,
	}, nil
}

func parseMetricTag(metricTag MetricTag) (ParsedMetricTag, error) {
	if metricTag.Tag != "" {
		return NewParsedSimpleMetricTag(metricTag.Tag), nil
	} else if metricTag.Match != "" && metricTag.Tags != nil {
		pattern, err := regexp.Compile(metricTag.Match)
		if err != nil {
			return nil, ConfigurationError{Msg: fmt.Sprintf("Failed to compile regex %q: %v", metricTag.Match, err)}
		}
		return NewParsedMatchMetricTag(metricTag.Tags, pattern), nil
	}
	return nil, ConfigurationError{Msg: fmt.Sprintf("A metric tag must specify either a tag or a mapping: %+v", metricTag)}
}
