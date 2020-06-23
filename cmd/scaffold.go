package cmd

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"math/rand"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	"github.com/dgraph-io/badger/v2"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/rs/zerolog"
	"github.com/spf13/pflag"

	"github.com/dapperlabs/flow-go/consensus/hotstuff/model"
	"github.com/dapperlabs/flow-go/crypto"
	"github.com/dapperlabs/flow-go/model/bootstrap"
	"github.com/dapperlabs/flow-go/model/dkg"
	"github.com/dapperlabs/flow-go/model/flow"
	"github.com/dapperlabs/flow-go/model/flow/filter"
	"github.com/dapperlabs/flow-go/module"
	"github.com/dapperlabs/flow-go/module/local"
	"github.com/dapperlabs/flow-go/module/metrics"
	"github.com/dapperlabs/flow-go/module/trace"
	jsoncodec "github.com/dapperlabs/flow-go/network/codec/json"
	"github.com/dapperlabs/flow-go/network/gossip/libp2p"
	"github.com/dapperlabs/flow-go/network/gossip/libp2p/validators"
	"github.com/dapperlabs/flow-go/state/dkg/wrapper"
	protocol "github.com/dapperlabs/flow-go/state/protocol/badger"
	"github.com/dapperlabs/flow-go/storage"
	storerr "github.com/dapperlabs/flow-go/storage"
	bstorage "github.com/dapperlabs/flow-go/storage/badger"
	"github.com/dapperlabs/flow-go/storage/badger/operation"
	sutil "github.com/dapperlabs/flow-go/storage/util"
	"github.com/dapperlabs/flow-go/utils/debug"
)

const notSet = "not set"

// BaseConfig is the general config for the FlowNodeBuilder
type BaseConfig struct {
	nodeIDHex        string
	bindAddr         string
	nodeRole         string
	timeout          time.Duration
	datadir          string
	level            string
	metricsPort      uint
	nClusters        uint
	BootstrapDir     string
	profilerEnabled  bool
	profilerDir      string
	profilerInterval time.Duration
	profilerDuration time.Duration
}

type Metrics struct {
	Network    module.NetworkMetrics
	Engine     module.EngineMetrics
	Compliance module.ComplianceMetrics
	Cache      module.CacheMetrics
	Mempool    module.MempoolMetrics
}

type Storage struct {
	Headers      storage.Headers
	Index        storage.Index
	Identities   storage.Identities
	Guarantees   storage.Guarantees
	Seals        storage.Seals
	Payloads     storage.Payloads
	Blocks       storage.Blocks
	Transactions storage.Transactions
	Collections  storage.Collections
}

type namedModuleFunc struct {
	fn   func(*FlowNodeBuilder) error
	name string
}

type namedComponentFunc struct {
	fn   func(*FlowNodeBuilder) (module.ReadyDoneAware, error)
	name string
}

type namedDoneObject struct {
	ob   module.ReadyDoneAware
	name string
}

type namedBootstrapFunc struct {
	fn   func(*protocol.State) error
	name string
}

// FlowNodeBuilder is the builder struct used for all flow nodes
// It runs a node process with following structure, in sequential order
// Base inits (network, storage, state, logger)
//   PostInit handlers, if any
//   Bootstrap handlers, if protocol state was bootstrapped
// Components handlers, if any, wait sequentially
// Run() <- main loop
// Components destructors, if any
type FlowNodeBuilder struct {
	BaseConfig        BaseConfig
	NodeID            flow.Identifier
	flags             *pflag.FlagSet
	Logger            zerolog.Logger
	Me                *local.Local
	Tracer            *trace.OpenTracer
	MetricsRegisterer prometheus.Registerer
	Metrics           Metrics
	DB                *badger.DB
	Storage           Storage
	State             *protocol.State
	DKGState          *wrapper.State
	Network           *libp2p.Network
	modules           []namedModuleFunc
	components        []namedComponentFunc
	doneObject        []namedDoneObject
	bootstraps        []namedBootstrapFunc
	sig               chan os.Signal
	postInitFns       []func(*FlowNodeBuilder)
	stakingKey        crypto.PrivateKey
	networkKey        crypto.PrivateKey
	MsgValidators     []validators.MessageValidator

	// root state information
	RootCommit           flow.StateCommitment // should be removed
	RootBlock            *flow.Block
	RootQC               *model.QuorumCertificate
	RootResult           *flow.ExecutionResult
	RootSeal             *flow.Seal
	RootAccountPublicKey *flow.AccountPublicKey
	RootTokenSupply      uint64
	RootChainID          flow.ChainID
}

