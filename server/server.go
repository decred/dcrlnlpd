package server

import (
	"context"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"path/filepath"
	"sync"
	"time"

	"github.com/decred/dcrd/chaincfg/v3"
	"github.com/decred/dcrd/dcrutil/v4"
	"github.com/decred/dcrd/wire"
	"github.com/decred/dcrlnd/lnrpc"
	"github.com/decred/dcrlnd/lnwire"
	"github.com/decred/dcrlnlpd/rpc"
	"github.com/decred/slog"
	"golang.org/x/sync/errgroup"
	"google.golang.org/grpc"
)

var (
	errPolicy              = errors.New("channel creation policy error")
	errTooSmallChan        = fmt.Errorf("%w: channel size is too small", errPolicy)
	errTooLargeChan        = fmt.Errorf("%w: channel size is too large", errPolicy)
	errTooManyOpenChans    = fmt.Errorf("%w: too many open chans to target node", errPolicy)
	errHasPendingOpenChan  = fmt.Errorf("%w: already has pending open channel request", errPolicy)
	errPeerUnconnected     = fmt.Errorf("%w: peer not connected to LP node", errPolicy)
	errNotEnoughUtxos      = fmt.Errorf("%w: not enough utxos to create channel", errPolicy)
	errTooManyPendingChans = fmt.Errorf("%w: node already has too many pending channels", errPolicy)

	// invoiceExpiration tracks how long the invoices generated to pay for
	// channels are valid for.
	invoiceExpiration = time.Hour
)

// Config holds the server config.
type Config struct {
	ChainParams *chaincfg.Params
	RootDir     string

	// Dcrlnd Connection Options
	LNRPCHost      string
	LNTLSCertPath  string
	LNMacaroonPath string
	LNNodeAddrs    []string
	Log            slog.Logger

	// Max number of channels the underlying dcrlnd instance can have as
	// pending.
	MaxPendingChans uint

	// Policy config
	MinChanSize        uint64
	MaxChanSize        uint64
	MaxNbChannels      uint
	CloseCheckInterval time.Duration
	MinChanLifetime    time.Duration
	RequiredAtomsSent  uint64
	RequiredInterval   time.Duration
	CreateKey          []byte

	// ChanInvoiceFeeRate is the fee rate to charge for creating a channel.
	// in atoms/channel-size-atoms.
	ChanInvoiceFeeRate float64
}

// Server is a server instance.
type Server struct {
	cfg         Config
	chainParams *chaincfg.Params
	conn        *grpc.ClientConn
	nodeID      rpc.NodeID
	log         slog.Logger
	lc          lnrpc.LightningClient
	root        string

	// pendingChans tracks nodes for which there's already a pending request
	// to create a channel to.
	pendingChansMtx sync.Mutex
	pendingChans    map[rpc.NodeID]time.Time
}

// New creates a new server instance.
func New(cfg *Config) (*Server, error) {
	log := cfg.Log
	if log == nil {
		log = svrLog
	}

	log.Debugf("Connecting to dcrlnd server %s", cfg.LNRPCHost)
	conn, err := connectToDcrlnd(cfg.LNRPCHost, cfg.LNTLSCertPath, cfg.LNMacaroonPath)
	if err != nil {
		return nil, fmt.Errorf("unable to connect to dcrlnd: %v", err)
	}

	lnRpc := lnrpc.NewLightningClient(conn)

	// Check chain and network (mainnet, testnet, etc).
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*10)
	defer cancel()
	lnInfo, err := lnRpc.GetInfo(ctx, &lnrpc.GetInfoRequest{})
	if err != nil {
		return nil, fmt.Errorf("unable to get dcrlnd node info: %v", err)
	}

	if len(lnInfo.Chains) != 1 {
		return nil, fmt.Errorf("unexpected number of chains in ln node (%d)",
			len(lnInfo.Chains))
	}
	chain := lnInfo.Chains[0]
	if chain.Chain != "decred" {
		return nil, fmt.Errorf("unexpected chain in ln node: %q", chain.Chain)
	}
	gotNetwork := chain.Network
	if gotNetwork == "testnet" { // Special case testnet due to resets.
		gotNetwork = "testnet3"
	}
	if gotNetwork != cfg.ChainParams.Name {
		return nil, fmt.Errorf("unexpected network in ln node: %q vs %q",
			gotNetwork, cfg.ChainParams.Name)
	}

	var nodeID rpc.NodeID
	if err := nodeID.FromString(lnInfo.IdentityPubkey); err != nil {
		return nil, err
	}
	log.Infof("Connected to %s LN node %s synced to height %d", chain.Network,
		lnInfo.IdentityPubkey, lnInfo.BlockHeight)

	s := &Server{
		cfg:          *cfg,
		nodeID:       nodeID,
		chainParams:  cfg.ChainParams,
		conn:         conn,
		log:          log,
		lc:           lnRpc,
		root:         cfg.RootDir,
		pendingChans: make(map[rpc.NodeID]time.Time),
	}

	return s, nil
}

