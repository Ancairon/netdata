// snmp.go
package snmp_dev

import (
	"encoding/json"
	"fmt"
	"log"
	"net"
	"strconv"
	"strings"
	"time"
)

const (
	DefaultOidBatchSize      = 10
	MaxFetchNumber           = 1000000
	LoaderTag                = "loader:python"
	DefaultDiscoveryInterval = 3600.0
	StatusOK                 = 0
	StatusWarning            = 1
	StatusCritical           = 2
)

// AgentCheck is an interface for metric submission.
type AgentCheck interface {
	Gauge(metric string, value float64, tags []string)
	MonotonicCount(metric string, value float64, tags []string)
	Rate(metric string, value float64, tags []string)
	ServiceCheck(name string, status int, tags []string, message string)
	Warning(message string)
	Log() *log.Logger
}

// SnmpCheck implements the SNMP check.
type SnmpCheck struct {
	AgentCheck
	CheckID                  string
	InitConfig               map[string]interface{}
	Instance                 map[string]interface{}
	OidBatchSize             int
	MibsPath                 string
	OptimizeMibMemoryUsage   bool
	IgnoreNonincreasingOid   bool
	RefreshOidsCacheInterval int
	Profiles                 map[string]map[string]interface{}
	ProfilesByOid            map[string]string
	Config                   *InstanceConfig
	LastFetchNumber          int
	SubmittedMetrics         int
	Executor                 *WorkerPool
	DiscoveryStarted         bool
	Logger                   *log.Logger
}

func NewSnmpCheck(checkID string, initConfig, instance map[string]interface{}, ac AgentCheck) *SnmpCheck {
	sc := &SnmpCheck{
		AgentCheck:               ac,
		CheckID:                  checkID,
		InitConfig:               initConfig,
		Instance:                 instance,
		OidBatchSize:             intFromMap(initConfig, "oid_batch_size", DefaultOidBatchSize),
		MibsPath:                 getStringFromMap(initConfig, "mibs_folder", ""),
		OptimizeMibMemoryUsage:   isAffirmative(getStringFromMap(initConfig, "optimize_mib_memory_usage", "false")),
		IgnoreNonincreasingOid:   isAffirmative(getStringFromMap(initConfig, "ignore_nonincreasing_oid", "false")),
		RefreshOidsCacheInterval: intFromMap(initConfig, "refresh_oids_cache_interval", 0),
		Logger:                   ac.Log(),
	}
	sc.Profiles = sc.loadProfiles()
	sc.ProfilesByOid = sc.getProfilesMapping()
	sc.Config = sc.buildConfig(instance)
	sc.LastFetchNumber = 0
	sc.SubmittedMetrics = 0
	sc.Executor = NewWorkerPool(sc.Config.Workers)
	return sc
}

func (sc *SnmpCheck) getNextFetchID() string {
	sc.LastFetchNumber = (sc.LastFetchNumber + 1) % MaxFetchNumber
	return fmt.Sprintf("%s-%d", sc.CheckID, sc.LastFetchNumber)
}

func (sc *SnmpCheck) loadProfiles() map[string]map[string]interface{} {
	configured, ok := sc.InitConfig["profiles"]
	if !ok {
		return GetDefaultProfilesAsMap() // Assume this function returns defaults.
	}
	profiles := make(map[string]map[string]interface{})
	for name, p := range configured.(map[string]interface{}) {
		def, err := getProfileDefinition(p.(map[string]interface{}))
		if err != nil {
			sc.Logger.Fatalf("Couldn't read profile %s: %v", name, err)
		}
		if err := recursivelyExpandBaseProfiles(def); err != nil {
			sc.Logger.Printf("Failed to expand base profiles in profile %s: %v", name, err)
			continue
		}
		profiles[name] = map[string]interface{}{"definition": def}
	}
	return profiles
}