func (fnb *FlowNodeBuilder) baseFlags() {
	homedir, _ := os.UserHomeDir()
	datadir := filepath.Join(homedir, ".flow", "database")
	// bind configuration parameters
	fnb.flags.StringVar(&fnb.BaseConfig.nodeIDHex, "nodeid", notSet, "identity of our node")
	fnb.flags.StringVar(&fnb.BaseConfig.bindAddr, "bind", notSet, "address to bind on")
	fnb.flags.StringVarP(&fnb.BaseConfig.BootstrapDir, "bootstrapdir", "b", "bootstrap", "path to the bootstrap directory")
	fnb.flags.DurationVarP(&fnb.BaseConfig.timeout, "timeout", "t", 1*time.Minute, "how long to try connecting to the network")
	fnb.flags.StringVarP(&fnb.BaseConfig.datadir, "datadir", "d", datadir, "directory to store the protocol state")
	fnb.flags.StringVarP(&fnb.BaseConfig.level, "loglevel", "l", "info", "level for logging output")
	fnb.flags.UintVarP(&fnb.BaseConfig.metricsPort, "metricport", "m", 8080, "port for /metrics endpoint")
	fnb.flags.UintVar(&fnb.BaseConfig.nClusters, "nclusters", 2, "number of collection node clusters")
	fnb.flags.BoolVar(&fnb.BaseConfig.profilerEnabled, "profiler-enabled", false, "whether to enable the auto-profiler")
	fnb.flags.StringVar(&fnb.BaseConfig.profilerDir, "profiler-dir", "profiler", "directory to create auto-profiler profiles")
	fnb.flags.DurationVar(&fnb.BaseConfig.profilerInterval, "profiler-interval", 15*time.Minute, "the interval between auto-profiler runs")
	fnb.flags.DurationVar(&fnb.BaseConfig.profilerDuration, "profiler-duration", 10*time.Second, "the duration to run the auto-profile for")
}

func (fnb *FlowNodeBuilder) enqueueNetworkInit() {
	fnb.Component("network", func(builder *FlowNodeBuilder) (module.ReadyDoneAware, error) {

		codec := jsoncodec.NewCodec()

		myAddr := fnb.Me.Address()
		if fnb.BaseConfig.bindAddr != notSet {
			myAddr = fnb.BaseConfig.bindAddr
		}

		mw, err := libp2p.NewMiddleware(fnb.Logger.Level(zerolog.ErrorLevel), codec, myAddr, fnb.Me.NodeID(),
			fnb.networkKey, fnb.Metrics.Network, libp2p.DefaultMaxPubSubMsgSize, fnb.GenesisBlock.ID().String(), fnb.MsgValidators...)
		if err != nil {
			return nil, fmt.Errorf("could not initialize middleware: %w", err)
		}

		participants, err := fnb.State.Final().Identities(filter.Any)
		if err != nil {
			return nil, fmt.Errorf("could not get network identities: %w", err)
		}

		nodeID, err := fnb.State.Final().Identity(fnb.Me.NodeID())
		if err != nil {
			return nil, fmt.Errorf("could not get node id: %w", err)
		}
		nodeRole := nodeID.Role
		topology := libp2p.NewRandPermTopology(nodeRole)

		net, err := libp2p.NewNetwork(fnb.Logger, codec, participants, fnb.Me, mw, 10e6, topology, fnb.Metrics.Network)
		if err != nil {
			return nil, fmt.Errorf("could not initialize network: %w", err)
		}

		fnb.Network = net
		return net, err
	})
}