// amountForNewChan returns the amount that needs to be paid by a remote host
// to create a channel of the specified size.
func (s *Server) amountForNewChan(chanSize uint64) uint64 {
	return uint64(float64(chanSize) * s.cfg.ChanInvoiceFeeRate)
}

func (s *Server) canCreateChannel(ctx context.Context, wv waitingInvoice) error {
	// Channel size policies.
	if wv.ChannelSize < s.cfg.MinChanSize {
		return errTooSmallChan
	}
	if wv.ChannelSize > s.cfg.MaxChanSize {
		return errTooLargeChan
	}

	// Verify if there are already opened channels to the target node. Note
	// that this does NOT differentiate between LP-initiated vs remote node
	// initiated channels.
	chansReq := &lnrpc.ListChannelsRequest{Peer: wv.TargetNode[:]}
	chansRes, err := s.lc.ListChannels(ctx, chansReq)
	if err != nil {
		return err
	}
	if uint(len(chansRes.Channels)) >= s.cfg.MaxNbChannels {
		return errTooManyOpenChans
	}

	return nil
}

// hasPendingCreateChanReqs returns true if there are pending requests to create
// a chan to the specified node. If there aren't, then this adds a request
// timed to the current time.
func (s *Server) hasPendingCreateChanReqs(node rpc.NodeID) bool {
	s.pendingChansMtx.Lock()
	t, ok := s.pendingChans[node]
	if ok && !t.Before(time.Now().Add(-invoiceExpiration)) {
		s.pendingChansMtx.Unlock()
		return true
	}
	s.pendingChans[node] = time.Now()
	s.pendingChansMtx.Unlock()
	return false
}

// CreateInvoice creates an invoice that, once paid, will trigger the underlying
// node to open a channel back to the specified client.
func (s *Server) CreateInvoice(ctx context.Context, node rpc.NodeID, chanSize uint64) (string, error) {
	invRecord := waitingInvoice{
		TargetNode:  node,
		ChannelSize: chanSize,
	}

	// Enforce policy.
	err := s.canCreateChannel(ctx, invRecord)
	if err == nil && !s.isConnectedToNode(ctx, node) {
		err = errPeerUnconnected
	}
	if err == nil && !s.hasUnspentForNewChannel(ctx, chanSize) {
		err = errNotEnoughUtxos
	}
	if err == nil && s.hasMaxPendingChannels(ctx) {
		err = errTooManyPendingChans
	}
	if err == nil && s.hasPendingCreateChanReqs(node) {
		err = errHasPendingOpenChan
	}
	if err != nil {
		s.log.Warnf("Rejected creation request for node %s size %.8f: %v",
			node, dcrutil.Amount(chanSize).ToCoin(), err)
		return "", err
	}

	// Create the invoice.
	amount := s.amountForNewChan(chanSize)
	req := &lnrpc.Invoice{
		Memo:   fmt.Sprintf("LP of %d for %s", chanSize, node),
		Value:  int64(amount),
		Expiry: int64(invoiceExpiration.Seconds()),
	}
	res, err := s.lc.AddInvoice(ctx, req)
	if err != nil {
		return "", err
	}

	// Save the data for this invoice.
	fpath := filepath.Join(s.root, invoicesDir, hex.EncodeToString(res.RHash))
	if err := s.writeJsonFile(fpath, invRecord); err != nil {
		return "", err
	}

	s.log.Infof("Created invoice %x of %s for chan to node %s size %.8f",
		res.RHash, dcrutil.Amount(amount), node, dcrutil.Amount(chanSize).ToCoin())

	return res.PaymentRequest, nil
}

