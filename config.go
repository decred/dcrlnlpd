package main

import (
	"crypto/tls"
	_ "embed"
	"errors"
	"fmt"
	"net"
	"os"
	"os/user"
	"path/filepath"
	"runtime"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/decred/dcrd/chaincfg/v3"
	"github.com/decred/dcrd/dcrutil/v4"
	"github.com/decred/dcrlnlpd/internal/version"
	"github.com/decred/dcrlnlpd/server"
	"github.com/decred/slog"
	"github.com/jessevdk/go-flags"
)

// appName is the generic name of the app.
var appName = "dcrlnlpd"

//go:embed dcrlnpd.conf
var defaultConfigFileContents []byte

type chainNetwork string

const (
	cnMainNet chainNetwork = "mainnet"
	cnTestNet chainNetwork = "testnet"
	cnSimNet  chainNetwork = "simnet"
)

// defaultListenPort is the default port to use with net.JoinHostPort().
func (c chainNetwork) defaultListenPort() string {
	switch c {
	case cnMainNet:
		return "9130"
	case cnTestNet:
		return "19130"
	case cnSimNet:
		return "29130"
	default:
		panic("unknown chainNetwork")
	}
}

func (c chainNetwork) chainParams() *chaincfg.Params {
	switch c {
	case cnMainNet:
		return chaincfg.MainNetParams()
	case cnTestNet:
		return chaincfg.TestNet3Params()
	case cnSimNet:
		return chaincfg.SimNetParams()
	default:
		panic("unknown chainNetwork")
	}
}

const (
	defaultLogLevel    = "info"
	defaultActiveNet   = cnMainNet
	defaultDataDirname = "data"
	defaultLogDirname  = "logs"
	defaultLNHost      = "localhost:10009"

	defaultOpenMinChanSize     = 0.00040000
	defaultOpenMaxChanSize     = 1.00000000
	defaultOpenMaxNbChannels   = 2
	defaultOpenInvoiceFeeRate  = 0.05
	defaultOpenMaxPendingChans = 1
	defaultInvoiceExpiration   = time.Hour

	defaultCloseCheckInterval    = time.Hour
	defaultCloseMinChanLifetime  = time.Hour * 24 * 7
	defaultCloseMinWalletBalance = 1.0
)

var (
	defaultConfigFilename     = appName + ".conf"
	defaultConfigDir          = dcrutil.AppDataDir(appName, false)
	defaultDataDir            = filepath.Join(defaultConfigDir, defaultDataDirname)
	defaultLogDir             = filepath.Join(defaultConfigDir, defaultLogDirname, string(defaultActiveNet))
	defaultConfigFile         = filepath.Join(defaultConfigDir, defaultConfigFilename)
	defaultDcrlndDir          = dcrutil.AppDataDir("dcrlnd", false)
	defaultDcrlndTLSCertPath  = filepath.Join(defaultDcrlndDir, "tls.cert")
	defaultDcrlndMacaroonPath = filepath.Join(defaultDcrlndDir, "data", "chain", "decred", string(defaultActiveNet), "admin.macaroon")

	errCmdDone = errors.New("cmd is done while parsing config options")
)

type openChanPolicyCfg struct {
	MinChanSize        float64       `long:"minchansize" description:"Minimum channel size in dcr"`
	MaxChanSize        float64       `long:"maxchansize" description:"Maximum channel size in dcr"`
	MaxNbChannels      uint          `long:"maxnbchans" description:"Maximum number of channels to open"`
	MaxPendingChannels uint          `long:"maxpendingchans" description:"Maximum number of pending channels the dcrlnd node is configured to accept"`
	InvoiceFeeRate     float64       `long:"invoicefeerate" description:"Fee rate to charge for opening the channel as a percentage of the channel size"`
	Key                string        `long:"key" description:"Only allow creating channels when the request comes with the specified key"`
	InvoiceExpiration  time.Duration `long:"invoiceexpiration" description:"Invoice expiration duration"`
}

type closeChanPolicyCfg struct {
	CheckInterval    time.Duration `long:"checkinterval" description:"How often to check for the close policy"`
	MinChanLifetime  time.Duration `long:"minchanlifetime" description:"Minimum amount of time the channel will be kept online"`
	MinWalletBalance float64       `long:"minwalletbalance" description:"The minimum wallet balance below which channels will start to be closed"`
}

type ignorePolicyCfg struct {
	Channels []string `long:"channels" description:"List of channels to ignore for management purposes"`
	Nodes    []string `long:"nodes" description:"List of nodes to ignore for management purposes"`
}

