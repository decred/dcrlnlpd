package server

import (
	"context"
	"errors"
	"io"
	"sync"

	"github.com/decred/dcrlnd/lnrpc"
	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
)

var errNoMoreResponses = errors.New("no more responses")

func nextResponse[T any](s *[]T) (T, error) {
	var v T
	if len(*s) == 0 {
		return v, errNoMoreResponses
	}

	v = (*s)[0]
	if len(*s) > 1 {
		*s = (*s)[1:]
	} else {
		*s = (*s)[:0]
	}
	return v, nil
}

type lightningCloseChannelClient struct {
	sync.Mutex
	clientStream
	updates []*lnrpc.CloseStatusUpdate
}

func (lccc *lightningCloseChannelClient) Recv() (*lnrpc.CloseStatusUpdate, error) {
	lccc.Lock()
	defer lccc.Unlock()
	v, err := nextResponse(&lccc.updates)
	if err != nil {
		return nil, io.EOF
	}
	return v, err
}

type lightningClientMock struct {
	sync.Mutex
	pendingChans  []*lnrpc.PendingChannelsResponse
	listChans     []*lnrpc.ListChannelsResponse
	wbalance      []*lnrpc.WalletBalanceResponse
	info          []*lnrpc.GetInfoResponse
	reqCloseChans []*lnrpc.CloseChannelRequest
	closeChans    []*lightningCloseChannelClient
	peers         []*lnrpc.ListPeersResponse
}

// lncli: `walletbalance`
// WalletBalance returns total unspent outputs(confirmed and unconfirmed), all
// confirmed unspent outputs and all unconfirmed unspent outputs under control
// of the wallet.
func (lc *lightningClientMock) WalletBalance(ctx context.Context, in *lnrpc.WalletBalanceRequest, opts ...grpc.CallOption) (*lnrpc.WalletBalanceResponse, error) {
	lc.Lock()
	defer lc.Unlock()
	return nextResponse(&lc.wbalance)
}

// lncli: `channelbalance`
// ChannelBalance returns the total funds available across all open channels
// in atoms.
func (lc *lightningClientMock) ChannelBalance(ctx context.Context, in *lnrpc.ChannelBalanceRequest, opts ...grpc.CallOption) (*lnrpc.ChannelBalanceResponse, error) {
	panic("not implemented") // TODO: Implement
}

// lncli: `listchaintxns`
// GetTransactions returns a list describing all the known transactions
// relevant to the wallet.
func (lc *lightningClientMock) GetTransactions(ctx context.Context, in *lnrpc.GetTransactionsRequest, opts ...grpc.CallOption) (*lnrpc.TransactionDetails, error) {
	panic("not implemented") // TODO: Implement
}

// lncli: `estimatefee`
// EstimateFee asks the chain backend to estimate the fee rate and total fees
// for a transaction that pays to multiple specified outputs.
//
// When using REST, the `AddrToAmount` map type can be set by appending
// `&AddrToAmount[<address>]=<amount_to_send>` to the URL. Unfortunately this
// map type doesn't appear in the REST API documentation because of a bug in
// the grpc-gateway library.
func (lc *lightningClientMock) EstimateFee(ctx context.Context, in *lnrpc.EstimateFeeRequest, opts ...grpc.CallOption) (*lnrpc.EstimateFeeResponse, error) {
	panic("not implemented") // TODO: Implement
}

// lncli: `sendcoins`
// SendCoins executes a request to send coins to a particular address. Unlike
// SendMany, this RPC call only allows creating a single output at a time. If
// neither target_conf, or atoms_per_byte are set, then the internal wallet
// will consult its fee model to determine a fee for the default confirmation
// target.
func (lc *lightningClientMock) SendCoins(ctx context.Context, in *lnrpc.SendCoinsRequest, opts ...grpc.CallOption) (*lnrpc.SendCoinsResponse, error) {
	panic("not implemented") // TODO: Implement
}

// lncli: `listunspent`
// Deprecated, use walletrpc.ListUnspent instead.
//
// ListUnspent returns a list of all utxos spendable by the wallet with a
// number of confirmations between the specified minimum and maximum.
func (lc *lightningClientMock) ListUnspent(ctx context.Context, in *lnrpc.ListUnspentRequest, opts ...grpc.CallOption) (*lnrpc.ListUnspentResponse, error) {
	panic("not implemented") // TODO: Implement
}