func (s *Server) openChannel(ctx context.Context, winv waitingInvoice) {
	// Sanity check the channel still follows the policy.
	if err := s.canCreateChannel(ctx, winv); err != nil {
		s.log.Errorf("Unable to send open channel request to "+
			"%s for size %.8f: %v", winv.TargetNode,
			dcrutil.Amount(winv.ChannelSize).ToCoin(), err)
		return
	}

	// Track attempts at opening the channel.
	const maxAttempts = 3
	var attempt int

	req := &lnrpc.OpenChannelRequest{
		NodePubkey:         winv.TargetNode[:],
		LocalFundingAmount: int64(winv.ChannelSize),
	}

	// We'll make 3 attempts at opening the channel.
	var ch lnrpc.Lightning_OpenChannelClient
	for attempt = 0; attempt < maxAttempts; attempt++ {
		s.log.Debugf("Attempt %d to open channel to %s of size %.8f",
			attempt+1, winv.TargetNode, dcrutil.Amount(winv.ChannelSize).ToCoin())

		// Make an attempt to open the channel.
		err := func() error {
			if !s.isConnectedToNode(ctx, winv.TargetNode) {
				return errPeerUnconnected
			}
			if !s.hasUnspentForNewChannel(ctx, winv.ChannelSize) {
				return errNotEnoughUtxos
			}

			var err error
			ch, err = s.lc.OpenChannel(ctx, req)
			return err
		}()

		if err == nil {
			// Attempt succeeded!
			break
		}

		// Attempt failed. Delay next attempt or quit.
		if attempt == maxAttempts-1 {
			s.log.Errorf("Unable to open channel to %s: %v."+
				"Giving up.", winv.TargetNode, err)
			return
		}

		delayAmount := 30 * time.Second * time.Duration(attempt+1)
		s.log.Warnf("Unable to open channel to %s: %v. Delaying "+
			"next attempt by %s", winv.TargetNode, err, delayAmount)
		select {
		case <-time.After(delayAmount):
		case <-ctx.Done():
			s.log.Warnf("Early return from opening "+
				"channel to %s of size %.8f",
				winv.TargetNode,
				dcrutil.Amount(winv.ChannelSize).ToCoin())
			return
		}
	}

	// Report on channel events.
	s.log.Infof("Requested open channel to %s of size %.8f",
		winv.TargetNode, dcrutil.Amount(winv.ChannelSize).ToCoin())
	for {
		update, err := ch.Recv()
		if err != nil {
			if errors.Is(err, context.Canceled) || errors.Is(err, io.EOF) {
				// End of stream.
				return
			}
			s.log.Errorf("Error fetching open channel update: %v", err)
			return
		}

		if updt, ok := update.Update.(*lnrpc.OpenStatusUpdate_ChanPending); ok {
			outp := wire.OutPoint{Index: updt.ChanPending.OutputIndex}
			copy(outp.Hash[:], updt.ChanPending.Txid)
			s.log.Infof("Channel to %s of size %.8f pending open: Channel ID %s",
				winv.TargetNode, dcrutil.Amount(winv.ChannelSize).ToCoin(),
				outp)
			continue
		}

		updt, ok := update.Update.(*lnrpc.OpenStatusUpdate_ChanOpen)
		if !ok {
			continue
		}
		txid, err := rpc.GetChanPointFundingTxid(updt.ChanOpen.ChannelPoint)
		if err != nil {
			s.log.Errorf("Unable to decode fundingTxid: %v", err)
			return
		}

		s.pendingChansMtx.Lock()
		delete(s.pendingChans, winv.TargetNode)
		s.pendingChansMtx.Unlock()

		outp := &wire.OutPoint{
			Hash:  *txid,
			Index: updt.ChanOpen.ChannelPoint.OutputIndex,
		}
		s.log.Infof("Channel to %s of size %.8f opened: Channel ID %s",
			winv.TargetNode, dcrutil.Amount(winv.ChannelSize).ToCoin(),
			outp)
	}
}

// listenToInvoices reacts to invoice events.
func (s *Server) listenToInvoices(ctx context.Context) error {
	stream, err := s.lc.SubscribeInvoices(ctx, &lnrpc.InvoiceSubscription{})
	if err != nil {
		return err
	}

	for {
		inv, err := stream.Recv()
		if err != nil {
			return err
		}

		switch {
		case inv.State == lnrpc.Invoice_CANCELED:
			fpath := filepath.Join(s.root, invoicesDir,
				hex.EncodeToString(inv.RHash))
			if err := s.removeFile(fpath); err != nil {
				return err
			}

		case inv.State == lnrpc.Invoice_SETTLED:
			fpath := filepath.Join(s.root, invoicesDir,
				hex.EncodeToString(inv.RHash))
			var winv waitingInvoice
			err := s.readJsonFile(fpath, &winv)
			if errors.Is(err, errNotExists) {
				// Payment for something that isn't channel
				// creation.
				continue
			}
			if err != nil {
				return err
			}

			wantAtoms := int64(s.amountForNewChan(winv.ChannelSize))
			if inv.AmtPaidAtoms < wantAtoms {
				s.log.Warnf("Received payment for invoice %x "+
					"lower than required (%d < %d)",
					inv.AmtPaidAtoms < wantAtoms)
				continue
			}

			// Create channel.
			if err := s.removeFile(fpath); err != nil {
				return err
			}
			go s.openChannel(ctx, winv)
		}
	}
}

