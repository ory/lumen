# Changelog

## [0.0.21](https://github.com/ory/lumen/compare/v0.0.20...v0.0.21) (2026-03-19)


### Features

* **cmd:** add search subcommand with --trace diagnostic flag ([e1f4bf2](https://github.com/ory/lumen/commit/e1f4bf2fa6c5bc9a9ac1e3ca04840f36dc1430cd))
* **hook,stdio:** background index pre-warming and TTL pre-population ([338fdc9](https://github.com/ory/lumen/commit/338fdc97c564eb2e0e2d20df3f96dbfd99460ea0))
* **index:** seed CLI index command from sibling worktree on first use ([ff55769](https://github.com/ory/lumen/commit/ff55769e25bd3a064a770f438bdd46465cd215d8))


### Bug Fixes

* **cmd:** lint cleanup in search subcommand ([736e2c6](https://github.com/ory/lumen/commit/736e2c69f77d247cc3f01e91fe90bb9bb38f350f))
* make all e2e and unit tests pass ([57db1ea](https://github.com/ory/lumen/commit/57db1eae94546518e7deba74d3af16ede097f16c))
* **stdio:** default to git repo root when no existing index found ([929b4c9](https://github.com/ory/lumen/commit/929b4c92015d52bf9c9e0e90819f7275add6f3d0))
* **stdio:** only adopt cwd as index root when a DB already exists there ([80fe5d4](https://github.com/ory/lumen/commit/80fe5d44f0fd88eaedf4cdd9fee25d6d24a43ded))
* **stdio:** resolve symlinks in search input paths before processing ([f1caa70](https://github.com/ory/lumen/commit/f1caa70f00f97350dce8411204249821703b7846))
* **tests:** prevent fork bomb and close indexer FDs in cmd tests ([90c3d94](https://github.com/ory/lumen/commit/90c3d94f89177279b84ed367023ee36edd2b0aee))


### Performance Improvements

* **stdio:** skip merkle walk within 30s freshness TTL ([0e2c7a4](https://github.com/ory/lumen/commit/0e2c7a4d4e590fd50bbdd37a6c7c3ad28b3eff04))

## [0.0.20](https://github.com/ory/lumen/compare/v0.0.19...v0.0.20) (2026-03-18)


### Bug Fixes

* **index:** cap parent index walk at git repo boundary + store perf improvements ([#58](https://github.com/ory/lumen/issues/58)) ([c5074e2](https://github.com/ory/lumen/commit/c5074e24409d0b81e9393a87501017c0a61f4f7d))

## [0.0.19](https://github.com/ory/lumen/compare/v0.0.18...v0.0.19) (2026-03-18)


### Bug Fixes

* **index:** cap parent index walk at git repository boundary ([#56](https://github.com/ory/lumen/issues/56)) ([10e9635](https://github.com/ory/lumen/commit/10e9635b1d984cbd5f60ed46788d4171c2ca8b40))

## [0.0.18](https://github.com/ory/lumen/compare/v0.0.17...v0.0.18) (2026-03-18)


### Bug Fixes

* **index:** exclude internal git worktrees from project index ([#54](https://github.com/ory/lumen/issues/54)) ([4890908](https://github.com/ory/lumen/commit/4890908636f40e57b10c1a1edab95c41a9d44c4d))

## [0.0.17](https://github.com/ory/lumen/compare/v0.0.16...v0.0.17) (2026-03-17)


### Features

* **index:** seed worktree indexes from sibling to skip full re-embedding ([#52](https://github.com/ory/lumen/issues/52)) ([76809bd](https://github.com/ory/lumen/commit/76809bddbd2cc6ee8d071758d3185d64a6775f1d))

## [0.0.16](https://github.com/ory/lumen/compare/v0.0.15...v0.0.16) (2026-03-17)


### Reverts

* distribute lumen binary via npm optional dependencies ([#47](https://github.com/ory/lumen/issues/47)) ([#50](https://github.com/ory/lumen/issues/50)) ([4ea2314](https://github.com/ory/lumen/commit/4ea23147406455d7aa1894fd77f4753c551f5758))

## [0.0.15](https://github.com/ory/lumen/compare/v0.0.14...v0.0.15) (2026-03-17)


### Features

* **bench-swe:** add PTerm-based TUI progress component ([#43](https://github.com/ory/lumen/issues/43)) ([edb13dc](https://github.com/ory/lumen/commit/edb13dc5fb5b669fa32c54ff90579393e96f21aa))

## [0.0.14](https://github.com/ory/lumen/compare/v0.0.13...v0.0.14) (2026-03-16)


### Features

* distribute lumen binary via npm optional dependencies ([#47](https://github.com/ory/lumen/issues/47)) ([6c382af](https://github.com/ory/lumen/commit/6c382af75e2c79199910edd265cb61892284bf80))


### Performance Improvements

* **e2e:** parallelize all E2E tests and fix openIndexDB race ([#45](https://github.com/ory/lumen/issues/45)) ([1ae2993](https://github.com/ory/lumen/commit/1ae29934166c0943e8c82da29865929e75a73c17))

## [0.0.13](https://github.com/ory/lumen/compare/v0.0.12...v0.0.13) (2026-03-16)


### Features

* **embedder:** add manutic/nomic-embed-code:7b to known models ([#42](https://github.com/ory/lumen/issues/42)) ([b2b8e40](https://github.com/ory/lumen/commit/b2b8e401e54ff2bcd52f7dbfdc05170a7648a8a0))

## [0.0.12](https://github.com/ory/lumen/compare/v0.0.11...v0.0.12) (2026-03-16)


### Features

* add PTerm-based TUI progress output for index command ([#38](https://github.com/ory/lumen/issues/38)) ([7fee513](https://github.com/ory/lumen/commit/7fee5136cb9a33c4328394937b92305e4ea21163))

## [0.0.11](https://github.com/ory/lumen/compare/v0.0.10...v0.0.11) (2026-03-11)


### Features

* improved benchmarks and further tree sitter fixes ([67f9dd3](https://github.com/ory/lumen/commit/67f9dd38f82b3b8804e48c2d9602c5f0605f4fcc))

## [0.0.10](https://github.com/ory/lumen/compare/v0.0.9...v0.0.10) (2026-03-10)


### Features

* better benchmarks with swe-bench approach and improved chunker ([#34](https://github.com/ory/lumen/issues/34)) ([c722d66](https://github.com/ory/lumen/commit/c722d66cc6405a3c838af3ec6d725f42b62bffec))
* **chunker:** add C# language support ([915e917](https://github.com/ory/lumen/commit/915e917af528109e77be8a48195758c61b43be7e))


### Bug Fixes

* update guidance for reindexing the Lumen index ([f08c3b9](https://github.com/ory/lumen/commit/f08c3b959bb7a0ff9c74b364d522dbd476e84466))

## [0.0.9](https://github.com/ory/lumen/compare/v0.0.8...v0.0.9) (2026-03-04)


### Bug Fixes

* **plugin:** remove invalid manifest fields and bump version to 0.0.8 ([34ddfba](https://github.com/ory/lumen/commit/34ddfba22693b4fe20e4107a6eee5b64259a9c69))

## [0.0.8](https://github.com/ory/lumen/compare/v0.0.7...v0.0.8) (2026-03-04)


### Features

* add cwd parameter to MCP tools for reliable index root detection ([b622e62](https://github.com/ory/lumen/commit/b622e62cd4724bee02cbf48a9975809ec6426a7e))

## [0.0.7](https://github.com/ory/lumen/compare/v0.0.6...v0.0.7) (2026-03-04)


### Bug Fixes

* **ci:** remove redundant test/vet/lint steps from release job ([2f33014](https://github.com/ory/lumen/commit/2f33014cf94e32bf3c118821f3d5db8428327fd1))

## [0.0.6](https://github.com/ory/lumen/compare/v0.0.5...v0.0.6) (2026-03-04)


### Bug Fixes

* **ci:** chain goreleaser into release-please workflow ([8ceb5d6](https://github.com/ory/lumen/commit/8ceb5d6b3df277b7e5850b7c8c5a4cf0eaf350d6))

## [0.0.5](https://github.com/ory/lumen/compare/v0.0.4...v0.0.5) (2026-03-04)


### Bug Fixes

* **scripts:** correct jsonpath field name in release-please extra-files config ([528d6a3](https://github.com/ory/lumen/commit/528d6a3e67a871eb685741812ae41b12f98d39a9))
* **scripts:** pin binary download to manifest version ([9b1f90d](https://github.com/ory/lumen/commit/9b1f90d7eb0c2d0eeb44317ab16918669b702027))

## [0.0.4](https://github.com/ory/lumen/compare/v0.0.3...v0.0.4) (2026-03-04)


### Bug Fixes

* **scripts:** resolve version from release-please manifest before GitHub API ([42a4e4a](https://github.com/ory/lumen/commit/42a4e4a5bf9b24b5ecb60d17999ef39a60bfc87e))

## [0.0.3](https://github.com/ory/lumen/compare/v0.0.2...v0.0.3) (2026-03-04)


### Bug Fixes

* **index:** serialize concurrent Index/EnsureFresh calls to prevent duplicate key errors ([6f27677](https://github.com/ory/lumen/commit/6f2767704c011530d6ed9a236f6759605cdcbbdf))

## [0.0.2](https://github.com/ory/lumen/compare/v0.0.1...v0.0.2) (2026-03-04)


### Features

* reuse parent index for subdirectory searches ([7e9f803](https://github.com/ory/lumen/commit/7e9f80393fdad18502b8769c1bc7585c2145b44e))