// SubscribeTransactions creates a uni-directional stream from the server to
// the client in which any newly discovered transactions relevant to the
// wallet are sent over.
func (lc *lightningClientMock) SubscribeTransactions(ctx context.Context, in *lnrpc.GetTransactionsRequest, opts ...grpc.CallOption) (lnrpc.Lightning_SubscribeTransactionsClient, error) {
	panic("not implemented") // TODO: Implement
}

// lncli: `sendmany`
// SendMany handles a request for a transaction that creates multiple specified
// outputs in parallel. If neither target_conf, or atoms_per_byte are set, then
// the internal wallet will consult its fee model to determine a fee for the
// default confirmation target.
func (lc *lightningClientMock) SendMany(ctx context.Context, in *lnrpc.SendManyRequest, opts ...grpc.CallOption) (*lnrpc.SendManyResponse, error) {
	panic("not implemented") // TODO: Implement
}

// lncli: `newaddress`
// NewAddress creates a new address under control of the local wallet.
func (lc *lightningClientMock) NewAddress(ctx context.Context, in *lnrpc.NewAddressRequest, opts ...grpc.CallOption) (*lnrpc.NewAddressResponse, error) {
	panic("not implemented") // TODO: Implement
}

// lncli: `signmessage`
// SignMessage signs a message with this node's private key. The returned
// signature string is `zbase32` encoded and pubkey recoverable, meaning that
// only the message digest and signature are needed for verification.
func (lc *lightningClientMock) SignMessage(ctx context.Context, in *lnrpc.SignMessageRequest, opts ...grpc.CallOption) (*lnrpc.SignMessageResponse, error) {
	panic("not implemented") // TODO: Implement
}

// lncli: `verifymessage`
// VerifyMessage verifies a signature over a msg. The signature must be
// zbase32 encoded and signed by an active node in the resident node's
// channel database. In addition to returning the validity of the signature,
// VerifyMessage also returns the recovered pubkey from the signature.
func (lc *lightningClientMock) VerifyMessage(ctx context.Context, in *lnrpc.VerifyMessageRequest, opts ...grpc.CallOption) (*lnrpc.VerifyMessageResponse, error) {
	panic("not implemented") // TODO: Implement
}

// lncli: `connect`
// ConnectPeer attempts to establish a connection to a remote peer. This is at
// the networking level, and is used for communication between nodes. This is
// distinct from establishing a channel with a peer.
func (lc *lightningClientMock) ConnectPeer(ctx context.Context, in *lnrpc.ConnectPeerRequest, opts ...grpc.CallOption) (*lnrpc.ConnectPeerResponse, error) {
	panic("not implemented") // TODO: Implement
}

// lncli: `disconnect`
// DisconnectPeer attempts to disconnect one peer from another identified by a
// given pubKey. In the case that we currently have a pending or active channel
// with the target peer, then this action will be not be allowed.
func (lc *lightningClientMock) DisconnectPeer(ctx context.Context, in *lnrpc.DisconnectPeerRequest, opts ...grpc.CallOption) (*lnrpc.DisconnectPeerResponse, error) {
	panic("not implemented") // TODO: Implement
}

// lncli: `listpeers`
// ListPeers returns a verbose listing of all currently active peers.
func (lc *lightningClientMock) ListPeers(ctx context.Context, in *lnrpc.ListPeersRequest, opts ...grpc.CallOption) (*lnrpc.ListPeersResponse, error) {
	lc.Lock()
	defer lc.Unlock()
	return nextResponse(&lc.peers)
}

// SubscribePeerEvents creates a uni-directional stream from the server to
// the client in which any events relevant to the state of peers are sent
// over. Events include peers going online and offline.
func (lc *lightningClientMock) SubscribePeerEvents(ctx context.Context, in *lnrpc.PeerEventSubscription, opts ...grpc.CallOption) (lnrpc.Lightning_SubscribePeerEventsClient, error) {
	panic("not implemented") // TODO: Implement
}

