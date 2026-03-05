#!/usr/bin/env bash
# reset-fixtures.sh — re-vendor ALL fixture files from upstream repos
# Ensures every file in testdata/fixtures/ is an exact copy from upstream.
set -euo pipefail

REPO="$(cd "$(dirname "$0")/.." && pwd)"
FIXTURES="$REPO/testdata/fixtures"
TMPDIR=$(mktemp -d)
trap 'rm -rf "$TMPDIR"' EXIT

# Counters
total=0
updated=0
failed=0
unchanged=0

fetch() {
  local url="$1" dest="$2" label="$3"
  total=$((total + 1))
  printf "  %-50s " "$label"

  if curl -sfL --max-time 30 "$url" -o "$TMPDIR/fetched" 2>/dev/null; then
    if [[ -f "$dest" ]] && diff -q "$TMPDIR/fetched" "$dest" >/dev/null 2>&1; then
      echo "OK (unchanged)"
      unchanged=$((unchanged + 1))
    else
      cp "$TMPDIR/fetched" "$dest"
      echo "UPDATED"
      updated=$((updated + 1))
    fi
  else
    echo "FAILED (fetch error)"
    failed=$((failed + 1))
  fi
  rm -f "$TMPDIR/fetched"
}

# ── Pinned commits ────────────────────────────────────────────────────────────
# Pin to specific commits for reproducibility. Update these when re-vendoring.
PROMETHEUS_SHA="main"
CLIENT_GOLANG_SHA="main"
DJANGO_SHA="main"
FLASK_SHA="main"
PETCLINIC_SHA="main"
LARAVEL_SHA="master"
EXPRESS_SHA="master"
FASTIFY_SHA="main"
NODE_SHA="main"
VSCODE_SHA="main"
RAILS_SHA="main"
SINATRA_SHA="main"
RIPGREP_SHA="master"
TOKIO_SHA="master"

GH="https://raw.githubusercontent.com"

echo "=== Go (prometheus/prometheus + client_golang) ==="
# prometheus/prometheus files — mapped by package declaration
fetch "$GH/prometheus/prometheus/$PROMETHEUS_SHA/rules/alerting.go" \
  "$FIXTURES/go/alerting.go" "go/alerting.go"
fetch "$GH/prometheus/prometheus/$PROMETHEUS_SHA/web/api/v1/api.go" \
  "$FIXTURES/go/api.go" "go/api.go"
fetch "$GH/prometheus/prometheus/$PROMETHEUS_SHA/promql/parser/ast.go" \
  "$FIXTURES/go/ast.go" "go/ast.go"
fetch "$GH/prometheus/prometheus/$PROMETHEUS_SHA/tsdb/block.go" \
  "$FIXTURES/go/block.go" "go/block.go"
fetch "$GH/prometheus/prometheus/$PROMETHEUS_SHA/tsdb/compact.go" \
  "$FIXTURES/go/compact.go" "go/compact.go"
fetch "$GH/prometheus/prometheus/$PROMETHEUS_SHA/config/config.go" \
  "$FIXTURES/go/config.go" "go/config.go"
fetch "$GH/prometheus/prometheus/$PROMETHEUS_SHA/tsdb/db.go" \
  "$FIXTURES/go/db.go" "go/db.go"
fetch "$GH/prometheus/prometheus/$PROMETHEUS_SHA/promql/engine.go" \
  "$FIXTURES/go/engine.go" "go/engine.go"
fetch "$GH/prometheus/prometheus/$PROMETHEUS_SHA/promql/functions.go" \
  "$FIXTURES/go/functions.go" "go/functions.go"
fetch "$GH/prometheus/prometheus/$PROMETHEUS_SHA/tsdb/head.go" \
  "$FIXTURES/go/head.go" "go/head.go"
fetch "$GH/prometheus/prometheus/$PROMETHEUS_SHA/model/histogram/histogram.go" \
  "$FIXTURES/go/histogram.go" "go/histogram.go"
fetch "$GH/prometheus/prometheus/$PROMETHEUS_SHA/storage/interface.go" \
  "$FIXTURES/go/interface.go" "go/interface.go"
