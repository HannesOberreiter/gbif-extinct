# Changelog

## [1.3.1](https://github.com/HannesOberreiter/gbif-extinct/compare/v1.3.0...v1.3.1) (2024-06-10)


### Bug Fixes

* :bug: add a timeout context to the render function ([#37](https://github.com/HannesOberreiter/gbif-extinct/issues/37)) ([1e86642](https://github.com/HannesOberreiter/gbif-extinct/commit/1e866427db865ba42d7042e184c1db297fddd2ce))


### Performance Improvements

* :zap: use points for large lists and refactor methods ([#35](https://github.com/HannesOberreiter/gbif-extinct/issues/35)) ([abc4ebe](https://github.com/HannesOberreiter/gbif-extinct/commit/abc4ebe31c80eef488f9bdfb6c68a886a7306660))

## [1.3.0](https://github.com/HannesOberreiter/gbif-extinct/compare/v1.2.2...v1.3.0) (2024-05-21)


### Features

* :art: add icons and logo ([#32](https://github.com/HannesOberreiter/gbif-extinct/issues/32)) ([996f266](https://github.com/HannesOberreiter/gbif-extinct/commit/996f26623a608d117db7856beff47f039ae63f60))


### Performance Improvements

* :zap: increase perf of counts and remove connection to prevent locking ([#30](https://github.com/HannesOberreiter/gbif-extinct/issues/30)) ([cbbefb9](https://github.com/HannesOberreiter/gbif-extinct/commit/cbbefb925c64a6f06c60148f2ff9a689c2e0e75d))

## [1.2.2](https://github.com/HannesOberreiter/gbif-extinct/compare/v1.2.1...v1.2.2) (2024-05-11)


### Bug Fixes

* :ambulance: reduce rows to prevent errors ([#28](https://github.com/HannesOberreiter/gbif-extinct/issues/28)) ([95044a5](https://github.com/HannesOberreiter/gbif-extinct/commit/95044a5c92101b8d583c09269fb4195ae42b58f5))

## [1.2.1](https://github.com/HannesOberreiter/gbif-extinct/compare/v1.2.0...v1.2.1) (2024-05-11)


### Bug Fixes

* :bug: remove gzip and fix nil errors ([#26](https://github.com/HannesOberreiter/gbif-extinct/issues/26)) ([bcaaead](https://github.com/HannesOberreiter/gbif-extinct/commit/bcaaead490785bf60b792273e4f38cdb8430417d))

## [1.2.0](https://github.com/HannesOberreiter/gbif-extinct/compare/v1.1.0...v1.2.0) (2024-05-11)


### Features

* :sparkles: add more info to about page ([#19](https://github.com/HannesOberreiter/gbif-extinct/issues/19)) ([cf18c3e](https://github.com/HannesOberreiter/gbif-extinct/commit/cf18c3ef16b4ef229eea8fb1ce6ae8966ea472d4))
* :sparkles: change synonyms logic to default to no synonyms ([#23](https://github.com/HannesOberreiter/gbif-extinct/issues/23)) ([cc0dea8](https://github.com/HannesOberreiter/gbif-extinct/commit/cc0dea81e607ef89323e3b6381bc5515af06cd22))
* ✨ added new button to download result as csv and added a cache buster to assets ([#25](https://github.com/HannesOberreiter/gbif-extinct/issues/25)) ([df309fb](https://github.com/HannesOberreiter/gbif-extinct/commit/df309fb83d10e68107944c2ecfd66d02c43e3e47))
* 📝 add note about Preserved Specimen ([#21](https://github.com/HannesOberreiter/gbif-extinct/issues/21)) ([e1271be](https://github.com/HannesOberreiter/gbif-extinct/commit/e1271be128036c5cc8cd6608edd8369aafd97905))

## [1.1.0](https://github.com/HannesOberreiter/gbif-extinct/compare/v1.0.0...v1.1.0) (2024-03-16)


### Features

* :sparkles: new import script ([#17](https://github.com/HannesOberreiter/gbif-extinct/issues/17)) ([4edc468](https://github.com/HannesOberreiter/gbif-extinct/commit/4edc46887bc657d7d84824eeb6a408335e937113))
* ✨ nicer styles and loading spinner ([#14](https://github.com/HannesOberreiter/gbif-extinct/issues/14)) ([99bf63d](https://github.com/HannesOberreiter/gbif-extinct/commit/99bf63d476aeb9c2268b3fd089163ca65553f011))


### Bug Fixes

* :bug: add missing readme to container ([#10](https://github.com/HannesOberreiter/gbif-extinct/issues/10)) ([916d2b8](https://github.com/HannesOberreiter/gbif-extinct/commit/916d2b808431ce51616294dc77a86384891e98e0))
* :bug: fix missing space for query ([#11](https://github.com/HannesOberreiter/gbif-extinct/issues/11)) ([fec0940](https://github.com/HannesOberreiter/gbif-extinct/commit/fec09403fc8450e0a29164345b30fb3a3856d4d0))
* :bug: more safety checks before insert ([#18](https://github.com/HannesOberreiter/gbif-extinct/issues/18)) ([5e3f31f](https://github.com/HannesOberreiter/gbif-extinct/commit/5e3f31f201a5c55662f14b992074225c2d8e893e))
* 🐛 stop reload full page on checkbox ([#6](https://github.com/HannesOberreiter/gbif-extinct/issues/6)) ([1f273c5](https://github.com/HannesOberreiter/gbif-extinct/commit/1f273c5c92af5c6fd7c7c4fbda6013dc1bf806f5))


### Performance Improvements

* :zap: increase scheduler frequency ([#8](https://github.com/HannesOberreiter/gbif-extinct/issues/8)) ([3642e51](https://github.com/HannesOberreiter/gbif-extinct/commit/3642e5136f8f032014e768e45a954777d643de2f))


### Reverts

* :rewind: change back scroll table ([#15](https://github.com/HannesOberreiter/gbif-extinct/issues/15)) ([eeaa8df](https://github.com/HannesOberreiter/gbif-extinct/commit/eeaa8dfdb70849bb83a45ce188c5908297654cd3))

## [1.0.0](https://github.com/HannesOberreiter/gbif-extinct/compare/v0.0.1-alpha.0...v1.0.0) (2024-02-28)


### ⚠ BREAKING CHANGES

* Release prototype ([#4](https://github.com/HannesOberreiter/gbif-extinct/issues/4))

### Features

* Release prototype ([#4](https://github.com/HannesOberreiter/gbif-extinct/issues/4)) ([f010473](https://github.com/HannesOberreiter/gbif-extinct/commit/f0104731d435f8e11a187e11b0acc93b7816eabf))


### Bug Fixes

* :bug: add certs to deployment for fetching ([e7074a2](https://github.com/HannesOberreiter/gbif-extinct/commit/e7074a2f38e6b10913fbfd02e3f80c24a2005b57))


### Miscellaneous Chores

* release 1.0.0 ([#5](https://github.com/HannesOberreiter/gbif-extinct/issues/5)) ([737c7bc](https://github.com/HannesOberreiter/gbif-extinct/commit/737c7bca5bc9c0e3ea5c444502ab9172d0f3d481))

## 0.0.1 (2024-02-26)

### Features

* :construction_worker: add docker CI push ([38caff2](https://github.com/HannesOberreiter/gbif-extinct/commit/38caff2f985abe80fb0b189f66f85fe6e3c41ae2))
* :construction: prototyping ([0901ad8](https://github.com/HannesOberreiter/gbif-extinct/commit/0901ad8f2c1282392c17a14c2f4f5a9746102408))
* :rocket: Prototype ([becd64a](https://github.com/HannesOberreiter/gbif-extinct/commit/becd64aaea7c38faf0880b4fa68f103734251f69))
* :sparkles: add compose file ([e475dd2](https://github.com/HannesOberreiter/gbif-extinct/commit/e475dd2553f1374ea5399ddfca35c49c2122bd56))
* :sparkles: add funding ([7bff602](https://github.com/HannesOberreiter/gbif-extinct/commit/7bff6028bb4fc0fda9f8ffadd49fe5a9ca1d0edb))

### Bug Fixes

* :bug: reduce max timeout ([dee9574](https://github.com/HannesOberreiter/gbif-extinct/commit/dee9574412613b6214bcb04fe28d431d789b6e43))
* :see_no_evil: add missing dockerignore ([12b15c5](https://github.com/HannesOberreiter/gbif-extinct/commit/12b15c5271bc61e5e1b4d9dd531eae1ad059220c))
* :see_no_evil: hide .env ([760c6f6](https://github.com/HannesOberreiter/gbif-extinct/commit/760c6f6e415fcb064114e769bba99516c3a68b72))
