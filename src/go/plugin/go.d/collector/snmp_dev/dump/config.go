// config.go
// (C) Datadog, Inc. 2010-present (translated)
// All rights reserved
// Package snmp_dev contains configuration and setup for the SNMP collector.
package snmp_dev

// // --- Constants for defaults ---
// const (
// 	DefaultRetries                  = 5
// 	DefaultTimeout                  = 5
// 	DefaultAllowedFailures          = 3
// 	DefaultBulkThreshold            = 0
// 	DefaultWorkers                  = 5
// 	DefaultRefreshOidsCacheInterval = 0 // 0 means disabled
// )

// // Placeholder constructors / functions.
// // You’ll need to implement or integrate these as needed.
// func NewMIBLoader() *MIBLoader                                             { return &MIBLoader{} }
// func (m *MIBLoader) CreateSnmpEngine(mibsPath string) (interface{}, error) { return struct{}{}, nil }
// func (m *MIBLoader) GetMibViewController(mibsPath string) (interface{}, error) {
// 	return struct{}{}, nil
// }
// func NewOIDResolver(mibViewController interface{}, enforceConstraints bool) *OIDResolver {
// 	return &OIDResolver{}
// }
// func (o *OIDResolver) ResolveOID(oid OID) (OIDMatch, error) { return OIDMatch{}, nil }
// func (o *OIDResolver) Register(oid OID, name string)        {}
// func parseMetrics(metrics []interface{}, resolver *OIDResolver, logger *log.Logger, bulkThreshold int) (scalarOids []OID, nextOids []OID, bulkOids []OID, parsedMetrics []ParsedMetric, err error) {
// 	// Implement your parsing logic here.
// 	return nil, nil, nil, nil, nil
// }
// func parseMetricTags(metricTags []interface{}, resolver *OIDResolver) (oids []OID, parsedTags []SymbolTag, err error) {
// 	// Implement your parsing logic here.
// 	return nil, nil, nil
// }
// func registerDeviceTarget(ip string, port, timeout, retries int, engine interface{}, authData interface{}, ctx ContextData) (interface{}, error) {
// 	// Implement your device target registration logic.
// 	return struct{}{}, nil
// }
// func NewCommunityData(community string, mpModel int) *CommunityData {
// 	return &CommunityData{Community: community, MpModel: mpModel}
// }
// func NewUsmUserData(user, authKey, privKey string, authProtocol, privProtocol interface{}) *UsmUserData {
// 	return &UsmUserData{User: user, AuthKey: authKey, PrivKey: privKey, AuthProtocol: authProtocol, PrivProtocol: privProtocol}
// }
// func DefaultAuthProtocol() interface{} { return nil }
// func DefaultPrivProtocol() interface{} { return nil }
// func NewOID(value string) OID          { return OID{Value: value} }
// func NewParsedSymbolMetric(name, forcedType string) ParsedMetric {
// 	return ParsedMetric{Name: name, ForcedType: forcedType}
// }

// // NewInstanceConfig initializes an InstanceConfig based on the provided configuration.
// // Many parameters are taken as raw maps or slices (from unmarshaled JSON/YAML).
// func NewInstanceConfig(
// 	instance map[string]interface{},
// 	globalMetrics []interface{},
// 	mibsPath string,
// 	refreshOidsCacheInterval int,
// 	profiles map[string]map[string]interface{},
// 	profilesByOid map[string]string,
// 	loader *MIBLoader,
// 	logger *log.Logger,
// ) (*InstanceConfig, error) {

// 	// Use defaults if nil.
// 	if globalMetrics == nil {
// 		globalMetrics = []interface{}{}
// 	}
// 	if profiles == nil {
// 		profiles = map[string]map[string]interface{}{}
// 	}
// 	if profilesByOid == nil {
// 		profilesByOid = map[string]string{}
// 	}
// 	if loader == nil {
// 		loader = NewMIBLoader()
// 	}
// 	if logger == nil {
// 		logger = log.Default()
// 	}

// 	// Clean empty or nil values from instance.
// 	for key, value := range instance {
// 		if value == nil {
// 			delete(instance, key)
// 		} else if str, ok := value.(string); ok && str == "" {
// 			delete(instance, key)
// 		}
// 	}