fetch "$GH/prometheus/prometheus/$PROMETHEUS_SHA/model/labels/labels_common.go" \
  "$FIXTURES/go/labels_common.go" "go/labels_common.go"
fetch "$GH/prometheus/prometheus/$PROMETHEUS_SHA/model/labels/matcher.go" \
  "$FIXTURES/go/labels_matcher.go" "go/labels_matcher.go"
fetch "$GH/prometheus/prometheus/$PROMETHEUS_SHA/model/labels/regexp.go" \
  "$FIXTURES/go/labels_regexp.go" "go/labels_regexp.go"
fetch "$GH/prometheus/prometheus/$PROMETHEUS_SHA/model/labels/labels_stringlabels.go" \
  "$FIXTURES/go/labels_stringlabels.go" "go/labels_stringlabels.go"
fetch "$GH/prometheus/prometheus/$PROMETHEUS_SHA/scrape/manager.go" \
  "$FIXTURES/go/manager.go" "go/manager.go"
fetch "$GH/prometheus/prometheus/$PROMETHEUS_SHA/model/labels/matcher.go" \
  "$FIXTURES/go/matcher.go" "go/matcher.go"
fetch "$GH/prometheus/prometheus/$PROMETHEUS_SHA/storage/merge.go" \
  "$FIXTURES/go/merge.go" "go/merge.go"
fetch "$GH/prometheus/prometheus/$PROMETHEUS_SHA/model/value/value.go" \
  "$FIXTURES/go/model_value.go" "go/model_value.go"
fetch "$GH/prometheus/prometheus/$PROMETHEUS_SHA/rules/recording.go" \
  "$FIXTURES/go/recording.go" "go/recording.go"
fetch "$GH/prometheus/prometheus/$PROMETHEUS_SHA/model/relabel/relabel.go" \
  "$FIXTURES/go/relabel.go" "go/relabel.go"
fetch "$GH/prometheus/prometheus/$PROMETHEUS_SHA/scrape/scrape.go" \
  "$FIXTURES/go/scrape.go" "go/scrape.go"
fetch "$GH/prometheus/prometheus/$PROMETHEUS_SHA/util/teststorage/storage.go" \
  "$FIXTURES/go/teststorage.go" "go/teststorage.go"
fetch "$GH/prometheus/prometheus/$PROMETHEUS_SHA/model/textparse/interface.go" \
  "$FIXTURES/go/textparse_interface.go" "go/textparse_interface.go"
fetch "$GH/prometheus/prometheus/$PROMETHEUS_SHA/model/timestamp/timestamp.go" \
  "$FIXTURES/go/timestamp.go" "go/timestamp.go"
fetch "$GH/prometheus/prometheus/$PROMETHEUS_SHA/web/ui/ui.go" \
  "$FIXTURES/go/ui.go" "go/ui.go"
fetch "$GH/prometheus/prometheus/$PROMETHEUS_SHA/promql/value.go" \
  "$FIXTURES/go/value.go" "go/value.go"

# prometheus/client_golang (prom_* files)
fetch "$GH/prometheus/client_golang/$CLIENT_GOLANG_SHA/prometheus/counter.go" \
  "$FIXTURES/go/prom_counter.go" "go/prom_counter.go"
fetch "$GH/prometheus/client_golang/$CLIENT_GOLANG_SHA/prometheus/gauge.go" \
  "$FIXTURES/go/prom_gauge.go" "go/prom_gauge.go"
fetch "$GH/prometheus/client_golang/$CLIENT_GOLANG_SHA/prometheus/histogram.go" \
  "$FIXTURES/go/prom_histogram.go" "go/prom_histogram.go"
fetch "$GH/prometheus/client_golang/$CLIENT_GOLANG_SHA/prometheus/registry.go" \
  "$FIXTURES/go/prom_registry.go" "go/prom_registry.go"

echo ""
echo "=== Java (spring-projects/spring-petclinic) ==="
PETCLINIC_BASE="$GH/spring-projects/spring-petclinic/$PETCLINIC_SHA/src/main/java/org/springframework/samples/petclinic"
fetch "$PETCLINIC_BASE/model/BaseEntity.java" \
  "$FIXTURES/java/BaseEntity.java" "java/BaseEntity.java"