func (fnb *FlowNodeBuilder) enqueueMetricsServerInit() {
	fnb.Component("metrics server", func(builder *FlowNodeBuilder) (module.ReadyDoneAware, error) {
		server := metrics.NewServer(fnb.Logger, fnb.BaseConfig.metricsPort, fnb.BaseConfig.profilerEnabled)
		return server, nil
	})
}

func (fnb *FlowNodeBuilder) registerBadgerMetrics() {
	metrics.RegisterBadgerMetrics()
}

func (fnb *FlowNodeBuilder) enqueueTracer() {
	fnb.Component("tracer", func(builder *FlowNodeBuilder) (module.ReadyDoneAware, error) {
		return fnb.Tracer, nil
	})
}

func (fnb *FlowNodeBuilder) initNodeInfo() {
	if fnb.BaseConfig.nodeIDHex == notSet {
		fnb.Logger.Fatal().Msg("cannot start without node ID")
	}

	nodeID, err := flow.HexStringToIdentifier(fnb.BaseConfig.nodeIDHex)
	if err != nil {
		fnb.Logger.Fatal().Err(err).Msg("could not parse hex ID")
	}

	info, err := loadPrivateNodeInfo(fnb.BaseConfig.BootstrapDir, nodeID)
	if err != nil {
		fnb.Logger.Fatal().Err(err).Msg("failed to load private node info")
	}

	fnb.NodeID = nodeID
	fnb.stakingKey = info.StakingPrivKey.PrivateKey
	fnb.networkKey = info.NetworkPrivKey.PrivateKey
}

func (fnb *FlowNodeBuilder) initLogger() {
	// configure logger with standard level, node ID and UTC timestamp
	zerolog.TimestampFunc = func() time.Time { return time.Now().UTC() }
	log := fnb.Logger.With().
		Timestamp().
		Str("node_role", fnb.BaseConfig.nodeRole).
		Str("node_id", fnb.BaseConfig.nodeIDHex).
		Logger()

	log.Info().Msgf("flow %s node starting up", fnb.BaseConfig.nodeRole)

	// parse config log level and apply to logger
	lvl, err := zerolog.ParseLevel(strings.ToLower(fnb.BaseConfig.level))
	if err != nil {
		log.Fatal().Err(err).Msg("invalid log level")
	}
	log = log.Level(lvl)

	fnb.Logger = log
}

func (fnb *FlowNodeBuilder) initMetrics() {
	tracer, err := trace.NewTracer(fnb.Logger, fnb.BaseConfig.nodeRole)
	fnb.MustNot(err).Msg("could not initialize tracer")
	fnb.MetricsRegisterer = prometheus.DefaultRegisterer
	fnb.Tracer = tracer

	mempools := metrics.NewMempoolCollector(5 * time.Second)

	fnb.Metrics = Metrics{
		Network:    metrics.NewNetworkCollector(),
		Engine:     metrics.NewEngineCollector(),
		Compliance: metrics.NewComplianceCollector(),
		Cache:      metrics.NewCacheCollector(fnb.RootChainID),
		Mempool:    mempools,
	}

	// registers mempools as a Component so that its Ready method is invoked upon startup
	fnb.Component("mempools metrics", func(builder *FlowNodeBuilder) (module.ReadyDoneAware, error) {
		return mempools, nil
	})
}

