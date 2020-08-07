package lotus_peer_id

import (
	"context"
	"fmt"
	"log"

	"github.com/influxdata/telegraf"
	"github.com/influxdata/telegraf/plugins/inputs/lotus/rpc"
	"github.com/influxdata/telegraf/plugins/processors"

	"github.com/filecoin-project/lotus/api"
	"github.com/libp2p/go-libp2p-core/peer"
)

const (
	DefaultConfig_DataPath      = "${HOME}/.lotus"
	DefaultConfig_APIListenAddr = "/ip4/0.0.0.0/tcp/1234"
)

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

`, DefaultConfig_DataPath, DefaultConfig_APIListenAddr)

type Exclude struct {
	Measurement string `toml:"measurement"`
}

type LotusPeerIDTagger struct {
	Config_DataPath      string `toml:"lotus_datapath"`
	Config_APIListenAddr string `toml:"lotus_api_multiaddr"`
	Config_APIToken      string `toml:"lotus_api_token"`

	Excludes []Exclude `toml:"exclude"`

	excludeMap map[string]struct{}

	peerID peer.ID
}

func (l *LotusPeerIDTagger) setDefaults() {
	if len(l.Config_DataPath) == 0 {
		l.Config_DataPath = DefaultConfig_DataPath
	}
	if len(l.Config_APIListenAddr) == 0 {
		l.Config_APIListenAddr = DefaultConfig_APIListenAddr
	}
}

func (l *LotusPeerIDTagger) setPeerID() {
	nodeAPI, nodeCloser, err := l.getAPIUsingLotusConfig()
	if err != nil {
		log.Println("E! Failed to get lotus api", "error", err.Error())
		return
	}
	defer nodeCloser()
	pid, err := nodeAPI.ID(context.Background())
	if err != nil {
		log.Println("E! Failed to get lotus peer ID", "error", err.Error())
		return
	}
	l.peerID = pid
}

func (l *LotusPeerIDTagger) setExcludeMap() {
	ex := make(map[string]struct{})
	for _, e := range l.Excludes {
		ex[e.Measurement] = struct{}{}
	}
	l.excludeMap = ex
}

func (l *LotusPeerIDTagger) getAPIUsingLotusConfig() (api.FullNode, func(), error) {
	var (
		nodeAPI    api.FullNode
		nodeCloser func()
	)
	if len(l.Config_APIListenAddr) > 0 && len(l.Config_APIToken) > 0 {
		api, apiCloser, err := rpc.GetFullNodeAPIUsingCredentials(l.Config_APIListenAddr, l.Config_APIToken)
		if err != nil {
			err = fmt.Errorf("connect with credentials: %v", err)
			return nil, nil, err
		}
		nodeAPI = api
		nodeCloser = apiCloser
	} else {
		api, apiCloser, err := rpc.GetFullNodeAPI(l.Config_DataPath)
		if err != nil {
			err = fmt.Errorf("connect from lotus state: %v", err)
			return nil, nil, err
		}
		nodeAPI = api
		nodeCloser = apiCloser
	}
	return nodeAPI, nodeCloser, nil
}

func (l *LotusPeerIDTagger) SampleConfig() string {
	return sampleConfig
}

func (l *LotusPeerIDTagger) Description() string {
	return "Decorate measurements that pass through this filter with the peer id of lotus daemon."
}

func (l *LotusPeerIDTagger) Apply(in ...telegraf.Metric) []telegraf.Metric {
	if l.peerID.Size() == 0 {
		log.Println("Setting up lotus_peer_id processor")
		// configure lotus connection
		l.setDefaults()
		// extract peerID from connected lotus daemon
		l.setPeerID()
		// create mapping of excluded metrics for faster processing.
		l.setExcludeMap()
	}

	for _, point := range in {
		if _, exclude := l.excludeMap[point.Name()]; !exclude {
			point.AddTag("lotus_peer_id", l.peerID.String())
		}
	}
	return in
}

func init() {
	processors.Add("lotus_peer_id", func() telegraf.Processor {
		return &LotusPeerIDTagger{}
	})
}