type config struct {
	ShowVersion bool `short:"V" long:"version" description:"Display version information and exit"`

	// Config Selection

	AppData    string `short:"A" long:"appdata" description:"Path to application home directory"`
	ConfigFile string `short:"C" long:"configfile" description:"Path to configuration file"`

	// General

	DebugLevel string `short:"d" long:"debuglevel" description:"Logging level for all subsystems {trace, debug, info, warn, error, critical} -- You may also specify <subsystem>=<level>,<subsystem2>=<level>,... to set the log level for individual subsystems -- Use show to list available subsystems"`

	// Listeners

	Listeners []string `long:"listen" description:"Add an interface/port to listen for connections"`
	Profile   string   `long:"profile" description:"Enable HTTP profiling on given [addr:]port -- NOTE port must be between 1024 and 65536"`

	// TLS related

	TLSCert    string `long:"tlscert" description:"File containing the TLS certificate"`
	TLSKey     string `long:"tlskey" description:"File containing the TLS certificate key"`
	DisableTLS bool   `long:"disabletls" description:"Disable TLS encryption"`

	// Network

	MainNet bool `long:"mainnet" description:"Use the main network"`
	TestNet bool `long:"testnet" description:"Use the test network"`
	SimNet  bool `long:"simnet" description:"Use the simulation test network"`

	// Dcrlnd Connection Options
	LNRPCHost      string   `long:"lnrpchost" description:"Server address of the dcrlnd daemon"`
	LNTLSCertPath  string   `long:"lntlscertpath" description:"Path to the tls.cert file of dcrlnd"`
	LNMacaroonPath string   `long:"lnmacaroonpath" description:"Path to the macaroon file"`
	LNNodeAddrs    []string `long:"lnnodeaddr" description:"Public address of the underlying LN node in the P2P network"`

	// Policy Config.
	OpenPolicy   openChanPolicyCfg  `group:"Open Channel Policy" namespace:"openpolicy"`
	ClosePolicy  closeChanPolicyCfg `group:"Close Channel Policy" namespace:"closepolicy"`
	IgnorePolicy ignorePolicyCfg    `group:"Ignore Policy" namespace:"ignorepolicy"`

	// The rest of the members of this struct are filled by loadConfig().

	activeNet chainNetwork
}

// listeners returns the interface listeners where connections to the http
// server should be accepted.
func (c *config) listeners() ([]net.Listener, error) {

	listenFunc := func(addr string) (net.Listener, error) {
		return net.Listen("tcp", addr)
	}

	if !c.DisableTLS {
		keypair, err := tls.LoadX509KeyPair(c.TLSCert, c.TLSKey)
		if err != nil {
			return nil, err
		}

		tlsConfig := tls.Config{
			Certificates: []tls.Certificate{keypair},
			MinVersion:   tls.VersionTLS12,
		}

		listenFunc = func(laddr string) (net.Listener, error) {
			return tls.Listen("tcp", laddr, &tlsConfig)
		}
	}

	list := make([]net.Listener, 0, len(c.Listeners))
	for _, addr := range c.Listeners {
		l, err := listenFunc(addr)
		if err != nil {
			// Cancel listening on the other addresses since we'll
			// return an error.
			for _, l := range list {
				// Ignore close errors since we'll be returning
				// an error anyway.
				l.Close()
			}
			return nil, fmt.Errorf("unable to listen on %s: %v", addr, err)
		}
		list = append(list, l)
	}
	return list, nil
}