func (fnb *FlowNodeBuilder) initProfiler() {
	if !fnb.BaseConfig.profilerEnabled {
		return
	}
	profiler, err := debug.NewAutoProfiler(
		fnb.Logger,
		fnb.BaseConfig.profilerDir,
		fnb.BaseConfig.profilerInterval,
		fnb.BaseConfig.profilerDuration,
	)
	fnb.MustNot(err).Msg("could not initialize profiler")
	fnb.Component("profiler", func(node *FlowNodeBuilder) (module.ReadyDoneAware, error) {
		return profiler, nil
	})
}

func (fnb *FlowNodeBuilder) initDB() {
	// Pre-create DB path (Badger creates only one-level dirs)
	err := os.MkdirAll(fnb.BaseConfig.datadir, 0700)
	fnb.MustNot(err).Str("dir", fnb.BaseConfig.datadir).Msg("could not create datadir")

	log := sutil.NewLogger(fnb.Logger)

	// we initialize the database with options that allow us to keep the maximum
	// item size in the trie itself (up to 1MB) and where we keep all level zero
	// tables in-memory as well; this slows down compaction and increases memory
	// usage, but it improves overall performance and disk i/o
	opts := badger.
		DefaultOptions(fnb.BaseConfig.datadir).
		WithKeepL0InMemory(true).
		WithLogger(log).
		WithValueLogFileSize(128 << 20). // Default is 1 GB
		WithValueLogMaxEntries(100000)   // Default is 1000000

	db, err := badger.Open(opts)
	fnb.MustNot(err).Msg("could not open key-value store")
	fnb.DB = db
}

func (fnb *FlowNodeBuilder) initStorage() {

	// in order to void long iterations with big keys when initializing with an
	// already populated database, we bootstrap the initial maximum key size
	// upon starting
	err := operation.RetryOnConflict(fnb.DB.Update, func(tx *badger.Txn) error {
		return operation.InitMax(tx)
	})
	fnb.MustNot(err).Msg("could not initialize max tracker")

	headers := bstorage.NewHeaders(fnb.Metrics.Cache, fnb.DB)
	identities := bstorage.NewIdentities(fnb.Metrics.Cache, fnb.DB)
	guarantees := bstorage.NewGuarantees(fnb.Metrics.Cache, fnb.DB)
	seals := bstorage.NewSeals(fnb.Metrics.Cache, fnb.DB)
	index := bstorage.NewIndex(fnb.Metrics.Cache, fnb.DB)
	payloads := bstorage.NewPayloads(fnb.DB, index, identities, guarantees, seals)
	blocks := bstorage.NewBlocks(fnb.DB, headers, payloads)
	transactions := bstorage.NewTransactions(fnb.Metrics.Cache, fnb.DB)
	collections := bstorage.NewCollections(fnb.DB, transactions)

	fnb.Storage = Storage{
		Headers:      headers,
		Identities:   identities,
		Guarantees:   guarantees,
		Seals:        seals,
		Index:        index,
		Payloads:     payloads,
		Blocks:       blocks,
		Transactions: transactions,
		Collections:  collections,
	}
}

