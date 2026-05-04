# SWE-Bench Detail Report

Generated: 2026-05-04 20:03 UTC

---

## ruby-hard [ruby]

**Issue:** Regression in Grape 2.4.0: wrong content type returned when Accept header is a wildcard

> When upgrading from Grape 2.3.0 to 2.4.0, a regression was introduced in how the API resolves the response content type when the client sends a wildcard Accept header.
> 
> In version 2.3.0, when a Grape API was configured with multiple content types and a designated default format, a request with `Accept: */*` would correctly resolve to the configured default format. For example, given an API configured like this:
> 
> ```ruby
> class MultipleFormatAPI < Grape::API
>   content_type :xml, 'application/xml'
>   content_type :json, 'application/json'
> 
>   # json is the intended default
>   get :hello do
>     { hello: 'world' }
>   end
> end
> ```
> 
> A `GET /hello` request with `Accept: */*` should return a JSON response. Instead, since version 2.4.0, it returns XML (the first registered content type) rather than the configured default.
> 
> Reproduction steps:
> 
> 1. Configure a Grape API with multiple content types (e.g., XML and JSON) and set a specific default format.
> 2. Send a request with the header `Accept: */*`.
> 3. Observe the response Content-Type.
> 
> Expected: The response uses the configured default format (e.g., `application/json`).
> Actual: The response uses the first registered content type (e.g., `application/xml`), ignoring the configured default.
> 
> The wildcard Accept header (`*/*`) should be treated as "I accept anything, use your default" rather than being matched against the list of registered content types. Currently the wildcard is being matched as if it were a specific media type preference, which causes it to pick the first registered type instead of falling back to the default.
> 
> This is a breaking change from previous versions and contradicts the documented behavior in the README.

### Metrics

| Scenario | Duration | Input Tok | Cache Read | Output Tok | Cost |
|----------|----------|-----------|------------|------------|------|
| **baseline** | 248.3s | 833 | 289298 | 10074 | $0.5357 |
| **with-lumen** | 149.1s | 28 | 408611 | 8405 | $0.3705 |

### baseline

**Rating: Good**

```diff
diff --git a/lib/grape/middleware/formatter.rb b/lib/grape/middleware/formatter.rb
index f69ac35..50cac42 100644
--- a/lib/grape/middleware/formatter.rb
+++ b/lib/grape/middleware/formatter.rb
@@ -140,6 +140,9 @@ module Grape
         accept_header = env['HTTP_ACCEPT'].try(:scrub)
         return if accept_header.blank?
 
+        # A purely wildcard Accept header means "I accept anything" — fall back to default_format
+        return if accept_header.split(',').map { |part| part.split(';').first.strip }.all?('*/*')
+
         media_type = Rack::Utils.best_q_match(accept_header, mime_types.keys)
         mime_types[media_type] if media_type
       end
diff --git a/spec/grape/middleware/formatter_spec.rb b/spec/grape/middleware/formatter_spec.rb
index cb353aa..7b2de25 100644
--- a/spec/grape/middleware/formatter_spec.rb
+++ b/spec/grape/middleware/formatter_spec.rb
@@ -166,6 +166,30 @@ describe Grape::Middleware::Formatter do
       expect(subject.env[Grape::Env::API_FORMAT]).to eq(:xml)
     end
 
+    it 'uses the default format when Accept is a wildcard' do
+      subject.call(Rack::PATH_INFO => '/info', 'HTTP_ACCEPT' => '*/*')
+      expect(subject.env[Grape::Env::API_FORMAT]).to eq(:txt)
+    end
+
+    it 'uses the default format when Accept is a wildcard with quality' do
+      subject.call(Rack::PATH_INFO => '/info', 'HTTP_ACCEPT' => '*/*;q=1.0')
+      expect(subject.env[Grape::Env::API_FORMAT]).to eq(:txt)
+    end
+
+    context 'with multiple content types and a specific default format' do
+      subject { described_class.new(app, content_types: { xml: 'application/xml', json: 'application/json' }, default_format: :json) }
+
+      it 'uses the configured default format when Accept is a wildcard' do
+        subject.call(Rack::PATH_INFO => '/info', 'HTTP_ACCEPT' => '*/*')
+        expect(subject.env[Grape::Env::API_FORMAT]).to eq(:json)
+      end
+
+      it 'still negotiates when Accept has a specific type alongside a wildcard' do
+        subject.call(Rack::PATH_INFO => '/info', 'HTTP_ACCEPT' => 'application/xml,*/*;q=0.5')
+        expect(subject.env[Grape::Env::API_FORMAT]).to eq(:xml)
+      end
+    end
+
     context 'with custom vendored content types' do
       context 'when registered' do
         subject { described_class.new(app, content_types: { custom: 'application/vnd.test+json' }) }
```

### with-lumen

**Rating: Good**

```diff
diff --git a/lib/grape/middleware/formatter.rb b/lib/grape/middleware/formatter.rb
index f69ac35..5cf9dc0 100644
--- a/lib/grape/middleware/formatter.rb
+++ b/lib/grape/middleware/formatter.rb
@@ -140,6 +140,9 @@ module Grape
         accept_header = env['HTTP_ACCEPT'].try(:scrub)
         return if accept_header.blank?
 
+        # A pure wildcard means "accept anything" — fall through to default_format
+        return if Rack::Utils.q_values(accept_header).all? { |mime_type, _| mime_type == '*/*' }
+
         media_type = Rack::Utils.best_q_match(accept_header, mime_types.keys)
         mime_types[media_type] if media_type
       end
diff --git a/spec/grape/middleware/formatter_spec.rb b/spec/grape/middleware/formatter_spec.rb
index cb353aa..bf278ff 100644
--- a/spec/grape/middleware/formatter_spec.rb
+++ b/spec/grape/middleware/formatter_spec.rb
@@ -188,6 +188,20 @@ describe Grape::Middleware::Formatter do
       subject.call(Rack::PATH_INFO => '/info', 'HTTP_ACCEPT' => 'application/xml', system_time: '091293')
       expect(subject.env[:system_time]).to eq('091293')
     end
+
+    context 'with a wildcard Accept header' do
+      subject { described_class.new(app, content_types: { xml: 'application/xml', json: 'application/json' }, default_format: :json) }
+
+      it 'falls back to the default format when Accept is */*' do
+        subject.call(Rack::PATH_INFO => '/info', 'HTTP_ACCEPT' => '*/*')
+        expect(subject.env[Grape::Env::API_FORMAT]).to eq(:json)
+      end
+
+      it 'uses specific type preference when */* accompanies explicit types' do
+        subject.call(Rack::PATH_INFO => '/info', 'HTTP_ACCEPT' => 'application/xml, */*;q=0.5')
+        expect(subject.env[Grape::Env::API_FORMAT]).to eq(:xml)
+      end
+    end
   end
 
   context 'content-type' do
```