func (sc *SnmpCheck) getProfilesMapping() map[string]string {
	result := make(map[string]string)
	for name, profile := range sc.Profiles {
		def := profile["definition"].(map[string]interface{})
		sysOIDsRaw, ok := def["sysobjectid"]
		if !ok {
			continue
		}
		var sysOIDs []string
		switch v := sysOIDsRaw.(type) {
		case string:
			sysOIDs = []string{v}
		case []interface{}:
			for _, item := range v {
				sysOIDs = append(sysOIDs, fmt.Sprintf("%v", item))
			}
		default:
			continue
		}
		for _, sysOID := range sysOIDs {
			if existing, exists := result[sysOID]; exists {
				sc.Logger.Fatalf("Profile %s has the same sysObjectID (%s) as %s", name, sysOID, existing)
			}
			result[sysOID] = name
		}
	}
	return result
}

func (sc *SnmpCheck) buildConfig(instance map[string]interface{}) *InstanceConfig {
	var loader *MIBLoader
	if sc.OptimizeMibMemoryUsage {
		loader = SharedMIBLoader()
	} else {
		loader = NewMIBLoader()
	}
	config := NewInstanceConfig(instance, getOptionalInterfaceSlice(sc.InitConfig, "global_metrics"), sc.MibsPath, sc.RefreshOidsCacheInterval, sc.Profiles, sc.ProfilesByOid, loader, sc.Logger)
	return config
}

func (sc *SnmpCheck) buildAutodiscoveryConfig(sourceInstance map[string]interface{}, ipAddress string) *InstanceConfig {
	instanceCopy := copyMap(sourceInstance)
	networkAddress, _ := instanceCopy["network_address"].(string)
	delete(instanceCopy, "network_address")
	instanceCopy["ip_address"] = ipAddress
	tags, _ := instanceCopy["tags"].([]string)
	instanceCopy["tags"] = append(tags, fmt.Sprintf("autodiscovery_subnet:%s", networkAddress))
	return sc.buildConfig(instanceCopy)
}

func (sc *SnmpCheck) getInstanceName(instance map[string]interface{}) string {
	if name, ok := instance["name"].(string); ok && name != "" {
		return name
	}
	ip, ipOk := instance["ip_address"].(string)
	port, portOk := instance["port"].(int)
	if ipOk && portOk {
		return fmt.Sprintf("%s:%d", ip, port)
	} else if ipOk {
		return ip
	}
	return ""
}

func (sc *SnmpCheck) fetchResults(config *InstanceConfig) (map[string]map[string]interface{}, []OID, error) {
	results := make(map[string]map[string]interface{})
	enforceConstraints := config.EnforceConstraints
	fetchID := sc.getNextFetchID()
	var allBinds []Bind

	binds, err := SnmpGet(config, config.OidConfig.ScalarOids, enforceConstraints)
	if err != nil {
		return nil, nil, err
	}
	allBinds = append(allBinds, binds...)

	for _, oid := range config.OidConfig.BulkOids {
		objType := oid.AsObjectType()
		sc.Logger.Printf("[%s] Running SNMP bulk on OID %s", fetchID, OIDPrinter(objType, false))
		bulkBinds, err := SnmpBulk(config, objType, 0, 25, enforceConstraints, sc.IgnoreNonincreasingOid)
		if err != nil {
			sc.Logger.Printf("[%s] Bulk error: %v", fetchID, err)
		}
		allBinds = append(allBinds, bulkBinds...)
	}

	var scalarOids []OID
	for _, bind := range allBinds {
		oid := NewOIDFromString(bind.Oid)
		scalarOids = append(scalarOids, oid)
		match := config.ResolveOID(oid)
		key := strings.Join(match.Indexes, ",")
		if _, exists := results[match.Name]; !exists {
			results[match.Name] = make(map[string]interface{})
		}
		results[match.Name][key] = bind.Value
	}
	sc.Logger.Printf("[%s] Raw results: %s", fetchID, OIDPrinter(results, false))
	return results, scalarOids, nil
}