func (fnb *FlowNodeBuilder) initState() {

	state, err := protocol.NewState(
		fnb.Metrics.Compliance,
		fnb.DB,
		fnb.Storage.Headers,
		fnb.Storage.Identities,
		fnb.Storage.Seals,
		fnb.Storage.Index,
		fnb.Storage.Payloads,
		fnb.Storage.Blocks,
		protocol.SetClusters(fnb.BaseConfig.nClusters),
	)

	fnb.MustNot(err).Msg("could not initialize flow state")

	// check if database is initialized
	_, err = state.Final().Head()
	if errors.Is(err, storerr.ErrNotFound) {
		// Bootstrap!

		fnb.Logger.Info().Msg("bootstrapping empty protocol state")

		// load the root block from bootstrap files and set the chain ID based on it
		fnb.RootBlock, err = loadRootBlock(fnb.BaseConfig.BootstrapDir)
		fnb.MustNot(err).Msg("could not load root block")

		// set the root chain ID based on the root block
		fnb.RootChainID = fnb.RootBlock.Header.ChainID

		// load the root QC data from bootstrap files
		fnb.RootQC, err = loadRootQC(fnb.BaseConfig.BootstrapDir)
		fnb.MustNot(err).Msg("could not load root QC")

		// load the root execution result from bootstrap files
		rootResult, err := loadRootResult(fnb.BaseConfig.BootstrapDir)
		fnb.MustNot(err).Msg("could not load root execution result")

		// load the root block seal from bootstrap files
		rootSeal, err := loadRootSeal(fnb.BaseConfig.BootstrapDir)
		fnb.MustNot(err).Msg("could not load root seal")

		// bootstrap the protocol state with the loaded data
		err = state.Mutate().Bootstrap(fnb.RootBlock, rootResult, rootSeal)
		fnb.MustNot(err).Msg("could not bootstrap protocol state")

		// apply the bootstrap functions to the protocol state
		for _, b := range fnb.bootstraps {
			err := b.fn(state)
			fnb.MustNot(err).Str("name", b.name).Msg("could not apply bootstrap function")
		}

		// load the DKG public data from bootstrap files
		dkgPubData, err := loadDKGPublicData(fnb.BaseConfig.BootstrapDir)
		fnb.MustNot(err).Msg("could not load public DKG data")

		// bootstrap the DKG state with the loaded data
		fnb.DKGState = wrapper.NewState(dkgPubData)

	} else if err != nil {
		fnb.Logger.Fatal().Err(err).Msg("could not check existing database")
	} else {

		// TODO: we shouldn't have to load any files again after bootstrapping; in
		// order to make it unnecessary, we need to changes:
		// 1) persist the root QC along the root block so it can be loaded from DB
		// => https://github.com/dapperlabs/flow-go/issues/4166
		// 2) bootstrap and persist DKG state in a similar fashion to protocol state
		// => https://github.com/dapperlabs/flow-go/issues/4165

		// load the root block from bootstrap files and set the chain ID based on it
		fnb.RootBlock, err = loadRootBlock(fnb.BaseConfig.BootstrapDir)
		fnb.MustNot(err).Msg("could not load root block")

		// set the chain ID based on the root header
		// TODO: as the root header can now be loaded from protocol state, we should
		// not use a global variable for chain ID anymore, but rely on the protocol
		// state as final authority on what the chain ID is
		// => https://github.com/dapperlabs/flow-go/issues/4167
		fnb.RootChainID = fnb.RootBlock.Header.ChainID

		// load the root QC data from bootstrap files
		fnb.RootQC, err = loadRootQC(fnb.BaseConfig.BootstrapDir)
		fnb.MustNot(err).Msg("could not load root QC")

		// load the DKG public data from bootstrap files
		dkgPubData, err := loadDKGPublicData(fnb.BaseConfig.BootstrapDir)
		fnb.MustNot(err).Msg("could not load public DKG data")

		// bootstrap the DKG state with the loaded data
		fnb.DKGState = wrapper.NewState(dkgPubData)
	}

	myID, err := flow.HexStringToIdentifier(fnb.BaseConfig.nodeIDHex)
	fnb.MustNot(err).Msg("could not parse node identifier")

	self, err := state.Final().Identity(myID)
	fnb.MustNot(err).Msg("could not get identity")

	// ensure that the configured staking/network keys are consistent with the protocol state
	if !self.NetworkPubKey.Equals(fnb.networkKey.PublicKey()) {
		fnb.Logger.Fatal().Msg("configured networking key does not match protocol state")
	}
	if !self.StakingPubKey.Equals(fnb.stakingKey.PublicKey()) {
		fnb.Logger.Fatal().Msg("configured staking key does not match protocol state")
	}

	fnb.Me, err = local.New(self, fnb.stakingKey)
	fnb.MustNot(err).Msg("could not initialize local")

	fnb.State = state
}