// 	cfg := &InstanceConfig{
// 		Instance:            instance,
// 		Tags:                []string{},
// 		Metrics:             []interface{}{},
// 		MetricTags:          []interface{}{},
// 		DiscoveredInstances: make(map[string]*InstanceConfig),
// 		FailingInstances:    make(map[string]int),
// 		Logger:              logger,
// 		IgnoredIPAddresses:  make(map[string]struct{}),
// 	}

// 	// Populate Tags.
// 	if tags, ok := instance["tags"].([]interface{}); ok {
// 		for _, t := range tags {
// 			if s, ok := t.(string); ok {
// 				cfg.Tags = append(cfg.Tags, s)
// 			}
// 		}
// 	}

// 	// Populate Metrics and MetricTags.
// 	if metrics, ok := instance["metrics"].([]interface{}); ok {
// 		cfg.Metrics = metrics
// 	}
// 	if metricTags, ok := instance["metric_tags"].([]interface{}); ok {
// 		cfg.MetricTags = metricTags
// 	}

// 	// Get profile name.
// 	var profile string
// 	if p, ok := instance["profile"].(string); ok {
// 		profile = p
// 	}

// 	// If use_global_metrics is affirmative, append globalMetrics.
// 	if isAffirmative(instance["use_global_metrics"], true) {
// 		cfg.Metrics = append(cfg.Metrics, globalMetrics...)
// 	}

// 	cfg.EnforceConstraints = isAffirmative(instance["enforce_mib_constraints"], true)

// 	// Create SNMP engine and resolver.
// 	snmpEngine, err := loader.CreateSnmpEngine(mibsPath)
// 	if err != nil {
// 		return nil, err
// 	}
// 	cfg.SnmpEngine = snmpEngine

// 	mibViewController, err := loader.GetMibViewController(mibsPath)
// 	if err != nil {
// 		return nil, err
// 	}
// 	cfg.Resolver = NewOIDResolver(mibViewController, cfg.EnforceConstraints)

// 	// Set discovery-related fields.
// 	cfg.Device = nil
// 	cfg.IPNetwork = nil
// 	cfg.AllowedFailures = getIntFromMap(instance, "discovery_allowed_failures", DefaultAllowedFailures)
// 	cfg.Workers = getIntFromMap(instance, "workers", DefaultWorkers)
// 	cfg.BulkThreshold = getIntFromMap(instance, "bulk_threshold", DefaultBulkThreshold)

// 	// Get authentication and context data.
// 	authData, err := getAuthData(instance)
// 	if err != nil {
// 		return nil, err
// 	}
// 	cfg.AuthData = authData
// 	cfg.ContextData = getContextData(instance)

// 	timeout := getIntFromMap(instance, "timeout", DefaultTimeout)
// 	retries := getIntFromMap(instance, "retries", DefaultRetries)

// 	// Get IP or network address.
// 	ipAddress, ipOk := instance["ip_address"].(string)
// 	networkAddress, netOk := instance["network_address"].(string)

// 	if !ipOk && !netOk {
// 		return nil, errors.New("an IP address or a network address needs to be specified")
// 	}
// 	if ipOk && netOk {
// 		return nil, errors.New("only one of IP address and network address must be specified")
// 	}

// 	if ipOk {
// 		port := getIntFromMap(instance, "port", 161)
// 		target, err := registerDeviceTarget(ipAddress, port, timeout, retries, snmpEngine, authData, cfg.ContextData)
// 		if err != nil {
// 			return nil, err
// 		}
// 		device := &Device{
// 			IP:     ipAddress,
// 			Port:   port,
// 			Target: target,
// 			Tags:   []string{}, // Populate device tags as needed.
// 		}
// 		cfg.Device = device
// 		cfg.Tags = append(cfg.Tags, device.Tags...)
// 	}

// 	if netOk {
// 		_, ipnet, err := net.ParseCIDR(networkAddress)
// 		if err != nil {
// 			return nil, fmt.Errorf("invalid network_address: %v", err)
// 		}
// 		cfg.IPNetwork = ipnet
// 	}