func (sc *SnmpCheck) fetchOids(config *InstanceConfig, scalarOids, nextOids []OID, enforceConstraints bool, fetchID string) ([]Bind, error) {
	var errorStr string
	var allBinds []Bind
	var scalarObjs []ObjectType
	for _, oid := range scalarOids {
		scalarObjs = append(scalarObjs, oid.AsObjectType())
	}
	var nextObjs []ObjectType
	for _, oid := range nextOids {
		nextObjs = append(nextObjs, oid.AsObjectType())
	}
	for _, batch := range Batches(scalarObjs, sc.OidBatchSize) {
		sc.Logger.Printf("[%s] Running SNMP GET on OIDs: %s", fetchID, OIDPrinter(batch, false))
		varBinds, err := SnmpGet(config, batch, enforceConstraints)
		if err != nil {
			errorStr = fmt.Sprintf("[%s] Error in GET: %v", fetchID, err)
			sc.Logger.Printf(errorStr)
		}
		allBinds = append(allBinds, varBinds...)
	}
	for _, batch := range Batches(nextObjs, sc.OidBatchSize) {
		sc.Logger.Printf("[%s] Running SNMP GETNEXT on OIDs: %s", fetchID, OIDPrinter(batch, false))
		binds, err := SnmpGetNext(config, batch, enforceConstraints, sc.IgnoreNonincreasingOid)
		if err != nil {
			errorStr = fmt.Sprintf("[%s] Error in GETNEXT: %v", fetchID, err)
			sc.Logger.Printf(errorStr)
		}
		allBinds = append(allBinds, binds...)
	}
	if errorStr != "" {
		return allBinds, fmt.Errorf(errorStr)
	}
	return allBinds, nil
}

func (sc *SnmpCheck) fetchSysobjectOid(config *InstanceConfig) string {
	oid := NewOIDFromParts([]int{1, 3, 6, 1, 2, 1, 1, 2, 0})
	objType := NewObjectTypeFromOID(oid)
	sc.Logger.Printf("Running SNMP GET on OID: %s", OIDPrinter(objType, false))
	varBinds, err := SnmpGet(config, []ObjectType{objType}, false)
	if err != nil || len(varBinds) == 0 {
		sc.Logger.Printf("Failed to fetch sysObjectID: %v", err)
		return ""
	}
	bind := varBinds[0]
	return bind.Value.(SNMPValue).PrettyPrint()
}

func (sc *SnmpCheck) profileForSysobjectOid(sysObjectOid string) (string, error) {
	matched := make(map[string]string)
	for oid, profileName := range sc.ProfilesByOid {
		if fnmatch(oid, sysObjectOid) { // Assume fnmatch is implemented.
			matched[oid] = profileName
		}
	}
	if len(matched) == 0 {
		return "", fmt.Errorf("No profile matching sysObjectID %s", sysObjectOid)
	}
	var bestOid string
	bestScore := -1
	for oid := range matched {
		score := oidPatternSpecificity(oid)
		if score > bestScore {
			bestScore = score
			bestOid = oid
		}
	}
	return matched[bestOid], nil
}

func (sc *SnmpCheck) startDiscovery() {
	cache, err := readPersistentCache(sc.CheckID)
	if err == nil && cache != "" {
		var hosts []string
		if err := json.Unmarshal([]byte(cache), &hosts); err == nil {
			for _, host := range hosts {
				if net.ParseIP(host) == nil {
					writePersistentCache(sc.CheckID, "[]")
					break
				}
				sc.Config.DiscoveredInstances[host] = sc.buildAutodiscoveryConfig(sc.Instance, host)
			}
		}
	}
	rawInterval, ok := sc.Config.Instance["discovery_interval"]
	discoveryInterval := DefaultDiscoveryInterval
	if ok {
		if v, err := strconv.ParseFloat(fmt.Sprintf("%v", rawInterval), 64); err == nil {
			discoveryInterval = v
		} else {
			sc.Logger.Fatalf("discovery_interval could not be parsed: %v", rawInterval)
		}
	}
	go func() {
		discoverInstances(sc.Config, discoveryInterval, sc)
	}()
}

