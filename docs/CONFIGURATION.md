# Lumen Configuration

Lumen can be configured with a YAML file and environment variables. The YAML
file is the best choice for persistent multi-server setups; environment
variables are useful for one-off overrides and backwards compatibility.

## Config file location

Lumen reads YAML config from:

- `$XDG_CONFIG_HOME/lumen/config.yaml` when `XDG_CONFIG_HOME` is set
- otherwise `~/.config/lumen/config.yaml`

The MCP server watches the config directory for changes where supported by the
underlying filesystem watcher and reloads when `config.yaml` is written or
created.

## Precedence

Configuration is applied in this order, with later layers overriding earlier
layers:

1. built-in defaults
2. YAML config file
3. environment variables
4. command/programmatic model overrides
5. `--model` / `--backend` server selection filters

## Minimal config

Ollama:

```yaml
servers:
  - backend: ollama
    host: http://localhost:11434
    model: ordis/jina-embeddings-v2-base-code
```

LM Studio:

```yaml
servers:
  - backend: lmstudio
    host: http://localhost:1234
    model: nomic-ai/nomic-embed-code-GGUF
```

## Full example

```yaml
log_level: info
max_chunk_tokens: 512
freshness_ttl: 60s
reindex_timeout: 0s

servers:
  - backend: ollama
    host: http://localhost:11434
    model: ordis/jina-embeddings-v2-base-code
    dims: 768
    ctx_length: 8192
    min_score: 0.35
```

## Top-level fields

| Field | Type | Default | Description |
| --- | --- | --- | --- |
| `log_level` | string | `info` | Logging verbosity. |
| `max_chunk_tokens` | integer | `512` | Maximum estimated tokens per chunk before splitting. |
| `freshness_ttl` | duration | `60s` | How long a confirmed-fresh index is trusted before rechecking. |
| `reindex_timeout` | duration | `0s` | Reindex timeout from config. `0s` means no config-level timeout; command/server code may still apply its own operational safeguards. |
| `servers` | list | one default Ollama server | Embedding backend configurations. |

Durations use Go duration syntax such as `30s`, `5m`, or `1h`.

## Server fields

| Field | Type | Required | Description |
| --- | --- | --- | --- |
| `backend` | string | yes | `ollama` or `lmstudio`. |
| `host` | URL | yes | HTTP(S) base URL for the embedding backend. |
| `model` | string | yes | Embedding model name. |
| `dims` | integer | for unknown models | Embedding vector dimension. Optional for known models. |
| `ctx_length` | integer | no | Embedding model context length. Optional for known models. |
| `min_score` | float | no | Default minimum cosine similarity threshold. |

## Known embedding models

Dimensions, context length, and default minimum score are configured
automatically for known models:

| Model | Backend | Dims | Context | Min score |
| --- | --- | ---: | ---: | ---: |
| `ordis/jina-embeddings-v2-base-code` | `ollama` | 768 | 8192 | 0.35 |
| `nomic-embed-text` | `ollama` | 768 | 8192 | 0.30 |
| `nomic-ai/nomic-embed-code-GGUF` | `lmstudio` | 3584 | 8192 | 0.15 |
| `qwen3-embedding:8b` | `ollama` | 4096 | 40960 | 0.30 |
| `qwen3-embedding:4b` | `ollama` | 2560 | 40960 | 0.30 |
| `qwen3-embedding:0.6b` | `ollama` | 1024 | 32768 | 0.30 |
| `all-minilm` | `ollama` | 384 | 512 | 0.20 |
| `manutic/nomic-embed-code:7b` | `ollama` | 3584 | 32768 | 0.15 |

`text-embedding-nomic-embed-code` is treated as an alias for
`nomic-ai/nomic-embed-code-GGUF`.

Switching models creates a separate index automatically because the model name
is part of the database path hash. The backend is not part of that hash, so use
distinct model names if the same model is served from multiple backends with
incompatible embeddings.

## Environment variable overrides

Environment variables are applied after the YAML config file and before command
or server-selection overrides.

| Environment variable | Overrides |
| --- | --- |
| `LUMEN_MAX_CHUNK_TOKENS` | `max_chunk_tokens` |
| `LUMEN_FRESHNESS_TTL` | `freshness_ttl` |
| `LUMEN_REINDEX_TIMEOUT` | `reindex_timeout` |
| `LUMEN_LOG_LEVEL` | `log_level` |
| `LUMEN_BACKEND` | `servers[0].backend`; resets server 0 to backend defaults first |
| `LUMEN_EMBED_MODEL` | `servers[0].model` |
| `LUMEN_EMBED_DIMS` | `servers[0].dims` |
| `LUMEN_EMBED_CTX` | `servers[0].ctx_length` |
| `OLLAMA_HOST` | `servers[0].host` when server 0 backend is `ollama` |
| `LM_STUDIO_HOST` | `servers[0].host` when server 0 backend is `lmstudio` |

Environment variables only modify `servers[0]`. They do not rewrite every
server in a multi-server config.

## Selecting a server with CLI flags

`lumen index` and `lumen search` accept `--model` / `-m` and `--backend` / `-b`
to select from the configured server list:

```bash
lumen index --model ordis/jina-embeddings-v2-base-code .
lumen search --backend ollama "authentication flow"
lumen index --model my-embed --backend lmstudio .
```

`--model` and `--backend` filter the configured server list. If multiple servers
match, order is preserved for failover. If no servers match, Lumen returns a
descriptive error that includes the configured `(backend, model)` pairs.

If `--model` is not configured in YAML but is a known registry model and
`--backend` is unset, Lumen falls back to overriding the default server's model.
That preserves legacy commands such as:

```bash
lumen index --model all-minilm .
```

## Multi-server and failover examples

```yaml
servers:
  - backend: ollama
    host: http://localhost:11434
    model: ordis/jina-embeddings-v2-base-code
  - backend: ollama
    host: http://backup-ollama.example:11434
    model: ordis/jina-embeddings-v2-base-code
  - backend: lmstudio
    host: http://localhost:1234
    model: nomic-ai/nomic-embed-code-GGUF
```

When more than one configured server matches the selected backend/model, Lumen
keeps the configured order and can fail over within that filtered subset.

## Unknown/custom models

If a model is not in the known model table or alias map, set `dims` explicitly:

```yaml
servers:
  - backend: ollama
    host: http://localhost:11434
    model: my-custom-embedding-model
    dims: 1024
    ctx_length: 8192
    min_score: 0.20
```

`ctx_length` and `min_score` are optional for custom models. If `min_score` is
omitted, Lumen derives a dimension-aware default from `dims`.

## Validation errors

Lumen validates configuration before using it. Common invalid configs include:

- empty `servers`
- missing `backend`
- unknown `backend`
- missing `host`
- invalid `host` URL, or a URL that is not `http://` or `https://`
- missing `model`
- unknown model with no explicit `dims`

## MCP/server behavior

The stdio MCP server loads the same configuration and uses the same precedence
rules. When watching is available, it watches the config directory and reloads
when the configured `config.yaml` file is written or created.

Agent hosts and plugin wrappers may add their own environment variables before
starting Lumen. Prefer YAML for stable multi-server setups, and use environment
variables for per-session overrides.
