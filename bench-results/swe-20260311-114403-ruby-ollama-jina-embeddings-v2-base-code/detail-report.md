# SWE-Bench Detail Report

Generated: 2026-03-11 10:51 UTC

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
| **baseline** | 185.5s | 23 | 516559 | 6143 | $0.5389 |
| **with-lumen** | 165.2s | 19 | 294588 | 5581 | $0.4114 |

### baseline

**Rating: Good**

```diff
diff --git a/lib/grape/middleware/formatter.rb b/lib/grape/middleware/formatter.rb
index f69ac35..444f88e 100644
--- a/lib/grape/middleware/formatter.rb
+++ b/lib/grape/middleware/formatter.rb
@@ -139,6 +139,7 @@ module Grape
       def format_from_header
         accept_header = env['HTTP_ACCEPT'].try(:scrub)
         return if accept_header.blank?
+        return if accept_header.strip == '*/*'
 
         media_type = Rack::Utils.best_q_match(accept_header, mime_types.keys)
         mime_types[media_type] if media_type
diff --git a/spec/grape/middleware/formatter_spec.rb b/spec/grape/middleware/formatter_spec.rb
index cb353aa..b997346 100644
--- a/spec/grape/middleware/formatter_spec.rb
+++ b/spec/grape/middleware/formatter_spec.rb
@@ -188,6 +188,14 @@ describe Grape::Middleware::Formatter do
       subject.call(Rack::PATH_INFO => '/info', 'HTTP_ACCEPT' => 'application/xml', system_time: '091293')
       expect(subject.env[:system_time]).to eq('091293')
     end
+
+    context 'when Accept header is a wildcard' do
+      it 'falls back to the configured default format' do
+        s = described_class.new(app, content_types: { xml: 'application/xml', json: 'application/json' }, default_format: :json)
+        _, headers, = s.call(Rack::PATH_INFO => '/info', 'HTTP_ACCEPT' => '*/*')
+        expect(headers[Rack::CONTENT_TYPE]).to eq('application/json')
+      end
+    end
   end
 
   context 'content-type' do
```

### with-lumen

**Rating: Good**

```diff
diff --git a/lib/grape/middleware/formatter.rb b/lib/grape/middleware/formatter.rb
index f69ac35..264968a 100644
--- a/lib/grape/middleware/formatter.rb
+++ b/lib/grape/middleware/formatter.rb
@@ -139,6 +139,7 @@ module Grape
       def format_from_header
         accept_header = env['HTTP_ACCEPT'].try(:scrub)
         return if accept_header.blank?
+        return if Rack::Utils.q_values(accept_header).all? { |media_type, _| media_type == '*/*' }
 
         media_type = Rack::Utils.best_q_match(accept_header, mime_types.keys)
         mime_types[media_type] if media_type
```


