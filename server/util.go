package server

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/decred/dcrlnd/lnrpc"
)

// channelHadActivity returns true if the parameters means a channel has had
// enough activity to justify remaining open.
func channelHadActivity(requiredSentAtoms, totalSentAtoms int64,
	requiredInterval, lifetime time.Duration) bool {

	if lifetime < requiredInterval {
		panic("lifetime cannot be smaller than required interval")
	}
	if lifetime == 0 {
		panic("lifetime cannot be zero")
	}
	if requiredSentAtoms == 0 {
		panic("required sent atoms cannot be zero")
	}

	// The basic equation for determining whether the channel had enough
	// activity is:
	//
	//   (totalSentAtoms / lifetime) >= (requiredSentAtoms / requiredInterval)
	//
	// In other words, the rate of atoms sent must be higher than the
	// required rate.
	//
	// By re-arranging the equation so that this can be done entirely using
	// integer numbers, we have:
	//
	//  (totalSentAtoms / requiredSentAtoms) / (lifetime / requiredTime) >= 1
	return (totalSentAtoms/requiredSentAtoms)/int64(lifetime/requiredInterval) >= 1
}

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
