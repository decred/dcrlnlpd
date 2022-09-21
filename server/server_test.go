package server

import (
	"context"
	"testing"
	"time"

	"github.com/decred/dcrd/chaincfg/v3"
	"github.com/decred/dcrd/dcrutil/v4"
	"github.com/decred/dcrlnd/lnrpc"
	"github.com/decred/dcrlnd/lnwire"
)

func TestManageChannels(t *testing.T) {
	dcr := dcrutil.Amount(1e8)
	chainParams := chaincfg.MainNetParams()
	minLifetime := time.Hour
	minWalletBal := 10 * dcr
	baseHeight := 100000
	chanPoint0 := "0000000000000000000000000000000000000000000000000000000000000000:0"
	chanPoint1 := "1111111111111111111111111111111111111111111111111111111111111111:1"
	chanPoint2 := "2222222222222222222222222222222222222222222222222222222222222222:2"
	var nextCidTxIndex uint32
	cidForLifetime := func(d time.Duration) uint64 {
		blocks := uint64(d / chainParams.TargetTimePerBlock)
		bh := uint64(baseHeight) - blocks
		cid := lnwire.ShortChannelID{BlockHeight: uint32(bh), TxIndex: nextCidTxIndex}
		nextCidTxIndex += 1
		return cid.ToUint64()
	}
	tests := []struct {
		name         string
		wbal         dcrutil.Amount
		pendingChans []int64
		chans        []*lnrpc.Channel
		wantClosed   map[string]struct{}
	}{{
		name: "wallet empty and no channels to close",
	}, {
		name: "wallet over min balance limit",
		wbal: minWalletBal,
		chans: []*lnrpc.Channel{{
			LocalBalance: int64(dcr),
		}},
	}, {
		name:         "wallet over min balance including limbo balance",
		wbal:         minWalletBal - 2,
		pendingChans: []int64{1, 3},
		chans: []*lnrpc.Channel{{
			ChannelPoint: chanPoint0,
			RemotePubkey: "remote",
			ChanId:       cidForLifetime(minLifetime + 1),
			Capacity:     int64(dcr),
			LocalBalance: int64(dcr),
			Initiator:    true,
		}},
	}, {
		name: "non-initiator channel is not closed",
		wbal: minWalletBal - 1,
		chans: []*lnrpc.Channel{{
			LocalBalance: int64(dcr),
			Initiator:    false,
		}},
	}, {
		name: "single available channel is before min duration",
		wbal: minWalletBal - 1,
		chans: []*lnrpc.Channel{{
			ChannelPoint: chanPoint0,
			RemotePubkey: "remote",
			ChanId:       cidForLifetime(minLifetime - 1),
			Capacity:     int64(dcr),
			LocalBalance: int64(dcr),
			Initiator:    true,
		}},
	}, {
		name: "single available channel is closed",
		wbal: minWalletBal - 1,
		chans: []*lnrpc.Channel{{
			ChannelPoint: chanPoint0,
			RemotePubkey: "remote",
			ChanId:       cidForLifetime(minLifetime + 1),
			Capacity:     int64(dcr),
			LocalBalance: int64(dcr),
			Initiator:    true,
		}},
		wantClosed: map[string]struct{}{chanPoint0: {}},
	}, {
		name: "lowest activity chan recovers enough balance",
		wbal: minWalletBal - 1,
		chans: []*lnrpc.Channel{{
			ChannelPoint:   chanPoint0,
			RemotePubkey:   "remote0",
			ChanId:         cidForLifetime(minLifetime + 1),
			Capacity:       int64(dcr),
			LocalBalance:   int64(dcr),
			TotalAtomsSent: 1,
			Initiator:      true,
		}, {
			ChannelPoint:   chanPoint1,
			RemotePubkey:   "remote1",
			ChanId:         cidForLifetime(minLifetime + 1),
			Capacity:       int64(dcr),
			LocalBalance:   int64(dcr),
			TotalAtomsSent: int64(dcr),
			Initiator:      true,
		}, {
			ChannelPoint: chanPoint2,
			RemotePubkey: "remote2",
			ChanId:       cidForLifetime(minLifetime + 1),
			Capacity:     int64(dcr),
			LocalBalance: int64(dcr),
			Initiator:    true,
		}},
		wantClosed: map[string]struct{}{chanPoint2: {}},
	}, {
		name: "multiple chans recover enough balance",
		wbal: minWalletBal - 2*dcr,
		chans: []*lnrpc.Channel{{
			ChannelPoint:   chanPoint0,
			RemotePubkey:   "remote0",
			ChanId:         cidForLifetime(minLifetime + 1),
			Capacity:       int64(dcr),
			LocalBalance:   int64(dcr),
			TotalAtomsSent: 1,
			Initiator:      true,
		}, {
			ChannelPoint:   chanPoint1,
			RemotePubkey:   "remote1",
			ChanId:         cidForLifetime(minLifetime + 1),
			Capacity:       int64(dcr),
			LocalBalance:   int64(dcr),
			TotalAtomsSent: int64(dcr),
			Initiator:      true,
		}, {
			ChannelPoint: chanPoint2,
			RemotePubkey: "remote2",
			ChanId:       cidForLifetime(minLifetime + 1),
			Capacity:     int64(dcr),
			LocalBalance: int64(dcr),
			Initiator:    true,
		}},
		wantClosed: map[string]struct{}{
			chanPoint0: {},
			chanPoint2: {},
		},
	}}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			lc := &lightningClientMock{
				wbalance: []*lnrpc.WalletBalanceResponse{{
					TotalBalance: int64(tc.wbal),
				}},
				pendingChans: []*lnrpc.PendingChannelsResponse{{}},
				listChans: []*lnrpc.ListChannelsResponse{{
					Channels: tc.chans,
				}},
				info: []*lnrpc.GetInfoResponse{{
					BlockHeight: uint32(baseHeight),
				}},
				peers: []*lnrpc.ListPeersResponse{{}},
			}
			for _, pcAmount := range tc.pendingChans {
				lc.pendingChans[0].PendingForceClosingChannels = append(lc.pendingChans[0].PendingForceClosingChannels,
					&lnrpc.PendingChannelsResponse_ForceClosedChannel{
						LimboBalance: pcAmount,
					})
			}
			for _, toClose := range tc.wantClosed {
				_ = toClose
				lccc := &lightningCloseChannelClient{
					updates: []*lnrpc.CloseStatusUpdate{{}},
				}
				lc.closeChans = append(lc.closeChans, lccc)
			}

			svr := &Server{
				log:         testLoggerSys(t, "SRVR"),
				lc:          lc,
				chainParams: chainParams,
				cfg: Config{
					ChainParams:      chainParams,
					MinWalletBalance: minWalletBal,
					MinChanLifetime:  minLifetime,
				},
			}

			ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			defer cancel()
			err := svr.manageChannels(ctx)
			if err != nil {
				t.Fatalf("Unexpected error: %v", err)
			}

			// Time for the close() call in the goroutine to be called.
			time.Sleep(100 * time.Millisecond)

			lc.Lock()
			gotClosed := lc.reqCloseChans
			lc.Unlock()

			if len(gotClosed) != len(tc.wantClosed) {
				t.Fatalf("Unexpected number of closed chans: "+
					"got %d, want %d", len(gotClosed), len(tc.wantClosed))
			}

			for _, gotChan := range gotClosed {
				gotChanPoint := mustFmtChannelPoint(gotChan.ChannelPoint)
				if _, ok := tc.wantClosed[gotChanPoint]; !ok {
					t.Fatalf("Unexpected closed chan: %s",
						gotChanPoint)
				}
				delete(tc.wantClosed, gotChanPoint)
			}
		})
	}
}
