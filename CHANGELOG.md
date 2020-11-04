<a name="unreleased"></a>
## [Unreleased]

### Chore
- Include RC releases in push docker images ([#36](https://github.com/filecoin-project/sentinel-drone/issues/36))


<a name="v1.0.0-rc1"></a>
## [v1.0.0-rc1] - 2020-11-02
### Dep
- **lotus:** Update to lotus v1.1.2 ([#35](https://github.com/filecoin-project/sentinel-drone/issues/35))

### Feat
- **lotus:** Remove Mpool Tracking; Adjust Metrics Names ([#34](https://github.com/filecoin-project/sentinel-drone/issues/34))


<a name="v0.3.0"></a>
## [v0.3.0] - 2020-10-11
### Chore
- Only push docker image on [a-z]*-master updates ([#32](https://github.com/filecoin-project/sentinel-drone/issues/32))
- Push docker image for semver tags ([#31](https://github.com/filecoin-project/sentinel-drone/issues/31))
- Update to lotus v0.8.0 ([#29](https://github.com/filecoin-project/sentinel-drone/issues/29))
- **dep:** Update to lotus[@v0](https://github.com/v0).9.0 ([#33](https://github.com/filecoin-project/sentinel-drone/issues/33))

### Dep
- Update lotus to v0.7.0
- **lotus:** Update to lotus[@3697a1af](https://github.com/3697a1af)

### Feat
- add CI and Dockerfile ([#25](https://github.com/filecoin-project/sentinel-drone/issues/25))

### Fix
- peer id test was expecting wrong tag name ([#28](https://github.com/filecoin-project/sentinel-drone/issues/28))
- **lotus:** Drain workerDeathCh; Use telegraf logger
- **lotus:** Nits. Close chans, logging, func order, mem leak, returns
- **lotus-plugin:** resilient to daemon uptime
- **lotus_info:** requires > 0 metrics


<a name="0.2.0+calibration-8.19.1"></a>
## [0.2.0+calibration-8.19.1] - 2020-08-20
### Dep
- Update to lotus[@b3f27c00](https://github.com/b3f27c00)
- **lotus:** Update to lotus[@0292c979](https://github.com/0292c979)

### Deps
- Update to lotus[@f8a6e4dc](https://github.com/f8a6e4dc)

### Feat
- set lotus_info from lotus plugin at startup
- add lotus_peer_id processor plugin

### Feedback
- add readme, fix error handling

### Fix
- **lotus:** Remove unused code breaking build
- **lotus:** Remove chain_economics
- **lotus:** Simplify version capture
- **lotus:** Remove redundant field lotus_info.recorded_at

### Polish
- add back off timing for api failures


<a name="0.1.0+calibration-8.8.0"></a>
## [0.1.0+calibration-8.8.0] - 2020-08-08
### Dep
- Update to lotus[@ntwk](https://github.com/ntwk)-calibration-8.8.0
- Update to lotus[@ec098d483](https://github.com/ec098d483)

### Feat
- subscribe to mpool updates & collect pending

### Fix
- **lotus:** Remove chain_actors, chain_messages

### Misc
- add .idea to .gitignore


<a name="0.1.0+calibration-7.24.0"></a>
## [0.1.0+calibration-7.24.0] - 2020-08-08
### Dep
- Update to lotus[@ntwk](https://github.com/ntwk)-calibration-7.24.0

### Deps
- Update lotus/chainwatch to capture rewards
- Upgrade lotus to v0.4.1
- Update lotus to latest master (4af9a209)

### Fix
- Lotus input no longer captures power metrics


<a name="0.1.0-interop.6.16.0"></a>
## 0.1.0-interop.6.16.0 - 2020-06-24
### Add
- Telegraf CouchDB Plugin

### Added
- server to tags

### Buffers
- fix bug when Write called before AddMetric

### Build
- add Makefile artifacts to .gitignore

### Cassandra
- update plugin supported prefix & fix panic

### Ceph
- sample config should reflect actual defaults ([#2228](https://github.com/filecoin-project/sentinel-drone/issues/2228))

### Ceph
- represent pgmap states using tags ([#2229](https://github.com/filecoin-project/sentinel-drone/issues/2229))

### Cgroup
- change fields -> files

### Config
- parse environment variables in the config file

### Core
- print error on output connect fail

### Doc
- **readme:** Added README.md.

### Docker
- optionally add labels as tags ([#2425](https://github.com/filecoin-project/sentinel-drone/issues/2425))

### Docker
- check type when totalling blkio & net metrics
- add container_id also to per cpu stats
- add container_id to blkio metrics

### Docs
- **readme:** Update to Amazon ECS

### Dovecot
- enabled global, user and ip queries
- remove extra newline from stats query

### Feat
- **config:** Support lotus plugin connecting via API multiaddr and token
- **docs:** README includes lotus_api_* and chain_walk_throttle opts
- **kubernetes:** Add kubernetes input plugin closes [#1774](https://github.com/filecoin-project/sentinel-drone/issues/1774)
- **lotus:** Update latest deps
- **lotus:** *lotus.chainWalkThrottle is configurable
- **nsq_consumer:** Add input plugin
- **telegraf:** update lotus dep to interop.6.16.0
- **timeout:** Use timeout setting
- **whitelist:** Converted black to whitelist

### Fix
- Last link on README
- **Godeps:** Added github.com/opencontainers/runc
- **build:** Fix filecoin-ffi build to follow lotus convension
- **build:** Include module update in deps
- **config:** Remove unused config values; Update config in README
- **config:** Made sample config consistent.
- **github:** remove github template from upstream remote
- **import:** Json parser lives outside internal
- **indent:** For configuration sample
- **kubernetes:** Only initialize RoundTripper once ([#1951](https://github.com/filecoin-project/sentinel-drone/issues/1951))
- **logging:** Reduce error verbosity from lotus plugin
- **lotus:** Remove historical chain sync; State management
- **mesos:** TOML annotation
- **prometheus:** Add support for bearer token to prometheus input plugin
- **sample:** Made TOML parser happy again
- **vet:** Range var used by goroutine

### Fix
- riak with read_repairs available

### Fixed
- differentiate stats gathered from multiple redis servers/instances

### Fixup
- BOM Trim -> TrimPrefix

### Globpath
- only walk tree if ** is defined

### Godep
- Add raidman riemann client

### Godep
- vendor all dependencies & add circle-test.sh

### Godeps
- update paho mqtt client dep
- update wvanbergen/kafka dependency

### Godeps
- Updated Godeps file for engine-api

### Godeps_windows
- update file

### Haproxy
- add README covering basics of the plugin
- update HAproxy docs URL
- clarify handling of http and socket addresses
- add support for socket name globbing
- move socket address detection to own function

### Haproxy_test
- extend tests to cover name globbing
- define expected results in one place

### Http_response
- override req.Host header properly

### Httpjson
- support configurable response_timeout ([#1651](https://github.com/filecoin-project/sentinel-drone/issues/1651))
- add unit test to verify that POST params get passed

### Input
- **docker:** Cleanup
- **docker:** Updated README
- **docker:** Fixed io sectors/io_time recursive
- **docker:** Fixed tests to work with engine-api
- **docker:** docker/engine-api

### Internal
- FlattenJSON, flatten arrays as well

### Ipmi_sensor
- allow @ symbol in password ([#2633](https://github.com/filecoin-project/sentinel-drone/issues/2633))

### Ipmi_sensors
- Allow : in password

### Jolokia
- handle multiple multi-dimensional attributes ([#1524](https://github.com/filecoin-project/sentinel-drone/issues/1524))
- use always POST
- add proxy mode

### Kafka
- Add support for using TLS authentication for the kafka output

### Kernel
- use strconv.ParseInt instead of strconv.Atoi

### Logparser
- allow numbers in ident & auth parameters

### Makefile
- ADVERTISED_HOST needs only be set during docker-compose target

### Memcached
- fix when a value contains a space

### Metric
- Fix negative number handling

### Mongodb
- Remove superfluous ReplSet log message
- dont print unecessary & inaccurate auth failure

### Mqtt_consumer
- option to set persistent session and client ID

### Nats_consumer
- buffer incoming messages

### Nstat
- fix nstat setting path for snmp6

### Ntpq
- correct number of seconds in an hour
- don't index ntp fields that dont exist

### Opentsdb
- add tcp:// prefix if not present

### Phpfpm
- add socket fcgi support

### Ping
- fix typo in README ([#2163](https://github.com/filecoin-project/sentinel-drone/issues/2163))

### Plugin
- **mesos:** Reversed removeGroup()
- **mesos:** Added goroutines.
- **mesos:** Initial commit

### Postgresql_extensible
- custom address in metrics output
- fix nil field values
- configurable measurement name
- Censor also other security related conn params

### Processes
- Don't return error if process exited ([#1283](https://github.com/filecoin-project/sentinel-drone/issues/1283))
- add 'unknown' procs (?)
- Fix zombie process procfs panic

### Procstat
- don't cache PIDs ([#2206](https://github.com/filecoin-project/sentinel-drone/issues/2206))

### Procstat
- field prefix fixup
- fix newlines in tags
- Add user, pidfile, pattern & exe tags

### README
- update golang requirement to 1.7 ([#2412](https://github.com/filecoin-project/sentinel-drone/issues/2412))
- Say when tagpass/tagdrop are valid from.

### RFR
- Initial support for ZFS on FreeBSD ([#1224](https://github.com/filecoin-project/sentinel-drone/issues/1224))

### Readme
- specify compression format for unpacking

### Redis
- support IPv6 addresses with no port

### Redis
- include per-db keyspace info

### Refactor
- **lotus:** Reorganize into single file; Remove unused files
- **naming:** For master specific settings

### Snmp
- support table indexes as tags ([#2366](https://github.com/filecoin-project/sentinel-drone/issues/2366))
- Allow lines with empty or missing tags ([#2172](https://github.com/filecoin-project/sentinel-drone/issues/2172))
- use a shared global translation cache
- make snmptranslate not required ([#2008](https://github.com/filecoin-project/sentinel-drone/issues/2008))
- fix initialization of table fields in manual tables ([#1836](https://github.com/filecoin-project/sentinel-drone/issues/1836))
- return error on unknown conversion type ([#1853](https://github.com/filecoin-project/sentinel-drone/issues/1853))

### Socket_listener
- clean up unix socket file on start & stop ([#2618](https://github.com/filecoin-project/sentinel-drone/issues/2618))

### Statsd
- allow template parsing fields. Default to value=
- If parsing a value to int fails, try to float and cast to int

### Statsd
- unit tests for gauges, sets, counters

### Test
- **unit:** Removed useless tests
- **unit:** Test for whitelisted metrics

### Typo
- prec -> perc

### Vagrantfile
- do a one-way rsync so that binaries don't get shared between VMs and host

### Win_perf_counters
- Format errors reported by pdh.dll in human-readable format ([#2338](https://github.com/filecoin-project/sentinel-drone/issues/2338))

### Reverts
- Add CLA check GitHub action ([#6479](https://github.com/filecoin-project/sentinel-drone/issues/6479))
- Update aerospike-client-go version to latest release ([#4128](https://github.com/filecoin-project/sentinel-drone/issues/4128))
- Add tengine input plugin ([#4160](https://github.com/filecoin-project/sentinel-drone/issues/4160))
- Undo Revert "Revert changes since 9b0af4478"
- New Particle Plugin
- bug fixes and refactoring
- Update README.md
- Updated README.md
- Small fixes
- Updated Test JSON
- Updated Test JSON
- New Particle.io Plugin for Telegraf
- Moving cgroup path name to field from tag to reduce cardinality ([#1457](https://github.com/filecoin-project/sentinel-drone/issues/1457))
- add pgbouncer plugin
- Revert graylog output
- exec plugin: allow using glob pattern in command list
- redis: support IPv6 addresses with no port
- PR [#59](https://github.com/filecoin-project/sentinel-drone/issues/59), implementation of multiple outputs
- Add log rotation to /etc/logrotate.d for deb and rpm packages
- Add log rotation to /etc/logrotate.d for deb and rpm packages


[Unreleased]: https://github.com/filecoin-project/sentinel-drone/compare/v1.0.0-rc1...HEAD
[v1.0.0-rc1]: https://github.com/filecoin-project/sentinel-drone/compare/v0.3.0...v1.0.0-rc1
[v0.3.0]: https://github.com/filecoin-project/sentinel-drone/compare/0.2.0+calibration-8.19.1...v0.3.0
[0.2.0+calibration-8.19.1]: https://github.com/filecoin-project/sentinel-drone/compare/0.1.0+calibration-8.8.0...0.2.0+calibration-8.19.1
[0.1.0+calibration-8.8.0]: https://github.com/filecoin-project/sentinel-drone/compare/0.1.0+calibration-7.24.0...0.1.0+calibration-8.8.0
[0.1.0+calibration-7.24.0]: https://github.com/filecoin-project/sentinel-drone/compare/0.1.0-interop.6.16.0...0.1.0+calibration-7.24.0
