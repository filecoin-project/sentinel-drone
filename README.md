# Sentinel Drone

> Metrics agent for Lotus 

[![build status](https://circleci.com/gh/filecoin-project/sentinel-drone.svg?style=svg)](https://app.circleci.com/pipelines/github/filecoin-project/sentinel-drone)

**Sentinel Drone** is a fork of Telegraf with input and processing plugins for Lotus, and the postgres output plugin. It's deployed as part of [Sentinel](https://github.com/filecoin-project/sentinel), the Filecoin Network Monitoring and Analysis System. 

**Telegraf** is an agent for collecting and writing metrics see https://github.com/influxdata/telegraf/

## Custom Plugins

Drone packages Telegraf with the following custom plugins:

* `input` [lotus](./plugins/inputs/lotus)
* `processor` [lotus_peer_id](./plugins/processors/lotus_peer_id)
* `output` [postgresql](./plugins/outputs/postgresql)

## Minimum Requirements

Telegraf shares the same [minimum requirements][] as Go:
- Linux kernel version 2.6.23 or later
- Windows 7 or later
- FreeBSD 11.2 or later
- MacOS 10.11 El Capitan or later

[minimum requirements]: https://github.com/golang/go/wiki/MinimumRequirements#minimum-requirements

## Installation:

### From Source:

Telegraf requires Go version 1.12 or newer, the Makefile requires GNU make.

1. [Install Go](https://golang.org/doc/install) >=1.12 (1.13 recommended)
2. Clone the Telegraf repository: `git clone https://github.com/filecoin-shipyard/sentinel-drone.git`
3. Run `make` from the source directory

## How to use it:

See usage with:

```
telegraf --help
```

#### Generate a telegraf config file:

```
telegraf config > telegraf.conf
```

#### Generate config with only cpu input & influxdb output plugins defined:

```
telegraf --section-filter agent:inputs:outputs --input-filter cpu --output-filter influxdb config
```

#### Run a single telegraf collection, outputing metrics to stdout:

```
telegraf --config telegraf.conf --test
```

#### Run telegraf with all plugins defined in config file:

```
telegraf --config telegraf.conf
```

#### Run telegraf, enabling the cpu & memory input, and influxdb output plugins:

```
telegraf --config telegraf.conf --input-filter cpu:mem --output-filter influxdb
```

## Documentation

[Latest Release Documentation][release docs].

For documentation on the latest development code see the [documentation index][devel docs].

[release docs]: https://docs.influxdata.com/telegraf
[devel docs]: docs