func (sc *SnmpCheck) Check() {
	startTime := time.Now()
	sc.SubmittedMetrics = 0
	config := sc.Config
	var tags []string
	if config.IPNetwork != nil {
		if !sc.DiscoveryStarted {
			sc.startDiscovery()
			sc.DiscoveryStarted = true
		}
		for _, discovered := range config.DiscoveredInstances {
			sc.Executor.Submit(func() (error, []string) {
				return sc.checkDevice(discovered)
			})
		}
		sc.Executor.Wait()
		tags = append([]string{
			fmt.Sprintf("network:%s", config.IPNetwork.String()),
			fmt.Sprintf("autodiscovery_subnet:%s", config.IPNetwork.String()),
		}, config.Tags...)
		sc.Gauge("snmp.discovered_devices_count", float64(len(config.DiscoveredInstances)), tags)
	} else {
		var err error
		var t []string
		err, t = sc.checkDevice(config)
		tags = t
		if err != nil {
			sc.Logger.Printf("Error in device check: %v", err)
		}
	}
	sc.submitTelemetryMetrics(startTime, tags)
}

func (sc *SnmpCheck) submitTelemetryMetrics(startTime time.Time, tags []string) {
	telemetryTags := append(tags, LoaderTag)
	checkDuration := time.Since(startTime).Seconds()
	sc.MonotonicCount("datadog.snmp.check_interval", float64(time.Now().Unix()), telemetryTags)
	sc.Gauge("datadog.snmp.check_duration", checkDuration, telemetryTags)
	sc.Gauge("datadog.snmp.submitted_metrics", float64(sc.SubmittedMetrics), telemetryTags)
}

func (sc *SnmpCheck) checkDevice(config *InstanceConfig) (error, []string) {
	if config.Device == nil {
		return fmt.Errorf("No device set"), nil
	}
	instance := config.Instance
	var err error
	var results map[string]map[string]interface{}
	tags := config.Tags
	if config.OidConfig.ShouldReset() {
		config.OidConfig.Reset()
	}
	if !config.OidConfig.HasOids() {
		sysObjOid := sc.fetchSysobjectOid(config)
		profile, err := sc.profileForSysobjectOid(sysObjOid)
		if err != nil {
			sc.Logger.Printf("Profile error: %v", err)
		} else {
			config.RefreshWithProfile(sc.Profiles[profile])
			config.AddProfileTag(profile)
		}
	}
	if config.OidConfig.HasOids() {
		sc.Logger.Printf("Querying device: %v", config.Device)
		config.AddUptimeMetric()
		results, scalarOids, fetchErr := sc.fetchResults(config)
		if fetchErr != nil {
			err = fetchErr
		}
		config.OidConfig.UpdateScalarOids(scalarOids)
		tags = append(tags, extractMetricTags(config.ParsedMetricTags, results)...)
		tags = append(tags, config.Tags...)
		sc.reportMetrics(config.ParsedMetrics, results, tags)
	}
	sc.Gauge("snmp.devices_monitored", 1, append(tags, LoaderTag))
	status := StatusOK
	if err != nil {
		status = StatusCritical
		if results != nil {
			status = StatusWarning
		}
	}
	sc.ServiceCheck(sc.SC_STATUS(), status, tags, errString(err))
	return err, tags
}

func (sc *SnmpCheck) reportMetrics(metrics []parsed_metrics.ParsedMetric, results map[string]map[string]interface{}, tags []string) {
	for _, metric := range metrics {
		name := metric.GetName()
		resultRows, ok := results[name]
		if !ok {
			sc.Logger.Printf("Ignoring metric %s", name)
			continue
		}
		if isTableMetric(metric) { // Stub: implement isTableMetric
			for index, val := range resultRows {
				metricTags := append(tags, getIndexTags(index, results, nil, nil)...) // Stub for index tags.
				sc.submitMetric(name, val, getForcedType(metric), metricTags, getOptions(metric), nil)
				sc.trySubmitBandwidthUsageMetricIfBandwidthMetric(name, index, results, metricTags)
			}
		} else {
			var firstVal interface{}
			for _, v := range resultRows {
				firstVal = v
				break
			}
			metricTags := append(tags, getMetricTags(metric)...)
			sc.submitMetric(name, firstVal, getForcedType(metric), metricTags, getOptions(metric), nil)
		}
	}
}

