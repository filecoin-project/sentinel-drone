package lotus

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/influxdata/telegraf"
	"github.com/influxdata/telegraf/plugins/inputs"
	"github.com/influxdata/telegraf/plugins/inputs/lotus/rpc"

	"github.com/filecoin-project/lotus/api"
	"github.com/filecoin-project/lotus/chain/types"
	"github.com/filecoin-project/lotus/node/repo"
)

const (
	DefaultConfig_DataPath          = "${HOME}/.lotus"
	DefaultConfig_APIListenAddr     = "/ip4/0.0.0.0/tcp/1234"
	DefaultConfig_ChainWalkThrottle = "1s"
)

type lotus struct {
	Config_DataPath          string `toml:"lotus_datapath"`
	Config_APIListenAddr     string `toml:"lotus_api_multiaddr"`
	Config_APIToken          string `toml:"lotus_api_token"`
	Config_ChainWalkThrottle string `toml:"chain_walk_throttle"`

	Log telegraf.Logger

	api               api.FullNode
	chainWalkThrottle time.Duration
	shutdown          func()
}

func newLotus() *lotus {
	return &lotus{}
}

func (l *lotus) setDefaults() error {
	if len(l.Config_DataPath) == 0 {
		l.Config_DataPath = DefaultConfig_DataPath
	}
	if len(l.Config_APIListenAddr) == 0 {
		l.Config_APIListenAddr = DefaultConfig_APIListenAddr
	}
	if len(l.Config_ChainWalkThrottle) == 0 {
		l.Config_ChainWalkThrottle = DefaultConfig_ChainWalkThrottle
	}
	throttleDuration, err := time.ParseDuration(l.Config_ChainWalkThrottle)
	if err != nil {
		return err
	}
	l.chainWalkThrottle = throttleDuration
	return nil
}

// Description will appear directly above the plugin definition in the config file
func (l *lotus) Description() string {
	return `Capture network metrics from Filecoin Lotus`
}

var sampleConfig = fmt.Sprintf(`
	## Provide the multiaddr that the lotus API is listening on. If API multiaddr is empty,
	## telegraf will check lotus_datapath for values. Default: "%[1]s"
	# lotus_api_multiaddr = "%[1]s"

	## Provide the token that telegraf can use to access the lotus API. The token only requires
	## "read" privileges. This is useful for restricting metrics to a seperate token.
	# lotus_api_token = ""

	## Provide the Lotus working path. The input plugin will scan the path contents
	## for the correct listening address/port and API token. Note: Path must be readable
	## by the telegraf process otherwise, the value should be manually set in api_listen_addr
	## and api_token. This value is ignored if lotus_api_* values are populated.
	## Default: "%[2]s"
	lotus_datapath = "%[2]s"

	## Set the shortest duration between the processing of each tipset state. Valid time units
	## are "ns", "us" (or "µs"), "ms", "s", "m", "h". When telegraf starts collecting metrics,
	## it will also begin walking over the entire chain to update past state changes for the
	## dashboard. This setting prevents that walk from going too quickly. Default: "%[3]s"
	chain_walk_throttle = "%[3]s"
`, DefaultConfig_DataPath, DefaultConfig_APIListenAddr, DefaultConfig_ChainWalkThrottle)

// SampleConfig will populate the sample configuration portion of the plugin's configuration
func (l *lotus) SampleConfig() string {
	return sampleConfig
}

// Gather is performed in Start
func (l *lotus) Gather(acc telegraf.Accumulator) error {
	return nil
}

func (l *lotus) getAPIUsingLotusConfig(ctx context.Context) (api.FullNode, func(), error) {
	var (
		nodeAPI    api.FullNode
		nodeCloser func()
	)
	if len(l.Config_APIListenAddr) > 0 && len(l.Config_APIToken) > 0 {
		lotusAPI, apiCloser, err := rpc.GetFullNodeAPIUsingCredentials(ctx, l.Config_APIListenAddr, l.Config_APIToken)
		if err != nil {
			return nil, nil, err
		}
		nodeAPI = lotusAPI
		nodeCloser = apiCloser
	} else {
		lotusAPI, apiCloser, err := rpc.GetFullNodeAPI(ctx, l.Config_DataPath)
		if err != nil {
			return nil, nil, err
		}
		nodeAPI = lotusAPI
		nodeCloser = apiCloser
	}
	return nodeAPI, nodeCloser, nil
}