fetch "$PETCLINIC_BASE/system/CacheConfiguration.java" \
  "$FIXTURES/java/CacheConfiguration.java" "java/CacheConfiguration.java"
fetch "$PETCLINIC_BASE/system/CrashController.java" \
  "$FIXTURES/java/CrashController.java" "java/CrashController.java"
fetch "$PETCLINIC_BASE/model/NamedEntity.java" \
  "$FIXTURES/java/NamedEntity.java" "java/NamedEntity.java"
fetch "$PETCLINIC_BASE/owner/Owner.java" \
  "$FIXTURES/java/Owner.java" "java/Owner.java"
fetch "$PETCLINIC_BASE/owner/OwnerController.java" \
  "$FIXTURES/java/OwnerController.java" "java/OwnerController.java"
fetch "$PETCLINIC_BASE/owner/OwnerRepository.java" \
  "$FIXTURES/java/OwnerRepository.java" "java/OwnerRepository.java"
fetch "$PETCLINIC_BASE/model/Person.java" \
  "$FIXTURES/java/Person.java" "java/Person.java"
fetch "$PETCLINIC_BASE/owner/Pet.java" \
  "$FIXTURES/java/Pet.java" "java/Pet.java"
fetch "$PETCLINIC_BASE/PetClinicApplication.java" \
  "$FIXTURES/java/PetClinicApplication.java" "java/PetClinicApplication.java"
fetch "$PETCLINIC_BASE/owner/PetController.java" \
  "$FIXTURES/java/PetController.java" "java/PetController.java"
fetch "$PETCLINIC_BASE/owner/PetType.java" \
  "$FIXTURES/java/PetType.java" "java/PetType.java"
fetch "$PETCLINIC_BASE/owner/PetTypeRepository.java" \
  "$FIXTURES/java/PetTypeRepository.java" "java/PetTypeRepository.java"
fetch "$PETCLINIC_BASE/vet/Specialty.java" \
  "$FIXTURES/java/Specialty.java" "java/Specialty.java"
fetch "$PETCLINIC_BASE/vet/Vet.java" \
  "$FIXTURES/java/Vet.java" "java/Vet.java"
fetch "$PETCLINIC_BASE/vet/VetController.java" \
  "$FIXTURES/java/VetController.java" "java/VetController.java"
fetch "$PETCLINIC_BASE/vet/VetRepository.java" \
  "$FIXTURES/java/VetRepository.java" "java/VetRepository.java"
fetch "$PETCLINIC_BASE/vet/Vets.java" \
  "$FIXTURES/java/Vets.java" "java/Vets.java"
fetch "$PETCLINIC_BASE/owner/Visit.java" \
  "$FIXTURES/java/Visit.java" "java/Visit.java"
fetch "$PETCLINIC_BASE/owner/VisitController.java" \
  "$FIXTURES/java/VisitController.java" "java/VisitController.java"

echo ""
echo "=== PHP (laravel/framework) ==="
LARAVEL_BASE="$GH/laravel/framework/$LARAVEL_SHA/src/Illuminate"
fetch "$LARAVEL_BASE/Auth/AuthManager.php" \
  "$FIXTURES/php/AuthManager.php" "php/AuthManager.php"
fetch "$LARAVEL_BASE/Database/Eloquent/Relations/BelongsTo.php" \
  "$FIXTURES/php/BelongsTo.php" "php/BelongsTo.php"
fetch "$LARAVEL_BASE/Database/Eloquent/Builder.php" \
  "$FIXTURES/php/Builder.php" "php/Builder.php"
fetch "$LARAVEL_BASE/Cache/CacheManager.php" \
  "$FIXTURES/php/CacheManager.php" "php/CacheManager.php"
fetch "$LARAVEL_BASE/Database/Connection.php" \
  "$FIXTURES/php/Connection.php" "php/Connection.php"
fetch "$LARAVEL_BASE/Container/Container.php" \
  "$FIXTURES/php/Container.php" "php/Container.php"
fetch "$LARAVEL_BASE/Container/ContextualBindingBuilder.php" \
  "$FIXTURES/php/ContextualBindingBuilder.php" "php/ContextualBindingBuilder.php"