// lncli: `getinfo`
// GetInfo returns general information concerning the lightning node including
// it's identity pubkey, alias, the chains it is connected to, and information
// concerning the number of open+pending channels.
func (lc *lightningClientMock) GetInfo(ctx context.Context, in *lnrpc.GetInfoRequest, opts ...grpc.CallOption) (*lnrpc.GetInfoResponse, error) {
	lc.Lock()
	defer lc.Unlock()
	return nextResponse(&lc.info)
}

// * lncli: `getrecoveryinfo`
// GetRecoveryInfo returns information concerning the recovery mode including
// whether it's in a recovery mode, whether the recovery is finished, and the
// progress made so far.
func (lc *lightningClientMock) GetRecoveryInfo(ctx context.Context, in *lnrpc.GetRecoveryInfoRequest, opts ...grpc.CallOption) (*lnrpc.GetRecoveryInfoResponse, error) {
	panic("not implemented") // TODO: Implement
}

// lncli: `pendingchannels`
// PendingChannels returns a list of all the channels that are currently
// considered "pending". A channel is pending if it has finished the funding
// workflow and is waiting for confirmations for the funding txn, or is in the
// process of closure, either initiated cooperatively or non-cooperatively.
func (lc *lightningClientMock) PendingChannels(ctx context.Context, in *lnrpc.PendingChannelsRequest, opts ...grpc.CallOption) (*lnrpc.PendingChannelsResponse, error) {
	lc.Lock()
	defer lc.Unlock()
	return nextResponse(&lc.pendingChans)
}

// lncli: `listchannels`
// ListChannels returns a description of all the open channels that this node
// is a participant in.
func (lc *lightningClientMock) ListChannels(ctx context.Context, in *lnrpc.ListChannelsRequest, opts ...grpc.CallOption) (*lnrpc.ListChannelsResponse, error) {
	lc.Lock()
	defer lc.Unlock()
	return nextResponse(&lc.listChans)
}

// SubscribeChannelEvents creates a uni-directional stream from the server to
// the client in which any updates relevant to the state of the channels are
// sent over. Events include new active channels, inactive channels, and closed
// channels.
func (lc *lightningClientMock) SubscribeChannelEvents(ctx context.Context, in *lnrpc.ChannelEventSubscription, opts ...grpc.CallOption) (lnrpc.Lightning_SubscribeChannelEventsClient, error) {
	panic("not implemented") // TODO: Implement
}

// lncli: `closedchannels`
// ClosedChannels returns a description of all the closed channels that
// this node was a participant in.
func (lc *lightningClientMock) ClosedChannels(ctx context.Context, in *lnrpc.ClosedChannelsRequest, opts ...grpc.CallOption) (*lnrpc.ClosedChannelsResponse, error) {
	panic("not implemented") // TODO: Implement
}

// OpenChannelSync is a synchronous version of the OpenChannel RPC call. This
// call is meant to be consumed by clients to the REST proxy. As with all
// other sync calls, all byte slices are intended to be populated as hex
// encoded strings.
func (lc *lightningClientMock) OpenChannelSync(ctx context.Context, in *lnrpc.OpenChannelRequest, opts ...grpc.CallOption) (*lnrpc.ChannelPoint, error) {
	panic("not implemented") // TODO: Implement
}

// lncli: `openchannel`
// OpenChannel attempts to open a singly funded channel specified in the
// request to a remote peer. Users are able to specify a target number of
// blocks that the funding transaction should be confirmed in, or a manual fee
// rate to us for the funding transaction. If neither are specified, then a
// lax block confirmation target is used. Each OpenStatusUpdate will return
// the pending channel ID of the in-progress channel. Depending on the
// arguments specified in the OpenChannelRequest, this pending channel ID can
// then be used to manually progress the channel funding flow.
func (lc *lightningClientMock) OpenChannel(ctx context.Context, in *lnrpc.OpenChannelRequest, opts ...grpc.CallOption) (lnrpc.Lightning_OpenChannelClient, error) {
	panic("not implemented") // TODO: Implement
}

// FundingStateStep is an advanced funding related call that allows the caller
// to either execute some preparatory steps for a funding workflow, or
// manually progress a funding workflow. The primary way a funding flow is
// identified is via its pending channel ID. As an example, this method can be
// used to specify that we're expecting a funding flow for a particular
// pending channel ID, for which we need to use specific parameters.
// Alternatively, this can be used to interactively drive PSBT signing for
// funding for partially complete funding transactions.
func (lc *lightningClientMock) FundingStateStep(ctx context.Context, in *lnrpc.FundingTransitionMsg, opts ...grpc.CallOption) (*lnrpc.FundingStateStepResp, error) {
	panic("not implemented") // TODO: Implement
}