func (sc *SnmpCheck) trySubmitBandwidthUsageMetricIfBandwidthMetric(name string, index string, results map[string]map[string]interface{}, tags []string) {
	if !isBandwidthMetric(name) { // Stub: implement isBandwidthMetric
		return
	}
	sc.submitBandwidthUsageMetricIfBandwidthMetric(name, index, results, tags)
}

func (sc *SnmpCheck) submitMetric(name string, snmpValue interface{}, forcedType string, tags []string, options map[string]interface{}, extractValuePattern interface{}) {
	if err := sc.doSubmitMetric(name, snmpValue, forcedType, tags, options, extractValuePattern); err != nil {
		sc.Logger.Printf("Unable to submit metric %s: %v", name, err)
	}
}

func (sc *SnmpCheck) doSubmitMetric(name string, snmpValue interface{}, forcedType string, tags []string, options map[string]interface{}, extractValuePattern interface{}) error {
	if replyInvalid(snmpValue) {
		sc.Logger.Printf("No such MIB available: %s", name)
		return nil
	}
	var metricName string
	if suffix, ok := options["metric_suffix"].(string); ok {
		metricName = normalize(fmt.Sprintf("%s.%s", name, suffix), "snmp")
	} else {
		metricName = normalize(name, "snmp")
	}
	if extractValuePattern != nil {
		snmpValue = extractValue(extractValuePattern, snmpValue.(SNMPValue).PrettyPrint())
	}
	var metric *MetricDefinition
	if forcedType != "" {
		metric = asMetricWithForcedType(snmpValue, forcedType, options)
	} else {
		metric = asMetricWithInferredType(snmpValue)
	}
	if metric == nil {
		return fmt.Errorf("Unsupported metric type for %s", metricName)
	}
	switch metric.Type {
	case "gauge":
		sc.Gauge(metricName, metric.Value, tags)
	case "rate":
		sc.Rate(metricName, metric.Value, tags)
	case "monotonic_count":
		sc.MonotonicCount(metricName, metric.Value, tags)
	case "monotonic_count_and_rate":
		sc.MonotonicCount(metricName, metric.Value, tags)
		sc.Rate(metricName+".rate", metric.Value, tags)
	default:
		return fmt.Errorf("Unknown metric type: %s", metric.Type)
	}
	sc.SubmittedMetrics++
	return nil
}

func (sc *SnmpCheck) SC_STATUS() string {
	return "snmp.can_check"
}

// Stub helper functions:
func isTableMetric(metric parsed_metrics.ParsedMetric) bool {
	return false
}
func getForcedType(metric parsed_metrics.ParsedMetric) string {
	return ""
}
func getOptions(metric parsed_metrics.ParsedMetric) map[string]interface{} {
	return map[string]interface{}{}
}
func getMetricTags(metric parsed_metrics.ParsedMetric) []string {
	return []string{}
}
func getIndexTags(index string, results map[string]map[string]interface{}, indexTags, columnTags any) []string {
	return []string{}
}
func isBandwidthMetric(name string) bool {
	return name == "ifHCInOctets" || name == "ifHCOutOctets"
}
func errString(err error) string {
	if err != nil {
		return err.Error()
	}
	return ""
}
func normalize(name, prefix string) string {
	return prefix + "." + name
}
func replyInvalid(val interface{}) bool {
	return val == nil
}

// SNMPValue is a placeholder.
type SNMPValue interface {
	PrettyPrint() string
}

// Bind represents an OID/value pair.
type Bind struct {
	Oid   string
	Value interface{}
}