// 	// Process ignored_ip_addresses.
// 	if ignored, ok := instance["ignored_ip_addresses"].([]interface{}); ok {
// 		for _, ip := range ignored {
// 			if s, ok := ip.(string); ok {
// 				cfg.IgnoredIPAddresses[s] = struct{}{}
// 			}
// 		}
// 	} else if _, exists := instance["ignored_ip_addresses"]; exists {
// 		return nil, errors.New("ignored_ip_addresses should be a list")
// 	}

// 	if len(cfg.Metrics) == 0 && len(profilesByOid) == 0 && profile == "" {
// 		return nil, errors.New("instance should specify at least one metric or profiles should be defined")
// 	}

// 	// Parse metrics.
// 	scalarOids, nextOids, bulkOids, parsedMetrics, err := parseMetrics(cfg.Metrics, cfg.Resolver, logger, cfg.BulkThreshold)
// 	if err != nil {
// 		return nil, err
// 	}
// 	cfg.ParsedMetrics = parsedMetrics

// 	tagOids, parsedMetricTags, err := parseMetricTags(cfg.MetricTags, cfg.Resolver)
// 	if err != nil {
// 		return nil, err
// 	}
// 	cfg.ParsedMetricTags = parsedMetricTags

// 	if len(tagOids) > 0 {
// 		scalarOids = append(scalarOids, tagOids...)
// 	}

// 	refreshIntervalSec := getIntFromMap(instance, "refresh_oids_cache_interval", refreshOidsCacheInterval)
// 	cfg.OidConfig = NewOIDConfig(refreshIntervalSec)
// 	cfg.OidConfig.AddParsedOids(scalarOids, nextOids, bulkOids)

// 	if profile != "" {
// 		prof, ok := profiles[profile]
// 		if !ok {
// 			return nil, fmt.Errorf("unknown profile '%s'", profile)
// 		}
// 		if err := cfg.RefreshWithProfile(prof); err != nil {
// 			return nil, err
// 		}
// 		cfg.AddProfileTag(profile)
// 	}

// 	cfg.UptimeMetricAdded = false

// 	return cfg, nil
// }

// // --- Helper Functions ---

// func getIntFromMap(m map[string]interface{}, key string, defaultVal int) int {
// 	if val, ok := m[key]; ok {
// 		switch v := val.(type) {
// 		case int:
// 			return v
// 		case float64:
// 			return int(v)
// 		case string:
// 			var i int
// 			if _, err := fmt.Sscanf(v, "%d", &i); err == nil {
// 				return i
// 			}
// 		}
// 	}
// 	return defaultVal
// }

// // isAffirmative returns true if the value is considered affirmative.
// func isAffirmative(value interface{}, defaultVal bool) bool {
// 	if value == nil {
// 		return defaultVal
// 	}
// 	switch v := value.(type) {
// 	case bool:
// 		return v
// 	case string:
// 		lower := strings.ToLower(v)
// 		return lower == "true" || lower == "yes" || lower == "1"
// 	case int:
// 		return v != 0
// 	case float64:
// 		return v != 0
// 	default:
// 		return defaultVal
// 	}
// }

// // getAuthData constructs authentication data based on the instance configuration.
// func getAuthData(instance map[string]interface{}) (interface{}, error) {
// 	if comm, ok := instance["community_string"].(string); ok {
// 		snmpVersion := 2
// 		if v, ok := instance["snmp_version"]; ok {
// 			switch t := v.(type) {
// 			case int:
// 				snmpVersion = t
// 			case string:
// 				fmt.Sscanf(t, "%d", &snmpVersion)
// 			}
// 		}
// 		if snmpVersion == 1 {
// 			return NewCommunityData(comm, 0), nil
// 		}
// 		return NewCommunityData(comm, 1), nil
// 	}
// 	if user, ok := instance["user"].(string); ok {
// 		authKey, _ := instance["authKey"].(string)
// 		privKey, _ := instance["privKey"].(string)
// 		var authProtocol, privProtocol interface{}
// 		if _, ok := instance["authProtocol"]; ok {
// 			authProtocol = DefaultAuthProtocol()
// 		}
// 		if _, ok := instance["privProtocol"]; ok {
// 			privProtocol = DefaultPrivProtocol()
// 		}
// 		return NewUsmUserData(user, authKey, privKey, authProtocol, privProtocol), nil
// 	}
// 	return nil, errors.New("an authentication method needs to be provided")
// }

