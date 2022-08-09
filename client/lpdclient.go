package client

import (
	"bytes"
	"context"
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/decred/dcrd/dcrutil/v4"
	"github.com/decred/dcrlnd/lnrpc"
	"github.com/decred/dcrlnlpd/rpc"
	"github.com/decred/dcrlnlpd/rpc/lprpc_v1"
)

type ServerPolicy = lprpc_v1.PolicyResponse

// Config holds the config needed to start a new run of an LPD client.
type Config struct {
	// LC is the lightning client used to pay for the invoice and is the
	// target of the created channels.
	LC lnrpc.LightningClient

	// Address is the address of the LP server.
	Address string

	// Key is the optional key needed to request invoices from the LP
	// server.
	Key string

	// PolicyFetched is called once the LP policy for creating channels is
	// fetched. If this function returns an error, then channel creation is
	// aborted.
	PolicyFetched func(policy ServerPolicy) error

	// PayingInvoice is called just before starting to pay the invoice.
	PayingInvoice func(payHash string)

	// InvoicePaid is called after the invoice was settled.
	InvoicePaid func()

	// PendingChannel is called when the channel opened by the remote host
	// is found as pending.
	PendingChannel func(chanID string, capacity uint64)

	// Certificates are the bytes for a PEM-encoded certificate chain used
	// for the TLS connection.
	Certificates []byte
}

// Client is a client to the LN Liquidity Provider. It can perform a request for
// inbound liquidity to a remote server.
type Client struct {
	cfg Config
	lc  lnrpc.LightningClient
	lpc http.Client
}

func New(cfg Config) (*Client, error) {
	var tlsConfig *tls.Config
	if len(cfg.Certificates) > 0 {
		pool := x509.NewCertPool()
		pool.AppendCertsFromPEM(cfg.Certificates)
		tlsConfig = &tls.Config{
			RootCAs: pool,
		}
	}

	client := http.Client{
		Transport: &http.Transport{
			TLSClientConfig: tlsConfig,
		},
	}

	c := &Client{
		cfg: cfg,
		lc:  cfg.LC,
		lpc: client,
	}
	return c, nil
}

func (c *Client) watchForChannel(ctx context.Context, lpNodeID rpc.NodeID, channelOpenedChan chan error) {
	stream, err := c.lc.SubscribeChannelEvents(ctx, &lnrpc.ChannelEventSubscription{})
	if err != nil {
		channelOpenedChan <- err
		return
	}

	lpNodeStr := lpNodeID.String()

	for {
		evt, err := stream.Recv()
		if err != nil {
			channelOpenedChan <- err
			return
		}

		switch evt.Type {
		case lnrpc.ChannelEventUpdate_OPEN_CHANNEL:
			ch := evt.Channel.(*lnrpc.ChannelEventUpdate_OpenChannel)
			_ = ch
			channelOpenedChan <- nil
			return

		case lnrpc.ChannelEventUpdate_PENDING_OPEN_CHANNEL:
			if c.cfg.PendingChannel == nil {
				continue
			}

			// This event doesn't return the channel id or the node
			// id of the counterparty, so try to figure out by
			// listing the channels.
			chans, err := c.lc.PendingChannels(ctx, &lnrpc.PendingChannelsRequest{})
			if err != nil {
				continue
			}

			for _, ch := range chans.PendingOpenChannels {
				if ch.Channel.RemoteNodePub == lpNodeStr {
					c.cfg.PendingChannel(ch.Channel.ChannelPoint,
						uint64(ch.Channel.Capacity))
				}
			}

		}
	}
}

func (c *Client) lprpcV1Call(ctx context.Context, path string, req, res interface{}) error {
	url := c.cfg.Address + path
	b := bytes.NewBuffer(nil)
	if err := lprpc_v1.Encode(b, req); err != nil {
		return err
	}

	httpReq, err := http.NewRequestWithContext(ctx, "POST", url, b)
	if err != nil {
		return err
	}
	httpRes, err := c.lpc.Do(httpReq)
	if err != nil {
		return err
	}
	if httpRes.StatusCode != http.StatusOK {
		data, err := io.ReadAll(httpRes.Body)
		httpRes.Body.Close()
		if err != nil {
			return fmt.Errorf("unable to read contents of response %d: %v",
				httpRes.StatusCode, err)
		}
		return fmt.Errorf("LP responded with error: %q", string(data))
	}
	err = lprpc_v1.Decode(httpRes.Body, res)
	httpRes.Body.Close()
	if err != nil {
		return fmt.Errorf("unable to decode response: %v", err)
	}

	return nil
}

