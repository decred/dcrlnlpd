package lprpc_v1

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"time"

	"github.com/decred/dcrlnlpd/rpc"
)

var ErrDecode = errors.New("unable to decode value")

type InvoiceRequest struct {
	TargetNode  rpc.NodeID `json:"target_node"`
	ChannelSize uint64     `json:"channel_size"`
	Key         string     `json:"key"`
}

type InvoiceResponse struct {
	Invoice string `json:"invoice"`
}

// PolicyResponse is the policy followed by a some LP node.
type PolicyResponse struct {
	Node               rpc.NodeID    `json:"node"`
	NodeAddresses      []string      `json:"addresses"`
	MinChanSize        uint64        `json:"min_chan_size"`
	MaxChanSize        uint64        `json:"max_chan_size"`
	MaxNbChannels      uint          `json:"max_nb_channels"`
	MinChanLifetime    time.Duration `json:"min_chan_lifetime"`
	RequiredAtomsSent  uint64        `json:"required_atoms_sent"`
	RequiredInterval   time.Duration `json:"required_interval"`
	ChanInvoiceFeeRate float64       `json:"chan_invoice_fee_rate"`
}

// Encode encodes the specified object in the stream according to the V1
// protocol version.
func Encode(w io.Writer, v interface{}) error {
	enc := json.NewEncoder(w)
	return enc.Encode(v)
}

// Decode decodes from the specified reader into the object, according to the
// V1 protocol version.
func Decode(r io.Reader, v interface{}) error {
	dec := json.NewDecoder(r)
	if err := dec.Decode(v); err != nil {
		return fmt.Errorf("%w: %v", ErrDecode, err)
	}
	return nil
}
