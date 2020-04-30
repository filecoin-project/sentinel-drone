package lotus

import (
	"context"
	"fmt"
	"math/big"
	"time"

	"github.com/filecoin-project/lotus/api"
	"github.com/filecoin-project/lotus/build"
	"github.com/filecoin-project/lotus/chain/types"
	"github.com/filecoin-project/specs-actors/actors/builtin"
	"github.com/influxdata/telegraf"
	"github.com/ipfs/go-cid"
	"github.com/multiformats/go-multihash"
)

func RecordTipsetPoints(ctx context.Context, api api.FullNode, acc telegraf.Accumulator, tipset *types.TipSet) error {
	ts := time.Unix(int64(tipset.MinTimestamp()), int64(0))
	height := fmt.Sprintf("%d", tipset.Height())
	for _, blockheader := range tipset.Blocks() {
		bs, err := blockheader.Serialize()
		if err != nil {
			return err
		}

		acc.AddFields("chain.election",
			map[string]interface{}{
				"election": 1,
			},
			map[string]string{
				"tipset_height": height,
				"miner":         blockheader.Miner.String(),
			}, ts)

		acc.AddFields("chain.block",
			map[string]interface{}{
				"header_size": len(bs),
			},
			map[string]string{
				"tipset_height": height,
				"miner":         blockheader.Miner.String(),
			}, ts)
	}

	return nil
}

func RecordTipsetStatePoints(ctx context.Context, api api.FullNode, acc telegraf.Accumulator, tipset *types.TipSet) error {
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

	acc.AddGauge("chain.economics",
		map[string]interface{}{
			"total_supply":       netBalFilFloat,
			"pledged_collateral": pcFilFloat,
		}, map[string]string{
			"tipset_height": fmt.Sprintf("%d", tipset.Height()),
		}, ts)

	miners, err := api.StateListMiners(ctx, tipset.Key())
	for _, miner := range miners {
		power, err := api.StateMinerPower(ctx, miner, tipset.Key())
		if err != nil {
			return err
		}

		acc.AddGauge("chain.power",
			map[string]interface{}{
				"quality_adjusted_power": power.MinerPower.QualityAdjPower.Int64(),
			},
			map[string]string{
				"miner": miner.String(),
			}, ts)
	}

	return nil
}

type msgTag struct {
	actor    string
	method   uint64
	exitcode uint8
}

func RecordTipsetMessagesPoints(ctx context.Context, api api.FullNode, acc telegraf.Accumulator, tipset *types.TipSet) error {
	ts := time.Unix(int64(tipset.MinTimestamp()), int64(0))
	cids := tipset.Cids()
	if len(cids) == 0 {
		return fmt.Errorf("no cids in tipset")
	}

	acc.AddFields("chain.tipset",
		map[string]interface{}{
			"tipset_height": fmt.Sprintf("%d", tipset.Height()),
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

		acc.AddHistogram("chain.messages",
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
		acc.AddFields("chain.actors",
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