func (c *config) serverConfig() (*server.Config, error) {
	var minChanSize, maxChanSize, minWalletBal dcrutil.Amount
	var err error
	if minChanSize, err = dcrutil.NewAmount(c.OpenPolicy.MinChanSize); err != nil {
		return nil, fmt.Errorf("invalid min chan size: %v", err)
	}
	if maxChanSize, err = dcrutil.NewAmount(c.OpenPolicy.MaxChanSize); err != nil {
		return nil, fmt.Errorf("invalid max chan size: %v", err)
	}
	if minWalletBal, err = dcrutil.NewAmount(c.ClosePolicy.MinWalletBalance); err != nil {
		return nil, fmt.Errorf("invalid min wallet balance: %v", err)
	}

	var createKey []byte
	if c.OpenPolicy.Key != "" {
		createKey = []byte(c.OpenPolicy.Key)
	}

	ignoreNodes := make(map[string]struct{}, len(c.IgnorePolicy.Nodes))
	for _, s := range c.IgnorePolicy.Nodes {
		ignoreNodes[s] = struct{}{}
	}
	ignoreChannels := make(map[string]struct{}, len(c.IgnorePolicy.Channels))
	for _, s := range c.IgnorePolicy.Channels {
		ignoreChannels[s] = struct{}{}
	}

	return &server.Config{
		ChainParams:    c.activeNet.chainParams(),
		LNRPCHost:      c.LNRPCHost,
		LNTLSCertPath:  c.LNTLSCertPath,
		LNMacaroonPath: c.LNMacaroonPath,
		LNNodeAddrs:    c.LNNodeAddrs,
		RootDir:        filepath.Join(c.AppData, string(c.activeNet)),
		Log:            srvrLog,

		MinChanSize:        uint64(minChanSize),
		MaxChanSize:        uint64(maxChanSize),
		MaxNbChannels:      c.OpenPolicy.MaxNbChannels,
		MaxPendingChans:    c.OpenPolicy.MaxPendingChannels,
		MinWalletBalance:   minWalletBal,
		ChanInvoiceFeeRate: c.OpenPolicy.InvoiceFeeRate,
		InvoiceExpiration:  c.OpenPolicy.InvoiceExpiration,
		CreateKey:          createKey,
		CloseCheckInterval: c.ClosePolicy.CheckInterval,
		MinChanLifetime:    c.ClosePolicy.MinChanLifetime,
		IgnoreNodes:        ignoreNodes,
		IgnoreChannels:     ignoreChannels,
	}, nil
}

// configuredNetwork returns the network configured in the currently setup
// settings.
func (c *config) configuredNetwork() (chainNetwork, error) {
	// Multiple networks can't be selected simultaneously.  Count number of
	// network flags passed and assign active network params.
	numNets := 0
	net := defaultActiveNet
	if c.MainNet {
		numNets++
		net = cnMainNet
	}
	if c.TestNet {
		numNets++
		net = cnTestNet
	}
	if c.SimNet {
		numNets++
		net = cnSimNet
	}
	if numNets > 1 {
		err := fmt.Errorf("mainnet, testnet and simnet params can't be " +
			"used together -- choose one of the three")
		return net, err
	}
	return net, nil
}

// validLogLevel returns whether or not logLevel is a valid debug log level.
func validLogLevel(logLevel string) bool {
	_, ok := slog.LevelFromString(logLevel)
	return ok
}

// supportedSubsystems returns a sorted slice of the supported subsystems for
// logging purposes.
func supportedSubsystems() []string {
	// Convert the subsystemLoggers map keys to a slice.
	subsystems := make([]string, 0, len(subsystemLoggers))
	for subsysID := range subsystemLoggers {
		subsystems = append(subsystems, subsysID)
	}

	// Sort the subsystems for stable display.
	sort.Strings(subsystems)
	return subsystems
}

// parseAndSetDebugLevels attempts to parse the specified debug level and set
// the levels accordingly.  An appropriate error is returned if anything is
// invalid.
func parseAndSetDebugLevels(debugLevel string) error {
	// When the specified string doesn't have any delimiters, treat it as
	// the log level for all subsystems.
	if !strings.Contains(debugLevel, ",") && !strings.Contains(debugLevel, "=") {
		// Validate debug log level.
		if !validLogLevel(debugLevel) {
			str := "the specified debug level [%v] is invalid"
			return fmt.Errorf(str, debugLevel)
		}

		// Change the logging level for all subsystems.
		setLogLevels(debugLevel)

		return nil
	}

	// Split the specified string into subsystem/level pairs while detecting
	// issues and update the log levels accordingly.
	for _, logLevelPair := range strings.Split(debugLevel, ",") {
		if !strings.Contains(logLevelPair, "=") {
			str := "the specified debug level contains an invalid " +
				"subsystem/level pair [%v]"
			return fmt.Errorf(str, logLevelPair)
		}

		// Extract the specified subsystem and log level.
		fields := strings.Split(logLevelPair, "=")
		subsysID, logLevel := fields[0], fields[1]

		// Validate subsystem.
		if _, exists := subsystemLoggers[subsysID]; !exists {
			str := "the specified subsystem [%v] is invalid -- " +
				"supported subsystems %v"
			return fmt.Errorf(str, subsysID, supportedSubsystems())
		}

		// Validate log level.
		if !validLogLevel(logLevel) {
			str := "the specified debug level [%v] is invalid"
			return fmt.Errorf(str, logLevel)
		}

		setLogLevel(subsysID, logLevel)
	}

	return nil
}

