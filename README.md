# DCR LN Liquidity Provider Daemon

This service allows running a Lightning Network Liquidity Provider in the Decred
network.

This LP allows remote clients to request the node associated to the LP to open
an LN channel back to the requesting client. This allows the requesting client
to have some inbound bandwidth to receive LN payments.

To create the channel, the LP charges some amount, specified as a percentage
of the desired channel size.

## Building & Running

Requires Go 1.17+. Download the code, then run

```
$ go install .
$ dcrlnlpd
```

## Configuration

A default config file is created in `~/.dcrlnlpd/dcrlnlpd.conf` if it does not
exist. CLI parameters can be found by running `-h/--help`.

## Client

A client implementation is available in the `github.com/decred/dcrlnlpd/client`
package. A CLI client is also provided in the `cmd/client` subdir.

An example invocation of the CLI client is the following:

```
go run ./cmd/client/ \
  --lnrpchost localhost:20100 \
  --lntlscertpath ~/dcrlndsimnetnodes/dcrlnd1/tls.cert \
  --lnmacaroonpath ~/dcrlndsimnetnodes/dcrlnd1/chain/decred/simnet/admin.macaroon \
  --addr http://localhost:29130 \
  --chansize 0.001
```

## License

dcrlnlpd is licensed under the [copyfree](http://copyfree.org) ISC License.

