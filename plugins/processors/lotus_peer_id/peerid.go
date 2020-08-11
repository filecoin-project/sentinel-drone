package lotus_peer_id

import (
	"context"
	"fmt"
	"log"
	"time"

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
	
	## Provide the measurments that should be excluded from receiving a peer_id tag.
	# exclude = [
		tag_name_to_exclude,
	# ]

`, DefaultConfig_APIListenAddr, DefaultConfig_DataPath)

type Exclude struct {
	Measurement string `toml:"measurement"`
}

type LotusPeerIDTagger struct {
	Config_DataPath      string    `toml:"lotus_datapath"`
	Config_APIListenAddr string    `toml:"lotus_api_multiaddr"`
	Config_APIToken      string    `toml:"lotus_api_token"`
	Config_Excluded      []Exclude `toml:"exclude"`

	apiBackOffThrottle time.Duration
	curAPIBackOff      time.Time

	initializedDefaults bool

	excludeMap map[string]struct{}

	peerID peer.ID
}

func (l *LotusPeerIDTagger) setAPIConfigDefaults() {
	if len(l.Config_DataPath) == 0 {
		l.Config_DataPath = DefaultConfig_DataPath
	}
	if len(l.Config_APIListenAddr) == 0 {
		l.Config_APIListenAddr = DefaultConfig_APIListenAddr
	}
	l.apiBackOffThrottle = time.Second * 10
	l.curAPIBackOff = time.Unix(0, 0)
}

func (l *LotusPeerIDTagger) setPeerID() error {
	nodeAPI, nodeCloser, err := l.getAPIUsingLotusConfig()
	if err != nil {
		return err
	}
	defer nodeCloser()
	pid, err := nodeAPI.ID(context.Background())
	if err != nil {
		return err
	}
	l.peerID = pid
	return nil
}

func (l *LotusPeerIDTagger) setExcludeMap() {
	ex := make(map[string]struct{})
	for _, e := range l.Config_Excluded {
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

func (l *LotusPeerIDTagger) SampleConfig() string {
	return sampleConfig
}

func (l *LotusPeerIDTagger) Description() string {
	return "Acquires the PeerID of a running Lotus daemon and includes it on measurements as a tag."
}

func (l *LotusPeerIDTagger) Apply(in ...telegraf.Metric) []telegraf.Metric {
	if !l.initializedDefaults {
		// configure lotus connection
		l.setAPIConfigDefaults()
		// create mapping of excluded metrics for faster processing.
		l.setExcludeMap()
		l.initializedDefaults = true
	}

	if time.Since(l.curAPIBackOff) < l.apiBackOffThrottle {
		return in
	}

	if l.peerID.Size() == 0 {
		log.Println("I! Setting up lotus_peer_id processor")
		// extract peerID from connected lotus daemon
		if err := l.setPeerID(); err != nil {
			log.Println("!E Failed to get peerID from lotus API", err.Error(), "Metrics Skipped:", len(in))
			l.curAPIBackOff = time.Now()
			return in
		}
	}

	for _, point := range in {
		if _, exclude := l.excludeMap[point.Name()]; !exclude {
			point.AddTag("peer_id", l.peerID.String())
		}
	}
	return in
}

func init() {
	processors.Add("lotus_peer_id", func() telegraf.Processor {
		return &LotusPeerIDTagger{}
	})
}
