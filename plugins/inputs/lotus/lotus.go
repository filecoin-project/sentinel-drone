package lotus

import (
	"context"
	"fmt"
	"log"
	"math/big"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/ipfs/go-cid"
	"github.com/multiformats/go-multihash"

	"github.com/influxdata/telegraf"
	"github.com/influxdata/telegraf/plugins/inputs"
	"github.com/influxdata/telegraf/plugins/inputs/lotus/rpc"

	"github.com/filecoin-project/lotus/api"
	"github.com/filecoin-project/lotus/build"
	"github.com/filecoin-project/lotus/chain/types"
	"github.com/filecoin-project/specs-actors/actors/builtin"
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

func (l *lotus) setDefaults() {
	if len(l.Config_DataPath) == 0 {
		l.Config_DataPath = DefaultConfig_DataPath
	}
	if len(l.Config_APIListenAddr) == 0 {
		l.Config_APIListenAddr = DefaultConfig_APIListenAddr
	}
	if len(l.Config_ChainWalkThrottle) == 0 {
		l.Config_ChainWalkThrottle = DefaultConfig_ChainWalkThrottle
	}
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

func (l *lotus) getAPIUsingLotusConfig() (api.FullNode, func(), error) {
	var (
		nodeAPI    api.FullNode
		nodeCloser func()
	)
	if len(l.Config_APIListenAddr) > 0 && len(l.Config_APIToken) > 0 {
		lotusAPI, apiCloser, err := rpc.GetFullNodeAPIUsingCredentials(l.Config_APIListenAddr, l.Config_APIToken)
		if err != nil {
			err = fmt.Errorf("connect with credentials: %v", err)
			return nil, nil, err
		}
		nodeAPI = lotusAPI
		nodeCloser = apiCloser
	} else {
		lotusAPI, apiCloser, err := rpc.GetFullNodeAPI(l.Config_DataPath)
		if err != nil {
			err = fmt.Errorf("connect from lotus state: %v", err)
			return nil, nil, err
		}
		nodeAPI = lotusAPI
		nodeCloser = apiCloser
	}
	return nodeAPI, nodeCloser, nil
}

// Start begins walking through the chain to update the datastore
func (l *lotus) Start(acc telegraf.Accumulator) error {
	l.setDefaults()

	throttleDuration, err := time.ParseDuration(l.Config_ChainWalkThrottle)
	if err != nil {
		return err
	}
	l.chainWalkThrottle = throttleDuration

	nodeAPI, nodeCloser, err := l.getAPIUsingLotusConfig()
	if err != nil {
		return err
	}
	l.api = nodeAPI

	if err := recordLotusInfoPoints(context.Background(), l.api, acc); err != nil {
		return err
	}

	chainHead, err := l.api.ChainHead(context.Background())
	if err != nil {
		return err
	}

	if err := recordMpoolPendingPoints(context.Background(), l.api, chainHead.Key(), acc, time.Now()); err != nil {
		return err
	}

	ctx, closeTipsChan := context.WithCancel(context.Background())
	tipsetsCh, err := rpc.GetTips(ctx, l.api, chainHead.Height(), 3)
	if err != nil {
		return err
	}

	ctx, closeBlocksChan := context.WithCancel(context.Background())
	headCh, err := l.api.SyncIncomingBlocks(ctx)
	if err != nil {
		return err
	}

	ctx, closeMsgChan := context.WithCancel(context.Background())
	msgCh, err := l.api.MpoolSub(ctx)
	if err != nil {
		return err
	}

	wg := new(sync.WaitGroup)
	wg.Add(3)
	l.shutdown = func() {
		closeTipsChan()
		closeBlocksChan()
		closeMsgChan()
		wg.Wait()
		nodeCloser()
	}

	// process tipsets
	processTipsets := func() {
		defer wg.Done()

		throttle := time.NewTicker(l.chainWalkThrottle)
		defer throttle.Stop()

		for range throttle.C {
			select {
			case t := <-tipsetsCh:
				go processTipset(ctx, l.api, acc, t, time.Now())
			case <-ctx.Done():
				return
			}
		}
	}
	go processTipsets()

	// process new headers
	processHeaders := func() {
		defer wg.Done()
		for {
			select {
			case head := <-headCh:
				go processHeader(ctx, acc, head, time.Now())
			case <-ctx.Done():
				return
			}
		}
	}
	go processHeaders()

	// process new messages
	processMpoolUpdates := func() {
		defer wg.Done()
		for {
			select {
			case mpu := <-msgCh:
				go processMpoolUpdate(ctx, acc, mpu, time.Now())
			case <-ctx.Done():
				return
			}
		}
	}
	go processMpoolUpdates()

	return nil
}

func processTipset(ctx context.Context, node api.FullNode, acc telegraf.Accumulator, newTipSet *types.TipSet, receivedAt time.Time) {
	height := newTipSet.Height()

	if err := recordTipsetMessagesPoints(ctx, node, acc, newTipSet, receivedAt); err != nil {
		log.Println("W! Failed to record messages", "height", height, "error", err)
		acc.AddError(fmt.Errorf("recording messages from tipset (@%d): %v", height, err))
		return
	}

	if err := recordTipsetStatePoints(ctx, node, acc, newTipSet); err != nil {
		log.Println("W! Failed to record state", "height", height, "error", err)
		acc.AddError(fmt.Errorf("recording state from tipset (@%d): %v", height, err))
		return
	}
	log.Println("I! Processed tipset height:", height)
}

func processHeader(ctx context.Context, acc telegraf.Accumulator, newHeader *types.BlockHeader, receivedAt time.Time) {
	err := recordBlockHeaderPoints(ctx, acc, newHeader, receivedAt)
	if err != nil {
		log.Println("W! Failed to record block header", "height", newHeader.Height, "error", err)
		acc.AddError(fmt.Errorf("recording block header (@%d cid: %s): %v", newHeader.Height, err))
		return
	}
	log.Println("I! Processed block header @ height:", newHeader.Height)
}

func processMpoolUpdate(ctx context.Context, acc telegraf.Accumulator, newMpoolUpdate api.MpoolUpdate, receivedAt time.Time) {
	if err := recordMpoolUpdatePoints(ctx, acc, newMpoolUpdate, receivedAt); err != nil {
		log.Println("W! Failed to record mpool update", "msg", newMpoolUpdate.Message.Cid().String(), "type", newMpoolUpdate.Type, "error", err.Error())
		acc.AddError(fmt.Errorf("recording mpool update (type: %d, cid: %s): %v", newMpoolUpdate.Type, newMpoolUpdate.Message.Cid(), err))
		return
	}
}

func (l *lotus) Stop() {
	l.shutdown()
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

func recordTipsetStatePoints(ctx context.Context, api api.FullNode, acc telegraf.Accumulator, tipset *types.TipSet) error {
	pc, err := api.StatePledgeCollateral(ctx, tipset.Key())
	if err != nil {
		return err
	}

	attoFil := types.NewInt(build.FilecoinPrecision).Int
	ts := time.Unix(int64(tipset.MinTimestamp()), int64(0))

	pcFil := new(big.Rat).SetFrac(pc.Int, attoFil)
	pcFilFloat, _ := pcFil.Float64()

	netBal, err := api.WalletBalance(ctx, builtin.RewardActorAddr)
	if err != nil {
		return err
	}

	netBalFil := new(big.Rat).SetFrac(netBal.Int, attoFil)
	netBalFilFloat, _ := netBalFil.Float64()

	// this is suppose to represent total miner power, but if full power can be
	// represented by 'chain.power' metric below, we should be able to simply
	// sum the total within the DB for each epoch.
	// ignoring this for now.
	//power, err := api.StateMinerPower(ctx, address.Address{}, tipset.Key())
	//if err != nil {
	//return err
	//}

	acc.AddGauge("chain_economics",
		map[string]interface{}{
			"total_supply":       netBalFilFloat,
			"pledged_collateral": pcFilFloat,
		}, map[string]string{
			"tipset_height": fmt.Sprintf("%d", tipset.Height()),
		}, ts)

	for _, blockHeader := range tipset.Blocks() {
		acc.AddFields("chain_election",
			map[string]interface{}{
				"election": 1,
			},
			map[string]string{
				"miner":         blockHeader.Miner.String(),
				"tipset_height": fmt.Sprintf("%d", tipset.Height()),
			}, ts)
	}
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

func recordMpoolPendingPoints(ctx context.Context, lotusAPI api.FullNode, head types.TipSetKey, acc telegraf.Accumulator, receivedAt time.Time) error {
	pendingMsgs, err := lotusAPI.MpoolPending(context.Background(), head)
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

type msgTag struct {
	actor    string
	method   uint64
	exitcode uint8
}

func recordTipsetMessagesPoints(ctx context.Context, api api.FullNode, acc telegraf.Accumulator, tipset *types.TipSet, receivedAt time.Time) error {
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

	msgs, err := api.ChainGetParentMessages(ctx, cids[0])
	if err != nil {
		return err
	}

	recp, err := api.ChainGetParentReceipts(ctx, cids[0])
	if err != nil {
		return err
	}

	msgn := make(map[msgTag][]cid.Cid)

	for i, msg := range msgs {
		bs, err := msg.Message.Serialize()
		if err != nil {
			return err
		}

		acc.AddHistogram("chain_messages",
			map[string]interface{}{
				"gas_price":    msg.Message.GasPrice.Int64(),
				"message_size": len(bs),
			}, map[string]string{}, ts)

		// capture actor message stats
		actor, err := api.StateGetActor(ctx, msg.Message.To, tipset.Key())
		if err != nil {
			return err
		}

		dm, err := multihash.Decode(actor.Code.Hash())
		if err != nil {
			continue
		}
		tag := msgTag{
			actor:    string(dm.Digest),
			method:   uint64(msg.Message.Method),
			exitcode: uint8(recp[i].ExitCode),
		}

		found := false
		for _, c := range msgn[tag] {
			if c.Equals(msg.Cid) {
				found = true
				break
			}
		}
		if !found {
			msgn[tag] = append(msgn[tag], msg.Cid)
		}
	}

	for t, m := range msgn {
		acc.AddFields("chain_actors",
			map[string]interface{}{
				"count": len(m),
			}, map[string]string{
				"actor":    t.actor,
				"method":   fmt.Sprintf("%d", t.method),
				"exitcode": fmt.Sprintf("%d", t.exitcode),
			}, ts)
	}

	return nil
}

func recordLotusInfoPoints(ctx context.Context, api api.FullNode, acc telegraf.Accumulator) error {
	nodePeerID, err := api.ID(context.Background())
	if err != nil {
		return err
	}

	v, err := api.Version(ctx)
	if err != nil {
		return err
	}

	network, err := api.StateNetworkName(ctx)
	if err != nil {
		return err
	}
	var commit string
	var version string
	versionTokens := strings.Split(v.Version, "+")
	if len(versionTokens) == 2 {
		version = versionTokens[0]
		commit = versionTokens[1]
	} else {
		log.Println("W! developer error, version string has changed format")
		version = v.Version
	}

	acc.AddFields("lotus_info",
		map[string]interface{}{
			"recorded_at": time.Now().UnixNano(),
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

func init() {
	var _ telegraf.ServiceInput = newLotus()
	inputs.Add("lotus", func() telegraf.Input {
		return newLotus()
	})
}
