# Filecoin Lotus Input Plugin

This [lotus](https://lotu.sh) plugin queries the JSON RPC endpoint to get
telemetry regarding the health of the node and network.

### Configuration

This section contains the default TOML to configure the plugin.  You can
generate it using `telegraf --usage lotus`.

```toml
[[inputs.lotus]]
	## Provide the multiaddr that the lotus API is listening on. If API multiaddr is empty,
	## telegraf will check lotus_datapath for values. Default: "${HOME}/.lotus"
	# lotus_api_multiaddr = "${HOME}/.lotus"

	## Provide the token that telegraf can use to access the lotus API. The token only requires
	## "read" privileges. This is useful for restricting metrics to a seperate token.
	# lotus_api_token = ""

	## Provide the Lotus working path. The input plugin will scan the path contents
	## for the correct listening address/port and API token. Note: Path must be readable
	## by the telegraf process otherwise, the value should be manually set in api_listen_addr
	## and api_token. This value is ignored if lotus_api_* values are populated.
	## Default: "/ip4/0.0.0.0/tcp/1234"
	lotus_datapath = "/ip4/0.0.0.0/tcp/1234"

	## Set the shortest duration between the processing of each tipset state. Valid time units
	## are "ns", "us" (or "µs"), "ms", "s", "m", "h". When telegraf starts collecting metrics,
	## it will also begin walking over the entire chain to update past state changes for the
	## dashboard. This setting prevents that walk from going too quickly. Default: "1s"
	chain_walk_throttle = "1s"
```

### Metrics (TODO)
Here you should add an optional description and links to where the user can
get more information about the measurements.

If the output is determined dynamically based on the input source, or there
are more metrics than can reasonably be listed, describe how the input is
mapped to the output.

- measurement1
  - tags:
    - tag1 (optional description)
    - tag2
  - fields:
    - field1 (type, unit)
    - field2 (float, percent)

- measurement2
  - tags:
    - tag3
  - fields:
    - field3 (integer, bytes)

### Sample Queries (TODO)

This section can contain some useful InfluxDB queries that can be used to get
started with the plugin or to generate dashboards.  For each query listed,
describe at a high level what data is returned.

Get the max, mean, and min for the measurement in the last hour:
```
SELECT max(field1), mean(field1), min(field1) FROM measurement1 WHERE tag1=bar AND time > now() - 1h GROUP BY tag
```

### Troubleshooting (TODO)

This optional section can provide basic troubleshooting steps that a user can
perform.

### Example Output (TODO)

This section shows example output in Line Protocol format.  You can often use
`telegraf --input-filter <plugin-name> --test` or use the `file` output to get
this information.

```
measurement1,tag1=foo,tag2=bar field1=1i,field2=2.1 1453831884664956455
measurement2,tag1=foo,tag2=bar,tag3=baz field3=1i 1453831884664956455
```