func (fnb *FlowNodeBuilder) handleModule(v namedModuleFunc) {
	err := v.fn(fnb)
	if err != nil {
		fnb.Logger.Fatal().Err(err).Str("module", v.name).Msg("module initialization failed")
	} else {
		fnb.Logger.Info().Str("module", v.name).Msg("module initialization complete")
	}
}

func (fnb *FlowNodeBuilder) handleComponent(v namedComponentFunc) {

	log := fnb.Logger.With().Str("component", v.name).Logger()

	readyAware, err := v.fn(fnb)
	if err != nil {
		log.Fatal().Err(err).Msg("component initialization failed")
	} else {
		log.Info().Msg("component initialization complete")
	}

	select {
	case <-readyAware.Ready():
		log.Info().Msg("component startup complete")
	case <-time.After(fnb.BaseConfig.timeout):
		log.Fatal().Msg("component startup timed out")
	case <-fnb.sig:
		log.Warn().Msg("component startup aborted")
		os.Exit(1)
	}

	fnb.doneObject = append(fnb.doneObject, namedDoneObject{
		readyAware, v.name,
	})
}

func (fnb *FlowNodeBuilder) handleDoneObject(v namedDoneObject) {

	log := fnb.Logger.With().Str("component", v.name).Logger()

	select {
	case <-v.ob.Done():
		log.Info().Msg("component shutdown complete")
	case <-time.After(fnb.BaseConfig.timeout):
		log.Fatal().Msg("component shutdown timed out")
	case <-fnb.sig:
		log.Warn().Msg("component shutdown aborted")
		os.Exit(1)
	}
}

// ExtraFlags enables binding additional flags beyond those defined in BaseConfig.
func (fnb *FlowNodeBuilder) ExtraFlags(f func(*pflag.FlagSet)) *FlowNodeBuilder {
	f(fnb.flags)
	return fnb
}

// Module enables setting up dependencies of the engine with the builder context.
func (fnb *FlowNodeBuilder) Module(name string, f func(builder *FlowNodeBuilder) error) *FlowNodeBuilder {
	fnb.modules = append(fnb.modules, namedModuleFunc{
		fn:   f,
		name: name,
	})
	return fnb
}

// MustNot asserts that the given error must not occur.
//
// If the error is nil, returns a nil log event (which acts as a no-op).
// If the error is not nil, returns a fatal log event containing the error.
func (fnb *FlowNodeBuilder) MustNot(err error) *zerolog.Event {
	if err != nil {
		return fnb.Logger.Fatal().Err(err)
	}
	return nil
}

// Component adds a new component to the node that conforms to the ReadyDone
// interface.
//
// When the node is run, this component will be started with `Ready`. When the
// node is stopped, we will wait for the component to exit gracefully with
// `Done`.
func (fnb *FlowNodeBuilder) Component(name string, f func(*FlowNodeBuilder) (module.ReadyDoneAware, error)) *FlowNodeBuilder {
	fnb.components = append(fnb.components, namedComponentFunc{
		fn:   f,
		name: name,
	})

	return fnb
}

func (fnb *FlowNodeBuilder) PostInit(f func(node *FlowNodeBuilder)) *FlowNodeBuilder {
	fnb.postInitFns = append(fnb.postInitFns, f)
	return fnb
}

// FlowNode creates a new Flow node builder with the given name.
func FlowNode(role string) *FlowNodeBuilder {

	builder := &FlowNodeBuilder{
		BaseConfig: BaseConfig{
			nodeRole: role,
		},
		Logger: zerolog.New(os.Stderr),
		flags:  pflag.CommandLine,
	}

	builder.baseFlags()

	builder.enqueueNetworkInit()

	builder.enqueueMetricsServerInit()

	builder.registerBadgerMetrics()

	builder.enqueueTracer()

	return builder
}