// ChannelAcceptor dispatches a bi-directional streaming RPC in which
// OpenChannel requests are sent to the client and the client responds with
// a boolean that tells LND whether or not to accept the channel. This allows
// node operators to specify their own criteria for accepting inbound channels
// through a single persistent connection.
func (lc *lightningClientMock) ChannelAcceptor(ctx context.Context, opts ...grpc.CallOption) (lnrpc.Lightning_ChannelAcceptorClient, error) {
	panic("not implemented") // TODO: Implement
}

// lncli: `closechannel`
// CloseChannel attempts to close an active channel identified by its channel
// outpoint (ChannelPoint). The actions of this method can additionally be
// augmented to attempt a force close after a timeout period in the case of an
// inactive peer. If a non-force close (cooperative closure) is requested,
// then the user can specify either a target number of blocks until the
// closure transaction is confirmed, or a manual fee rate. If neither are
// specified, then a default lax, block confirmation target is used.
func (lc *lightningClientMock) CloseChannel(ctx context.Context, in *lnrpc.CloseChannelRequest, opts ...grpc.CallOption) (lnrpc.Lightning_CloseChannelClient, error) {
	lc.Lock()
	defer lc.Unlock()
	lc.reqCloseChans = append(lc.reqCloseChans, in)

	return nextResponse(&lc.closeChans)
}

// lncli: `abandonchannel`
// AbandonChannel removes all channel state from the database except for a
// close summary. This method can be used to get rid of permanently unusable
// channels due to bugs fixed in newer versions of lnd. This method can also be
// used to remove externally funded channels where the funding transaction was
// never broadcast. Only available for non-externally funded channels in dev
// build.
func (lc *lightningClientMock) AbandonChannel(ctx context.Context, in *lnrpc.AbandonChannelRequest, opts ...grpc.CallOption) (*lnrpc.AbandonChannelResponse, error) {
	panic("not implemented") // TODO: Implement
}

// Deprecated: Do not use.
// lncli: `sendpayment`
// Deprecated, use routerrpc.SendPaymentV2. SendPayment dispatches a
// bi-directional streaming RPC for sending payments through the Lightning
// Network. A single RPC invocation creates a persistent bi-directional
// stream allowing clients to rapidly send payments through the Lightning
// Network with a single persistent connection.
func (lc *lightningClientMock) SendPayment(ctx context.Context, opts ...grpc.CallOption) (lnrpc.Lightning_SendPaymentClient, error) {
	panic("not implemented") // TODO: Implement
}

// SendPaymentSync is the synchronous non-streaming version of SendPayment.
// This RPC is intended to be consumed by clients of the REST proxy.
// Additionally, this RPC expects the destination's public key and the payment
// hash (if any) to be encoded as hex strings.
func (lc *lightningClientMock) SendPaymentSync(ctx context.Context, in *lnrpc.SendRequest, opts ...grpc.CallOption) (*lnrpc.SendResponse, error) {
	panic("not implemented") // TODO: Implement
}

// Deprecated: Do not use.
// lncli: `sendtoroute`
// Deprecated, use routerrpc.SendToRouteV2. SendToRoute is a bi-directional
// streaming RPC for sending payment through the Lightning Network. This
// method differs from SendPayment in that it allows users to specify a full
// route manually. This can be used for things like rebalancing, and atomic
// swaps.
func (lc *lightningClientMock) SendToRoute(ctx context.Context, opts ...grpc.CallOption) (lnrpc.Lightning_SendToRouteClient, error) {
	panic("not implemented") // TODO: Implement
}

// SendToRouteSync is a synchronous version of SendToRoute. It Will block
// until the payment either fails or succeeds.
func (lc *lightningClientMock) SendToRouteSync(ctx context.Context, in *lnrpc.SendToRouteRequest, opts ...grpc.CallOption) (*lnrpc.SendResponse, error) {
	panic("not implemented") // TODO: Implement
}

