# Changelog

## [0.1.0](https://github.com/ory/lumen/compare/v0.0.32...v0.1.0) (2026-04-07)


### ⚠ BREAKING CHANGES

* `lumen install` and `lumen uninstall` commands removed. Use `claude plugin install lumen` or `claude --plugin-dir .` instead.

### Features

* add cwd parameter to MCP tools for reliable index root detection ([b622e62](https://github.com/ory/lumen/commit/b622e62cd4724bee02cbf48a9975809ec6426a7e))
* add go/ast chunker for function/method/type/interface/const/var extraction ([8efb510](https://github.com/ory/lumen/commit/8efb510bf99e6035f2e290e882cfd16d7f1e7ff1))
* add index orchestrator with merkle-based incremental indexing ([da1fa90](https://github.com/ory/lumen/commit/da1fa90481382c31d5221492400da00a4a1105b6))
* add internal/config package for shared configuration ([234ca1e](https://github.com/ory/lumen/commit/234ca1ebc5476d2ce844496a2905cc3235753de3))
* add LUMEN_EMBED_DIMS/LUMEN_EMBED_CTX env vars for custom models ([#80](https://github.com/ory/lumen/issues/80)) ([2b124b1](https://github.com/ory/lumen/commit/2b124b1664086ad34a177b78906e26387bac789d))
* add Markdown/MDX chunker splitting by ATX headings ([81863f8](https://github.com/ory/lumen/commit/81863f8f9fd81b05dcec12749ffa452e6aa5f159))
* add merkle tree change detection for file hashing and diffing ([05de922](https://github.com/ory/lumen/commit/05de922ac51acbe86adf22c7df93da6437440b44))
* add model registry mapping Ollama model names to specs ([332e51e](https://github.com/ory/lumen/commit/332e51eb5336c94d8401d099e2e8bcc5dfcb1163))
* add MultiChunker and DefaultLanguages for all supported file types ([4c9d237](https://github.com/ory/lumen/commit/4c9d23798c921c4b8b3631e32f183af4d74e6a69))
* add ollama embedder with batching and retry ([a489800](https://github.com/ory/lumen/commit/a48980075656fa59f13400c501380b8e9ed62586))
* add PHP tree-sitter chunker ([f109b8b](https://github.com/ory/lumen/commit/f109b8b5cadfbcc5e29bb0eb1aa948a9f0889cf7))
* add PTerm-based TUI progress output for index command ([#38](https://github.com/ory/lumen/issues/38)) ([7fee513](https://github.com/ory/lumen/commit/7fee5136cb9a33c4328394937b92305e4ea21163))
* add sqlite + sqlite-vec store with vector search and batch inserts ([e12bb2d](https://github.com/ory/lumen/commit/e12bb2dc8f309ae6cb36865b4c39a367959dee89))
* add StructuredChunker for YAML/JSON files with TDD ([7d78504](https://github.com/ory/lumen/commit/7d7850402fd47fc37dc7f2944c2318b66b0f0487))
* add TreeSitterChunker for multi-language AST parsing ([cf25133](https://github.com/ory/lumen/commit/cf2513345550b9dc7e0caa67d33f8433a13c36b0))
* add YAML+JSON data chunker splitting by top-level keys ([466e9fd](https://github.com/ory/lumen/commit/466e9fd5302035183d46a171be595965a2e590bf))
* align Lumen packaging across Codex, Cursor, and OpenCode ([#73](https://github.com/ory/lumen/issues/73)) ([665c8db](https://github.com/ory/lumen/commit/665c8db8eef8f8d3b9b48e6f0781af766106e0b1))
* auto-reset vec_chunks on embedding dimension mismatch ([ff3279a](https://github.com/ory/lumen/commit/ff3279a97ac48525397597795e4ac43f4a1d4a39))
* **bench-swe:** add PTerm-based TUI progress component ([#43](https://github.com/ory/lumen/issues/43)) ([edb13dc](https://github.com/ory/lumen/commit/edb13dc5fb5b669fa32c54ff90579393e96f21aa))
* better benchmarks with swe-bench approach and improved chunker ([#34](https://github.com/ory/lumen/issues/34)) ([c722d66](https://github.com/ory/lumen/commit/c722d66cc6405a3c838af3ec6d725f42b62bffec))
* change default limit to 50, default min_score to 0.5; handle min_score=-1 as no-filter ([becc603](https://github.com/ory/lumen/commit/becc60350cfe047fb8dacbd15831ced15569a518))
* **chunker:** add C# language support ([915e917](https://github.com/ory/lumen/commit/915e917af528109e77be8a48195758c61b43be7e))
* **chunker:** add Dart language support ([#69](https://github.com/ory/lumen/issues/69)) ([00a985b](https://github.com/ory/lumen/commit/00a985b5deca29a2a1c5b10143304b6d0ebb0f1f))
* **cmd:** add search subcommand with --trace diagnostic flag ([e1f4bf2](https://github.com/ory/lumen/commit/e1f4bf2fa6c5bc9a9ac1e3ca04840f36dc1430cd))
* **cmd:** Close() waits for background reindex goroutines ([d7b7cad](https://github.com/ory/lumen/commit/d7b7cad177fe7604bdfcfe7a7ee2c3a3922e6840))
* **cmd:** non-blocking ensureIndexed with 15s timeout and background reindex ([fa883d7](https://github.com/ory/lumen/commit/fa883d7ed355cb1e349221a09819347510fb6afc))
* **cmd:** render StaleWarning in semantic_search output ([ce2503b](https://github.com/ory/lumen/commit/ce2503b84c9ecfd7f6677a7cefa9c3575d200040))
* **config:** add LUMEN_EMBED_DIMS/LUMEN_EMBED_CTX escape hatch for custom models ([#88](https://github.com/ory/lumen/issues/88)) ([29ad682](https://github.com/ory/lumen/commit/29ad682a5decb37193aab2d179e0d055635be88d))
* **config:** resolve ModelAliases before KnownModels lookup ([08989a4](https://github.com/ory/lumen/commit/08989a401ddf9879930d0eddcb0a70ffbd215ec2))
* distribute lumen binary via npm optional dependencies ([#47](https://github.com/ory/lumen/issues/47)) ([6c382af](https://github.com/ory/lumen/commit/6c382af75e2c79199910edd265cb61892284bf80))
* **embedder:** add manutic/nomic-embed-code:7b to known models ([#42](https://github.com/ory/lumen/issues/42)) ([b2b8e40](https://github.com/ory/lumen/commit/b2b8e401e54ff2bcd52f7dbfdc05170a7648a8a0))
* go-ast + sqlite-vec + mcp server ([8b13f64](https://github.com/ory/lumen/commit/8b13f645c6adadf2462a6af9c14aae4dbe8cd8ba))
* **hook,stdio:** background index pre-warming and TTL pre-population ([338fdc9](https://github.com/ory/lumen/commit/338fdc97c564eb2e0e2d20df3f96dbfd99460ea0))
* **hook:** spawn detached background indexer on session start ([d943ebf](https://github.com/ory/lumen/commit/d943ebf4cbfa4694ece3c4e9a8f7b8eee4a0dd07))
* ignore known vendor dirs and gitignore ([67668f1](https://github.com/ory/lumen/commit/67668f177fe54e2bdfe379f30dd32f202d6810df))
* improved benchmarks and further tree sitter fixes ([67f9dd3](https://github.com/ory/lumen/commit/67f9dd38f82b3b8804e48c2d9602c5f0605f4fcc))
* **index:** acquire flock before indexing, thread ctx, cancel on SIGTERM ([9409971](https://github.com/ory/lumen/commit/94099719b231084b06460df462f6beb6e896182f))
* **index:** add root hashes and worktree info to indexing plan log ([bd237e3](https://github.com/ory/lumen/commit/bd237e38d1450d1d77378c3ee5292947c9537320))
* **index:** add slog to background indexer and enrich Stats with change breakdown ([919eef1](https://github.com/ory/lumen/commit/919eef16911115be70d6e88794aac142e642f1bc))
* **indexlock:** add flock-based advisory lock for index coordination ([2a55737](https://github.com/ory/lumen/commit/2a5573715840ab9911824e1e975a4f47e3466da4))
* **index:** reuse ancestor index for non-git subdirectories ([3f89427](https://github.com/ory/lumen/commit/3f89427e965207a2718e893deaadcf7cbdd8a87f))
* **index:** seed CLI index command from sibling worktree on first use ([ff55769](https://github.com/ory/lumen/commit/ff55769e25bd3a064a770f438bdd46465cd215d8))
* **index:** seed worktree indexes from sibling to skip full re-embedding ([#52](https://github.com/ory/lumen/issues/52)) ([76809bd](https://github.com/ory/lumen/commit/76809bddbd2cc6ee8d071758d3185d64a6775f1d))
* **install:** show only supported models with local availability status ([6d5a5a0](https://github.com/ory/lumen/commit/6d5a5a0c59cbd6e3f9f6a14ac462ca6614ed2a89))
* LM Studio backend, chunk deduplication, and quality improvements ([0bd3fe3](https://github.com/ory/lumen/commit/0bd3fe37aa3e87034719b274cfd33adebb060ad4))
* make embedding dimensions configurable via AGENT_INDEX_EMBED_DIMS ([87944cd](https://github.com/ory/lumen/commit/87944cd52fe17b7351325d873dd377cc3fbc88dc))
* **merkle:** load global gitignore (core.excludesFile) ([04e7bcb](https://github.com/ory/lumen/commit/04e7bcbe99d139811d592189ed06d008cbc39a1c))
* **merkle:** support linguist-vendored in .gitattributes ([c7492d4](https://github.com/ory/lumen/commit/c7492d48e7090f83875926135bb84b326500475a))
* pass num_ctx to Ollama based on model's context length ([d5a4540](https://github.com/ory/lumen/commit/d5a4540c87aa17a1b58d4a667c52b11143ae1241))
* replace key-based DataChunker with plain text splitting for JSON and YAML ([4cff686](https://github.com/ory/lumen/commit/4cff6864d39e711d2ece96cf1f9933f0b5c360a7))
* reuse parent index for subdirectory searches ([7e9f803](https://github.com/ory/lumen/commit/7e9f80393fdad18502b8769c1bc7585c2145b44e))
* split oversized chunks at line boundaries before embedding ([cae8a90](https://github.com/ory/lumen/commit/cae8a908fb223321ecffbb06c102dadee9b090c0))
* **stdio:** group search results by file with score-based ranking ([967e23e](https://github.com/ory/lumen/commit/967e23e4bb4303e4e916ab2d5e529539ba4535f1))
* **stdio:** skip EnsureFresh when background indexer holds lock ([64a8d03](https://github.com/ory/lumen/commit/64a8d03c14ee9f0b15727d39365de9e8ddabc6fa))
* switch default model to qwen3-embedding:8b, rename module ([0df840a](https://github.com/ory/lumen/commit/0df840a83a769e2731568f38e5d02a5e6693ef65))
* use XML-tagged output format for search results ([5e34a89](https://github.com/ory/lumen/commit/5e34a8971a4dc6544da0d6605412a19ccb1d5d6c))
* wire MCP server with semantic_search and index_status tools ([a57bebc](https://github.com/ory/lumen/commit/a57bebcb642343d2aeea659f041a4396934ff105))
* wire MultiChunker and MakeExtSkip for multi-language indexing ([0caed1f](https://github.com/ory/lumen/commit/0caed1f1db6d4cf9ac97f338854dad7d148bc7f0))
* wire StructuredChunker into DefaultLanguages for yaml/yml/json files ([91ebc82](https://github.com/ory/lumen/commit/91ebc8234f51f431cedacca69879613b7b1cb4f3))


### Bug Fixes

* broken json ([5d4496e](https://github.com/ory/lumen/commit/5d4496e8599b2cf915cf4942b371a902449cb768))
* **build:** fix cross-platform build on Apple Silicon ([597821d](https://github.com/ory/lumen/commit/597821d211ace2c0de8a88124a116ea13170e784))
* C pointer-return query; add test coverage for all languages ([4446d1f](https://github.com/ory/lumen/commit/4446d1fe6b74a5ff82e71f5e61a5995b014c8299))
* **chunker:** cap leading comments to 10 lines ([34f56eb](https://github.com/ory/lumen/commit/34f56ebaf117d19d634f9dc947a97378793ed258))
* **chunker:** deduplicate overlapping tree-sitter chunks ([0581cf0](https://github.com/ory/lumen/commit/0581cf0c09cbf203bb18765db5131f009b4d71bd))
* **ci:** chain goreleaser into release-please workflow ([8ceb5d6](https://github.com/ory/lumen/commit/8ceb5d6b3df277b7e5850b7c8c5a4cf0eaf350d6))
* **ci:** remove redundant test/vet/lint steps from release job ([2f33014](https://github.com/ory/lumen/commit/2f33014cf94e32bf3c118821f3d5db8428327fd1))
* **cmd:** add TestMain to prevent fork-bomb in TestSpawnBackgroundIndexer ([19849aa](https://github.com/ory/lumen/commit/19849aac8dbe447f31ff5e76f74a8ee637c56389))
* **cmd:** close indexers on shutdown and set up signals before DB open ([859a274](https://github.com/ory/lumen/commit/859a27482898a4f3504d5c3571116d8ce07e6c28))
* **cmd:** eliminate reindex fragmentation causing constant cpu usage ([c7eb27e](https://github.com/ory/lumen/commit/c7eb27e77629d8bca02b5a24c97503a8509b7e35))
* **cmd:** lint cleanup in search subcommand ([736e2c6](https://github.com/ory/lumen/commit/736e2c69f77d247cc3f01e91fe90bb9bb38f350f))
* **cmd:** remove lumberjack rotation; add discardLog to test fixtures ([502f454](https://github.com/ory/lumen/commit/502f4541df1287083f4490039488498b8b778f58))
* convert cosine distance to similarity score, explicit ORDER BY score DESC ([85bc4eb](https://github.com/ory/lumen/commit/85bc4ebb8460682287ddea2bd5fa05be6cb0375c))
* **e2e:** add LUMEN_REINDEX_TIMEOUT to prevent premature search during indexing ([1131bbb](https://github.com/ory/lumen/commit/1131bbb4203743357d0ca9de79a81c9dc90c6af6))
* **e2e:** increase reindex timeout to 10m for PHP fixture set ([19e6b83](https://github.com/ory/lumen/commit/19e6b83dd9486b3baef2d52aa84621366bd30253))
* explicitly ignore f.Close() error in parseLinguistGenerated ([f95ca35](https://github.com/ory/lumen/commit/f95ca352065ced6fcb70bd1a35b519cf1ec887e0))
* fix 5 pre-existing E2E test failures ([64377cc](https://github.com/ory/lumen/commit/64377cc21e1975346371a7708ef892e58fdb88d0))
* harden concurrent indexer lifecycle (close cancellation, reindex dedup, reader consistency) ([e943a79](https://github.com/ory/lumen/commit/e943a7980fdce68c35fcbf7dfad16c17d856a793))
* **hook:** add license headers, smoke test, and Windows stub guidance ([2cffb63](https://github.com/ory/lumen/commit/2cffb63c6b92d510e766e5216900503dfc21a595))
* **index:** cap parent index walk at git repo boundary + store perf improvements ([#58](https://github.com/ory/lumen/issues/58)) ([c5074e2](https://github.com/ory/lumen/commit/c5074e24409d0b81e9393a87501017c0a61f4f7d))
* **index:** cap parent index walk at git repository boundary ([#56](https://github.com/ory/lumen/issues/56)) ([10e9635](https://github.com/ory/lumen/commit/10e9635b1d984cbd5f60ed46788d4171c2ca8b40))
* **index:** defer UpsertFile until chunks are flushed ([9daac4f](https://github.com/ory/lumen/commit/9daac4fac8877dd6e301834c55a7f287debca758))
* **index:** exclude internal git worktrees from project index ([#54](https://github.com/ory/lumen/issues/54)) ([4890908](https://github.com/ory/lumen/commit/4890908636f40e57b10c1a1edab95c41a9d44c4d))
* **index:** force reindex removes deleted files and purges stale extensions ([b3b3b48](https://github.com/ory/lumen/commit/b3b3b482f45a39a7a9cdf3e7f50f796238938227))
* **index:** prevent triple indexing when non-git parent contains git repos ([#81](https://github.com/ory/lumen/issues/81)) ([adf4e54](https://github.com/ory/lumen/commit/adf4e5488ae342bb0a616349a8c990a18879eed0))
* **index:** purge stale unsupported-extension records from donor seeding + freshness TTL cache + debug logging ([cea7687](https://github.com/ory/lumen/commit/cea768792bb7d60b66b9333e2dd1fbe9ef4bc7d4))
* **index:** serialize concurrent Index/EnsureFresh calls to prevent duplicate key errors ([6f27677](https://github.com/ory/lumen/commit/6f2767704c011530d6ed9a236f6759605cdcbbdf))
* **index:** simplify isBinaryContent using slices.Contains ([d04216a](https://github.com/ory/lumen/commit/d04216a48efa2fc896c97593dd0a91e4bd73738d))
* **index:** skip binary files based on NUL-byte detection ([4fef1d2](https://github.com/ory/lumen/commit/4fef1d24c6e09b52d057ad51627fef08c6796a13))
* **index:** write skip message to stderr, document signal-race trade-off ([bbc63f1](https://github.com/ory/lumen/commit/bbc63f1a2eea7dea49d25126c3844ff3f61e485a))
* **lint:** wrap deferred Close calls to satisfy errcheck ([5a200a2](https://github.com/ory/lumen/commit/5a200a242179082b2de0b86a29eb77bf309bda37))
* **log:** use lumberjack for log rotation; remove repro test; fix imports ([24d6906](https://github.com/ory/lumen/commit/24d6906fbacb0d957b55fd4d9c80728e7724de49))
* make all e2e and unit tests pass ([57db1ea](https://github.com/ory/lumen/commit/57db1eae94546518e7deba74d3af16ede097f16c))
* **mcp:** rename n_results to limit in semantic_search tool ([ca74157](https://github.com/ory/lumen/commit/ca74157885694100e3352747b478ef7a9ca92527))
* **merkle:** skip symlinks and files &gt;10MB during tree walk ([3f3de05](https://github.com/ory/lumen/commit/3f3de05db66226f058098a36ada7c56ad4d59696))
* **opencode:** invoke run.cmd directly instead of wrapping with sh ([f3fd945](https://github.com/ory/lumen/commit/f3fd945878f3bd3e97cd5c01be54fd82eba5401c))
* **plugin:** align plugin files with Claude Code plugin reference ([89cdb08](https://github.com/ory/lumen/commit/89cdb081e65ac023e056f90ad4b48f5ebf837840))
* **plugin:** remove invalid manifest fields and bump version to 0.0.8 ([34ddfba](https://github.com/ory/lumen/commit/34ddfba22693b4fe20e4107a6eee5b64259a9c69))
* prevent Search() from blocking during indexing by using separate read connection ([bdb837a](https://github.com/ory/lumen/commit/bdb837a46a1123a73608c6b84e886d949052461a)), closes [#94](https://github.com/ory/lumen/issues/94)
* reduce cyclomatic complexity of major functions ([8faef25](https://github.com/ory/lumen/commit/8faef25c3e4fece1f09da3d2c88998cb2dd863c3))
* reindex stability — sentinel resume, donor safety, worktree subdirs, progress bar ([#67](https://github.com/ory/lumen/issues/67)) ([07782d6](https://github.com/ory/lumen/commit/07782d63c97cc91c0c4dee1169d7965221d733ee))
* remove package-level chunk from Go AST chunker ([e53d95b](https://github.com/ory/lumen/commit/e53d95bf02b92e8c9e7cb6ac44c693f329e248c2))
* remove worktrees ([61384a7](https://github.com/ory/lumen/commit/61384a781f25e38d35be4e92bafcc309c6476358))
* replace custom minInt with builtin min in structured_test.go ([8aa1269](https://github.com/ory/lumen/commit/8aa126950c7bcd320a9c6a6706bad7952d0685b5))
* replace custom time.Sleep backoff with go-retry for context-aware retries ([91a14c1](https://github.com/ory/lumen/commit/91a14c184874abd6bef944e61ee4b6588632e59c))
* resolve import and style issues in merkle package ([fd8c1a8](https://github.com/ory/lumen/commit/fd8c1a85ac6157c86cb967da8bf52e450e73069d))
* **scripts:** correct jsonpath field name in release-please extra-files config ([528d6a3](https://github.com/ory/lumen/commit/528d6a3e67a871eb685741812ae41b12f98d39a9))
* **scripts:** pin binary download to manifest version ([9b1f90d](https://github.com/ory/lumen/commit/9b1f90d7eb0c2d0eeb44317ab16918669b702027))
* **scripts:** point to ory/lumen and download raw binaries ([fef815d](https://github.com/ory/lumen/commit/fef815d15180e8bfac86f09f0140cf8c5337afe8))
* **scripts:** resolve version from release-please manifest before GitHub API ([42a4e4a](https://github.com/ory/lumen/commit/42a4e4a5bf9b24b5ecb60d17999ef39a60bfc87e))
* **search:** pass db file path instead of directory to setupIndexer ([#75](https://github.com/ory/lumen/issues/75)) ([81cb4db](https://github.com/ory/lumen/commit/81cb4dbdbaff09713a38dfb28635805199c6d97b))
* set MaxOpenConns(1) for SQLite, fix pragma test to scan int ([7f82484](https://github.com/ory/lumen/commit/7f82484d59a75e0d3b61b3b384c98c28a1ef3529))
* skip permission-denied files instead of aborting index/search ([#84](https://github.com/ory/lumen/issues/84)) ([fde97b4](https://github.com/ory/lumen/commit/fde97b4dd15218eb3b56bb8a6d3619a7b5cfb830))
* **stdio:** default to git repo root when no existing index found ([929b4c9](https://github.com/ory/lumen/commit/929b4c92015d52bf9c9e0e90819f7275add6f3d0))
* **stdio:** increase scanner buffer to 1MB for long lines ([36c0435](https://github.com/ory/lumen/commit/36c0435fe97bac8fe217bb48763cb2d045f6c5a7))
* **stdio:** only adopt cwd as index root when a DB already exists there ([80fe5d4](https://github.com/ory/lumen/commit/80fe5d44f0fd88eaedf4cdd9fee25d6d24a43ded))
* **stdio:** propagate LUMEN_FRESHNESS_TTL from config to indexerCache ([cb4029b](https://github.com/ory/lumen/commit/cb4029b7eb13545425910ee0291dd6ec0748416b))
* **stdio:** resolve symlinks in search input paths before processing ([f1caa70](https://github.com/ory/lumen/commit/f1caa70f00f97350dce8411204249821703b7846))
* **stdio:** reuse parent index when search path points into internal worktree ([e96b464](https://github.com/ory/lumen/commit/e96b4643ec097e9052efcd3dc73086d71c4a7188))
* **stdio:** skip force reindex when background indexer holds lock ([a867910](https://github.com/ory/lumen/commit/a8679108c099b061528657d701d14f3e620b72ec))
* **store,index:** auto-recover from SQLite database corruption ([652b418](https://github.com/ory/lumen/commit/652b41800fc7ab45a06b4508c05a224c332b29fe))
* **store:** increase busy_timeout to 120s and use INSERT OR REPLACE for vec_chunks ([25a0e01](https://github.com/ory/lumen/commit/25a0e0184c9075d7637cf854bf4c001a47ae301a))
* **store:** increase busy_timeout to 30s for slow embedding batches ([91a2883](https://github.com/ory/lumen/commit/91a2883518bb1f4a289625ce6c3627d2a568f7e0))
* **store:** wrap dimension-reset deletes in a transaction ([33ea835](https://github.com/ory/lumen/commit/33ea835feaf8c49719be9d118c0fb1dceef0ff71))
* **test:** resolve symlinks in e2e test paths for macOS compatibility ([fca3aec](https://github.com/ory/lumen/commit/fca3aec7dfef0bff40db0be033d6667dbe62c50a))
* **tests:** prevent fork bomb and close indexer FDs in cmd tests ([90c3d94](https://github.com/ory/lumen/commit/90c3d94f89177279b84ed367023ee36edd2b0aee))
* tool description to increase invokation ([5f92ca4](https://github.com/ory/lumen/commit/5f92ca4f6e6b3ff5217cb6fced5f5455757fd635))
* **tui:** disable ShowElapsedTime to prevent pterm ticker goroutine race ([cec32c5](https://github.com/ory/lumen/commit/cec32c52e3df45b379bf7e5da6480f769582cf37))
* update guidance for reindexing the Lumen index ([f08c3b9](https://github.com/ory/lumen/commit/f08c3b959bb7a0ff9c74b364d522dbd476e84466))
* update MinScoreFilter test to use -1 for all-results and 0.5 threshold ([562e474](https://github.com/ory/lumen/commit/562e474093af1f3c701c0fdc5f5607f404c18744))
* use query_only pragma instead of read-only mode for WAL compatibility ([34e5123](https://github.com/ory/lumen/commit/34e5123d8770e37d4c6b8732b95e507ec4fd0563))
* version to 0.0.1-alpha.4 ([816452a](https://github.com/ory/lumen/commit/816452aa7263f65e675d4f01a5c5583f9fd6c693))


### Performance Improvements

* add SQLite write pragmas and chunk column indexes ([5bac05f](https://github.com/ory/lumen/commit/5bac05f1fdbef5a17c27a6d1bffd3ea5be0c01ae))
* avoid double Merkle tree build by sharing tree between EnsureFresh and indexWithTree ([91f3cae](https://github.com/ory/lumen/commit/91f3cae471d962f8b002fc74389b993d1c31a262))
* combine Stats() into one query, add GetMetaBatch for Status() ([80300f9](https://github.com/ory/lumen/commit/80300f9c300262a2c5890cc2cf4f3bf1d12e15c1))
* **e2e:** parallelize all E2E tests and fix openIndexDB race ([#45](https://github.com/ory/lumen/issues/45)) ([1ae2993](https://github.com/ory/lumen/commit/1ae29934166c0943e8c82da29865929e75a73c17))
* make Status() DB-only by storing total_files in metadata, remove StaleFiles ([59826c2](https://github.com/ory/lumen/commit/59826c2b7d6184000eab94ae4ebc10497a30d785))
* parallelize file reads in merkle.BuildTree with worker pool ([02e262e](https://github.com/ory/lumen/commit/02e262e9397e604955def1409611d0d062a23192))
* **stdio:** skip merkle walk within 30s freshness TTL ([0e2c7a4](https://github.com/ory/lumen/commit/0e2c7a4d4e590fd50bbdd37a6c7c3ad28b3eff04))
* stream chunk embed+insert in batches of 256 to bound memory ([7441642](https://github.com/ory/lumen/commit/7441642c59f688dfff06101cb2becd52430b78c9))
* use RWMutex with double-checked locking in indexer cache ([5d1c2b6](https://github.com/ory/lumen/commit/5d1c2b61069792d6cf3a44edff1cd0e0cfd6ade2))


### Reverts

* distribute lumen binary via npm optional dependencies ([#47](https://github.com/ory/lumen/issues/47)) ([#50](https://github.com/ory/lumen/issues/50)) ([4ea2314](https://github.com/ory/lumen/commit/4ea23147406455d7aa1894fd77f4753c551f5758))


### Code Refactoring

* migrate from install/uninstall to Claude Code plugin system ([0920f3d](https://github.com/ory/lumen/commit/0920f3d33f64ec836a9e5b2dc9bef570393524d4))

## [0.0.32](https://github.com/ory/lumen/compare/v0.0.31...v0.0.32) (2026-04-07)


### Bug Fixes

* **opencode:** invoke run.cmd directly instead of wrapping with sh ([f3fd945](https://github.com/ory/lumen/commit/f3fd945878f3bd3e97cd5c01be54fd82eba5401c))

## [0.0.31](https://github.com/ory/lumen/compare/v0.0.30...v0.0.31) (2026-04-06)


### Features

* **config:** resolve ModelAliases before KnownModels lookup ([08989a4](https://github.com/ory/lumen/commit/08989a401ddf9879930d0eddcb0a70ffbd215ec2))


### Bug Fixes

* harden concurrent indexer lifecycle (close cancellation, reindex dedup, reader consistency) ([e943a79](https://github.com/ory/lumen/commit/e943a7980fdce68c35fcbf7dfad16c17d856a793))
* **mcp:** rename n_results to limit in semantic_search tool ([ca74157](https://github.com/ory/lumen/commit/ca74157885694100e3352747b478ef7a9ca92527))

## [0.0.30](https://github.com/ory/lumen/compare/v0.0.29...v0.0.30) (2026-04-05)


### Features

* align Lumen packaging across Codex, Cursor, and OpenCode ([#73](https://github.com/ory/lumen/issues/73)) ([665c8db](https://github.com/ory/lumen/commit/665c8db8eef8f8d3b9b48e6f0781af766106e0b1))


### Bug Fixes

* **e2e:** add LUMEN_REINDEX_TIMEOUT to prevent premature search during indexing ([1131bbb](https://github.com/ory/lumen/commit/1131bbb4203743357d0ca9de79a81c9dc90c6af6))
* **e2e:** increase reindex timeout to 10m for PHP fixture set ([19e6b83](https://github.com/ory/lumen/commit/19e6b83dd9486b3baef2d52aa84621366bd30253))
* prevent Search() from blocking during indexing by using separate read connection ([bdb837a](https://github.com/ory/lumen/commit/bdb837a46a1123a73608c6b84e886d949052461a)), closes [#94](https://github.com/ory/lumen/issues/94)
* use query_only pragma instead of read-only mode for WAL compatibility ([34e5123](https://github.com/ory/lumen/commit/34e5123d8770e37d4c6b8732b95e507ec4fd0563))

## [0.0.29](https://github.com/ory/lumen/compare/v0.0.28...v0.0.29) (2026-04-03)


### Features

* **index:** reuse ancestor index for non-git subdirectories ([3f89427](https://github.com/ory/lumen/commit/3f89427e965207a2718e893deaadcf7cbdd8a87f))


### Bug Fixes

* **test:** resolve symlinks in e2e test paths for macOS compatibility ([fca3aec](https://github.com/ory/lumen/commit/fca3aec7dfef0bff40db0be033d6667dbe62c50a))

## [0.0.28](https://github.com/ory/lumen/compare/v0.0.27...v0.0.28) (2026-04-02)


### Features

* **config:** add LUMEN_EMBED_DIMS/LUMEN_EMBED_CTX escape hatch for custom models ([#88](https://github.com/ory/lumen/issues/88)) ([29ad682](https://github.com/ory/lumen/commit/29ad682a5decb37193aab2d179e0d055635be88d))


### Bug Fixes

* **index:** prevent triple indexing when non-git parent contains git repos ([#81](https://github.com/ory/lumen/issues/81)) ([adf4e54](https://github.com/ory/lumen/commit/adf4e5488ae342bb0a616349a8c990a18879eed0))
* skip permission-denied files instead of aborting index/search ([#84](https://github.com/ory/lumen/issues/84)) ([fde97b4](https://github.com/ory/lumen/commit/fde97b4dd15218eb3b56bb8a6d3619a7b5cfb830))

## [0.0.27](https://github.com/ory/lumen/compare/v0.0.26...v0.0.27) (2026-04-02)


### Features

* add LUMEN_EMBED_DIMS/LUMEN_EMBED_CTX env vars for custom models ([#80](https://github.com/ory/lumen/issues/80)) ([2b124b1](https://github.com/ory/lumen/commit/2b124b1664086ad34a177b78906e26387bac789d))

## [0.0.26](https://github.com/ory/lumen/compare/v0.0.25...v0.0.26) (2026-04-01)


### Bug Fixes

* **search:** pass db file path instead of directory to setupIndexer ([#75](https://github.com/ory/lumen/issues/75)) ([81cb4db](https://github.com/ory/lumen/commit/81cb4dbdbaff09713a38dfb28635805199c6d97b))

## [0.0.25](https://github.com/ory/lumen/compare/v0.0.24...v0.0.25) (2026-03-28)


### Features

* **chunker:** add Dart language support ([#69](https://github.com/ory/lumen/issues/69)) ([00a985b](https://github.com/ory/lumen/commit/00a985b5deca29a2a1c5b10143304b6d0ebb0f1f))

## [0.0.24](https://github.com/ory/lumen/compare/v0.0.23...v0.0.24) (2026-03-27)


### Bug Fixes

* reindex stability — sentinel resume, donor safety, worktree subdirs, progress bar ([#67](https://github.com/ory/lumen/issues/67)) ([07782d6](https://github.com/ory/lumen/commit/07782d63c97cc91c0c4dee1169d7965221d733ee))

## [0.0.23](https://github.com/ory/lumen/compare/v0.0.22...v0.0.23) (2026-03-24)


### Features

* **cmd:** Close() waits for background reindex goroutines ([d7b7cad](https://github.com/ory/lumen/commit/d7b7cad177fe7604bdfcfe7a7ee2c3a3922e6840))
* **cmd:** non-blocking ensureIndexed with 15s timeout and background reindex ([fa883d7](https://github.com/ory/lumen/commit/fa883d7ed355cb1e349221a09819347510fb6afc))
* **cmd:** render StaleWarning in semantic_search output ([ce2503b](https://github.com/ory/lumen/commit/ce2503b84c9ecfd7f6677a7cefa9c3575d200040))
* **index:** add root hashes and worktree info to indexing plan log ([bd237e3](https://github.com/ory/lumen/commit/bd237e38d1450d1d77378c3ee5292947c9537320))
* **index:** add slog to background indexer and enrich Stats with change breakdown ([919eef1](https://github.com/ory/lumen/commit/919eef16911115be70d6e88794aac142e642f1bc))


### Bug Fixes

* **cmd:** eliminate reindex fragmentation causing constant cpu usage ([c7eb27e](https://github.com/ory/lumen/commit/c7eb27e77629d8bca02b5a24c97503a8509b7e35))
* **store,index:** auto-recover from SQLite database corruption ([652b418](https://github.com/ory/lumen/commit/652b41800fc7ab45a06b4508c05a224c332b29fe))
* **store:** increase busy_timeout to 120s and use INSERT OR REPLACE for vec_chunks ([25a0e01](https://github.com/ory/lumen/commit/25a0e0184c9075d7637cf854bf4c001a47ae301a))

## [0.0.22](https://github.com/ory/lumen/compare/v0.0.21...v0.0.22) (2026-03-22)


### Features

* **hook:** spawn detached background indexer on session start ([d943ebf](https://github.com/ory/lumen/commit/d943ebf4cbfa4694ece3c4e9a8f7b8eee4a0dd07))
* **index:** acquire flock before indexing, thread ctx, cancel on SIGTERM ([9409971](https://github.com/ory/lumen/commit/94099719b231084b06460df462f6beb6e896182f))
* **indexlock:** add flock-based advisory lock for index coordination ([2a55737](https://github.com/ory/lumen/commit/2a5573715840ab9911824e1e975a4f47e3466da4))
* **merkle:** load global gitignore (core.excludesFile) ([04e7bcb](https://github.com/ory/lumen/commit/04e7bcbe99d139811d592189ed06d008cbc39a1c))
* **merkle:** support linguist-vendored in .gitattributes ([c7492d4](https://github.com/ory/lumen/commit/c7492d48e7090f83875926135bb84b326500475a))
* **stdio:** skip EnsureFresh when background indexer holds lock ([64a8d03](https://github.com/ory/lumen/commit/64a8d03c14ee9f0b15727d39365de9e8ddabc6fa))


### Bug Fixes

* **chunker:** cap leading comments to 10 lines ([34f56eb](https://github.com/ory/lumen/commit/34f56ebaf117d19d634f9dc947a97378793ed258))
* **chunker:** deduplicate overlapping tree-sitter chunks ([0581cf0](https://github.com/ory/lumen/commit/0581cf0c09cbf203bb18765db5131f009b4d71bd))
* **cmd:** add TestMain to prevent fork-bomb in TestSpawnBackgroundIndexer ([19849aa](https://github.com/ory/lumen/commit/19849aac8dbe447f31ff5e76f74a8ee637c56389))
* **cmd:** close indexers on shutdown and set up signals before DB open ([859a274](https://github.com/ory/lumen/commit/859a27482898a4f3504d5c3571116d8ce07e6c28))
* **cmd:** remove lumberjack rotation; add discardLog to test fixtures ([502f454](https://github.com/ory/lumen/commit/502f4541df1287083f4490039488498b8b778f58))
* **hook:** add license headers, smoke test, and Windows stub guidance ([2cffb63](https://github.com/ory/lumen/commit/2cffb63c6b92d510e766e5216900503dfc21a595))
* **index:** defer UpsertFile until chunks are flushed ([9daac4f](https://github.com/ory/lumen/commit/9daac4fac8877dd6e301834c55a7f287debca758))
* **index:** force reindex removes deleted files and purges stale extensions ([b3b3b48](https://github.com/ory/lumen/commit/b3b3b482f45a39a7a9cdf3e7f50f796238938227))
* **index:** purge stale unsupported-extension records from donor seeding + freshness TTL cache + debug logging ([cea7687](https://github.com/ory/lumen/commit/cea768792bb7d60b66b9333e2dd1fbe9ef4bc7d4))
* **index:** simplify isBinaryContent using slices.Contains ([d04216a](https://github.com/ory/lumen/commit/d04216a48efa2fc896c97593dd0a91e4bd73738d))
* **index:** skip binary files based on NUL-byte detection ([4fef1d2](https://github.com/ory/lumen/commit/4fef1d24c6e09b52d057ad51627fef08c6796a13))
* **index:** write skip message to stderr, document signal-race trade-off ([bbc63f1](https://github.com/ory/lumen/commit/bbc63f1a2eea7dea49d25126c3844ff3f61e485a))
* **lint:** wrap deferred Close calls to satisfy errcheck ([5a200a2](https://github.com/ory/lumen/commit/5a200a242179082b2de0b86a29eb77bf309bda37))
* **log:** use lumberjack for log rotation; remove repro test; fix imports ([24d6906](https://github.com/ory/lumen/commit/24d6906fbacb0d957b55fd4d9c80728e7724de49))
* **merkle:** skip symlinks and files &gt;10MB during tree walk ([3f3de05](https://github.com/ory/lumen/commit/3f3de05db66226f058098a36ada7c56ad4d59696))
* remove worktrees ([61384a7](https://github.com/ory/lumen/commit/61384a781f25e38d35be4e92bafcc309c6476358))
* **stdio:** increase scanner buffer to 1MB for long lines ([36c0435](https://github.com/ory/lumen/commit/36c0435fe97bac8fe217bb48763cb2d045f6c5a7))
* **stdio:** propagate LUMEN_FRESHNESS_TTL from config to indexerCache ([cb4029b](https://github.com/ory/lumen/commit/cb4029b7eb13545425910ee0291dd6ec0748416b))
* **stdio:** reuse parent index when search path points into internal worktree ([e96b464](https://github.com/ory/lumen/commit/e96b4643ec097e9052efcd3dc73086d71c4a7188))
* **stdio:** skip force reindex when background indexer holds lock ([a867910](https://github.com/ory/lumen/commit/a8679108c099b061528657d701d14f3e620b72ec))
* **store:** increase busy_timeout to 30s for slow embedding batches ([91a2883](https://github.com/ory/lumen/commit/91a2883518bb1f4a289625ce6c3627d2a568f7e0))
* **store:** wrap dimension-reset deletes in a transaction ([33ea835](https://github.com/ory/lumen/commit/33ea835feaf8c49719be9d118c0fb1dceef0ff71))
* **tui:** disable ShowElapsedTime to prevent pterm ticker goroutine race ([cec32c5](https://github.com/ory/lumen/commit/cec32c52e3df45b379bf7e5da6480f769582cf37))

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
