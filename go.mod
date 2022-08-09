module github.com/decred/dcrlnlpd

go 1.18

require (
	github.com/davecgh/go-spew v1.1.1
	github.com/decred/dcrd/chaincfg/chainhash v1.0.3
	github.com/decred/dcrd/chaincfg/v3 v3.1.1
	github.com/decred/dcrd/dcrutil/v4 v4.0.0
	github.com/decred/dcrd/wire v1.5.0
	github.com/decred/dcrlnd v0.3.4
	github.com/decred/slog v1.2.0
	github.com/gorilla/mux v1.7.4
	github.com/jessevdk/go-flags v1.5.0
	github.com/jrick/logrotate v1.0.0
	golang.org/x/sync v0.0.0-20210220032951-036812b2e83c
	google.golang.org/grpc v1.46.0
	gopkg.in/macaroon.v2 v2.1.0
)

require (
	decred.org/cspp/v2 v2.0.0 // indirect
	decred.org/dcrwallet/v2 v2.0.0 // indirect
	github.com/agl/ed25519 v0.0.0-20170116200512-5312a6153412 // indirect
	github.com/beorn7/perks v1.0.1 // indirect
	github.com/btcsuite/btcwallet/walletdb v1.4.0 // indirect
	github.com/companyzero/sntrup4591761 v0.0.0-20200131011700-2b0d299dbd22 // indirect
	github.com/coreos/bbolt v1.3.3 // indirect
	github.com/coreos/go-semver v0.2.0 // indirect
	github.com/coreos/go-systemd v0.0.0-20190620071333-e64a0ec8b42a // indirect
	github.com/coreos/pkg v0.0.0-20180108230652-97fdf19511ea // indirect
	github.com/dchest/siphash v1.2.2 // indirect
	github.com/decred/base58 v1.0.3 // indirect
	github.com/decred/dcrd v1.2.1-0.20220120180802-74a67dce2480 // indirect
	github.com/decred/dcrd/bech32 v1.1.2 // indirect
	github.com/decred/dcrd/blockchain/stake/v4 v4.0.0 // indirect
	github.com/decred/dcrd/blockchain/standalone/v2 v2.1.0 // indirect
	github.com/decred/dcrd/blockchain/v4 v4.0.0 // indirect
	github.com/decred/dcrd/certgen v1.1.1 // indirect
	github.com/decred/dcrd/chaincfg v1.5.2 // indirect
	github.com/decred/dcrd/connmgr v1.1.0 // indirect
	github.com/decred/dcrd/crypto/blake256 v1.0.0 // indirect
	github.com/decred/dcrd/crypto/ripemd160 v1.0.1 // indirect
	github.com/decred/dcrd/database/v3 v3.0.0 // indirect
	github.com/decred/dcrd/dcrec v1.0.0 // indirect
	github.com/decred/dcrd/dcrec/edwards/v2 v2.0.2 // indirect
	github.com/decred/dcrd/dcrec/secp256k1/v4 v4.0.1 // indirect
	github.com/decred/dcrd/dcrjson/v4 v4.0.0 // indirect
	github.com/decred/dcrd/gcs/v3 v3.0.0 // indirect
	github.com/decred/dcrd/hdkeychain/v3 v3.1.0 // indirect
	github.com/decred/dcrd/lru v1.1.1 // indirect
	github.com/decred/dcrd/rpc/jsonrpc/types/v3 v3.0.0 // indirect
	github.com/decred/dcrd/rpcclient/v7 v7.0.0 // indirect
	github.com/decred/dcrd/txscript/v4 v4.0.0 // indirect
	github.com/decred/go-socks v1.1.0 // indirect
	github.com/decred/lightning-onion/v4 v4.0.0 // indirect
	github.com/dgrijalva/jwt-go v3.0.0+incompatible // indirect
	github.com/dustin/go-humanize v1.0.0 // indirect
	github.com/go-errors/errors v1.0.1 // indirect
	github.com/gogo/protobuf v1.2.1 // indirect
	github.com/golang/protobuf v1.5.2 // indirect
	github.com/golang/snappy v0.0.4 // indirect
	github.com/google/btree v1.0.0 // indirect
	github.com/google/uuid v1.1.2 // indirect
	github.com/gorilla/websocket v1.4.2 // indirect
	github.com/grpc-ecosystem/go-grpc-middleware v1.0.0 // indirect
	github.com/grpc-ecosystem/go-grpc-prometheus v1.2.0 // indirect
	github.com/grpc-ecosystem/grpc-gateway v1.16.0 // indirect
	github.com/jonboulle/clockwork v0.1.0 // indirect
	github.com/jrick/bitset v1.0.0 // indirect
	github.com/jrick/wsrpc/v2 v2.3.4 // indirect
	github.com/json-iterator/go v1.1.7 // indirect
	github.com/juju/loggo v0.0.0-20180524022052-584905176618 // indirect
	github.com/konsorten/go-windows-terminal-sequences v1.0.1 // indirect
	github.com/matheusd/etcd v0.5.0-alpha.5.0.20201012144914-e63dcc3d7528 // indirect
	github.com/matttproud/golang_protobuf_extensions v1.0.2-0.20181231171920-c182affec369 // indirect
	github.com/miekg/dns v1.1.3 // indirect
	github.com/modern-go/concurrent v0.0.0-20180306012644-bacd9c7ef1dd // indirect
	github.com/modern-go/reflect2 v1.0.1 // indirect
	github.com/prometheus/client_golang v0.9.3 // indirect
	github.com/prometheus/client_model v0.0.0-20190812154241-14fe0d1b01d4 // indirect
	github.com/prometheus/common v0.4.0 // indirect
	github.com/prometheus/procfs v0.0.0-20190507164030-5867b95ac084 // indirect
	github.com/rogpeppe/fastuuid v1.2.0 // indirect
	github.com/sirupsen/logrus v1.2.0 // indirect
	github.com/soheilhy/cmux v0.1.4 // indirect
	github.com/syndtr/goleveldb v1.0.1-0.20210819022825-2ae1ddf74ef7 // indirect
	github.com/tmc/grpc-websocket-proxy v0.0.0-20170815181823-89b8d40f7ca8 // indirect
	github.com/xiang90/probing v0.0.0-20160813154853-07dd2e8dfe18 // indirect
	go.etcd.io/bbolt v1.3.5 // indirect
	go.uber.org/atomic v1.6.0 // indirect
	go.uber.org/multierr v1.5.0 // indirect
	go.uber.org/zap v1.10.0 // indirect
	golang.org/x/crypto v0.0.0-20210711020723-a769d52b0f97 // indirect
	golang.org/x/net v0.0.0-20210226172049-e18ecbb05110 // indirect
	golang.org/x/sys v0.0.0-20211216021012-1d35b9e2eb4e // indirect
	golang.org/x/term v0.0.0-20201126162022-7de9c90e9dd1 // indirect
	golang.org/x/text v0.3.3 // indirect
	golang.org/x/time v0.0.0-20191024005414-555d28b269f0 // indirect
	google.golang.org/genproto v0.0.0-20200924141100-a14c0a98937d // indirect
	google.golang.org/protobuf v1.27.1 // indirect
	gopkg.in/errgo.v1 v1.0.0 // indirect
	gopkg.in/macaroon-bakery.v2 v2.1.0 // indirect
	gopkg.in/yaml.v2 v2.3.0 // indirect
	sigs.k8s.io/yaml v1.1.0 // indirect
)
