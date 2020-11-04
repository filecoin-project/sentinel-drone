# Changelog
All notable changes to this project will be documented in this file.

The format is a variant of [Keep a Changelog](https://keepachangelog.com/en/1.0.0/) combined with categories from [Conventional Commits](https://www.conventionalcommits.org/en/v1.0.0/)

This project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html). Breaking changes should trigger an increment to the major version. Features increment the minor version and fixes or other changes increment the patch number.

<a name="v0.4.0"></a>
## [v0.4.0] - 2020-11-03
### Feat
- add changelog generator
- **lotus:** Remove Mpool Tracking; Adjust Metrics Names ([#34](https://github.com/filecoin-project/sentinel-drone/issues/34))

### Chore
- Include RC releases in push docker images ([#36](https://github.com/filecoin-project/sentinel-drone/issues/36))

### Dep
- **lotus:** Update to lotus v1.1.2 ([#35](https://github.com/filecoin-project/sentinel-drone/issues/35))


<a name="v0.3.0"></a>
## [v0.3.0] - 2020-10-11
### Feat
- add CI and Dockerfile ([#25](https://github.com/filecoin-project/sentinel-drone/issues/25))

### Fix
- peer id test was expecting wrong tag name ([#28](https://github.com/filecoin-project/sentinel-drone/issues/28))
- **lotus:** Drain workerDeathCh; Use telegraf logger
- **lotus:** Nits. Close chans, logging, func order, mem leak, returns
- **lotus-plugin:** resilient to daemon uptime
- **lotus_info:** requires > 0 metrics

### Chore
- Only push docker image on [a-z]*-master updates ([#32](https://github.com/filecoin-project/sentinel-drone/issues/32))
- Push docker image for semver tags ([#31](https://github.com/filecoin-project/sentinel-drone/issues/31))
- Update to lotus v0.8.0 ([#29](https://github.com/filecoin-project/sentinel-drone/issues/29))
- **dep:** Update to lotus[@v0](https://github.com/v0).9.0 ([#33](https://github.com/filecoin-project/sentinel-drone/issues/33))

### Dep
- Update lotus to v0.7.0
- **lotus:** Update to lotus[@3697a1af](https://github.com/3697a1af)


<a name="0.2.0+calibration-8.19.1"></a>
## [0.2.0+calibration-8.19.1] - 2020-08-20
### Feat
- set lotus_info from lotus plugin at startup
- add lotus_peer_id processor plugin

### Fix
- **lotus:** Remove unused code breaking build
- **lotus:** Remove chain_economics
- **lotus:** Simplify version capture
- **lotus:** Remove redundant field lotus_info.recorded_at

### Dep
- Update to lotus[@b3f27c00](https://github.com/b3f27c00)
- **lotus:** Update to lotus[@0292c979](https://github.com/0292c979)

### Deps
- Update to lotus[@f8a6e4dc](https://github.com/f8a6e4dc)

### Feedback
- add readme, fix error handling

### Polish
- add back off timing for api failures


<a name="0.1.0+calibration-8.8.0"></a>
## [0.1.0+calibration-8.8.0] - 2020-08-08
### Feat
- subscribe to mpool updates & collect pending

### Fix
- **lotus:** Remove chain_actors, chain_messages

### Dep
- Update to lotus[@ntwk](https://github.com/ntwk)-calibration-8.8.0
- Update to lotus[@ec098d483](https://github.com/ec098d483)

### Misc
- add .idea to .gitignore


<a name="0.1.0+calibration-7.24.0"></a>
## [0.1.0+calibration-7.24.0] - 2020-08-08
### Fix
- Lotus input no longer captures power metrics

### Dep
- Update to lotus[@ntwk](https://github.com/ntwk)-calibration-7.24.0

### Deps
- Update lotus/chainwatch to capture rewards
- Upgrade lotus to v0.4.1
- Update lotus to latest master (4af9a209)


<a name="0.1.0-interop.6.16.0"></a>
## 0.1.0-interop.6.16.0 - 2020-06-24
- Forked from [upstream] repo.

[Unreleased]: https://github.com/filecoin-project/sentinel-drone/compare/v0.4.0...HEAD
[v0.4.0]: https://github.com/filecoin-project/sentinel-drone/compare/v0.3.0...v0.4.0
[v0.3.0]: https://github.com/filecoin-project/sentinel-drone/compare/0.2.0+calibration-8.19.1...v0.3.0
[0.2.0+calibration-8.19.1]: https://github.com/filecoin-project/sentinel-drone/compare/0.1.0+calibration-8.8.0...0.2.0+calibration-8.19.1
[0.1.0+calibration-8.8.0]: https://github.com/filecoin-project/sentinel-drone/compare/0.1.0+calibration-7.24.0...0.1.0+calibration-8.8.0
[0.1.0+calibration-7.24.0]: https://github.com/filecoin-project/sentinel-drone/compare/0.1.0-interop.6.16.0...0.1.0+calibration-7.24.0
[upstream]: https://github.com/influxdata/telegraf
