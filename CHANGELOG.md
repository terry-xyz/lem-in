# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [1.0.0](https://github.com/terry-xyz/lem-in/compare/v0.1.1...v1.0.0) (2026-04-24)

### Added

- *(cmd/visualizer-tui)* add animated transitions, Nord theme, and fix step-mode input handling ([#14](https://github.com/terry-xyz/lem-in/commit/66f59e5fa8b69db9f79bb518fe6281b3a25a0ce6))
- *(cmd/visualizer-tui)* add slider mode, auto-replay, and harden terminal lifecycle ([#15](https://github.com/terry-xyz/lem-in/commit/58a62d593392d0abc1cdc83659996a79b8bf5a9e))
- *(cmd/visualizer-web)* add depth fog, tube edges, particle bursts, timeline scrubber, and completion highlight ([#16](https://github.com/terry-xyz/lem-in/commit/0e9aa270e26e8db9b8241b4e9d0f45695c0541b8))
- *(cmd/visualizer-web)* redesign with colony theme, embedded GLB model, and radial BFS layout ([#19](https://github.com/terry-xyz/lem-in/commit/20d96dbfbb8617678fc4a42cfc0531de72faba06))
- *(cmd/visualizer-web)* add per-vertex cave darkening with depth gradient ([#20](https://github.com/terry-xyz/lem-in/commit/a4600a8fb5cf7969756208637aae03f926a9d30d))
- *(cmd/visualizer-web)* add show colony visibility toggle ([#28](https://github.com/terry-xyz/lem-in/commit/0d175b3cd4f52faf50a4ac14c905494d3d3482ef))
- *(cmd/visualizer-web/main.go)* add colony camera controls ([#29](https://github.com/terry-xyz/lem-in/commit/6dd2ad64c560715e8ed8c0b2fb74bb135241211d))

### Fixed

- *(cmd/visualizer-tui)* resolve graceful shutdown on MINGW64 piped execution ([#17](https://github.com/terry-xyz/lem-in/commit/8290987533dd2129517eefe31089bba98c602848))
- *(cmd/visualizer-web)* show ants through colony occlusion ([#26](https://github.com/terry-xyz/lem-in/commit/46a0d79131ebd7509848fb875f5de2b2a139bb9e))
- *(cmd/visualizer-web)* keep sparse rooms near colony ([#27](https://github.com/terry-xyz/lem-in/commit/9905b11c40ae631cb3871319b8acdef35b5513e1))

### Documentation

- *(lessons)* add comprehensive codebase learning guide ([#21](https://github.com/terry-xyz/lem-in/commit/e8407a08d7b1cfdd46c083f586f2b40e586aadab))
- *(lessons)* replace tutorial set with audit notes ([#22](https://github.com/terry-xyz/lem-in/commit/a4dd3e5c05028583a012d04dfcb80c1d404c74e2))
- *(README.md)* streamline repository guide ([#23](https://github.com/terry-xyz/lem-in/commit/d70aac62db666dc68d4d7d989c941c41d14e8a74))
- *(LICENSE)* add MIT license ([#31](https://github.com/terry-xyz/lem-in/commit/4ab917841fefa6b4c181ddcc7f2ec3e9dc0955cc))
- *(cmd)* add function-level comments to entrypoints and visualizers ([#32](https://github.com/terry-xyz/lem-in/commit/b04c58ec9e93c61f492d6476774465203702516f))
- *(internal)* add function-level comments to core packages ([#33](https://github.com/terry-xyz/lem-in/commit/fe20f780666baa57f273917ee0a9bf2396b6ef69))
- *(lessons)* remove generated audit notes ([#34](https://github.com/terry-xyz/lem-in/commit/4294fad1c62f8f42a8b2ebf8997d6c60ca023570))

### Other

- *(cmd/lem-in)* make solver CLI the single entrypoint ([#24](https://github.com/terry-xyz/lem-in/commit/ed5045093fb1c8c03539d6569e6553aba113942d))
- *(Makefile)* add web visualizer output targets ([#25](https://github.com/terry-xyz/lem-in/commit/f825af144d8a8197412ec49ee3edb42ea26c2d0a))

## [0.1.1](https://github.com/terry-xyz/lem-in/compare/v0.1.0...v0.1.1) (2026-03-05)

### Changed

- *(Makefile)* limit test parallelism to prevent memory spikes ([#10](https://github.com/terry-xyz/lem-in/commit/97111f1e28c6ff9116fde16ae9f27deea9a32e4b))

### Fixed

- *(cmd)* align TUI and web visualizers with spec requirements ([#8](https://github.com/terry-xyz/lem-in/commit/9e665f03c59f962356f89f5be1d4a89b3e325607))
- *(internal)* optimize path subset selection and harden API surface ([#11](https://github.com/terry-xyz/lem-in/commit/fbbafb9ea5622b2a342970ad339d1e2cd5de5254))

### Documentation

- *(CHANGELOG.md)* add changelog for v0.1.1 ([#12](https://github.com/terry-xyz/lem-in/commit/93ec0edec6781f5923a357b09183e8416cf357c0))

## 0.1.0 (2026-03-05)

### Added

- *(cmd,internal,examples)* implement complete lem-in solver pipeline ([#2](https://github.com/terry-xyz/lem-in/commit/c7f6f06a737992ec92ce9cc11cbeca6904642a46))
- *(cmd,internal/format)* add visualizers, format tests, and README ([#4](https://github.com/terry-xyz/lem-in/commit/7021a9d4183ea60127258e03e2737f4cc03c476d))

### Documentation

- create `.gitignore` ([#1](https://github.com/terry-xyz/lem-in/commit/8e4b588b235f7defc9dfe85e7d07bf5f8a545424))
- *(CHANGELOG.md)* add changelog for v0.1.0 ([#6](https://github.com/terry-xyz/lem-in/commit/7f92d8470ec89fa58d4a75e029c319c4d17c7eaa))
