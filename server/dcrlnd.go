package server

import (
	"context"
	"fmt"
	"math"
	"os"
	"time"

	"github.com/decred/dcrlnd/lnrpc"
	"github.com/decred/dcrlnd/macaroons"
	"github.com/decred/dcrlnlpd/rpc"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"gopkg.in/macaroon.v2"
)

func connectToDcrlnd(addr, tlsCertPath, macaroonPath string) (*grpc.ClientConn, error) {
	// First attempt to establish a connection to lnd's RPC sever.
	creds, err := credentials.NewClientTLSFromFile(tlsCertPath, "")
	if err != nil {
		return nil, fmt.Errorf("unable to read cert file: %v", err)
	}
	opts := []grpc.DialOption{grpc.WithTransportCredentials(creds)}

	// Load the specified macaroon file.
	macBytes, err := os.ReadFile(macaroonPath)
	if err != nil {
		return nil, err
	}
	mac := &macaroon.Macaroon{}
	if err = mac.UnmarshalBinary(macBytes); err != nil {
		return nil, err
	}

	macOpt, err := macaroons.NewMacaroonCredential(mac)
	if err != nil {
		return nil, err
	}

	// Now we append the macaroon credentials to the dial options.
	opts = append(
		opts,
		grpc.WithPerRPCCredentials(macOpt),
		grpc.WithBlock(),
	)

	// Block until connection happens or fail.
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	conn, err := grpc.DialContext(ctx, addr, opts...)
	if err != nil {
		return nil, fmt.Errorf("unable to dial to dcrlnd's gRPC server at %v: %v", addr, err)
	}
	return conn, nil
}

// CheckDcrlnd connects and performs sanity checks on the given dcrlnd instance.
func CheckDcrlnd(addr, tlsCertPath, macaroonPath string) error {
	_, err := connectToDcrlnd(addr, tlsCertPath, macaroonPath)
	return err
}

// isConnectedToNode returns true if the server is connected to the specified
// node.
func (s *Server) isConnectedToNode(ctx context.Context, node rpc.NodeID) bool {
	// Maybe cache the list of peers for some time?
	wantPeer := node.String()
	peers, err := s.lc.ListPeers(ctx, &lnrpc.ListPeersRequest{})
	if err != nil {
		s.log.Errorf("Unable to list peers: %v", err)
		return false
	}
	for _, peer := range peers.Peers {
		if peer.PubKey == wantPeer {
			return true
		}
	}
	return false
}

func (s *Server) hasMaxPendingChannels(ctx context.Context) bool {
	chans, err := s.lc.PendingChannels(ctx, &lnrpc.PendingChannelsRequest{})
	if err != nil {
		s.log.Errorf("Unable to query pending channels: %v", err)
		return true
	}
	return uint(len(chans.PendingOpenChannels)) >= s.cfg.MaxPendingChans
}

// hasUnspentForNewChannel returns true if there are enough unspent utxos for a
// channel of the given size to be created.
func (s *Server) hasUnspentForNewChannel(ctx context.Context, chanSize uint64) bool {
	req := &lnrpc.ListUnspentRequest{MinConfs: 1, MaxConfs: math.MaxInt32}
	res, err := s.lc.ListUnspent(ctx, req)
	if err != nil {
		s.log.Errorf("Unable to query utxos: %v", err)
		return false
	}

	var sum uint64
	for _, utxo := range res.Utxos {
		sum += uint64(utxo.AmountAtoms)
		if sum > chanSize {
			return true
		}
	}
	return false
}
