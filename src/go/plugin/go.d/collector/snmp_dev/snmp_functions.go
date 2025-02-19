// commands.go
// (C) Datadog, Inc. 2020-present (translated)
// All rights reserved
// Translated to Go by keeping the original logic as much as possible.

package snmp_dev

import (
	"fmt"

	"github.com/gosnmp/gosnmp"
)

// SnmpGet performs an SNMP GET on the provided OIDs.
// The lookupMib flag is currently not implemented (MIB resolution is not supported).
func SnmpGet(cfg *SNMPConfig, oids []string, lookupMib bool) ([]gosnmp.SnmpPDU, error) {
	if cfg.Target == "" {
		return nil, fmt.Errorf("no target device set")
	}

	gs := &gosnmp.GoSNMP{
		Target:    cfg.Target,
		Port:      cfg.Port,
		Community: cfg.Community,
		Version:   cfg.Version,
		Timeout:   cfg.Timeout,
		Retries:   cfg.Retries,
	}
	if err := gs.Connect(); err != nil {
		return nil, fmt.Errorf("error connecting to %s: %v", cfg.Target, err)
	}
	defer gs.Conn.Close()

	result, err := gs.Get(oids) // synchronous GET call
	if err != nil {
		return nil, err
	}

	pdus := result.Variables
	if lookupMib {
		pdus = unmakeVarbinds(pdus, lookupMib)
	}
	return pdus, nil
}

// SnmpGetNext performs SNMP GETNEXT on the given list of OIDs.
// It iterates until the returned OID no longer begins with the original OID prefix.
func SnmpGetNext(cfg *SNMPConfig, oids []string, lookupMib bool, ignoreNonIncreasingOid bool) ([]gosnmp.SnmpPDU, error) {
	if cfg.Target == "" {
		return nil, fmt.Errorf("no target device set")
	}

	gs := &gosnmp.GoSNMP{
		Target:    cfg.Target,
		Port:      cfg.Port,
		Community: cfg.Community,
		Version:   cfg.Version,
		Timeout:   cfg.Timeout,
		Retries:   cfg.Retries,
	}
	if err := gs.Connect(); err != nil {
		return nil, fmt.Errorf("error connecting to %s: %v", cfg.Target, err)
	}
	defer gs.Conn.Close()

	var results []gosnmp.SnmpPDU
	// For each base OID, repeatedly perform GETNEXT until the OID prefix changes.
	for _, baseOid := range oids {
		currentOid := baseOid
		for {
			pkt, err := gs.GetNext([]string{currentOid})
			if err != nil {
				return results, err
			}
			if len(pkt.Variables) == 0 {
				break
			}

			pdu := pkt.Variables[0]
			if lookupMib {
				// MIB resolution is not implemented; unmakeVarbinds is a no-op for now.
				list := unmakeVarbinds([]gosnmp.SnmpPDU{pdu}, lookupMib)
				if len(list) > 0 {
					pdu = list[0]
				}
			}

			// If the returned OID no longer starts with the base OID, stop.
			if !isPrefix(baseOid, pdu.Name) {
				break
			}

			results = append(results, pdu)
			currentOid = pdu.Name
		}
	}
	return results, nil
}

// SnmpBulk performs an SNMP GETBULK on the provided OID.
// It repeatedly calls GETBULK until it reaches the end of the subtree.
func SnmpBulk(cfg *SNMPConfig, oid string, nonRepeaters, maxRepetitions int, lookupMib bool, ignoreNonIncreasingOid bool) ([]gosnmp.SnmpPDU, error) {
	if cfg.Target == "" {
		return nil, fmt.Errorf("no target device set")
	}

	gs := &gosnmp.GoSNMP{
		Target:    cfg.Target,
		Port:      cfg.Port,
		Community: cfg.Community,
		Version:   cfg.Version,
		Timeout:   cfg.Timeout,
		Retries:   cfg.Retries,
	}
	if err := gs.Connect(); err != nil {
		return nil, fmt.Errorf("error connecting to %s: %v", cfg.Target, err)
	}
	defer gs.Conn.Close()

	var results []gosnmp.SnmpPDU
	baseOid := oid
	currentOid := oid

	for {
		pkt, err := gs.GetBulk([]string{currentOid}, nonRepeaters, maxRepetitions)
		if err != nil {
			return results, err
		}
		if len(pkt.Variables) == 0 {
			break
		}

		stop := false
		for _, pdu := range pkt.Variables {
			// Check for end-of-MIB (gosnmp provides EndOfMibView)
			if pdu.Type == gosnmp.EndOfMibView {
				stop = true
				break
			}
			if lookupMib {
				list := unmakeVarbinds([]gosnmp.SnmpPDU{pdu}, lookupMib)
				if len(list) > 0 {
					pdu = list[0]
				}
			}
			// If the returned OID no longer begins with the base, we stop.
			if !isPrefix(baseOid, pdu.Name) {
				stop = true
				break
			}
			results = append(results, pdu)
			currentOid = pdu.Name
		}
		if stop {
			break
		}
	}
	return results, nil
}
