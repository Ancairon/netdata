// pysnmp_types.go
package snmp_dev

import "regexp"

// Asn1Type is a placeholder for pyasn1.type.base.Asn1Type.
type Asn1Type interface{}

// CommunityData represents SNMP community data.
type CommunityData struct {
	Community string
	MpModel   int // 0 for SNMPv1, 1 for SNMPv2
}

// ParseMetricsResult is the output of parsing the metrics section.
type ParseMetricsResult struct {
	Oids          []OID
	NextOids      []OID
	BulkOids      []OID
	ParsedMetrics []ParsedMetric
} // Intermediate types used for parsed results:
type IndexMapping struct {
	Tag     string
	Index   int
	Mapping map[string]interface{}
}
type TableBatchKey struct {
	Mib   string
	Table string
}
type TableBatch struct {
	TableOid OID
	Oids     []OID
}

type IndexTag struct {
	ParsedMetricTag ParsedMetricTag
	Index           int
}

type ColumnTag struct {
	ParsedMetricTag ParsedMetricTag
	Column          string
	IndexSlices     []IndexSlice
}

// OIDTreeNode is a node in the OID trie.
type OIDTreeNode struct {
	Name     *string
	Children map[int]*OIDTreeNode
}

// ParsedSymbolMetric holds data for a symbol metric.
type ParsedSymbolMetric struct {
	Name                string
	Tags                []string
	ForcedType          string
	EnforceScalar       bool
	Options             map[string]interface{}
	ExtractValuePattern *regexp.Regexp
}

// Device represents an SNMP device.
type Device struct {
	IP     string
	Port   int
	Target string
}

// ParsedMatchMetricTag holds a regex-based parsed metric tag.
type ParsedMatchMetricTag struct {
	Tags    map[string]string
	Pattern *regexp.Regexp
}

// ParsedMetricTag is a union of ParsedSimpleMetricTag and ParsedMatchMetricTag.
type ParsedMetricTag interface {
	MatchedTags(value interface{}) []string
}

// ParsedTableMetric holds data for a table metric.
type ParsedTableMetric struct {
	Name                string
	IndexTags           []any
	ColumnTags          []any
	ForcedType          string
	Options             map[string]interface{}
	ExtractValuePattern *regexp.Regexp
}

// ParsedMetric is a union of ParsedSymbolMetric and ParsedTableMetric.
type ParsedMetric interface {
	GetName() string
}

// ParsedSimpleMetricTag holds a simple parsed metric tag.
type ParsedSimpleMetricTag struct {
	Name string
}

// IndexSlice represents a slice with a start and end index.
type IndexSlice struct {
	Start int
	Stop  int
} // OIDTrie stores OID prefixes.
type OIDTrie struct {
	Root *OIDTreeNode
} // OIDMatch represents the result of resolving an OID.
type OIDMatch struct {
	Name    string
	Indexes []string
}

// OIDResolver resolves OIDs.
type OIDResolver struct {
	MibViewController  MibViewController
	Resolver           *OIDTrie
	IndexResolvers     map[string]map[int]map[int]string
	EnforceConstraints bool
}

// MIBSymbol is a dummy structure returned by getMIBSymbol.
type MIBSymbol struct {
	Mib    string
	Symbol string
	Prefix []string
}
type ParsedColumnMetricTag struct {
	OidsToResolve map[string]OID
	TableBatches  map[TableBatchKey]TableBatch
	ColumnTags    []ColumnTag
}

type ParsedIndexMetricTag struct {
	IndexTags     []IndexTag
	IndexMappings map[int]map[string]interface{}
}
type ParsedTableMetricTag interface{}

// ParsedSymbol holds parsed data from a symbol.
type ParsedSymbol struct {
	name                string
	oid                 OID
	extractValuePattern *regexp.Regexp
	oidsToResolve       map[string]OID
}
type TableBatches map[TableBatchKey]TableBatch

// MetricParseResult holds intermediary parsed data.
type MetricParseResult struct {
	OidsToFetch   []OID
	OidsToResolve map[string]OID
	IndexMappings []IndexMapping
	TableBatches  TableBatches
	ParsedMetrics []ParsedMetric
}

type MetricTag struct {
	Symbol string            `json:"symbol,omitempty"`
	MIB    string            `json:"MIB,omitempty"`
	OID    string            `json:"OID,omitempty"`
	Tag    string            `json:"tag,omitempty"`
	Match  string            `json:"match,omitempty"`
	Tags   map[string]string `json:"tags,omitempty"`
}
type MetricTagParseResult struct {
	Oid           OID
	SymbolTag     SymbolTag
	OidsToResolve map[string]OID
}
type SymbolTag struct {
	ParsedMetricTag ParsedMetricTag
	Symbol          string
}
type metricTagParseResult struct {
	Oid           OID
	SymbolTag     SymbolTag
	OidsToResolve map[string]OID
}

// OID represents an SNMP Object Identifier.
type OID struct {
	Parts []int
}

// ContextData represents SNMP context parameters.
type ContextData struct {
	ContextEngineID string
	ContextName     string
}

// ObjectIdentity represents an SNMP object identity.
type ObjectIdentity interface {
	ResolveWithMib(mibViewController MibViewController) error
	GetMibSymbol() (mib string, symbol string, indexes []ObjectName, err error)
}

// ObjectType represents an SNMP object type.
type ObjectType interface {
	GetObjectIdentity() ObjectIdentity
}

// SnmpEngine represents the SNMP engine.
type SnmpEngine struct {
	MsgAndPduDsp *MsgAndPduDispatcher
}

// UdpTransportTarget represents the UDP transport target for SNMP queries.
type UdpTransportTarget struct {
	Address string
	Port    int
	Timeout float64
	Retries int
}

// NewUdpTransportTarget creates a new UDP transport target.
func NewUdpTransportTarget(ip string, port int, timeout float64, retries int) UdpTransportTarget {
	return UdpTransportTarget{
		Address: ip,
		Port:    port,
		Timeout: timeout,
		Retries: retries,
	}
}

// UsmUserData represents SNMPv3 user data.
type UsmUserData struct {
	User         string
	AuthKey      string
	PrivKey      string
	AuthProtocol interface{}
	PrivProtocol interface{}
}

var UsmDESPrivProtocol = "usmDESPrivProtocol"
var UsmHMACMD5AuthProtocol = "usmHMACMD5AuthProtocol"

// lcd is a placeholder.
var lcd = "lcd_placeholder"

// AbstractTransportTarget is a placeholder.
type AbstractTransportTarget interface{}

// ObjectName represents an SNMP object name.
type ObjectName interface {
	PrettyPrint() string
}

// Opaque represents an opaque SNMP type.
type Opaque interface{}

// MsgAndPduDispatcher is a placeholder.
type MsgAndPduDispatcher struct{}

// DirMibSource represents a directory containing MIBs.
type DirMibSource string

// MibBuilder is a placeholder.
type MibBuilder struct{}

// MibInstrumController is a placeholder.
type MibInstrumController struct {
	Builder *MibBuilder
}

// MibViewController is a placeholder.
type MibViewController struct {
	Builder *MibBuilder
}

var EndOfMibView = "endOfMibView"
var NoSuchInstance = "noSuchInstance"
var NoSuchObject = "noSuchObject"