// // getContextData constructs context parameters based on the instance configuration.
// func getContextData(instance map[string]interface{}) ContextData {
// 	contextEngineID := ""
// 	contextName := ""
// 	if _, ok := instance["user"]; ok {
// 		if val, ok := instance["context_engine_id"].(string); ok {
// 			contextEngineID = val
// 		}
// 		if val, ok := instance["context_name"].(string); ok {
// 			contextName = val
// 		}
// 	}
// 	return ContextData{
// 		ContextEngineID: contextEngineID,
// 		ContextName:     contextName,
// 	}
// }

// // --- InstanceConfig Methods ---

// // ResolveOID returns the resolved OID match.
// func (cfg *InstanceConfig) ResolveOID(oid OID) (OIDMatch, error) {
// 	return cfg.Resolver.ResolveOID(oid)
// }

// // RefreshWithProfile updates the configuration with metrics and tags from the given profile.
// func (cfg *InstanceConfig) RefreshWithProfile(profile map[string]interface{}) error {
// 	def, ok := profile["definition"].(map[string]interface{})
// 	if !ok {
// 		return errors.New("invalid profile definition")
// 	}
// 	metricsIface, _ := def["metrics"].([]interface{})
// 	scalarOids, nextOids, bulkOids, parsedMetrics, err := parseMetrics(metricsIface, cfg.Resolver, cfg.Logger, cfg.BulkThreshold)
// 	if err != nil {
// 		return err
// 	}
// 	metricTagsIface, _ := def["metric_tags"].([]interface{})
// 	tagOids, parsedMetricTags, err := parseMetricTags(metricTagsIface, cfg.Resolver)
// 	if err != nil {
// 		return err
// 	}
// 	deviceIface, _ := def["device"].(map[string]interface{})
// 	cfg.AddDeviceTags(deviceIface)

// 	// Note: Duplication of metrics/tags is possible.
// 	cfg.Metrics = append(cfg.Metrics, metricsIface...)
// 	cfg.OidConfig.AddParsedOids(append(scalarOids, tagOids...), nextOids, bulkOids)
// 	cfg.ParsedMetrics = append(cfg.ParsedMetrics, parsedMetrics...)
// 	cfg.ParsedMetricTags = append(cfg.ParsedMetricTags, parsedMetricTags...)
// 	return nil
// }

// // AddProfileTag appends a profile tag.
// func (cfg *InstanceConfig) AddProfileTag(profileName string) {
// 	cfg.Tags = append(cfg.Tags, fmt.Sprintf("snmp_profile:%s", profileName))
// }

// // AddDeviceTags appends device tags from the provided device map.
// func (cfg *InstanceConfig) AddDeviceTags(device map[string]interface{}) {
// 	supported := []string{"vendor"}
// 	for _, tagKey := range supported {
// 		if val, ok := device[tagKey].(string); ok && val != "" {
// 			cfg.Tags = append(cfg.Tags, fmt.Sprintf("device_%s:%s", tagKey, val))
// 		}
// 	}
// }

// // NetworkHosts returns a slice of host IPs within the configured network,
// // excluding those already discovered or ignored. (IPv4 only.)
// func (cfg *InstanceConfig) NetworkHosts() ([]string, error) {
// 	if cfg.IPNetwork == nil {
// 		return nil, errors.New("expected ip_network to be set to iterate over network hosts")
// 	}
// 	var hosts []string
// 	ip := cfg.IPNetwork.IP.To4()
// 	if ip == nil {
// 		return nil, errors.New("only IPv4 is supported in network_hosts")
// 	}
// 	maskSize, bits := cfg.IPNetwork.Mask.Size()
// 	total := 1 << (bits - maskSize)
// 	start := ipToInt(ip)
// 	for i := 1; i < total-1; i++ {
// 		hostIP := intToIP(start + i)
// 		hostStr := hostIP.String()
// 		if _, exists := cfg.DiscoveredInstances[hostStr]; exists {
// 			continue
// 		}
// 		if _, ignored := cfg.IgnoredIPAddresses[hostStr]; ignored {
// 			continue
// 		}
// 		hosts = append(hosts, hostStr)
// 	}
// 	return hosts, nil
// }

