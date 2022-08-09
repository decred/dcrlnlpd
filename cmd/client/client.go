package main

import (
	"context"
	"errors"
	"fmt"
	"os"

	"github.com/davecgh/go-spew/spew"
	"github.com/decred/dcrd/dcrutil/v4"
	"github.com/decred/dcrlnd/lnrpc"
	"github.com/decred/dcrlnlpd/client"
	"github.com/decred/slog"
	"github.com/jessevdk/go-flags"
)

type config struct {
	Address        string  `long:"addr" description:"Address of the LP server"`
	Key            string  `long:"key" description:"Key to enable creating the channel in the LP"`
	ChannelSize    float64 `long:"chansize" description:"Channel size to create"`
	Debug          bool    `long:"debug" description:"Log at debug level"`
	ServerCertPath string  `long:"servercertpath" description:"Path to the server TLS cert if it's self-signed"`

	// Dcrlnd Connection Options
	LNRPCHost      string `long:"lnrpchost" description:"Server address of the dcrlnd daemon"`
	LNTLSCertPath  string `long:"lntlscertpath" description:"Path to the tls.cert file of dcrlnd"`
	LNMacaroonPath string `long:"lnmacaroonpath" description:"Path to the macaroon file"`
}

var errCmdDone = errors.New("command done")

func _main() error {
	cfg := config{}
	parser := flags.NewParser(&cfg, flags.HelpFlag)
	_, err := parser.Parse()
	if err != nil {
		if e, ok := err.(*flags.Error); ok && e.Type == flags.ErrHelp {
			fmt.Fprintln(os.Stderr, err)
			return errCmdDone
		}
		return err
	}

	logBknd := slog.NewBackend(os.Stderr)
	log := logBknd.Logger("CLNT")
	if cfg.Debug {
		log.SetLevel(slog.LevelDebug)
	}

	chanSize, err := dcrutil.NewAmount(cfg.ChannelSize)
	if err != nil {
		return err
	}

	// Connect to dcrlnd.
	conn, err := connectToDcrlnd(cfg.LNRPCHost, cfg.LNTLSCertPath, cfg.LNMacaroonPath)
	if err != nil {
		return fmt.Errorf("unable to connect to dcrlnd: %v", err)
	}
	lc := lnrpc.NewLightningClient(conn)

	var cert []byte
	if cfg.ServerCertPath != "" {
		cert, err = os.ReadFile(cfg.ServerCertPath)
		if err != nil {
			return fmt.Errorf("unable to read server cert file: %v", err)
		}
	}

	ccfg := client.Config{
		LC:           lc,
		Address:      cfg.Address,
		Key:          cfg.Key,
		Certificates: cert,

		PolicyFetched: func(policy client.ServerPolicy) error {
			estInvoice := client.EstimatedInvoiceAmount(uint64(chanSize),
				policy.ChanInvoiceFeeRate)
			log.Infof("Fetched server policy. Estimated Invoice amount: %s",
				dcrutil.Amount(estInvoice))
			log.Debugf("Full server policy: %s", spew.Sdump(policy))
			return nil
		},

		PayingInvoice: func(payHash string) {
			log.Infof("Paying for invoice %s", payHash)
		},

		InvoicePaid: func() {
			log.Infof("Invoice paid. Waiting for channel to be opened")
		},

		PendingChannel: func(channelPoint string, capacity uint64) {
			log.Infof("Detected new pending channel %s with LP node with capacity %s",
				channelPoint, dcrutil.Amount(capacity))
		},
	}
	c, err := client.New(ccfg)
	if err != nil {
		return fmt.Errorf("unable to create client: %v", err)
	}

	ctx := context.Background()
	log.Infof("Requesting a channel of size %s", chanSize)
	err = c.RequestChannel(ctx, uint64(chanSize))
	if err != nil {
		return err
	}

	log.Infof("Channel opened successfully!")
	return nil
}

func main() {
	if err := _main(); err != nil && !errors.Is(err, errCmdDone) {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
