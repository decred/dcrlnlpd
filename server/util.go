package server

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/decred/dcrd/chaincfg/chainhash"
	"github.com/decred/dcrlnd/lnrpc"
)

func parseStrChannelPoint(s string) (lnrpc.ChannelPoint, error) {
	p := strings.Index(s, ":")
	if p < 0 {
		return lnrpc.ChannelPoint{}, fmt.Errorf("channel point does not have ':'")
	}

	txid := s[:p]
	idx, err := strconv.Atoi(s[p+1:])
	if err != nil {
		return lnrpc.ChannelPoint{}, err
	}

	return lnrpc.ChannelPoint{
		FundingTxid: &lnrpc.ChannelPoint_FundingTxidStr{FundingTxidStr: txid},
		OutputIndex: uint32(idx),
	}, nil
}

func fmtChannelPoint(cp *lnrpc.ChannelPoint) (string, error) {
	var txid []byte

	// A channel point's funding txid can be get/set as a byte slice or a
	// string. In the case it is a string, decode it.
	switch cp.GetFundingTxid().(type) {
	case *lnrpc.ChannelPoint_FundingTxidBytes:
		txid = cp.GetFundingTxidBytes()
	case *lnrpc.ChannelPoint_FundingTxidStr:
		s := cp.GetFundingTxidStr()
		h, err := chainhash.NewHashFromStr(s)
		if err != nil {
			return "", err

		}

		txid = h[:]
	}

	ch, err := chainhash.NewHash(txid)
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("%s:%d", ch, cp.GetOutputIndex()), nil
}

func mustFmtChannelPoint(cp *lnrpc.ChannelPoint) string {
	s, err := fmtChannelPoint(cp)
	if err != nil {
		panic(err)
	}
	return s
}
