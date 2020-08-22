package lotus

import (
	"context"
	"fmt"
	"log"
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

func (l *lotus) runWorkers(ctx context.Context, acc telegraf.Accumulator, chainHead *types.TipSet, die chan struct{}) error {
	tipsetsCh, err := rpc.GetTips(ctx, l.api, chainHead.Height(), 3)
	if err != nil {
		return err
	}

	headCh, err := l.api.SyncIncomingBlocks(ctx)
	if err != nil {
		return err
	}

	msgCh, err := l.api.MpoolSub(ctx)
	if err != nil {
		return err
	}
	// process new tipsets
	go l.processTipSets(ctx, tipsetsCh, acc, die)

	// process new headers
	go l.processBlockHeaders(ctx, headCh, acc, die)

	// process new messages
	go l.processMpoolUpdates(ctx, msgCh, acc, die)

	log.Println("!D started workers")
	return nil
}

func (l *lotus) run(ctx context.Context, acc telegraf.Accumulator, fatalErrors chan error, workerDie chan struct{}) error {
	var nodeAPI api.FullNode
	var nodeCloser func()
	var err error

	// wait for the node to come online
	for range time.Tick(2 * time.Second) {
		nodeAPI, nodeCloser, err = l.getAPIUsingLotusConfig(ctx)
		if err != nil {
			if err == repo.ErrNoAPIEndpoint {
				log.Println("W! Api not online retrying...")
				continue
			}
			fatalErrors <- err
			continue
		}
		break
	}
	// great were online, lets get to work
	l.api = nodeAPI

	// collect the current head
	chainHead, err := l.api.ChainHead(ctx)
	if err != nil {
		fatalErrors <- err
		nodeCloser()
	}

	// record node info
	if err := l.recordLotusInfoPoints(ctx, acc); err != nil {
		fatalErrors <- err
		nodeCloser()
	}
	log.Println("!D recorded lotus info")

	// record info about pending messages
	if err := l.recordMpoolPendingPoints(ctx, acc, chainHead, time.Now()); err != nil {
		fatalErrors <- err
		nodeCloser()
	}
	log.Println("!D recorded lotus pending message info")

	cctx, closeRpcChans := context.WithCancel(ctx)
	if err := l.runWorkers(cctx, acc, chainHead, workerDie); err != nil {
		fatalErrors <- err
		nodeCloser()
		closeRpcChans()
	}

	log.Println("Waiting for a worker to die or for context cancellation")
	select {
	case <-ctx.Done():
		nodeCloser()
		closeRpcChans()
		log.Println("Run context canceled")
		return ctx.Err()
	case <-workerDie:
		nodeCloser()
		closeRpcChans()
		return fmt.Errorf("A worker died")
	}

}

// Start begins walking through the chain to update the datastore
func (l *lotus) Start(acc telegraf.Accumulator) error {
	if err := l.setDefaults(); err != nil {
		return err
	}

	var ctx context.Context
	ctx, l.shutdown = context.WithCancel(context.Background())

	fatalErrors := make(chan error)
	go func(ctx context.Context, errChan chan error) {
		for {
			select {
			case <-ctx.Done():
				return
			case err := <-fatalErrors:
				log.Println("!E Fatal error received: ", err.Error())
			}
		}
	}(ctx, fatalErrors)

	go func() {
		for {
			workerDie := make(chan struct{})
			if err := l.run(ctx, acc, fatalErrors, workerDie); err != nil {
				log.Println("!E Stopped running", "error", err.Error())
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
		map[string]interface{}{},
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
			go processMpoolUpdate(ctx, acc, mpu, time.Now())
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
			go processHeader(ctx, acc, head, time.Now())
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
			go processTipset(ctx, acc, t, time.Now())
		case <-ctx.Done():
			return
		}
	}
}

func processTipset(ctx context.Context, acc telegraf.Accumulator, newTipSet *types.TipSet, receivedAt time.Time) {
	height := newTipSet.Height()

	if err := recordTipsetMessagesPoints(ctx, acc, newTipSet, receivedAt); err != nil {
		log.Println("W! Failed to record messages", "height", height, "error", err)
		acc.AddError(fmt.Errorf("recording messages from tipset (@%d): %v", height, err))
		return
	}

	log.Println("I! Processed tipset height", "height", height)
}

func processHeader(ctx context.Context, acc telegraf.Accumulator, newHeader *types.BlockHeader, receivedAt time.Time) {
	err := recordBlockHeaderPoints(ctx, acc, newHeader, receivedAt)
	if err != nil {
		log.Println("W! Failed to record block header", "height", newHeader.Height, "error", err)
		acc.AddError(fmt.Errorf("recording block header (@%d cid: %s): %v", newHeader.Height, err))
		return
	}
	log.Println(" I! Processed block header", "height", newHeader.Height)
}

func processMpoolUpdate(ctx context.Context, acc telegraf.Accumulator, newMpoolUpdate api.MpoolUpdate, receivedAt time.Time) {
	if err := recordMpoolUpdatePoints(ctx, acc, newMpoolUpdate, receivedAt); err != nil {
		log.Println("W! Failed to record mpool update", "msg", newMpoolUpdate.Message.Cid().String(), "type", newMpoolUpdate.Type, "error", err.Error())
		acc.AddError(fmt.Errorf("recording mpool update (type: %d, cid: %s): %v", newMpoolUpdate.Type, newMpoolUpdate.Message.Cid(), err))
		return
	}
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
