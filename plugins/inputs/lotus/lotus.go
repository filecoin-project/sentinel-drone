package lotus

import (
	"context"
	"fmt"
	"strconv"
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

const (
	MpoolBootstrap = -1
	MpoolAdd       = api.MpoolAdd
	MpoolRemove    = api.MpoolRemove
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
		lotusAPI, apiCloser, err := rpc.GetFullNodeAPIUsingCredentials(l.Config_APIListenAddr, l.Config_APIToken)
		if err != nil {
			return nil, nil, err
		}
		nodeAPI = lotusAPI
		nodeCloser = apiCloser
	} else {
		lotusAPI, apiCloser, err := rpc.GetFullNodeAPI(l.Config_DataPath)
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

	msgCh, err := l.api.MpoolSub(ctx) // closed with consumer
	if err != nil {
		return fmt.Errorf("getting mpool notify source: %v", err)
	}
	// process new tipsets
	go l.processTipSets(ctx, tipsetsCh, acc, die)

	// process new headers
	go l.processBlockHeaders(ctx, headCh, acc, die)

	// process new messages
	go l.processMpoolUpdates(ctx, msgCh, acc, die)

	return nil
}

func (l *lotus) run(ctx context.Context, acc telegraf.Accumulator, warnErrors chan error, workerDie chan struct{}) error {
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

	// record node info
	if err := l.recordLotusInfoPoints(ctx, acc); err != nil {
		warnErrors <- fmt.Errorf("Recording lotus info: %v", err)
		nodeCloser()
	} else {
		l.Log.Debug("Recorded lotus info")
	}

	// collect the current head
	chainHead, err := l.api.ChainHead(ctx)
	if err != nil {
		nodeCloser()
		return fmt.Errorf("Getting latest chainhead: %v", err)
	}

	// record info about pending messages
	if err := l.recordMpoolPendingPoints(ctx, acc, chainHead, time.Now()); err != nil {
		nodeCloser()
		return fmt.Errorf("Recording pending mpool msgs: %v", err)
	}
	l.Log.Debug("Recorded pending mpool messages")

	cctx, closeRpcChans := context.WithCancel(ctx)
	if err := l.startWorkers(cctx, acc, chainHead, workerDie); err != nil {
		nodeCloser()
		closeRpcChans()
		return fmt.Errorf("Run workers: %v", err)
	}

	l.Log.Info("Service workers started")

	select {
	case <-ctx.Done():
		nodeCloser()
		closeRpcChans()
		return ctx.Err()
	case <-workerDie:
		nodeCloser()
		closeRpcChans()
		return fmt.Errorf("Service worker datasource closed unexpectedly")
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

	go func() {
		for {
			// workerDie can recieve from all workers, ensure
			// channel buffers all recieves and doesn't block
			workerDie := make(chan struct{}, 3)
			defer func() {
				// drain and close channel
				for {
					select {
					case <-workerDie:
					default:
						close(workerDie)
					}
				}
			}()
			if err := l.run(ctx, acc, warnErr, workerDie); err != nil {
				l.Log.Errorf("Service ended fatally: %v", err)
			}

			select {
			case <-ctx.Done():
				return
			default:
				continue
			}
		}
	}()

	return nil
}

func (l *lotus) Stop() {
	l.shutdown()
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

	acc.AddFields("lotus_info",
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

func (l *lotus) recordMpoolPendingPoints(ctx context.Context, acc telegraf.Accumulator, chainHead *types.TipSet, receivedAt time.Time) error {
	pendingMsgs, err := l.api.MpoolPending(ctx, chainHead.Key())
	if err != nil {
		return err
	}

	for _, m := range pendingMsgs {
		if err := recordMpoolUpdatePoints(ctx, acc, api.MpoolUpdate{MpoolBootstrap, m}, receivedAt); err != nil {
			return err
		}
	}
	return nil
}

func (l *lotus) processMpoolUpdates(ctx context.Context, msgCh <-chan api.MpoolUpdate, acc telegraf.Accumulator, die chan struct{}) {
	for {
		select {
		case mpu, ok := <-msgCh:
			if !ok {
				die <- struct{}{}
				return
			}
			go l.processMpoolUpdate(ctx, acc, mpu, time.Now())
		case <-ctx.Done():
			return
		}
	}
}

func (l *lotus) processBlockHeaders(ctx context.Context, headCh <-chan *types.BlockHeader, acc telegraf.Accumulator, die chan struct{}) {
	// TODO maybe a waitgroup?
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

func (l *lotus) processMpoolUpdate(ctx context.Context, acc telegraf.Accumulator, newMpoolUpdate api.MpoolUpdate, receivedAt time.Time) {
	if err := recordMpoolUpdatePoints(ctx, acc, newMpoolUpdate, receivedAt); err != nil {
		err = fmt.Errorf("Recording mpool update (cid: %s, type: %s): %v", newMpoolUpdate.Message.Cid().String(), newMpoolUpdate.Type, err)
		acc.AddError(err)
		l.Log.Warn(err.Error())
		return
	}
	l.Log.Debugf("Recorded mpool update (cid: %s, type: %s)", newMpoolUpdate.Message.Cid().String(), newMpoolUpdate.Type)
}

func recordBlockHeaderPoints(ctx context.Context, acc telegraf.Accumulator, newHeader *types.BlockHeader, receivedAt time.Time) error {
	bs, err := newHeader.Serialize()
	if err != nil {
		return err
	}
	acc.AddFields("chain_block",
		map[string]interface{}{
			"tipset_height":    newHeader.Height,
			"election":         1,
			"header_size":      len(bs),
			"header_timestamp": time.Unix(int64(newHeader.Timestamp), 0).UnixNano(),
			"recorded_at":      receivedAt.UnixNano(),
		},
		map[string]string{
			"header_cid_tag":    newHeader.Cid().String(),
			"tipset_height_tag": strconv.Itoa(int(newHeader.Height)),
			"miner_tag":         newHeader.Miner.String(),
		},
		receivedAt)
	return nil
}

func recordMpoolUpdatePoints(ctx context.Context, acc telegraf.Accumulator, newMpoolUpdate api.MpoolUpdate, receivedAt time.Time) error {
	var updateType string
	switch newMpoolUpdate.Type {
	case MpoolAdd:
		updateType = "add"
	case MpoolRemove:
		updateType = "rm"
	case MpoolBootstrap:
		updateType = "init"
	default:
		return fmt.Errorf("unknown mpool update type: %v", newMpoolUpdate.Type)
	}

	acc.AddFields("chain_mpool",
		map[string]interface{}{
			"message_size": newMpoolUpdate.Message.Size(),
			"recorded_at":  receivedAt.UnixNano(),
		},
		map[string]string{
			"message_cid_tag":       newMpoolUpdate.Message.Cid().String(),
			"mpool_update_type_tag": updateType,
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

	acc.AddFields("chain_tipset",
		map[string]interface{}{
			"recorded_at":   receivedAt.UnixNano(),
			"tipset_height": int(tipset.Height()),
			"block_count":   len(cids),
		},
		map[string]string{}, ts)

	return nil
}

func init() {
	var _ telegraf.ServiceInput = newLotus()
	inputs.Add("lotus", func() telegraf.Input {
		return newLotus()
	})
}