fetch "$LARAVEL_BASE/Events/Dispatcher.php" \
  "$FIXTURES/php/Dispatcher.php" "php/Dispatcher.php"
fetch "$LARAVEL_BASE/Database/Eloquent/Relations/HasMany.php" \
  "$FIXTURES/php/HasMany.php" "php/HasMany.php"
fetch "$LARAVEL_BASE/Log/Logger.php" \
  "$FIXTURES/php/Logger.php" "php/Logger.php"
fetch "$LARAVEL_BASE/Database/Eloquent/Model.php" \
  "$FIXTURES/php/Model.php" "php/Model.php"
fetch "$LARAVEL_BASE/Pipeline/Pipeline.php" \
  "$FIXTURES/php/Pipeline.php" "php/Pipeline.php"
fetch "$LARAVEL_BASE/Queue/QueueManager.php" \
  "$FIXTURES/php/QueueManager.php" "php/QueueManager.php"
fetch "$LARAVEL_BASE/Cache/Repository.php" \
  "$FIXTURES/php/Repository.php" "php/Repository.php"
fetch "$LARAVEL_BASE/Http/Request.php" \
  "$FIXTURES/php/Request.php" "php/Request.php"
fetch "$LARAVEL_BASE/Http/Response.php" \
  "$FIXTURES/php/Response.php" "php/Response.php"
fetch "$LARAVEL_BASE/Routing/Route.php" \
  "$FIXTURES/php/Route.php" "php/Route.php"
fetch "$LARAVEL_BASE/Routing/Router.php" \
  "$FIXTURES/php/Router.php" "php/Router.php"
fetch "$LARAVEL_BASE/Support/ServiceProvider.php" \
  "$FIXTURES/php/ServiceProvider.php" "php/ServiceProvider.php"
fetch "$LARAVEL_BASE/Session/Store.php" \
  "$FIXTURES/php/Store.php" "php/Store.php"
fetch "$LARAVEL_BASE/Support/Str.php" \
  "$FIXTURES/php/Str.php" "php/Str.php"
fetch "$LARAVEL_BASE/Validation/Validator.php" \
  "$FIXTURES/php/Validator.php" "php/Validator.php"
fetch "$LARAVEL_BASE/Queue/Worker.php" \
  "$FIXTURES/php/Worker.php" "php/Worker.php"

echo ""
echo "=== Python (django/django + pallets/flask) ==="
DJANGO_BASE="$GH/django/django/$DJANGO_SHA/django"
fetch "$DJANGO_BASE/db/backends/base/base.py" \
  "$FIXTURES/python/django-backends.py" "python/django-backends.py"
fetch "$DJANGO_BASE/views/generic/base.py" \
  "$FIXTURES/python/django-base.py" "python/django-base.py"
fetch "$DJANGO_BASE/views/generic/dates.py" \
  "$FIXTURES/python/django-common.py" "python/django-common.py"
fetch "$DJANGO_BASE/views/generic/detail.py" \
  "$FIXTURES/python/django-detail.py" "python/django-detail.py"
fetch "$DJANGO_BASE/core/exceptions.py" \
  "$FIXTURES/python/django-exceptions.py" "python/django-exceptions.py"
fetch "$DJANGO_BASE/views/generic/list.py" \
  "$FIXTURES/python/django-list.py" "python/django-list.py"
fetch "$DJANGO_BASE/db/models/manager.py" \
  "$FIXTURES/python/django-manager.py" "python/django-manager.py"
fetch "$DJANGO_BASE/db/models/base.py" \
  "$FIXTURES/python/django-models.py" "python/django-models.py"
fetch "$DJANGO_BASE/db/models/query_utils.py" \
  "$FIXTURES/python/django-q.py" "python/django-q.py"
fetch "$DJANGO_BASE/db/models/query.py" \
  "$FIXTURES/python/django-query.py" "python/django-query.py"
fetch "$DJANGO_BASE/http/request.py" \
  "$FIXTURES/python/django-request.py" "python/django-request.py"
fetch "$DJANGO_BASE/urls/resolvers.py" \
  "$FIXTURES/python/django-resolvers.py" "python/django-resolvers.py"