// lncli: `addinvoice`
// AddInvoice attempts to add a new invoice to the invoice database. Any
// duplicated invoices are rejected, therefore all invoices *must* have a
// unique payment preimage.
func (lc *lightningClientMock) AddInvoice(ctx context.Context, in *lnrpc.Invoice, opts ...grpc.CallOption) (*lnrpc.AddInvoiceResponse, error) {
	panic("not implemented") // TODO: Implement
}

// lncli: `listinvoices`
// ListInvoices returns a list of all the invoices currently stored within the
// database. Any active debug invoices are ignored. It has full support for
// paginated responses, allowing users to query for specific invoices through
// their add_index. This can be done by using either the first_index_offset or
// last_index_offset fields included in the response as the index_offset of the
// next request. By default, the first 100 invoices created will be returned.
// Backwards pagination is also supported through the Reversed flag.
func (lc *lightningClientMock) ListInvoices(ctx context.Context, in *lnrpc.ListInvoiceRequest, opts ...grpc.CallOption) (*lnrpc.ListInvoiceResponse, error) {
	panic("not implemented") // TODO: Implement
}

// lncli: `lookupinvoice`
// LookupInvoice attempts to look up an invoice according to its payment hash.
// The passed payment hash *must* be exactly 32 bytes, if not, an error is
// returned.
func (lc *lightningClientMock) LookupInvoice(ctx context.Context, in *lnrpc.PaymentHash, opts ...grpc.CallOption) (*lnrpc.Invoice, error) {
	panic("not implemented") // TODO: Implement
}

// SubscribeInvoices returns a uni-directional stream (server -> client) for
// notifying the client of newly added/settled invoices. The caller can
// optionally specify the add_index and/or the settle_index. If the add_index
// is specified, then we'll first start by sending add invoice events for all
// invoices with an add_index greater than the specified value. If the
// settle_index is specified, the next, we'll send out all settle events for
// invoices with a settle_index greater than the specified value. One or both
// of these fields can be set. If no fields are set, then we'll only send out
// the latest add/settle events.
func (lc *lightningClientMock) SubscribeInvoices(ctx context.Context, in *lnrpc.InvoiceSubscription, opts ...grpc.CallOption) (lnrpc.Lightning_SubscribeInvoicesClient, error) {
	panic("not implemented") // TODO: Implement
}

// lncli: `decodepayreq`
// DecodePayReq takes an encoded payment request string and attempts to decode
// it, returning a full description of the conditions encoded within the
// payment request.
func (lc *lightningClientMock) DecodePayReq(ctx context.Context, in *lnrpc.PayReqString, opts ...grpc.CallOption) (*lnrpc.PayReq, error) {
	panic("not implemented") // TODO: Implement
}

// lncli: `listpayments`
// ListPayments returns a list of all outgoing payments.
func (lc *lightningClientMock) ListPayments(ctx context.Context, in *lnrpc.ListPaymentsRequest, opts ...grpc.CallOption) (*lnrpc.ListPaymentsResponse, error) {
	panic("not implemented") // TODO: Implement
}

// DeleteAllPayments deletes all outgoing payments from DB.
func (lc *lightningClientMock) DeleteAllPayments(ctx context.Context, in *lnrpc.DeleteAllPaymentsRequest, opts ...grpc.CallOption) (*lnrpc.DeleteAllPaymentsResponse, error) {
	panic("not implemented") // TODO: Implement
}

// lncli: `describegraph`
// DescribeGraph returns a description of the latest graph state from the
// point of view of the node. The graph information is partitioned into two
// components: all the nodes/vertexes, and all the edges that connect the
// vertexes themselves. As this is a directed graph, the edges also contain
// the node directional specific routing policy which includes: the time lock
// delta, fee information, etc.
func (lc *lightningClientMock) DescribeGraph(ctx context.Context, in *lnrpc.ChannelGraphRequest, opts ...grpc.CallOption) (*lnrpc.ChannelGraph, error) {
	panic("not implemented") // TODO: Implement
}

// lncli: `getnodemetrics`
// GetNodeMetrics returns node metrics calculated from the graph. Currently
// the only supported metric is betweenness centrality of individual nodes.
func (lc *lightningClientMock) GetNodeMetrics(ctx context.Context, in *lnrpc.NodeMetricsRequest, opts ...grpc.CallOption) (*lnrpc.NodeMetricsResponse, error) {
	panic("not implemented") // TODO: Implement
}