// Run initiates all common components (logger, database, protocol state etc.)
// then starts each component. It also sets up a channel to gracefully shut
// down each component if a SIGINT is received.
func (fnb *FlowNodeBuilder) Run() {

	// initialize signal catcher
	fnb.sig = make(chan os.Signal, 1)
	signal.Notify(fnb.sig, os.Interrupt, syscall.SIGTERM)

	// parse configuration parameters
	pflag.Parse()

	// seed random generator
	rand.Seed(time.Now().UnixNano())

	fnb.initNodeInfo()

	fnb.initLogger()

	fnb.initProfiler()

	fnb.initDB()

	fnb.initMetrics()

	fnb.initStorage()

	fnb.initState()

	for _, f := range fnb.postInitFns {
		fnb.handlePostInit(f)
	}

	// set up all modules
	for _, f := range fnb.modules {
		fnb.handleModule(f)
	}

	// initialize all components
	for _, f := range fnb.components {
		fnb.handleComponent(f)
	}

	fnb.Logger.Info().Msgf("%s node startup complete", fnb.BaseConfig.nodeRole)

	<-fnb.sig

	fnb.Logger.Info().Msgf("%s node shutting down", fnb.BaseConfig.nodeRole)

	for i := len(fnb.doneObject) - 1; i >= 0; i-- {
		doneObject := fnb.doneObject[i]

		fnb.handleDoneObject(doneObject)
	}

	fnb.closeDatabase()

	fnb.Logger.Info().Msgf("%s node shutdown complete", fnb.BaseConfig.nodeRole)

	os.Exit(0)
}

func (fnb *FlowNodeBuilder) handlePostInit(f func(node *FlowNodeBuilder)) {
	f(fnb)
}

func (fnb *FlowNodeBuilder) closeDatabase() {
	err := fnb.DB.Close()
	if err != nil {
		fnb.Logger.Error().
			Err(err).
			Msg("could not close database")
	}
}

func loadRootBlock(dir string) (*flow.Block, error) {
	data, err := ioutil.ReadFile(filepath.Join(dir, bootstrap.PathRootBlock))
	if err != nil {
		return nil, err
	}
	var block flow.Block
	err = json.Unmarshal(data, &block)
	return &block, err

}

func loadRootQC(dir string) (*model.QuorumCertificate, error) {
	data, err := ioutil.ReadFile(filepath.Join(dir, bootstrap.PathRootQC))
	if err != nil {
		return nil, err
	}
	var qc model.QuorumCertificate
	err = json.Unmarshal(data, &qc)
	return &qc, err
}

func loadRootResult(dir string) (*flow.ExecutionResult, error) {
	data, err := ioutil.ReadFile(filepath.Join(dir, bootstrap.PathRootResult))
	if err != nil {
		return nil, err
	}
	var result flow.ExecutionResult
	err = json.Unmarshal(data, &result)
	return &result, err
}

func loadRootSeal(dir string) (*flow.Seal, error) {
	data, err := ioutil.ReadFile(filepath.Join(dir, bootstrap.PathRootSeal))
	if err != nil {
		return nil, err
	}
	var seal flow.Seal
	err = json.Unmarshal(data, &seal)
	return &seal, err
}

func loadDKGPublicData(dir string) (*dkg.PublicData, error) {
	data, err := ioutil.ReadFile(filepath.Join(dir, bootstrap.PathDKGDataPub))
	if err != nil {
		return nil, err
	}
	dkgPubData := &bootstrap.EncodableDKGDataPub{}
	err = json.Unmarshal(data, dkgPubData)
	return dkgPubData.ForHotStuff(), err
}

// Loads the private info for this node from disk (eg. private staking/network keys).
func loadPrivateNodeInfo(dir string, myID flow.Identifier) (*bootstrap.NodeInfoPriv, error) {
	data, err := ioutil.ReadFile(filepath.Join(dir, fmt.Sprintf(bootstrap.PathNodeInfoPriv, myID)))
	if err != nil {
		return nil, err
	}
	var info bootstrap.NodeInfoPriv
	err = json.Unmarshal(data, &info)
	return &info, err
}