fetch "$DJANGO_BASE/http/response.py" \
  "$FIXTURES/python/django-response.py" "python/django-response.py"
fetch "$DJANGO_BASE/db/models/sql/query.py" \
  "$FIXTURES/python/django-sql-query.py" "python/django-sql-query.py"
fetch "$DJANGO_BASE/views/generic/edit.py" \
  "$FIXTURES/python/django-views.py" "python/django-views.py"

FLASK_BASE="$GH/pallets/flask/$FLASK_SHA/src/flask"
fetch "$FLASK_BASE/app.py" \
  "$FIXTURES/python/flask-app.py" "python/flask-app.py"
fetch "$FLASK_BASE/blueprints.py" \
  "$FIXTURES/python/flask-blueprints.py" "python/flask-blueprints.py"
fetch "$FLASK_BASE/cli.py" \
  "$FIXTURES/python/flask-cli.py" "python/flask-cli.py"
fetch "$FLASK_BASE/config.py" \
  "$FIXTURES/python/flask-config.py" "python/flask-config.py"
fetch "$FLASK_BASE/ctx.py" \
  "$FIXTURES/python/flask-ctx.py" "python/flask-ctx.py"
fetch "$FLASK_BASE/helpers.py" \
  "$FIXTURES/python/flask-helpers.py" "python/flask-helpers.py"
fetch "$FLASK_BASE/sessions.py" \
  "$FIXTURES/python/flask-sessions.py" "python/flask-sessions.py"
fetch "$FLASK_BASE/views.py" \
  "$FIXTURES/python/flask-views.py" "python/flask-views.py"
fetch "$FLASK_BASE/wrappers.py" \
  "$FIXTURES/python/flask-wrappers.py" "python/flask-wrappers.py"

echo ""
echo "=== JavaScript (expressjs/express + fastify + nodejs/node) ==="
fetch "$GH/expressjs/express/$EXPRESS_SHA/lib/application.js" \
  "$FIXTURES/js/express-application.js" "js/express-application.js"
fetch "$GH/expressjs/express/$EXPRESS_SHA/lib/express.js" \
  "$FIXTURES/js/express-express.js" "js/express-express.js"
fetch "$GH/expressjs/express/$EXPRESS_SHA/lib/request.js" \
  "$FIXTURES/js/express-request.js" "js/express-request.js"
fetch "$GH/expressjs/express/$EXPRESS_SHA/lib/response.js" \
  "$FIXTURES/js/express-response.js" "js/express-response.js"
fetch "$GH/expressjs/express/$EXPRESS_SHA/lib/utils.js" \
  "$FIXTURES/js/express-utils.js" "js/express-utils.js"
fetch "$GH/expressjs/express/$EXPRESS_SHA/lib/view.js" \
  "$FIXTURES/js/express-view.js" "js/express-view.js"

fetch "$GH/fastify/fastify/$FASTIFY_SHA/lib/reply.js" \
  "$FIXTURES/js/fastify-reply.js" "js/fastify-reply.js"
fetch "$GH/fastify/fastify/$FASTIFY_SHA/lib/request.js" \
  "$FIXTURES/js/fastify-request.js" "js/fastify-request.js"
fetch "$GH/fastify/fastify/$FASTIFY_SHA/lib/route.js" \
  "$FIXTURES/js/fastify-route.js" "js/fastify-route.js"
fetch "$GH/fastify/fastify/$FASTIFY_SHA/lib/server.js" \
  "$FIXTURES/js/fastify-server.js" "js/fastify-server.js"
fetch "$GH/fastify/fastify/$FASTIFY_SHA/fastify.js" \
  "$FIXTURES/js/fastify.js" "js/fastify.js"

fetch "$GH/nodejs/node/$NODE_SHA/lib/buffer.js" \
  "$FIXTURES/js/node-buffer.js" "js/node-buffer.js"
fetch "$GH/nodejs/node/$NODE_SHA/lib/events.js" \
  "$FIXTURES/js/node-events.js" "js/node-events.js"
fetch "$GH/nodejs/node/$NODE_SHA/lib/http.js" \
  "$FIXTURES/js/node-http.js" "js/node-http.js"