// lncli: `getchaninfo`
// GetChanInfo returns the latest authenticated network announcement for the
// given channel identified by its channel ID: an 8-byte integer which
// uniquely identifies the location of transaction's funding output within the
// blockchain.
func (lc *lightningClientMock) GetChanInfo(ctx context.Context, in *lnrpc.ChanInfoRequest, opts ...grpc.CallOption) (*lnrpc.ChannelEdge, error) {
	panic("not implemented") // TODO: Implement
}

// lncli: `getnodeinfo`
// GetNodeInfo returns the latest advertised, aggregated, and authenticated
// channel information for the specified node identified by its public key.
func (lc *lightningClientMock) GetNodeInfo(ctx context.Context, in *lnrpc.NodeInfoRequest, opts ...grpc.CallOption) (*lnrpc.NodeInfo, error) {
	panic("not implemented") // TODO: Implement
}

// lncli: `queryroutes`
// QueryRoutes attempts to query the daemon's Channel Router for a possible
// route to a target destination capable of carrying a specific amount of
// atoms. The retuned route contains the full details required to craft and
// send an HTLC, also including the necessary information that should be
// present within the Sphinx packet encapsulated within the HTLC.
//
// When using REST, the `dest_custom_records` map type can be set by appending
// `&dest_custom_records[<record_number>]=<record_data_base64_url_encoded>`
// to the URL. Unfortunately this map type doesn't appear in the REST API
// documentation because of a bug in the grpc-gateway library.
func (lc *lightningClientMock) QueryRoutes(ctx context.Context, in *lnrpc.QueryRoutesRequest, opts ...grpc.CallOption) (*lnrpc.QueryRoutesResponse, error) {
	panic("not implemented") // TODO: Implement
}

// lncli: `getnetworkinfo`
// GetNetworkInfo returns some basic stats about the known channel graph from
// the point of view of the node.
func (lc *lightningClientMock) GetNetworkInfo(ctx context.Context, in *lnrpc.NetworkInfoRequest, opts ...grpc.CallOption) (*lnrpc.NetworkInfo, error) {
	panic("not implemented") // TODO: Implement
}

// lncli: `stop`
// StopDaemon will send a shutdown request to the interrupt handler, triggering
// a graceful shutdown of the daemon.
func (lc *lightningClientMock) StopDaemon(ctx context.Context, in *lnrpc.StopRequest, opts ...grpc.CallOption) (*lnrpc.StopResponse, error) {
	panic("not implemented") // TODO: Implement
}

// SubscribeChannelGraph launches a streaming RPC that allows the caller to
// receive notifications upon any changes to the channel graph topology from
// the point of view of the responding node. Events notified include: new
// nodes coming online, nodes updating their authenticated attributes, new
// channels being advertised, updates in the routing policy for a directional
// channel edge, and when channels are closed on-chain.
func (lc *lightningClientMock) SubscribeChannelGraph(ctx context.Context, in *lnrpc.GraphTopologySubscription, opts ...grpc.CallOption) (lnrpc.Lightning_SubscribeChannelGraphClient, error) {
	panic("not implemented") // TODO: Implement
}

// lncli: `debuglevel`
// DebugLevel allows a caller to programmatically set the logging verbosity of
// lnd. The logging can be targeted according to a coarse daemon-wide logging
// level, or in a granular fashion to specify the logging for a target
// sub-system.
func (lc *lightningClientMock) DebugLevel(ctx context.Context, in *lnrpc.DebugLevelRequest, opts ...grpc.CallOption) (*lnrpc.DebugLevelResponse, error) {
	panic("not implemented") // TODO: Implement
}

// lncli: `feereport`
// FeeReport allows the caller to obtain a report detailing the current fee
// schedule enforced by the node globally for each channel.
func (lc *lightningClientMock) FeeReport(ctx context.Context, in *lnrpc.FeeReportRequest, opts ...grpc.CallOption) (*lnrpc.FeeReportResponse, error) {
	panic("not implemented") // TODO: Implement
}

