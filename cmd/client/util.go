package main

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/decred/dcrlnd/macaroons"
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
		return nil, fmt.Errorf("unable to dial to dcrlnd's gRPC server: %v", err)
	}
	return conn, nil
}