fetch "$GH/nodejs/node/$NODE_SHA/lib/https.js" \
  "$FIXTURES/js/node-https.js" "js/node-https.js"
fetch "$GH/nodejs/node/$NODE_SHA/lib/net.js" \
  "$FIXTURES/js/node-net.js" "js/node-net.js"
fetch "$GH/nodejs/node/$NODE_SHA/lib/os.js" \
  "$FIXTURES/js/node-os.js" "js/node-os.js"
fetch "$GH/nodejs/node/$NODE_SHA/lib/path.js" \
  "$FIXTURES/js/node-path.js" "js/node-path.js"
fetch "$GH/nodejs/node/$NODE_SHA/lib/url.js" \
  "$FIXTURES/js/node-url.js" "js/node-url.js"
fetch "$GH/nodejs/node/$NODE_SHA/lib/util.js" \
  "$FIXTURES/js/node-util.js" "js/node-util.js"

echo ""
echo "=== TypeScript (microsoft/vscode) ==="
VSCODE_BASE="$GH/microsoft/vscode/$VSCODE_SHA/src/vs/base/common"
for tsfile in arrays async cancellation collections color errors event glob hash iterator json lazy lifecycle map network objects path platform resources stream strings types uri uuid; do
  fetch "$VSCODE_BASE/$tsfile.ts" \
    "$FIXTURES/ts/$tsfile.ts" "ts/$tsfile.ts"
done

echo ""
echo "=== Ruby (rails/rails + sinatra/sinatra) ==="
fetch "$GH/rails/rails/$RAILS_SHA/railties/lib/rails/application.rb" \
  "$FIXTURES/ruby/application.rb" "ruby/application.rb"
fetch "$GH/rails/rails/$RAILS_SHA/activerecord/lib/active_record/associations.rb" \
  "$FIXTURES/ruby/associations.rb" "ruby/associations.rb"
fetch "$GH/rails/rails/$RAILS_SHA/actionpack/lib/action_controller/base.rb" \
  "$FIXTURES/ruby/base.rb" "ruby/base.rb"
fetch "$GH/rails/rails/$RAILS_SHA/activesupport/lib/active_support/cache.rb" \
  "$FIXTURES/ruby/cache.rb" "ruby/cache.rb"
fetch "$GH/rails/rails/$RAILS_SHA/activerecord/lib/active_record/callbacks.rb" \
  "$FIXTURES/ruby/callbacks.rb" "ruby/callbacks.rb"
fetch "$GH/rails/rails/$RAILS_SHA/activesupport/lib/active_support/concern.rb" \
  "$FIXTURES/ruby/concern.rb" "ruby/concern.rb"
fetch "$GH/rails/rails/$RAILS_SHA/railties/lib/rails/engine.rb" \
  "$FIXTURES/ruby/engine.rb" "ruby/engine.rb"
fetch "$GH/rails/rails/$RAILS_SHA/actionpack/lib/action_controller/metal.rb" \
  "$FIXTURES/ruby/metal.rb" "ruby/metal.rb"
fetch "$GH/rails/rails/$RAILS_SHA/activesupport/lib/active_support/notifications.rb" \
  "$FIXTURES/ruby/notifications.rb" "ruby/notifications.rb"
fetch "$GH/rails/rails/$RAILS_SHA/activerecord/lib/active_record/relation.rb" \
  "$FIXTURES/ruby/relation.rb" "ruby/relation.rb"
fetch "$GH/rails/rails/$RAILS_SHA/activerecord/lib/active_record/validations.rb" \
  "$FIXTURES/ruby/validations.rb" "ruby/validations.rb"

fetch "$GH/sinatra/sinatra/$SINATRA_SHA/lib/sinatra/base.rb" \
  "$FIXTURES/ruby/sinatra-base.rb" "ruby/sinatra-base.rb"
fetch "$GH/sinatra/sinatra/$SINATRA_SHA/lib/sinatra/main.rb" \
  "$FIXTURES/ruby/sinatra-main.rb" "ruby/sinatra-main.rb"