// lncli: `updatechanpolicy`
// UpdateChannelPolicy allows the caller to update the fee schedule and
// channel policies for all channels globally, or a particular channel.
func (lc *lightningClientMock) UpdateChannelPolicy(ctx context.Context, in *lnrpc.PolicyUpdateRequest, opts ...grpc.CallOption) (*lnrpc.PolicyUpdateResponse, error) {
	panic("not implemented") // TODO: Implement
}

// lncli: `fwdinghistory`
// ForwardingHistory allows the caller to query the htlcswitch for a record of
// all HTLCs forwarded within the target time range, and integer offset
// within that time range. If no time-range is specified, then the first chunk
// of the past 24 hrs of forwarding history are returned.
//
// A list of forwarding events are returned. The size of each forwarding event
// is 40 bytes, and the max message size able to be returned in gRPC is 4 MiB.
// As a result each message can only contain 50k entries. Each response has
// the index offset of the last entry. The index offset can be provided to the
// request to allow the caller to skip a series of records.
func (lc *lightningClientMock) ForwardingHistory(ctx context.Context, in *lnrpc.ForwardingHistoryRequest, opts ...grpc.CallOption) (*lnrpc.ForwardingHistoryResponse, error) {
	panic("not implemented") // TODO: Implement
}

// lncli: `exportchanbackup`
// ExportChannelBackup attempts to return an encrypted static channel backup
// for the target channel identified by it channel point. The backup is
// encrypted with a key generated from the aezeed seed of the user. The
// returned backup can either be restored using the RestoreChannelBackup
// method once lnd is running, or via the InitWallet and UnlockWallet methods
// from the WalletUnlocker service.
func (lc *lightningClientMock) ExportChannelBackup(ctx context.Context, in *lnrpc.ExportChannelBackupRequest, opts ...grpc.CallOption) (*lnrpc.ChannelBackup, error) {
	panic("not implemented") // TODO: Implement
}

// ExportAllChannelBackups returns static channel backups for all existing
// channels known to lnd. A set of regular singular static channel backups for
// each channel are returned. Additionally, a multi-channel backup is returned
// as well, which contains a single encrypted blob containing the backups of
// each channel.
func (lc *lightningClientMock) ExportAllChannelBackups(ctx context.Context, in *lnrpc.ChanBackupExportRequest, opts ...grpc.CallOption) (*lnrpc.ChanBackupSnapshot, error) {
	panic("not implemented") // TODO: Implement
}

// VerifyChanBackup allows a caller to verify the integrity of a channel backup
// snapshot. This method will accept either a packed Single or a packed Multi.
// Specifying both will result in an error.
func (lc *lightningClientMock) VerifyChanBackup(ctx context.Context, in *lnrpc.ChanBackupSnapshot, opts ...grpc.CallOption) (*lnrpc.VerifyChanBackupResponse, error) {
	panic("not implemented") // TODO: Implement
}

// lncli: `restorechanbackup`
// RestoreChannelBackups accepts a set of singular channel backups, or a
// single encrypted multi-chan backup and attempts to recover any funds
// remaining within the channel. If we are able to unpack the backup, then the
// new channel will be shown under listchannels, as well as pending channels.
func (lc *lightningClientMock) RestoreChannelBackups(ctx context.Context, in *lnrpc.RestoreChanBackupRequest, opts ...grpc.CallOption) (*lnrpc.RestoreBackupResponse, error) {
	panic("not implemented") // TODO: Implement
}

// SubscribeChannelBackups allows a client to sub-subscribe to the most up to
// date information concerning the state of all channel backups. Each time a
// new channel is added, we return the new set of channels, along with a
// multi-chan backup containing the backup info for all channels. Each time a
// channel is closed, we send a new update, which contains new new chan back
// ups, but the updated set of encrypted multi-chan backups with the closed
// channel(s) removed.
func (lc *lightningClientMock) SubscribeChannelBackups(ctx context.Context, in *lnrpc.ChannelBackupSubscription, opts ...grpc.CallOption) (lnrpc.Lightning_SubscribeChannelBackupsClient, error) {
	panic("not implemented") // TODO: Implement
}

// lncli: `bakemacaroon`
// BakeMacaroon allows the creation of a new macaroon with custom read and
// write permissions. No first-party caveats are added since this can be done
// offline.
func (lc *lightningClientMock) BakeMacaroon(ctx context.Context, in *lnrpc.BakeMacaroonRequest, opts ...grpc.CallOption) (*lnrpc.BakeMacaroonResponse, error) {
	panic("not implemented") // TODO: Implement
}

