// pysnmp_inspect.go
// (C) Datadog, Inc. 2020-present (translated)
// All rights reserved
//
// Helpers for inspecting SNMP objects.
// Note: Depending on your SNMP library design, you might not need these functions.

package snmp_dev

// // ObjectIdentityFromObjectType extracts an ObjectIdentity from an ObjectType.
// // It assumes that the ObjectType interface has a method GetObjectIdentity().
// func ObjectIdentityFromObjectType(objectType ObjectType) ObjectIdentity {
// 	return objectType.GetObjectIdentity()
// }

// // SNMP_COUNTER_CLASSES defines the set of class names that represent SNMP counters.
// var SNMP_COUNTER_CLASSES = map[string]struct{}{
// 	"Counter32":          {},
// 	"Counter64":          {},
// 	"ZeroBasedCounter64": {},
// }

// // SNMP_GAUGE_CLASSES defines the set of class names that represent SNMP gauges.
// var SNMP_GAUGE_CLASSES = map[string]struct{}{
// 	"Gauge32":             {},
// 	"Integer":             {},
// 	"Integer32":           {},
// 	"Unsigned32":          {},
// 	"CounterBasedGauge64": {},
// }

// // IsCounter returns true if the provided object's type name is in the set of counter classes.
// func IsCounter(obj interface{}) bool {
// 	typeName := reflect.TypeOf(obj).Name()
// 	_, exists := SNMP_COUNTER_CLASSES[typeName]
// 	return exists
// }

// // IsGauge returns true if the provided object's type name is in the set of gauge classes.
// func IsGauge(obj interface{}) bool {
// 	typeName := reflect.TypeOf(obj).Name()
// 	_, exists := SNMP_GAUGE_CLASSES[typeName]
// 	return exists
// }

// // IsOpaque returns true if the provided object's type name is "Opaque".
// func IsOpaque(obj interface{}) bool {
// 	return reflect.TypeOf(obj).Name() == "Opaque"
// }