echo ""
echo "=== Rust (BurntSushi/ripgrep + tokio-rs/tokio) ==="
fetch "$GH/BurntSushi/ripgrep/$RIPGREP_SHA/crates/cli/src/lib.rs" \
  "$FIXTURES/rust/rg-cli-lib.rs" "rust/rg-cli-lib.rs"
fetch "$GH/BurntSushi/ripgrep/$RIPGREP_SHA/crates/globset/src/lib.rs" \
  "$FIXTURES/rust/rg-globset-lib.rs" "rust/rg-globset-lib.rs"
fetch "$GH/BurntSushi/ripgrep/$RIPGREP_SHA/crates/searcher/src/searcher/glue.rs" \
  "$FIXTURES/rust/rg-haystack.rs" "rust/rg-haystack.rs"
fetch "$GH/BurntSushi/ripgrep/$RIPGREP_SHA/crates/ignore/src/lib.rs" \
  "$FIXTURES/rust/rg-ignore-lib.rs" "rust/rg-ignore-lib.rs"
fetch "$GH/BurntSushi/ripgrep/$RIPGREP_SHA/crates/core/main.rs" \
  "$FIXTURES/rust/rg-main.rs" "rust/rg-main.rs"
fetch "$GH/BurntSushi/ripgrep/$RIPGREP_SHA/crates/matcher/src/lib.rs" \
  "$FIXTURES/rust/rg-matcher-lib.rs" "rust/rg-matcher-lib.rs"
fetch "$GH/BurntSushi/ripgrep/$RIPGREP_SHA/crates/core/messages.rs" \
  "$FIXTURES/rust/rg-messages.rs" "rust/rg-messages.rs"
fetch "$GH/BurntSushi/ripgrep/$RIPGREP_SHA/crates/printer/src/lib.rs" \
  "$FIXTURES/rust/rg-printer-lib.rs" "rust/rg-printer-lib.rs"
fetch "$GH/BurntSushi/ripgrep/$RIPGREP_SHA/crates/core/search.rs" \
  "$FIXTURES/rust/rg-search.rs" "rust/rg-search.rs"
fetch "$GH/BurntSushi/ripgrep/$RIPGREP_SHA/crates/searcher/src/lib.rs" \
  "$FIXTURES/rust/rg-searcher-lib.rs" "rust/rg-searcher-lib.rs"

fetch "$GH/tokio-rs/tokio/$TOKIO_SHA/tokio/src/runtime/builder.rs" \
  "$FIXTURES/rust/tokio-builder.rs" "rust/tokio-builder.rs"
fetch "$GH/tokio-rs/tokio/$TOKIO_SHA/tokio/src/net/tcp/listener.rs" \
  "$FIXTURES/rust/tokio-listener.rs" "rust/tokio-listener.rs"
fetch "$GH/tokio-rs/tokio/$TOKIO_SHA/tokio/src/process/mod.rs" \
  "$FIXTURES/rust/tokio-mod.rs" "rust/tokio-mod.rs"
fetch "$GH/tokio-rs/tokio/$TOKIO_SHA/tokio/src/sync/mutex.rs" \
  "$FIXTURES/rust/tokio-mutex.rs" "rust/tokio-mutex.rs"
fetch "$GH/tokio-rs/tokio/$TOKIO_SHA/tokio/src/sync/oneshot.rs" \
  "$FIXTURES/rust/tokio-oneshot.rs" "rust/tokio-oneshot.rs"
fetch "$GH/tokio-rs/tokio/$TOKIO_SHA/tokio/src/sync/rwlock.rs" \
  "$FIXTURES/rust/tokio-rwlock.rs" "rust/tokio-rwlock.rs"
fetch "$GH/tokio-rs/tokio/$TOKIO_SHA/tokio/src/net/tcp/stream.rs" \
  "$FIXTURES/rust/tokio-stream.rs" "rust/tokio-stream.rs"

echo ""
echo "========================================================"
echo "Total: $total files"
echo "  Unchanged: $unchanged"
echo "  Updated:   $updated"
echo "  Failed:    $failed"
echo "========================================================"

if [[ $failed -gt 0 ]]; then
  echo ""
  echo "WARNING: $failed files failed to fetch. Check URLs above."
  exit 1
fi
