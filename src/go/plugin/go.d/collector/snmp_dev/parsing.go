package main

import (
	"fmt"
	"os"
	"reflect"
	"regexp"

	"github.com/davecgh/go-spew/spew"
)

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

		// here in the python implementation a registration happens to their OIDResolver. I will not support this atm
		indexes_to_resolve = append(indexes_to_resolve, result.indexMappings...)

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
	castedTableMetricTags := []TableMetricTag{}
	if len(metric.MetricTags) > 0 {
		// fmt.Println("parseMetric MetricTags switch, have:")
		// spew.Dump(metric.MetricTags)
		// TODO investigate if there are metric tags and not only table metric tags
		switch metric.MetricTags[0].(type) {
		case string:
			os.Exit(-40)
			castedStringMetricTags = sliceToStrings(metric.MetricTags)

		case MetricTag:
			os.Exit(-40)
			castedTableMetricTags = sliceToTableMetricTags(metric.MetricTags)
		}
		// case map[string]interface{} :
		// 	for _,item := range metric.MetricTags{
		// 		fmt.Println("ITEM", item, m)

		// 	}
		}
			// 	oid, okOID := s["OID"].(string)
		// name, okName := s["name"].(string)

		// if !okOID || !okName {
		// 	fmt.Errorf("invalid symbol format: %+v", s)
		// 	return ParsedSymbol{}
		// }

		// return ParsedSymbol{
		// 	Name:                name,
		// 	OID:                 oid,
		// 	ExtractValuePattern: nil,
		// 	OIDsToResolve:       map[string]string{name: oid},
		// }
		// }
	

	fmt.Println(metric)
	if len(metric.OID) > 0 {
		// TODO investigate if this exists in the yamls
		// fmt.Printf("parseMetric/Parsing OID metric: %s\n", metric.Name)
		return (parseOIDMetric(OIDMetric{Name: metric.Name, OID: metric.OID, MetricTags: castedStringMetricTags, ForcedType: metric.MetricType, Options: metric.Options}))
	} else if len(metric.MIB) == 0 {
		fmt.Errorf("Unsupported metric {%v}", metric)
	} else if metric.Symbol != (Symbol{}) {
		// single metric
		// fmt.Printf("parseMetric/Parsing Single Metric: %s\n", metric.Symbol)

		return (parseSymbolMetric(SymbolMetric{MIB: metric.MIB, Symbol: metric.Symbol, ForcedType: metric.MetricType, MetricTags: castedStringMetricTags, Options: metric.Options}))
	} else if metric.Table != nil {
		//table
		// fmt.Printf("parseMetric/Parsing Table: %s\n", metric.Table)

		if len(metric.MetricTags) > 0 {
			fmt.Println("parseMetric MetricTags switch, have:")
			spew.Dump(metric.MetricTags)
			for _,rawItem := range metric.MetricTags{
				item, ok := rawItem.(map[string]interface{})
    			if !ok {
        		continue
    			}
				fmt.Println("ITEM", item["symbol"])
				
				var index int
				if val, exists := item["Index"]; exists {
					if i, ok := val.(int); ok {
						index = i
					} else {
						index = -1
					}
				}

				// For a mapping, you can simply check and assign.
				var mapping map[int]string
				if val, exists := item["Mapping"]; exists {
					if m, ok := val.(map[int]string); ok {
						mapping = m
					}
				}

				// For a string value:
				var tag string
				if val, exists := item["Tag"]; exists {
					if s, ok := val.(string); ok {
						tag = s
					}
				}

				// For Symbol, if you have your custom unmarshaler set up:
				var symbol Symbol
				if rawSymbol, exists := item["symbol"]; exists {
					if symMap, ok := rawSymbol.(map[string]interface{}); ok {
						var oid, name string
				
						if v, exists := symMap["OID"]; exists {
							if oidStr, ok := v.(string); ok {
								oid = oidStr
							}
						}
				
						if v, exists := symMap["name"]; exists {
							if nameStr, ok := v.(string); ok {
								name = nameStr
							}
						}
				
						symbol = Symbol{OID: oid, Name: name}
					} else {
						// Optionally handle the case where the type assertion fails.
						fmt.Println("symbol is not a map[string]interface{}")
					}
				}

				// For Table (assuming it's a string here):
				var table string
				if val, exists := item["Table"]; exists {
					if s, ok := val.(string); ok {
						table = s
					} else {
						table=""
					}
				}

				// For Table (assuming it's a string here):
				var mib string
				if val, exists := item["MIB"]; exists {
					if s, ok := val.(string); ok {
						mib = s
					} else {
						mib=""
					}
				}

				// For IndexTransform (assuming []IndexSlice is defined somewhere)
				var indexTransform []IndexSlice
				if val, exists := item["IndexTransform"]; exists {
					if xs, ok := val.([]IndexSlice); ok {
						indexTransform = xs
					}
				}

				castedTableMetricTags = append(castedTableMetricTags, TableMetricTag{Index: index,Mapping: mapping,Tag: tag, MIB: mib,Symbol: symbol,Table: table,IndexTransform: indexTransform})
					// oid, okOID := s["OID"].(string)
					// name, okName := s["name"].(string)

					// if !okOID || !okName {
					// 	fmt.Errorf("invalid symbol format: %+v", s)
					// 	return ParsedSymbol{}
					// }


				
		}}

		if metric.Symbols == nil {
			fmt.Errorf("When specifying a table, you must specify a list of symbols %s", metric)
		}

		return (parseTableMetric(TableMetric{MIB: metric.MIB, Table: metric.Table, Symbols: metric.Symbols, ForcedType: metric.MetricType, MetricTags: castedTableMetricTags, Options: metric.Options}))

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

	// TODO can't find a profile with this metric type

	parsed_symbol_metric := ParsedSymbolMetric{
		Name:          name,
		Tags:          metric.MetricTags,
		ForcedType:    metric.ForcedType,
		EnforceScalar: true,
		Options:       metric.Options,
		baseoid:       oid,
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
		baseoid:             parsed_symbol.OID,
	}
	fmt.Println("parseSymbolMetric metric parsed result:", MetricParseResult{
		oidsToFetch:   []string{parsed_symbol.OID},
		oidsToResolve: parsed_symbol.OIDsToResolve,
		parsedMetrics: []ParsedMetric{parsed_symbol_metric},
		tableBatches:  nil,
		indexMappings: nil,
	})

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
	// fmt.Printf("attempting to parse table with parseSymbol %s\n", metric.Table)
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
			
			// fmt.Println("====================raw parsedtablemetrictag inside parseTableMetric")
			// spew.Dump(parsed_table_metric_tag)

			if parsed_table_metric_tag.OIDsToResolve != nil {
				oids_to_resolve = mergeStringMaps(oids_to_resolve, parsed_table_metric_tag.OIDsToResolve)

				column_tags = append(column_tags, parsed_table_metric_tag.ColumnTags...)

				fmt.Println("====================column_tags")
				spew.Dump(column_tags)


				table_batches = mergeTableBatches(table_batches, parsed_table_metric_tag.TableBatches)
			} else {

				index_tags = append(index_tags, parsed_table_metric_tag.IndexTags...)

				for index, mapping := range parsed_table_metric_tag.IndexMappings {
					for _, symbol := range metric.Symbols {
						index_mappings = append(index_mappings, IndexMapping{Tag: symbol.Name, Index: index, Mapping: mapping})
					}

					for _, tag := range metric.MetricTags {
						if reflect.DeepEqual(tag.Symbol, Symbol{}) {
							tag = TableMetricTag{
								Tag:    tag.Tag,
								Symbol: tag.Symbol,
							}
							index_mappings = append(index_mappings, IndexMapping{
								Tag:     tag.Symbol.Name,
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

		// fmt.Printf("PARSED SYMBOL\n")
		// spew.Dump(parsed_symbol)
		// fmt.Printf("\n\nINDEX TAGS\n")
		// spew.Dump(column_tags)

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
			baseoid:             parsed_symbol.OID,
		}

		// fmt.Printf("PARSED TABLE METRIC\n")
		// spew.Dump(parsed_table_metric)

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
	if metric_tag.Symbol != (Symbol{}) {
		metric_tag_mib := metric_tag.MIB

		if metric_tag.Table != "" {
			return parseOtherTableColumnMetricTag(metric_tag_mib, metric_tag.Table, metric_tag)
		}

		if mib != metric_tag_mib {
			fmt.Errorf("When tagging from a different MIB, the table must be specified")
		}
		fmt.Println("\n\n\n\nPARSECOLUMNMETRICTAG\n")
		spew.Dump(parseColumnMetricTag(mib, parsed_table, metric_tag))
		fmt.Println("END OF OUTPUT")
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
	parsed_column := parseSymbol(mib, metric_tag.Symbol)

	// fmt.Println("PARSED COLUMN")
	// spew.Dump(parsed_column)

	batches := map[TableBatchKey]TableBatch{TableBatchKey{MIB: mib, Table: parsed_table.Name}: TableBatch{TableOID: parsed_table.OID, OIDs: []string{parsed_column.OID}}}

	return ParsedTableMetricTag{
		OIDsToResolve: parsed_column.OIDsToResolve,
		ColumnTags: []ColumnTag{{
			DEDUPParsedMetricTag: parseMetricTag(MetricTag{MIB: metric_tag.MIB, OID: "", Tag: metric_tag.Tag, Symbol: metric_tag.Symbol}),
			Column:               parsed_column.Name,
			IndexSlices:          parseIndexSlices(metric_tag),
		},
		},
		TableBatches: batches,
	}
}

/*
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
*/
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

	// fmt.Printf("ParsingSymbol %s for MIB %s with type %s\n", symbol, mib, reflect.TypeOf(symbol))

	switch s := symbol.(type) {
	case Symbol:
		// fmt.Printf("Symbol found\n")
		oid := s.OID
		name := s.Name
		if s.ExtractValue != "" {
			extractValuePattern, err := regexp.Compile(s.ExtractValue)
			if err != nil {

				return ParsedSymbol{}
				// return "", fmt.Errorf("Failed to compile regular expression %q: %v", symbol.ExtractValue, err)
			}
			// fmt.Printf("Returning regexcase %s", ParsedSymbol{
			// 	name,
			// 	oid,
			// 	extractValuePattern,
			// 	map[string]string{name: oid},
			// })
			return ParsedSymbol{
				name,
				oid,
				extractValuePattern,
				map[string]string{name: oid},
			}
		} else {
			// fmt.Printf("Returning %s\n", ParsedSymbol{
			// 	name,
			// 	oid,
			// 	nil,
			// 	map[string]string{name: oid},
			// })
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
