package rpc

import (
	"github.com/decred/dcrd/chaincfg/chainhash"
	"github.com/decred/dcrlnd/lnrpc"
)

// GetChanPointFundingTxid returns the given channel point's funding txid in
// raw bytes.
func GetChanPointFundingTxid(chanPoint *lnrpc.ChannelPoint) (*chainhash.Hash, error) {
	var txid []byte

	// A channel point's funding txid can be get/set as a byte slice or a
	// string. In the case it is a string, decode it.
	switch chanPoint.GetFundingTxid().(type) {
	case *lnrpc.ChannelPoint_FundingTxidBytes:
		txid = chanPoint.GetFundingTxidBytes()
	case *lnrpc.ChannelPoint_FundingTxidStr:
		s := chanPoint.GetFundingTxidStr()
		h, err := chainhash.NewHashFromStr(s)
		if err != nil {
			return nil, err

		}

		txid = h[:]

	}

	return chainhash.NewHash(txid)
}