// cleanAndExpandPath expands environment variables and leading ~ in the passed
// path, cleans the result, and returns it.
func cleanAndExpandPath(path string) string {
	// Nothing to do when no path is given.
	if path == "" {
		return path
	}

	// NOTE: The os.ExpandEnv doesn't work with Windows cmd.exe-style
	// %VARIABLE%, but the variables can still be expanded via POSIX-style
	// $VARIABLE.
	path = os.ExpandEnv(path)

	if !strings.HasPrefix(path, "~") {
		return filepath.Clean(path)
	}

	// Expand initial ~ to the current user's home directory, or ~otheruser
	// to otheruser's home directory.  On Windows, both forward and backward
	// slashes can be used.
	path = path[1:]

	var pathSeparators string
	if runtime.GOOS == "windows" {
		pathSeparators = string(os.PathSeparator) + "/"
	} else {
		pathSeparators = string(os.PathSeparator)
	}

	userName := ""
	if i := strings.IndexAny(path, pathSeparators); i != -1 {
		userName = path[:i]
		path = path[i:]
	}

	homeDir := ""
	var u *user.User
	var err error
	if userName == "" {
		u, err = user.Current()
	} else {
		u, err = user.Lookup(userName)
	}
	if err == nil {
		homeDir = u.HomeDir
	}
	// Fallback to CWD if user lookup fails or user has no home directory.
	if homeDir == "" {
		homeDir = "."
	}

	return filepath.Join(homeDir, path)
}