// lncli: `listmacaroonids`
// ListMacaroonIDs returns all root key IDs that are in use.
func (lc *lightningClientMock) ListMacaroonIDs(ctx context.Context, in *lnrpc.ListMacaroonIDsRequest, opts ...grpc.CallOption) (*lnrpc.ListMacaroonIDsResponse, error) {
	panic("not implemented") // TODO: Implement
}

// lncli: `deletemacaroonid`
// DeleteMacaroonID deletes the specified macaroon ID and invalidates all
// macaroons derived from that ID.
func (lc *lightningClientMock) DeleteMacaroonID(ctx context.Context, in *lnrpc.DeleteMacaroonIDRequest, opts ...grpc.CallOption) (*lnrpc.DeleteMacaroonIDResponse, error) {
	panic("not implemented") // TODO: Implement
}

// lncli: `listpermissions`
// ListPermissions lists all RPC method URIs and their required macaroon
// permissions to access them.
func (lc *lightningClientMock) ListPermissions(ctx context.Context, in *lnrpc.ListPermissionsRequest, opts ...grpc.CallOption) (*lnrpc.ListPermissionsResponse, error) {
	panic("not implemented") // TODO: Implement
}

// lncli: `enforcenodeping`
// EnforceNodePing attempts to ping the specified peer. If the request is
// canceled before a response is received from the remote peer, then this
// forces lnd to disconnect from the peer (and potentially attempt to reconnect).
func (lc *lightningClientMock) EnforceNodePing(ctx context.Context, in *lnrpc.EnforceNodePingRequest, opts ...grpc.CallOption) (*lnrpc.EnforceNodePingResponse, error) {
	panic("not implemented") // TODO: Implement
}

type clientStream struct{}

// Header returns the header metadata received from the server if there
// is any. It blocks if the metadata is not ready to read.
func (cs *clientStream) Header() (metadata.MD, error) {
	panic("not implemented") // TODO: Implement
}

// Trailer returns the trailer metadata from the server, if there is any.
// It must only be called after stream.CloseAndRecv has returned, or
// stream.Recv has returned a non-nil error (including io.EOF).
func (cs *clientStream) Trailer() metadata.MD {
	panic("not implemented") // TODO: Implement
}

// CloseSend closes the send direction of the stream. It closes the stream
// when non-nil error is met. It is also not safe to call CloseSend
// concurrently with SendMsg.
func (cs *clientStream) CloseSend() error {
	panic("not implemented") // TODO: Implement
}

// Context returns the context for this stream.
//
// It should not be called until after Header or RecvMsg has returned. Once
// called, subsequent client-side retries are disabled.
func (cs *clientStream) Context() context.Context {
	panic("not implemented") // TODO: Implement
}

// SendMsg is generally called by generated code. On error, SendMsg aborts
// the stream. If the error was generated by the client, the status is
// returned directly; otherwise, io.EOF is returned and the status of
// the stream may be discovered using RecvMsg.
//
// SendMsg blocks until:
//   - There is sufficient flow control to schedule m with the transport, or
//   - The stream is done, or
//   - The stream breaks.
//
// SendMsg does not wait until the message is received by the server. An
// untimely stream closure may result in lost messages. To ensure delivery,
// users should ensure the RPC completed successfully using RecvMsg.
//
// It is safe to have a goroutine calling SendMsg and another goroutine
// calling RecvMsg on the same stream at the same time, but it is not safe
// to call SendMsg on the same stream in different goroutines. It is also
// not safe to call CloseSend concurrently with SendMsg.
func (cs *clientStream) SendMsg(m interface{}) error {
	panic("not implemented") // TODO: Implement
}

// RecvMsg blocks until it receives a message into m or the stream is
// done. It returns io.EOF when the stream completes successfully. On
// any other error, the stream is aborted and the error contains the RPC
// status.
//
// It is safe to have a goroutine calling SendMsg and another goroutine
// calling RecvMsg on the same stream at the same time, but it is not
// safe to call RecvMsg on the same stream in different goroutines.
func (cs *clientStream) RecvMsg(m interface{}) error {
	panic("not implemented") // TODO: Implement
}
