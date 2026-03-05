# Fixture Sources

All files in this directory are vendored from open-source projects for use as
E2E test fixtures. They are used solely for testing purposes. Files must be
exact copies from upstream — never hand-edit or simplify vendored code.

## Go

- **prometheus/prometheus** (Apache 2.0) —
  https://github.com/prometheus/prometheus
  - Branch: `main` (commit `7a9c0577`, 2026-02-24)
  - `tsdb/block.go` → `go/block.go`
  - Other files: `tsdb/compact.go`, `tsdb/db.go`, `tsdb/head.go`, etc.
- **prometheus/client_golang** (Apache 2.0) —
  https://github.com/prometheus/client_golang

## Java

- **spring-projects/spring-petclinic** (Apache 2.0) —
  https://github.com/spring-projects/spring-petclinic
  - Branch: `main`
  - `src/main/java/.../model/Person.java` → `java/Person.java`
  - `src/main/java/.../model/BaseEntity.java` → `java/BaseEntity.java`
  - `src/main/java/.../model/NamedEntity.java` → `java/NamedEntity.java`
  - `src/main/java/.../owner/PetTypeRepository.java` →
    `java/PetTypeRepository.java`

## PHP

- **laravel/framework** (MIT) — https://github.com/laravel/framework
  - Branch: `master`
  - `src/Illuminate/Container/ContextualBindingBuilder.php` →
    `php/ContextualBindingBuilder.php`
  - `src/Illuminate/Support/ServiceProvider.php` → `php/ServiceProvider.php`

## JavaScript

- **expressjs/express** (MIT) — https://github.com/expressjs/express
- **fastify/fastify** (MIT) — https://github.com/fastify/fastify
- **nodejs/node** (MIT) — https://github.com/nodejs/node

## TypeScript

- **microsoft/vscode** (MIT) — https://github.com/microsoft/vscode

## Ruby

- **rails/rails** (MIT) — https://github.com/rails/rails
- **sinatra/sinatra** (MIT) — https://github.com/sinatra/sinatra

## Python

- **pallets/flask** (BSD-3-Clause) — https://github.com/pallets/flask
- **django/django** (BSD-3-Clause) — https://github.com/django/django
  - Branch: `main` (commit `36be97b9`, 2026-03-04)
  - `django/db/models/query_utils.py` → `python/django-q.py`
  - `django/db/models/sql/query.py` → `python/django-sql-query.py`

## Rust

- **BurntSushi/ripgrep** (MIT/Unlicense) — https://github.com/BurntSushi/ripgrep
- **tokio-rs/tokio** (MIT) — https://github.com/tokio-rs/tokio

## YAML

- **kubernetes/website** (Apache 2.0) — https://github.com/kubernetes/website
  (example manifests)
- **docker/awesome-compose** (Apache 2.0) —
  https://github.com/docker/awesome-compose
- **prometheus-community/helm-charts** (Apache 2.0) —
  https://github.com/prometheus-community/helm-charts
- Various GH Actions workflow files from the above projects (ci.yml, tests.yml,
  etc.)
- Hand-authored standard Kubernetes manifest examples (Deployment, Service,
  StatefulSet, Ingress, ConfigMap)

## Markdown

- **rust-lang/book** (MIT/Apache 2.0) — https://github.com/rust-lang/book

## JSON

- **microsoft/TypeScript** (Apache 2.0) —
  https://github.com/microsoft/TypeScript
- **microsoft/vscode** (MIT) — https://github.com/microsoft/vscode
- Various package.json files from the above projects (expressjs/express,
  facebook/react, eslint/eslint, prettier/prettier)
