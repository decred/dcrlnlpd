package rpc

import (
	"encoding/hex"
	"encoding/json"
	"fmt"
)

// NodeID is an LN node ID.
type NodeID [33]byte

// String returns the hex encoding of the NodeID.
func (u NodeID) String() string {
	return hex.EncodeToString(u[:])
}

// MarshalJSON marshals the id into a json string.
func (u NodeID) MarshalJSON() ([]byte, error) {
	return json.Marshal(u.String())
}

// UnmarshalJSON unmarshals the json representation of a NodeID.
func (u *NodeID) UnmarshalJSON(b []byte) error {
	var s string
	if err := json.Unmarshal(b, &s); err != nil {
		return err
	}
	return u.FromString(s)
}

// FromString decodes s into an NodeID. s must contain an hex-encoded ID of the
// correct length.
func (u *NodeID) FromString(s string) error {
	h, err := hex.DecodeString(s)
	if err != nil {
		return err
	}
	if len(h) != len(u) {
		return fmt.Errorf("invalid NodeID length: %d", len(h))
	}
	copy(u[:], h)
	return nil
}