func (l *lotus) startWorkers(ctx context.Context, acc telegraf.Accumulator, chainHead *types.TipSet, die chan struct{}) error {
	tipsetsCh, err := rpc.GetTips(ctx, l.api, chainHead.Height(), 3) // closed with consumer
	if err != nil {
		return fmt.Errorf("getting tipset notify source: %v", err)
	}

	headCh, err := l.api.SyncIncomingBlocks(ctx) // closed with consumer
	if err != nil {
		return fmt.Errorf("getting blocks notify source: %v", err)
	}

	// process new tipsets
	go l.processTipSets(ctx, tipsetsCh, acc, die)

	// process new headers
	go l.processBlockHeaders(ctx, headCh, acc, die)

	return nil
}

func (l *lotus) run(ctx context.Context, acc telegraf.Accumulator, warnErrors chan error) error {
	var nodeAPI api.FullNode
	var nodeCloser func()
	var err error

	// wait for the node to come online
	nodeCheckPeriod := time.NewTicker(10 * time.Second)
	for range nodeCheckPeriod.C {
		nodeAPI, nodeCloser, err = l.getAPIUsingLotusConfig(ctx)
		if err != nil {
			if err == repo.ErrNoAPIEndpoint {
				l.Log.Warn("Api not online, retrying...")
				continue
			}
			warnErrors <- err
			continue
		}
		break
	}
	nodeCheckPeriod.Stop()

	// great we're online, lets get to work
	l.api = nodeAPI

	// Ensure the connection to the node is closed when we are done.
	defer nodeCloser()

	// record node info
	if err := l.recordLotusInfoPoints(ctx, acc); err != nil {
		return fmt.Errorf("Recording lotus info: %w", err)
	}

	l.Log.Debug("Recorded lotus info")

	// collect the current head
	chainHead, err := l.api.ChainHead(ctx)
	if err != nil {
		return fmt.Errorf("Getting latest chainhead: %w", err)
	}

	// This derived context allows us to notify workers to terminate when this function exits
	cctx, closeRpcChans := context.WithCancel(ctx)
	defer closeRpcChans()

	// A worker sends on workerDie to signal that it has failed
	workerDie := make(chan struct{}, 2)

	if err := l.startWorkers(cctx, acc, chainHead, workerDie); err != nil {
		return fmt.Errorf("Run workers: %v", err)
	}

	l.Log.Info("Service workers started")

	// Wait for context to be cancelled or a worker to die
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-workerDie:
		return fmt.Errorf("Service worker closed unexpectedly")
	}
}

// Start begins walking through the chain to update the datastore
func (l *lotus) Start(acc telegraf.Accumulator) error {
	if err := l.setDefaults(); err != nil {
		return err
	}

	var ctx context.Context
	ctx, l.shutdown = context.WithCancel(context.Background())

	warnErr := make(chan error) // closed with consumer

	// Goroutine to log errors emitted by workers
	go func(ctx context.Context, warnErrCh chan error) {
		defer close(warnErrCh)
		for {
			select {
			case <-ctx.Done():
				return
			case w := <-warnErrCh:
				l.Log.Warn(w.Error())
			}
		}
	}(ctx, warnErr)

	// Gather metrics until the context is cancelled, restarting if fatal error encountered
	go func(ctx context.Context, warnErrCh chan error, acc telegraf.Accumulator) {
		for {
			if err := l.run(ctx, acc, warnErrCh); err != nil {
				l.Log.Errorf("Service ended fatally: %v", err)
			}

			select {
			case <-ctx.Done():
				return
			default:
				continue
			}
		}
	}(ctx, warnErr, acc)

	return nil
}

func (l *lotus) Stop() {
	if l.shutdown != nil {
		l.shutdown()
	}
}

