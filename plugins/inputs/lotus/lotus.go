package lotus

import (
	"context"
	"fmt"
	"log"
	"sync"

	"github.com/filecoin-project/specs-actors/actors/abi"
	"github.com/influxdata/telegraf"
	"github.com/influxdata/telegraf/plugins/inputs"
	"github.com/influxdata/telegraf/plugins/inputs/lotus/internal"
)

type lotus struct {
	Auth_token string `toml:"auth_token"`
	Lotus_data string `toml:"lotus_data"`

	lastHeight int
	shutdown   func()
}

func newLotus() *lotus {
	return &lotus{}
}

// Description will appear directly above the plugin definition in the config file
func (l *lotus) Description() string {
	return `Capture telemetry produced by a local instance of Lotus`
}

const sampleConfig = `
  ## Provide the authentication token from Lotus to be used when accessing
  ## the JSON RPC api.
  auth_token = ""

  ## Provide the Lotus working path.
  lotus_data = "${HOME}/.lotus"
`

// SampleConfig will populate the sample configuration portion of the plugin's configuration
func (l *lotus) SampleConfig() string {
	return sampleConfig
}

// Gather is performed in Start
func (l *lotus) Gather(acc telegraf.Accumulator) error {
	return nil
}

// Start begins walking through the chain to update the datastore
func (l *lotus) Start(acc telegraf.Accumulator) error {
	api, apiCloser, err := internal.GetFullNodeAPI(l.Lotus_data)
	if err != nil {
		acc.AddError(err)
		log.Println(err)
		return err
	}

	if err := l.inflateState(); err != nil {
		acc.AddError(err)
		log.Println(err)
		return err
	}

	ctx, closeTipsChan := context.WithCancel(context.Background())

	wg := new(sync.WaitGroup)
	wg.Add(1)
	l.shutdown = func() {
		apiCloser()
		closeTipsChan()
		wg.Wait()
		l.persistState()
	}

	tipsetsCh, err := internal.GetTips(ctx, api, abi.ChainEpoch(0), 3)
	if err != nil {
		acc.AddError(err)
		log.Println(err)
		return err
	}

	go func() {
		for tipset := range tipsetsCh {
			height := tipset.Height()
			l.lastHeight = int(height)

			if err := RecordTipsetPoints(ctx, api, acc, tipset); err != nil {
				log.Println("W! Failed to record tipset", "height", height, "error", err)
				acc.AddError(fmt.Errorf("recording tipset (@%d): %v", height, err))
				continue
			}

			if err := RecordTipsetMessagesPoints(ctx, api, acc, tipset); err != nil {
				log.Println("W! Failed to record messages", "height", height, "error", err)
				acc.AddError(fmt.Errorf("recording messages from tipset (@%d): %v", height, err))
				continue
			}

			if err := RecordTipsetStatePoints(ctx, api, acc, tipset); err != nil {
				log.Println("W! Failed to record state", "height", height, "error", err)
				acc.AddError(fmt.Errorf("recording state from tipset (@%d): %v", height, err))
				continue
			}
		}
		wg.Done()
	}()

	return nil
}

func (l *lotus) Stop() {
	l.shutdown()
}

func init() {
	var _ telegraf.ServiceInput = newLotus()
	inputs.Add("lotus", func() telegraf.Input {
		return newLotus()
	})
}
