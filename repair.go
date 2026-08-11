package api

import (
	"fmt"
	"strconv"
)

// RepairSessionIDHeader carries the Traffic Ops UserServiceSession ID on a
// unicast repair request.
const RepairSessionIDHeader = "X-Repair-Session-ID"

// RepairSessionID is the canonical positive-decimal Traffic Ops
// UserServiceSession ID used to route a repair request to its sender handler.
type RepairSessionID string

// NewRepairSessionID converts a Traffic Ops session ID to its wire form.
func NewRepairSessionID(sessionID uint) (RepairSessionID, error) {
	if sessionID == 0 {
		return "", fmt.Errorf("repair session ID must be positive")
	}
	return RepairSessionID(strconv.FormatUint(uint64(sessionID), 10)), nil
}

// ParseRepairSessionID validates and canonicalizes a repair-session wire value.
func ParseRepairSessionID(value string) (RepairSessionID, error) {
	sessionID, err := strconv.ParseUint(value, 10, 64)
	if err != nil || sessionID == 0 || strconv.FormatUint(sessionID, 10) != value {
		return "", fmt.Errorf("invalid repair session ID %q", value)
	}
	return RepairSessionID(value), nil
}