func loadConfig() (*config, []string, error) {
	// Default config.
	cfg := config{
		ConfigFile:     defaultConfigFile,
		DebugLevel:     defaultLogLevel,
		LNRPCHost:      defaultLNHost,
		LNTLSCertPath:  defaultDcrlndTLSCertPath,
		LNMacaroonPath: defaultDcrlndMacaroonPath,

		OpenPolicy: openChanPolicyCfg{
			MinChanSize:        defaultOpenMinChanSize,
			MaxChanSize:        defaultOpenMaxChanSize,
			MaxNbChannels:      defaultOpenMaxNbChannels,
			MaxPendingChannels: defaultOpenMaxPendingChans,
			InvoiceFeeRate:     defaultOpenInvoiceFeeRate,
			InvoiceExpiration:  defaultInvoiceExpiration,
		},
		ClosePolicy: closeChanPolicyCfg{
			CheckInterval:    defaultCloseCheckInterval,
			MinChanLifetime:  defaultCloseMinChanLifetime,
			MinWalletBalance: defaultCloseMinWalletBalance,
		},
	}

	// Pre-parse the command line options to see if an alternative config
	// file was specified.  Any errors aside from the
	// help message error can be ignored here since they will be caught by
	// the final parse below.
	preCfg := cfg
	preParser := flags.NewParser(&preCfg, flags.HelpFlag)
	_, err := preParser.Parse()
	if err != nil {
		if e, ok := err.(*flags.Error); ok && e.Type == flags.ErrHelp {
			fmt.Fprintln(os.Stderr, err)
			return nil, nil, errCmdDone
		}
	}

	// Show the version and exit if the version flag was specified.
	appName := filepath.Base(os.Args[0])
	appName = strings.TrimSuffix(appName, filepath.Ext(appName))
	usageMessage := fmt.Sprintf("Use %s -h to show usage", appName)
	if preCfg.ShowVersion {
		fmt.Printf("%s version %s (Go version %s %s/%s)\n",
			appName, version.String(),
			runtime.Version(), runtime.GOOS, runtime.GOARCH)
		return nil, nil, errCmdDone
	}

	// Special show command to list supported subsystems and exit.
	if preCfg.DebugLevel == "show" {
		fmt.Println("Supported subsystems", supportedSubsystems())
		return nil, nil, errCmdDone
	}

	// Modify default dirs if a network was specified in the command line.
	cmdLineNet, err := preCfg.configuredNetwork()
	if err != nil {
		return nil, nil, err
	}
	if cmdLineNet != defaultActiveNet {
		cfg.LNMacaroonPath = filepath.Join(defaultDcrlndDir, "data",
			"chain", "decred", string(cmdLineNet), "admin.macaroon")
	}

	// If the default config file was specified and it does not exist,
	// create it.
	if preCfg.ConfigFile == defaultConfigFile {
		if _, err := os.Stat(defaultConfigFile); os.IsNotExist(err) {
			if err := os.MkdirAll(defaultConfigDir, 0700); err != nil {
				return nil, nil, err
			}
			err := os.WriteFile(defaultConfigFile,
				defaultConfigFileContents, 0700)
			if err != nil {
				return nil, nil, fmt.Errorf("unable to create "+
					"default config file %q: %v", defaultConfigFile,
					err)
			}
		}
	}

	// Update the home directory if --appdata is specified. Since the home
	// directory is updated, other variables need to be updated to reflect
	// the new changes.
	if preCfg.AppData != "" {
		cfg.AppData, _ = filepath.Abs(cleanAndExpandPath(preCfg.AppData))

		if preCfg.ConfigFile == defaultConfigFile {
			defaultConfigFile = filepath.Join(cfg.AppData,
				defaultConfigFilename)
			preCfg.ConfigFile = defaultConfigFile
			cfg.ConfigFile = defaultConfigFile
		} else {
			cfg.ConfigFile = preCfg.ConfigFile
		}
		defaultDataDir = filepath.Join(cfg.AppData, defaultDataDirname)
		defaultLogDir = filepath.Join(cfg.AppData, defaultLogDirname, string(defaultActiveNet))
	}

	// Load additional config from file.
	var configFileError error
	parser := flags.NewParser(&cfg, flags.Default)

	err = flags.NewIniParser(parser).ParseFile(preCfg.ConfigFile)
	if err != nil {
		if _, ok := err.(*os.PathError); !ok {
			fmt.Fprintf(os.Stderr, "Error parsing config "+
				"file: %v\n", err)
			fmt.Fprintln(os.Stderr, usageMessage)
			return nil, nil, err
		}
		configFileError = err
	}

	// If the AppData dir in the cfg file is not empty and a precfg AppData
	// was not specified, then use the config file's AppData for data and
	// log dir.
	if cfg.AppData != "" && preCfg.AppData == "" {
		defaultDataDir = cleanAndExpandPath(filepath.Join(cfg.AppData, defaultDataDirname))
		defaultLogDir = cleanAndExpandPath(filepath.Join(cfg.AppData, defaultLogDirname, string(defaultActiveNet)))
	}

	// Parse command line options again to ensure they take precedence.
	remainingArgs, err := parser.Parse()
	if err != nil {
		if e, ok := err.(*flags.Error); !ok || e.Type != flags.ErrHelp {
			fmt.Fprintln(os.Stderr, usageMessage)
		}
		return nil, nil, err
	}

	// Create the home directory if it doesn't already exist.
	funcName := "loadConfig"
	err = os.MkdirAll(defaultDataDir, 0700)
	if err != nil {
		// Show a nicer error message if it's because a symlink is
		// linked to a directory that does not exist (probably because
		// it's not mounted).
		if e, ok := err.(*os.PathError); ok && os.IsExist(err) {
			if link, lerr := os.Readlink(e.Path); lerr == nil {
				str := "is symlink %s -> %s mounted?"
				err = fmt.Errorf(str, e.Path, link)
			}
		}

		str := "%s: Failed to create home directory: %v"
		err := fmt.Errorf(str, funcName, err)
		fmt.Fprintln(os.Stderr, err)
		return nil, nil, err
	}

	// Multiple networks can't be selected simultaneously.  Count number of
	// network flags passed and assign active network params.
	numNets := 0
	cfg.activeNet = defaultActiveNet
	if cfg.MainNet {
		numNets++
		cfg.activeNet = cnMainNet
	}
	if cfg.TestNet {
		numNets++
		cfg.activeNet = cnTestNet
	}
	if cfg.SimNet {
		numNets++
		cfg.activeNet = cnSimNet
	}
	if numNets > 1 {
		str := "%s: mainnet, testnet and simnet params can't be " +
			"used together -- choose one of the three"
		err := fmt.Errorf(str, funcName)
		fmt.Fprintln(os.Stderr, err)
		fmt.Fprintln(os.Stderr, usageMessage)
		return nil, nil, err
	}

	// Validate policy options.
	if cfg.OpenPolicy.MinChanSize < 0.0004 {
		return nil, nil, fmt.Errorf("openpolicy.minchansize cannot be smaller than 40000 atoms")
	}
	maxDcrlndChanSize := dcrutil.Amount((1 << 30) - 1).ToCoin()
	if cfg.OpenPolicy.MaxChanSize > maxDcrlndChanSize {
		return nil, nil, fmt.Errorf("openpolicy.maxchansize cannot be larger than %.8f",
			maxDcrlndChanSize)
	}
	if cfg.OpenPolicy.MaxNbChannels <= 0 {
		return nil, nil, fmt.Errorf("openpolicy.maxnbchannels cannot be zero")
	}
	if cfg.OpenPolicy.InvoiceFeeRate <= 0 {
		return nil, nil, fmt.Errorf("openpolicy.invoicefeerate cannot be zero")
	}
	if cfg.OpenPolicy.InvoiceExpiration < time.Second {
		return nil, nil, fmt.Errorf("openpolicy.invoiceexpiration cannot be smaller than 1 second")
	}
	if cfg.ClosePolicy.CheckInterval < time.Second {
		return nil, nil, fmt.Errorf("closepolicy.checkinterval cannot be smaller than 1 second")
	}
	blockTime := cfg.activeNet.chainParams().TargetTimePerBlock
	if cfg.ClosePolicy.MinChanLifetime < blockTime {
		return nil, nil, fmt.Errorf("closepolicy.minchanlifetime cannot be smaller than "+
			"the network target block time (%s)", blockTime)
	}
	if cfg.ClosePolicy.MinWalletBalance < 0 {
		return nil, nil, errors.New("closepolicy.minWalletbalance " +
			"cannot be smaller than zero")
	}

	// Verify TLS options.
	if !cfg.DisableTLS {
		if cfg.TLSKey == "" {
			return nil, nil, fmt.Errorf("tlskey cannot be empty when TLS is enabled")
		}
		if cfg.TLSCert == "" {
			return nil, nil, fmt.Errorf("tlscert cannot be empty when TLS is enabled")
		}
		cfg.TLSKey = cleanAndExpandPath(cfg.TLSKey)
		cfg.TLSCert = cleanAndExpandPath(cfg.TLSCert)
	}

	// Initialize log rotation.  After log rotation has been initialized,
	// the logger variables may be used.
	logDir := strings.Replace(defaultLogDir, string(defaultActiveNet),
		string(cfg.activeNet), 1)
	logPath := filepath.Join(logDir, appName+".log")
	initLogRotator(logPath)
	setLogLevels(defaultLogLevel)

	// Parse, validate, and set debug log level(s).
	if err := parseAndSetDebugLevels(cfg.DebugLevel); err != nil {
		err := fmt.Errorf("%s: %v", funcName, err.Error())
		fmt.Fprintln(os.Stderr, err)
		fmt.Fprintln(os.Stderr, usageMessage)
		return nil, nil, err
	}

	// Add the default listener if none were specified. The default
	// listener is all addresses on the listen port for the network we are
	// to connect to.
	if len(cfg.Listeners) == 0 {
		cfg.Listeners = []string{
			net.JoinHostPort("", cfg.activeNet.defaultListenPort()),
		}
	}

	// Validate format of profile, can be an address:port, or just a port.
	if cfg.Profile != "" {
		// If profile is just a number, then add a default host of
		// "127.0.0.1" such that Profile is a valid tcp address.
		if _, err := strconv.Atoi(cfg.Profile); err == nil {
			cfg.Profile = net.JoinHostPort("127.0.0.1", cfg.Profile)
		}

		// Check the Profile is a valid address.
		_, portStr, err := net.SplitHostPort(cfg.Profile)
		if err != nil {
			return nil, nil, fmt.Errorf("invalid profile host/port: %v", err)
		}

		// Finally, check the port is in range.
		if port, _ := strconv.Atoi(portStr); port < 1024 || port > 65535 {
			return nil, nil, fmt.Errorf("profile address %s: port "+
				"must be between 1024 and 65535", cfg.Profile)
		}
	}

	// Expand file paths.
	cfg.LNTLSCertPath = cleanAndExpandPath(cfg.LNTLSCertPath)
	cfg.LNMacaroonPath = cleanAndExpandPath(cfg.LNMacaroonPath)

	// Attempt an early connection to the dcrlnd server and verify if it's a
	// reasonable server for operations.
	err = server.CheckDcrlnd(cfg.LNRPCHost, cfg.LNTLSCertPath, cfg.LNMacaroonPath)
	if err != nil {
		return nil, nil, fmt.Errorf("error while checking underlying "+
			"dcrlnd node: %v", err)
	}

	// Warn about missing config file only after all other configuration is
	// done.  This prevents the warning on help messages and invalid
	// options.  Note this should go directly before the return.
	if configFileError != nil {
		log.Warnf("%v", configFileError)
	}

	return &cfg, remainingArgs, nil
}