// EstimatedInvoiceAmount returns the estimated invoice amount from the LP
// server when the fee rate for creating the channel is the specified one.
func EstimatedInvoiceAmount(channelSize uint64, feeRate float64) uint64 {
	return uint64(float64(channelSize) * feeRate)
}

// RequestChannel requests a new channel from the liquidity provider with the
// specified target size (in atoms).
func (c *Client) RequestChannel(ctx context.Context, channelSize uint64) error {
	// Request the policy info from the remote node.
	var policy lprpc_v1.PolicyResponse
	if err := c.lprpcV1Call(ctx, "/api/v1/policy", nil, &policy); err != nil {
		return err
	}

	if len(policy.NodeAddresses) == 0 {
		return fmt.Errorf("remote node returned policy without public node addresses")
	}

	if c.cfg.PolicyFetched != nil {
		err := c.cfg.PolicyFetched(policy)
		if err != nil {
			return err
		}
	}

	// Connect to the remote node.
	//
	// TODO: improve how to select the address.
	lnAddr := &lnrpc.LightningAddress{
		Pubkey: policy.Node.String(),
		Host:   policy.NodeAddresses[0],
	}
	_, err := c.lc.ConnectPeer(ctx, &lnrpc.ConnectPeerRequest{Addr: lnAddr})
	if err != nil {
		if !strings.Contains(err.Error(), "already connected to peer:") {
			return fmt.Errorf("unable to connect to LP peer over LN: %v", err)
		}
	}

	nodeInfo, err := c.lc.GetInfo(ctx, &lnrpc.GetInfoRequest{})
	if err != nil {
		return err
	}

	var myID rpc.NodeID
	if err := myID.FromString(nodeInfo.IdentityPubkey); err != nil {
		return err
	}

	// Request the Invoice.
	invoiceReq := lprpc_v1.InvoiceRequest{
		TargetNode:  myID,
		ChannelSize: channelSize,
		Key:         c.cfg.Key,
	}
	var invoiceRes lprpc_v1.InvoiceResponse
	if err := c.lprpcV1Call(ctx, "/api/v1/invoice", &invoiceReq, &invoiceRes); err != nil {
		return err
	}

	// Decode and validate invoice.
	decodedInv, err := c.lc.DecodePayReq(ctx, &lnrpc.PayReqString{PayReq: invoiceRes.Invoice})
	if err != nil {
		return fmt.Errorf("unable to decode invoice: %v", err)
	}
	wantAtoms := EstimatedInvoiceAmount(channelSize, policy.ChanInvoiceFeeRate)
	if wantAtoms != uint64(decodedInv.NumAtoms) {
		return fmt.Errorf("LP sent invoice amount %s which is different than "+
			"expected %s", dcrutil.Amount(decodedInv.NumAtoms),
			dcrutil.Amount(wantAtoms))
	}

	// Watch to see if channel will be opened.
	watchCtx, cancel := context.WithCancel(ctx)
	defer cancel()
	chanOpenedChan := make(chan error)
	go c.watchForChannel(watchCtx, policy.Node, chanOpenedChan)

	// Pay for the invoice.
	if c.cfg.PayingInvoice != nil {
		c.cfg.PayingInvoice(decodedInv.PaymentHash)
	}
	payReq := &lnrpc.SendRequest{
		PaymentRequest: invoiceRes.Invoice,
	}
	payStream, err := c.lc.SendPayment(ctx)
	if err != nil {
		return err
	}
	if err := payStream.Send(payReq); err != nil {
		return err
	}

	// See if invoice will be settled.
	updt, err := payStream.Recv()
	if err != nil {
		return fmt.Errorf("error while waiting for invoice payment: %v", err)
	}

	if updt.PaymentError != "" {
		return fmt.Errorf("payment error: %s", updt.PaymentError)
	}

	if c.cfg.InvoicePaid != nil {
		c.cfg.InvoicePaid()
	}

	// Wait for channel to be opened.
	return <-chanOpenedChan
}