// // AddUptimeMetric adds the sysUpTimeInstance metric if it hasn't been added already.
// func (cfg *InstanceConfig) AddUptimeMetric() {
// 	if cfg.UptimeMetricAdded {
// 		return
// 	}
// 	// Reference sysUpTimeInstance (OID 1.3.6.1.2.1.1.3.0).
// 	uptimeOID := NewOID("1.3.6.1.2.1.1.3.0")
// 	cfg.OidConfig.AddParsedOids([]OID{uptimeOID}, nil, nil)
// 	cfg.Resolver.Register(uptimeOID, "sysUpTimeInstance")
// 	parsedMetric := NewParsedSymbolMetric("sysUpTimeInstance", "gauge")
// 	cfg.ParsedMetrics = append(cfg.ParsedMetrics, parsedMetric)
// 	cfg.UptimeMetricAdded = true
// }

// func NewOIDConfig(refreshIntervalSec int) *OIDConfig {
// 	return &OIDConfig{
// 		RefreshIntervalSec: refreshIntervalSec,
// 		ScalarOids:         []OID{},
// 		NextOids:           []OID{},
// 		BulkOids:           []OID{},
// 		AllScalarOids:      []OID{},
// 		UseScalarOidsCache: false,
// 	}
// }

// // GetScalarOids returns scalar OIDs (using cache if enabled).
// func (o *OIDConfig) GetScalarOids() []OID {
// 	if o.UseScalarOidsCache {
// 		return o.AllScalarOids
// 	}
// 	return o.ScalarOids
// }

// // GetNextOids returns next OIDs (empty if cache is enabled).
// func (o *OIDConfig) GetNextOids() []OID {
// 	if o.UseScalarOidsCache {
// 		return []OID{}
// 	}
// 	return o.NextOids
// }

// // GetBulkOids returns bulk OIDs (empty if cache is enabled).
// func (o *OIDConfig) GetBulkOids() []OID {
// 	if o.UseScalarOidsCache {
// 		return []OID{}
// 	}
// 	return o.BulkOids
// }

// // AddParsedOids appends parsed OIDs and resets the cache.
// func (o *OIDConfig) AddParsedOids(scalarOids, nextOids, bulkOids []OID) {
// 	if scalarOids != nil {
// 		o.ScalarOids = append(o.ScalarOids, scalarOids...)
// 	}
// 	if nextOids != nil {
// 		o.NextOids = append(o.NextOids, nextOids...)
// 	}
// 	if bulkOids != nil {
// 		o.BulkOids = append(o.BulkOids, bulkOids...)
// 	}
// 	o.Reset()
// }

// // HasOids returns true if there are any OIDs to fetch.
// func (o *OIDConfig) HasOids() bool {
// 	return len(o.GetScalarOids()) > 0 || len(o.GetNextOids()) > 0 || len(o.GetBulkOids()) > 0
// }

// func (o *OIDConfig) isCacheEnabled() bool {
// 	return o.RefreshIntervalSec > 0
// }

// // UpdateScalarOids enables the scalar OIDs cache.
// func (o *OIDConfig) UpdateScalarOids(newScalarOids []OID) {
// 	if !o.isCacheEnabled() {
// 		return
// 	}
// 	if o.UseScalarOidsCache {
// 		return
// 	}
// 	o.AllScalarOids = newScalarOids
// 	o.UseScalarOidsCache = true
// 	o.LastTs = time.Now()
// }

// // ShouldReset returns true if the cache should be reset based on the refresh interval.
// func (o *OIDConfig) ShouldReset() bool {
// 	if !o.isCacheEnabled() {
// 		return false
// 	}
// 	elapsed := time.Since(o.LastTs)
// 	return int(elapsed.Seconds()) > o.RefreshIntervalSec
// }

// // Reset clears the scalar OIDs cache.
// func (o *OIDConfig) Reset() {
// 	o.AllScalarOids = []OID{}
// 	o.UseScalarOidsCache = false
// }

// // --- Utility Functions for IP Conversions (IPv4) ---
// func ipToInt(ip net.IP) int {
// 	ip = ip.To4()
// 	return int(ip[0])<<24 + int(ip[1])<<16 + int(ip[2])<<8 + int(ip[3])
// }

// func intToIP(n int) net.IP {
// 	return net.IPv4(byte(n>>24), byte(n>>16&0xFF), byte(n>>8&0xFF), byte(n&0xFF))
// }
