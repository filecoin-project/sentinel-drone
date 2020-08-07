package lotus_peer_id

import (
	"github.com/bmizerany/assert"
	"testing"
	"time"

	"github.com/libp2p/go-libp2p-core/peer"
	"github.com/stretchr/testify/require"

	"github.com/influxdata/telegraf"
	"github.com/influxdata/telegraf/metric"
)

func newMetric(name string, tags map[string]string, fields map[string]interface{}) telegraf.Metric {
	if tags == nil {
		tags = map[string]string{}
	}
	if fields == nil {
		fields = map[string]interface{}{}
	}
	m, _ := metric.New(name, tags, fields, time.Now())
	return m
}

func TestTagMetricWithPeerID(t *testing.T) {
	pid, err := peer.Decode("12D3KooWRHNnsJmKEsJwqxesunwuQXS4V7n74PoK6Ca5NJ87bUcu")
	require.NoError(t, err)

	p := LotusPeerIDTagger{
		peerID: pid, // setting the peerID causes LotusPeerIDTagger to skip api init allowing Apply to be called
		Excludes:             []Exclude{
			{Measurement: "bar"}, // we will ignore measures with this name
		},
	}

	includeM := newMetric("foo", map[string]string{"hostname": "localhost", "region": "earth"}, nil)
	result := p.Apply(includeM)
	require.Len(t, result, 1)
	assert.Equal(t, map[string]string{"hostname": "localhost", "region": "earth", "lotus_peer_id": pid.String()},result[0].Tags())

	excludeM := newMetric("bar", map[string]string{"hostname": "localhost", "region": "earth"}, nil)
	result = p.Apply(excludeM)
	require.Len(t, result, 1)
	assert.Equal(t, map[string]string{"hostname": "localhost", "region": "earth"},result[0].Tags())

}
