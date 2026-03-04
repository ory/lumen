# Changelog

## [0.1.0](https://github.com/ory/lumen/compare/v0.0.1...v0.1.0) (2026-03-04)


### ⚠ BREAKING CHANGES

* `lumen install` and `lumen uninstall` commands removed. Use `claude plugin install lumen` or `claude --plugin-dir .` instead.

### Features

* add go/ast chunker for function/method/type/interface/const/var extraction ([8efb510](https://github.com/ory/lumen/commit/8efb510bf99e6035f2e290e882cfd16d7f1e7ff1))
* add index orchestrator with merkle-based incremental indexing ([da1fa90](https://github.com/ory/lumen/commit/da1fa90481382c31d5221492400da00a4a1105b6))
* add internal/config package for shared configuration ([234ca1e](https://github.com/ory/lumen/commit/234ca1ebc5476d2ce844496a2905cc3235753de3))
* add Markdown/MDX chunker splitting by ATX headings ([81863f8](https://github.com/ory/lumen/commit/81863f8f9fd81b05dcec12749ffa452e6aa5f159))
* add merkle tree change detection for file hashing and diffing ([05de922](https://github.com/ory/lumen/commit/05de922ac51acbe86adf22c7df93da6437440b44))
* add model registry mapping Ollama model names to specs ([332e51e](https://github.com/ory/lumen/commit/332e51eb5336c94d8401d099e2e8bcc5dfcb1163))
* add MultiChunker and DefaultLanguages for all supported file types ([4c9d237](https://github.com/ory/lumen/commit/4c9d23798c921c4b8b3631e32f183af4d74e6a69))
* add ollama embedder with batching and retry ([a489800](https://github.com/ory/lumen/commit/a48980075656fa59f13400c501380b8e9ed62586))
* add PHP tree-sitter chunker ([f109b8b](https://github.com/ory/lumen/commit/f109b8b5cadfbcc5e29bb0eb1aa948a9f0889cf7))
* add sqlite + sqlite-vec store with vector search and batch inserts ([e12bb2d](https://github.com/ory/lumen/commit/e12bb2dc8f309ae6cb36865b4c39a367959dee89))
* add StructuredChunker for YAML/JSON files with TDD ([7d78504](https://github.com/ory/lumen/commit/7d7850402fd47fc37dc7f2944c2318b66b0f0487))
* add TreeSitterChunker for multi-language AST parsing ([cf25133](https://github.com/ory/lumen/commit/cf2513345550b9dc7e0caa67d33f8433a13c36b0))
* add YAML+JSON data chunker splitting by top-level keys ([466e9fd](https://github.com/ory/lumen/commit/466e9fd5302035183d46a171be595965a2e590bf))
* auto-reset vec_chunks on embedding dimension mismatch ([ff3279a](https://github.com/ory/lumen/commit/ff3279a97ac48525397597795e4ac43f4a1d4a39))
* change default limit to 50, default min_score to 0.5; handle min_score=-1 as no-filter ([becc603](https://github.com/ory/lumen/commit/becc60350cfe047fb8dacbd15831ced15569a518))
* go-ast + sqlite-vec + mcp server ([8b13f64](https://github.com/ory/lumen/commit/8b13f645c6adadf2462a6af9c14aae4dbe8cd8ba))
* ignore known vendor dirs and gitignore ([67668f1](https://github.com/ory/lumen/commit/67668f177fe54e2bdfe379f30dd32f202d6810df))
* **install:** show only supported models with local availability status ([6d5a5a0](https://github.com/ory/lumen/commit/6d5a5a0c59cbd6e3f9f6a14ac462ca6614ed2a89))
* LM Studio backend, chunk deduplication, and quality improvements ([0bd3fe3](https://github.com/ory/lumen/commit/0bd3fe37aa3e87034719b274cfd33adebb060ad4))
* make embedding dimensions configurable via AGENT_INDEX_EMBED_DIMS ([87944cd](https://github.com/ory/lumen/commit/87944cd52fe17b7351325d873dd377cc3fbc88dc))
* pass num_ctx to Ollama based on model's context length ([d5a4540](https://github.com/ory/lumen/commit/d5a4540c87aa17a1b58d4a667c52b11143ae1241))
* replace key-based DataChunker with plain text splitting for JSON and YAML ([4cff686](https://github.com/ory/lumen/commit/4cff6864d39e711d2ece96cf1f9933f0b5c360a7))
* split oversized chunks at line boundaries before embedding ([cae8a90](https://github.com/ory/lumen/commit/cae8a908fb223321ecffbb06c102dadee9b090c0))
* **stdio:** group search results by file with score-based ranking ([967e23e](https://github.com/ory/lumen/commit/967e23e4bb4303e4e916ab2d5e529539ba4535f1))
* switch default model to qwen3-embedding:8b, rename module ([0df840a](https://github.com/ory/lumen/commit/0df840a83a769e2731568f38e5d02a5e6693ef65))
* use XML-tagged output format for search results ([5e34a89](https://github.com/ory/lumen/commit/5e34a8971a4dc6544da0d6605412a19ccb1d5d6c))
* wire MCP server with semantic_search and index_status tools ([a57bebc](https://github.com/ory/lumen/commit/a57bebcb642343d2aeea659f041a4396934ff105))
* wire MultiChunker and MakeExtSkip for multi-language indexing ([0caed1f](https://github.com/ory/lumen/commit/0caed1f1db6d4cf9ac97f338854dad7d148bc7f0))
* wire StructuredChunker into DefaultLanguages for yaml/yml/json files ([91ebc82](https://github.com/ory/lumen/commit/91ebc8234f51f431cedacca69879613b7b1cb4f3))


### Bug Fixes

* broken json ([5d4496e](https://github.com/ory/lumen/commit/5d4496e8599b2cf915cf4942b371a902449cb768))
* **build:** fix cross-platform build on Apple Silicon ([597821d](https://github.com/ory/lumen/commit/597821d211ace2c0de8a88124a116ea13170e784))
* C pointer-return query; add test coverage for all languages ([4446d1f](https://github.com/ory/lumen/commit/4446d1fe6b74a5ff82e71f5e61a5995b014c8299))
* convert cosine distance to similarity score, explicit ORDER BY score DESC ([85bc4eb](https://github.com/ory/lumen/commit/85bc4ebb8460682287ddea2bd5fa05be6cb0375c))
* explicitly ignore f.Close() error in parseLinguistGenerated ([f95ca35](https://github.com/ory/lumen/commit/f95ca352065ced6fcb70bd1a35b519cf1ec887e0))
* fix 5 pre-existing E2E test failures ([64377cc](https://github.com/ory/lumen/commit/64377cc21e1975346371a7708ef892e58fdb88d0))
* **plugin:** align plugin files with Claude Code plugin reference ([89cdb08](https://github.com/ory/lumen/commit/89cdb081e65ac023e056f90ad4b48f5ebf837840))
* reduce cyclomatic complexity of major functions ([8faef25](https://github.com/ory/lumen/commit/8faef25c3e4fece1f09da3d2c88998cb2dd863c3))
* remove package-level chunk from Go AST chunker ([e53d95b](https://github.com/ory/lumen/commit/e53d95bf02b92e8c9e7cb6ac44c693f329e248c2))
* replace custom minInt with builtin min in structured_test.go ([8aa1269](https://github.com/ory/lumen/commit/8aa126950c7bcd320a9c6a6706bad7952d0685b5))
* replace custom time.Sleep backoff with go-retry for context-aware retries ([91a14c1](https://github.com/ory/lumen/commit/91a14c184874abd6bef944e61ee4b6588632e59c))
* resolve import and style issues in merkle package ([fd8c1a8](https://github.com/ory/lumen/commit/fd8c1a85ac6157c86cb967da8bf52e450e73069d))
* **scripts:** point to ory/lumen and download raw binaries ([fef815d](https://github.com/ory/lumen/commit/fef815d15180e8bfac86f09f0140cf8c5337afe8))
* set MaxOpenConns(1) for SQLite, fix pragma test to scan int ([7f82484](https://github.com/ory/lumen/commit/7f82484d59a75e0d3b61b3b384c98c28a1ef3529))
* tool description to increase invokation ([5f92ca4](https://github.com/ory/lumen/commit/5f92ca4f6e6b3ff5217cb6fced5f5455757fd635))
* update MinScoreFilter test to use -1 for all-results and 0.5 threshold ([562e474](https://github.com/ory/lumen/commit/562e474093af1f3c701c0fdc5f5607f404c18744))


### Performance Improvements

* add SQLite write pragmas and chunk column indexes ([5bac05f](https://github.com/ory/lumen/commit/5bac05f1fdbef5a17c27a6d1bffd3ea5be0c01ae))
* avoid double Merkle tree build by sharing tree between EnsureFresh and indexWithTree ([91f3cae](https://github.com/ory/lumen/commit/91f3cae471d962f8b002fc74389b993d1c31a262))
* combine Stats() into one query, add GetMetaBatch for Status() ([80300f9](https://github.com/ory/lumen/commit/80300f9c300262a2c5890cc2cf4f3bf1d12e15c1))
* make Status() DB-only by storing total_files in metadata, remove StaleFiles ([59826c2](https://github.com/ory/lumen/commit/59826c2b7d6184000eab94ae4ebc10497a30d785))
* parallelize file reads in merkle.BuildTree with worker pool ([02e262e](https://github.com/ory/lumen/commit/02e262e9397e604955def1409611d0d062a23192))
* stream chunk embed+insert in batches of 256 to bound memory ([7441642](https://github.com/ory/lumen/commit/7441642c59f688dfff06101cb2becd52430b78c9))
* use RWMutex with double-checked locking in indexer cache ([5d1c2b6](https://github.com/ory/lumen/commit/5d1c2b61069792d6cf3a44edff1cd0e0cfd6ade2))


### Code Refactoring

* migrate from install/uninstall to Claude Code plugin system ([0920f3d](https://github.com/ory/lumen/commit/0920f3d33f64ec836a9e5b2dc9bef570393524d4))