func (l *lotus) recordLotusInfoPoints(ctx context.Context, acc telegraf.Accumulator) error {
	nodePeerID, err := l.api.ID(context.Background())
	if err != nil {
		return err
	}

	v, err := l.api.Version(ctx)
	if err != nil {
		return err
	}
	versionTokens := strings.SplitN(v.Version, "+", 2)
	version := versionTokens[0]
	commit := versionTokens[1]

	network, err := l.api.StateNetworkName(ctx)
	if err != nil {
		return err
	}

	acc.AddFields("observed_info",
		map[string]interface{}{
			"nothing": 0,
		},
		map[string]string{
			"api_version": v.APIVersion.String(),
			"commit":      commit,
			"network":     string(network),
			"peer_id":     nodePeerID.String(),
			"version":     version,
		})

	return nil
}

func (l *lotus) processBlockHeaders(ctx context.Context, headCh <-chan *types.BlockHeader, acc telegraf.Accumulator, die chan struct{}) {
	for {
		select {
		case head, ok := <-headCh:
			if !ok {
				die <- struct{}{}
				return
			}
			go l.processHeader(ctx, acc, head, time.Now())
		case <-ctx.Done():
			return
		}
	}

}

func (l *lotus) processTipSets(ctx context.Context, tipsetsCh <-chan *types.TipSet, acc telegraf.Accumulator, die chan struct{}) {
	throttle := time.NewTicker(l.chainWalkThrottle)
	defer throttle.Stop()

	for range throttle.C {
		select {
		case t, ok := <-tipsetsCh:
			if !ok {
				die <- struct{}{}
				return
			}
			go l.processTipset(ctx, acc, t, time.Now())
		case <-ctx.Done():
			return
		}
	}
}

func (l *lotus) processTipset(ctx context.Context, acc telegraf.Accumulator, newTipSet *types.TipSet, receivedAt time.Time) {
	height := newTipSet.Height()

	if err := recordTipsetMessagesPoints(ctx, acc, newTipSet, receivedAt); err != nil {
		err = fmt.Errorf("Recording tipset (h: %d): %v", height, err)
		acc.AddError(err)
		l.Log.Warn(err.Error())
		return
	}
	l.Log.Debugf("Recorded tipset (h: %d)", height)
}

func (l *lotus) processHeader(ctx context.Context, acc telegraf.Accumulator, newHeader *types.BlockHeader, receivedAt time.Time) {
	err := recordBlockHeaderPoints(ctx, acc, newHeader, receivedAt)
	if err != nil {
		err = fmt.Errorf("Recording block header (h: %d): %v", newHeader.Height, err)
		acc.AddError(err)
		l.Log.Warn(err.Error())
		return
	}
	l.Log.Debugf("Recorded block header (h: %d)", newHeader.Height)
}

func recordBlockHeaderPoints(ctx context.Context, acc telegraf.Accumulator, newHeader *types.BlockHeader, receivedAt time.Time) error {
	var parentBlocks = ""
	for _, cid := range newHeader.Parents {
		parentBlocks += cid.String() + ","
	}
	acc.AddFields("observed_headers",
		map[string]interface{}{
			"tipset_height":    newHeader.Height,
			"header_timestamp": time.Unix(int64(newHeader.Timestamp), 0).Unix(),
		},
		map[string]string{
			"header_cid":        newHeader.Cid().String(),
			"miner_id":          newHeader.Miner.String(),
			"parent_state_root": newHeader.ParentStateRoot.String(),
			"parent_weight":     newHeader.ParentWeight.String(),
			"parents":           parentBlocks,
		},
		receivedAt)
	return nil
}

func recordTipsetMessagesPoints(ctx context.Context, acc telegraf.Accumulator, tipset *types.TipSet, receivedAt time.Time) error {
	ts := time.Unix(int64(tipset.MinTimestamp()), int64(0))
	cids := tipset.Cids()
	if len(cids) == 0 {
		return fmt.Errorf("no cids in tipset")
	}

	acc.AddFields("observed_tipsets",
		map[string]interface{}{
			"tipset_height": int(tipset.Height()),
			"block_count":   len(cids),
			"recorded_at":   receivedAt.UnixNano(),
		},
		map[string]string{
			"tipset_key": tipset.Key().String(),
		}, ts)

	return nil
}

func init() {
	var _ telegraf.ServiceInput = newLotus()
	inputs.Add("lotus", func() telegraf.Input {
		return newLotus()
	})
}
