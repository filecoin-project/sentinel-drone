# Lotus Peer ID Processor Plugin

The `lotus_peer_id` processor tags measurements with the configured lotus node's libp2p PeerID.

### Configuration:

```toml
[[processors.lotus_peer_id]]
    ## Provide the Lotus working path. The input plugin will scan the path contents
    ## for the correct listening address/port and API token. Note: Path must be readable
    ## by the telegraf process otherwise, the value should be manually set in api_listen_addr
    ## and api_token. This value is ignored if lotus_api_* values are populated.
    ## Default: "${HOME}/.lotus"
    lotus_datapath = "${HOME}/.lotus"

    ## Provide the token that telegraf can use to access the lotus API. The token only requires
    ## "read" privileges. This is useful for restricting metrics to a seperate token.
    # lotus_api_token = ""

    ## Provide the multiaddr that the lotus API is listening on. If API multiaddr is empty,
    ## telegraf will check lotus_datapath for values.
    ## Default: "/ip4/0.0.0.0/tcp/1234"
    # lotus_api_multiaddr = "/ip4/0.0.0.0/tcp/1234"

    ## Provide the measurments that should be excluded from receiving a peer_id tag.
    # exclude = [
    #    "metric_name_to_exclude",
    # ]

```

### Example processing:

```diff
- measurement1,tag1=foo,tag2=bar field1=1i,field2=2.1 1453831884664956455
+ measurement1,tag1=foo,tag2=bar,peer_id=12D3KooWPbdJEL5wV2MWxgHRwyUnWgnNA2jknBH9J7re8Krw4hyW field1=1i,field2=2.1 1453831884664956455
```
