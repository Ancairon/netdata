// exceptions.go
// (C) Datadog, Inc. 2020-present (translated)
// All rights reserved
//
// This file provides SNMP integration error types,
// re-creating the exception classes from the original Python code.

package snmp_dev

import "fmt"

// SNMPException is the base error type for SNMP integration errors.
type SNMPException struct {
	Message string
}

// Error implements the error interface.
func (e *SNMPException) Error() string {
	return e.Message
}

// CouldNotDecodeOID is used when a value cannot be decoded as an OID.
type CouldNotDecodeOID struct {
	*SNMPException
}

// NewCouldNotDecodeOID creates a new CouldNotDecodeOID error.
func NewCouldNotDecodeOID(msg string) error {
	return &CouldNotDecodeOID{
		SNMPException: &SNMPException{Message: fmt.Sprintf("CouldNotDecodeOID: %s", msg)},
	}
}

// UnresolvedOID is used when trying to access an OID that is not available.
type UnresolvedOID struct {
	*SNMPException
}

// NewUnresolvedOID creates a new UnresolvedOID error.
func NewUnresolvedOID(msg string) error {
	return &UnresolvedOID{
		SNMPException: &SNMPException{Message: fmt.Sprintf("UnresolvedOID: %s", msg)},
	}
}
