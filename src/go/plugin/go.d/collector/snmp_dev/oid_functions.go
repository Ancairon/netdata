package snmp_dev

import (
	"fmt"
	"strconv"
	"strings"
)

func newOIDTreeNode() *OIDTreeNode {
	return &OIDTreeNode{Children: make(map[int]*OIDTreeNode)}
}

func NewOIDTrie() *OIDTrie {
	return &OIDTrie{Root: newOIDTreeNode()}
}

func (t *OIDTrie) Set(parts []int, name string) {
	node := t.Root
	for _, part := range parts {
		child, ok := node.Children[part]
		if !ok {
			child = newOIDTreeNode()
			node.Children[part] = child
		}
		node = child
	}
	node.Name = &name
}
func (t *OIDTrie) Match(parts []int) (matched []int, name *string) {
	node := t.Root
	var result []int
	var lastName *string
	for _, part := range parts {
		child, ok := node.Children[part]
		if !ok {
			break
		}
		node = child
		result = append(result, part)
		if node.Name != nil {
			lastName = node.Name
		}
	}
	return result, lastName
}

func NewOIDResolver(mibViewController MibViewController, enforceConstraints bool) *OIDResolver {
	return &OIDResolver{
		MibViewController:  mibViewController,
		Resolver:           NewOIDTrie(),
		IndexResolvers:     make(map[string]map[int]map[int]string),
		EnforceConstraints: enforceConstraints,
	}
}

func (r *OIDResolver) Register(oid *OID, name string) {
	r.Resolver.Set(oid.Parts, name)
}

func (r *OIDResolver) RegisterIndex(tag string, index int, mapping map[int]string) {
	if _, ok := r.IndexResolvers[tag]; !ok {
		r.IndexResolvers[tag] = make(map[int]map[int]string)
	}
	r.IndexResolvers[tag][index] = mapping
}

func (r *OIDResolver) resolveFromMibs(oid *OID) OIDMatch {
	if !r.EnforceConstraints {
		// Normally, force MIB resolution here.
	}
	ms := getMIBSymbol(oid)
	return OIDMatch{
		Name:    ms.Symbol,
		Indexes: ms.Prefix,
	}
}

func (r *OIDResolver) resolveTagIndex(tail []int, name string) []string {
	mappings, ok := r.IndexResolvers[name]
	if !ok {
		var tags []string
		for _, part := range tail {
			tags = append(tags, strconv.Itoa(part))
		}
		return tags
	}
	var tags []string
	for i, part := range tail {
		idx := i + 1
		if mapping, exists := mappings[idx]; exists {
			if tag, ok := mapping[part]; ok {
				tags = append(tags, tag)
			} else {
				tags = append(tags, strconv.Itoa(part))
			}
		} else {
			tags = append(tags, strconv.Itoa(part))
		}
	}
	return tags
}

// NewOIDFromString parses a dot-separated string into an OID.
func NewOIDFromString(value string) OID {
	value = strings.TrimPrefix(value, ".")
	partsStr := strings.Split(value, ".")
	parts := make([]int, len(partsStr))
	for i, p := range partsStr {
		n, err := strconv.Atoi(p)
		if err != nil {
			panic(fmt.Sprintf("Invalid OID part %q: %v", p, err))
		}
		parts[i] = n
	}
	return OID{Parts: parts}
}
func (o OID) String() string {
	var strParts []string
	for _, part := range o.Parts {
		strParts = append(strParts, strconv.Itoa(part))
	}
	return strings.Join(strParts, ".")
}

// isPrefix checks whether the OID string "oid" starts with the prefix "prefix".
// It splits the OIDs on dots and compares numeric components.
func isPrefix(prefix, oid string) bool {
	// Remove any leading dot.
	prefix = strings.TrimPrefix(prefix, ".")
	oid = strings.TrimPrefix(oid, ".")

	prefixParts := strings.Split(prefix, ".")
	oidParts := strings.Split(oid, ".")

	if len(prefixParts) > len(oidParts) {
		return false
	}
	for i, part := range prefixParts {
		if part != oidParts[i] {
			return false
		}
	}
	return true
}
func (r *OIDResolver) ResolveOID(oid *OID) OIDMatch {
	parts := oid.Parts
	prefix, namePtr := r.Resolver.Match(parts)
	if namePtr == nil {
		return r.resolveFromMibs(oid)
	}
	tail := parts[len(prefix):]
	tagIndex := r.resolveTagIndex(tail, *namePtr)
	return OIDMatch{
		Name:    *namePtr,
		Indexes: tagIndex,
	}
}

func getMIBSymbol(oid *OID) MIBSymbol {
	// Dummy implementation.
	return MIBSymbol{
		Mib:    "DummyMIB",
		Symbol: "dummySymbol",
		Prefix: []string{},
	}
}

// FormatAsOIDString formats a slice of ints as a dot-separated string.
func FormatAsOIDString(parts []int) string {
	var strParts []string
	for _, part := range parts {
		strParts = append(strParts, strconv.Itoa(part))
	}
	return strings.Join(strParts, ".")
}