// closeChannel closes the given channel.
func (s *Server) closeChannel(ctx context.Context, nodeID rpc.NodeID, channelPoint string) {
	cp, err := parseStrChannelPoint(channelPoint)
	if err != nil {
		s.log.Errorf("Unable to parse channel point %s: %v",
			channelPoint, err)
		return
	}

	force := !s.isConnectedToNode(ctx, nodeID)
	req := &lnrpc.CloseChannelRequest{
		ChannelPoint: &cp,
		Force:        force,
	}

	res, err := s.lc.CloseChannel(ctx, req)
	if err != nil {
		s.log.Errorf("Unable to close channel %s: %v", channelPoint, err)
		return
	}

	for {
		updt, err := res.Recv()
		if err != nil {
			if !errors.Is(err, io.EOF) && !errors.Is(err, context.Canceled) {
				s.log.Errorf("Error waiting for close channel update: %v", err)
			}
			return
		}

		switch updt := updt.Update.(type) {
		case *lnrpc.CloseStatusUpdate_ClosePending:
			s.log.Infof("Pending channel close %s with tx %x",
				channelPoint, updt.ClosePending.Txid)
		case *lnrpc.CloseStatusUpdate_ChanClose:
			s.log.Infof("Channel %s closed with tx %x",
				channelPoint, updt.ChanClose.ClosingTxid)

		default:
			s.log.Warnf("Unknown channel close update type: %T", updt)
		}
	}
}

// manageChannels manages opened channels where the local node is the initiator.
func (s *Server) manageChannels(ctx context.Context) error {
	for {
		select {
		case <-time.After(s.cfg.CloseCheckInterval):
		case <-ctx.Done():
			return ctx.Err()
		}

		// Time to check the channels. Fetch list of channels.
		s.log.Debugf("Managing channels")
		chans, err := s.lc.ListChannels(ctx, &lnrpc.ListChannelsRequest{})
		if err != nil {
			return err
		}

		// Fetch the current block time to figure out channel lifetime.
		info, err := s.lc.GetInfo(ctx, &lnrpc.GetInfoRequest{})
		if err != nil {
			return err
		}
		bh := info.BlockHeight

		var nonInit, beforeMinDur, okActivity, closing int
		for _, c := range chans.Channels {
			// Ignore channels where we are not the initiator.
			if !c.Initiator {
				nonInit += 1
				continue
			}

			cid := lnwire.NewShortChanIDFromInt(c.ChanId)
			lifetime := time.Duration(bh-cid.BlockHeight) * s.chainParams.TargetTimePerBlock
			if lifetime < s.cfg.MinChanLifetime || lifetime < s.cfg.RequiredInterval {
				beforeMinDur += 1
				continue
			}

			hadActivity := channelHadActivity(int64(s.cfg.RequiredAtomsSent),
				c.TotalAtomsSent, s.cfg.RequiredInterval, lifetime)
			if !hadActivity {
				// Time to close channel!
				s.log.Infof("Closing channel %s with %s due to "+
					"low activity (sent %.8f, lifetime %s)",
					c.ChannelPoint, c.RemotePubkey,
					dcrutil.Amount(c.TotalAtomsSent).ToCoin(),
					lifetime)
				closing += 1
				var nodeID rpc.NodeID
				nodeID.FromString(c.RemotePubkey)
				go s.closeChannel(ctx, nodeID, c.ChannelPoint)
			} else {
				okActivity += 1
			}
		}

		s.log.Infof("Managed channels. Non-initiator: %d, before min duration: %d "+
			"ok activity: %d, closing: %d", nonInit, beforeMinDur,
			okActivity, closing)

	}
}

// Run starts all service goroutines and blocks until the passed context is
// canceled.
func (s *Server) Run(ctx context.Context) error {
	g, gctx := errgroup.WithContext(ctx)

	// React to paid invoices.
	g.Go(func() error { return s.listenToInvoices(gctx) })

	// Cleanup stale entries in pendingChans.
	g.Go(func() error {
		for {
			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-time.After(invoiceExpiration):
			}

			s.pendingChansMtx.Lock()
			limit := time.Now().Add(-invoiceExpiration)
			count := 0
			for k, v := range s.pendingChans {
				if v.Before(limit) {
					delete(s.pendingChans, k)
					count += 1
				}
			}
			s.pendingChansMtx.Unlock()
			if count > 0 {
				s.log.Infof("Cleaned up %d stale pending channels", count)
			}

		}
	})

	// Close low activity channels.
	g.Go(func() error { return s.manageChannels(ctx) })

	// Shutdown conn once an error occurrs. This unblocks any outstanding
	// calls.
	g.Go(func() error {
		<-gctx.Done()
		if err := s.conn.Close(); err != nil {
			s.log.Warnf("Error while closing conn: %v", err)
		}
		return gctx.Err()
	})

	return g.Wait()
}
