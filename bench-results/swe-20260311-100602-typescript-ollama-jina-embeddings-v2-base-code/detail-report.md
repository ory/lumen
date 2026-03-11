# SWE-Bench Detail Report

Generated: 2026-03-11 09:09 UTC

---

## typescript-hard [typescript]

**Issue:** Negative flag (--no-) negation does not propagate to aliases or main option

> When using the `--no-` prefix to negate a boolean flag that has aliases, the negation is not correctly propagated between the alias and the main option name.
> 
> For example, given a boolean argument `verbose` with default `true` and alias `v`:
> 
> - Running with `--no-v` sets `v=false` but leaves `verbose=true` (the default), because the propagation logic only copies values when the target is `undefined`. Since `verbose` has a default of `true`, it is not `undefined`, so the negated value from the alias never overwrites it.
> 
> - Similarly, running with `--no-verbose` sets `verbose=false` but does not propagate the negation to the alias `v`.
> 
> Reproduction:
> 
> ```ts
> import { parseRawArgs } from 'citty';
> 
> // Case 1: --no-alias should propagate to main option
> const result1 = parseRawArgs(['--no-v'], {
>   boolean: ['verbose'],
>   alias: { v: ['verbose'] },
>   default: { verbose: true },
> });
> console.log(result1);
> // Actual:   { _: [], v: false, verbose: true }
> // Expected: { _: [], v: false, verbose: false }
> 
> // Case 2: --no-main should propagate to aliases
> const result2 = parseRawArgs(['--no-verbose'], {
>   boolean: ['verbose'],
>   alias: { v: ['verbose'] },
>   default: { verbose: true },
> });
> console.log(result2);
> // Actual:   { _: [], verbose: false }  (v is missing/undefined)
> // Expected: { _: [], v: false, verbose: false }
> ```
> 
> The root cause is in the argument parser's handling of negated flags. When a `--no-` prefixed flag is encountered, the parser sets only the literal flag name to `false` without resolving it through the alias mapping. The negation needs to be resolved bidirectionally: if the negated name is an alias, the main option should also be set to `false`; if the negated name is the main option, all its aliases should also be set to `false`. This must happen before the general alias propagation step, which skips values that are already defined (non-undefined), so default values block the propagation of negated flags.

### Metrics

| Scenario | Duration | Input Tok | Cache Read | Output Tok | Cost |
|----------|----------|-----------|------------|------------|------|
| **baseline** | 84.4s | 8 | 119935 | 4994 | $0.1859 |
| **with-lumen** | 56.3s | 14 | 182721 | 1813 | $0.1356 |

### baseline

**Rating: Good**

```diff
diff --git a/package-lock.json b/package-lock.json
new file mode 100644
index 0000000..2c16875
--- /dev/null
+++ b/package-lock.json
@@ -0,0 +1,6406 @@
+{
+  "name": "citty",
+  "version": "0.2.0",
+  "lockfileVersion": 3,
+  "requires": true,
+  "packages": {
+    "": {
+      "name": "citty",
+      "version": "0.2.0",
+      "license": "MIT",
+      "devDependencies": {
+        "@types/node": "^25.0.9",
+        "@vitest/coverage-v8": "^4.0.17",
+        "automd": "^0.4.2",
+        "changelogen": "^0.6.2",
+        "eslint": "^9.39.2",
+        "eslint-config-unjs": "^0.6.2",
+        "obuild": "^0.4.18",
+        "prettier": "^3.8.0",
+        "scule": "^1.3.0",
+        "typescript": "^5.9.3",
+        "vitest": "^4.0.17"
+      }
+    },
+    "node_modules/@babel/generator": {
+      "version": "8.0.0-rc.2",
+      "resolved": "https://registry.npmjs.org/@babel/generator/-/generator-8.0.0-rc.2.tgz",
+      "integrity": "sha512-oCQ1IKPwkzCeJzAPb7Fv8rQ9k5+1sG8mf2uoHiMInPYvkRfrDJxbTIbH51U+jstlkghus0vAi3EBvkfvEsYNLQ==",
+      "dev": true,
+      "license": "MIT",
+      "dependencies": {
+        "@babel/parser": "^8.0.0-rc.2",
+        "@babel/types": "^8.0.0-rc.2",
+        "@jridgewell/gen-mapping": "^0.3.12",
+        "@jridgewell/trace-mapping": "^0.3.28",
+        "@types/jsesc": "^2.5.0",
+        "jsesc": "^3.0.2"
+      },
+      "engines": {
+        "node": "^20.19.0 || >=22.12.0"
+      }
+    },
+    "node_modules/@babel/generator/node_modules/@babel/helper-string-parser": {
+      "version": "8.0.0-rc.2",
+      "resolved": "https://registry.npmjs.org/@babel/helper-string-parser/-/helper-string-parser-8.0.0-rc.2.tgz",
+      "integrity": "sha512-noLx87RwlBEMrTzncWd/FvTxoJ9+ycHNg0n8yyYydIoDsLZuxknKgWRJUqcrVkNrJ74uGyhWQzQaS3q8xfGAhQ==",
+      "dev": true,
+      "license": "MIT",
+      "engines": {
+        "node": "^20.19.0 || >=22.12.0"
+      }
+    },
+    "node_modules/@babel/generator/node_modules/@babel/helper-validator-identifier": {
+      "version": "8.0.0-rc.2",
+      "resolved": "https://registry.npmjs.org/@babel/helper-validator-identifier/-/helper-validator-identifier-8.0.0-rc.2.tgz",
+      "integrity": "sha512-xExUBkuXWJjVuIbO7z6q7/BA9bgfJDEhVL0ggrggLMbg0IzCUWGT1hZGE8qUH7Il7/RD/a6cZ3AAFrrlp1LF/A==",
+      "dev": true,
+      "license": "MIT",
+      "engines": {
+        "node": "^20.19.0 || >=22.12.0"
+      }
+    },
+    "node_modules/@babel/generator/node_modules/@babel/parser": {
+      "version": "8.0.0-rc.2",
+      "resolved": "https://registry.npmjs.org/@babel/parser/-/parser-8.0.0-rc.2.tgz",
+      "integrity": "sha512-29AhEtcq4x8Dp3T72qvUMZHx0OMXCj4Jy/TEReQa+KWLln524Cj1fWb3QFi0l/xSpptQBR6y9RNEXuxpFvwiUQ==",
+      "dev": true,
+      "license": "MIT",
+      "dependencies": {
+        "@babel/types": "^8.0.0-rc.2"
+      },
+      "bin": {
+        "parser": "bin/babel-parser.js"
+      },
+      "engines": {
+        "node": "^20.19.0 || >=22.12.0"
+      }
+    },
+    "node_modules/@babel/generator/node_modules/@babel/types": {
+      "version": "8.0.0-rc.2",
+      "resolved": "https://registry.npmjs.org/@babel/types/-/types-8.0.0-rc.2.tgz",
+      "integrity": "sha512-91gAaWRznDwSX4E2tZ1YjBuIfnQVOFDCQ2r0Toby0gu4XEbyF623kXLMA8d4ZbCu+fINcrudkmEcwSUHgDDkNw==",
+      "dev": true,
+      "license": "MIT",
+      "dependencies": {
+        "@babel/helper-string-parser": "^8.0.0-rc.2",
+        "@babel/helper-validator-identifier": "^8.0.0-rc.2"
+      },
+      "engines": {
+        "node": "^20.19.0 || >=22.12.0"
+      }
+    },
+    "node_modules/@babel/helper-string-parser": {
+      "version": "7.27.1",
+      "resolved": "https://registry.npmjs.org/@babel/helper-string-parser/-/helper-string-parser-7.27.1.tgz",
+      "integrity": "sha512-qMlSxKbpRlAridDExk92nSobyDdpPijUq2DW6oDnUqd0iOGxmQjyqhMIihI9+zv4LPyZdRje2cavWPbCbWm3eA==",
+      "dev": true,
+      "license": "MIT",
+      "engines": {
+        "node": ">=6.9.0"
+      }
+    },
+    "node_modules/@babel/helper-validator-identifier": {
+      "version": "7.28.5",
+      "resolved": "https://registry.npmjs.org/@babel/helper-validator-identifier/-/helper-validator-identifier-7.28.5.tgz",
+      "integrity": "sha512-qSs4ifwzKJSV39ucNjsvc6WVHs6b7S03sOh2OcHF9UHfVPqWWALUsNUVzhSBiItjRZoLHx7nIarVjqKVusUZ1Q==",
+      "dev": true,
+      "license": "MIT",
+      "engines": {
+        "node": ">=6.9.0"
+      }
+    },
+    "node_modules/@babel/parser": {
+      "version": "7.29.0",
+      "resolved": "https://registry.npmjs.org/@babel/parser/-/parser-7.29.0.tgz",
+      "integrity": "sha512-IyDgFV5GeDUVX4YdF/3CPULtVGSXXMLh1xVIgdCgxApktqnQV0r7/8Nqthg+8YLGaAtdyIlo2qIdZrbCv4+7ww==",
+      "dev": true,
+      "license": "MIT",
+      "dependencies": {
+        "@babel/types": "^7.29.0"
+      },
+      "bin": {
+        "parser": "bin/babel-parser.js"
+      },
+      "engines": {
+        "node": ">=6.0.0"
+      }
+    },
+    "node_modules/@babel/runtime": {
+      "version": "7.28.6",
+      "resolved": "https://registry.npmjs.org/@babel/runtime/-/runtime-7.28.6.tgz",
+      "integrity": "sha512-05WQkdpL9COIMz4LjTxGpPNCdlpyimKppYNoJ5Di5EUObifl8t4tuLuUBBZEpoLYOmfvIWrsp9fCl0HoPRVTdA==",
+      "dev": true,
+      "license": "MIT",
+      "engines": {
+        "node": ">=6.9.0"
+      }
+    },
+    "node_modules/@babel/types": {
+      "version": "7.29.0",
+      "resolved": "https://registry.npmjs.org/@babel/types/-/types-7.29.0.tgz",
+      "integrity": "sha512-LwdZHpScM4Qz8Xw2iKSzS+cfglZzJGvofQICy7W7v4caru4EaAmyUuO6BGrbyQ2mYV11W0U8j5mBhd14dd3B0A==",
+      "dev": true,
+      "license": "MIT",
+      "dependencies": {
+        "@babel/helper-string-parser": "^7.27.1",
+        "@babel/helper-validator-identifier": "^7.28.5"
+      },
+      "engines": {
+        "node": ">=6.9.0"
+      }
+    },
+    "node_modules/@bcoe/v8-coverage": {
+      "version": "1.0.2",
+      "resolved": "https://registry.npmjs.org/@bcoe/v8-coverage/-/v8-coverage-1.0.2.tgz",
+      "integrity": "sha512-6zABk/ECA/QYSCQ1NGiVwwbQerUCZ+TQbp64Q3AgmfNvurHH0j8TtXa1qbShXA6qqkpAj4V5W8pP6mLe1mcMqA==",
+      "dev": true,
+      "license": "MIT",
+      "engines": {
+        "node": ">=18"
+      }
+    },
+    "node_modules/@emnapi/core": {
+      "version": "1.8.1",
+      "resolved": "https://registry.npmjs.org/@emnapi/core/-/core-1.8.1.tgz",
+      "integrity": "sha512-AvT9QFpxK0Zd8J0jopedNm+w/2fIzvtPKPjqyw9jwvBaReTTqPBk9Hixaz7KbjimP+QNz605/XnjFcDAL2pqBg==",
+      "dev": true,
+      "license": "MIT",
+      "optional": true,
+      "dependencies": {
+        "@emnapi/wasi-threads": "1.1.0",
+        "tslib": "^2.4.0"
+      }
+    },
+    "node_modules/@emnapi/runtime": {
+      "version": "1.8.1",
+      "resolved": "https://registry.npmjs.org/@emnapi/runtime/-/runtime-1.8.1.tgz",
+      "integrity": "sha512-mehfKSMWjjNol8659Z8KxEMrdSJDDot5SXMq00dM8BN4o+CLNXQ0xH2V7EchNHV4RmbZLmmPdEaXZc5H2FXmDg==",
+      "dev": true,
+      "license": "MIT",
+      "optional": true,
+      "dependencies": {
+        "tslib": "^2.4.0"
+      }
+    },
+    "node_modules/@emnapi/wasi-threads": {
+      "version": "1.1.0",
+      "resolved": "https://registry.npmjs.org/@emnapi/wasi-threads/-/wasi-threads-1.1.0.tgz",
+      "integrity": "sha512-WI0DdZ8xFSbgMjR1sFsKABJ/C5OnRrjT06JXbZKexJGrDuPTzZdDYfFlsgcCXCyf+suG5QU2e/y1Wo2V/OapLQ==",
+      "dev": true,
+      "license": "MIT",
+      "optional": true,
+      "dependencies": {
+        "tslib": "^2.4.0"
+      }
+    },
+    "node_modules/@esbuild/aix-ppc64": {
+      "version": "0.27.3",
+      "resolved": "https://registry.npmjs.org/@esbuild/aix-ppc64/-/aix-ppc64-0.27.3.tgz",
+      "integrity": "sha512-9fJMTNFTWZMh5qwrBItuziu834eOCUcEqymSH7pY+zoMVEZg3gcPuBNxH1EvfVYe9h0x/Ptw8KBzv7qxb7l8dg==",
+      "cpu": [
+        "ppc64"
+      ],
+      "dev": true,
+      "license": "MIT",
+      "optional": true,
+      "os": [
+        "aix"
+      ],
+      "engines": {
+        "node": ">=18"
+      }
+    },
+    "node_modules/@esbuild/android-arm": {
+      "version": "0.27.3",
+      "resolved": "https://registry.npmjs.org/@esbuild/android-arm/-/android-arm-0.27.3.tgz",
+      "integrity": "sha512-i5D1hPY7GIQmXlXhs2w8AWHhenb00+GxjxRncS2ZM7YNVGNfaMxgzSGuO8o8SJzRc/oZwU2bcScvVERk03QhzA==",
+      "cpu": [
+        "arm"
+      ],
+      "dev": true,
+      "license": "MIT",
+      "optional": true,
+      "os": [
+        "android"
+      ],
+      "engines": {
+        "node": ">=18"
+      }
+    },
+    "node_modules/@esbuild/android-arm64": {
+      "version": "0.27.3",
+      "resolved": "https://registry.npmjs.org/@esbuild/android-arm64/-/android-arm64-0.27.3.tgz",
+      "integrity": "sha512-YdghPYUmj/FX2SYKJ0OZxf+iaKgMsKHVPF1MAq/P8WirnSpCStzKJFjOjzsW0QQ7oIAiccHdcqjbHmJxRb/dmg==",
+      "cpu": [
+        "arm64"
+      ],
+      "dev": true,
+      "license": "MIT",
+      "optional": true,
+      "os": [
+        "android"
+      ],
+      "engines": {
+        "node": ">=18"
+      }
+    },
+    "node_modules/@esbuild/android-x64": {
+      "version": "0.27.3",
+      "resolved": "https://registry.npmjs.org/@esbuild/android-x64/-/android-x64-0.27.3.tgz",
+      "integrity": "sha512-IN/0BNTkHtk8lkOM8JWAYFg4ORxBkZQf9zXiEOfERX/CzxW3Vg1ewAhU7QSWQpVIzTW+b8Xy+lGzdYXV6UZObQ==",
+      "cpu": [
+        "x64"
+      ],
+      "dev": true,
+      "license": "MIT",
+      "optional": true,
+      "os": [
+        "android"
+      ],
+      "engines": {
+        "node": ">=18"
+      }
+    },
+    "node_modules/@esbuild/darwin-arm64": {
+      "version": "0.27.3",
+      "resolved": "https://registry.npmjs.org/@esbuild/darwin-arm64/-/darwin-arm64-0.27.3.tgz",
+      "integrity": "sha512-Re491k7ByTVRy0t3EKWajdLIr0gz2kKKfzafkth4Q8A5n1xTHrkqZgLLjFEHVD+AXdUGgQMq+Godfq45mGpCKg==",
+      "cpu": [
+        "arm64"
+      ],
+      "dev": true,
+      "license": "MIT",
+      "optional": true,
+      "os": [
+        "darwin"
+      ],
+      "engines": {
+        "node": ">=18"
+      }
+    },
+    "node_modules/@esbuild/darwin-x64": {
+      "version": "0.27.3",
+      "resolved": "https://registry.npmjs.org/@esbuild/darwin-x64/-/darwin-x64-0.27.3.tgz",
+      "integrity": "sha512-vHk/hA7/1AckjGzRqi6wbo+jaShzRowYip6rt6q7VYEDX4LEy1pZfDpdxCBnGtl+A5zq8iXDcyuxwtv3hNtHFg==",
+      "cpu": [
+        "x64"
+      ],
+      "dev": true,
+      "license": "MIT",
+      "optional": true,
+      "os": [
+        "darwin"
+      ],
+      "engines": {
+        "node": ">=18"
+      }
+    },
+    "node_modules/@esbuild/freebsd-arm64": {
+      "version": "0.27.3",
+      "resolved": "https://registry.npmjs.org/@esbuild/freebsd-arm64/-/freebsd-arm64-0.27.3.tgz",
+      "integrity": "sha512-ipTYM2fjt3kQAYOvo6vcxJx3nBYAzPjgTCk7QEgZG8AUO3ydUhvelmhrbOheMnGOlaSFUoHXB6un+A7q4ygY9w==",
+      "cpu": [
+        "arm64"
+      ],
+      "dev": true,
+      "license": "MIT",
+      "optional": true,
+      "os": [
+        "freebsd"
+      ],
+      "engines": {
+        "node": ">=18"
+      }
+    },
+    "node_modules/@esbuild/freebsd-x64": {
+      "version": "0.27.3",
+      "resolved": "https://registry.npmjs.org/@esbuild/freebsd-x64/-/freebsd-x64-0.27.3.tgz",
+      "integrity": "sha512-dDk0X87T7mI6U3K9VjWtHOXqwAMJBNN2r7bejDsc+j03SEjtD9HrOl8gVFByeM0aJksoUuUVU9TBaZa2rgj0oA==",
+      "cpu": [
+        "x64"
+      ],
+      "dev": true,
+      "license": "MIT",
+      "optional": true,
+      "os": [
+        "freebsd"
+      ],
+      "engines": {
+        "node": ">=18"
+      }
+    },
+    "node_modules/@esbuild/linux-arm": {
+      "version": "0.27.3",
+      "resolved": "https://registry.npmjs.org/@esbuild/linux-arm/-/linux-arm-0.27.3.tgz",
+      "integrity": "sha512-s6nPv2QkSupJwLYyfS+gwdirm0ukyTFNl3KTgZEAiJDd+iHZcbTPPcWCcRYH+WlNbwChgH2QkE9NSlNrMT8Gfw==",
+      "cpu": [
+        "arm"
+      ],
+      "dev": true,
+      "license": "MIT",
+      "optional": true,
+      "os": [
+        "linux"
+      ],
+      "engines": {
+        "node": ">=18"
+      }
+    },
+    "node_modules/@esbuild/linux-arm64": {
+      "version": "0.27.3",
+      "resolved": "https://registry.npmjs.org/@esbuild/linux-arm64/-/linux-arm64-0.27.3.tgz",
+      "integrity": "sha512-sZOuFz/xWnZ4KH3YfFrKCf1WyPZHakVzTiqji3WDc0BCl2kBwiJLCXpzLzUBLgmp4veFZdvN5ChW4Eq/8Fc2Fg==",
+      "cpu": [
+        "arm64"
+      ],
+      "dev": true,
+      "license": "MIT",
+      "optional": true,
+      "os": [
+        "linux"
+      ],
+      "engines": {
+        "node": ">=18"
+      }
+    },
+    "node_modules/@esbuild/linux-ia32": {
+      "version": "0.27.3",
+      "resolved": "https://registry.npmjs.org/@esbuild/linux-ia32/-/linux-ia32-0.27.3.tgz",
+      "integrity": "sha512-yGlQYjdxtLdh0a3jHjuwOrxQjOZYD/C9PfdbgJJF3TIZWnm/tMd/RcNiLngiu4iwcBAOezdnSLAwQDPqTmtTYg==",
+      "cpu": [
+        "ia32"
+      ],
+      "dev": true,
+      "license": "MIT",
+      "optional": true,
+      "os": [
+        "linux"
+      ],
+      "engines": {
+        "node": ">=18"
+      }
+    },
+    "node_modules/@esbuild/linux-loong64": {
+      "version": "0.27.3",
+      "resolved": "https://registry.npmjs.org/@esbuild/linux-loong64/-/linux-loong64-0.27.3.tgz",
+      "integrity": "sha512-WO60Sn8ly3gtzhyjATDgieJNet/KqsDlX5nRC5Y3oTFcS1l0KWba+SEa9Ja1GfDqSF1z6hif/SkpQJbL63cgOA==",
+      "cpu": [
+        "loong64"
+      ],
+      "dev": true,
+      "license": "MIT",
+      "optional": true,
+      "os": [
+        "linux"
+      ],
+      "engines": {
+        "node": ">=18"
+      }
+    },
+    "node_modules/@esbuild/linux-mips64el": {
+      "version": "0.27.3",
+      "resolved": "https://registry.npmjs.org/@esbuild/linux-mips64el/-/linux-mips64el-0.27.3.tgz",
+      "integrity": "sha512-APsymYA6sGcZ4pD6k+UxbDjOFSvPWyZhjaiPyl/f79xKxwTnrn5QUnXR5prvetuaSMsb4jgeHewIDCIWljrSxw==",
+      "cpu": [
+        "mips64el"
+      ],
+      "dev": true,
+      "license": "MIT",
+      "optional": true,
+      "os": [
+        "linux"
+      ],
+      "engines": {
+        "node": ">=18"
+      }
+    },
+    "node_modules/@esbuild/linux-ppc64": {
+      "version": "0.27.3",
+      "resolved": "https://registry.npmjs.org/@esbuild/linux-ppc64/-/linux-ppc64-0.27.3.tgz",
+      "integrity": "sha512-eizBnTeBefojtDb9nSh4vvVQ3V9Qf9Df01PfawPcRzJH4gFSgrObw+LveUyDoKU3kxi5+9RJTCWlj4FjYXVPEA==",
+      "cpu": [
+        "ppc64"
+      ],
+      "dev": true,
+      "license": "MIT",
+      "optional": true,
+      "os": [
+        "linux"
+      ],
+      "engines": {
+        "node": ">=18"
+      }
+    },
+    "node_modules/@esbuild/linux-riscv64": {
+      "version": "0.27.3",
+      "resolved": "https://registry.npmjs.org/@esbuild/linux-riscv64/-/linux-riscv64-0.27.3.tgz",
+      "integrity": "sha512-3Emwh0r5wmfm3ssTWRQSyVhbOHvqegUDRd0WhmXKX2mkHJe1SFCMJhagUleMq+Uci34wLSipf8Lagt4LlpRFWQ==",
+      "cpu": [
+        "riscv64"
+      ],
+      "dev": true,
+      "license": "MIT",
+      "optional": true,
+      "os": [
+        "linux"
+      ],
+      "engines": {
+        "node": ">=18"
+      }
+    },
+    "node_modules/@esbuild/linux-s390x": {
+      "version": "0.27.3",
+      "resolved": "https://registry.npmjs.org/@esbuild/linux-s390x/-/linux-s390x-0.27.3.tgz",
+      "integrity": "sha512-pBHUx9LzXWBc7MFIEEL0yD/ZVtNgLytvx60gES28GcWMqil8ElCYR4kvbV2BDqsHOvVDRrOxGySBM9Fcv744hw==",
+      "cpu": [
+        "s390x"
+      ],
+      "dev": true,
+      "license": "MIT",
+      "optional": true,
+      "os": [
+        "linux"
+      ],
+      "engines": {
+        "node": ">=18"
+      }
+    },
+    "node_modules/@esbuild/linux-x64": {
+      "version": "0.27.3",
+      "resolved": "https://registry.npmjs.org/@esbuild/linux-x64/-/linux-x64-0.27.3.tgz",
+      "integrity": "sha512-Czi8yzXUWIQYAtL/2y6vogER8pvcsOsk5cpwL4Gk5nJqH5UZiVByIY8Eorm5R13gq+DQKYg0+JyQoytLQas4dA==",
+      "cpu": [
+        "x64"
+      ],
+      "dev": true,
+      "license": "MIT",
+      "optional": true,
+      "os": [
+        "linux"
+      ],
+      "engines": {
+        "node": ">=18"
+      }
+    },
+    "node_modules/@esbuild/netbsd-arm64": {
+      "version": "0.27.3",
+      "resolved": "https://registry.npmjs.org/@esbuild/netbsd-arm64/-/netbsd-arm64-0.27.3.tgz",
+      "integrity": "sha512-sDpk0RgmTCR/5HguIZa9n9u+HVKf40fbEUt+iTzSnCaGvY9kFP0YKBWZtJaraonFnqef5SlJ8/TiPAxzyS+UoA==",
+      "cpu": [
+        "arm64"
+      ],
+      "dev": true,
+      "license": "MIT",
+      "optional": true,
+      "os": [
+        "netbsd"
+      ],
+      "engines": {
+        "node": ">=18"
+      }
+    },
+    "node_modules/@esbuild/netbsd-x64": {
+      "version": "0.27.3",
+      "resolved": "https://registry.npmjs.org/@esbuild/netbsd-x64/-/netbsd-x64-0.27.3.tgz",
+      "integrity": "sha512-P14lFKJl/DdaE00LItAukUdZO5iqNH7+PjoBm+fLQjtxfcfFE20Xf5CrLsmZdq5LFFZzb5JMZ9grUwvtVYzjiA==",
+      "cpu": [
+        "x64"
+      ],
+      "dev": true,
+      "license": "MIT",
+      "optional": true,
+      "os": [
+        "netbsd"
+      ],
+      "engines": {
+        "node": ">=18"
+      }
+    },
+    "node_modules/@esbuild/openbsd-arm64": {
+      "version": "0.27.3",
+      "resolved": "https://registry.npmjs.org/@esbuild/openbsd-arm64/-/openbsd-arm64-0.27.3.tgz",
+      "integrity": "sha512-AIcMP77AvirGbRl/UZFTq5hjXK+2wC7qFRGoHSDrZ5v5b8DK/GYpXW3CPRL53NkvDqb9D+alBiC/dV0Fb7eJcw==",
+      "cpu": [
+        "arm64"
+      ],
+      "dev": true,
+      "license": "MIT",
+      "optional": true,
+      "os": [
+        "openbsd"
+      ],
+      "engines": {
+        "node": ">=18"
+      }
+    },
+    "node_modules/@esbuild/openbsd-x64": {
+      "version": "0.27.3",
+      "resolved": "https://registry.npmjs.org/@esbuild/openbsd-x64/-/openbsd-x64-0.27.3.tgz",
+      "integrity": "sha512-DnW2sRrBzA+YnE70LKqnM3P+z8vehfJWHXECbwBmH/CU51z6FiqTQTHFenPlHmo3a8UgpLyH3PT+87OViOh1AQ==",
+      "cpu": [
+        "x64"
+      ],
+      "dev": true,
+      "license": "MIT",
+      "optional": true,
+      "os": [
+        "openbsd"
+      ],
+      "engines": {
+        "node": ">=18"
+      }
+    },
+    "node_modules/@esbuild/openharmony-arm64": {
+      "version": "0.27.3",
+      "resolved": "https://registry.npmjs.org/@esbuild/openharmony-arm64/-/openharmony-arm64-0.27.3.tgz",
+      "integrity": "sha512-NinAEgr/etERPTsZJ7aEZQvvg/A6IsZG/LgZy+81wON2huV7SrK3e63dU0XhyZP4RKGyTm7aOgmQk0bGp0fy2g==",
+      "cpu": [
+        "arm64"
+      ],
+      "dev": true,
+      "license": "MIT",
+      "optional": true,
+      "os": [
+        "openharmony"
+      ],
+      "engines": {
+        "node": ">=18"
+      }
+    },
+    "node_modules/@esbuild/sunos-x64": {
+      "version": "0.27.3",
+      "resolved": "https://registry.npmjs.org/@esbuild/sunos-x64/-/sunos-x64-0.27.3.tgz",
+      "integrity": "sha512-PanZ+nEz+eWoBJ8/f8HKxTTD172SKwdXebZ0ndd953gt1HRBbhMsaNqjTyYLGLPdoWHy4zLU7bDVJztF5f3BHA==",
+      "cpu": [
+        "x64"
+      ],
+      "dev": true,
+      "license": "MIT",
+      "optional": true,
+      "os": [
+        "sunos"
+      ],
+      "engines": {
+        "node": ">=18"
+      }
+    },
+    "node_modules/@esbuild/win32-arm64": {
+      "version": "0.27.3",
+      "resolved": "https://registry.npmjs.org/@esbuild/win32-arm64/-/win32-arm64-0.27.3.tgz",
+      "integrity": "sha512-B2t59lWWYrbRDw/tjiWOuzSsFh1Y/E95ofKz7rIVYSQkUYBjfSgf6oeYPNWHToFRr2zx52JKApIcAS/D5TUBnA==",
+      "cpu": [
+        "arm64"
+      ],
+      "dev": true,
+      "license": "MIT",
+      "optional": true,
+      "os": [
+        "win32"
+      ],
+      "engines": {
+        "node": ">=18"
+      }
+    },
+    "node_modules/@esbuild/win32-ia32": {
+      "version": "0.27.3",
+      "resolved": "https://registry.npmjs.org/@esbuild/win32-ia32/-/win32-ia32-0.27.3.tgz",
+      "integrity": "sha512-QLKSFeXNS8+tHW7tZpMtjlNb7HKau0QDpwm49u0vUp9y1WOF+PEzkU84y9GqYaAVW8aH8f3GcBck26jh54cX4Q==",
+      "cpu": [
+        "ia32"
+      ],
+      "dev": true,
+      "license": "MIT",
+      "optional": true,
+      "os": [
+        "win32"
+      ],
+      "engines": {
+        "node": ">=18"
+      }
+    },
+    "node_modules/@esbuild/win32-x64": {
+      "version": "0.27.3",
+      "resolved": "https://registry.npmjs.org/@esbuild/win32-x64/-/win32-x64-0.27.3.tgz",
+      "integrity": "sha512-4uJGhsxuptu3OcpVAzli+/gWusVGwZZHTlS63hh++ehExkVT8SgiEf7/uC/PclrPPkLhZqGgCTjd0VWLo6xMqA==",
+      "cpu": [
+        "x64"
+      ],
+      "dev": true,
+      "license": "MIT",
+      "optional": true,
+      "os": [
+        "win32"
+      ],
+      "engines": {
+        "node": ">=18"
+      }
+    },
+    "node_modules/@eslint-community/eslint-utils": {
+      "version": "4.9.1",
+      "resolved": "https://registry.npmjs.org/@eslint-community/eslint-utils/-/eslint-utils-4.9.1.tgz",
+      "integrity": "sha512-phrYmNiYppR7znFEdqgfWHXR6NCkZEK7hwWDHZUjit/2/U0r6XvkDl0SYnoM51Hq7FhCGdLDT6zxCCOY1hexsQ==",
+      "dev": true,
+      "license": "MIT",
+      "dependencies": {
+        "eslint-visitor-keys": "^3.4.3"
+      },
+      "engines": {
+        "node": "^12.22.0 || ^14.17.0 || >=16.0.0"
+      },
+      "funding": {
+        "url": "https://opencollective.com/eslint"
+      },
+      "peerDependencies": {
+        "eslint": "^6.0.0 || ^7.0.0 || >=8.0.0"
+      }
+    },
+    "node_modules/@eslint-community/eslint-utils/node_modules/eslint-visitor-keys": {
+      "version": "3.4.3",
+      "resolved": "https://registry.npmjs.org/eslint-visitor-keys/-/eslint-visitor-keys-3.4.3.tgz",
+      "integrity": "sha512-wpc+LXeiyiisxPlEkUzU6svyS1frIO3Mgxj1fdy7Pm8Ygzguax2N3Fa/D/ag1WqbOprdI+uY6wMUl8/a2G+iag==",
+      "dev": true,
+      "license": "Apache-2.0",
+      "engines": {
+        "node": "^12.22.0 || ^14.17.0 || >=16.0.0"
+      },
+      "funding": {
+        "url": "https://opencollective.com/eslint"
+      }
+    },
+    "node_modules/@eslint-community/regexpp": {
+      "version": "4.12.2",
+      "resolved": "https://registry.npmjs.org/@eslint-community/regexpp/-/regexpp-4.12.2.tgz",
+      "integrity": "sha512-EriSTlt5OC9/7SXkRSCAhfSxxoSUgBm33OH+IkwbdpgoqsSsUg7y3uh+IICI/Qg4BBWr3U2i39RpmycbxMq4ew==",
+      "dev": true,
+      "license": "MIT",
+      "engines": {
+        "node": "^12.0.0 || ^14.0.0 || >=16.0.0"
+      }
+    },
+    "node_modules/@eslint/config-array": {
+      "version": "0.21.2",
+      "resolved": "https://registry.npmjs.org/@eslint/config-array/-/config-array-0.21.2.tgz",
+      "integrity": "sha512-nJl2KGTlrf9GjLimgIru+V/mzgSK0ABCDQRvxw5BjURL7WfH5uoWmizbH7QB6MmnMBd8cIC9uceWnezL1VZWWw==",
+      "dev": true,
+      "license": "Apache-2.0",
+      "dependencies": {
+        "@eslint/object-schema": "^2.1.7",
+        "debug": "^4.3.1",
+        "minimatch": "^3.1.5"
+      },
+      "engines": {
+        "node": "^18.18.0 || ^20.9.0 || >=21.1.0"
+      }
+    },
+    "node_modules/@eslint/config-helpers": {
+      "version": "0.4.2",
+      "resolved": "https://registry.npmjs.org/@eslint/config-helpers/-/config-helpers-0.4.2.tgz",
+      "integrity": "sha512-gBrxN88gOIf3R7ja5K9slwNayVcZgK6SOUORm2uBzTeIEfeVaIhOpCtTox3P6R7o2jLFwLFTLnC7kU/RGcYEgw==",
+      "dev": true,
+      "license": "Apache-2.0",
+      "dependencies": {
+        "@eslint/core": "^0.17.0"
+      },
+      "engines": {
+        "node": "^18.18.0 || ^20.9.0 || >=21.1.0"
+      }
+    },
+    "node_modules/@eslint/core": {
+      "version": "0.17.0",
+      "resolved": "https://registry.npmjs.org/@eslint/core/-/core-0.17.0.tgz",
+      "integrity": "sha512-yL/sLrpmtDaFEiUj1osRP4TI2MDz1AddJL+jZ7KSqvBuliN4xqYY54IfdN8qD8Toa6g1iloph1fxQNkjOxrrpQ==",
+      "dev": true,
+      "license": "Apache-2.0",
+      "dependencies": {
+        "@types/json-schema": "^7.0.15"
+      },
+      "engines": {
+        "node": "^18.18.0 || ^20.9.0 || >=21.1.0"
+      }
+    },
+    "node_modules/@eslint/eslintrc": {
+      "version": "3.3.5",
+      "resolved": "https://registry.npmjs.org/@eslint/eslintrc/-/eslintrc-3.3.5.tgz",
+      "integrity": "sha512-4IlJx0X0qftVsN5E+/vGujTRIFtwuLbNsVUe7TO6zYPDR1O6nFwvwhIKEKSrl6dZchmYBITazxKoUYOjdtjlRg==",
+      "dev": true,
+      "license": "MIT",
+      "dependencies": {
+        "ajv": "^6.14.0",
+        "debug": "^4.3.2",
+        "espree": "^10.0.1",
+        "globals": "^14.0.0",
+        "ignore": "^5.2.0",
+        "import-fresh": "^3.2.1",
+        "js-yaml": "^4.1.1",
+        "minimatch": "^3.1.5",
+        "strip-json-comments": "^3.1.1"
+      },
+      "engines": {
+        "node": "^18.18.0 || ^20.9.0 || >=21.1.0"
+      },
+      "funding": {
+        "url": "https://opencollective.com/eslint"
+      }
+    },
+    "node_modules/@eslint/js": {
+      "version": "9.39.4",
+      "resolved": "https://registry.npmjs.org/@eslint/js/-/js-9.39.4.tgz",
+      "integrity": "sha512-nE7DEIchvtiFTwBw4Lfbu59PG+kCofhjsKaCWzxTpt4lfRjRMqG6uMBzKXuEcyXhOHoUp9riAm7/aWYGhXZ9cw==",
+      "dev": true,
+      "license": "MIT",
+      "engines": {
+        "node": "^18.18.0 || ^20.9.0 || >=21.1.0"
+      },
+      "funding": {
+        "url": "https://eslint.org/donate"
+      }
+    },
+    "node_modules/@eslint/markdown": {
+      "version": "7.5.1",
+      "resolved": "https://registry.npmjs.org/@eslint/markdown/-/markdown-7.5.1.tgz",
+      "integrity": "sha512-R8uZemG9dKTbru/DQRPblbJyXpObwKzo8rv1KYGGuPUPtjM4LXBYM9q5CIZAComzZupws3tWbDwam5AFpPLyJQ==",
+      "dev": true,
+      "license": "MIT",
+      "workspaces": [
+        "examples/*"
+      ],
+      "dependencies": {
+        "@eslint/core": "^0.17.0",
+        "@eslint/plugin-kit": "^0.4.1",
+        "github-slugger": "^2.0.0",
+        "mdast-util-from-markdown": "^2.0.2",
+        "mdast-util-frontmatter": "^2.0.1",
+        "mdast-util-gfm": "^3.1.0",
+        "micromark-extension-frontmatter": "^2.0.0",
+        "micromark-extension-gfm": "^3.0.0",
+        "micromark-util-normalize-identifier": "^2.0.1"
+      },
+      "engines": {
+        "node": "^18.18.0 || ^20.9.0 || >=21.1.0"
+      }
+    },
+    "node_modules/@eslint/object-schema": {
+      "version": "2.1.7",
+      "resolved": "https://registry.npmjs.org/@eslint/object-schema/-/object-schema-2.1.7.tgz",
+      "integrity": "sha512-VtAOaymWVfZcmZbp6E2mympDIHvyjXs/12LqWYjVw6qjrfF+VK+fyG33kChz3nnK+SU5/NeHOqrTEHS8sXO3OA==",
+      "dev": true,
+      "license": "Apache-2.0",
+      "engines": {
+        "node": "^18.18.0 || ^20.9.0 || >=21.1.0"
+      }
+    },
+    "node_modules/@eslint/plugin-kit": {
+      "version": "0.4.1",
+      "resolved": "https://registry.npmjs.org/@eslint/plugin-kit/-/plugin-kit-0.4.1.tgz",
+      "integrity": "sha512-43/qtrDUokr7LJqoF2c3+RInu/t4zfrpYdoSDfYyhg52rwLV6TnOvdG4fXm7IkSB3wErkcmJS9iEhjVtOSEjjA==",
+      "dev": true,
+      "license": "Apache-2.0",
+      "dependencies": {
+        "@eslint/core": "^0.17.0",
+        "levn": "^0.4.1"
+      },
+      "engines": {
+        "node": "^18.18.0 || ^20.9.0 || >=21.1.0"
+      }
+    },
+    "node_modules/@humanfs/core": {
+      "version": "0.19.1",
+      "resolved": "https://registry.npmjs.org/@humanfs/core/-/core-0.19.1.tgz",
+      "integrity": "sha512-5DyQ4+1JEUzejeK1JGICcideyfUbGixgS9jNgex5nqkW+cY7WZhxBigmieN5Qnw9ZosSNVC9KQKyb+GUaGyKUA==",
+      "dev": true,
+      "license": "Apache-2.0",
+      "engines": {
+        "node": ">=18.18.0"
+      }
+    },
+    "node_modules/@humanfs/node": {
+      "version": "0.16.7",
+      "resolved": "https://registry.npmjs.org/@humanfs/node/-/node-0.16.7.tgz",
+      "integrity": "sha512-/zUx+yOsIrG4Y43Eh2peDeKCxlRt/gET6aHfaKpuq267qXdYDFViVHfMaLyygZOnl0kGWxFIgsBy8QFuTLUXEQ==",
+      "dev": true,
+      "license": "Apache-2.0",
+      "dependencies": {
+        "@humanfs/core": "^0.19.1",
+        "@humanwhocodes/retry": "^0.4.0"
+      },
+      "engines": {
+        "node": ">=18.18.0"
+      }
+    },
+    "node_modules/@humanwhocodes/module-importer": {
+      "version": "1.0.1",
+      "resolved": "https://registry.npmjs.org/@humanwhocodes/module-importer/-/module-importer-1.0.1.tgz",
+      "integrity": "sha512-bxveV4V8v5Yb4ncFTT3rPSgZBOpCkjfK0y4oVVVJwIuDVBRMDXrPyXRL988i5ap9m9bnyEEjWfm5WkBmtffLfA==",
+      "dev": true,
+      "license": "Apache-2.0",
+      "engines": {
+        "node": ">=12.22"
+      },
+      "funding": {
+        "type": "github",
+        "url": "https://github.com/sponsors/nzakas"
+      }
+    },
+    "node_modules/@humanwhocodes/retry": {
+      "version": "0.4.3",
+      "resolved": "https://registry.npmjs.org/@humanwhocodes/retry/-/retry-0.4.3.tgz",
+      "integrity": "sha512-bV0Tgo9K4hfPCek+aMAn81RppFKv2ySDQeMoSZuvTASywNTnVJCArCZE2FWqpvIatKu7VMRLWlR1EazvVhDyhQ==",
+      "dev": true,
+      "license": "Apache-2.0",
+      "engines": {
+        "node": ">=18.18"
+      },
+      "funding": {
+        "type": "github",
+        "url": "https://github.com/sponsors/nzakas"
+      }
+    },
+    "node_modules/@jridgewell/gen-mapping": {
+      "version": "0.3.13",
+      "resolved": "https://registry.npmjs.org/@jridgewell/gen-mapping/-/gen-mapping-0.3.13.tgz",
+      "integrity": "sha512-2kkt/7niJ6MgEPxF0bYdQ6etZaA+fQvDcLKckhy1yIQOzaoKjBBjSj63/aLVjYE3qhRt5dvM+uUyfCg6UKCBbA==",
+      "dev": true,
+      "license": "MIT",
+      "dependencies": {
+        "@jridgewell/sourcemap-codec": "^1.5.0",
+        "@jridgewell/trace-mapping": "^0.3.24"
+      }
+    },
+    "node_modules/@jridgewell/resolve-uri": {
+      "version": "3.1.2",
+      "resolved": "https://registry.npmjs.org/@jridgewell/resolve-uri/-/resolve-uri-3.1.2.tgz",
+      "integrity": "sha512-bRISgCIjP20/tbWSPWMEi54QVPRZExkuD9lJL+UIxUKtwVJA8wW1Trb1jMs1RFXo1CBTNZ/5hpC9QvmKWdopKw==",
+      "dev": true,
+      "license": "MIT",
+      "engines": {
+        "node": ">=6.0.0"
+      }
+    },
+    "node_modules/@jridgewell/sourcemap-codec": {
+      "version": "1.5.5",
+      "resolved": "https://registry.npmjs.org/@jridgewell/sourcemap-codec/-/sourcemap-codec-1.5.5.tgz",
+      "integrity": "sha512-cYQ9310grqxueWbl+WuIUIaiUaDcj7WOq5fVhEljNVgRfOUhY9fy2zTvfoqWsnebh8Sl70VScFbICvJnLKB0Og==",
+      "dev": true,
+      "license": "MIT"
+    },
+    "node_modules/@jridgewell/trace-mapping": {
+      "version": "0.3.31",
+      "resolved": "https://registry.npmjs.org/@jridgewell/trace-mapping/-/trace-mapping-0.3.31.tgz",
+      "integrity": "sha512-zzNR+SdQSDJzc8joaeP8QQoCQr8NuYx2dIIytl1QeBEZHJ9uW6hebsrYgbz8hJwUQao3TWCMtmfV8Nu1twOLAw==",
+      "dev": true,
+      "license": "MIT",
+      "dependencies": {
+        "@jridgewell/resolve-uri": "^3.1.0",
+        "@jridgewell/sourcemap-codec": "^1.4.14"
+      }
+    },
+    "node_modules/@napi-rs/wasm-runtime": {
+      "version": "1.1.1",
+      "resolved": "https://registry.npmjs.org/@napi-rs/wasm-runtime/-/wasm-runtime-1.1.1.tgz",
+      "integrity": "sha512-p64ah1M1ld8xjWv3qbvFwHiFVWrq1yFvV4f7w+mzaqiR4IlSgkqhcRdHwsGgomwzBH51sRY4NEowLxnaBjcW/A==",
+      "dev": true,
+      "license": "MIT",
+      "optional": true,
+      "dependencies": {
+        "@emnapi/core": "^1.7.1",
+        "@emnapi/runtime": "^1.7.1",
+        "@tybys/wasm-util": "^0.10.1"
+      },
+      "funding": {
+        "type": "github",
+        "url": "https://github.com/sponsors/Brooooooklyn"
+      }
+    },
+    "node_modules/@oxc-project/types": {
+      "version": "0.115.0",
+      "resolved": "https://registry.npmjs.org/@oxc-project/types/-/types-0.115.0.tgz",
+      "integrity": "sha512-4n91DKnebUS4yjUHl2g3/b2T+IUdCfmoZGhmwsovZCDaJSs+QkVAM+0AqqTxHSsHfeiMuueT75cZaZcT/m0pSw==",
+      "dev": true,
+      "license": "MIT",
+      "funding": {
+        "url": "https://github.com/sponsors/Boshen"
+      }
+    },
+    "node_modules/@parcel/watcher": {
+      "version": "2.5.6",
+      "resolved": "https://registry.npmjs.org/@parcel/watcher/-/watcher-2.5.6.tgz",
+      "integrity": "sha512-tmmZ3lQxAe/k/+rNnXQRawJ4NjxO2hqiOLTHvWchtGZULp4RyFeh6aU4XdOYBFe2KE1oShQTv4AblOs2iOrNnQ==",
+      "dev": true,
+      "hasInstallScript": true,
+      "license": "MIT",
+      "dependencies": {
+        "detect-libc": "^2.0.3",
+        "is-glob": "^4.0.3",
+        "node-addon-api": "^7.0.0",
+        "picomatch": "^4.0.3"
+      },
+      "engines": {
+        "node": ">= 10.0.0"
+      },
+      "funding": {
+        "type": "opencollective",
+        "url": "https://opencollective.com/parcel"
+      },
+      "optionalDependencies": {
+        "@parcel/watcher-android-arm64": "2.5.6",
+        "@parcel/watcher-darwin-arm64": "2.5.6",
+        "@parcel/watcher-darwin-x64": "2.5.6",
+        "@parcel/watcher-freebsd-x64": "2.5.6",
+        "@parcel/watcher-linux-arm-glibc": "2.5.6",
+        "@parcel/watcher-linux-arm-musl": "2.5.6",
+        "@parcel/watcher-linux-arm64-glibc": "2.5.6",
+        "@parcel/watcher-linux-arm64-musl": "2.5.6",
+        "@parcel/watcher-linux-x64-glibc": "2.5.6",
+        "@parcel/watcher-linux-x64-musl": "2.5.6",
+        "@parcel/watcher-win32-arm64": "2.5.6",
+        "@parcel/watcher-win32-ia32": "2.5.6",
+        "@parcel/watcher-win32-x64": "2.5.6"
+      }
+    },
+    "node_modules/@parcel/watcher-android-arm64": {
+      "version": "2.5.6",
+      "resolved": "https://registry.npmjs.org/@parcel/watcher-android-arm64/-/watcher-android-arm64-2.5.6.tgz",
+      "integrity": "sha512-YQxSS34tPF/6ZG7r/Ih9xy+kP/WwediEUsqmtf0cuCV5TPPKw/PQHRhueUo6JdeFJaqV3pyjm0GdYjZotbRt/A==",
+      "cpu": [
+        "arm64"
+      ],
+      "dev": true,
+      "license": "MIT",
+      "optional": true,
+      "os": [
+        "android"
+      ],
+      "engines": {
+        "node": ">= 10.0.0"
+      },
+      "funding": {
+        "type": "opencollective",
+        "url": "https://opencollective.com/parcel"
+      }
+    },
+    "node_modules/@parcel/watcher-darwin-arm64": {
+      "version": "2.5.6",
+      "resolved": "https://registry.npmjs.org/@parcel/watcher-darwin-arm64/-/watcher-darwin-arm64-2.5.6.tgz",
+      "integrity": "sha512-Z2ZdrnwyXvvvdtRHLmM4knydIdU9adO3D4n/0cVipF3rRiwP+3/sfzpAwA/qKFL6i1ModaabkU7IbpeMBgiVEA==",
+      "cpu": [
+        "arm64"
+      ],
+      "dev": true,
+      "license": "MIT",
+      "optional": true,
+      "os": [
+        "darwin"
+      ],
+      "engines": {
+        "node": ">= 10.0.0"
+      },
+      "funding": {
+        "type": "opencollective",
+        "url": "https://opencollective.com/parcel"
+      }
+    },
+    "node_modules/@parcel/watcher-darwin-x64": {
+      "version": "2.5.6",
+      "resolved": "https://registry.npmjs.org/@parcel/watcher-darwin-x64/-/watcher-darwin-x64-2.5.6.tgz",
+      "integrity": "sha512-HgvOf3W9dhithcwOWX9uDZyn1lW9R+7tPZ4sug+NGrGIo4Rk1hAXLEbcH1TQSqxts0NYXXlOWqVpvS1SFS4fRg==",
+      "cpu": [
+        "x64"
+      ],
+      "dev": true,
+      "license": "MIT",
+      "optional": true,
+      "os": [
+        "darwin"
+      ],
+      "engines": {
+        "node": ">= 10.0.0"
+      },
+      "funding": {
+        "type": "opencollective",
+        "url": "https://opencollective.com/parcel"
+      }
+    },
+    "node_modules/@parcel/watcher-freebsd-x64": {
+      "version": "2.5.6",
+      "resolved": "https://registry.npmjs.org/@parcel/watcher-freebsd-x64/-/watcher-freebsd-x64-2.5.6.tgz",
+      "integrity": "sha512-vJVi8yd/qzJxEKHkeemh7w3YAn6RJCtYlE4HPMoVnCpIXEzSrxErBW5SJBgKLbXU3WdIpkjBTeUNtyBVn8TRng==",
+      "cpu": [
+        "x64"
+      ],
+      "dev": true,
+      "license": "MIT",
+      "optional": true,
+      "os": [
+        "freebsd"
+      ],
+      "engines": {
+        "node": ">= 10.0.0"
+      },
+      "funding": {
+        "type": "opencollective",
+        "url": "https://opencollective.com/parcel"
+      }
+    },
+    "node_modules/@parcel/watcher-linux-arm-glibc": {
+      "version": "2.5.6",
+      "resolved": "https://registry.npmjs.org/@parcel/watcher-linux-arm-glibc/-/watcher-linux-arm-glibc-2.5.6.tgz",
+      "integrity": "sha512-9JiYfB6h6BgV50CCfasfLf/uvOcJskMSwcdH1PHH9rvS1IrNy8zad6IUVPVUfmXr+u+Km9IxcfMLzgdOudz9EQ==",
+      "cpu": [
+        "arm"
+      ],
+      "dev": true,
+      "license": "MIT",
+      "optional": true,
+      "os": [
+        "linux"
+      ],
+      "engines": {
+        "node": ">= 10.0.0"
+      },
+      "funding": {
+        "type": "opencollective",
+        "url": "https://opencollective.com/parcel"
+      }
+    },
+    "node_modules/@parcel/watcher-linux-arm-musl": {
+      "version": "2.5.6",
+      "resolved": "https://registry.npmjs.org/@parcel/watcher-linux-arm-musl/-/watcher-linux-arm-musl-2.5.6.tgz",
+      "integrity": "sha512-Ve3gUCG57nuUUSyjBq/MAM0CzArtuIOxsBdQ+ftz6ho8n7s1i9E1Nmk/xmP323r2YL0SONs1EuwqBp2u1k5fxg==",
+      "cpu": [
+        "arm"
+      ],
+      "dev": true,
+      "license": "MIT",
+      "optional": true,
+      "os": [
+        "linux"
+      ],
+      "engines": {
+        "node": ">= 10.0.0"
+      },
+      "funding": {
+        "type": "opencollective",
+        "url": "https://opencollective.com/parcel"
+      }
+    },
+    "node_modules/@parcel/watcher-linux-arm64-glibc": {
+      "version": "2.5.6",
+      "resolved": "https://registry.npmjs.org/@parcel/watcher-linux-arm64-glibc/-/watcher-linux-arm64-glibc-2.5.6.tgz",
+      "integrity": "sha512-f2g/DT3NhGPdBmMWYoxixqYr3v/UXcmLOYy16Bx0TM20Tchduwr4EaCbmxh1321TABqPGDpS8D/ggOTaljijOA==",
+      "cpu": [
+        "arm64"
+      ],
+      "dev": true,
+      "license": "MIT",
+      "optional": true,
+      "os": [
+        "linux"
+      ],
+      "engines": {
+        "node": ">= 10.0.0"
+      },
+      "funding": {
+        "type": "opencollective",
+        "url": "https://opencollective.com/parcel"
+      }
+    },
+    "node_modules/@parcel/watcher-linux-arm64-musl": {
+      "version": "2.5.6",
+      "resolved": "https://registry.npmjs.org/@parcel/watcher-linux-arm64-musl/-/watcher-linux-arm64-musl-2.5.6.tgz",
+      "integrity": "sha512-qb6naMDGlbCwdhLj6hgoVKJl2odL34z2sqkC7Z6kzir8b5W65WYDpLB6R06KabvZdgoHI/zxke4b3zR0wAbDTA==",
+      "cpu": [
+        "arm64"
+      ],
+      "dev": true,
+      "license": "MIT",
+      "optional": true,
+      "os": [
+        "linux"
+      ],
+      "engines": {
+        "node": ">= 10.0.0"
+      },
+      "funding": {
+        "type": "opencollective",
+        "url": "https://opencollective.com/parcel"
+      }
+    },
+    "node_modules/@parcel/watcher-linux-x64-glibc": {
+      "version": "2.5.6",
+      "resolved": "https://registry.npmjs.org/@parcel/watcher-linux-x64-glibc/-/watcher-linux-x64-glibc-2.5.6.tgz",
+      "integrity": "sha512-kbT5wvNQlx7NaGjzPFu8nVIW1rWqV780O7ZtkjuWaPUgpv2NMFpjYERVi0UYj1msZNyCzGlaCWEtzc+exjMGbQ==",
+      "cpu": [
+        "x64"
+      ],
+      "dev": true,
+      "license": "MIT",
+      "optional": true,
+      "os": [
+        "linux"
+      ],
+      "engines": {
+        "node": ">= 10.0.0"
+      },
+      "funding": {
+        "type": "opencollective",
+        "url": "https://opencollective.com/parcel"
+      }
+    },
+    "node_modules/@parcel/watcher-linux-x64-musl": {
+      "version": "2.5.6",
+      "resolved": "https://registry.npmjs.org/@parcel/watcher-linux-x64-musl/-/watcher-linux-x64-musl-2.5.6.tgz",
+      "integrity": "sha512-1JRFeC+h7RdXwldHzTsmdtYR/Ku8SylLgTU/reMuqdVD7CtLwf0VR1FqeprZ0eHQkO0vqsbvFLXUmYm/uNKJBg==",
+      "cpu": [
+        "x64"
+      ],
+      "dev": true,
+      "license": "MIT",
+      "optional": true,
+      "os": [
+        "linux"
+      ],
+      "engines": {
+        "node": ">= 10.0.0"
+      },
+      "funding": {
+        "type": "opencollective",
+        "url": "https://opencollective.com/parcel"
+      }
+    },
+    "node_modules/@parcel/watcher-win32-arm64": {
+      "version": "2.5.6",
+      "resolved": "https://registry.npmjs.org/@parcel/watcher-win32-arm64/-/watcher-win32-arm64-2.5.6.tgz",
+      "integrity": "sha512-3ukyebjc6eGlw9yRt678DxVF7rjXatWiHvTXqphZLvo7aC5NdEgFufVwjFfY51ijYEWpXbqF5jtrK275z52D4Q==",
+      "cpu": [
+        "arm64"
+      ],
+      "dev": true,
+      "license": "MIT",
+      "optional": true,
+      "os": [
+        "win32"
+      ],
+      "engines": {
+        "node": ">= 10.0.0"
+      },
+      "funding": {
+        "type": "opencollective",
+        "url": "https://opencollective.com/parcel"
+      }
+    },
+    "node_modules/@parcel/watcher-win32-ia32": {
+      "version": "2.5.6",
+      "resolved": "https://registry.npmjs.org/@parcel/watcher-win32-ia32/-/watcher-win32-ia32-2.5.6.tgz",
+      "integrity": "sha512-k35yLp1ZMwwee3Ez/pxBi5cf4AoBKYXj00CZ80jUz5h8prpiaQsiRPKQMxoLstNuqe2vR4RNPEAEcjEFzhEz/g==",
+      "cpu": [
+        "ia32"
+      ],
+      "dev": true,
+      "license": "MIT",
+      "optional": true,
+      "os": [
+        "win32"
+      ],
+      "engines": {
+        "node": ">= 10.0.0"
+      },
+      "funding": {
+        "type": "opencollective",
+        "url": "https://opencollective.com/parcel"
+      }
+    },
+    "node_modules/@parcel/watcher-win32-x64": {
+      "version": "2.5.6",
+      "resolved": "https://registry.npmjs.org/@parcel/watcher-win32-x64/-/watcher-win32-x64-2.5.6.tgz",
+      "integrity": "sha512-hbQlYcCq5dlAX9Qx+kFb0FHue6vbjlf0FrNzSKdYK2APUf7tGfGxQCk2ihEREmbR6ZMc0MVAD5RIX/41gpUzTw==",
+      "cpu": [
+        "x64"
+      ],
+      "dev": true,
+      "license": "MIT",
+      "optional": true,
+      "os": [
+        "win32"
+      ],
+      "engines": {
+        "node": ">= 10.0.0"
+      },
+      "funding": {
+        "type": "opencollective",
+        "url": "https://opencollective.com/parcel"
+      }
+    },
+    "node_modules/@rolldown/binding-android-arm64": {
+      "version": "1.0.0-rc.8",
+      "resolved": "https://registry.npmjs.org/@rolldown/binding-android-arm64/-/binding-android-arm64-1.0.0-rc.8.tgz",
+      "integrity": "sha512-5bcmMQDWEfWUq3m79Mcf/kbO6e5Jr6YjKSsA1RnpXR6k73hQ9z1B17+4h93jXpzHvS18p7bQHM1HN/fSd+9zog==",
+      "cpu": [
+        "arm64"
+      ],
+      "dev": true,
+      "license": "MIT",
+      "optional": true,
+      "os": [
+        "android"
+      ],
+      "engines": {
+        "node": "^20.19.0 || >=22.12.0"
+      }
+    },
+    "node_modules/@rolldown/binding-darwin-arm64": {
+      "version": "1.0.0-rc.8",
+      "resolved": "https://registry.npmjs.org/@rolldown/binding-darwin-arm64/-/binding-darwin-arm64-1.0.0-rc.8.tgz",
+      "integrity": "sha512-dcHPd5N4g9w2iiPRJmAvO0fsIWzF2JPr9oSuTjxLL56qu+oML5aMbBMNwWbk58Mt3pc7vYs9CCScwLxdXPdRsg==",
+      "cpu": [
+        "arm64"
+      ],
+      "dev": true,
+      "license": "MIT",
+      "optional": true,
+      "os": [
+        "darwin"
+      ],
+      "engines": {
+        "node": "^20.19.0 || >=22.12.0"
+      }
+    },
+    "node_modules/@rolldown/binding-darwin-x64": {
+      "version": "1.0.0-rc.8",
+      "resolved": "https://registry.npmjs.org/@rolldown/binding-darwin-x64/-/binding-darwin-x64-1.0.0-rc.8.tgz",
+      "integrity": "sha512-mw0VzDvoj8AuR761QwpdCFN0sc/jspuc7eRYJetpLWd+XyansUrH3C7IgNw6swBOgQT9zBHNKsVCjzpfGJlhUA==",
+      "cpu": [
+        "x64"
+      ],
+      "dev": true,
+      "license": "MIT",
+      "optional": true,
+      "os": [
+        "darwin"
+      ],
+      "engines": {
+        "node": "^20.19.0 || >=22.12.0"
+      }
+    },
+    "node_modules/@rolldown/binding-freebsd-x64": {
+      "version": "1.0.0-rc.8",
+      "resolved": "https://registry.npmjs.org/@rolldown/binding-freebsd-x64/-/binding-freebsd-x64-1.0.0-rc.8.tgz",
+      "integrity": "sha512-xNrRa6mQ9NmMIJBdJtPMPG8Mso0OhM526pDzc/EKnRrIrrkHD1E0Z6tONZRmUeJElfsQ6h44lQQCcDilSNIvSQ==",
+      "cpu": [
+        "x64"
+      ],
+      "dev": true,
+      "license": "MIT",
+      "optional": true,
+      "os": [
+        "freebsd"
+      ],
+      "engines": {
+        "node": "^20.19.0 || >=22.12.0"
+      }
+    },
+    "node_modules/@rolldown/binding-linux-arm-gnueabihf": {
+      "version": "1.0.0-rc.8",
+      "resolved": "https://registry.npmjs.org/@rolldown/binding-linux-arm-gnueabihf/-/binding-linux-arm-gnueabihf-1.0.0-rc.8.tgz",
+      "integrity": "sha512-WgCKoO6O/rRUwimWfEJDeztwJJmuuX0N2bYLLRxmXDTtCwjToTOqk7Pashl/QpQn3H/jHjx0b5yCMbcTVYVpNg==",
+      "cpu": [
+        "arm"
+      ],
+      "dev": true,
+      "license": "MIT",
+      "optional": true,
+      "os": [
+        "linux"
+      ],
+      "engines": {
+        "node": "^20.19.0 || >=22.12.0"
+      }
+    },
+    "node_modules/@rolldown/binding-linux-arm64-gnu": {
+      "version": "1.0.0-rc.8",
+      "resolved": "https://registry.npmjs.org/@rolldown/binding-linux-arm64-gnu/-/binding-linux-arm64-gnu-1.0.0-rc.8.tgz",
+      "integrity": "sha512-tOHgTOQa8G4Z3ULj4G3NYOGGJEsqPHR91dT72u63OtVsZ7B6wFJKOx+ZKv+pvwzxWz92/I2ycaqi2/Ll4l+rlg==",
+      "cpu": [
+        "arm64"
+      ],
+      "dev": true,
+      "license": "MIT",
+      "optional": true,
+      "os": [
+        "linux"
+      ],
+      "engines": {
+        "node": "^20.19.0 || >=22.12.0"
+      }
+    },
+    "node_modules/@rolldown/binding-linux-arm64-musl": {
+      "version": "1.0.0-rc.8",
+      "resolved": "https://registry.npmjs.org/@rolldown/binding-linux-arm64-musl/-/binding-linux-arm64-musl-1.0.0-rc.8.tgz",
+      "integrity": "sha512-oRbxcgDujCi2Yp1GTxoUFsIFlZsuPHU4OV4AzNc3/6aUmR4lfm9FK0uwQu82PJsuUwnF2jFdop3Ep5c1uK7Uxg==",
+      "cpu": [
+        "arm64"
+      ],
+      "dev": true,
+      "license": "MIT",
+      "optional": true,
+      "os": [
+        "linux"
+      ],
+      "engines": {
+        "node": "^20.19.0 || >=22.12.0"
+      }
+    },
+    "node_modules/@rolldown/binding-linux-ppc64-gnu": {
+      "version": "1.0.0-rc.8",
+      "resolved": "https://registry.npmjs.org/@rolldown/binding-linux-ppc64-gnu/-/binding-linux-ppc64-gnu-1.0.0-rc.8.tgz",
+      "integrity": "sha512-oaLRyUHw8kQE5M89RqrDJZ10GdmGJcMeCo8tvaE4ukOofqgjV84AbqBSH6tTPjeT2BHv+xlKj678GBuIb47lKA==",
+      "cpu": [
+        "ppc64"
+      ],
+      "dev": true,
+      "license": "MIT",
+      "optional": true,
+      "os": [
+        "linux"
+      ],
+      "engines": {
+        "node": "^20.19.0 || >=22.12.0"
+      }
+    },
+    "node_modules/@rolldown/binding-linux-s390x-gnu": {
+      "version": "1.0.0-rc.8",
+      "resolved": "https://registry.npmjs.org/@rolldown/binding-linux-s390x-gnu/-/binding-linux-s390x-gnu-1.0.0-rc.8.tgz",
+      "integrity": "sha512-1hjSKFrod5MwBBdLOOA0zpUuSfSDkYIY+QqcMcIU1WOtswZtZdUkcFcZza9b2HcAb0bnpmmyo0LZcaxLb2ov1g==",
+      "cpu": [
+        "s390x"
+      ],
+      "dev": true,
+      "license": "MIT",
+      "optional": true,
+      "os": [
+        "linux"
+      ],
+      "engines": {
+        "node": "^20.19.0 || >=22.12.0"
+      }
+    },
+    "node_modules/@rolldown/binding-linux-x64-gnu": {
+      "version": "1.0.0-rc.8",
+      "resolved": "https://registry.npmjs.org/@rolldown/binding-linux-x64-gnu/-/binding-linux-x64-gnu-1.0.0-rc.8.tgz",
+      "integrity": "sha512-a1+F0aV4Wy9tT3o+cHl3XhOy6aFV+B8Ll+/JFj98oGkb6lGk3BNgrxd+80RwYRVd23oLGvj3LwluKYzlv1PEuw==",
+      "cpu": [
+        "x64"
+      ],
+      "dev": true,
+      "license": "MIT",
+      "optional": true,
+      "os": [
+        "linux"
+      ],
+      "engines": {
+        "node": "^20.19.0 || >=22.12.0"
+      }
+    },
+    "node_modules/@rolldown/binding-linux-x64-musl": {
+      "version": "1.0.0-rc.8",
+      "resolved": "https://registry.npmjs.org/@rolldown/binding-linux-x64-musl/-/binding-linux-x64-musl-1.0.0-rc.8.tgz",
+      "integrity": "sha512-bGyXCFU11seFrf7z8PcHSwGEiFVkZ9vs+auLacVOQrVsI8PFHJzzJROF3P6b0ODDmXr0m6Tj5FlDhcXVk0Jp8w==",
+      "cpu": [
+        "x64"
+      ],
+      "dev": true,
+      "license": "MIT",
+      "optional": true,
+      "os": [
+        "linux"
+      ],
+      "engines": {
+        "node": "^20.19.0 || >=22.12.0"
+      }
+    },
+    "node_modules/@rolldown/binding-openharmony-arm64": {
+      "version": "1.0.0-rc.8",
+      "resolved": "https://registry.npmjs.org/@rolldown/binding-openharmony-arm64/-/binding-openharmony-arm64-1.0.0-rc.8.tgz",
+      "integrity": "sha512-n8d+L2bKgf9G3+AM0bhHFWdlz9vYKNim39ujRTieukdRek0RAo2TfG2uEnV9spa4r4oHUfL9IjcY3M9SlqN1gw==",
+      "cpu": [
+        "arm64"
+      ],
+      "dev": true,
+      "license": "MIT",
+      "optional": true,
+      "os": [
+        "openharmony"
+      ],
+      "engines": {
+        "node": "^20.19.0 || >=22.12.0"
+      }
+    },
+    "node_modules/@rolldown/binding-wasm32-wasi": {
+      "version": "1.0.0-rc.8",
+      "resolved": "https://registry.npmjs.org/@rolldown/binding-wasm32-wasi/-/binding-wasm32-wasi-1.0.0-rc.8.tgz",
+      "integrity": "sha512-4R4iJDIk7BrJdteAbEAICXPoA7vZoY/M0OBfcRlQxzQvUYMcEp2GbC/C8UOgQJhu2TjGTpX1H8vVO1xHWcRqQA==",
+      "cpu": [
+        "wasm32"
+      ],
+      "dev": true,
+      "license": "MIT",
+      "optional": true,
+      "dependencies": {
+        "@napi-rs/wasm-runtime": "^1.1.1"
+      },
+      "engines": {
+        "node": ">=14.0.0"
+      }
+    },
+    "node_modules/@rolldown/binding-win32-arm64-msvc": {
+      "version": "1.0.0-rc.8",
+      "resolved": "https://registry.npmjs.org/@rolldown/binding-win32-arm64-msvc/-/binding-win32-arm64-msvc-1.0.0-rc.8.tgz",
+      "integrity": "sha512-3lwnklba9qQOpFnQ7EW+A1m4bZTWXZE4jtehsZ0YOl2ivW1FQqp5gY7X2DLuKITggesyuLwcmqS11fA7NtrmrA==",
+      "cpu": [
+        "arm64"
+      ],
+      "dev": true,
+      "license": "MIT",
+      "optional": true,
+      "os": [
+        "win32"
+      ],
+      "engines": {
+        "node": "^20.19.0 || >=22.12.0"
+      }
+    },
+    "node_modules/@rolldown/binding-win32-x64-msvc": {
+      "version": "1.0.0-rc.8",
+      "resolved": "https://registry.npmjs.org/@rolldown/binding-win32-x64-msvc/-/binding-win32-x64-msvc-1.0.0-rc.8.tgz",
+      "integrity": "sha512-VGjCx9Ha1P/r3tXGDZyG0Fcq7Q0Afnk64aaKzr1m40vbn1FL8R3W0V1ELDvPgzLXaaqK/9PnsqSaLWXfn6JtGQ==",
+      "cpu": [
+        "x64"
+      ],
+      "dev": true,
+      "license": "MIT",
+      "optional": true,
+      "os": [
+        "win32"
+      ],
+      "engines": {
+        "node": "^20.19.0 || >=22.12.0"
+      }
+    },
+    "node_modules/@rolldown/pluginutils": {
+      "version": "1.0.0-rc.8",
+      "resolved": "https://registry.npmjs.org/@rolldown/pluginutils/-/pluginutils-1.0.0-rc.8.tgz",
+      "integrity": "sha512-wzJwL82/arVfeSP3BLr1oTy40XddjtEdrdgtJ4lLRBu06mP3q/8HGM6K0JRlQuTA3XB0pNJx2so/nmpY4xyOew==",
+      "dev": true,
+      "license": "MIT"
+    },
+    "node_modules/@rollup/rollup-android-arm-eabi": {
+      "version": "4.59.0",
+      "resolved": "https://registry.npmjs.org/@rollup/rollup-android-arm-eabi/-/rollup-android-arm-eabi-4.59.0.tgz",
+      "integrity": "sha512-upnNBkA6ZH2VKGcBj9Fyl9IGNPULcjXRlg0LLeaioQWueH30p6IXtJEbKAgvyv+mJaMxSm1l6xwDXYjpEMiLMg==",
+      "cpu": [
+        "arm"
+      ],
+      "dev": true,
+      "license": "MIT",
+      "optional": true,
+      "os": [
+        "android"
+      ]
+    },
+    "node_modules/@rollup/rollup-android-arm64": {
+      "version": "4.59.0",
+      "resolved": "https://registry.npmjs.org/@rollup/rollup-android-arm64/-/rollup-android-arm64-4.59.0.tgz",
+      "integrity": "sha512-hZ+Zxj3SySm4A/DylsDKZAeVg0mvi++0PYVceVyX7hemkw7OreKdCvW2oQ3T1FMZvCaQXqOTHb8qmBShoqk69Q==",
+      "cpu": [
+        "arm64"
+      ],
+      "dev": true,
+      "license": "MIT",
+      "optional": true,
+      "os": [
+        "android"
+      ]
+    },
+    "node_modules/@rollup/rollup-darwin-arm64": {
+      "version": "4.59.0",
+      "resolved": "https://registry.npmjs.org/@rollup/rollup-darwin-arm64/-/rollup-darwin-arm64-4.59.0.tgz",
+      "integrity": "sha512-W2Psnbh1J8ZJw0xKAd8zdNgF9HRLkdWwwdWqubSVk0pUuQkoHnv7rx4GiF9rT4t5DIZGAsConRE3AxCdJ4m8rg==",
+      "cpu": [
+        "arm64"
+      ],
+      "dev": true,
+      "license": "MIT",
+      "optional": true,
+      "os": [
+        "darwin"
+      ]
+    },
+    "node_modules/@rollup/rollup-darwin-x64": {
+      "version": "4.59.0",
+      "resolved": "https://registry.npmjs.org/@rollup/rollup-darwin-x64/-/rollup-darwin-x64-4.59.0.tgz",
+      "integrity": "sha512-ZW2KkwlS4lwTv7ZVsYDiARfFCnSGhzYPdiOU4IM2fDbL+QGlyAbjgSFuqNRbSthybLbIJ915UtZBtmuLrQAT/w==",
+      "cpu": [
+        "x64"
+      ],
+      "dev": true,
+      "license": "MIT",
+      "optional": true,
+      "os": [
+        "darwin"
+      ]
+    },
+    "node_modules/@rollup/rollup-freebsd-arm64": {
+      "version": "4.59.0",
+      "resolved": "https://registry.npmjs.org/@rollup/rollup-freebsd-arm64/-/rollup-freebsd-arm64-4.59.0.tgz",
+      "integrity": "sha512-EsKaJ5ytAu9jI3lonzn3BgG8iRBjV4LxZexygcQbpiU0wU0ATxhNVEpXKfUa0pS05gTcSDMKpn3Sx+QB9RlTTA==",
+      "cpu": [
+        "arm64"
+      ],
+      "dev": true,
+      "license": "MIT",
+      "optional": true,
+      "os": [
+        "freebsd"
+      ]
+    },
+    "node_modules/@rollup/rollup-freebsd-x64": {
+      "version": "4.59.0",
+      "resolved": "https://registry.npmjs.org/@rollup/rollup-freebsd-x64/-/rollup-freebsd-x64-4.59.0.tgz",
+      "integrity": "sha512-d3DuZi2KzTMjImrxoHIAODUZYoUUMsuUiY4SRRcJy6NJoZ6iIqWnJu9IScV9jXysyGMVuW+KNzZvBLOcpdl3Vg==",
+      "cpu": [
+        "x64"
+      ],
+      "dev": true,
+      "license": "MIT",
+      "optional": true,
+      "os": [
+        "freebsd"
+      ]
+    },
+    "node_modules/@rollup/rollup-linux-arm-gnueabihf": {
+      "version": "4.59.0",
+      "resolved": "https://registry.npmjs.org/@rollup/rollup-linux-arm-gnueabihf/-/rollup-linux-arm-gnueabihf-4.59.0.tgz",
+      "integrity": "sha512-t4ONHboXi/3E0rT6OZl1pKbl2Vgxf9vJfWgmUoCEVQVxhW6Cw/c8I6hbbu7DAvgp82RKiH7TpLwxnJeKv2pbsw==",
+      "cpu": [
+        "arm"
+      ],
+      "dev": true,
+      "license": "MIT",
+      "optional": true,
+      "os": [
+        "linux"
+      ]
+    },
+    "node_modules/@rollup/rollup-linux-arm-musleabihf": {
+      "version": "4.59.0",
+      "resolved": "https://registry.npmjs.org/@rollup/rollup-linux-arm-musleabihf/-/rollup-linux-arm-musleabihf-4.59.0.tgz",
+      "integrity": "sha512-CikFT7aYPA2ufMD086cVORBYGHffBo4K8MQ4uPS/ZnY54GKj36i196u8U+aDVT2LX4eSMbyHtyOh7D7Zvk2VvA==",
+      "cpu": [
+        "arm"
+      ],
+      "dev": true,
+      "license": "MIT",
+      "optional": true,
+      "os": [
+        "linux"
+      ]
+    },
+    "node_modules/@rollup/rollup-linux-arm64-gnu": {
+      "version": "4.59.0",
+      "resolved": "https://registry.npmjs.org/@rollup/rollup-linux-arm64-gnu/-/rollup-linux-arm64-gnu-4.59.0.tgz",
+      "integrity": "sha512-jYgUGk5aLd1nUb1CtQ8E+t5JhLc9x5WdBKew9ZgAXg7DBk0ZHErLHdXM24rfX+bKrFe+Xp5YuJo54I5HFjGDAA==",
+      "cpu": [
+        "arm64"
+      ],
+      "dev": true,
+      "license": "MIT",
+      "optional": true,
+      "os": [
+        "linux"
+      ]
+    },
+    "node_modules/@rollup/rollup-linux-arm64-musl": {
+      "version": "4.59.0",
+      "resolved": "https://registry.npmjs.org/@rollup/rollup-linux-arm64-musl/-/rollup-linux-arm64-musl-4.59.0.tgz",
+      "integrity": "sha512-peZRVEdnFWZ5Bh2KeumKG9ty7aCXzzEsHShOZEFiCQlDEepP1dpUl/SrUNXNg13UmZl+gzVDPsiCwnV1uI0RUA==",
+      "cpu": [
+        "arm64"
+      ],
+      "dev": true,
+      "license": "MIT",
+      "optional": true,
+      "os": [
+        "linux"
+      ]
+    },
+    "node_modules/@rollup/rollup-linux-loong64-gnu": {
+      "version": "4.59.0",
+      "resolved": "https://registry.npmjs.org/@rollup/rollup-linux-loong64-gnu/-/rollup-linux-loong64-gnu-4.59.0.tgz",
+      "integrity": "sha512-gbUSW/97f7+r4gHy3Jlup8zDG190AuodsWnNiXErp9mT90iCy9NKKU0Xwx5k8VlRAIV2uU9CsMnEFg/xXaOfXg==",
+      "cpu": [
+        "loong64"
+      ],
+      "dev": true,
+      "license": "MIT",
+      "optional": true,
+      "os": [
+        "linux"
+      ]
+    },
+    "node_modules/@rollup/rollup-linux-loong64-musl": {
+      "version": "4.59.0",
+      "resolved": "https://registry.npmjs.org/@rollup/rollup-linux-loong64-musl/-/rollup-linux-loong64-musl-4.59.0.tgz",
+      "integrity": "sha512-yTRONe79E+o0FWFijasoTjtzG9EBedFXJMl888NBEDCDV9I2wGbFFfJQQe63OijbFCUZqxpHz1GzpbtSFikJ4Q==",
+      "cpu": [
+        "loong64"
+      ],
+      "dev": true,
+      "license": "MIT",
+      "optional": true,
+      "os": [
+        "linux"
+      ]
+    },
+    "node_modules/@rollup/rollup-linux-ppc64-gnu": {
+      "version": "4.59.0",
+      "resolved": "https://registry.npmjs.org/@rollup/rollup-linux-ppc64-gnu/-/rollup-linux-ppc64-gnu-4.59.0.tgz",
+      "integrity": "sha512-sw1o3tfyk12k3OEpRddF68a1unZ5VCN7zoTNtSn2KndUE+ea3m3ROOKRCZxEpmT9nsGnogpFP9x6mnLTCaoLkA==",
+      "cpu": [
+        "ppc64"
+      ],
+      "dev": true,
+      "license": "MIT",
+      "optional": true,
+      "os": [
+        "linux"
+      ]
+    },
+    "node_modules/@rollup/rollup-linux-ppc64-musl": {
+      "version": "4.59.0",
+      "resolved": "https://registry.npmjs.org/@rollup/rollup-linux-ppc64-musl/-/rollup-linux-ppc64-musl-4.59.0.tgz",
+      "integrity": "sha512-+2kLtQ4xT3AiIxkzFVFXfsmlZiG5FXYW7ZyIIvGA7Bdeuh9Z0aN4hVyXS/G1E9bTP/vqszNIN/pUKCk/BTHsKA==",
+      "cpu": [
+        "ppc64"
+      ],
+      "dev": true,
+      "license": "MIT",
+      "optional": true,
+      "os": [
+        "linux"
+      ]
+    },
+    "node_modules/@rollup/rollup-linux-riscv64-gnu": {
+      "version": "4.59.0",
+      "resolved": "https://registry.npmjs.org/@rollup/rollup-linux-riscv64-gnu/-/rollup-linux-riscv64-gnu-4.59.0.tgz",
+      "integrity": "sha512-NDYMpsXYJJaj+I7UdwIuHHNxXZ/b/N2hR15NyH3m2qAtb/hHPA4g4SuuvrdxetTdndfj9b1WOmy73kcPRoERUg==",
+      "cpu": [
+        "riscv64"
+      ],
+      "dev": true,
+      "license": "MIT",
+      "optional": true,
+      "os": [
+        "linux"
+      ]
+    },
+    "node_modules/@rollup/rollup-linux-riscv64-musl": {
+      "version": "4.59.0",
+      "resolved": "https://registry.npmjs.org/@rollup/rollup-linux-riscv64-musl/-/rollup-linux-riscv64-musl-4.59.0.tgz",
+      "integrity": "sha512-nLckB8WOqHIf1bhymk+oHxvM9D3tyPndZH8i8+35p/1YiVoVswPid2yLzgX7ZJP0KQvnkhM4H6QZ5m0LzbyIAg==",
+      "cpu": [
+        "riscv64"
+      ],
+      "dev": true,
+      "license": "MIT",
+      "optional": true,
+      "os": [
+        "linux"
+      ]
+    },
+    "node_modules/@rollup/rollup-linux-s390x-gnu": {
+      "version": "4.59.0",
+      "resolved": "https://registry.npmjs.org/@rollup/rollup-linux-s390x-gnu/-/rollup-linux-s390x-gnu-4.59.0.tgz",
+      "integrity": "sha512-oF87Ie3uAIvORFBpwnCvUzdeYUqi2wY6jRFWJAy1qus/udHFYIkplYRW+wo+GRUP4sKzYdmE1Y3+rY5Gc4ZO+w==",
+      "cpu": [
+        "s390x"
+      ],
+      "dev": true,
+      "license": "MIT",
+      "optional": true,
+      "os": [
+        "linux"
+      ]
+    },
+    "node_modules/@rollup/rollup-linux-x64-gnu": {
+      "version": "4.59.0",
+      "resolved": "https://registry.npmjs.org/@rollup/rollup-linux-x64-gnu/-/rollup-linux-x64-gnu-4.59.0.tgz",
+      "integrity": "sha512-3AHmtQq/ppNuUspKAlvA8HtLybkDflkMuLK4DPo77DfthRb71V84/c4MlWJXixZz4uruIH4uaa07IqoAkG64fg==",
+      "cpu": [
+        "x64"
+      ],
+      "dev": true,
+      "license": "MIT",
+      "optional": true,
+      "os": [
+        "linux"
+      ]
+    },
+    "node_modules/@rollup/rollup-linux-x64-musl": {
+      "version": "4.59.0",
+      "resolved": "https://registry.npmjs.org/@rollup/rollup-linux-x64-musl/-/rollup-linux-x64-musl-4.59.0.tgz",
+      "integrity": "sha512-2UdiwS/9cTAx7qIUZB/fWtToJwvt0Vbo0zmnYt7ED35KPg13Q0ym1g442THLC7VyI6JfYTP4PiSOWyoMdV2/xg==",
+      "cpu": [
+        "x64"
+      ],
+      "dev": true,
+      "license": "MIT",
+      "optional": true,
+      "os": [
+        "linux"
+      ]
+    },
+    "node_modules/@rollup/rollup-openbsd-x64": {
+      "version": "4.59.0",
+      "resolved": "https://registry.npmjs.org/@rollup/rollup-openbsd-x64/-/rollup-openbsd-x64-4.59.0.tgz",
+      "integrity": "sha512-M3bLRAVk6GOwFlPTIxVBSYKUaqfLrn8l0psKinkCFxl4lQvOSz8ZrKDz2gxcBwHFpci0B6rttydI4IpS4IS/jQ==",
+      "cpu": [
+        "x64"
+      ],
+      "dev": true,
+      "license": "MIT",
+      "optional": true,
+      "os": [
+        "openbsd"
+      ]
+    },
+    "node_modules/@rollup/rollup-openharmony-arm64": {
+      "version": "4.59.0",
+      "resolved": "https://registry.npmjs.org/@rollup/rollup-openharmony-arm64/-/rollup-openharmony-arm64-4.59.0.tgz",
+      "integrity": "sha512-tt9KBJqaqp5i5HUZzoafHZX8b5Q2Fe7UjYERADll83O4fGqJ49O1FsL6LpdzVFQcpwvnyd0i+K/VSwu/o/nWlA==",
+      "cpu": [
+        "arm64"
+      ],
+      "dev": true,
+      "license": "MIT",
+      "optional": true,
+      "os": [
+        "openharmony"
+      ]
+    },
+    "node_modules/@rollup/rollup-win32-arm64-msvc": {
+      "version": "4.59.0",
+      "resolved": "https://registry.npmjs.org/@rollup/rollup-win32-arm64-msvc/-/rollup-win32-arm64-msvc-4.59.0.tgz",
+      "integrity": "sha512-V5B6mG7OrGTwnxaNUzZTDTjDS7F75PO1ae6MJYdiMu60sq0CqN5CVeVsbhPxalupvTX8gXVSU9gq+Rx1/hvu6A==",
+      "cpu": [
+        "arm64"
+      ],
+      "dev": true,
+      "license": "MIT",
+      "optional": true,
+      "os": [
+        "win32"
+      ]
+    },
+    "node_modules/@rollup/rollup-win32-ia32-msvc": {
+      "version": "4.59.0",
+      "resolved": "https://registry.npmjs.org/@rollup/rollup-win32-ia32-msvc/-/rollup-win32-ia32-msvc-4.59.0.tgz",
+      "integrity": "sha512-UKFMHPuM9R0iBegwzKF4y0C4J9u8C6MEJgFuXTBerMk7EJ92GFVFYBfOZaSGLu6COf7FxpQNqhNS4c4icUPqxA==",
+      "cpu": [
+        "ia32"
+      ],
+      "dev": true,
+      "license": "MIT",
+      "optional": true,
+      "os": [
+        "win32"
+      ]
+    },
+    "node_modules/@rollup/rollup-win32-x64-gnu": {
+      "version": "4.59.0",
+      "resolved": "https://registry.npmjs.org/@rollup/rollup-win32-x64-gnu/-/rollup-win32-x64-gnu-4.59.0.tgz",
+      "integrity": "sha512-laBkYlSS1n2L8fSo1thDNGrCTQMmxjYY5G0WFWjFFYZkKPjsMBsgJfGf4TLxXrF6RyhI60L8TMOjBMvXiTcxeA==",
+      "cpu": [
+        "x64"
+      ],
+      "dev": true,
+      "license": "MIT",
+      "optional": true,
+      "os": [
+        "win32"
+      ]
+    },
+    "node_modules/@rollup/rollup-win32-x64-msvc": {
+      "version": "4.59.0",
+      "resolved": "https://registry.npmjs.org/@rollup/rollup-win32-x64-msvc/-/rollup-win32-x64-msvc-4.59.0.tgz",
+      "integrity": "sha512-2HRCml6OztYXyJXAvdDXPKcawukWY2GpR5/nxKp4iBgiO3wcoEGkAaqctIbZcNB6KlUQBIqt8VYkNSj2397EfA==",
+      "cpu": [
+        "x64"
+      ],
+      "dev": true,
+      "license": "MIT",
+      "optional": true,
+      "os": [
+        "win32"
+      ]
+    },
+    "node_modules/@standard-schema/spec": {
+      "version": "1.1.0",
+      "resolved": "https://registry.npmjs.org/@standard-schema/spec/-/spec-1.1.0.tgz",
+      "integrity": "sha512-l2aFy5jALhniG5HgqrD6jXLi/rUWrKvqN/qJx6yoJsgKhblVd+iqqU4RCXavm/jPityDo5TCvKMnpjKnOriy0w==",
+      "dev": true,
+      "license": "MIT"
+    },
+    "node_modules/@tybys/wasm-util": {
+      "version": "0.10.1",
+      "resolved": "https://registry.npmjs.org/@tybys/wasm-util/-/wasm-util-0.10.1.tgz",
+      "integrity": "sha512-9tTaPJLSiejZKx+Bmog4uSubteqTvFrVrURwkmHixBo0G4seD0zUxp98E1DzUBJxLQ3NPwXrGKDiVjwx/DpPsg==",
+      "dev": true,
+      "license": "MIT",
+      "optional": true,
+      "dependencies": {
+        "tslib": "^2.4.0"
+      }
+    },
+    "node_modules/@types/chai": {
+      "version": "5.2.3",
+      "resolved": "https://registry.npmjs.org/@types/chai/-/chai-5.2.3.tgz",
+      "integrity": "sha512-Mw558oeA9fFbv65/y4mHtXDs9bPnFMZAL/jxdPFUpOHHIXX91mcgEHbS5Lahr+pwZFR8A7GQleRWeI6cGFC2UA==",
+      "dev": true,
+      "license": "MIT",
+      "dependencies": {
+        "@types/deep-eql": "*",
+        "assertion-error": "^2.0.1"
+      }
+    },
+    "node_modules/@types/debug": {
+      "version": "4.1.12",
+      "resolved": "https://registry.npmjs.org/@types/debug/-/debug-4.1.12.tgz",
+      "integrity": "sha512-vIChWdVG3LG1SMxEvI/AK+FWJthlrqlTu7fbrlywTkkaONwk/UAGaULXRlf8vkzFBLVm0zkMdCquhL5aOjhXPQ==",
+      "dev": true,
+      "license": "MIT",
+      "dependencies": {
+        "@types/ms": "*"
+      }
+    },
+    "node_modules/@types/deep-eql": {
+      "version": "4.0.2",
+      "resolved": "https://registry.npmjs.org/@types/deep-eql/-/deep-eql-4.0.2.tgz",
+      "integrity": "sha512-c9h9dVVMigMPc4bwTvC5dxqtqJZwQPePsWjPlpSOnojbor6pGqdk541lfA7AqFQr5pB1BRdq0juY9db81BwyFw==",
+      "dev": true,
+      "license": "MIT"
+    },
+    "node_modules/@types/estree": {
+      "version": "1.0.8",
+      "resolved": "https://registry.npmjs.org/@types/estree/-/estree-1.0.8.tgz",
+      "integrity": "sha512-dWHzHa2WqEXI/O1E9OjrocMTKJl2mSrEolh1Iomrv6U+JuNwaHXsXx9bLu5gG7BUWFIN0skIQJQ/L1rIex4X6w==",
+      "dev": true,
+      "license": "MIT"
+    },
+    "node_modules/@types/jsesc": {
+      "version": "2.5.1",
+      "resolved": "https://registry.npmjs.org/@types/jsesc/-/jsesc-2.5.1.tgz",
+      "integrity": "sha512-9VN+6yxLOPLOav+7PwjZbxiID2bVaeq0ED4qSQmdQTdjnXJSaCVKTR58t15oqH1H5t8Ng2ZX1SabJVoN9Q34bw==",
+      "dev": true,
+      "license": "MIT"
+    },
+    "node_modules/@types/json-schema": {
+      "version": "7.0.15",
+      "resolved": "https://registry.npmjs.org/@types/json-schema/-/json-schema-7.0.15.tgz",
+      "integrity": "sha512-5+fP8P8MFNC+AyZCDxrB2pkZFPGzqQWUzpSeuuVLvm8VMcorNYavBqoFcxK8bQz4Qsbn4oUEEem4wDLfcysGHA==",
+      "dev": true,
+      "license": "MIT"
+    },
+    "node_modules/@types/mdast": {
+      "version": "4.0.4",
+      "resolved": "https://registry.npmjs.org/@types/mdast/-/mdast-4.0.4.tgz",
+      "integrity": "sha512-kGaNbPh1k7AFzgpud/gMdvIm5xuECykRR+JnWKQno9TAXVa6WIVCGTPvYGekIDL4uwCZQSYbUxNBSb1aUo79oA==",
+      "dev": true,
+      "license": "MIT",
+      "dependencies": {
+        "@types/unist": "*"
+      }
+    },
+    "node_modules/@types/ms": {
+      "version": "2.1.0",
+      "resolved": "https://registry.npmjs.org/@types/ms/-/ms-2.1.0.tgz",
+      "integrity": "sha512-GsCCIZDE/p3i96vtEqx+7dBUGXrc7zeSK3wwPHIaRThS+9OhWIXRqzs4d6k1SVU8g91DrNRWxWUGhp5KXQb2VA==",
+      "dev": true,
+      "license": "MIT"
+    },
+    "node_modules/@types/node": {
+      "version": "25.4.0",
+      "resolved": "https://registry.npmjs.org/@types/node/-/node-25.4.0.tgz",
+      "integrity": "sha512-9wLpoeWuBlcbBpOY3XmzSTG3oscB6xjBEEtn+pYXTfhyXhIxC5FsBer2KTopBlvKEiW9l13po9fq+SJY/5lkhw==",
+      "dev": true,
+      "license": "MIT",
+      "dependencies": {
+        "undici-types": "~7.18.0"
+      }
+    },
+    "node_modules/@types/unist": {
+      "version": "3.0.3",
+      "resolved": "https://registry.npmjs.org/@types/unist/-/unist-3.0.3.tgz",
+      "integrity": "sha512-ko/gIFJRv177XgZsZcBwnqJN5x/Gien8qNOn0D5bQU/zAzVf9Zt3BlcUiLqhV9y4ARk0GbT3tnUiPNgnTXzc/Q==",
+      "dev": true,
+      "license": "MIT"
+    },
+    "node_modules/@typescript-eslint/eslint-plugin": {
+      "version": "8.57.0",
+      "resolved": "https://registry.npmjs.org/@typescript-eslint/eslint-plugin/-/eslint-plugin-8.57.0.tgz",
+      "integrity": "sha512-qeu4rTHR3/IaFORbD16gmjq9+rEs9fGKdX0kF6BKSfi+gCuG3RCKLlSBYzn/bGsY9Tj7KE/DAQStbp8AHJGHEQ==",
+      "dev": true,
+      "license": "MIT",
+      "dependencies": {
+        "@eslint-community/regexpp": "^4.12.2",
+        "@typescript-eslint/scope-manager": "8.57.0",
+        "@typescript-eslint/type-utils": "8.57.0",
+        "@typescript-eslint/utils": "8.57.0",
+        "@typescript-eslint/visitor-keys": "8.57.0",
+        "ignore": "^7.0.5",
+        "natural-compare": "^1.4.0",
+        "ts-api-utils": "^2.4.0"
+      },
+      "engines": {
+        "node": "^18.18.0 || ^20.9.0 || >=21.1.0"
+      },
+      "funding": {
+        "type": "opencollective",
+        "url": "https://opencollective.com/typescript-eslint"
+      },
+      "peerDependencies": {
+        "@typescript-eslint/parser": "^8.57.0",
+        "eslint": "^8.57.0 || ^9.0.0 || ^10.0.0",
+        "typescript": ">=4.8.4 <6.0.0"
+      }
+    },
+    "node_modules/@typescript-eslint/eslint-plugin/node_modules/ignore": {
+      "version": "7.0.5",
+      "resolved": "https://registry.npmjs.org/ignore/-/ignore-7.0.5.tgz",
+      "integrity": "sha512-Hs59xBNfUIunMFgWAbGX5cq6893IbWg4KnrjbYwX3tx0ztorVgTDA6B2sxf8ejHJ4wz8BqGUMYlnzNBer5NvGg==",
+      "dev": true,
+      "license": "MIT",
+      "engines": {
+        "node": ">= 4"
+      }
+    },
+    "node_modules/@typescript-eslint/parser": {
+      "version": "8.57.0",
+      "resolved": "https://registry.npmjs.org/@typescript-eslint/parser/-/parser-8.57.0.tgz",
+      "integrity": "sha512-XZzOmihLIr8AD1b9hL9ccNMzEMWt/dE2u7NyTY9jJG6YNiNthaD5XtUHVF2uCXZ15ng+z2hT3MVuxnUYhq6k1g==",
+      "dev": true,
+      "license": "MIT",
+      "dependencies": {
+        "@typescript-eslint/scope-manager": "8.57.0",
+        "@typescript-eslint/types": "8.57.0",
+        "@typescript-eslint/typescript-estree": "8.57.0",
+        "@typescript-eslint/visitor-keys": "8.57.0",
+        "debug": "^4.4.3"
+      },
+      "engines": {
+        "node": "^18.18.0 || ^20.9.0 || >=21.1.0"
+      },
+      "funding": {
+        "type": "opencollective",
+        "url": "https://opencollective.com/typescript-eslint"
+      },
+      "peerDependencies": {
+        "eslint": "^8.57.0 || ^9.0.0 || ^10.0.0",
+        "typescript": ">=4.8.4 <6.0.0"
+      }
+    },
+    "node_modules/@typescript-eslint/project-service": {
+      "version": "8.57.0",
+      "resolved": "https://registry.npmjs.org/@typescript-eslint/project-service/-/project-service-8.57.0.tgz",
+      "integrity": "sha512-pR+dK0BlxCLxtWfaKQWtYr7MhKmzqZxuii+ZjuFlZlIGRZm22HnXFqa2eY+90MUz8/i80YJmzFGDUsi8dMOV5w==",
+      "dev": true,
+      "license": "MIT",
+      "dependencies": {
+        "@typescript-eslint/tsconfig-utils": "^8.57.0",
+        "@typescript-eslint/types": "^8.57.0",
+        "debug": "^4.4.3"
+      },
+      "engines": {
+        "node": "^18.18.0 || ^20.9.0 || >=21.1.0"
+      },
+      "funding": {
+        "type": "opencollective",
+        "url": "https://opencollective.com/typescript-eslint"
+      },
+      "peerDependencies": {
+        "typescript": ">=4.8.4 <6.0.0"
+      }
+    },
+    "node_modules/@typescript-eslint/scope-manager": {
+      "version": "8.57.0",
+      "resolved": "https://registry.npmjs.org/@typescript-eslint/scope-manager/-/scope-manager-8.57.0.tgz",
+      "integrity": "sha512-nvExQqAHF01lUM66MskSaZulpPL5pgy5hI5RfrxviLgzZVffB5yYzw27uK/ft8QnKXI2X0LBrHJFr1TaZtAibw==",
+      "dev": true,
+      "license": "MIT",
+      "dependencies": {
+        "@typescript-eslint/types": "8.57.0",
+        "@typescript-eslint/visitor-keys": "8.57.0"
+      },
+      "engines": {
+        "node": "^18.18.0 || ^20.9.0 || >=21.1.0"
+      },
+      "funding": {
+        "type": "opencollective",
+        "url": "https://opencollective.com/typescript-eslint"
+      }
+    },
+    "node_modules/@typescript-eslint/tsconfig-utils": {
+      "version": "8.57.0",
+      "resolved": "https://registry.npmjs.org/@typescript-eslint/tsconfig-utils/-/tsconfig-utils-8.57.0.tgz",
+      "integrity": "sha512-LtXRihc5ytjJIQEH+xqjB0+YgsV4/tW35XKX3GTZHpWtcC8SPkT/d4tqdf1cKtesryHm2bgp6l555NYcT2NLvA==",
+      "dev": true,
+      "license": "MIT",
+      "engines": {
+        "node": "^18.18.0 || ^20.9.0 || >=21.1.0"
+      },
+      "funding": {
+        "type": "opencollective",
+        "url": "https://opencollective.com/typescript-eslint"
+      },
+      "peerDependencies": {
+        "typescript": ">=4.8.4 <6.0.0"
+      }
+    },
+    "node_modules/@typescript-eslint/type-utils": {
+      "version": "8.57.0",
+      "resolved": "https://registry.npmjs.org/@typescript-eslint/type-utils/-/type-utils-8.57.0.tgz",
+      "integrity": "sha512-yjgh7gmDcJ1+TcEg8x3uWQmn8ifvSupnPfjP21twPKrDP/pTHlEQgmKcitzF/rzPSmv7QjJ90vRpN4U+zoUjwQ==",
+      "dev": true,
+      "license": "MIT",
+      "dependencies": {
+        "@typescript-eslint/types": "8.57.0",
+        "@typescript-eslint/typescript-estree": "8.57.0",
+        "@typescript-eslint/utils": "8.57.0",
+        "debug": "^4.4.3",
+        "ts-api-utils": "^2.4.0"
+      },
+      "engines": {
+        "node": "^18.18.0 || ^20.9.0 || >=21.1.0"
+      },
+      "funding": {
+        "type": "opencollective",
+        "url": "https://opencollective.com/typescript-eslint"
+      },
+      "peerDependencies": {
+        "eslint": "^8.57.0 || ^9.0.0 || ^10.0.0",
+        "typescript": ">=4.8.4 <6.0.0"
+      }
+    },
+    "node_modules/@typescript-eslint/types": {
+      "version": "8.57.0",
+      "resolved": "https://registry.npmjs.org/@typescript-eslint/types/-/types-8.57.0.tgz",
+      "integrity": "sha512-dTLI8PEXhjUC7B9Kre+u0XznO696BhXcTlOn0/6kf1fHaQW8+VjJAVHJ3eTI14ZapTxdkOmc80HblPQLaEeJdg==",
+      "dev": true,
+      "license": "MIT",
+      "engines": {
+        "node": "^18.18.0 || ^20.9.0 || >=21.1.0"
+      },
+      "funding": {
+        "type": "opencollective",
+        "url": "https://opencollective.com/typescript-eslint"
+      }
+    },
+    "node_modules/@typescript-eslint/typescript-estree": {
+      "version": "8.57.0",
+      "resolved": "https://registry.npmjs.org/@typescript-eslint/typescript-estree/-/typescript-estree-8.57.0.tgz",
+      "integrity": "sha512-m7faHcyVg0BT3VdYTlX8GdJEM7COexXxS6KqGopxdtkQRvBanK377QDHr4W/vIPAR+ah9+B/RclSW5ldVniO1Q==",
+      "dev": true,
+      "license": "MIT",
+      "dependencies": {
+        "@typescript-eslint/project-service": "8.57.0",
+        "@typescript-eslint/tsconfig-utils": "8.57.0",
+        "@typescript-eslint/types": "8.57.0",
+        "@typescript-eslint/visitor-keys": "8.57.0",
+        "debug": "^4.4.3",
+        "minimatch": "^10.2.2",
+        "semver": "^7.7.3",
+        "tinyglobby": "^0.2.15",
+        "ts-api-utils": "^2.4.0"
+      },
+      "engines": {
+        "node": "^18.18.0 || ^20.9.0 || >=21.1.0"
+      },
+      "funding": {
+        "type": "opencollective",
+        "url": "https://opencollective.com/typescript-eslint"
+      },
+      "peerDependencies": {
+        "typescript": ">=4.8.4 <6.0.0"
+      }
+    },
+    "node_modules/@typescript-eslint/typescript-estree/node_modules/balanced-match": {
+      "version": "4.0.4",
+      "resolved": "https://registry.npmjs.org/balanced-match/-/balanced-match-4.0.4.tgz",
+      "integrity": "sha512-BLrgEcRTwX2o6gGxGOCNyMvGSp35YofuYzw9h1IMTRmKqttAZZVU67bdb9Pr2vUHA8+j3i2tJfjO6C6+4myGTA==",
+      "dev": true,
+      "license": "MIT",
+      "engines": {
+        "node": "18 || 20 || >=22"
+      }
+    },
+    "node_modules/@typescript-eslint/typescript-estree/node_modules/brace-expansion": {
+      "version": "5.0.4",
+      "resolved": "https://registry.npmjs.org/brace-expansion/-/brace-expansion-5.0.4.tgz",
+      "integrity": "sha512-h+DEnpVvxmfVefa4jFbCf5HdH5YMDXRsmKflpf1pILZWRFlTbJpxeU55nJl4Smt5HQaGzg1o6RHFPJaOqnmBDg==",
+      "dev": true,
+      "license": "MIT",
+      "dependencies": {
+        "balanced-match": "^4.0.2"
+      },
+      "engines": {
+        "node": "18 || 20 || >=22"
+      }
+    },
+    "node_modules/@typescript-eslint/typescript-estree/node_modules/minimatch": {
+      "version": "10.2.4",
+      "resolved": "https://registry.npmjs.org/minimatch/-/minimatch-10.2.4.tgz",
+      "integrity": "sha512-oRjTw/97aTBN0RHbYCdtF1MQfvusSIBQM0IZEgzl6426+8jSC0nF1a/GmnVLpfB9yyr6g6FTqWqiZVbxrtaCIg==",
+      "dev": true,
+      "license": "BlueOak-1.0.0",
+      "dependencies": {
+        "brace-expansion": "^5.0.2"
+      },
+      "engines": {
+        "node": "18 || 20 || >=22"
+      },
+      "funding": {
+        "url": "https://github.com/sponsors/isaacs"
+      }
+    },
+    "node_modules/@typescript-eslint/utils": {
+      "version": "8.57.0",
+      "resolved": "https://registry.npmjs.org/@typescript-eslint/utils/-/utils-8.57.0.tgz",
+      "integrity": "sha512-5iIHvpD3CZe06riAsbNxxreP+MuYgVUsV0n4bwLH//VJmgtt54sQeY2GszntJ4BjYCpMzrfVh2SBnUQTtys2lQ==",
+      "dev": true,
+      "license": "MIT",
+      "dependencies": {
+        "@eslint-community/eslint-utils": "^4.9.1",
+        "@typescript-eslint/scope-manager": "8.57.0",
+        "@typescript-eslint/types": "8.57.0",
+        "@typescript-eslint/typescript-estree": "8.57.0"
+      },
+      "engines": {
+        "node": "^18.18.0 || ^20.9.0 || >=21.1.0"
+      },
+      "funding": {
+        "type": "opencollective",
+        "url": "https://opencollective.com/typescript-eslint"
+      },
+      "peerDependencies": {
+        "eslint": "^8.57.0 || ^9.0.0 || ^10.0.0",
+        "typescript": ">=4.8.4 <6.0.0"
+      }
+    },
+    "node_modules/@typescript-eslint/visitor-keys": {
+      "version": "8.57.0",
+      "resolved": "https://registry.npmjs.org/@typescript-eslint/visitor-keys/-/visitor-keys-8.57.0.tgz",
+      "integrity": "sha512-zm6xx8UT/Xy2oSr2ZXD0pZo7Jx2XsCoID2IUh9YSTFRu7z+WdwYTRk6LhUftm1crwqbuoF6I8zAFeCMw0YjwDg==",
+      "dev": true,
+      "license": "MIT",
+      "dependencies": {
+        "@typescript-eslint/types": "8.57.0",
+        "eslint-visitor-keys": "^5.0.0"
+      },
+      "engines": {
+        "node": "^18.18.0 || ^20.9.0 || >=21.1.0"
+      },
+      "funding": {
+        "type": "opencollective",
+        "url": "https://opencollective.com/typescript-eslint"
+      }
+    },
+    "node_modules/@typescript-eslint/visitor-keys/node_modules/eslint-visitor-keys": {
+      "version": "5.0.1",
+      "resolved": "https://registry.npmjs.org/eslint-visitor-keys/-/eslint-visitor-keys-5.0.1.tgz",
+      "integrity": "sha512-tD40eHxA35h0PEIZNeIjkHoDR4YjjJp34biM0mDvplBe//mB+IHCqHDGV7pxF+7MklTvighcCPPZC7ynWyjdTA==",
+      "dev": true,
+      "license": "Apache-2.0",
+      "engines": {
+        "node": "^20.19.0 || ^22.13.0 || >=24"
+      },
+      "funding": {
+        "url": "https://opencollective.com/eslint"
+      }
+    },
+    "node_modules/@vitest/coverage-v8": {
+      "version": "4.0.18",
+      "resolved": "https://registry.npmjs.org/@vitest/coverage-v8/-/coverage-v8-4.0.18.tgz",
+      "integrity": "sha512-7i+N2i0+ME+2JFZhfuz7Tg/FqKtilHjGyGvoHYQ6iLV0zahbsJ9sljC9OcFcPDbhYKCet+sG8SsVqlyGvPflZg==",
+      "dev": true,
+      "license": "MIT",
+      "dependencies": {
+        "@bcoe/v8-coverage": "^1.0.2",
+        "@vitest/utils": "4.0.18",
+        "ast-v8-to-istanbul": "^0.3.10",
+        "istanbul-lib-coverage": "^3.2.2",
+        "istanbul-lib-report": "^3.0.1",
+        "istanbul-reports": "^3.2.0",
+        "magicast": "^0.5.1",
+        "obug": "^2.1.1",
+        "std-env": "^3.10.0",
+        "tinyrainbow": "^3.0.3"
+      },
+      "funding": {
+        "url": "https://opencollective.com/vitest"
+      },
+      "peerDependencies": {
+        "@vitest/browser": "4.0.18",
+        "vitest": "4.0.18"
+      },
+      "peerDependenciesMeta": {
+        "@vitest/browser": {
+          "optional": true
+        }
+      }
+    },
+    "node_modules/@vitest/expect": {
+      "version": "4.0.18",
+      "resolved": "https://registry.npmjs.org/@vitest/expect/-/expect-4.0.18.tgz",
+      "integrity": "sha512-8sCWUyckXXYvx4opfzVY03EOiYVxyNrHS5QxX3DAIi5dpJAAkyJezHCP77VMX4HKA2LDT/Jpfo8i2r5BE3GnQQ==",
+      "dev": true,
+      "license": "MIT",
+      "dependencies": {
+        "@standard-schema/spec": "^1.0.0",
+        "@types/chai": "^5.2.2",
+        "@vitest/spy": "4.0.18",
+        "@vitest/utils": "4.0.18",
+        "chai": "^6.2.1",
+        "tinyrainbow": "^3.0.3"
+      },
+      "funding": {
+        "url": "https://opencollective.com/vitest"
+      }
+    },
+    "node_modules/@vitest/mocker": {
+      "version": "4.0.18",
+      "resolved": "https://registry.npmjs.org/@vitest/mocker/-/mocker-4.0.18.tgz",
+      "integrity": "sha512-HhVd0MDnzzsgevnOWCBj5Otnzobjy5wLBe4EdeeFGv8luMsGcYqDuFRMcttKWZA5vVO8RFjexVovXvAM4JoJDQ==",
+      "dev": true,
+      "license": "MIT",
+      "dependencies": {
+        "@vitest/spy": "4.0.18",
+        "estree-walker": "^3.0.3",
+        "magic-string": "^0.30.21"
+      },
+      "funding": {
+        "url": "https://opencollective.com/vitest"
+      },
+      "peerDependencies": {
+        "msw": "^2.4.9",
+        "vite": "^6.0.0 || ^7.0.0-0"
+      },
+      "peerDependenciesMeta": {
+        "msw": {
+          "optional": true
+        },
+        "vite": {
+          "optional": true
+        }
+      }
+    },
+    "node_modules/@vitest/pretty-format": {
+      "version": "4.0.18",
+      "resolved": "https://registry.npmjs.org/@vitest/pretty-format/-/pretty-format-4.0.18.tgz",
+      "integrity": "sha512-P24GK3GulZWC5tz87ux0m8OADrQIUVDPIjjj65vBXYG17ZeU3qD7r+MNZ1RNv4l8CGU2vtTRqixrOi9fYk/yKw==",
+      "dev": true,
+      "license": "MIT",
+      "dependencies": {
+        "tinyrainbow": "^3.0.3"
+      },
+      "funding": {
+        "url": "https://opencollective.com/vitest"
+      }
+    },
+    "node_modules/@vitest/runner": {
+      "version": "4.0.18",
+      "resolved": "https://registry.npmjs.org/@vitest/runner/-/runner-4.0.18.tgz",
+      "integrity": "sha512-rpk9y12PGa22Jg6g5M3UVVnTS7+zycIGk9ZNGN+m6tZHKQb7jrP7/77WfZy13Y/EUDd52NDsLRQhYKtv7XfPQw==",
+      "dev": true,
+      "license": "MIT",
+      "dependencies": {
+        "@vitest/utils": "4.0.18",
+        "pathe": "^2.0.3"
+      },
+      "funding": {
+        "url": "https://opencollective.com/vitest"
+      }
+    },
+    "node_modules/@vitest/snapshot": {
+      "version": "4.0.18",
+      "resolved": "https://registry.npmjs.org/@vitest/snapshot/-/snapshot-4.0.18.tgz",
+      "integrity": "sha512-PCiV0rcl7jKQjbgYqjtakly6T1uwv/5BQ9SwBLekVg/EaYeQFPiXcgrC2Y7vDMA8dM1SUEAEV82kgSQIlXNMvA==",
+      "dev": true,
+      "license": "MIT",
+      "dependencies": {
+        "@vitest/pretty-format": "4.0.18",
+        "magic-string": "^0.30.21",
+        "pathe": "^2.0.3"
+      },
+      "funding": {
+        "url": "https://opencollective.com/vitest"
+      }
+    },
+    "node_modules/@vitest/spy": {
+      "version": "4.0.18",
+      "resolved": "https://registry.npmjs.org/@vitest/spy/-/spy-4.0.18.tgz",
+      "integrity": "sha512-cbQt3PTSD7P2OARdVW3qWER5EGq7PHlvE+QfzSC0lbwO+xnt7+XH06ZzFjFRgzUX//JmpxrCu92VdwvEPlWSNw==",
+      "dev": true,
+      "license": "MIT",
+      "funding": {
+        "url": "https://opencollective.com/vitest"
+      }
+    },
+    "node_modules/@vitest/utils": {
+      "version": "4.0.18",
+      "resolved": "https://registry.npmjs.org/@vitest/utils/-/utils-4.0.18.tgz",
+      "integrity": "sha512-msMRKLMVLWygpK3u2Hybgi4MNjcYJvwTb0Ru09+fOyCXIgT5raYP041DRRdiJiI3k/2U6SEbAETB3YtBrUkCFA==",
+      "dev": true,
+      "license": "MIT",
+      "dependencies": {
+        "@vitest/pretty-format": "4.0.18",
+        "tinyrainbow": "^3.0.3"
+      },
+      "funding": {
+        "url": "https://opencollective.com/vitest"
+      }
+    },
+    "node_modules/acorn": {
+      "version": "8.16.0",
+      "resolved": "https://registry.npmjs.org/acorn/-/acorn-8.16.0.tgz",
+      "integrity": "sha512-UVJyE9MttOsBQIDKw1skb9nAwQuR5wuGD3+82K6JgJlm/Y+KI92oNsMNGZCYdDsVtRHSak0pcV5Dno5+4jh9sw==",
+      "dev": true,
+      "license": "MIT",
+      "bin": {
+        "acorn": "bin/acorn"
+      },
+      "engines": {
+        "node": ">=0.4.0"
+      }
+    },
+    "node_modules/acorn-jsx": {
+      "version": "5.3.2",
+      "resolved": "https://registry.npmjs.org/acorn-jsx/-/acorn-jsx-5.3.2.tgz",
+      "integrity": "sha512-rq9s+JNhf0IChjtDXxllJ7g41oZk5SlXtp0LHwyA5cejwn7vKmKp4pPri6YEePv2PU65sAsegbXtIinmDFDXgQ==",
+      "dev": true,
+      "license": "MIT",
+      "peerDependencies": {
+        "acorn": "^6.0.0 || ^7.0.0 || ^8.0.0"
+      }
+    },
+    "node_modules/ajv": {
+      "version": "6.14.0",
+      "resolved": "https://registry.npmjs.org/ajv/-/ajv-6.14.0.tgz",
+      "integrity": "sha512-IWrosm/yrn43eiKqkfkHis7QioDleaXQHdDVPKg0FSwwd/DuvyX79TZnFOnYpB7dcsFAMmtFztZuXPDvSePkFw==",
+      "dev": true,
+      "license": "MIT",
+      "dependencies": {
+        "fast-deep-equal": "^3.1.1",
+        "fast-json-stable-stringify": "^2.0.0",
+        "json-schema-traverse": "^0.4.1",
+        "uri-js": "^4.2.2"
+      },
+      "funding": {
+        "type": "github",
+        "url": "https://github.com/sponsors/epoberezkin"
+      }
+    },
+    "node_modules/ansi-styles": {
+      "version": "4.3.0",
+      "resolved": "https://registry.npmjs.org/ansi-styles/-/ansi-styles-4.3.0.tgz",
+      "integrity": "sha512-zbB9rCJAT1rbjiVDb2hqKFHNYLxgtk8NURxZ3IZwD3F6NtxbXZQCnnSi1Lkx+IDohdPlFp222wVALIheZJQSEg==",
+      "dev": true,
+      "license": "MIT",
+      "dependencies": {
+        "color-convert": "^2.0.1"
+      },
+      "engines": {
+        "node": ">=8"
+      },
+      "funding": {
+        "url": "https://github.com/chalk/ansi-styles?sponsor=1"
+      }
+    },
+    "node_modules/argparse": {
+      "version": "2.0.1",
+      "resolved": "https://registry.npmjs.org/argparse/-/argparse-2.0.1.tgz",
+      "integrity": "sha512-8+9WqebbFzpX9OR+Wa6O29asIogeRMzcGtAINdpMHHyAg10f05aSFVBbcEqGf/PXw1EjAZ+q2/bEBg3DvurK3Q==",
+      "dev": true,
+      "license": "Python-2.0"
+    },
+    "node_modules/array-find-index": {
+      "version": "1.0.2",
+      "resolved": "https://registry.npmjs.org/array-find-index/-/array-find-index-1.0.2.tgz",
+      "integrity": "sha512-M1HQyIXcBGtVywBt8WVdim+lrNaK7VHp99Qt5pSNziXznKHViIBbXWtfRTpEFpF/c4FdfxNAsCCwPp5phBYJtw==",
+      "dev": true,
+      "license": "MIT",
+      "engines": {
+        "node": ">=0.10.0"
+      }
+    },
+    "node_modules/assertion-error": {
+      "version": "2.0.1",
+      "resolved": "https://registry.npmjs.org/assertion-error/-/assertion-error-2.0.1.tgz",
+      "integrity": "sha512-Izi8RQcffqCeNVgFigKli1ssklIbpHnCYc6AknXGYoB6grJqyeby7jv12JUQgmTAnIDnbck1uxksT4dzN3PWBA==",
+      "dev": true,
+      "license": "MIT",
+      "engines": {
+        "node": ">=12"
+      }
+    },
+    "node_modules/ast-kit": {
+      "version": "3.0.0-beta.1",
+      "resolved": "https://registry.npmjs.org/ast-kit/-/ast-kit-3.0.0-beta.1.tgz",
+      "integrity": "sha512-trmleAnZ2PxN/loHWVhhx1qeOHSRXq4TDsBBxq3GqeJitfk3+jTQ+v/C1km/KYq9M7wKqCewMh+/NAvVH7m+bw==",
+      "dev": true,
+      "license": "MIT",
+      "dependencies": {
+        "@babel/parser": "^8.0.0-beta.4",
+        "estree-walker": "^3.0.3",
+        "pathe": "^2.0.3"
+      },
+      "engines": {
+        "node": ">=20.19.0"
+      },
+      "funding": {
+        "url": "https://github.com/sponsors/sxzz"
+      }
+    },
+    "node_modules/ast-kit/node_modules/@babel/helper-string-parser": {
+      "version": "8.0.0-rc.2",
+      "resolved": "https://registry.npmjs.org/@babel/helper-string-parser/-/helper-string-parser-8.0.0-rc.2.tgz",
+      "integrity": "sha512-noLx87RwlBEMrTzncWd/FvTxoJ9+ycHNg0n8yyYydIoDsLZuxknKgWRJUqcrVkNrJ74uGyhWQzQaS3q8xfGAhQ==",
+      "dev": true,
+      "license": "MIT",
+      "engines": {
+        "node": "^20.19.0 || >=22.12.0"
+      }
+    },
+    "node_modules/ast-kit/node_modules/@babel/helper-validator-identifier": {
+      "version": "8.0.0-rc.2",
+      "resolved": "https://registry.npmjs.org/@babel/helper-validator-identifier/-/helper-validator-identifier-8.0.0-rc.2.tgz",
+      "integrity": "sha512-xExUBkuXWJjVuIbO7z6q7/BA9bgfJDEhVL0ggrggLMbg0IzCUWGT1hZGE8qUH7Il7/RD/a6cZ3AAFrrlp1LF/A==",
+      "dev": true,
+      "license": "MIT",
+      "engines": {
+        "node": "^20.19.0 || >=22.12.0"
+      }
+    },
+    "node_modules/ast-kit/node_modules/@babel/parser": {
+      "version": "8.0.0-rc.2",
+      "resolved": "https://registry.npmjs.org/@babel/parser/-/parser-8.0.0-rc.2.tgz",
+      "integrity": "sha512-29AhEtcq4x8Dp3T72qvUMZHx0OMXCj4Jy/TEReQa+KWLln524Cj1fWb3QFi0l/xSpptQBR6y9RNEXuxpFvwiUQ==",
+      "dev": true,
+      "license": "MIT",
+      "dependencies": {
+        "@babel/types": "^8.0.0-rc.2"
+      },
+      "bin": {
+        "parser": "bin/babel-parser.js"
+      },
+      "engines": {
+        "node": "^20.19.0 || >=22.12.0"
+      }
+    },
+    "node_modules/ast-kit/node_modules/@babel/types": {
+      "version": "8.0.0-rc.2",
+      "resolved": "https://registry.npmjs.org/@babel/types/-/types-8.0.0-rc.2.tgz",
+      "integrity": "sha512-91gAaWRznDwSX4E2tZ1YjBuIfnQVOFDCQ2r0Toby0gu4XEbyF623kXLMA8d4ZbCu+fINcrudkmEcwSUHgDDkNw==",
+      "dev": true,
+      "license": "MIT",
+      "dependencies": {
+        "@babel/helper-string-parser": "^8.0.0-rc.2",
+        "@babel/helper-validator-identifier": "^8.0.0-rc.2"
+      },
+      "engines": {
+        "node": "^20.19.0 || >=22.12.0"
+      }
+    },
+    "node_modules/ast-v8-to-istanbul": {
+      "version": "0.3.12",
+      "resolved": "https://registry.npmjs.org/ast-v8-to-istanbul/-/ast-v8-to-istanbul-0.3.12.tgz",
+      "integrity": "sha512-BRRC8VRZY2R4Z4lFIL35MwNXmwVqBityvOIwETtsCSwvjl0IdgFsy9NhdaA6j74nUdtJJlIypeRhpDam19Wq3g==",
+      "dev": true,
+      "license": "MIT",
+      "dependencies": {
+        "@jridgewell/trace-mapping": "^0.3.31",
+        "estree-walker": "^3.0.3",
+        "js-tokens": "^10.0.0"
+      }
+    },
+    "node_modules/automd": {
+      "version": "0.4.3",
+      "resolved": "https://registry.npmjs.org/automd/-/automd-0.4.3.tgz",
+      "integrity": "sha512-5WJNEiaNpFm8h0OmQzhnESthadUQhJwQfka/TmmJpMudZ8qU9MZao9p0G1g7WYA9pVTz6FMMOSvxnfQ9g8q9vQ==",
+      "dev": true,
+      "license": "MIT",
+      "dependencies": {
+        "@parcel/watcher": "^2.5.6",
+        "c12": "^3.3.3",
+        "citty": "^0.2.0",
+        "consola": "^3.4.2",
+        "defu": "^6.1.4",
+        "destr": "^2.0.5",
+        "didyoumean2": "^7.0.4",
+        "magic-string": "^0.30.21",
+        "mdbox": "^0.1.1",
+        "mlly": "^1.8.0",
+        "ofetch": "^1.5.1",
+        "pathe": "^2.0.3",
+        "perfect-debounce": "^2.1.0",
+        "pkg-types": "^2.3.0",
+        "scule": "^1.3.0",
+        "tinyglobby": "^0.2.15",
+        "untyped": "^2.0.0"
+      },
+      "bin": {
+        "automd": "dist/cli.mjs"
+      }
+    },
+    "node_modules/balanced-match": {
+      "version": "1.0.2",
+      "resolved": "https://registry.npmjs.org/balanced-match/-/balanced-match-1.0.2.tgz",
+      "integrity": "sha512-3oSeUO0TMV67hN1AmbXsK4yaqU7tjiHlbxRDZOpH0KW9+CeX4bRAaX0Anxt0tx2MrpRpWwQaPwIlISEJhYU5Pw==",
+      "dev": true,
+      "license": "MIT"
+    },
+    "node_modules/baseline-browser-mapping": {
+      "version": "2.10.0",
+      "resolved": "https://registry.npmjs.org/baseline-browser-mapping/-/baseline-browser-mapping-2.10.0.tgz",
+      "integrity": "sha512-lIyg0szRfYbiy67j9KN8IyeD7q7hcmqnJ1ddWmNt19ItGpNN64mnllmxUNFIOdOm6by97jlL6wfpTTJrmnjWAA==",
+      "dev": true,
+      "license": "Apache-2.0",
+      "bin": {
+        "baseline-browser-mapping": "dist/cli.cjs"
+      },
+      "engines": {
+        "node": ">=6.0.0"
+      }
+    },
+    "node_modules/birpc": {
+      "version": "4.0.0",
+      "resolved": "https://registry.npmjs.org/birpc/-/birpc-4.0.0.tgz",
+      "integrity": "sha512-LShSxJP0KTmd101b6DRyGBj57LZxSDYWKitQNW/mi8GRMvZb078Uf9+pveax1DrVL89vm7mWe+TovdI/UDOuPw==",
+      "dev": true,
+      "license": "MIT",
+      "funding": {
+        "url": "https://github.com/sponsors/antfu"
+      }
+    },
+    "node_modules/brace-expansion": {
+      "version": "1.1.12",
+      "resolved": "https://registry.npmjs.org/brace-expansion/-/brace-expansion-1.1.12.tgz",
+      "integrity": "sha512-9T9UjW3r0UW5c1Q7GTwllptXwhvYmEzFhzMfZ9H7FQWt+uZePjZPjBP/W1ZEyZ1twGWom5/56TF4lPcqjnDHcg==",
+      "dev": true,
+      "license": "MIT",
+      "dependencies": {
+        "balanced-match": "^1.0.0",
+        "concat-map": "0.0.1"
+      }
+    },
+    "node_modules/browserslist": {
+      "version": "4.28.1",
+      "resolved": "https://registry.npmjs.org/browserslist/-/browserslist-4.28.1.tgz",
+      "integrity": "sha512-ZC5Bd0LgJXgwGqUknZY/vkUQ04r8NXnJZ3yYi4vDmSiZmC/pdSN0NbNRPxZpbtO4uAfDUAFffO8IZoM3Gj8IkA==",
+      "dev": true,
+      "funding": [
+        {
+          "type": "opencollective",
+          "url": "https://opencollective.com/browserslist"
+        },
+        {
+          "type": "tidelift",
+          "url": "https://tidelift.com/funding/github/npm/browserslist"
+        },
+        {
+          "type": "github",
+          "url": "https://github.com/sponsors/ai"
+        }
+      ],
+      "license": "MIT",
+      "dependencies": {
+        "baseline-browser-mapping": "^2.9.0",
+        "caniuse-lite": "^1.0.30001759",
+        "electron-to-chromium": "^1.5.263",
+        "node-releases": "^2.0.27",
+        "update-browserslist-db": "^1.2.0"
+      },
+      "bin": {
+        "browserslist": "cli.js"
+      },
+      "engines": {
+        "node": "^6 || ^7 || ^8 || ^9 || ^10 || ^11 || ^12 || >=13.7"
+      }
+    },
+    "node_modules/builtin-modules": {
+      "version": "5.0.0",
+      "resolved": "https://registry.npmjs.org/builtin-modules/-/builtin-modules-5.0.0.tgz",
+      "integrity": "sha512-bkXY9WsVpY7CvMhKSR6pZilZu9Ln5WDrKVBUXf2S443etkmEO4V58heTecXcUIsNsi4Rx8JUO4NfX1IcQl4deg==",
+      "dev": true,
+      "license": "MIT",
+      "engines": {
+        "node": ">=18.20"
+      },
+      "funding": {
+        "url": "https://github.com/sponsors/sindresorhus"
+      }
+    },
+    "node_modules/bundle-name": {
+      "version": "4.1.0",
+      "resolved": "https://registry.npmjs.org/bundle-name/-/bundle-name-4.1.0.tgz",
+      "integrity": "sha512-tjwM5exMg6BGRI+kNmTntNsvdZS1X8BFYS6tnJ2hdH0kVxM6/eVZ2xy+FqStSWvYmtfFMDLIxurorHwDKfDz5Q==",
+      "dev": true,
+      "license": "MIT",
+      "dependencies": {
+        "run-applescript": "^7.0.0"
+      },
+      "engines": {
+        "node": ">=18"
+      },
+      "funding": {
+        "url": "https://github.com/sponsors/sindresorhus"
+      }
+    },
+    "node_modules/c12": {
+      "version": "3.3.3",
+      "resolved": "https://registry.npmjs.org/c12/-/c12-3.3.3.tgz",
+      "integrity": "sha512-750hTRvgBy5kcMNPdh95Qo+XUBeGo8C7nsKSmedDmaQI+E0r82DwHeM6vBewDe4rGFbnxoa4V9pw+sPh5+Iz8Q==",
+      "dev": true,
+      "license": "MIT",
+      "dependencies": {
+        "chokidar": "^5.0.0",
+        "confbox": "^0.2.2",
+        "defu": "^6.1.4",
+        "dotenv": "^17.2.3",
+        "exsolve": "^1.0.8",
+        "giget": "^2.0.0",
+        "jiti": "^2.6.1",
+        "ohash": "^2.0.11",
+        "pathe": "^2.0.3",
+        "perfect-debounce": "^2.0.0",
+        "pkg-types": "^2.3.0",
+        "rc9": "^2.1.2"
+      },
+      "peerDependencies": {
+        "magicast": "*"
+      },
+      "peerDependenciesMeta": {
+        "magicast": {
+          "optional": true
+        }
+      }
+    },
+    "node_modules/callsites": {
+      "version": "3.1.0",
+      "resolved": "https://registry.npmjs.org/callsites/-/callsites-3.1.0.tgz",
+      "integrity": "sha512-P8BjAsXvZS+VIDUI11hHCQEv74YT67YUi5JJFNWIqL235sBmjX4+qx9Muvls5ivyNENctx46xQLQ3aTuE7ssaQ==",
+      "dev": true,
+      "license": "MIT",
+      "engines": {
+        "node": ">=6"
+      }
+    },
+    "node_modules/caniuse-lite": {
+      "version": "1.0.30001777",
+      "resolved": "https://registry.npmjs.org/caniuse-lite/-/caniuse-lite-1.0.30001777.tgz",
+      "integrity": "sha512-tmN+fJxroPndC74efCdp12j+0rk0RHwV5Jwa1zWaFVyw2ZxAuPeG8ZgWC3Wz7uSjT3qMRQ5XHZ4COgQmsCMJAQ==",
+      "dev": true,
+      "funding": [
+        {
+          "type": "opencollective",
+          "url": "https://opencollective.com/browserslist"
+        },
+        {
+          "type": "tidelift",
+          "url": "https://tidelift.com/funding/github/npm/caniuse-lite"
+        },
+        {
+          "type": "github",
+          "url": "https://github.com/sponsors/ai"
+        }
+      ],
+      "license": "CC-BY-4.0"
+    },
+    "node_modules/ccount": {
+      "version": "2.0.1",
+      "resolved": "https://registry.npmjs.org/ccount/-/ccount-2.0.1.tgz",
+      "integrity": "sha512-eyrF0jiFpY+3drT6383f1qhkbGsLSifNAjA61IUjZjmLCWjItY6LB9ft9YhoDgwfmclB2zhu51Lc7+95b8NRAg==",
+      "dev": true,
+      "license": "MIT",
+      "funding": {
+        "type": "github",
+        "url": "https://github.com/sponsors/wooorm"
+      }
+    },
+    "node_modules/chai": {
+      "version": "6.2.2",
+      "resolved": "https://registry.npmjs.org/chai/-/chai-6.2.2.tgz",
+      "integrity": "sha512-NUPRluOfOiTKBKvWPtSD4PhFvWCqOi0BGStNWs57X9js7XGTprSmFoz5F0tWhR4WPjNeR9jXqdC7/UpSJTnlRg==",
+      "dev": true,
+      "license": "MIT",
+      "engines": {
+        "node": ">=18"
+      }
+    },
+    "node_modules/chalk": {
+      "version": "4.1.2",
+      "resolved": "https://registry.npmjs.org/chalk/-/chalk-4.1.2.tgz",
+      "integrity": "sha512-oKnbhFyRIXpUuez8iBMmyEa4nbj4IOQyuhc/wy9kY7/WVPcwIO9VA668Pu8RkO7+0G76SLROeyw9CpQ061i4mA==",
+      "dev": true,
+      "license": "MIT",
+      "dependencies": {
+        "ansi-styles": "^4.1.0",
+        "supports-color": "^7.1.0"
+      },
+      "engines": {
+        "node": ">=10"
+      },
+      "funding": {
+        "url": "https://github.com/chalk/chalk?sponsor=1"
+      }
+    },
+    "node_modules/change-case": {
+      "version": "5.4.4",
+      "resolved": "https://registry.npmjs.org/change-case/-/change-case-5.4.4.tgz",
+      "integrity": "sha512-HRQyTk2/YPEkt9TnUPbOpr64Uw3KOicFWPVBb+xiHvd6eBx/qPr9xqfBFDT8P2vWsvvz4jbEkfDe71W3VyNu2w==",
+      "dev": true,
+      "license": "MIT"
+    },
+    "node_modules/changelogen": {
+      "version": "0.6.2",
+      "resolved": "https://registry.npmjs.org/changelogen/-/changelogen-0.6.2.tgz",
+      "integrity": "sha512-QtC7+r9BxoUm+XDAwhLbz3CgU134J1ytfE3iCpLpA4KFzX2P1e6s21RrWDwUBzfx66b1Rv+6lOA2nS2btprd+A==",
+      "dev": true,
+      "license": "MIT",
+      "dependencies": {
+        "c12": "^3.0.4",
+        "confbox": "^0.2.2",
+        "consola": "^3.4.2",
+        "convert-gitmoji": "^0.1.5",
+        "mri": "^1.2.0",
+        "node-fetch-native": "^1.6.6",
+        "ofetch": "^1.4.1",
+        "open": "^10.1.2",
+        "pathe": "^2.0.3",
+        "pkg-types": "^2.2.0",
+        "scule": "^1.3.0",
+        "semver": "^7.7.2",
+        "std-env": "^3.9.0"
+      },
+      "bin": {
+        "changelogen": "dist/cli.mjs"
+      }
+    },
+    "node_modules/character-entities": {
+      "version": "2.0.2",
+      "resolved": "https://registry.npmjs.org/character-entities/-/character-entities-2.0.2.tgz",
+      "integrity": "sha512-shx7oQ0Awen/BRIdkjkvz54PnEEI/EjwXDSIZp86/KKdbafHh1Df/RYGBhn4hbe2+uKC9FnT5UCEdyPz3ai9hQ==",
+      "dev": true,
+      "license": "MIT",
+      "funding": {
+        "type": "github",
+        "url": "https://github.com/sponsors/wooorm"
+      }
+    },
+    "node_modules/chokidar": {
+      "version": "5.0.0",
+      "resolved": "https://registry.npmjs.org/chokidar/-/chokidar-5.0.0.tgz",
+      "integrity": "sha512-TQMmc3w+5AxjpL8iIiwebF73dRDF4fBIieAqGn9RGCWaEVwQ6Fb2cGe31Yns0RRIzii5goJ1Y7xbMwo1TxMplw==",
+      "dev": true,
+      "license": "MIT",
+      "dependencies": {
+        "readdirp": "^5.0.0"
+      },
+      "engines": {
+        "node": ">= 20.19.0"
+      },
+      "funding": {
+        "url": "https://paulmillr.com/funding/"
+      }
+    },
+    "node_modules/ci-info": {
+      "version": "4.4.0",
+      "resolved": "https://registry.npmjs.org/ci-info/-/ci-info-4.4.0.tgz",
+      "integrity": "sha512-77PSwercCZU2Fc4sX94eF8k8Pxte6JAwL4/ICZLFjJLqegs7kCuAsqqj/70NQF6TvDpgFjkubQB2FW2ZZddvQg==",
+      "dev": true,
+      "funding": [
+        {
+          "type": "github",
+          "url": "https://github.com/sponsors/sibiraj-s"
+        }
+      ],
+      "license": "MIT",
+      "engines": {
+        "node": ">=8"
+      }
+    },
+    "node_modules/citty": {
+      "version": "0.2.1",
+      "resolved": "https://registry.npmjs.org/citty/-/citty-0.2.1.tgz",
+      "integrity": "sha512-kEV95lFBhQgtogAPlQfJJ0WGVSokvLr/UEoFPiKKOXF7pl98HfUVUD0ejsuTCld/9xH9vogSywZ5KqHzXrZpqg==",
+      "dev": true,
+      "license": "MIT"
+    },
+    "node_modules/clean-regexp": {
+      "version": "1.0.0",
+      "resolved": "https://registry.npmjs.org/clean-regexp/-/clean-regexp-1.0.0.tgz",
+      "integrity": "sha512-GfisEZEJvzKrmGWkvfhgzcz/BllN1USeqD2V6tg14OAOgaCD2Z/PUEuxnAZ/nPvmaHRG7a8y77p1T/IRQ4D1Hw==",
+      "dev": true,
+      "license": "MIT",
+      "dependencies": {
+        "escape-string-regexp": "^1.0.5"
+      },
+      "engines": {
+        "node": ">=4"
+      }
+    },
+    "node_modules/clean-regexp/node_modules/escape-string-regexp": {
+      "version": "1.0.5",
+      "resolved": "https://registry.npmjs.org/escape-string-regexp/-/escape-string-regexp-1.0.5.tgz",
+      "integrity": "sha512-vbRorB5FUQWvla16U8R/qgaFIya2qGzwDrNmCZuYKrbdSUMG6I1ZCGQRefkRVhuOkIGVne7BQ35DSfo1qvJqFg==",
+      "dev": true,
+      "license": "MIT",
+      "engines": {
+        "node": ">=0.8.0"
+      }
+    },
+    "node_modules/color-convert": {
+      "version": "2.0.1",
+      "resolved": "https://registry.npmjs.org/color-convert/-/color-convert-2.0.1.tgz",
+      "integrity": "sha512-RRECPsj7iu/xb5oKYcsFHSppFNnsj/52OVTRKb4zP5onXwVF3zVmmToNcOfGC+CRDpfK/U584fMg38ZHCaElKQ==",
+      "dev": true,
+      "license": "MIT",
+      "dependencies": {
+        "color-name": "~1.1.4"
+      },
+      "engines": {
+        "node": ">=7.0.0"
+      }
+    },
+    "node_modules/color-name": {
+      "version": "1.1.4",
+      "resolved": "https://registry.npmjs.org/color-name/-/color-name-1.1.4.tgz",
+      "integrity": "sha512-dOy+3AuW3a2wNbZHIuMZpTcgjGuLU/uBL/ubcZF9OXbDo8ff4O8yVp5Bf0efS8uEoYo5q4Fx7dY9OgQGXgAsQA==",
+      "dev": true,
+      "license": "MIT"
+    },
+    "node_modules/commenting": {
+      "version": "1.1.0",
+      "resolved": "https://registry.npmjs.org/commenting/-/commenting-1.1.0.tgz",
+      "integrity": "sha512-YeNK4tavZwtH7jEgK1ZINXzLKm6DZdEMfsaaieOsCAN0S8vsY7UeuO3Q7d/M018EFgE+IeUAuBOKkFccBZsUZA==",
+      "dev": true,
+      "license": "MIT"
+    },
+    "node_modules/concat-map": {
+      "version": "0.0.1",
+      "resolved": "https://registry.npmjs.org/concat-map/-/concat-map-0.0.1.tgz",
+      "integrity": "sha512-/Srv4dswyQNBfohGpz9o6Yb3Gz3SrUDqBH5rTuhGR7ahtlbYKnVxw2bCFMRljaA7EXHaXZ8wsHdodFvbkhKmqg==",
+      "dev": true,
+      "license": "MIT"
+    },
+    "node_modules/confbox": {
+      "version": "0.2.4",
+      "resolved": "https://registry.npmjs.org/confbox/-/confbox-0.2.4.tgz",
+      "integrity": "sha512-ysOGlgTFbN2/Y6Cg3Iye8YKulHw+R2fNXHrgSmXISQdMnomY6eNDprVdW9R5xBguEqI954+S6709UyiO7B+6OQ==",
+      "dev": true,
+      "license": "MIT"
+    },
+    "node_modules/consola": {
+      "version": "3.4.2",
+      "resolved": "https://registry.npmjs.org/consola/-/consola-3.4.2.tgz",
+      "integrity": "sha512-5IKcdX0nnYavi6G7TtOhwkYzyjfJlatbjMjuLSfE2kYT5pMDOilZ4OvMhi637CcDICTmz3wARPoyhqyX1Y+XvA==",
+      "dev": true,
+      "license": "MIT",
+      "engines": {
+        "node": "^14.18.0 || >=16.10.0"
+      }
+    },
+    "node_modules/convert-gitmoji": {
+      "version": "0.1.5",
+      "resolved": "https://registry.npmjs.org/convert-gitmoji/-/convert-gitmoji-0.1.5.tgz",
+      "integrity": "sha512-4wqOafJdk2tqZC++cjcbGcaJ13BZ3kwldf06PTiAQRAB76Z1KJwZNL1SaRZMi2w1FM9RYTgZ6QErS8NUl/GBmQ==",
+      "dev": true,
+      "license": "MIT"
+    },
+    "node_modules/core-js-compat": {
+      "version": "3.48.0",
+      "resolved": "https://registry.npmjs.org/core-js-compat/-/core-js-compat-3.48.0.tgz",
+      "integrity": "sha512-OM4cAF3D6VtH/WkLtWvyNC56EZVXsZdU3iqaMG2B4WvYrlqU831pc4UtG5yp0sE9z8Y02wVN7PjW5Zf9Gt0f1Q==",
+      "dev": true,
+      "license": "MIT",
+      "dependencies": {
+        "browserslist": "^4.28.1"
+      },
+      "funding": {
+        "type": "opencollective",
+        "url": "https://opencollective.com/core-js"
+      }
+    },
+    "node_modules/cross-spawn": {
+      "version": "7.0.6",
+      "resolved": "https://registry.npmjs.org/cross-spawn/-/cross-spawn-7.0.6.tgz",
+      "integrity": "sha512-uV2QOWP2nWzsy2aMp8aRibhi9dlzF5Hgh5SHaB9OiTGEyDTiJJyx0uy51QXdyWbtAHNua4XJzUKca3OzKUd3vA==",
+      "dev": true,
+      "license": "MIT",
+      "dependencies": {
+        "path-key": "^3.1.0",
+        "shebang-command": "^2.0.0",
+        "which": "^2.0.1"
+      },
+      "engines": {
+        "node": ">= 8"
+      }
+    },
+    "node_modules/debug": {
+      "version": "4.4.3",
+      "resolved": "https://registry.npmjs.org/debug/-/debug-4.4.3.tgz",
+      "integrity": "sha512-RGwwWnwQvkVfavKVt22FGLw+xYSdzARwm0ru6DhTVA3umU5hZc28V3kO4stgYryrTlLpuvgI9GiijltAjNbcqA==",
+      "dev": true,
+      "license": "MIT",
+      "dependencies": {
+        "ms": "^2.1.3"
+      },
+      "engines": {
+        "node": ">=6.0"
+      },
+      "peerDependenciesMeta": {
+        "supports-color": {
+          "optional": true
+        }
+      }
+    },
+    "node_modules/decode-named-character-reference": {
+      "version": "1.3.0",
+      "resolved": "https://registry.npmjs.org/decode-named-character-reference/-/decode-named-character-reference-1.3.0.tgz",
+      "integrity": "sha512-GtpQYB283KrPp6nRw50q3U9/VfOutZOe103qlN7BPP6Ad27xYnOIWv4lPzo8HCAL+mMZofJ9KEy30fq6MfaK6Q==",
+      "dev": true,
+      "license": "MIT",
+      "dependencies": {
+        "character-entities": "^2.0.0"
+      },
+      "funding": {
+        "type": "github",
+        "url": "https://github.com/sponsors/wooorm"
+      }
+    },
+    "node_modules/deep-is": {
+      "version": "0.1.4",
+      "resolved": "https://registry.npmjs.org/deep-is/-/deep-is-0.1.4.tgz",
+      "integrity": "sha512-oIPzksmTg4/MriiaYGO+okXDT7ztn/w3Eptv/+gSIdMdKsJo0u4CfYNFJPy+4SKMuCqGw2wxnA+URMg3t8a/bQ==",
+      "dev": true,
+      "license": "MIT"
+    },
+    "node_modules/default-browser": {
+      "version": "5.5.0",
+      "resolved": "https://registry.npmjs.org/default-browser/-/default-browser-5.5.0.tgz",
+      "integrity": "sha512-H9LMLr5zwIbSxrmvikGuI/5KGhZ8E2zH3stkMgM5LpOWDutGM2JZaj460Udnf1a+946zc7YBgrqEWwbk7zHvGw==",
+      "dev": true,
+      "license": "MIT",
+      "dependencies": {
+        "bundle-name": "^4.1.0",
+        "default-browser-id": "^5.0.0"
+      },
+      "engines": {
+        "node": ">=18"
+      },
+      "funding": {
+        "url": "https://github.com/sponsors/sindresorhus"
+      }
+    },
+    "node_modules/default-browser-id": {
+      "version": "5.0.1",
+      "resolved": "https://registry.npmjs.org/default-browser-id/-/default-browser-id-5.0.1.tgz",
+      "integrity": "sha512-x1VCxdX4t+8wVfd1so/9w+vQ4vx7lKd2Qp5tDRutErwmR85OgmfX7RlLRMWafRMY7hbEiXIbudNrjOAPa/hL8Q==",
+      "dev": true,
+      "license": "MIT",
+      "engines": {
+        "node": ">=18"
+      },
+      "funding": {
+        "url": "https://github.com/sponsors/sindresorhus"
+      }
+    },
+    "node_modules/define-lazy-prop": {
+      "version": "3.0.0",
+      "resolved": "https://registry.npmjs.org/define-lazy-prop/-/define-lazy-prop-3.0.0.tgz",
+      "integrity": "sha512-N+MeXYoqr3pOgn8xfyRPREN7gHakLYjhsHhWGT3fWAiL4IkAt0iDw14QiiEm2bE30c5XX5q0FtAA3CK5f9/BUg==",
+      "dev": true,
+      "license": "MIT",
+      "engines": {
+        "node": ">=12"
+      },
+      "funding": {
+        "url": "https://github.com/sponsors/sindresorhus"
+      }
+    },
+    "node_modules/defu": {
+      "version": "6.1.4",
+      "resolved": "https://registry.npmjs.org/defu/-/defu-6.1.4.tgz",
+      "integrity": "sha512-mEQCMmwJu317oSz8CwdIOdwf3xMif1ttiM8LTufzc3g6kR+9Pe236twL8j3IYT1F7GfRgGcW6MWxzZjLIkuHIg==",
+      "dev": true,
+      "license": "MIT"
+    },
+    "node_modules/dequal": {
+      "version": "2.0.3",
+      "resolved": "https://registry.npmjs.org/dequal/-/dequal-2.0.3.tgz",
+      "integrity": "sha512-0je+qPKHEMohvfRTCEo3CrPG6cAzAYgmzKyxRiYSSDkS6eGJdyVJm7WaYA5ECaAD9wLB2T4EEeymA5aFVcYXCA==",
+      "dev": true,
+      "license": "MIT",
+      "engines": {
+        "node": ">=6"
+      }
+    },
+    "node_modules/destr": {
+      "version": "2.0.5",
+      "resolved": "https://registry.npmjs.org/destr/-/destr-2.0.5.tgz",
+      "integrity": "sha512-ugFTXCtDZunbzasqBxrK93Ik/DRYsO6S/fedkWEMKqt04xZ4csmnmwGDBAb07QWNaGMAmnTIemsYZCksjATwsA==",
+      "dev": true,
+      "license": "MIT"
+    },
+    "node_modules/detect-libc": {
+      "version": "2.1.2",
+      "resolved": "https://registry.npmjs.org/detect-libc/-/detect-libc-2.1.2.tgz",
+      "integrity": "sha512-Btj2BOOO83o3WyH59e8MgXsxEQVcarkUOpEYrubB0urwnN10yQ364rsiByU11nZlqWYZm05i/of7io4mzihBtQ==",
+      "dev": true,
+      "license": "Apache-2.0",
+      "engines": {
+        "node": ">=8"
+      }
+    },
+    "node_modules/devlop": {
+      "version": "1.1.0",
+      "resolved": "https://registry.npmjs.org/devlop/-/devlop-1.1.0.tgz",
+      "integrity": "sha512-RWmIqhcFf1lRYBvNmr7qTNuyCt/7/ns2jbpp1+PalgE/rDQcBT0fioSMUpJ93irlUhC5hrg4cYqe6U+0ImW0rA==",
+      "dev": true,
+      "license": "MIT",
+      "dependencies": {
+        "dequal": "^2.0.0"
+      },
+      "funding": {
+        "type": "github",
+        "url": "https://github.com/sponsors/wooorm"
+      }
+    },
+    "node_modules/didyoumean2": {
+      "version": "7.0.4",
+      "resolved": "https://registry.npmjs.org/didyoumean2/-/didyoumean2-7.0.4.tgz",
+      "integrity": "sha512-+yW4SNY7W2DOWe2Jx5H4c2qMTFbLGM6wIyoDPkAPy66X+sD1KfYjBPAIWPVsYqMxelflaMQCloZDudELIPhLqA==",
+      "dev": true,
+      "license": "MIT",
+      "dependencies": {
+        "@babel/runtime": "^7.24.8",
+        "fastest-levenshtein": "^1.0.16",
+        "lodash.deburr": "^4.1.0"
+      },
+      "engines": {
+        "node": "^18.12.0 || >=20.9.0"
+      }
+    },
+    "node_modules/dotenv": {
+      "version": "17.3.1",
+      "resolved": "https://registry.npmjs.org/dotenv/-/dotenv-17.3.1.tgz",
+      "integrity": "sha512-IO8C/dzEb6O3F9/twg6ZLXz164a2fhTnEWb95H23Dm4OuN+92NmEAlTrupP9VW6Jm3sO26tQlqyvyi4CsnY9GA==",
+      "dev": true,
+      "license": "BSD-2-Clause",
+      "engines": {
+        "node": ">=12"
+      },
+      "funding": {
+        "url": "https://dotenvx.com"
+      }
+    },
+    "node_modules/dts-resolver": {
+      "version": "2.1.3",
+      "resolved": "https://registry.npmjs.org/dts-resolver/-/dts-resolver-2.1.3.tgz",
+      "integrity": "sha512-bihc7jPC90VrosXNzK0LTE2cuLP6jr0Ro8jk+kMugHReJVLIpHz/xadeq3MhuwyO4TD4OA3L1Q8pBBFRc08Tsw==",
+      "dev": true,
+      "license": "MIT",
+      "engines": {
+        "node": ">=20.19.0"
+      },
+      "funding": {
+        "url": "https://github.com/sponsors/sxzz"
+      },
+      "peerDependencies": {
+        "oxc-resolver": ">=11.0.0"
+      },
+      "peerDependenciesMeta": {
+        "oxc-resolver": {
+          "optional": true
+        }
+      }
+    },
+    "node_modules/electron-to-chromium": {
+      "version": "1.5.307",
+      "resolved": "https://registry.npmjs.org/electron-to-chromium/-/electron-to-chromium-1.5.307.tgz",
+      "integrity": "sha512-5z3uFKBWjiNR44nFcYdkcXjKMbg5KXNdciu7mhTPo9tB7NbqSNP2sSnGR+fqknZSCwKkBN+oxiiajWs4dT6ORg==",
+      "dev": true,
+      "license": "ISC"
+    },
+    "node_modules/es-module-lexer": {
+      "version": "1.7.0",
+      "resolved": "https://registry.npmjs.org/es-module-lexer/-/es-module-lexer-1.7.0.tgz",
+      "integrity": "sha512-jEQoCwk8hyb2AZziIOLhDqpm5+2ww5uIE6lkO/6jcOCusfk6LhMHpXXfBLXTZ7Ydyt0j4VoUQv6uGNYbdW+kBA==",
+      "dev": true,
+      "license": "MIT"
+    },
+    "node_modules/esbuild": {
+      "version": "0.27.3",
+      "resolved": "https://registry.npmjs.org/esbuild/-/esbuild-0.27.3.tgz",
+      "integrity": "sha512-8VwMnyGCONIs6cWue2IdpHxHnAjzxnw2Zr7MkVxB2vjmQ2ivqGFb4LEG3SMnv0Gb2F/G/2yA8zUaiL1gywDCCg==",
+      "dev": true,
+      "hasInstallScript": true,
+      "license": "MIT",
+      "bin": {
+        "esbuild": "bin/esbuild"
+      },
+      "engines": {
+        "node": ">=18"
+      },
+      "optionalDependencies": {
+        "@esbuild/aix-ppc64": "0.27.3",
+        "@esbuild/android-arm": "0.27.3",
+        "@esbuild/android-arm64": "0.27.3",
+        "@esbuild/android-x64": "0.27.3",
+        "@esbuild/darwin-arm64": "0.27.3",
+        "@esbuild/darwin-x64": "0.27.3",
+        "@esbuild/freebsd-arm64": "0.27.3",
+        "@esbuild/freebsd-x64": "0.27.3",
+        "@esbuild/linux-arm": "0.27.3",
+        "@esbuild/linux-arm64": "0.27.3",
+        "@esbuild/linux-ia32": "0.27.3",
+        "@esbuild/linux-loong64": "0.27.3",
+        "@esbuild/linux-mips64el": "0.27.3",
+        "@esbuild/linux-ppc64": "0.27.3",
+        "@esbuild/linux-riscv64": "0.27.3",
+        "@esbuild/linux-s390x": "0.27.3",
+        "@esbuild/linux-x64": "0.27.3",
+        "@esbuild/netbsd-arm64": "0.27.3",
+        "@esbuild/netbsd-x64": "0.27.3",
+        "@esbuild/openbsd-arm64": "0.27.3",
+        "@esbuild/openbsd-x64": "0.27.3",
+        "@esbuild/openharmony-arm64": "0.27.3",
+        "@esbuild/sunos-x64": "0.27.3",
+        "@esbuild/win32-arm64": "0.27.3",
+        "@esbuild/win32-ia32": "0.27.3",
+        "@esbuild/win32-x64": "0.27.3"
+      }
+    },
+    "node_modules/escalade": {
+      "version": "3.2.0",
+      "resolved": "https://registry.npmjs.org/escalade/-/escalade-3.2.0.tgz",
+      "integrity": "sha512-WUj2qlxaQtO4g6Pq5c29GTcWGDyd8itL8zTlipgECz3JesAiiOKotd8JU6otB3PACgG6xkJUyVhboMS+bje/jA==",
+      "dev": true,
+      "license": "MIT",
+      "engines": {
+        "node": ">=6"
+      }
+    },
+    "node_modules/escape-string-regexp": {
+      "version": "4.0.0",
+      "resolved": "https://registry.npmjs.org/escape-string-regexp/-/escape-string-regexp-4.0.0.tgz",
+      "integrity": "sha512-TtpcNJ3XAzx3Gq8sWRzJaVajRs0uVxA2YAkdb1jm2YkPz4G6egUFAyA3n5vtEIZefPk5Wa4UXbKuS5fKkJWdgA==",
+      "dev": true,
+      "license": "MIT",
+      "engines": {
+        "node": ">=10"
+      },
+      "funding": {
+        "url": "https://github.com/sponsors/sindresorhus"
+      }
+    },
+    "node_modules/eslint": {
+      "version": "9.39.4",
+      "resolved": "https://registry.npmjs.org/eslint/-/eslint-9.39.4.tgz",
+      "integrity": "sha512-XoMjdBOwe/esVgEvLmNsD3IRHkm7fbKIUGvrleloJXUZgDHig2IPWNniv+GwjyJXzuNqVjlr5+4yVUZjycJwfQ==",
+      "dev": true,
+      "license": "MIT",
+      "dependencies": {
+        "@eslint-community/eslint-utils": "^4.8.0",
+        "@eslint-community/regexpp": "^4.12.1",
+        "@eslint/config-array": "^0.21.2",
+        "@eslint/config-helpers": "^0.4.2",
+        "@eslint/core": "^0.17.0",
+        "@eslint/eslintrc": "^3.3.5",
+        "@eslint/js": "9.39.4",
+        "@eslint/plugin-kit": "^0.4.1",
+        "@humanfs/node": "^0.16.6",
+        "@humanwhocodes/module-importer": "^1.0.1",
+        "@humanwhocodes/retry": "^0.4.2",
+        "@types/estree": "^1.0.6",
+        "ajv": "^6.14.0",
+        "chalk": "^4.0.0",
+        "cross-spawn": "^7.0.6",
+        "debug": "^4.3.2",
+        "escape-string-regexp": "^4.0.0",
+        "eslint-scope": "^8.4.0",
+        "eslint-visitor-keys": "^4.2.1",
+        "espree": "^10.4.0",
+        "esquery": "^1.5.0",
+        "esutils": "^2.0.2",
+        "fast-deep-equal": "^3.1.3",
+        "file-entry-cache": "^8.0.0",
+        "find-up": "^5.0.0",
+        "glob-parent": "^6.0.2",
+        "ignore": "^5.2.0",
+        "imurmurhash": "^0.1.4",
+        "is-glob": "^4.0.0",
+        "json-stable-stringify-without-jsonify": "^1.0.1",
+        "lodash.merge": "^4.6.2",
+        "minimatch": "^3.1.5",
+        "natural-compare": "^1.4.0",
+        "optionator": "^0.9.3"
+      },
+      "bin": {
+        "eslint": "bin/eslint.js"
+      },
+      "engines": {
+        "node": "^18.18.0 || ^20.9.0 || >=21.1.0"
+      },
+      "funding": {
+        "url": "https://eslint.org/donate"
+      },
+      "peerDependencies": {
+        "jiti": "*"
+      },
+      "peerDependenciesMeta": {
+        "jiti": {
+          "optional": true
+        }
+      }
+    },
+    "node_modules/eslint-config-unjs": {
+      "version": "0.6.2",
+      "resolved": "https://registry.npmjs.org/eslint-config-unjs/-/eslint-config-unjs-0.6.2.tgz",
+      "integrity": "sha512-u80AQy4W4B+gbd/Ow1xlUrvNhTq/tqs1mpfYNzFfazp1wF5L0ZY4gizspfXgr8QDeugMfo+rno6Iwd89jKSJSw==",
+      "dev": true,
+      "license": "MIT",
+      "dependencies": {
+        "@eslint/js": "^9.39.2",
+        "@eslint/markdown": "^7.5.1",
+        "eslint-plugin-unicorn": "^62.0.0",
+        "globals": "^17.0.0",
+        "typescript-eslint": "^8.51.0"
+      },
+      "peerDependencies": {
+        "eslint": "*",
+        "typescript": "*"
+      }
+    },
+    "node_modules/eslint-config-unjs/node_modules/globals": {
+      "version": "17.4.0",
+      "resolved": "https://registry.npmjs.org/globals/-/globals-17.4.0.tgz",
+      "integrity": "sha512-hjrNztw/VajQwOLsMNT1cbJiH2muO3OROCHnbehc8eY5JyD2gqz4AcMHPqgaOR59DjgUjYAYLeH699g/eWi2jw==",
+      "dev": true,
+      "license": "MIT",
+      "engines": {
+        "node": ">=18"
+      },
+      "funding": {
+        "url": "https://github.com/sponsors/sindresorhus"
+      }
+    },
+    "node_modules/eslint-plugin-unicorn": {
+      "version": "62.0.0",
+      "resolved": "https://registry.npmjs.org/eslint-plugin-unicorn/-/eslint-plugin-unicorn-62.0.0.tgz",
+      "integrity": "sha512-HIlIkGLkvf29YEiS/ImuDZQbP12gWyx5i3C6XrRxMvVdqMroCI9qoVYCoIl17ChN+U89pn9sVwLxhIWj5nEc7g==",
+      "dev": true,
+      "license": "MIT",
+      "dependencies": {
+        "@babel/helper-validator-identifier": "^7.28.5",
+        "@eslint-community/eslint-utils": "^4.9.0",
+        "@eslint/plugin-kit": "^0.4.0",
+        "change-case": "^5.4.4",
+        "ci-info": "^4.3.1",
+        "clean-regexp": "^1.0.0",
+        "core-js-compat": "^3.46.0",
+        "esquery": "^1.6.0",
+        "find-up-simple": "^1.0.1",
+        "globals": "^16.4.0",
+        "indent-string": "^5.0.0",
+        "is-builtin-module": "^5.0.0",
+        "jsesc": "^3.1.0",
+        "pluralize": "^8.0.0",
+        "regexp-tree": "^0.1.27",
+        "regjsparser": "^0.13.0",
+        "semver": "^7.7.3",
+        "strip-indent": "^4.1.1"
+      },
+      "engines": {
+        "node": "^20.10.0 || >=21.0.0"
+      },
+      "funding": {
+        "url": "https://github.com/sindresorhus/eslint-plugin-unicorn?sponsor=1"
+      },
+      "peerDependencies": {
+        "eslint": ">=9.38.0"
+      }
+    },
+    "node_modules/eslint-plugin-unicorn/node_modules/globals": {
+      "version": "16.5.0",
+      "resolved": "https://registry.npmjs.org/globals/-/globals-16.5.0.tgz",
+      "integrity": "sha512-c/c15i26VrJ4IRt5Z89DnIzCGDn9EcebibhAOjw5ibqEHsE1wLUgkPn9RDmNcUKyU87GeaL633nyJ+pplFR2ZQ==",
+      "dev": true,
+      "license": "MIT",
+      "engines": {
+        "node": ">=18"
+      },
+      "funding": {
+        "url": "https://github.com/sponsors/sindresorhus"
+      }
+    },
+    "node_modules/eslint-scope": {
+      "version": "8.4.0",
+      "resolved": "https://registry.npmjs.org/eslint-scope/-/eslint-scope-8.4.0.tgz",
+      "integrity": "sha512-sNXOfKCn74rt8RICKMvJS7XKV/Xk9kA7DyJr8mJik3S7Cwgy3qlkkmyS2uQB3jiJg6VNdZd/pDBJu0nvG2NlTg==",
+      "dev": true,
+      "license": "BSD-2-Clause",
+      "dependencies": {
+        "esrecurse": "^4.3.0",
+        "estraverse": "^5.2.0"
+      },
+      "engines": {
+        "node": "^18.18.0 || ^20.9.0 || >=21.1.0"
+      },
+      "funding": {
+        "url": "https://opencollective.com/eslint"
+      }
+    },
+    "node_modules/eslint-visitor-keys": {
+      "version": "4.2.1",
+      "resolved": "https://registry.npmjs.org/eslint-visitor-keys/-/eslint-visitor-keys-4.2.1.tgz",
+      "integrity": "sha512-Uhdk5sfqcee/9H/rCOJikYz67o0a2Tw2hGRPOG2Y1R2dg7brRe1uG0yaNQDHu+TO/uQPF/5eCapvYSmHUjt7JQ==",
+      "dev": true,
+      "license": "Apache-2.0",
+      "engines": {
+        "node": "^18.18.0 || ^20.9.0 || >=21.1.0"
+      },
+      "funding": {
+        "url": "https://opencollective.com/eslint"
+      }
+    },
+    "node_modules/espree": {
+      "version": "10.4.0",
+      "resolved": "https://registry.npmjs.org/espree/-/espree-10.4.0.tgz",
+      "integrity": "sha512-j6PAQ2uUr79PZhBjP5C5fhl8e39FmRnOjsD5lGnWrFU8i2G776tBK7+nP8KuQUTTyAZUwfQqXAgrVH5MbH9CYQ==",
+      "dev": true,
+      "license": "BSD-2-Clause",
+      "dependencies": {
+        "acorn": "^8.15.0",
+        "acorn-jsx": "^5.3.2",
+        "eslint-visitor-keys": "^4.2.1"
+      },
+      "engines": {
+        "node": "^18.18.0 || ^20.9.0 || >=21.1.0"
+      },
+      "funding": {
+        "url": "https://opencollective.com/eslint"
+      }
+    },
+    "node_modules/esquery": {
+      "version": "1.7.0",
+      "resolved": "https://registry.npmjs.org/esquery/-/esquery-1.7.0.tgz",
+      "integrity": "sha512-Ap6G0WQwcU/LHsvLwON1fAQX9Zp0A2Y6Y/cJBl9r/JbW90Zyg4/zbG6zzKa2OTALELarYHmKu0GhpM5EO+7T0g==",
+      "dev": true,
+      "license": "BSD-3-Clause",
+      "dependencies": {
+        "estraverse": "^5.1.0"
+      },
+      "engines": {
+        "node": ">=0.10"
+      }
+    },
+    "node_modules/esrecurse": {
+      "version": "4.3.0",
+      "resolved": "https://registry.npmjs.org/esrecurse/-/esrecurse-4.3.0.tgz",
+      "integrity": "sha512-KmfKL3b6G+RXvP8N1vr3Tq1kL/oCFgn2NYXEtqP8/L3pKapUA4G8cFVaoF3SU323CD4XypR/ffioHmkti6/Tag==",
+      "dev": true,
+      "license": "BSD-2-Clause",
+      "dependencies": {
+        "estraverse": "^5.2.0"
+      },
+      "engines": {
+        "node": ">=4.0"
+      }
+    },
+    "node_modules/estraverse": {
+      "version": "5.3.0",
+      "resolved": "https://registry.npmjs.org/estraverse/-/estraverse-5.3.0.tgz",
+      "integrity": "sha512-MMdARuVEQziNTeJD8DgMqmhwR11BRQ/cBP+pLtYdSTnf3MIO8fFeiINEbX36ZdNlfU/7A9f3gUw49B3oQsvwBA==",
+      "dev": true,
+      "license": "BSD-2-Clause",
+      "engines": {
+        "node": ">=4.0"
+      }
+    },
+    "node_modules/estree-walker": {
+      "version": "3.0.3",
+      "resolved": "https://registry.npmjs.org/estree-walker/-/estree-walker-3.0.3.tgz",
+      "integrity": "sha512-7RUKfXgSMMkzt6ZuXmqapOurLGPPfgj6l9uRZ7lRGolvk0y2yocc35LdcxKC5PQZdn2DMqioAQ2NoWcrTKmm6g==",
+      "dev": true,
+      "license": "MIT",
+      "dependencies": {
+        "@types/estree": "^1.0.0"
+      }
+    },
+    "node_modules/esutils": {
+      "version": "2.0.3",
+      "resolved": "https://registry.npmjs.org/esutils/-/esutils-2.0.3.tgz",
+      "integrity": "sha512-kVscqXk4OCp68SZ0dkgEKVi6/8ij300KBWTJq32P/dYeWTSwK41WyTxalN1eRmA5Z9UU/LX9D7FWSmV9SAYx6g==",
+      "dev": true,
+      "license": "BSD-2-Clause",
+      "engines": {
+        "node": ">=0.10.0"
+      }
+    },
+    "node_modules/expect-type": {
+      "version": "1.3.0",
+      "resolved": "https://registry.npmjs.org/expect-type/-/expect-type-1.3.0.tgz",
+      "integrity": "sha512-knvyeauYhqjOYvQ66MznSMs83wmHrCycNEN6Ao+2AeYEfxUIkuiVxdEa1qlGEPK+We3n0THiDciYSsCcgW/DoA==",
+      "dev": true,
+      "license": "Apache-2.0",
+      "engines": {
+        "node": ">=12.0.0"
+      }
+    },
+    "node_modules/exsolve": {
+      "version": "1.0.8",
+      "resolved": "https://registry.npmjs.org/exsolve/-/exsolve-1.0.8.tgz",
+      "integrity": "sha512-LmDxfWXwcTArk8fUEnOfSZpHOJ6zOMUJKOtFLFqJLoKJetuQG874Uc7/Kki7zFLzYybmZhp1M7+98pfMqeX8yA==",
+      "dev": true,
+      "license": "MIT"
+    },
+    "node_modules/fast-deep-equal": {
+      "version": "3.1.3",
+      "resolved": "https://registry.npmjs.org/fast-deep-equal/-/fast-deep-equal-3.1.3.tgz",
+      "integrity": "sha512-f3qQ9oQy9j2AhBe/H9VC91wLmKBCCU/gDOnKNAYG5hswO7BLKj09Hc5HYNz9cGI++xlpDCIgDaitVs03ATR84Q==",
+      "dev": true,
+      "license": "MIT"
+    },
+    "node_modules/fast-json-stable-stringify": {
+      "version": "2.1.0",
+      "resolved": "https://registry.npmjs.org/fast-json-stable-stringify/-/fast-json-stable-stringify-2.1.0.tgz",
+      "integrity": "sha512-lhd/wF+Lk98HZoTCtlVraHtfh5XYijIjalXck7saUtuanSDyLMxnHhSXEDJqHxD7msR8D0uCmqlkwjCV8xvwHw==",
+      "dev": true,
+      "license": "MIT"
+    },
+    "node_modules/fast-levenshtein": {
+      "version": "2.0.6",
+      "resolved": "https://registry.npmjs.org/fast-levenshtein/-/fast-levenshtein-2.0.6.tgz",
+      "integrity": "sha512-DCXu6Ifhqcks7TZKY3Hxp3y6qphY5SJZmrWMDrKcERSOXWQdMhU9Ig/PYrzyw/ul9jOIyh0N4M0tbC5hodg8dw==",
+      "dev": true,
+      "license": "MIT"
+    },
+    "node_modules/fastest-levenshtein": {
+      "version": "1.0.16",
+      "resolved": "https://registry.npmjs.org/fastest-levenshtein/-/fastest-levenshtein-1.0.16.tgz",
+      "integrity": "sha512-eRnCtTTtGZFpQCwhJiUOuxPQWRXVKYDn0b2PeHfXL6/Zi53SLAzAHfVhVWK2AryC/WH05kGfxhFIPvTF0SXQzg==",
+      "dev": true,
+      "license": "MIT",
+      "engines": {
+        "node": ">= 4.9.1"
+      }
+    },
+    "node_modules/fault": {
+      "version": "2.0.1",
+      "resolved": "https://registry.npmjs.org/fault/-/fault-2.0.1.tgz",
+      "integrity": "sha512-WtySTkS4OKev5JtpHXnib4Gxiurzh5NCGvWrFaZ34m6JehfTUhKZvn9njTfw48t6JumVQOmrKqpmGcdwxnhqBQ==",
+      "dev": true,
+      "license": "MIT",
+      "dependencies": {
+        "format": "^0.2.0"
+      },
+      "funding": {
+        "type": "github",
+        "url": "https://github.com/sponsors/wooorm"
+      }
+    },
+    "node_modules/fdir": {
+      "version": "6.5.0",
+      "resolved": "https://registry.npmjs.org/fdir/-/fdir-6.5.0.tgz",
+      "integrity": "sha512-tIbYtZbucOs0BRGqPJkshJUYdL+SDH7dVM8gjy+ERp3WAUjLEFJE+02kanyHtwjWOnwrKYBiwAmM0p4kLJAnXg==",
+      "dev": true,
+      "license": "MIT",
+      "engines": {
+        "node": ">=12.0.0"
+      },
+      "peerDependencies": {
+        "picomatch": "^3 || ^4"
+      },
+      "peerDependenciesMeta": {
+        "picomatch": {
+          "optional": true
+        }
+      }
+    },
+    "node_modules/file-entry-cache": {
+      "version": "8.0.0",
+      "resolved": "https://registry.npmjs.org/file-entry-cache/-/file-entry-cache-8.0.0.tgz",
+      "integrity": "sha512-XXTUwCvisa5oacNGRP9SfNtYBNAMi+RPwBFmblZEF7N7swHYQS6/Zfk7SRwx4D5j3CH211YNRco1DEMNVfZCnQ==",
+      "dev": true,
+      "license": "MIT",
+      "dependencies": {
+        "flat-cache": "^4.0.0"
+      },
+      "engines": {
+        "node": ">=16.0.0"
+      }
+    },
+    "node_modules/find-up": {
+      "version": "5.0.0",
+      "resolved": "https://registry.npmjs.org/find-up/-/find-up-5.0.0.tgz",
+      "integrity": "sha512-78/PXT1wlLLDgTzDs7sjq9hzz0vXD+zn+7wypEe4fXQxCmdmqfGsEPQxmiCSQI3ajFV91bVSsvNtrJRiW6nGng==",
+      "dev": true,
+      "license": "MIT",
+      "dependencies": {
+        "locate-path": "^6.0.0",
+        "path-exists": "^4.0.0"
+      },
+      "engines": {
+        "node": ">=10"
+      },
+      "funding": {
+        "url": "https://github.com/sponsors/sindresorhus"
+      }
+    },
+    "node_modules/find-up-simple": {
+      "version": "1.0.1",
+      "resolved": "https://registry.npmjs.org/find-up-simple/-/find-up-simple-1.0.1.tgz",
+      "integrity": "sha512-afd4O7zpqHeRyg4PfDQsXmlDe2PfdHtJt6Akt8jOWaApLOZk5JXs6VMR29lz03pRe9mpykrRCYIYxaJYcfpncQ==",
+      "dev": true,
+      "license": "MIT",
+      "engines": {
+        "node": ">=18"
+      },
+      "funding": {
+        "url": "https://github.com/sponsors/sindresorhus"
+      }
+    },
+    "node_modules/flat-cache": {
+      "version": "4.0.1",
+      "resolved": "https://registry.npmjs.org/flat-cache/-/flat-cache-4.0.1.tgz",
+      "integrity": "sha512-f7ccFPK3SXFHpx15UIGyRJ/FJQctuKZ0zVuN3frBo4HnK3cay9VEW0R6yPYFHC0AgqhukPzKjq22t5DmAyqGyw==",
+      "dev": true,
+      "license": "MIT",
+      "dependencies": {
+        "flatted": "^3.2.9",
+        "keyv": "^4.5.4"
+      },
+      "engines": {
+        "node": ">=16"
+      }
+    },
+    "node_modules/flatted": {
+      "version": "3.4.1",
+      "resolved": "https://registry.npmjs.org/flatted/-/flatted-3.4.1.tgz",
+      "integrity": "sha512-IxfVbRFVlV8V/yRaGzk0UVIcsKKHMSfYw66T/u4nTwlWteQePsxe//LjudR1AMX4tZW3WFCh3Zqa/sjlqpbURQ==",
+      "dev": true,
+      "license": "ISC"
+    },
+    "node_modules/format": {
+      "version": "0.2.2",
+      "resolved": "https://registry.npmjs.org/format/-/format-0.2.2.tgz",
+      "integrity": "sha512-wzsgA6WOq+09wrU1tsJ09udeR/YZRaeArL9e1wPbFg3GG2yDnC2ldKpxs4xunpFF9DgqCqOIra3bc1HWrJ37Ww==",
+      "dev": true,
+      "engines": {
+        "node": ">=0.4.x"
+      }
+    },
+    "node_modules/fsevents": {
+      "version": "2.3.3",
+      "resolved": "https://registry.npmjs.org/fsevents/-/fsevents-2.3.3.tgz",
+      "integrity": "sha512-5xoDfX+fL7faATnagmWPpbFtwh/R77WmMMqqHGS65C3vvB0YHrgF+B1YmZ3441tMj5n63k0212XNoJwzlhffQw==",
+      "dev": true,
+      "hasInstallScript": true,
+      "license": "MIT",
+      "optional": true,
+      "os": [
+        "darwin"
+      ],
+      "engines": {
+        "node": "^8.16.0 || ^10.6.0 || >=11.0.0"
+      }
+    },
+    "node_modules/get-tsconfig": {
+      "version": "4.13.6",
+      "resolved": "https://registry.npmjs.org/get-tsconfig/-/get-tsconfig-4.13.6.tgz",
+      "integrity": "sha512-shZT/QMiSHc/YBLxxOkMtgSid5HFoauqCE3/exfsEcwg1WkeqjG+V40yBbBrsD+jW2HDXcs28xOfcbm2jI8Ddw==",
+      "dev": true,
+      "license": "MIT",
+      "dependencies": {
+        "resolve-pkg-maps": "^1.0.0"
+      },
+      "funding": {
+        "url": "https://github.com/privatenumber/get-tsconfig?sponsor=1"
+      }
+    },
+    "node_modules/giget": {
+      "version": "2.0.0",
+      "resolved": "https://registry.npmjs.org/giget/-/giget-2.0.0.tgz",
+      "integrity": "sha512-L5bGsVkxJbJgdnwyuheIunkGatUF/zssUoxxjACCseZYAVbaqdh9Tsmmlkl8vYan09H7sbvKt4pS8GqKLBrEzA==",
+      "dev": true,
+      "license": "MIT",
+      "dependencies": {
+        "citty": "^0.1.6",
+        "consola": "^3.4.0",
+        "defu": "^6.1.4",
+        "node-fetch-native": "^1.6.6",
+        "nypm": "^0.6.0",
+        "pathe": "^2.0.3"
+      },
+      "bin": {
+        "giget": "dist/cli.mjs"
+      }
+    },
+    "node_modules/giget/node_modules/citty": {
+      "version": "0.1.6",
+      "resolved": "https://registry.npmjs.org/citty/-/citty-0.1.6.tgz",
+      "integrity": "sha512-tskPPKEs8D2KPafUypv2gxwJP8h/OaJmC82QQGGDQcHvXX43xF2VDACcJVmZ0EuSxkpO9Kc4MlrA3q0+FG58AQ==",
+      "dev": true,
+      "license": "MIT",
+      "dependencies": {
+        "consola": "^3.2.3"
+      }
+    },
+    "node_modules/github-slugger": {
+      "version": "2.0.0",
+      "resolved": "https://registry.npmjs.org/github-slugger/-/github-slugger-2.0.0.tgz",
+      "integrity": "sha512-IaOQ9puYtjrkq7Y0Ygl9KDZnrf/aiUJYUpVf89y8kyaxbRG7Y1SrX/jaumrv81vc61+kiMempujsM3Yw7w5qcw==",
+      "dev": true,
+      "license": "ISC"
+    },
+    "node_modules/glob-parent": {
+      "version": "6.0.2",
+      "resolved": "https://registry.npmjs.org/glob-parent/-/glob-parent-6.0.2.tgz",
+      "integrity": "sha512-XxwI8EOhVQgWp6iDL+3b0r86f4d6AX6zSU55HfB4ydCEuXLXc5FcYeOu+nnGftS4TEju/11rt4KJPTMgbfmv4A==",
+      "dev": true,
+      "license": "ISC",
+      "dependencies": {
+        "is-glob": "^4.0.3"
+      },
+      "engines": {
+        "node": ">=10.13.0"
+      }
+    },
+    "node_modules/globals": {
+      "version": "14.0.0",
+      "resolved": "https://registry.npmjs.org/globals/-/globals-14.0.0.tgz",
+      "integrity": "sha512-oahGvuMGQlPw/ivIYBjVSrWAfWLBeku5tpPE2fOPLi+WHffIWbuh2tCjhyQhTBPMf5E9jDEH4FOmTYgYwbKwtQ==",
+      "dev": true,
+      "license": "MIT",
+      "engines": {
+        "node": ">=18"
+      },
+      "funding": {
+        "url": "https://github.com/sponsors/sindresorhus"
+      }
+    },
+    "node_modules/has-flag": {
+      "version": "4.0.0",
+      "resolved": "https://registry.npmjs.org/has-flag/-/has-flag-4.0.0.tgz",
+      "integrity": "sha512-EykJT/Q1KjTWctppgIAgfSO0tKVuZUjhgMr17kqTumMl6Afv3EISleU7qZUzoXDFTAHTDC4NOoG/ZxU3EvlMPQ==",
+      "dev": true,
+      "license": "MIT",
+      "engines": {
+        "node": ">=8"
+      }
+    },
+    "node_modules/html-escaper": {
+      "version": "2.0.2",
+      "resolved": "https://registry.npmjs.org/html-escaper/-/html-escaper-2.0.2.tgz",
+      "integrity": "sha512-H2iMtd0I4Mt5eYiapRdIDjp+XzelXQ0tFE4JS7YFwFevXXMmOp9myNrUvCg0D6ws8iqkRPBfKHgbwig1SmlLfg==",
+      "dev": true,
+      "license": "MIT"
+    },
+    "node_modules/ignore": {
+      "version": "5.3.2",
+      "resolved": "https://registry.npmjs.org/ignore/-/ignore-5.3.2.tgz",
+      "integrity": "sha512-hsBTNUqQTDwkWtcdYI2i06Y/nUBEsNEDJKjWdigLvegy8kDuJAS8uRlpkkcQpyEXL0Z/pjDy5HBmMjRCJ2gq+g==",
+      "dev": true,
+      "license": "MIT",
+      "engines": {
+        "node": ">= 4"
+      }
+    },
+    "node_modules/import-fresh": {
+      "version": "3.3.1",
+      "resolved": "https://registry.npmjs.org/import-fresh/-/import-fresh-3.3.1.tgz",
+      "integrity": "sha512-TR3KfrTZTYLPB6jUjfx6MF9WcWrHL9su5TObK4ZkYgBdWKPOFoSoQIdEuTuR82pmtxH2spWG9h6etwfr1pLBqQ==",
+      "dev": true,
+      "license": "MIT",
+      "dependencies": {
+        "parent-module": "^1.0.0",
+        "resolve-from": "^4.0.0"
+      },
+      "engines": {
+        "node": ">=6"
+      },
+      "funding": {
+        "url": "https://github.com/sponsors/sindresorhus"
+      }
+    },
+    "node_modules/imurmurhash": {
+      "version": "0.1.4",
+      "resolved": "https://registry.npmjs.org/imurmurhash/-/imurmurhash-0.1.4.tgz",
+      "integrity": "sha512-JmXMZ6wuvDmLiHEml9ykzqO6lwFbof0GG4IkcGaENdCRDDmMVnny7s5HsIgHCbaq0w2MyPhDqkhTUgS2LU2PHA==",
+      "dev": true,
+      "license": "MIT",
+      "engines": {
+        "node": ">=0.8.19"
+      }
+    },
+    "node_modules/indent-string": {
+      "version": "5.0.0",
+      "resolved": "https://registry.npmjs.org/indent-string/-/indent-string-5.0.0.tgz",
+      "integrity": "sha512-m6FAo/spmsW2Ab2fU35JTYwtOKa2yAwXSwgjSv1TJzh4Mh7mC3lzAOVLBprb72XsTrgkEIsl7YrFNAiDiRhIGg==",
+      "dev": true,
+      "license": "MIT",
+      "engines": {
+        "node": ">=12"
+      },
+      "funding": {
+        "url": "https://github.com/sponsors/sindresorhus"
+      }
+    },
+    "node_modules/is-builtin-module": {
+      "version": "5.0.0",
+      "resolved": "https://registry.npmjs.org/is-builtin-module/-/is-builtin-module-5.0.0.tgz",
+      "integrity": "sha512-f4RqJKBUe5rQkJ2eJEJBXSticB3hGbN9j0yxxMQFqIW89Jp9WYFtzfTcRlstDKVUTRzSOTLKRfO9vIztenwtxA==",
+      "dev": true,
+      "license": "MIT",
+      "dependencies": {
+        "builtin-modules": "^5.0.0"
+      },
+      "engines": {
+        "node": ">=18.20"
+      },
+      "funding": {
+        "url": "https://github.com/sponsors/sindresorhus"
+      }
+    },
+    "node_modules/is-docker": {
+      "version": "3.0.0",
+      "resolved": "https://registry.npmjs.org/is-docker/-/is-docker-3.0.0.tgz",
+      "integrity": "sha512-eljcgEDlEns/7AXFosB5K/2nCM4P7FQPkGc/DWLy5rmFEWvZayGrik1d9/QIY5nJ4f9YsVvBkA6kJpHn9rISdQ==",
+      "dev": true,
+      "license": "MIT",
+      "bin": {
+        "is-docker": "cli.js"
+      },
+      "engines": {
+        "node": "^12.20.0 || ^14.13.1 || >=16.0.0"
+      },
+      "funding": {
+        "url": "https://github.com/sponsors/sindresorhus"
+      }
+    },
+    "node_modules/is-extglob": {
+      "version": "2.1.1",
+      "resolved": "https://registry.npmjs.org/is-extglob/-/is-extglob-2.1.1.tgz",
+      "integrity": "sha512-SbKbANkN603Vi4jEZv49LeVJMn4yGwsbzZworEoyEiutsN3nJYdbO36zfhGJ6QEDpOZIFkDtnq5JRxmvl3jsoQ==",
+      "dev": true,
+      "license": "MIT",
+      "engines": {
+        "node": ">=0.10.0"
+      }
+    },
+    "node_modules/is-glob": {
+      "version": "4.0.3",
+      "resolved": "https://registry.npmjs.org/is-glob/-/is-glob-4.0.3.tgz",
+      "integrity": "sha512-xelSayHH36ZgE7ZWhli7pW34hNbNl8Ojv5KVmkJD4hBdD3th8Tfk9vYasLM+mXWOZhFkgZfxhLSnrwRr4elSSg==",
+      "dev": true,
+      "license": "MIT",
+      "dependencies": {
+        "is-extglob": "^2.1.1"
+      },
+      "engines": {
+        "node": ">=0.10.0"
+      }
+    },
+    "node_modules/is-inside-container": {
+      "version": "1.0.0",
+      "resolved": "https://registry.npmjs.org/is-inside-container/-/is-inside-container-1.0.0.tgz",
+      "integrity": "sha512-KIYLCCJghfHZxqjYBE7rEy0OBuTd5xCHS7tHVgvCLkx7StIoaxwNW3hCALgEUjFfeRk+MG/Qxmp/vtETEF3tRA==",
+      "dev": true,
+      "license": "MIT",
+      "dependencies": {
+        "is-docker": "^3.0.0"
+      },
+      "bin": {
+        "is-inside-container": "cli.js"
+      },
+      "engines": {
+        "node": ">=14.16"
+      },
+      "funding": {
+        "url": "https://github.com/sponsors/sindresorhus"
+      }
+    },
+    "node_modules/is-wsl": {
+      "version": "3.1.1",
+      "resolved": "https://registry.npmjs.org/is-wsl/-/is-wsl-3.1.1.tgz",
+      "integrity": "sha512-e6rvdUCiQCAuumZslxRJWR/Doq4VpPR82kqclvcS0efgt430SlGIk05vdCN58+VrzgtIcfNODjozVielycD4Sw==",
+      "dev": true,
+      "license": "MIT",
+      "dependencies": {
+        "is-inside-container": "^1.0.0"
+      },
+      "engines": {
+        "node": ">=16"
+      },
+      "funding": {
+        "url": "https://github.com/sponsors/sindresorhus"
+      }
+    },
+    "node_modules/isexe": {
+      "version": "2.0.0",
+      "resolved": "https://registry.npmjs.org/isexe/-/isexe-2.0.0.tgz",
+      "integrity": "sha512-RHxMLp9lnKHGHRng9QFhRCMbYAcVpn69smSGcq3f36xjgVVWThj4qqLbTLlq7Ssj8B+fIQ1EuCEGI2lKsyQeIw==",
+      "dev": true,
+      "license": "ISC"
+    },
+    "node_modules/istanbul-lib-coverage": {
+      "version": "3.2.2",
+      "resolved": "https://registry.npmjs.org/istanbul-lib-coverage/-/istanbul-lib-coverage-3.2.2.tgz",
+      "integrity": "sha512-O8dpsF+r0WV/8MNRKfnmrtCWhuKjxrq2w+jpzBL5UZKTi2LeVWnWOmWRxFlesJONmc+wLAGvKQZEOanko0LFTg==",
+      "dev": true,
+      "license": "BSD-3-Clause",
+      "engines": {
+        "node": ">=8"
+      }
+    },
+    "node_modules/istanbul-lib-report": {
+      "version": "3.0.1",
+      "resolved": "https://registry.npmjs.org/istanbul-lib-report/-/istanbul-lib-report-3.0.1.tgz",
+      "integrity": "sha512-GCfE1mtsHGOELCU8e/Z7YWzpmybrx/+dSTfLrvY8qRmaY6zXTKWn6WQIjaAFw069icm6GVMNkgu0NzI4iPZUNw==",
+      "dev": true,
+      "license": "BSD-3-Clause",
+      "dependencies": {
+        "istanbul-lib-coverage": "^3.0.0",
+        "make-dir": "^4.0.0",
+        "supports-color": "^7.1.0"
+      },
+      "engines": {
+        "node": ">=10"
+      }
+    },
+    "node_modules/istanbul-reports": {
+      "version": "3.2.0",
+      "resolved": "https://registry.npmjs.org/istanbul-reports/-/istanbul-reports-3.2.0.tgz",
+      "integrity": "sha512-HGYWWS/ehqTV3xN10i23tkPkpH46MLCIMFNCaaKNavAXTF1RkqxawEPtnjnGZ6XKSInBKkiOA5BKS+aZiY3AvA==",
+      "dev": true,
+      "license": "BSD-3-Clause",
+      "dependencies": {
+        "html-escaper": "^2.0.0",
+        "istanbul-lib-report": "^3.0.0"
+      },
+      "engines": {
+        "node": ">=8"
+      }
+    },
+    "node_modules/jiti": {
+      "version": "2.6.1",
+      "resolved": "https://registry.npmjs.org/jiti/-/jiti-2.6.1.tgz",
+      "integrity": "sha512-ekilCSN1jwRvIbgeg/57YFh8qQDNbwDb9xT/qu2DAHbFFZUicIl4ygVaAvzveMhMVr3LnpSKTNnwt8PoOfmKhQ==",
+      "dev": true,
+      "license": "MIT",
+      "bin": {
+        "jiti": "lib/jiti-cli.mjs"
+      }
+    },
+    "node_modules/js-tokens": {
+      "version": "10.0.0",
+      "resolved": "https://registry.npmjs.org/js-tokens/-/js-tokens-10.0.0.tgz",
+      "integrity": "sha512-lM/UBzQmfJRo9ABXbPWemivdCW8V2G8FHaHdypQaIy523snUjog0W71ayWXTjiR+ixeMyVHN2XcpnTd/liPg/Q==",
+      "dev": true,
+      "license": "MIT"
+    },
+    "node_modules/js-yaml": {
+      "version": "4.1.1",
+      "resolved": "https://registry.npmjs.org/js-yaml/-/js-yaml-4.1.1.tgz",
+      "integrity": "sha512-qQKT4zQxXl8lLwBtHMWwaTcGfFOZviOJet3Oy/xmGk2gZH677CJM9EvtfdSkgWcATZhj/55JZ0rmy3myCT5lsA==",
+      "dev": true,
+      "license": "MIT",
+      "dependencies": {
+        "argparse": "^2.0.1"
+      },
+      "bin": {
+        "js-yaml": "bin/js-yaml.js"
+      }
+    },
+    "node_modules/jsesc": {
+      "version": "3.1.0",
+      "resolved": "https://registry.npmjs.org/jsesc/-/jsesc-3.1.0.tgz",
+      "integrity": "sha512-/sM3dO2FOzXjKQhJuo0Q173wf2KOo8t4I8vHy6lF9poUp7bKT0/NHE8fPX23PwfhnykfqnC2xRxOnVw5XuGIaA==",
+      "dev": true,
+      "license": "MIT",
+      "bin": {
+        "jsesc": "bin/jsesc"
+      },
+      "engines": {
+        "node": ">=6"
+      }
+    },
+    "node_modules/json-buffer": {
+      "version": "3.0.1",
+      "resolved": "https://registry.npmjs.org/json-buffer/-/json-buffer-3.0.1.tgz",
+      "integrity": "sha512-4bV5BfR2mqfQTJm+V5tPPdf+ZpuhiIvTuAB5g8kcrXOZpTT/QwwVRWBywX1ozr6lEuPdbHxwaJlm9G6mI2sfSQ==",
+      "dev": true,
+      "license": "MIT"
+    },
+    "node_modules/json-schema-traverse": {
+      "version": "0.4.1",
+      "resolved": "https://registry.npmjs.org/json-schema-traverse/-/json-schema-traverse-0.4.1.tgz",
+      "integrity": "sha512-xbbCH5dCYU5T8LcEhhuh7HJ88HXuW3qsI3Y0zOZFKfZEHcpWiHU/Jxzk629Brsab/mMiHQti9wMP+845RPe3Vg==",
+      "dev": true,
+      "license": "MIT"
+    },
+    "node_modules/json-stable-stringify-without-jsonify": {
+      "version": "1.0.1",
+      "resolved": "https://registry.npmjs.org/json-stable-stringify-without-jsonify/-/json-stable-stringify-without-jsonify-1.0.1.tgz",
+      "integrity": "sha512-Bdboy+l7tA3OGW6FjyFHWkP5LuByj1Tk33Ljyq0axyzdk9//JSi2u3fP1QSmd1KNwq6VOKYGlAu87CisVir6Pw==",
+      "dev": true,
+      "license": "MIT"
+    },
+    "node_modules/keyv": {
+      "version": "4.5.4",
+      "resolved": "https://registry.npmjs.org/keyv/-/keyv-4.5.4.tgz",
+      "integrity": "sha512-oxVHkHR/EJf2CNXnWxRLW6mg7JyCCUcG0DtEGmL2ctUo1PNTin1PUil+r/+4r5MpVgC/fn1kjsx7mjSujKqIpw==",
+      "dev": true,
+      "license": "MIT",
+      "dependencies": {
+        "json-buffer": "3.0.1"
+      }
+    },
+    "node_modules/knitwork": {
+      "version": "1.3.0",
+      "resolved": "https://registry.npmjs.org/knitwork/-/knitwork-1.3.0.tgz",
+      "integrity": "sha512-4LqMNoONzR43B1W0ek0fhXMsDNW/zxa1NdFAVMY+k28pgZLovR4G3PB5MrpTxCy1QaZCqNoiaKPr5w5qZHfSNw==",
+      "dev": true,
+      "license": "MIT"
+    },
+    "node_modules/levn": {
+      "version": "0.4.1",
+      "resolved": "https://registry.npmjs.org/levn/-/levn-0.4.1.tgz",
+      "integrity": "sha512-+bT2uH4E5LGE7h/n3evcS/sQlJXCpIp6ym8OWJ5eV6+67Dsql/LaaT7qJBAt2rzfoa/5QBGBhxDix1dMt2kQKQ==",
+      "dev": true,
+      "license": "MIT",
+      "dependencies": {
+        "prelude-ls": "^1.2.1",
+        "type-check": "~0.4.0"
+      },
+      "engines": {
+        "node": ">= 0.8.0"
+      }
+    },
+    "node_modules/locate-path": {
+      "version": "6.0.0",
+      "resolved": "https://registry.npmjs.org/locate-path/-/locate-path-6.0.0.tgz",
+      "integrity": "sha512-iPZK6eYjbxRu3uB4/WZ3EsEIMJFMqAoopl3R+zuq0UjcAm/MO6KCweDgPfP3elTztoKP3KtnVHxTn2NHBSDVUw==",
+      "dev": true,
+      "license": "MIT",
+      "dependencies": {
+        "p-locate": "^5.0.0"
+      },
+      "engines": {
+        "node": ">=10"
+      },
+      "funding": {
+        "url": "https://github.com/sponsors/sindresorhus"
+      }
+    },
+    "node_modules/lodash": {
+      "version": "4.17.23",
+      "resolved": "https://registry.npmjs.org/lodash/-/lodash-4.17.23.tgz",
+      "integrity": "sha512-LgVTMpQtIopCi79SJeDiP0TfWi5CNEc/L/aRdTh3yIvmZXTnheWpKjSZhnvMl8iXbC1tFg9gdHHDMLoV7CnG+w==",
+      "dev": true,
+      "license": "MIT"
+    },
+    "node_modules/lodash.deburr": {
+      "version": "4.1.0",
+      "resolved": "https://registry.npmjs.org/lodash.deburr/-/lodash.deburr-4.1.0.tgz",
+      "integrity": "sha512-m/M1U1f3ddMCs6Hq2tAsYThTBDaAKFDX3dwDo97GEYzamXi9SqUpjWi/Rrj/gf3X2n8ktwgZrlP1z6E3v/IExQ==",
+      "dev": true,
+      "license": "MIT"
+    },
+    "node_modules/lodash.merge": {
+      "version": "4.6.2",
+      "resolved": "https://registry.npmjs.org/lodash.merge/-/lodash.merge-4.6.2.tgz",
+      "integrity": "sha512-0KpjqXRVvrYyCsX1swR/XTK0va6VQkQM6MNo7PqW77ByjAhoARA8EfrP1N4+KlKj8YS0ZUCtRT/YUuhyYDujIQ==",
+      "dev": true,
+      "license": "MIT"
+    },
+    "node_modules/longest-streak": {
+      "version": "3.1.0",
+      "resolved": "https://registry.npmjs.org/longest-streak/-/longest-streak-3.1.0.tgz",
+      "integrity": "sha512-9Ri+o0JYgehTaVBBDoMqIl8GXtbWg711O3srftcHhZ0dqnETqLaoIK0x17fUw9rFSlK/0NlsKe0Ahhyl5pXE2g==",
+      "dev": true,
+      "license": "MIT",
+      "funding": {
+        "type": "github",
+        "url": "https://github.com/sponsors/wooorm"
+      }
+    },
+    "node_modules/magic-string": {
+      "version": "0.30.21",
+      "resolved": "https://registry.npmjs.org/magic-string/-/magic-string-0.30.21.tgz",
+      "integrity": "sha512-vd2F4YUyEXKGcLHoq+TEyCjxueSeHnFxyyjNp80yg0XV4vUhnDer/lvvlqM/arB5bXQN5K2/3oinyCRyx8T2CQ==",
+      "dev": true,
+      "license": "MIT",
+      "dependencies": {
+        "@jridgewell/sourcemap-codec": "^1.5.5"
+      }
+    },
+    "node_modules/magicast": {
+      "version": "0.5.2",
+      "resolved": "https://registry.npmjs.org/magicast/-/magicast-0.5.2.tgz",
+      "integrity": "sha512-E3ZJh4J3S9KfwdjZhe2afj6R9lGIN5Pher1pF39UGrXRqq/VDaGVIGN13BjHd2u8B61hArAGOnso7nBOouW3TQ==",
+      "dev": true,
+      "license": "MIT",
+      "dependencies": {
+        "@babel/parser": "^7.29.0",
+        "@babel/types": "^7.29.0",
+        "source-map-js": "^1.2.1"
+      }
+    },
+    "node_modules/make-dir": {
+      "version": "4.0.0",
+      "resolved": "https://registry.npmjs.org/make-dir/-/make-dir-4.0.0.tgz",
+      "integrity": "sha512-hXdUTZYIVOt1Ex//jAQi+wTZZpUpwBj/0QsOzqegb3rGMMeJiSEu5xLHnYfBrRV4RH2+OCSOO95Is/7x1WJ4bw==",
+      "dev": true,
+      "license": "MIT",
+      "dependencies": {
+        "semver": "^7.5.3"
+      },
+      "engines": {
+        "node": ">=10"
+      },
+      "funding": {
+        "url": "https://github.com/sponsors/sindresorhus"
+      }
+    },
+    "node_modules/markdown-table": {
+      "version": "3.0.4",
+      "resolved": "https://registry.npmjs.org/markdown-table/-/markdown-table-3.0.4.tgz",
+      "integrity": "sha512-wiYz4+JrLyb/DqW2hkFJxP7Vd7JuTDm77fvbM8VfEQdmSMqcImWeeRbHwZjBjIFki/VaMK2BhFi7oUUZeM5bqw==",
+      "dev": true,
+      "license": "MIT",
+      "funding": {
+        "type": "github",
+        "url": "https://github.com/sponsors/wooorm"
+      }
+    },
+    "node_modules/md4w": {
+      "version": "0.2.7",
+      "resolved": "https://registry.npmjs.org/md4w/-/md4w-0.2.7.tgz",
+      "integrity": "sha512-lFM7vwk3d4MzkV2mija7aPkK6OjKXZDQsH2beX+e2cvccBoqc6RraheMtAO0Wcr/gjj5L+WS5zhb+06AmuGZrg==",
+      "dev": true,
+      "license": "MIT"
+    },
+    "node_modules/mdast-util-find-and-replace": {
+      "version": "3.0.2",
+      "resolved": "https://registry.npmjs.org/mdast-util-find-and-replace/-/mdast-util-find-and-replace-3.0.2.tgz",
+      "integrity": "sha512-Tmd1Vg/m3Xz43afeNxDIhWRtFZgM2VLyaf4vSTYwudTyeuTneoL3qtWMA5jeLyz/O1vDJmmV4QuScFCA2tBPwg==",
+      "dev": true,
+      "license": "MIT",
+      "dependencies": {
+        "@types/mdast": "^4.0.0",
+        "escape-string-regexp": "^5.0.0",
+        "unist-util-is": "^6.0.0",
+        "unist-util-visit-parents": "^6.0.0"
+      },
+      "funding": {
+        "type": "opencollective",
+        "url": "https://opencollective.com/unified"
+      }
+    },
+    "node_modules/mdast-util-find-and-replace/node_modules/escape-string-regexp": {
+      "version": "5.0.0",
+      "resolved": "https://registry.npmjs.org/escape-string-regexp/-/escape-string-regexp-5.0.0.tgz",
+      "integrity": "sha512-/veY75JbMK4j1yjvuUxuVsiS/hr/4iHs9FTT6cgTexxdE0Ly/glccBAkloH/DofkjRbZU3bnoj38mOmhkZ0lHw==",
+      "dev": true,
+      "license": "MIT",
+      "engines": {
+        "node": ">=12"
+      },
+      "funding": {
+        "url": "https://github.com/sponsors/sindresorhus"
+      }
+    },
+    "node_modules/mdast-util-from-markdown": {
+      "version": "2.0.3",
+      "resolved": "https://registry.npmjs.org/mdast-util-from-markdown/-/mdast-util-from-markdown-2.0.3.tgz",
+      "integrity": "sha512-W4mAWTvSlKvf8L6J+VN9yLSqQ9AOAAvHuoDAmPkz4dHf553m5gVj2ejadHJhoJmcmxEnOv6Pa8XJhpxE93kb8Q==",
+      "dev": true,
+      "license": "MIT",
+      "dependencies": {
+        "@types/mdast": "^4.0.0",
+        "@types/unist": "^3.0.0",
+        "decode-named-character-reference": "^1.0.0",
+        "devlop": "^1.0.0",
+        "mdast-util-to-string": "^4.0.0",
+        "micromark": "^4.0.0",
+        "micromark-util-decode-numeric-character-reference": "^2.0.0",
+        "micromark-util-decode-string": "^2.0.0",
+        "micromark-util-normalize-identifier": "^2.0.0",
+        "micromark-util-symbol": "^2.0.0",
+        "micromark-util-types": "^2.0.0",
+        "unist-util-stringify-position": "^4.0.0"
+      },
+      "funding": {
+        "type": "opencollective",
+        "url": "https://opencollective.com/unified"
+      }
+    },
+    "node_modules/mdast-util-frontmatter": {
+      "version": "2.0.1",
+      "resolved": "https://registry.npmjs.org/mdast-util-frontmatter/-/mdast-util-frontmatter-2.0.1.tgz",
+      "integrity": "sha512-LRqI9+wdgC25P0URIJY9vwocIzCcksduHQ9OF2joxQoyTNVduwLAFUzjoopuRJbJAReaKrNQKAZKL3uCMugWJA==",
+      "dev": true,
+      "license": "MIT",
+      "dependencies": {
+        "@types/mdast": "^4.0.0",
+        "devlop": "^1.0.0",
+        "escape-string-regexp": "^5.0.0",
+        "mdast-util-from-markdown": "^2.0.0",
+        "mdast-util-to-markdown": "^2.0.0",
+        "micromark-extension-frontmatter": "^2.0.0"
+      },
+      "funding": {
+        "type": "opencollective",
+        "url": "https://opencollective.com/unified"
+      }
+    },
+    "node_modules/mdast-util-frontmatter/node_modules/escape-string-regexp": {
+      "version": "5.0.0",
+      "resolved": "https://registry.npmjs.org/escape-string-regexp/-/escape-string-regexp-5.0.0.tgz",
+      "integrity": "sha512-/veY75JbMK4j1yjvuUxuVsiS/hr/4iHs9FTT6cgTexxdE0Ly/glccBAkloH/DofkjRbZU3bnoj38mOmhkZ0lHw==",
+      "dev": true,
+      "license": "MIT",
+      "engines": {
+        "node": ">=12"
+      },
+      "funding": {
+        "url": "https://github.com/sponsors/sindresorhus"
+      }
+    },
+    "node_modules/mdast-util-gfm": {
+      "version": "3.1.0",
+      "resolved": "https://registry.npmjs.org/mdast-util-gfm/-/mdast-util-gfm-3.1.0.tgz",
+      "integrity": "sha512-0ulfdQOM3ysHhCJ1p06l0b0VKlhU0wuQs3thxZQagjcjPrlFRqY215uZGHHJan9GEAXd9MbfPjFJz+qMkVR6zQ==",
+      "dev": true,
+      "license": "MIT",
+      "dependencies": {
+        "mdast-util-from-markdown": "^2.0.0",
+        "mdast-util-gfm-autolink-literal": "^2.0.0",
+        "mdast-util-gfm-footnote": "^2.0.0",
+        "mdast-util-gfm-strikethrough": "^2.0.0",
+        "mdast-util-gfm-table": "^2.0.0",
+        "mdast-util-gfm-task-list-item": "^2.0.0",
+        "mdast-util-to-markdown": "^2.0.0"
+      },
+      "funding": {
+        "type": "opencollective",
+        "url": "https://opencollective.com/unified"
+      }
+    },
+    "node_modules/mdast-util-gfm-autolink-literal": {
+      "version": "2.0.1",
+      "resolved": "https://registry.npmjs.org/mdast-util-gfm-autolink-literal/-/mdast-util-gfm-autolink-literal-2.0.1.tgz",
+      "integrity": "sha512-5HVP2MKaP6L+G6YaxPNjuL0BPrq9orG3TsrZ9YXbA3vDw/ACI4MEsnoDpn6ZNm7GnZgtAcONJyPhOP8tNJQavQ==",
+      "dev": true,
+      "license": "MIT",
+      "dependencies": {
+        "@types/mdast": "^4.0.0",
+        "ccount": "^2.0.0",
+        "devlop": "^1.0.0",
+        "mdast-util-find-and-replace": "^3.0.0",
+        "micromark-util-character": "^2.0.0"
+      },
+      "funding": {
+        "type": "opencollective",
+        "url": "https://opencollective.com/unified"
+      }
+    },
+    "node_modules/mdast-util-gfm-footnote": {
+      "version": "2.1.0",
+      "resolved": "https://registry.npmjs.org/mdast-util-gfm-footnote/-/mdast-util-gfm-footnote-2.1.0.tgz",
+      "integrity": "sha512-sqpDWlsHn7Ac9GNZQMeUzPQSMzR6Wv0WKRNvQRg0KqHh02fpTz69Qc1QSseNX29bhz1ROIyNyxExfawVKTm1GQ==",
+      "dev": true,
+      "license": "MIT",
+      "dependencies": {
+        "@types/mdast": "^4.0.0",
+        "devlop": "^1.1.0",
+        "mdast-util-from-markdown": "^2.0.0",
+        "mdast-util-to-markdown": "^2.0.0",
+        "micromark-util-normalize-identifier": "^2.0.0"
+      },
+      "funding": {
+        "type": "opencollective",
+        "url": "https://opencollective.com/unified"
+      }
+    },
+    "node_modules/mdast-util-gfm-strikethrough": {
+      "version": "2.0.0",
+      "resolved": "https://registry.npmjs.org/mdast-util-gfm-strikethrough/-/mdast-util-gfm-strikethrough-2.0.0.tgz",
+      "integrity": "sha512-mKKb915TF+OC5ptj5bJ7WFRPdYtuHv0yTRxK2tJvi+BDqbkiG7h7u/9SI89nRAYcmap2xHQL9D+QG/6wSrTtXg==",
+      "dev": true,
+      "license": "MIT",
+      "dependencies": {
+        "@types/mdast": "^4.0.0",
+        "mdast-util-from-markdown": "^2.0.0",
+        "mdast-util-to-markdown": "^2.0.0"
+      },
+      "funding": {
+        "type": "opencollective",
+        "url": "https://opencollective.com/unified"
+      }
+    },
+    "node_modules/mdast-util-gfm-table": {
+      "version": "2.0.0",
+      "resolved": "https://registry.npmjs.org/mdast-util-gfm-table/-/mdast-util-gfm-table-2.0.0.tgz",
+      "integrity": "sha512-78UEvebzz/rJIxLvE7ZtDd/vIQ0RHv+3Mh5DR96p7cS7HsBhYIICDBCu8csTNWNO6tBWfqXPWekRuj2FNOGOZg==",
+      "dev": true,
+      "license": "MIT",
+      "dependencies": {
+        "@types/mdast": "^4.0.0",
+        "devlop": "^1.0.0",
+        "markdown-table": "^3.0.0",
+        "mdast-util-from-markdown": "^2.0.0",
+        "mdast-util-to-markdown": "^2.0.0"
+      },
+      "funding": {
+        "type": "opencollective",
+        "url": "https://opencollective.com/unified"
+      }
+    },
+    "node_modules/mdast-util-gfm-task-list-item": {
+      "version": "2.0.0",
+      "resolved": "https://registry.npmjs.org/mdast-util-gfm-task-list-item/-/mdast-util-gfm-task-list-item-2.0.0.tgz",
+      "integrity": "sha512-IrtvNvjxC1o06taBAVJznEnkiHxLFTzgonUdy8hzFVeDun0uTjxxrRGVaNFqkU1wJR3RBPEfsxmU6jDWPofrTQ==",
+      "dev": true,
+      "license": "MIT",
+      "dependencies": {
+        "@types/mdast": "^4.0.0",
+        "devlop": "^1.0.0",
+        "mdast-util-from-markdown": "^2.0.0",
+        "mdast-util-to-markdown": "^2.0.0"
+      },
+      "funding": {
+        "type": "opencollective",
+        "url": "https://opencollective.com/unified"
+      }
+    },
+    "node_modules/mdast-util-phrasing": {
+      "version": "4.1.0",
+      "resolved": "https://registry.npmjs.org/mdast-util-phrasing/-/mdast-util-phrasing-4.1.0.tgz",
+      "integrity": "sha512-TqICwyvJJpBwvGAMZjj4J2n0X8QWp21b9l0o7eXyVJ25YNWYbJDVIyD1bZXE6WtV6RmKJVYmQAKWa0zWOABz2w==",
+      "dev": true,
+      "license": "MIT",
+      "dependencies": {
+        "@types/mdast": "^4.0.0",
+        "unist-util-is": "^6.0.0"
+      },
+      "funding": {
+        "type": "opencollective",
+        "url": "https://opencollective.com/unified"
+      }
+    },
+    "node_modules/mdast-util-to-markdown": {
+      "version": "2.1.2",
+      "resolved": "https://registry.npmjs.org/mdast-util-to-markdown/-/mdast-util-to-markdown-2.1.2.tgz",
+      "integrity": "sha512-xj68wMTvGXVOKonmog6LwyJKrYXZPvlwabaryTjLh9LuvovB/KAH+kvi8Gjj+7rJjsFi23nkUxRQv1KqSroMqA==",
+      "dev": true,
+      "license": "MIT",
+      "dependencies": {
+        "@types/mdast": "^4.0.0",
+        "@types/unist": "^3.0.0",
+        "longest-streak": "^3.0.0",
+        "mdast-util-phrasing": "^4.0.0",
+        "mdast-util-to-string": "^4.0.0",
+        "micromark-util-classify-character": "^2.0.0",
+        "micromark-util-decode-string": "^2.0.0",
+        "unist-util-visit": "^5.0.0",
+        "zwitch": "^2.0.0"
+      },
+      "funding": {
+        "type": "opencollective",
+        "url": "https://opencollective.com/unified"
+      }
+    },
+    "node_modules/mdast-util-to-string": {
+      "version": "4.0.0",
+      "resolved": "https://registry.npmjs.org/mdast-util-to-string/-/mdast-util-to-string-4.0.0.tgz",
+      "integrity": "sha512-0H44vDimn51F0YwvxSJSm0eCDOJTRlmN0R1yBh4HLj9wiV1Dn0QoXGbvFAWj2hSItVTlCmBF1hqKlIyUBVFLPg==",
+      "dev": true,
+      "license": "MIT",
+      "dependencies": {
+        "@types/mdast": "^4.0.0"
+      },
+      "funding": {
+        "type": "opencollective",
+        "url": "https://opencollective.com/unified"
+      }
+    },
+    "node_modules/mdbox": {
+      "version": "0.1.1",
+      "resolved": "https://registry.npmjs.org/mdbox/-/mdbox-0.1.1.tgz",
+      "integrity": "sha512-jvLISenzbLRPWWamTG3THlhTcMbKWzJQNyTi61AVXhCBOC+gsldNTUfUNH8d3Vay83zGehFw3wZpF3xChzkTIQ==",
+      "dev": true,
+      "license": "MIT",
+      "dependencies": {
+        "md4w": "^0.2.6"
+      }
+    },
+    "node_modules/micromark": {
+      "version": "4.0.2",
+      "resolved": "https://registry.npmjs.org/micromark/-/micromark-4.0.2.tgz",
+      "integrity": "sha512-zpe98Q6kvavpCr1NPVSCMebCKfD7CA2NqZ+rykeNhONIJBpc1tFKt9hucLGwha3jNTNI8lHpctWJWoimVF4PfA==",
+      "dev": true,
+      "funding": [
+        {
+          "type": "GitHub Sponsors",
+          "url": "https://github.com/sponsors/unifiedjs"
+        },
+        {
+          "type": "OpenCollective",
+          "url": "https://opencollective.com/unified"
+        }
+      ],
+      "license": "MIT",
+      "dependencies": {
+        "@types/debug": "^4.0.0",
+        "debug": "^4.0.0",
+        "decode-named-character-reference": "^1.0.0",
+        "devlop": "^1.0.0",
+        "micromark-core-commonmark": "^2.0.0",
+        "micromark-factory-space": "^2.0.0",
+        "micromark-util-character": "^2.0.0",
+        "micromark-util-chunked": "^2.0.0",
+        "micromark-util-combine-extensions": "^2.0.0",
+        "micromark-util-decode-numeric-character-reference": "^2.0.0",
+        "micromark-util-encode": "^2.0.0",
+        "micromark-util-normalize-identifier": "^2.0.0",
+        "micromark-util-resolve-all": "^2.0.0",
+        "micromark-util-sanitize-uri": "^2.0.0",
+        "micromark-util-subtokenize": "^2.0.0",
+        "micromark-util-symbol": "^2.0.0",
+        "micromark-util-types": "^2.0.0"
+      }
+    },
+    "node_modules/micromark-core-commonmark": {
+      "version": "2.0.3",
+      "resolved": "https://registry.npmjs.org/micromark-core-commonmark/-/micromark-core-commonmark-2.0.3.tgz",
+      "integrity": "sha512-RDBrHEMSxVFLg6xvnXmb1Ayr2WzLAWjeSATAoxwKYJV94TeNavgoIdA0a9ytzDSVzBy2YKFK+emCPOEibLeCrg==",
+      "dev": true,
+      "funding": [
+        {
+          "type": "GitHub Sponsors",
+          "url": "https://github.com/sponsors/unifiedjs"
+        },
+        {
+          "type": "OpenCollective",
+          "url": "https://opencollective.com/unified"
+        }
+      ],
+      "license": "MIT",
+      "dependencies": {
+        "decode-named-character-reference": "^1.0.0",
+        "devlop": "^1.0.0",
+        "micromark-factory-destination": "^2.0.0",
+        "micromark-factory-label": "^2.0.0",
+        "micromark-factory-space": "^2.0.0",
+        "micromark-factory-title": "^2.0.0",
+        "micromark-factory-whitespace": "^2.0.0",
+        "micromark-util-character": "^2.0.0",
+        "micromark-util-chunked": "^2.0.0",
+        "micromark-util-classify-character": "^2.0.0",
+        "micromark-util-html-tag-name": "^2.0.0",
+        "micromark-util-normalize-identifier": "^2.0.0",
+        "micromark-util-resolve-all": "^2.0.0",
+        "micromark-util-subtokenize": "^2.0.0",
+        "micromark-util-symbol": "^2.0.0",
+        "micromark-util-types": "^2.0.0"
+      }
+    },
+    "node_modules/micromark-extension-frontmatter": {
+      "version": "2.0.0",
+      "resolved": "https://registry.npmjs.org/micromark-extension-frontmatter/-/micromark-extension-frontmatter-2.0.0.tgz",
+      "integrity": "sha512-C4AkuM3dA58cgZha7zVnuVxBhDsbttIMiytjgsM2XbHAB2faRVaHRle40558FBN+DJcrLNCoqG5mlrpdU4cRtg==",
+      "dev": true,
+      "license": "MIT",
+      "dependencies": {
+        "fault": "^2.0.0",
+        "micromark-util-character": "^2.0.0",
+        "micromark-util-symbol": "^2.0.0",
+        "micromark-util-types": "^2.0.0"
+      },
+      "funding": {
+        "type": "opencollective",
+        "url": "https://opencollective.com/unified"
+      }
+    },
+    "node_modules/micromark-extension-gfm": {
+      "version": "3.0.0",
+      "resolved": "https://registry.npmjs.org/micromark-extension-gfm/-/micromark-extension-gfm-3.0.0.tgz",
+      "integrity": "sha512-vsKArQsicm7t0z2GugkCKtZehqUm31oeGBV/KVSorWSy8ZlNAv7ytjFhvaryUiCUJYqs+NoE6AFhpQvBTM6Q4w==",
+      "dev": true,
+      "license": "MIT",
+      "dependencies": {
+        "micromark-extension-gfm-autolink-literal": "^2.0.0",
+        "micromark-extension-gfm-footnote": "^2.0.0",
+        "micromark-extension-gfm-strikethrough": "^2.0.0",
+        "micromark-extension-gfm-table": "^2.0.0",
+        "micromark-extension-gfm-tagfilter": "^2.0.0",
+        "micromark-extension-gfm-task-list-item": "^2.0.0",
+        "micromark-util-combine-extensions": "^2.0.0",
+        "micromark-util-types": "^2.0.0"
+      },
+      "funding": {
+        "type": "opencollective",
+        "url": "https://opencollective.com/unified"
+      }
+    },
+    "node_modules/micromark-extension-gfm-autolink-literal": {
+      "version": "2.1.0",
+      "resolved": "https://registry.npmjs.org/micromark-extension-gfm-autolink-literal/-/micromark-extension-gfm-autolink-literal-2.1.0.tgz",
+      "integrity": "sha512-oOg7knzhicgQ3t4QCjCWgTmfNhvQbDDnJeVu9v81r7NltNCVmhPy1fJRX27pISafdjL+SVc4d3l48Gb6pbRypw==",
+      "dev": true,
+      "license": "MIT",
+      "dependencies": {
+        "micromark-util-character": "^2.0.0",
+        "micromark-util-sanitize-uri": "^2.0.0",
+        "micromark-util-symbol": "^2.0.0",
+        "micromark-util-types": "^2.0.0"
+      },
+      "funding": {
+        "type": "opencollective",
+        "url": "https://opencollective.com/unified"
+      }
+    },
+    "node_modules/micromark-extension-gfm-footnote": {
+      "version": "2.1.0",
+      "resolved": "https://registry.npmjs.org/micromark-extension-gfm-footnote/-/micromark-extension-gfm-footnote-2.1.0.tgz",
+      "integrity": "sha512-/yPhxI1ntnDNsiHtzLKYnE3vf9JZ6cAisqVDauhp4CEHxlb4uoOTxOCJ+9s51bIB8U1N1FJ1RXOKTIlD5B/gqw==",
+      "dev": true,
+      "license": "MIT",
+      "dependencies": {
+        "devlop": "^1.0.0",
+        "micromark-core-commonmark": "^2.0.0",
+        "micromark-factory-space": "^2.0.0",
+        "micromark-util-character": "^2.0.0",
+        "micromark-util-normalize-identifier": "^2.0.0",
+        "micromark-util-sanitize-uri": "^2.0.0",
+        "micromark-util-symbol": "^2.0.0",
+        "micromark-util-types": "^2.0.0"
+      },
+      "funding": {
+        "type": "opencollective",
+        "url": "https://opencollective.com/unified"
+      }
+    },
+    "node_modules/micromark-extension-gfm-strikethrough": {
+      "version": "2.1.0",
+      "resolved": "https://registry.npmjs.org/micromark-extension-gfm-strikethrough/-/micromark-extension-gfm-strikethrough-2.1.0.tgz",
+      "integrity": "sha512-ADVjpOOkjz1hhkZLlBiYA9cR2Anf8F4HqZUO6e5eDcPQd0Txw5fxLzzxnEkSkfnD0wziSGiv7sYhk/ktvbf1uw==",
+      "dev": true,
+      "license": "MIT",
+      "dependencies": {
+        "devlop": "^1.0.0",
+        "micromark-util-chunked": "^2.0.0",
+        "micromark-util-classify-character": "^2.0.0",
+        "micromark-util-resolve-all": "^2.0.0",
+        "micromark-util-symbol": "^2.0.0",
+        "micromark-util-types": "^2.0.0"
+      },
+      "funding": {
+        "type": "opencollective",
+        "url": "https://opencollective.com/unified"
+      }
+    },
+    "node_modules/micromark-extension-gfm-table": {
+      "version": "2.1.1",
+      "resolved": "https://registry.npmjs.org/micromark-extension-gfm-table/-/micromark-extension-gfm-table-2.1.1.tgz",
+      "integrity": "sha512-t2OU/dXXioARrC6yWfJ4hqB7rct14e8f7m0cbI5hUmDyyIlwv5vEtooptH8INkbLzOatzKuVbQmAYcbWoyz6Dg==",
+      "dev": true,
+      "license": "MIT",
+      "dependencies": {
+        "devlop": "^1.0.0",
+        "micromark-factory-space": "^2.0.0",
+        "micromark-util-character": "^2.0.0",
+        "micromark-util-symbol": "^2.0.0",
+        "micromark-util-types": "^2.0.0"
+      },
+      "funding": {
+        "type": "opencollective",
+        "url": "https://opencollective.com/unified"
+      }
+    },
+    "node_modules/micromark-extension-gfm-tagfilter": {
+      "version": "2.0.0",
+      "resolved": "https://registry.npmjs.org/micromark-extension-gfm-tagfilter/-/micromark-extension-gfm-tagfilter-2.0.0.tgz",
+      "integrity": "sha512-xHlTOmuCSotIA8TW1mDIM6X2O1SiX5P9IuDtqGonFhEK0qgRI4yeC6vMxEV2dgyr2TiD+2PQ10o+cOhdVAcwfg==",
+      "dev": true,
+      "license": "MIT",
+      "dependencies": {
+        "micromark-util-types": "^2.0.0"
+      },
+      "funding": {
+        "type": "opencollective",
+        "url": "https://opencollective.com/unified"
+      }
+    },
+    "node_modules/micromark-extension-gfm-task-list-item": {
+      "version": "2.1.0",
+      "resolved": "https://registry.npmjs.org/micromark-extension-gfm-task-list-item/-/micromark-extension-gfm-task-list-item-2.1.0.tgz",
+      "integrity": "sha512-qIBZhqxqI6fjLDYFTBIa4eivDMnP+OZqsNwmQ3xNLE4Cxwc+zfQEfbs6tzAo2Hjq+bh6q5F+Z8/cksrLFYWQQw==",
+      "dev": true,
+      "license": "MIT",
+      "dependencies": {
+        "devlop": "^1.0.0",
+        "micromark-factory-space": "^2.0.0",
+        "micromark-util-character": "^2.0.0",
+        "micromark-util-symbol": "^2.0.0",
+        "micromark-util-types": "^2.0.0"
+      },
+      "funding": {
+        "type": "opencollective",
+        "url": "https://opencollective.com/unified"
+      }
+    },
+    "node_modules/micromark-factory-destination": {
+      "version": "2.0.1",
+      "resolved": "https://registry.npmjs.org/micromark-factory-destination/-/micromark-factory-destination-2.0.1.tgz",
+      "integrity": "sha512-Xe6rDdJlkmbFRExpTOmRj9N3MaWmbAgdpSrBQvCFqhezUn4AHqJHbaEnfbVYYiexVSs//tqOdY/DxhjdCiJnIA==",
+      "dev": true,
+      "funding": [
+        {
+          "type": "GitHub Sponsors",
+          "url": "https://github.com/sponsors/unifiedjs"
+        },
+        {
+          "type": "OpenCollective",
+          "url": "https://opencollective.com/unified"
+        }
+      ],
+      "license": "MIT",
+      "dependencies": {
+        "micromark-util-character": "^2.0.0",
+        "micromark-util-symbol": "^2.0.0",
+        "micromark-util-types": "^2.0.0"
+      }
+    },
+    "node_modules/micromark-factory-label": {
+      "version": "2.0.1",
+      "resolved": "https://registry.npmjs.org/micromark-factory-label/-/micromark-factory-label-2.0.1.tgz",
+      "integrity": "sha512-VFMekyQExqIW7xIChcXn4ok29YE3rnuyveW3wZQWWqF4Nv9Wk5rgJ99KzPvHjkmPXF93FXIbBp6YdW3t71/7Vg==",
+      "dev": true,
+      "funding": [
+        {
+          "type": "GitHub Sponsors",
+          "url": "https://github.com/sponsors/unifiedjs"
+        },
+        {
+          "type": "OpenCollective",
+          "url": "https://opencollective.com/unified"
+        }
+      ],
+      "license": "MIT",
+      "dependencies": {
+        "devlop": "^1.0.0",
+        "micromark-util-character": "^2.0.0",
+        "micromark-util-symbol": "^2.0.0",
+        "micromark-util-types": "^2.0.0"
+      }
+    },
+    "node_modules/micromark-factory-space": {
+      "version": "2.0.1",
+      "resolved": "https://registry.npmjs.org/micromark-factory-space/-/micromark-factory-space-2.0.1.tgz",
+      "integrity": "sha512-zRkxjtBxxLd2Sc0d+fbnEunsTj46SWXgXciZmHq0kDYGnck/ZSGj9/wULTV95uoeYiK5hRXP2mJ98Uo4cq/LQg==",
+      "dev": true,
+      "funding": [
+        {
+          "type": "GitHub Sponsors",
+          "url": "https://github.com/sponsors/unifiedjs"
+        },
+        {
+          "type": "OpenCollective",
+          "url": "https://opencollective.com/unified"
+        }
+      ],
+      "license": "MIT",
+      "dependencies": {
+        "micromark-util-character": "^2.0.0",
+        "micromark-util-types": "^2.0.0"
+      }
+    },
+    "node_modules/micromark-factory-title": {
+      "version": "2.0.1",
+      "resolved": "https://registry.npmjs.org/micromark-factory-title/-/micromark-factory-title-2.0.1.tgz",
+      "integrity": "sha512-5bZ+3CjhAd9eChYTHsjy6TGxpOFSKgKKJPJxr293jTbfry2KDoWkhBb6TcPVB4NmzaPhMs1Frm9AZH7OD4Cjzw==",
+      "dev": true,
+      "funding": [
+        {
+          "type": "GitHub Sponsors",
+          "url": "https://github.com/sponsors/unifiedjs"
+        },
+        {
+          "type": "OpenCollective",
+          "url": "https://opencollective.com/unified"
+        }
+      ],
+      "license": "MIT",
+      "dependencies": {
+        "micromark-factory-space": "^2.0.0",
+        "micromark-util-character": "^2.0.0",
+        "micromark-util-symbol": "^2.0.0",
+        "micromark-util-types": "^2.0.0"
+      }
+    },
+    "node_modules/micromark-factory-whitespace": {
+      "version": "2.0.1",
+      "resolved": "https://registry.npmjs.org/micromark-factory-whitespace/-/micromark-factory-whitespace-2.0.1.tgz",
+      "integrity": "sha512-Ob0nuZ3PKt/n0hORHyvoD9uZhr+Za8sFoP+OnMcnWK5lngSzALgQYKMr9RJVOWLqQYuyn6ulqGWSXdwf6F80lQ==",
+      "dev": true,
+      "funding": [
+        {
+          "type": "GitHub Sponsors",
+          "url": "https://github.com/sponsors/unifiedjs"
+        },
+        {
+          "type": "OpenCollective",
+          "url": "https://opencollective.com/unified"
+        }
+      ],
+      "license": "MIT",
+      "dependencies": {
+        "micromark-factory-space": "^2.0.0",
+        "micromark-util-character": "^2.0.0",
+        "micromark-util-symbol": "^2.0.0",
+        "micromark-util-types": "^2.0.0"
+      }
+    },
+    "node_modules/micromark-util-character": {
+      "version": "2.1.1",
+      "resolved": "https://registry.npmjs.org/micromark-util-character/-/micromark-util-character-2.1.1.tgz",
+      "integrity": "sha512-wv8tdUTJ3thSFFFJKtpYKOYiGP2+v96Hvk4Tu8KpCAsTMs6yi+nVmGh1syvSCsaxz45J6Jbw+9DD6g97+NV67Q==",
+      "dev": true,
+      "funding": [
+        {
+          "type": "GitHub Sponsors",
+          "url": "https://github.com/sponsors/unifiedjs"
+        },
+        {
+          "type": "OpenCollective",
+          "url": "https://opencollective.com/unified"
+        }
+      ],
+      "license": "MIT",
+      "dependencies": {
+        "micromark-util-symbol": "^2.0.0",
+        "micromark-util-types": "^2.0.0"
+      }
+    },
+    "node_modules/micromark-util-chunked": {
+      "version": "2.0.1",
+      "resolved": "https://registry.npmjs.org/micromark-util-chunked/-/micromark-util-chunked-2.0.1.tgz",
+      "integrity": "sha512-QUNFEOPELfmvv+4xiNg2sRYeS/P84pTW0TCgP5zc9FpXetHY0ab7SxKyAQCNCc1eK0459uoLI1y5oO5Vc1dbhA==",
+      "dev": true,
+      "funding": [
+        {
+          "type": "GitHub Sponsors",
+          "url": "https://github.com/sponsors/unifiedjs"
+        },
+        {
+          "type": "OpenCollective",
+          "url": "https://opencollective.com/unified"
+        }
+      ],
+      "license": "MIT",
+      "dependencies": {
+        "micromark-util-symbol": "^2.0.0"
+      }
+    },
+    "node_modules/micromark-util-classify-character": {
+      "version": "2.0.1",
+      "resolved": "https://registry.npmjs.org/micromark-util-classify-character/-/micromark-util-classify-character-2.0.1.tgz",
+      "integrity": "sha512-K0kHzM6afW/MbeWYWLjoHQv1sgg2Q9EccHEDzSkxiP/EaagNzCm7T/WMKZ3rjMbvIpvBiZgwR3dKMygtA4mG1Q==",
+      "dev": true,
+      "funding": [
+        {
+          "type": "GitHub Sponsors",
+          "url": "https://github.com/sponsors/unifiedjs"
+        },
+        {
+          "type": "OpenCollective",
+          "url": "https://opencollective.com/unified"
+        }
+      ],
+      "license": "MIT",
+      "dependencies": {
+        "micromark-util-character": "^2.0.0",
+        "micromark-util-symbol": "^2.0.0",
+        "micromark-util-types": "^2.0.0"
+      }
+    },
+    "node_modules/micromark-util-combine-extensions": {
+      "version": "2.0.1",
+      "resolved": "https://registry.npmjs.org/micromark-util-combine-extensions/-/micromark-util-combine-extensions-2.0.1.tgz",
+      "integrity": "sha512-OnAnH8Ujmy59JcyZw8JSbK9cGpdVY44NKgSM7E9Eh7DiLS2E9RNQf0dONaGDzEG9yjEl5hcqeIsj4hfRkLH/Bg==",
+      "dev": true,
+      "funding": [
+        {
+          "type": "GitHub Sponsors",
+          "url": "https://github.com/sponsors/unifiedjs"
+        },
+        {
+          "type": "OpenCollective",
+          "url": "https://opencollective.com/unified"
+        }
+      ],
+      "license": "MIT",
+      "dependencies": {
+        "micromark-util-chunked": "^2.0.0",
+        "micromark-util-types": "^2.0.0"
+      }
+    },
+    "node_modules/micromark-util-decode-numeric-character-reference": {
+      "version": "2.0.2",
+      "resolved": "https://registry.npmjs.org/micromark-util-decode-numeric-character-reference/-/micromark-util-decode-numeric-character-reference-2.0.2.tgz",
+      "integrity": "sha512-ccUbYk6CwVdkmCQMyr64dXz42EfHGkPQlBj5p7YVGzq8I7CtjXZJrubAYezf7Rp+bjPseiROqe7G6foFd+lEuw==",
+      "dev": true,
+      "funding": [
+        {
+          "type": "GitHub Sponsors",
+          "url": "https://github.com/sponsors/unifiedjs"
+        },
+        {
+          "type": "OpenCollective",
+          "url": "https://opencollective.com/unified"
+        }
+      ],
+      "license": "MIT",
+      "dependencies": {
+        "micromark-util-symbol": "^2.0.0"
+      }
+    },
+    "node_modules/micromark-util-decode-string": {
+      "version": "2.0.1",
+      "resolved": "https://registry.npmjs.org/micromark-util-decode-string/-/micromark-util-decode-string-2.0.1.tgz",
+      "integrity": "sha512-nDV/77Fj6eH1ynwscYTOsbK7rR//Uj0bZXBwJZRfaLEJ1iGBR6kIfNmlNqaqJf649EP0F3NWNdeJi03elllNUQ==",
+      "dev": true,
+      "funding": [
+        {
+          "type": "GitHub Sponsors",
+          "url": "https://github.com/sponsors/unifiedjs"
+        },
+        {
+          "type": "OpenCollective",
+          "url": "https://opencollective.com/unified"
+        }
+      ],
+      "license": "MIT",
+      "dependencies": {
+        "decode-named-character-reference": "^1.0.0",
+        "micromark-util-character": "^2.0.0",
+        "micromark-util-decode-numeric-character-reference": "^2.0.0",
+        "micromark-util-symbol": "^2.0.0"
+      }
+    },
+    "node_modules/micromark-util-encode": {
+      "version": "2.0.1",
+      "resolved": "https://registry.npmjs.org/micromark-util-encode/-/micromark-util-encode-2.0.1.tgz",
+      "integrity": "sha512-c3cVx2y4KqUnwopcO9b/SCdo2O67LwJJ/UyqGfbigahfegL9myoEFoDYZgkT7f36T0bLrM9hZTAaAyH+PCAXjw==",
+      "dev": true,
+      "funding": [
+        {
+          "type": "GitHub Sponsors",
+          "url": "https://github.com/sponsors/unifiedjs"
+        },
+        {
+          "type": "OpenCollective",
+          "url": "https://opencollective.com/unified"
+        }
+      ],
+      "license": "MIT"
+    },
+    "node_modules/micromark-util-html-tag-name": {
+      "version": "2.0.1",
+      "resolved": "https://registry.npmjs.org/micromark-util-html-tag-name/-/micromark-util-html-tag-name-2.0.1.tgz",
+      "integrity": "sha512-2cNEiYDhCWKI+Gs9T0Tiysk136SnR13hhO8yW6BGNyhOC4qYFnwF1nKfD3HFAIXA5c45RrIG1ub11GiXeYd1xA==",
+      "dev": true,
+      "funding": [
+        {
+          "type": "GitHub Sponsors",
+          "url": "https://github.com/sponsors/unifiedjs"
+        },
+        {
+          "type": "OpenCollective",
+          "url": "https://opencollective.com/unified"
+        }
+      ],
+      "license": "MIT"
+    },
+    "node_modules/micromark-util-normalize-identifier": {
+      "version": "2.0.1",
+      "resolved": "https://registry.npmjs.org/micromark-util-normalize-identifier/-/micromark-util-normalize-identifier-2.0.1.tgz",
+      "integrity": "sha512-sxPqmo70LyARJs0w2UclACPUUEqltCkJ6PhKdMIDuJ3gSf/Q+/GIe3WKl0Ijb/GyH9lOpUkRAO2wp0GVkLvS9Q==",
+      "dev": true,
+      "funding": [
+        {
+          "type": "GitHub Sponsors",
+          "url": "https://github.com/sponsors/unifiedjs"
+        },
+        {
+          "type": "OpenCollective",
+          "url": "https://opencollective.com/unified"
+        }
+      ],
+      "license": "MIT",
+      "dependencies": {
+        "micromark-util-symbol": "^2.0.0"
+      }
+    },
+    "node_modules/micromark-util-resolve-all": {
+      "version": "2.0.1",
+      "resolved": "https://registry.npmjs.org/micromark-util-resolve-all/-/micromark-util-resolve-all-2.0.1.tgz",
+      "integrity": "sha512-VdQyxFWFT2/FGJgwQnJYbe1jjQoNTS4RjglmSjTUlpUMa95Htx9NHeYW4rGDJzbjvCsl9eLjMQwGeElsqmzcHg==",
+      "dev": true,
+      "funding": [
+        {
+          "type": "GitHub Sponsors",
+          "url": "https://github.com/sponsors/unifiedjs"
+        },
+        {
+          "type": "OpenCollective",
+          "url": "https://opencollective.com/unified"
+        }
+      ],
+      "license": "MIT",
+      "dependencies": {
+        "micromark-util-types": "^2.0.0"
+      }
+    },
+    "node_modules/micromark-util-sanitize-uri": {
+      "version": "2.0.1",
+      "resolved": "https://registry.npmjs.org/micromark-util-sanitize-uri/-/micromark-util-sanitize-uri-2.0.1.tgz",
+      "integrity": "sha512-9N9IomZ/YuGGZZmQec1MbgxtlgougxTodVwDzzEouPKo3qFWvymFHWcnDi2vzV1ff6kas9ucW+o3yzJK9YB1AQ==",
+      "dev": true,
+      "funding": [
+        {
+          "type": "GitHub Sponsors",
+          "url": "https://github.com/sponsors/unifiedjs"
+        },
+        {
+          "type": "OpenCollective",
+          "url": "https://opencollective.com/unified"
+        }
+      ],
+      "license": "MIT",
+      "dependencies": {
+        "micromark-util-character": "^2.0.0",
+        "micromark-util-encode": "^2.0.0",
+        "micromark-util-symbol": "^2.0.0"
+      }
+    },
+    "node_modules/micromark-util-subtokenize": {
+      "version": "2.1.0",
+      "resolved": "https://registry.npmjs.org/micromark-util-subtokenize/-/micromark-util-subtokenize-2.1.0.tgz",
+      "integrity": "sha512-XQLu552iSctvnEcgXw6+Sx75GflAPNED1qx7eBJ+wydBb2KCbRZe+NwvIEEMM83uml1+2WSXpBAcp9IUCgCYWA==",
+      "dev": true,
+      "funding": [
+        {
+          "type": "GitHub Sponsors",
+          "url": "https://github.com/sponsors/unifiedjs"
+        },
+        {
+          "type": "OpenCollective",
+          "url": "https://opencollective.com/unified"
+        }
+      ],
+      "license": "MIT",
+      "dependencies": {
+        "devlop": "^1.0.0",
+        "micromark-util-chunked": "^2.0.0",
+        "micromark-util-symbol": "^2.0.0",
+        "micromark-util-types": "^2.0.0"
+      }
+    },
+    "node_modules/micromark-util-symbol": {
+      "version": "2.0.1",
+      "resolved": "https://registry.npmjs.org/micromark-util-symbol/-/micromark-util-symbol-2.0.1.tgz",
+      "integrity": "sha512-vs5t8Apaud9N28kgCrRUdEed4UJ+wWNvicHLPxCa9ENlYuAY31M0ETy5y1vA33YoNPDFTghEbnh6efaE8h4x0Q==",
+      "dev": true,
+      "funding": [
+        {
+          "type": "GitHub Sponsors",
+          "url": "https://github.com/sponsors/unifiedjs"
+        },
+        {
+          "type": "OpenCollective",
+          "url": "https://opencollective.com/unified"
+        }
+      ],
+      "license": "MIT"
+    },
+    "node_modules/micromark-util-types": {
+      "version": "2.0.2",
+      "resolved": "https://registry.npmjs.org/micromark-util-types/-/micromark-util-types-2.0.2.tgz",
+      "integrity": "sha512-Yw0ECSpJoViF1qTU4DC6NwtC4aWGt1EkzaQB8KPPyCRR8z9TWeV0HbEFGTO+ZY1wB22zmxnJqhPyTpOVCpeHTA==",
+      "dev": true,
+      "funding": [
+        {
+          "type": "GitHub Sponsors",
+          "url": "https://github.com/sponsors/unifiedjs"
+        },
+        {
+          "type": "OpenCollective",
+          "url": "https://opencollective.com/unified"
+        }
+      ],
+      "license": "MIT"
+    },
+    "node_modules/minimatch": {
+      "version": "3.1.5",
+      "resolved": "https://registry.npmjs.org/minimatch/-/minimatch-3.1.5.tgz",
+      "integrity": "sha512-VgjWUsnnT6n+NUk6eZq77zeFdpW2LWDzP6zFGrCbHXiYNul5Dzqk2HHQ5uFH2DNW5Xbp8+jVzaeNt94ssEEl4w==",
+      "dev": true,
+      "license": "ISC",
+      "dependencies": {
+        "brace-expansion": "^1.1.7"
+      },
+      "engines": {
+        "node": "*"
+      }
+    },
+    "node_modules/mlly": {
+      "version": "1.8.1",
+      "resolved": "https://registry.npmjs.org/mlly/-/mlly-1.8.1.tgz",
+      "integrity": "sha512-SnL6sNutTwRWWR/vcmCYHSADjiEesp5TGQQ0pXyLhW5IoeibRlF/CbSLailbB3CNqJUk9cVJ9dUDnbD7GrcHBQ==",
+      "dev": true,
+      "license": "MIT",
+      "dependencies": {
+        "acorn": "^8.16.0",
+        "pathe": "^2.0.3",
+        "pkg-types": "^1.3.1",
+        "ufo": "^1.6.3"
+      }
+    },
+    "node_modules/mlly/node_modules/confbox": {
+      "version": "0.1.8",
+      "resolved": "https://registry.npmjs.org/confbox/-/confbox-0.1.8.tgz",
+      "integrity": "sha512-RMtmw0iFkeR4YV+fUOSucriAQNb9g8zFR52MWCtl+cCZOFRNL6zeB395vPzFhEjjn4fMxXudmELnl/KF/WrK6w==",
+      "dev": true,
+      "license": "MIT"
+    },
+    "node_modules/mlly/node_modules/pkg-types": {
+      "version": "1.3.1",
+      "resolved": "https://registry.npmjs.org/pkg-types/-/pkg-types-1.3.1.tgz",
+      "integrity": "sha512-/Jm5M4RvtBFVkKWRu2BLUTNP8/M2a+UwuAX+ae4770q1qVGtfjG+WTCupoZixokjmHiry8uI+dlY8KXYV5HVVQ==",
+      "dev": true,
+      "license": "MIT",
+      "dependencies": {
+        "confbox": "^0.1.8",
+        "mlly": "^1.7.4",
+        "pathe": "^2.0.1"
+      }
+    },
+    "node_modules/moment": {
+      "version": "2.30.1",
+      "resolved": "https://registry.npmjs.org/moment/-/moment-2.30.1.tgz",
+      "integrity": "sha512-uEmtNhbDOrWPFS+hdjFCBfy9f2YoyzRpwcl+DqpC6taX21FzsTLQVbMV/W7PzNSX6x/bhC1zA3c2UQ5NzH6how==",
+      "dev": true,
+      "license": "MIT",
+      "engines": {
+        "node": "*"
+      }
+    },
+    "node_modules/mri": {
+      "version": "1.2.0",
+      "resolved": "https://registry.npmjs.org/mri/-/mri-1.2.0.tgz",
+      "integrity": "sha512-tzzskb3bG8LvYGFF/mDTpq3jpI6Q9wc3LEmBaghu+DdCssd1FakN7Bc0hVNmEyGq1bq3RgfkCb3cmQLpNPOroA==",
+      "dev": true,
+      "license": "MIT",
+      "engines": {
+        "node": ">=4"
+      }
+    },
+    "node_modules/ms": {
+      "version": "2.1.3",
+      "resolved": "https://registry.npmjs.org/ms/-/ms-2.1.3.tgz",
+      "integrity": "sha512-6FlzubTLZG3J2a/NVCAleEhjzq5oxgHyaCU9yYXvcLsvoVaHJq/s5xXI6/XXP6tz7R9xAOtHnSO/tXtF3WRTlA==",
+      "dev": true,
+      "license": "MIT"
+    },
+    "node_modules/nanoid": {
+      "version": "3.3.11",
+      "resolved": "https://registry.npmjs.org/nanoid/-/nanoid-3.3.11.tgz",
+      "integrity": "sha512-N8SpfPUnUp1bK+PMYW8qSWdl9U+wwNWI4QKxOYDy9JAro3WMX7p2OeVRF9v+347pnakNevPmiHhNmZ2HbFA76w==",
+      "dev": true,
+      "funding": [
+        {
+          "type": "github",
+          "url": "https://github.com/sponsors/ai"
+        }
+      ],
+      "license": "MIT",
+      "bin": {
+        "nanoid": "bin/nanoid.cjs"
+      },
+      "engines": {
+        "node": "^10 || ^12 || ^13.7 || ^14 || >=15.0.1"
+      }
+    },
+    "node_modules/natural-compare": {
+      "version": "1.4.0",
+      "resolved": "https://registry.npmjs.org/natural-compare/-/natural-compare-1.4.0.tgz",
+      "integrity": "sha512-OWND8ei3VtNC9h7V60qff3SVobHr996CTwgxubgyQYEpg290h9J0buyECNNJexkFm5sOajh5G116RYA1c8ZMSw==",
+      "dev": true,
+      "license": "MIT"
+    },
+    "node_modules/node-addon-api": {
+      "version": "7.1.1",
+      "resolved": "https://registry.npmjs.org/node-addon-api/-/node-addon-api-7.1.1.tgz",
+      "integrity": "sha512-5m3bsyrjFWE1xf7nz7YXdN4udnVtXK6/Yfgn5qnahL6bCkf2yKt4k3nuTKAtT4r3IG8JNR2ncsIMdZuAzJjHQQ==",
+      "dev": true,
+      "license": "MIT"
+    },
+    "node_modules/node-fetch-native": {
+      "version": "1.6.7",
+      "resolved": "https://registry.npmjs.org/node-fetch-native/-/node-fetch-native-1.6.7.tgz",
+      "integrity": "sha512-g9yhqoedzIUm0nTnTqAQvueMPVOuIY16bqgAJJC8XOOubYFNwz6IER9qs0Gq2Xd0+CecCKFjtdDTMA4u4xG06Q==",
+      "dev": true,
+      "license": "MIT"
+    },
+    "node_modules/node-releases": {
+      "version": "2.0.36",
+      "resolved": "https://registry.npmjs.org/node-releases/-/node-releases-2.0.36.tgz",
+      "integrity": "sha512-TdC8FSgHz8Mwtw9g5L4gR/Sh9XhSP/0DEkQxfEFXOpiul5IiHgHan2VhYYb6agDSfp4KuvltmGApc8HMgUrIkA==",
+      "dev": true,
+      "license": "MIT"
+    },
+    "node_modules/nypm": {
+      "version": "0.6.5",
+      "resolved": "https://registry.npmjs.org/nypm/-/nypm-0.6.5.tgz",
+      "integrity": "sha512-K6AJy1GMVyfyMXRVB88700BJqNUkByijGJM8kEHpLdcAt+vSQAVfkWWHYzuRXHSY6xA2sNc5RjTj0p9rE2izVQ==",
+      "dev": true,
+      "license": "MIT",
+      "dependencies": {
+        "citty": "^0.2.0",
+        "pathe": "^2.0.3",
+        "tinyexec": "^1.0.2"
+      },
+      "bin": {
+        "nypm": "dist/cli.mjs"
+      },
+      "engines": {
+        "node": ">=18"
+      }
+    },
+    "node_modules/obug": {
+      "version": "2.1.1",
+      "resolved": "https://registry.npmjs.org/obug/-/obug-2.1.1.tgz",
+      "integrity": "sha512-uTqF9MuPraAQ+IsnPf366RG4cP9RtUi7MLO1N3KEc+wb0a6yKpeL0lmk2IB1jY5KHPAlTc6T/JRdC/YqxHNwkQ==",
+      "dev": true,
+      "funding": [
+        "https://github.com/sponsors/sxzz",
+        "https://opencollective.com/debug"
+      ],
+      "license": "MIT"
+    },
+    "node_modules/obuild": {
+      "version": "0.4.32",
+      "resolved": "https://registry.npmjs.org/obuild/-/obuild-0.4.32.tgz",
+      "integrity": "sha512-U6NNt0ZL6lCE6B6aU+qcBwbFU4g2DwrO/68aKhyXNUXgq0jf+fyHg9s9+ETZOLmlfJvQwub5mHF3aohGgWGYKQ==",
+      "dev": true,
+      "license": "MIT",
+      "dependencies": {
+        "c12": "^4.0.0-beta.3",
+        "consola": "^3.4.2",
+        "defu": "^6.1.4",
+        "exsolve": "^1.0.8",
+        "magic-string": "^0.30.21",
+        "pathe": "^2.0.3",
+        "pretty-bytes": "^7.1.0",
+        "rolldown": "^1.0.0-rc.7",
+        "rolldown-plugin-dts": "^0.22.4",
+        "rollup-plugin-license": "^3.7.0",
+        "tinyglobby": "^0.2.15"
+      },
+      "bin": {
+        "obuild": "dist/cli.mjs"
+      }
+    },
+    "node_modules/obuild/node_modules/c12": {
+      "version": "4.0.0-beta.3",
+      "resolved": "https://registry.npmjs.org/c12/-/c12-4.0.0-beta.3.tgz",
+      "integrity": "sha512-15pHxeM4kKKnF1zPgvXq/ETPhf9DRLGg0Id3GAyQhQqYxt8WkUCbo0NipHjQUUNC5VPP8TQA32pGPDUZmAi/3g==",
+      "dev": true,
+      "license": "MIT",
+      "dependencies": {
+        "confbox": "^0.2.4",
+        "defu": "^6.1.4",
+        "exsolve": "^1.0.8",
+        "pathe": "^2.0.3",
+        "pkg-types": "^2.3.0",
+        "rc9": "^3.0.0"
+      },
+      "peerDependencies": {
+        "chokidar": "^5",
+        "dotenv": "*",
+        "giget": "*",
+        "jiti": "*",
+        "magicast": "*"
+      },
+      "peerDependenciesMeta": {
+        "chokidar": {
+          "optional": true
+        },
+        "dotenv": {
+          "optional": true
+        },
+        "giget": {
+          "optional": true
+        },
+        "jiti": {
+          "optional": true
+        },
+        "magicast": {
+          "optional": true
+        }
+      }
+    },
+    "node_modules/obuild/node_modules/rc9": {
+      "version": "3.0.0",
+      "resolved": "https://registry.npmjs.org/rc9/-/rc9-3.0.0.tgz",
+      "integrity": "sha512-MGOue0VqscKWQ104udASX/3GYDcKyPI4j4F8gu/jHHzglpmy9a/anZK3PNe8ug6aZFl+9GxLtdhe3kVZuMaQbA==",
+      "dev": true,
+      "license": "MIT",
+      "dependencies": {
+        "defu": "^6.1.4",
+        "destr": "^2.0.5"
+      }
+    },
+    "node_modules/ofetch": {
+      "version": "1.5.1",
+      "resolved": "https://registry.npmjs.org/ofetch/-/ofetch-1.5.1.tgz",
+      "integrity": "sha512-2W4oUZlVaqAPAil6FUg/difl6YhqhUR7x2eZY4bQCko22UXg3hptq9KLQdqFClV+Wu85UX7hNtdGTngi/1BxcA==",
+      "dev": true,
+      "license": "MIT",
+      "dependencies": {
+        "destr": "^2.0.5",
+        "node-fetch-native": "^1.6.7",
+        "ufo": "^1.6.1"
+      }
+    },
+    "node_modules/ohash": {
+      "version": "2.0.11",
+      "resolved": "https://registry.npmjs.org/ohash/-/ohash-2.0.11.tgz",
+      "integrity": "sha512-RdR9FQrFwNBNXAr4GixM8YaRZRJ5PUWbKYbE5eOsrwAjJW0q2REGcf79oYPsLyskQCZG1PLN+S/K1V00joZAoQ==",
+      "dev": true,
+      "license": "MIT"
+    },
+    "node_modules/open": {
+      "version": "10.2.0",
+      "resolved": "https://registry.npmjs.org/open/-/open-10.2.0.tgz",
+      "integrity": "sha512-YgBpdJHPyQ2UE5x+hlSXcnejzAvD0b22U2OuAP+8OnlJT+PjWPxtgmGqKKc+RgTM63U9gN0YzrYc71R2WT/hTA==",
+      "dev": true,
+      "license": "MIT",
+      "dependencies": {
+        "default-browser": "^5.2.1",
+        "define-lazy-prop": "^3.0.0",
+        "is-inside-container": "^1.0.0",
+        "wsl-utils": "^0.1.0"
+      },
+      "engines": {
+        "node": ">=18"
+      },
+      "funding": {
+        "url": "https://github.com/sponsors/sindresorhus"
+      }
+    },
+    "node_modules/optionator": {
+      "version": "0.9.4",
+      "resolved": "https://registry.npmjs.org/optionator/-/optionator-0.9.4.tgz",
+      "integrity": "sha512-6IpQ7mKUxRcZNLIObR0hz7lxsapSSIYNZJwXPGeF0mTVqGKFIXj1DQcMoT22S3ROcLyY/rz0PWaWZ9ayWmad9g==",
+      "dev": true,
+      "license": "MIT",
+      "dependencies": {
+        "deep-is": "^0.1.3",
+        "fast-levenshtein": "^2.0.6",
+        "levn": "^0.4.1",
+        "prelude-ls": "^1.2.1",
+        "type-check": "^0.4.0",
+        "word-wrap": "^1.2.5"
+      },
+      "engines": {
+        "node": ">= 0.8.0"
+      }
+    },
+    "node_modules/p-limit": {
+      "version": "3.1.0",
+      "resolved": "https://registry.npmjs.org/p-limit/-/p-limit-3.1.0.tgz",
+      "integrity": "sha512-TYOanM3wGwNGsZN2cVTYPArw454xnXj5qmWF1bEoAc4+cU/ol7GVh7odevjp1FNHduHc3KZMcFduxU5Xc6uJRQ==",
+      "dev": true,
+      "license": "MIT",
+      "dependencies": {
+        "yocto-queue": "^0.1.0"
+      },
+      "engines": {
+        "node": ">=10"
+      },
+      "funding": {
+        "url": "https://github.com/sponsors/sindresorhus"
+      }
+    },
+    "node_modules/p-locate": {
+      "version": "5.0.0",
+      "resolved": "https://registry.npmjs.org/p-locate/-/p-locate-5.0.0.tgz",
+      "integrity": "sha512-LaNjtRWUBY++zB5nE/NwcaoMylSPk+S+ZHNB1TzdbMJMny6dynpAGt7X/tl/QYq3TIeE6nxHppbo2LGymrG5Pw==",
+      "dev": true,
+      "license": "MIT",
+      "dependencies": {
+        "p-limit": "^3.0.2"
+      },
+      "engines": {
+        "node": ">=10"
+      },
+      "funding": {
+        "url": "https://github.com/sponsors/sindresorhus"
+      }
+    },
+    "node_modules/package-name-regex": {
+      "version": "2.0.6",
+      "resolved": "https://registry.npmjs.org/package-name-regex/-/package-name-regex-2.0.6.tgz",
+      "integrity": "sha512-gFL35q7kbE/zBaPA3UKhp2vSzcPYx2ecbYuwv1ucE9Il6IIgBDweBlH8D68UFGZic2MkllKa2KHCfC1IQBQUYA==",
+      "dev": true,
+      "license": "MIT",
+      "engines": {
+        "node": ">=12"
+      },
+      "funding": {
+        "url": "https://github.com/sponsors/dword-design"
+      }
+    },
+    "node_modules/parent-module": {
+      "version": "1.0.1",
+      "resolved": "https://registry.npmjs.org/parent-module/-/parent-module-1.0.1.tgz",
+      "integrity": "sha512-GQ2EWRpQV8/o+Aw8YqtfZZPfNRWZYkbidE9k5rpl/hC3vtHHBfGm2Ifi6qWV+coDGkrUKZAxE3Lot5kcsRlh+g==",
+      "dev": true,
+      "license": "MIT",
+      "dependencies": {
+        "callsites": "^3.0.0"
+      },
+      "engines": {
+        "node": ">=6"
+      }
+    },
+    "node_modules/path-exists": {
+      "version": "4.0.0",
+      "resolved": "https://registry.npmjs.org/path-exists/-/path-exists-4.0.0.tgz",
+      "integrity": "sha512-ak9Qy5Q7jYb2Wwcey5Fpvg2KoAc/ZIhLSLOSBmRmygPsGwkVVt0fZa0qrtMz+m6tJTAHfZQ8FnmB4MG4LWy7/w==",
+      "dev": true,
+      "license": "MIT",
+      "engines": {
+        "node": ">=8"
+      }
+    },
+    "node_modules/path-key": {
+      "version": "3.1.1",
+      "resolved": "https://registry.npmjs.org/path-key/-/path-key-3.1.1.tgz",
+      "integrity": "sha512-ojmeN0qd+y0jszEtoY48r0Peq5dwMEkIlCOu6Q5f41lfkswXuKtYrhgoTpLnyIcHm24Uhqx+5Tqm2InSwLhE6Q==",
+      "dev": true,
+      "license": "MIT",
+      "engines": {
+        "node": ">=8"
+      }
+    },
+    "node_modules/pathe": {
+      "version": "2.0.3",
+      "resolved": "https://registry.npmjs.org/pathe/-/pathe-2.0.3.tgz",
+      "integrity": "sha512-WUjGcAqP1gQacoQe+OBJsFA7Ld4DyXuUIjZ5cc75cLHvJ7dtNsTugphxIADwspS+AraAUePCKrSVtPLFj/F88w==",
+      "dev": true,
+      "license": "MIT"
+    },
+    "node_modules/perfect-debounce": {
+      "version": "2.1.0",
+      "resolved": "https://registry.npmjs.org/perfect-debounce/-/perfect-debounce-2.1.0.tgz",
+      "integrity": "sha512-LjgdTytVFXeUgtHZr9WYViYSM/g8MkcTPYDlPa3cDqMirHjKiSZPYd6DoL7pK8AJQr+uWkQvCjHNdiMqsrJs+g==",
+      "dev": true,
+      "license": "MIT"
+    },
+    "node_modules/picocolors": {
+      "version": "1.1.1",
+      "resolved": "https://registry.npmjs.org/picocolors/-/picocolors-1.1.1.tgz",
+      "integrity": "sha512-xceH2snhtb5M9liqDsmEw56le376mTZkEX/jEb/RxNFyegNul7eNslCXP9FDj/Lcu0X8KEyMceP2ntpaHrDEVA==",
+      "dev": true,
+      "license": "ISC"
+    },
+    "node_modules/picomatch": {
+      "version": "4.0.3",
+      "resolved": "https://registry.npmjs.org/picomatch/-/picomatch-4.0.3.tgz",
+      "integrity": "sha512-5gTmgEY/sqK6gFXLIsQNH19lWb4ebPDLA4SdLP7dsWkIXHWlG66oPuVvXSGFPppYZz8ZDZq0dYYrbHfBCVUb1Q==",
+      "dev": true,
+      "license": "MIT",
+      "engines": {
+        "node": ">=12"
+      },
+      "funding": {
+        "url": "https://github.com/sponsors/jonschlinkert"
+      }
+    },
+    "node_modules/pkg-types": {
+      "version": "2.3.0",
+      "resolved": "https://registry.npmjs.org/pkg-types/-/pkg-types-2.3.0.tgz",
+      "integrity": "sha512-SIqCzDRg0s9npO5XQ3tNZioRY1uK06lA41ynBC1YmFTmnY6FjUjVt6s4LoADmwoig1qqD0oK8h1p/8mlMx8Oig==",
+      "dev": true,
+      "license": "MIT",
+      "dependencies": {
+        "confbox": "^0.2.2",
+        "exsolve": "^1.0.7",
+        "pathe": "^2.0.3"
+      }
+    },
+    "node_modules/pluralize": {
+      "version": "8.0.0",
+      "resolved": "https://registry.npmjs.org/pluralize/-/pluralize-8.0.0.tgz",
+      "integrity": "sha512-Nc3IT5yHzflTfbjgqWcCPpo7DaKy4FnpB0l/zCAW0Tc7jxAiuqSxHasntB3D7887LSrA93kDJ9IXovxJYxyLCA==",
+      "dev": true,
+      "license": "MIT",
+      "engines": {
+        "node": ">=4"
+      }
+    },
+    "node_modules/postcss": {
+      "version": "8.5.8",
+      "resolved": "https://registry.npmjs.org/postcss/-/postcss-8.5.8.tgz",
+      "integrity": "sha512-OW/rX8O/jXnm82Ey1k44pObPtdblfiuWnrd8X7GJ7emImCOstunGbXUpp7HdBrFQX6rJzn3sPT397Wp5aCwCHg==",
+      "dev": true,
+      "funding": [
+        {
+          "type": "opencollective",
+          "url": "https://opencollective.com/postcss/"
+        },
+        {
+          "type": "tidelift",
+          "url": "https://tidelift.com/funding/github/npm/postcss"
+        },
+        {
+          "type": "github",
+          "url": "https://github.com/sponsors/ai"
+        }
+      ],
+      "license": "MIT",
+      "dependencies": {
+        "nanoid": "^3.3.11",
+        "picocolors": "^1.1.1",
+        "source-map-js": "^1.2.1"
+      },
+      "engines": {
+        "node": "^10 || ^12 || >=14"
+      }
+    },
+    "node_modules/prelude-ls": {
+      "version": "1.2.1",
+      "resolved": "https://registry.npmjs.org/prelude-ls/-/prelude-ls-1.2.1.tgz",
+      "integrity": "sha512-vkcDPrRZo1QZLbn5RLGPpg/WmIQ65qoWWhcGKf/b5eplkkarX0m9z8ppCat4mlOqUsWpyNuYgO3VRyrYHSzX5g==",
+      "dev": true,
+      "license": "MIT",
+      "engines": {
+        "node": ">= 0.8.0"
+      }
+    },
+    "node_modules/prettier": {
+      "version": "3.8.1",
+      "resolved": "https://registry.npmjs.org/prettier/-/prettier-3.8.1.tgz",
+      "integrity": "sha512-UOnG6LftzbdaHZcKoPFtOcCKztrQ57WkHDeRD9t/PTQtmT0NHSeWWepj6pS0z/N7+08BHFDQVUrfmfMRcZwbMg==",
+      "dev": true,
+      "license": "MIT",
+      "bin": {
+        "prettier": "bin/prettier.cjs"
+      },
+      "engines": {
+        "node": ">=14"
+      },
+      "funding": {
+        "url": "https://github.com/prettier/prettier?sponsor=1"
+      }
+    },
+    "node_modules/pretty-bytes": {
+      "version": "7.1.0",
+      "resolved": "https://registry.npmjs.org/pretty-bytes/-/pretty-bytes-7.1.0.tgz",
+      "integrity": "sha512-nODzvTiYVRGRqAOvE84Vk5JDPyyxsVk0/fbA/bq7RqlnhksGpset09XTxbpvLTIjoaF7K8Z8DG8yHtKGTPSYRw==",
+      "dev": true,
+      "license": "MIT",
+      "engines": {
+        "node": ">=20"
+      },
+      "funding": {
+        "url": "https://github.com/sponsors/sindresorhus"
+      }
+    },
+    "node_modules/punycode": {
+      "version": "2.3.1",
+      "resolved": "https://registry.npmjs.org/punycode/-/punycode-2.3.1.tgz",
+      "integrity": "sha512-vYt7UD1U9Wg6138shLtLOvdAu+8DsC/ilFtEVHcH+wydcSpNE20AfSOduf6MkRFahL5FY7X1oU7nKVZFtfq8Fg==",
+      "dev": true,
+      "license": "MIT",
+      "engines": {
+        "node": ">=6"
+      }
+    },
+    "node_modules/rc9": {
+      "version": "2.1.2",
+      "resolved": "https://registry.npmjs.org/rc9/-/rc9-2.1.2.tgz",
+      "integrity": "sha512-btXCnMmRIBINM2LDZoEmOogIZU7Qe7zn4BpomSKZ/ykbLObuBdvG+mFq11DL6fjH1DRwHhrlgtYWG96bJiC7Cg==",
+      "dev": true,
+      "license": "MIT",
+      "dependencies": {
+        "defu": "^6.1.4",
+        "destr": "^2.0.3"
+      }
+    },
+    "node_modules/readdirp": {
+      "version": "5.0.0",
+      "resolved": "https://registry.npmjs.org/readdirp/-/readdirp-5.0.0.tgz",
+      "integrity": "sha512-9u/XQ1pvrQtYyMpZe7DXKv2p5CNvyVwzUB6uhLAnQwHMSgKMBR62lc7AHljaeteeHXn11XTAaLLUVZYVZyuRBQ==",
+      "dev": true,
+      "license": "MIT",
+      "engines": {
+        "node": ">= 20.19.0"
+      },
+      "funding": {
+        "type": "individual",
+        "url": "https://paulmillr.com/funding/"
+      }
+    },
+    "node_modules/regexp-tree": {
+      "version": "0.1.27",
+      "resolved": "https://registry.npmjs.org/regexp-tree/-/regexp-tree-0.1.27.tgz",
+      "integrity": "sha512-iETxpjK6YoRWJG5o6hXLwvjYAoW+FEZn9os0PD/b6AP6xQwsa/Y7lCVgIixBbUPMfhu+i2LtdeAqVTgGlQarfA==",
+      "dev": true,
+      "license": "MIT",
+      "bin": {
+        "regexp-tree": "bin/regexp-tree"
+      }
+    },
+    "node_modules/regjsparser": {
+      "version": "0.13.0",
+      "resolved": "https://registry.npmjs.org/regjsparser/-/regjsparser-0.13.0.tgz",
+      "integrity": "sha512-NZQZdC5wOE/H3UT28fVGL+ikOZcEzfMGk/c3iN9UGxzWHMa1op7274oyiUVrAG4B2EuFhus8SvkaYnhvW92p9Q==",
+      "dev": true,
+      "license": "BSD-2-Clause",
+      "dependencies": {
+        "jsesc": "~3.1.0"
+      },
+      "bin": {
+        "regjsparser": "bin/parser"
+      }
+    },
+    "node_modules/resolve-from": {
+      "version": "4.0.0",
+      "resolved": "https://registry.npmjs.org/resolve-from/-/resolve-from-4.0.0.tgz",
+      "integrity": "sha512-pb/MYmXstAkysRFx8piNI1tGFNQIFA3vkE3Gq4EuA1dF6gHp/+vgZqsCGJapvy8N3Q+4o7FwvquPJcnZ7RYy4g==",
+      "dev": true,
+      "license": "MIT",
+      "engines": {
+        "node": ">=4"
+      }
+    },
+    "node_modules/resolve-pkg-maps": {
+      "version": "1.0.0",
+      "resolved": "https://registry.npmjs.org/resolve-pkg-maps/-/resolve-pkg-maps-1.0.0.tgz",
+      "integrity": "sha512-seS2Tj26TBVOC2NIc2rOe2y2ZO7efxITtLZcGSOnHHNOQ7CkiUBfw0Iw2ck6xkIhPwLhKNLS8BO+hEpngQlqzw==",
+      "dev": true,
+      "license": "MIT",
+      "funding": {
+        "url": "https://github.com/privatenumber/resolve-pkg-maps?sponsor=1"
+      }
+    },
+    "node_modules/rolldown": {
+      "version": "1.0.0-rc.8",
+      "resolved": "https://registry.npmjs.org/rolldown/-/rolldown-1.0.0-rc.8.tgz",
+      "integrity": "sha512-RGOL7mz/aoQpy/y+/XS9iePBfeNRDUdozrhCEJxdpJyimW8v6yp4c30q6OviUU5AnUJVLRL9GP//HUs6N3ALrQ==",
+      "dev": true,
+      "license": "MIT",
+      "dependencies": {
+        "@oxc-project/types": "=0.115.0",
+        "@rolldown/pluginutils": "1.0.0-rc.8"
+      },
+      "bin": {
+        "rolldown": "bin/cli.mjs"
+      },
+      "engines": {
+        "node": "^20.19.0 || >=22.12.0"
+      },
+      "optionalDependencies": {
+        "@rolldown/binding-android-arm64": "1.0.0-rc.8",
+        "@rolldown/binding-darwin-arm64": "1.0.0-rc.8",
+        "@rolldown/binding-darwin-x64": "1.0.0-rc.8",
+        "@rolldown/binding-freebsd-x64": "1.0.0-rc.8",
+        "@rolldown/binding-linux-arm-gnueabihf": "1.0.0-rc.8",
+        "@rolldown/binding-linux-arm64-gnu": "1.0.0-rc.8",
+        "@rolldown/binding-linux-arm64-musl": "1.0.0-rc.8",
+        "@rolldown/binding-linux-ppc64-gnu": "1.0.0-rc.8",
+        "@rolldown/binding-linux-s390x-gnu": "1.0.0-rc.8",
+        "@rolldown/binding-linux-x64-gnu": "1.0.0-rc.8",
+        "@rolldown/binding-linux-x64-musl": "1.0.0-rc.8",
+        "@rolldown/binding-openharmony-arm64": "1.0.0-rc.8",
+        "@rolldown/binding-wasm32-wasi": "1.0.0-rc.8",
+        "@rolldown/binding-win32-arm64-msvc": "1.0.0-rc.8",
+        "@rolldown/binding-win32-x64-msvc": "1.0.0-rc.8"
+      }
+    },
+    "node_modules/rolldown-plugin-dts": {
+      "version": "0.22.4",
+      "resolved": "https://registry.npmjs.org/rolldown-plugin-dts/-/rolldown-plugin-dts-0.22.4.tgz",
+      "integrity": "sha512-pueqTPyN1N6lWYivyDGad+j+GO3DT67pzpct8s8e6KGVIezvnrDjejuw1AXFeyDRas3xTq4Ja6Lj5R5/04C5GQ==",
+      "dev": true,
+      "license": "MIT",
+      "dependencies": {
+        "@babel/generator": "8.0.0-rc.2",
+        "@babel/helper-validator-identifier": "8.0.0-rc.2",
+        "@babel/parser": "8.0.0-rc.2",
+        "@babel/types": "8.0.0-rc.2",
+        "ast-kit": "^3.0.0-beta.1",
+        "birpc": "^4.0.0",
+        "dts-resolver": "^2.1.3",
+        "get-tsconfig": "^4.13.6",
+        "obug": "^2.1.1"
+      },
+      "engines": {
+        "node": ">=20.19.0"
+      },
+      "funding": {
+        "url": "https://github.com/sponsors/sxzz"
+      },
+      "peerDependencies": {
+        "@ts-macro/tsc": "^0.3.6",
+        "@typescript/native-preview": ">=7.0.0-dev.20250601.1",
+        "rolldown": "^1.0.0-rc.3",
+        "typescript": "^5.0.0 || ^6.0.0-beta",
+        "vue-tsc": "~3.2.0"
+      },
+      "peerDependenciesMeta": {
+        "@ts-macro/tsc": {
+          "optional": true
+        },
+        "@typescript/native-preview": {
+          "optional": true
+        },
+        "typescript": {
+          "optional": true
+        },
+        "vue-tsc": {
+          "optional": true
+        }
+      }
+    },
+    "node_modules/rolldown-plugin-dts/node_modules/@babel/helper-string-parser": {
+      "version": "8.0.0-rc.2",
+      "resolved": "https://registry.npmjs.org/@babel/helper-string-parser/-/helper-string-parser-8.0.0-rc.2.tgz",
+      "integrity": "sha512-noLx87RwlBEMrTzncWd/FvTxoJ9+ycHNg0n8yyYydIoDsLZuxknKgWRJUqcrVkNrJ74uGyhWQzQaS3q8xfGAhQ==",
+      "dev": true,
+      "license": "MIT",
+      "engines": {
+        "node": "^20.19.0 || >=22.12.0"
+      }
+    },
+    "node_modules/rolldown-plugin-dts/node_modules/@babel/helper-validator-identifier": {
+      "version": "8.0.0-rc.2",
+      "resolved": "https://registry.npmjs.org/@babel/helper-validator-identifier/-/helper-validator-identifier-8.0.0-rc.2.tgz",
+      "integrity": "sha512-xExUBkuXWJjVuIbO7z6q7/BA9bgfJDEhVL0ggrggLMbg0IzCUWGT1hZGE8qUH7Il7/RD/a6cZ3AAFrrlp1LF/A==",
+      "dev": true,
+      "license": "MIT",
+      "engines": {
+        "node": "^20.19.0 || >=22.12.0"
+      }
+    },
+    "node_modules/rolldown-plugin-dts/node_modules/@babel/parser": {
+      "version": "8.0.0-rc.2",
+      "resolved": "https://registry.npmjs.org/@babel/parser/-/parser-8.0.0-rc.2.tgz",
+      "integrity": "sha512-29AhEtcq4x8Dp3T72qvUMZHx0OMXCj4Jy/TEReQa+KWLln524Cj1fWb3QFi0l/xSpptQBR6y9RNEXuxpFvwiUQ==",
+      "dev": true,
+      "license": "MIT",
+      "dependencies": {
+        "@babel/types": "^8.0.0-rc.2"
+      },
+      "bin": {
+        "parser": "bin/babel-parser.js"
+      },
+      "engines": {
+        "node": "^20.19.0 || >=22.12.0"
+      }
+    },
+    "node_modules/rolldown-plugin-dts/node_modules/@babel/types": {
+      "version": "8.0.0-rc.2",
+      "resolved": "https://registry.npmjs.org/@babel/types/-/types-8.0.0-rc.2.tgz",
+      "integrity": "sha512-91gAaWRznDwSX4E2tZ1YjBuIfnQVOFDCQ2r0Toby0gu4XEbyF623kXLMA8d4ZbCu+fINcrudkmEcwSUHgDDkNw==",
+      "dev": true,
+      "license": "MIT",
+      "dependencies": {
+        "@babel/helper-string-parser": "^8.0.0-rc.2",
+        "@babel/helper-validator-identifier": "^8.0.0-rc.2"
+      },
+      "engines": {
+        "node": "^20.19.0 || >=22.12.0"
+      }
+    },
+    "node_modules/rollup": {
+      "version": "4.59.0",
+      "resolved": "https://registry.npmjs.org/rollup/-/rollup-4.59.0.tgz",
+      "integrity": "sha512-2oMpl67a3zCH9H79LeMcbDhXW/UmWG/y2zuqnF2jQq5uq9TbM9TVyXvA4+t+ne2IIkBdrLpAaRQAvo7YI/Yyeg==",
+      "dev": true,
+      "license": "MIT",
+      "dependencies": {
+        "@types/estree": "1.0.8"
+      },
+      "bin": {
+        "rollup": "dist/bin/rollup"
+      },
+      "engines": {
+        "node": ">=18.0.0",
+        "npm": ">=8.0.0"
+      },
+      "optionalDependencies": {
+        "@rollup/rollup-android-arm-eabi": "4.59.0",
+        "@rollup/rollup-android-arm64": "4.59.0",
+        "@rollup/rollup-darwin-arm64": "4.59.0",
+        "@rollup/rollup-darwin-x64": "4.59.0",
+        "@rollup/rollup-freebsd-arm64": "4.59.0",
+        "@rollup/rollup-freebsd-x64": "4.59.0",
+        "@rollup/rollup-linux-arm-gnueabihf": "4.59.0",
+        "@rollup/rollup-linux-arm-musleabihf": "4.59.0",
+        "@rollup/rollup-linux-arm64-gnu": "4.59.0",
+        "@rollup/rollup-linux-arm64-musl": "4.59.0",
+        "@rollup/rollup-linux-loong64-gnu": "4.59.0",
+        "@rollup/rollup-linux-loong64-musl": "4.59.0",
+        "@rollup/rollup-linux-ppc64-gnu": "4.59.0",
+        "@rollup/rollup-linux-ppc64-musl": "4.59.0",
+        "@rollup/rollup-linux-riscv64-gnu": "4.59.0",
+        "@rollup/rollup-linux-riscv64-musl": "4.59.0",
+        "@rollup/rollup-linux-s390x-gnu": "4.59.0",
+        "@rollup/rollup-linux-x64-gnu": "4.59.0",
+        "@rollup/rollup-linux-x64-musl": "4.59.0",
+        "@rollup/rollup-openbsd-x64": "4.59.0",
+        "@rollup/rollup-openharmony-arm64": "4.59.0",
+        "@rollup/rollup-win32-arm64-msvc": "4.59.0",
+        "@rollup/rollup-win32-ia32-msvc": "4.59.0",
+        "@rollup/rollup-win32-x64-gnu": "4.59.0",
+        "@rollup/rollup-win32-x64-msvc": "4.59.0",
+        "fsevents": "~2.3.2"
+      }
+    },
+    "node_modules/rollup-plugin-license": {
+      "version": "3.7.0",
+      "resolved": "https://registry.npmjs.org/rollup-plugin-license/-/rollup-plugin-license-3.7.0.tgz",
+      "integrity": "sha512-RvvOIF+GH3fBR3wffgc/vmjQn6qOn72WjppWVDp/v+CLpT0BbcRBdSkPeeIOL6U5XccdYgSIMjUyXgxlKEEFcw==",
+      "dev": true,
+      "license": "MIT",
+      "dependencies": {
+        "commenting": "^1.1.0",
+        "fdir": "^6.4.3",
+        "lodash": "^4.17.21",
+        "magic-string": "^0.30.0",
+        "moment": "^2.30.1",
+        "package-name-regex": "^2.0.6",
+        "spdx-expression-validate": "^2.0.0",
+        "spdx-satisfies": "^5.0.1"
+      },
+      "engines": {
+        "node": ">=14.0.0"
+      },
+      "peerDependencies": {
+        "rollup": "^1.0.0 || ^2.0.0 || ^3.0.0 || ^4.0.0"
+      }
+    },
+    "node_modules/run-applescript": {
+      "version": "7.1.0",
+      "resolved": "https://registry.npmjs.org/run-applescript/-/run-applescript-7.1.0.tgz",
+      "integrity": "sha512-DPe5pVFaAsinSaV6QjQ6gdiedWDcRCbUuiQfQa2wmWV7+xC9bGulGI8+TdRmoFkAPaBXk8CrAbnlY2ISniJ47Q==",
+      "dev": true,
+      "license": "MIT",
+      "engines": {
+        "node": ">=18"
+      },
+      "funding": {
+        "url": "https://github.com/sponsors/sindresorhus"
+      }
+    },
+    "node_modules/scule": {
+      "version": "1.3.0",
+      "resolved": "https://registry.npmjs.org/scule/-/scule-1.3.0.tgz",
+      "integrity": "sha512-6FtHJEvt+pVMIB9IBY+IcCJ6Z5f1iQnytgyfKMhDKgmzYG+TeH/wx1y3l27rshSbLiSanrR9ffZDrEsmjlQF2g==",
+      "dev": true,
+      "license": "MIT"
+    },
+    "node_modules/semver": {
+      "version": "7.7.4",
+      "resolved": "https://registry.npmjs.org/semver/-/semver-7.7.4.tgz",
+      "integrity": "sha512-vFKC2IEtQnVhpT78h1Yp8wzwrf8CM+MzKMHGJZfBtzhZNycRFnXsHk6E5TxIkkMsgNS7mdX3AGB7x2QM2di4lA==",
+      "dev": true,
+      "license": "ISC",
+      "bin": {
+        "semver": "bin/semver.js"
+      },
+      "engines": {
+        "node": ">=10"
+      }
+    },
+    "node_modules/shebang-command": {
+      "version": "2.0.0",
+      "resolved": "https://registry.npmjs.org/shebang-command/-/shebang-command-2.0.0.tgz",
+      "integrity": "sha512-kHxr2zZpYtdmrN1qDjrrX/Z1rR1kG8Dx+gkpK1G4eXmvXswmcE1hTWBWYUzlraYw1/yZp6YuDY77YtvbN0dmDA==",
+      "dev": true,
+      "license": "MIT",
+      "dependencies": {
+        "shebang-regex": "^3.0.0"
+      },
+      "engines": {
+        "node": ">=8"
+      }
+    },
+    "node_modules/shebang-regex": {
+      "version": "3.0.0",
+      "resolved": "https://registry.npmjs.org/shebang-regex/-/shebang-regex-3.0.0.tgz",
+      "integrity": "sha512-7++dFhtcx3353uBaq8DDR4NuxBetBzC7ZQOhmTQInHEd6bSrXdiEyzCvG07Z44UYdLShWUyXt5M/yhz8ekcb1A==",
+      "dev": true,
+      "license": "MIT",
+      "engines": {
+        "node": ">=8"
+      }
+    },
+    "node_modules/siginfo": {
+      "version": "2.0.0",
+      "resolved": "https://registry.npmjs.org/siginfo/-/siginfo-2.0.0.tgz",
+      "integrity": "sha512-ybx0WO1/8bSBLEWXZvEd7gMW3Sn3JFlW3TvX1nREbDLRNQNaeNN8WK0meBwPdAaOI7TtRRRJn/Es1zhrrCHu7g==",
+      "dev": true,
+      "license": "ISC"
+    },
+    "node_modules/source-map-js": {
+      "version": "1.2.1",
+      "resolved": "https://registry.npmjs.org/source-map-js/-/source-map-js-1.2.1.tgz",
+      "integrity": "sha512-UXWMKhLOwVKb728IUtQPXxfYU+usdybtUrK/8uGE8CQMvrhOpwvzDBwj0QhSL7MQc7vIsISBG8VQ8+IDQxpfQA==",
+      "dev": true,
+      "license": "BSD-3-Clause",
+      "engines": {
+        "node": ">=0.10.0"
+      }
+    },
+    "node_modules/spdx-compare": {
+      "version": "1.0.0",
+      "resolved": "https://registry.npmjs.org/spdx-compare/-/spdx-compare-1.0.0.tgz",
+      "integrity": "sha512-C1mDZOX0hnu0ep9dfmuoi03+eOdDoz2yvK79RxbcrVEG1NO1Ph35yW102DHWKN4pk80nwCgeMmSY5L25VE4D9A==",
+      "dev": true,
+      "license": "MIT",
+      "dependencies": {
+        "array-find-index": "^1.0.2",
+        "spdx-expression-parse": "^3.0.0",
+        "spdx-ranges": "^2.0.0"
+      }
+    },
+    "node_modules/spdx-exceptions": {
+      "version": "2.5.0",
+      "resolved": "https://registry.npmjs.org/spdx-exceptions/-/spdx-exceptions-2.5.0.tgz",
+      "integrity": "sha512-PiU42r+xO4UbUS1buo3LPJkjlO7430Xn5SVAhdpzzsPHsjbYVflnnFdATgabnLude+Cqu25p6N+g2lw/PFsa4w==",
+      "dev": true,
+      "license": "CC-BY-3.0"
+    },
+    "node_modules/spdx-expression-parse": {
+      "version": "3.0.1",
+      "resolved": "https://registry.npmjs.org/spdx-expression-parse/-/spdx-expression-parse-3.0.1.tgz",
+      "integrity": "sha512-cbqHunsQWnJNE6KhVSMsMeH5H/L9EpymbzqTQ3uLwNCLZ1Q481oWaofqH7nO6V07xlXwY6PhQdQ2IedWx/ZK4Q==",
+      "dev": true,
+      "license": "MIT",
+      "dependencies": {
+        "spdx-exceptions": "^2.1.0",
+        "spdx-license-ids": "^3.0.0"
+      }
+    },
+    "node_modules/spdx-expression-validate": {
+      "version": "2.0.0",
+      "resolved": "https://registry.npmjs.org/spdx-expression-validate/-/spdx-expression-validate-2.0.0.tgz",
+      "integrity": "sha512-b3wydZLM+Tc6CFvaRDBOF9d76oGIHNCLYFeHbftFXUWjnfZWganmDmvtM5sm1cRwJc/VDBMLyGGrsLFd1vOxbg==",
+      "dev": true,
+      "license": "(MIT AND CC-BY-3.0)",
+      "dependencies": {
+        "spdx-expression-parse": "^3.0.0"
+      }
+    },
+    "node_modules/spdx-license-ids": {
+      "version": "3.0.23",
+      "resolved": "https://registry.npmjs.org/spdx-license-ids/-/spdx-license-ids-3.0.23.tgz",
+      "integrity": "sha512-CWLcCCH7VLu13TgOH+r8p1O/Znwhqv/dbb6lqWy67G+pT1kHmeD/+V36AVb/vq8QMIQwVShJ6Ssl5FPh0fuSdw==",
+      "dev": true,
+      "license": "CC0-1.0"
+    },
+    "node_modules/spdx-ranges": {
+      "version": "2.1.1",
+      "resolved": "https://registry.npmjs.org/spdx-ranges/-/spdx-ranges-2.1.1.tgz",
+      "integrity": "sha512-mcdpQFV7UDAgLpXEE/jOMqvK4LBoO0uTQg0uvXUewmEFhpiZx5yJSZITHB8w1ZahKdhfZqP5GPEOKLyEq5p8XA==",
+      "dev": true,
+      "license": "(MIT AND CC-BY-3.0)"
+    },
+    "node_modules/spdx-satisfies": {
+      "version": "5.0.1",
+      "resolved": "https://registry.npmjs.org/spdx-satisfies/-/spdx-satisfies-5.0.1.tgz",
+      "integrity": "sha512-Nwor6W6gzFp8XX4neaKQ7ChV4wmpSh2sSDemMFSzHxpTw460jxFYeOn+jq4ybnSSw/5sc3pjka9MQPouksQNpw==",
+      "dev": true,
+      "license": "MIT",
+      "dependencies": {
+        "spdx-compare": "^1.0.0",
+        "spdx-expression-parse": "^3.0.0",
+        "spdx-ranges": "^2.0.0"
+      }
+    },
+    "node_modules/stackback": {
+      "version": "0.0.2",
+      "resolved": "https://registry.npmjs.org/stackback/-/stackback-0.0.2.tgz",
+      "integrity": "sha512-1XMJE5fQo1jGH6Y/7ebnwPOBEkIEnT4QF32d5R1+VXdXveM0IBMJt8zfaxX1P3QhVwrYe+576+jkANtSS2mBbw==",
+      "dev": true,
+      "license": "MIT"
+    },
+    "node_modules/std-env": {
+      "version": "3.10.0",
+      "resolved": "https://registry.npmjs.org/std-env/-/std-env-3.10.0.tgz",
+      "integrity": "sha512-5GS12FdOZNliM5mAOxFRg7Ir0pWz8MdpYm6AY6VPkGpbA7ZzmbzNcBJQ0GPvvyWgcY7QAhCgf9Uy89I03faLkg==",
+      "dev": true,
+      "license": "MIT"
+    },
+    "node_modules/strip-indent": {
+      "version": "4.1.1",
+      "resolved": "https://registry.npmjs.org/strip-indent/-/strip-indent-4.1.1.tgz",
+      "integrity": "sha512-SlyRoSkdh1dYP0PzclLE7r0M9sgbFKKMFXpFRUMNuKhQSbC6VQIGzq3E0qsfvGJaUFJPGv6Ws1NZ/haTAjfbMA==",
+      "dev": true,
+      "license": "MIT",
+      "engines": {
+        "node": ">=12"
+      },
+      "funding": {
+        "url": "https://github.com/sponsors/sindresorhus"
+      }
+    },
+    "node_modules/strip-json-comments": {
+      "version": "3.1.1",
+      "resolved": "https://registry.npmjs.org/strip-json-comments/-/strip-json-comments-3.1.1.tgz",
+      "integrity": "sha512-6fPc+R4ihwqP6N/aIv2f1gMH8lOVtWQHoqC4yK6oSDVVocumAsfCqjkXnqiYMhmMwS/mEHLp7Vehlt3ql6lEig==",
+      "dev": true,
+      "license": "MIT",
+      "engines": {
+        "node": ">=8"
+      },
+      "funding": {
+        "url": "https://github.com/sponsors/sindresorhus"
+      }
+    },
+    "node_modules/supports-color": {
+      "version": "7.2.0",
+      "resolved": "https://registry.npmjs.org/supports-color/-/supports-color-7.2.0.tgz",
+      "integrity": "sha512-qpCAvRl9stuOHveKsn7HncJRvv501qIacKzQlO/+Lwxc9+0q2wLyv4Dfvt80/DPn2pqOBsJdDiogXGR9+OvwRw==",
+      "dev": true,
+      "license": "MIT",
+      "dependencies": {
+        "has-flag": "^4.0.0"
+      },
+      "engines": {
+        "node": ">=8"
+      }
+    },
+    "node_modules/tinybench": {
+      "version": "2.9.0",
+      "resolved": "https://registry.npmjs.org/tinybench/-/tinybench-2.9.0.tgz",
+      "integrity": "sha512-0+DUvqWMValLmha6lr4kD8iAMK1HzV0/aKnCtWb9v9641TnP/MFb7Pc2bxoxQjTXAErryXVgUOfv2YqNllqGeg==",
+      "dev": true,
+      "license": "MIT"
+    },
+    "node_modules/tinyexec": {
+      "version": "1.0.2",
+      "resolved": "https://registry.npmjs.org/tinyexec/-/tinyexec-1.0.2.tgz",
+      "integrity": "sha512-W/KYk+NFhkmsYpuHq5JykngiOCnxeVL8v8dFnqxSD8qEEdRfXk1SDM6JzNqcERbcGYj9tMrDQBYV9cjgnunFIg==",
+      "dev": true,
+      "license": "MIT",
+      "engines": {
+        "node": ">=18"
+      }
+    },
+    "node_modules/tinyglobby": {
+      "version": "0.2.15",
+      "resolved": "https://registry.npmjs.org/tinyglobby/-/tinyglobby-0.2.15.tgz",
+      "integrity": "sha512-j2Zq4NyQYG5XMST4cbs02Ak8iJUdxRM0XI5QyxXuZOzKOINmWurp3smXu3y5wDcJrptwpSjgXHzIQxR0omXljQ==",
+      "dev": true,
+      "license": "MIT",
+      "dependencies": {
+        "fdir": "^6.5.0",
+        "picomatch": "^4.0.3"
+      },
+      "engines": {
+        "node": ">=12.0.0"
+      },
+      "funding": {
+        "url": "https://github.com/sponsors/SuperchupuDev"
+      }
+    },
+    "node_modules/tinyrainbow": {
+      "version": "3.0.3",
+      "resolved": "https://registry.npmjs.org/tinyrainbow/-/tinyrainbow-3.0.3.tgz",
+      "integrity": "sha512-PSkbLUoxOFRzJYjjxHJt9xro7D+iilgMX/C9lawzVuYiIdcihh9DXmVibBe8lmcFrRi/VzlPjBxbN7rH24q8/Q==",
+      "dev": true,
+      "license": "MIT",
+      "engines": {
+        "node": ">=14.0.0"
+      }
+    },
+    "node_modules/ts-api-utils": {
+      "version": "2.4.0",
+      "resolved": "https://registry.npmjs.org/ts-api-utils/-/ts-api-utils-2.4.0.tgz",
+      "integrity": "sha512-3TaVTaAv2gTiMB35i3FiGJaRfwb3Pyn/j3m/bfAvGe8FB7CF6u+LMYqYlDh7reQf7UNvoTvdfAqHGmPGOSsPmA==",
+      "dev": true,
+      "license": "MIT",
+      "engines": {
+        "node": ">=18.12"
+      },
+      "peerDependencies": {
+        "typescript": ">=4.8.4"
+      }
+    },
+    "node_modules/tslib": {
+      "version": "2.8.1",
+      "resolved": "https://registry.npmjs.org/tslib/-/tslib-2.8.1.tgz",
+      "integrity": "sha512-oJFu94HQb+KVduSUQL7wnpmqnfmLsOA/nAh6b6EH0wCEoK0/mPeXU6c3wKDV83MkOuHPRHtSXKKU99IBazS/2w==",
+      "dev": true,
+      "license": "0BSD",
+      "optional": true
+    },
+    "node_modules/type-check": {
+      "version": "0.4.0",
+      "resolved": "https://registry.npmjs.org/type-check/-/type-check-0.4.0.tgz",
+      "integrity": "sha512-XleUoc9uwGXqjWwXaUTZAmzMcFZ5858QA2vvx1Ur5xIcixXIP+8LnFDgRplU30us6teqdlskFfu+ae4K79Ooew==",
+      "dev": true,
+      "license": "MIT",
+      "dependencies": {
+        "prelude-ls": "^1.2.1"
+      },
+      "engines": {
+        "node": ">= 0.8.0"
+      }
+    },
+    "node_modules/typescript": {
+      "version": "5.9.3",
+      "resolved": "https://registry.npmjs.org/typescript/-/typescript-5.9.3.tgz",
+      "integrity": "sha512-jl1vZzPDinLr9eUt3J/t7V6FgNEw9QjvBPdysz9KfQDD41fQrC2Y4vKQdiaUpFT4bXlb1RHhLpp8wtm6M5TgSw==",
+      "dev": true,
+      "license": "Apache-2.0",
+      "bin": {
+        "tsc": "bin/tsc",
+        "tsserver": "bin/tsserver"
+      },
+      "engines": {
+        "node": ">=14.17"
+      }
+    },
+    "node_modules/typescript-eslint": {
+      "version": "8.57.0",
+      "resolved": "https://registry.npmjs.org/typescript-eslint/-/typescript-eslint-8.57.0.tgz",
+      "integrity": "sha512-W8GcigEMEeB07xEZol8oJ26rigm3+bfPHxHvwbYUlu1fUDsGuQ7Hiskx5xGW/xM4USc9Ephe3jtv7ZYPQntHeA==",
+      "dev": true,
+      "license": "MIT",
+      "dependencies": {
+        "@typescript-eslint/eslint-plugin": "8.57.0",
+        "@typescript-eslint/parser": "8.57.0",
+        "@typescript-eslint/typescript-estree": "8.57.0",
+        "@typescript-eslint/utils": "8.57.0"
+      },
+      "engines": {
+        "node": "^18.18.0 || ^20.9.0 || >=21.1.0"
+      },
+      "funding": {
+        "type": "opencollective",
+        "url": "https://opencollective.com/typescript-eslint"
+      },
+      "peerDependencies": {
+        "eslint": "^8.57.0 || ^9.0.0 || ^10.0.0",
+        "typescript": ">=4.8.4 <6.0.0"
+      }
+    },
+    "node_modules/ufo": {
+      "version": "1.6.3",
+      "resolved": "https://registry.npmjs.org/ufo/-/ufo-1.6.3.tgz",
+      "integrity": "sha512-yDJTmhydvl5lJzBmy/hyOAA0d+aqCBuwl818haVdYCRrWV84o7YyeVm4QlVHStqNrrJSTb6jKuFAVqAFsr+K3Q==",
+      "dev": true,
+      "license": "MIT"
+    },
+    "node_modules/undici-types": {
+      "version": "7.18.2",
+      "resolved": "https://registry.npmjs.org/undici-types/-/undici-types-7.18.2.tgz",
+      "integrity": "sha512-AsuCzffGHJybSaRrmr5eHr81mwJU3kjw6M+uprWvCXiNeN9SOGwQ3Jn8jb8m3Z6izVgknn1R0FTCEAP2QrLY/w==",
+      "dev": true,
+      "license": "MIT"
+    },
+    "node_modules/unist-util-is": {
+      "version": "6.0.1",
+      "resolved": "https://registry.npmjs.org/unist-util-is/-/unist-util-is-6.0.1.tgz",
+      "integrity": "sha512-LsiILbtBETkDz8I9p1dQ0uyRUWuaQzd/cuEeS1hoRSyW5E5XGmTzlwY1OrNzzakGowI9Dr/I8HVaw4hTtnxy8g==",
+      "dev": true,
+      "license": "MIT",
+      "dependencies": {
+        "@types/unist": "^3.0.0"
+      },
+      "funding": {
+        "type": "opencollective",
+        "url": "https://opencollective.com/unified"
+      }
+    },
+    "node_modules/unist-util-stringify-position": {
+      "version": "4.0.0",
+      "resolved": "https://registry.npmjs.org/unist-util-stringify-position/-/unist-util-stringify-position-4.0.0.tgz",
+      "integrity": "sha512-0ASV06AAoKCDkS2+xw5RXJywruurpbC4JZSm7nr7MOt1ojAzvyyaO+UxZf18j8FCF6kmzCZKcAgN/yu2gm2XgQ==",
+      "dev": true,
+      "license": "MIT",
+      "dependencies": {
+        "@types/unist": "^3.0.0"
+      },
+      "funding": {
+        "type": "opencollective",
+        "url": "https://opencollective.com/unified"
+      }
+    },
+    "node_modules/unist-util-visit": {
+      "version": "5.1.0",
+      "resolved": "https://registry.npmjs.org/unist-util-visit/-/unist-util-visit-5.1.0.tgz",
+      "integrity": "sha512-m+vIdyeCOpdr/QeQCu2EzxX/ohgS8KbnPDgFni4dQsfSCtpz8UqDyY5GjRru8PDKuYn7Fq19j1CQ+nJSsGKOzg==",
+      "dev": true,
+      "license": "MIT",
+      "dependencies": {
+        "@types/unist": "^3.0.0",
+        "unist-util-is": "^6.0.0",
+        "unist-util-visit-parents": "^6.0.0"
+      },
+      "funding": {
+        "type": "opencollective",
+        "url": "https://opencollective.com/unified"
+      }
+    },
+    "node_modules/unist-util-visit-parents": {
+      "version": "6.0.2",
+      "resolved": "https://registry.npmjs.org/unist-util-visit-parents/-/unist-util-visit-parents-6.0.2.tgz",
+      "integrity": "sha512-goh1s1TBrqSqukSc8wrjwWhL0hiJxgA8m4kFxGlQ+8FYQ3C/m11FcTs4YYem7V664AhHVvgoQLk890Ssdsr2IQ==",
+      "dev": true,
+      "license": "MIT",
+      "dependencies": {
+        "@types/unist": "^3.0.0",
+        "unist-util-is": "^6.0.0"
+      },
+      "funding": {
+        "type": "opencollective",
+        "url": "https://opencollective.com/unified"
+      }
+    },
+    "node_modules/untyped": {
+      "version": "2.0.0",
+      "resolved": "https://registry.npmjs.org/untyped/-/untyped-2.0.0.tgz",
+      "integrity": "sha512-nwNCjxJTjNuLCgFr42fEak5OcLuB3ecca+9ksPFNvtfYSLpjf+iJqSIaSnIile6ZPbKYxI5k2AfXqeopGudK/g==",
+      "dev": true,
+      "license": "MIT",
+      "dependencies": {
+        "citty": "^0.1.6",
+        "defu": "^6.1.4",
+        "jiti": "^2.4.2",
+        "knitwork": "^1.2.0",
+        "scule": "^1.3.0"
+      },
+      "bin": {
+        "untyped": "dist/cli.mjs"
+      }
+    },
+    "node_modules/untyped/node_modules/citty": {
+      "version": "0.1.6",
+      "resolved": "https://registry.npmjs.org/citty/-/citty-0.1.6.tgz",
+      "integrity": "sha512-tskPPKEs8D2KPafUypv2gxwJP8h/OaJmC82QQGGDQcHvXX43xF2VDACcJVmZ0EuSxkpO9Kc4MlrA3q0+FG58AQ==",
+      "dev": true,
+      "license": "MIT",
+      "dependencies": {
+        "consola": "^3.2.3"
+      }
+    },
+    "node_modules/update-browserslist-db": {
+      "version": "1.2.3",
+      "resolved": "https://registry.npmjs.org/update-browserslist-db/-/update-browserslist-db-1.2.3.tgz",
+      "integrity": "sha512-Js0m9cx+qOgDxo0eMiFGEueWztz+d4+M3rGlmKPT+T4IS/jP4ylw3Nwpu6cpTTP8R1MAC1kF4VbdLt3ARf209w==",
+      "dev": true,
+      "funding": [
+        {
+          "type": "opencollective",
+          "url": "https://opencollective.com/browserslist"
+        },
+        {
+          "type": "tidelift",
+          "url": "https://tidelift.com/funding/github/npm/browserslist"
+        },
+        {
+          "type": "github",
+          "url": "https://github.com/sponsors/ai"
+        }
+      ],
+      "license": "MIT",
+      "dependencies": {
+        "escalade": "^3.2.0",
+        "picocolors": "^1.1.1"
+      },
+      "bin": {
+        "update-browserslist-db": "cli.js"
+      },
+      "peerDependencies": {
+        "browserslist": ">= 4.21.0"
+      }
+    },
+    "node_modules/uri-js": {
+      "version": "4.4.1",
+      "resolved": "https://registry.npmjs.org/uri-js/-/uri-js-4.4.1.tgz",
+      "integrity": "sha512-7rKUyy33Q1yc98pQ1DAmLtwX109F7TIfWlW1Ydo8Wl1ii1SeHieeh0HHfPeL2fMXK6z0s8ecKs9frCuLJvndBg==",
+      "dev": true,
+      "license": "BSD-2-Clause",
+      "dependencies": {
+        "punycode": "^2.1.0"
+      }
+    },
+    "node_modules/vite": {
+      "version": "7.3.1",
+      "resolved": "https://registry.npmjs.org/vite/-/vite-7.3.1.tgz",
+      "integrity": "sha512-w+N7Hifpc3gRjZ63vYBXA56dvvRlNWRczTdmCBBa+CotUzAPf5b7YMdMR/8CQoeYE5LX3W4wj6RYTgonm1b9DA==",
+      "dev": true,
+      "license": "MIT",
+      "dependencies": {
+        "esbuild": "^0.27.0",
+        "fdir": "^6.5.0",
+        "picomatch": "^4.0.3",
+        "postcss": "^8.5.6",
+        "rollup": "^4.43.0",
+        "tinyglobby": "^0.2.15"
+      },
+      "bin": {
+        "vite": "bin/vite.js"
+      },
+      "engines": {
+        "node": "^20.19.0 || >=22.12.0"
+      },
+      "funding": {
+        "url": "https://github.com/vitejs/vite?sponsor=1"
+      },
+      "optionalDependencies": {
+        "fsevents": "~2.3.3"
+      },
+      "peerDependencies": {
+        "@types/node": "^20.19.0 || >=22.12.0",
+        "jiti": ">=1.21.0",
+        "less": "^4.0.0",
+        "lightningcss": "^1.21.0",
+        "sass": "^1.70.0",
+        "sass-embedded": "^1.70.0",
+        "stylus": ">=0.54.8",
+        "sugarss": "^5.0.0",
+        "terser": "^5.16.0",
+        "tsx": "^4.8.1",
+        "yaml": "^2.4.2"
+      },
+      "peerDependenciesMeta": {
+        "@types/node": {
+          "optional": true
+        },
+        "jiti": {
+          "optional": true
+        },
+        "less": {
+          "optional": true
+        },
+        "lightningcss": {
+          "optional": true
+        },
+        "sass": {
+          "optional": true
+        },
+        "sass-embedded": {
+          "optional": true
+        },
+        "stylus": {
+          "optional": true
+        },
+        "sugarss": {
+          "optional": true
+        },
+        "terser": {
+          "optional": true
+        },
+        "tsx": {
+          "optional": true
+        },
+        "yaml": {
+          "optional": true
+        }
+      }
+    },
+    "node_modules/vitest": {
+      "version": "4.0.18",
+      "resolved": "https://registry.npmjs.org/vitest/-/vitest-4.0.18.tgz",
+      "integrity": "sha512-hOQuK7h0FGKgBAas7v0mSAsnvrIgAvWmRFjmzpJ7SwFHH3g1k2u37JtYwOwmEKhK6ZO3v9ggDBBm0La1LCK4uQ==",
+      "dev": true,
+      "license": "MIT",
+      "dependencies": {
+        "@vitest/expect": "4.0.18",
+        "@vitest/mocker": "4.0.18",
+        "@vitest/pretty-format": "4.0.18",
+        "@vitest/runner": "4.0.18",
+        "@vitest/snapshot": "4.0.18",
+        "@vitest/spy": "4.0.18",
+        "@vitest/utils": "4.0.18",
+        "es-module-lexer": "^1.7.0",
+        "expect-type": "^1.2.2",
+        "magic-string": "^0.30.21",
+        "obug": "^2.1.1",
+        "pathe": "^2.0.3",
+        "picomatch": "^4.0.3",
+        "std-env": "^3.10.0",
+        "tinybench": "^2.9.0",
+        "tinyexec": "^1.0.2",
+        "tinyglobby": "^0.2.15",
+        "tinyrainbow": "^3.0.3",
+        "vite": "^6.0.0 || ^7.0.0",
+        "why-is-node-running": "^2.3.0"
+      },
+      "bin": {
+        "vitest": "vitest.mjs"
+      },
+      "engines": {
+        "node": "^20.0.0 || ^22.0.0 || >=24.0.0"
+      },
+      "funding": {
+        "url": "https://opencollective.com/vitest"
+      },
+      "peerDependencies": {
+        "@edge-runtime/vm": "*",
+        "@opentelemetry/api": "^1.9.0",
+        "@types/node": "^20.0.0 || ^22.0.0 || >=24.0.0",
+        "@vitest/browser-playwright": "4.0.18",
+        "@vitest/browser-preview": "4.0.18",
+        "@vitest/browser-webdriverio": "4.0.18",
+        "@vitest/ui": "4.0.18",
+        "happy-dom": "*",
+        "jsdom": "*"
+      },
+      "peerDependenciesMeta": {
+        "@edge-runtime/vm": {
+          "optional": true
+        },
+        "@opentelemetry/api": {
+          "optional": true
+        },
+        "@types/node": {
+          "optional": true
+        },
+        "@vitest/browser-playwright": {
+          "optional": true
+        },
+        "@vitest/browser-preview": {
+          "optional": true
+        },
+        "@vitest/browser-webdriverio": {
+          "optional": true
+        },
+        "@vitest/ui": {
+          "optional": true
+        },
+        "happy-dom": {
+          "optional": true
+        },
+        "jsdom": {
+          "optional": true
+        }
+      }
+    },
+    "node_modules/which": {
+      "version": "2.0.2",
+      "resolved": "https://registry.npmjs.org/which/-/which-2.0.2.tgz",
+      "integrity": "sha512-BLI3Tl1TW3Pvl70l3yq3Y64i+awpwXqsGBYWkkqMtnbXgrMD+yj7rhW0kuEDxzJaYXGjEW5ogapKNMEKNMjibA==",
+      "dev": true,
+      "license": "ISC",
+      "dependencies": {
+        "isexe": "^2.0.0"
+      },
+      "bin": {
+        "node-which": "bin/node-which"
+      },
+      "engines": {
+        "node": ">= 8"
+      }
+    },
+    "node_modules/why-is-node-running": {
+      "version": "2.3.0",
+      "resolved": "https://registry.npmjs.org/why-is-node-running/-/why-is-node-running-2.3.0.tgz",
+      "integrity": "sha512-hUrmaWBdVDcxvYqnyh09zunKzROWjbZTiNy8dBEjkS7ehEDQibXJ7XvlmtbwuTclUiIyN+CyXQD4Vmko8fNm8w==",
+      "dev": true,
+      "license": "MIT",
+      "dependencies": {
+        "siginfo": "^2.0.0",
+        "stackback": "0.0.2"
+      },
+      "bin": {
+        "why-is-node-running": "cli.js"
+      },
+      "engines": {
+        "node": ">=8"
+      }
+    },
+    "node_modules/word-wrap": {
+      "version": "1.2.5",
+      "resolved": "https://registry.npmjs.org/word-wrap/-/word-wrap-1.2.5.tgz",
+      "integrity": "sha512-BN22B5eaMMI9UMtjrGd5g5eCYPpCPDUy0FJXbYsaT5zYxjFOckS53SQDE3pWkVoWpHXVb3BrYcEN4Twa55B5cA==",
+      "dev": true,
+      "license": "MIT",
+      "engines": {
+        "node": ">=0.10.0"
+      }
+    },
+    "node_modules/wsl-utils": {
+      "version": "0.1.0",
+      "resolved": "https://registry.npmjs.org/wsl-utils/-/wsl-utils-0.1.0.tgz",
+      "integrity": "sha512-h3Fbisa2nKGPxCpm89Hk33lBLsnaGBvctQopaBSOW/uIs6FTe1ATyAnKFJrzVs9vpGdsTe73WF3V4lIsk4Gacw==",
+      "dev": true,
+      "license": "MIT",
+      "dependencies": {
+        "is-wsl": "^3.1.0"
+      },
+      "engines": {
+        "node": ">=18"
+      },
+      "funding": {
+        "url": "https://github.com/sponsors/sindresorhus"
+      }
+    },
+    "node_modules/yocto-queue": {
+      "version": "0.1.0",
+      "resolved": "https://registry.npmjs.org/yocto-queue/-/yocto-queue-0.1.0.tgz",
+      "integrity": "sha512-rVksvsnNCdJ/ohGc6xgPwyN8eheCxsiLM8mxuE/t/mOVqJewPuO1miLpTHQiRgTKCLexL4MeAFVagts7HmNZ2Q==",
+      "dev": true,
+      "license": "MIT",
+      "engines": {
+        "node": ">=10"
+      },
+      "funding": {
+        "url": "https://github.com/sponsors/sindresorhus"
+      }
+    },
+    "node_modules/zwitch": {
+      "version": "2.0.4",
+      "resolved": "https://registry.npmjs.org/zwitch/-/zwitch-2.0.4.tgz",
+      "integrity": "sha512-bXE4cR/kVZhKZX/RjPEflHaKVhUVl85noU3v6b8apfQEc1x4A+zBxjZ4lN8LqGd6WZ3dl98pY4o717VFmoPp+A==",
+      "dev": true,
+      "license": "MIT",
+      "funding": {
+        "type": "github",
+        "url": "https://github.com/sponsors/wooorm"
+      }
+    }
+  }
+}
diff --git a/src/_parser.ts b/src/_parser.ts
index 5483e44..6da85bc 100644
--- a/src/_parser.ts
+++ b/src/_parser.ts
@@ -145,6 +145,18 @@ export function parseRawArgs<T = Record<string, any>>(
   // Apply negated flags
   for (const [name] of Object.entries(negatedFlags)) {
     (out as any)[name] = false;
+    // Propagate negation to main option if this is an alias
+    const main = aliasToMain.get(name);
+    if (main !== undefined) {
+      (out as any)[main] = false;
+    }
+    // Propagate negation to all aliases if this is a main option
+    const aliases = mainToAliases.get(name);
+    if (aliases) {
+      for (const alias of aliases) {
+        (out as any)[alias] = false;
+      }
+    }
   }
 
   // Propagate values between aliases
```

### with-lumen

**Rating: Good**

```diff
diff --git a/package-lock.json b/package-lock.json
new file mode 100644
index 0000000..2c16875
--- /dev/null
+++ b/package-lock.json
@@ -0,0 +1,6406 @@
+{
+  "name": "citty",
+  "version": "0.2.0",
+  "lockfileVersion": 3,
+  "requires": true,
+  "packages": {
+    "": {
+      "name": "citty",
+      "version": "0.2.0",
+      "license": "MIT",
+      "devDependencies": {
+        "@types/node": "^25.0.9",
+        "@vitest/coverage-v8": "^4.0.17",
+        "automd": "^0.4.2",
+        "changelogen": "^0.6.2",
+        "eslint": "^9.39.2",
+        "eslint-config-unjs": "^0.6.2",
+        "obuild": "^0.4.18",
+        "prettier": "^3.8.0",
+        "scule": "^1.3.0",
+        "typescript": "^5.9.3",
+        "vitest": "^4.0.17"
+      }
+    },
+    "node_modules/@babel/generator": {
+      "version": "8.0.0-rc.2",
+      "resolved": "https://registry.npmjs.org/@babel/generator/-/generator-8.0.0-rc.2.tgz",
+      "integrity": "sha512-oCQ1IKPwkzCeJzAPb7Fv8rQ9k5+1sG8mf2uoHiMInPYvkRfrDJxbTIbH51U+jstlkghus0vAi3EBvkfvEsYNLQ==",
+      "dev": true,
+      "license": "MIT",
+      "dependencies": {
+        "@babel/parser": "^8.0.0-rc.2",
+        "@babel/types": "^8.0.0-rc.2",
+        "@jridgewell/gen-mapping": "^0.3.12",
+        "@jridgewell/trace-mapping": "^0.3.28",
+        "@types/jsesc": "^2.5.0",
+        "jsesc": "^3.0.2"
+      },
+      "engines": {
+        "node": "^20.19.0 || >=22.12.0"
+      }
+    },
+    "node_modules/@babel/generator/node_modules/@babel/helper-string-parser": {
+      "version": "8.0.0-rc.2",
+      "resolved": "https://registry.npmjs.org/@babel/helper-string-parser/-/helper-string-parser-8.0.0-rc.2.tgz",
+      "integrity": "sha512-noLx87RwlBEMrTzncWd/FvTxoJ9+ycHNg0n8yyYydIoDsLZuxknKgWRJUqcrVkNrJ74uGyhWQzQaS3q8xfGAhQ==",
+      "dev": true,
+      "license": "MIT",
+      "engines": {
+        "node": "^20.19.0 || >=22.12.0"
+      }
+    },
+    "node_modules/@babel/generator/node_modules/@babel/helper-validator-identifier": {
+      "version": "8.0.0-rc.2",
+      "resolved": "https://registry.npmjs.org/@babel/helper-validator-identifier/-/helper-validator-identifier-8.0.0-rc.2.tgz",
+      "integrity": "sha512-xExUBkuXWJjVuIbO7z6q7/BA9bgfJDEhVL0ggrggLMbg0IzCUWGT1hZGE8qUH7Il7/RD/a6cZ3AAFrrlp1LF/A==",
+      "dev": true,
+      "license": "MIT",
+      "engines": {
+        "node": "^20.19.0 || >=22.12.0"
+      }
+    },
+    "node_modules/@babel/generator/node_modules/@babel/parser": {
+      "version": "8.0.0-rc.2",
+      "resolved": "https://registry.npmjs.org/@babel/parser/-/parser-8.0.0-rc.2.tgz",
+      "integrity": "sha512-29AhEtcq4x8Dp3T72qvUMZHx0OMXCj4Jy/TEReQa+KWLln524Cj1fWb3QFi0l/xSpptQBR6y9RNEXuxpFvwiUQ==",
+      "dev": true,
+      "license": "MIT",
+      "dependencies": {
+        "@babel/types": "^8.0.0-rc.2"
+      },
+      "bin": {
+        "parser": "bin/babel-parser.js"
+      },
+      "engines": {
+        "node": "^20.19.0 || >=22.12.0"
+      }
+    },
+    "node_modules/@babel/generator/node_modules/@babel/types": {
+      "version": "8.0.0-rc.2",
+      "resolved": "https://registry.npmjs.org/@babel/types/-/types-8.0.0-rc.2.tgz",
+      "integrity": "sha512-91gAaWRznDwSX4E2tZ1YjBuIfnQVOFDCQ2r0Toby0gu4XEbyF623kXLMA8d4ZbCu+fINcrudkmEcwSUHgDDkNw==",
+      "dev": true,
+      "license": "MIT",
+      "dependencies": {
+        "@babel/helper-string-parser": "^8.0.0-rc.2",
+        "@babel/helper-validator-identifier": "^8.0.0-rc.2"
+      },
+      "engines": {
+        "node": "^20.19.0 || >=22.12.0"
+      }
+    },
+    "node_modules/@babel/helper-string-parser": {
+      "version": "7.27.1",
+      "resolved": "https://registry.npmjs.org/@babel/helper-string-parser/-/helper-string-parser-7.27.1.tgz",
+      "integrity": "sha512-qMlSxKbpRlAridDExk92nSobyDdpPijUq2DW6oDnUqd0iOGxmQjyqhMIihI9+zv4LPyZdRje2cavWPbCbWm3eA==",
+      "dev": true,
+      "license": "MIT",
+      "engines": {
+        "node": ">=6.9.0"
+      }
+    },
+    "node_modules/@babel/helper-validator-identifier": {
+      "version": "7.28.5",
+      "resolved": "https://registry.npmjs.org/@babel/helper-validator-identifier/-/helper-validator-identifier-7.28.5.tgz",
+      "integrity": "sha512-qSs4ifwzKJSV39ucNjsvc6WVHs6b7S03sOh2OcHF9UHfVPqWWALUsNUVzhSBiItjRZoLHx7nIarVjqKVusUZ1Q==",
+      "dev": true,
+      "license": "MIT",
+      "engines": {
+        "node": ">=6.9.0"
+      }
+    },
+    "node_modules/@babel/parser": {
+      "version": "7.29.0",
+      "resolved": "https://registry.npmjs.org/@babel/parser/-/parser-7.29.0.tgz",
+      "integrity": "sha512-IyDgFV5GeDUVX4YdF/3CPULtVGSXXMLh1xVIgdCgxApktqnQV0r7/8Nqthg+8YLGaAtdyIlo2qIdZrbCv4+7ww==",
+      "dev": true,
+      "license": "MIT",
+      "dependencies": {
+        "@babel/types": "^7.29.0"
+      },
+      "bin": {
+        "parser": "bin/babel-parser.js"
+      },
+      "engines": {
+        "node": ">=6.0.0"
+      }
+    },
+    "node_modules/@babel/runtime": {
+      "version": "7.28.6",
+      "resolved": "https://registry.npmjs.org/@babel/runtime/-/runtime-7.28.6.tgz",
+      "integrity": "sha512-05WQkdpL9COIMz4LjTxGpPNCdlpyimKppYNoJ5Di5EUObifl8t4tuLuUBBZEpoLYOmfvIWrsp9fCl0HoPRVTdA==",
+      "dev": true,
+      "license": "MIT",
+      "engines": {
+        "node": ">=6.9.0"
+      }
+    },
+    "node_modules/@babel/types": {
+      "version": "7.29.0",
+      "resolved": "https://registry.npmjs.org/@babel/types/-/types-7.29.0.tgz",
+      "integrity": "sha512-LwdZHpScM4Qz8Xw2iKSzS+cfglZzJGvofQICy7W7v4caru4EaAmyUuO6BGrbyQ2mYV11W0U8j5mBhd14dd3B0A==",
+      "dev": true,
+      "license": "MIT",
+      "dependencies": {
+        "@babel/helper-string-parser": "^7.27.1",
+        "@babel/helper-validator-identifier": "^7.28.5"
+      },
+      "engines": {
+        "node": ">=6.9.0"
+      }
+    },
+    "node_modules/@bcoe/v8-coverage": {
+      "version": "1.0.2",
+      "resolved": "https://registry.npmjs.org/@bcoe/v8-coverage/-/v8-coverage-1.0.2.tgz",
+      "integrity": "sha512-6zABk/ECA/QYSCQ1NGiVwwbQerUCZ+TQbp64Q3AgmfNvurHH0j8TtXa1qbShXA6qqkpAj4V5W8pP6mLe1mcMqA==",
+      "dev": true,
+      "license": "MIT",
+      "engines": {
+        "node": ">=18"
+      }
+    },
+    "node_modules/@emnapi/core": {
+      "version": "1.8.1",
+      "resolved": "https://registry.npmjs.org/@emnapi/core/-/core-1.8.1.tgz",
+      "integrity": "sha512-AvT9QFpxK0Zd8J0jopedNm+w/2fIzvtPKPjqyw9jwvBaReTTqPBk9Hixaz7KbjimP+QNz605/XnjFcDAL2pqBg==",
+      "dev": true,
+      "license": "MIT",
+      "optional": true,
+      "dependencies": {
+        "@emnapi/wasi-threads": "1.1.0",
+        "tslib": "^2.4.0"
+      }
+    },
+    "node_modules/@emnapi/runtime": {
+      "version": "1.8.1",
+      "resolved": "https://registry.npmjs.org/@emnapi/runtime/-/runtime-1.8.1.tgz",
+      "integrity": "sha512-mehfKSMWjjNol8659Z8KxEMrdSJDDot5SXMq00dM8BN4o+CLNXQ0xH2V7EchNHV4RmbZLmmPdEaXZc5H2FXmDg==",
+      "dev": true,
+      "license": "MIT",
+      "optional": true,
+      "dependencies": {
+        "tslib": "^2.4.0"
+      }
+    },
+    "node_modules/@emnapi/wasi-threads": {
+      "version": "1.1.0",
+      "resolved": "https://registry.npmjs.org/@emnapi/wasi-threads/-/wasi-threads-1.1.0.tgz",
+      "integrity": "sha512-WI0DdZ8xFSbgMjR1sFsKABJ/C5OnRrjT06JXbZKexJGrDuPTzZdDYfFlsgcCXCyf+suG5QU2e/y1Wo2V/OapLQ==",
+      "dev": true,
+      "license": "MIT",
+      "optional": true,
+      "dependencies": {
+        "tslib": "^2.4.0"
+      }
+    },
+    "node_modules/@esbuild/aix-ppc64": {
+      "version": "0.27.3",
+      "resolved": "https://registry.npmjs.org/@esbuild/aix-ppc64/-/aix-ppc64-0.27.3.tgz",
+      "integrity": "sha512-9fJMTNFTWZMh5qwrBItuziu834eOCUcEqymSH7pY+zoMVEZg3gcPuBNxH1EvfVYe9h0x/Ptw8KBzv7qxb7l8dg==",
+      "cpu": [
+        "ppc64"
+      ],
+      "dev": true,
+      "license": "MIT",
+      "optional": true,
+      "os": [
+        "aix"
+      ],
+      "engines": {
+        "node": ">=18"
+      }
+    },
+    "node_modules/@esbuild/android-arm": {
+      "version": "0.27.3",
+      "resolved": "https://registry.npmjs.org/@esbuild/android-arm/-/android-arm-0.27.3.tgz",
+      "integrity": "sha512-i5D1hPY7GIQmXlXhs2w8AWHhenb00+GxjxRncS2ZM7YNVGNfaMxgzSGuO8o8SJzRc/oZwU2bcScvVERk03QhzA==",
+      "cpu": [
+        "arm"
+      ],
+      "dev": true,
+      "license": "MIT",
+      "optional": true,
+      "os": [
+        "android"
+      ],
+      "engines": {
+        "node": ">=18"
+      }
+    },
+    "node_modules/@esbuild/android-arm64": {
+      "version": "0.27.3",
+      "resolved": "https://registry.npmjs.org/@esbuild/android-arm64/-/android-arm64-0.27.3.tgz",
+      "integrity": "sha512-YdghPYUmj/FX2SYKJ0OZxf+iaKgMsKHVPF1MAq/P8WirnSpCStzKJFjOjzsW0QQ7oIAiccHdcqjbHmJxRb/dmg==",
+      "cpu": [
+        "arm64"
+      ],
+      "dev": true,
+      "license": "MIT",
+      "optional": true,
+      "os": [
+        "android"
+      ],
+      "engines": {
+        "node": ">=18"
+      }
+    },
+    "node_modules/@esbuild/android-x64": {
+      "version": "0.27.3",
+      "resolved": "https://registry.npmjs.org/@esbuild/android-x64/-/android-x64-0.27.3.tgz",
+      "integrity": "sha512-IN/0BNTkHtk8lkOM8JWAYFg4ORxBkZQf9zXiEOfERX/CzxW3Vg1ewAhU7QSWQpVIzTW+b8Xy+lGzdYXV6UZObQ==",
+      "cpu": [
+        "x64"
+      ],
+      "dev": true,
+      "license": "MIT",
+      "optional": true,
+      "os": [
+        "android"
+      ],
+      "engines": {
+        "node": ">=18"
+      }
+    },
+    "node_modules/@esbuild/darwin-arm64": {
+      "version": "0.27.3",
+      "resolved": "https://registry.npmjs.org/@esbuild/darwin-arm64/-/darwin-arm64-0.27.3.tgz",
+      "integrity": "sha512-Re491k7ByTVRy0t3EKWajdLIr0gz2kKKfzafkth4Q8A5n1xTHrkqZgLLjFEHVD+AXdUGgQMq+Godfq45mGpCKg==",
+      "cpu": [
+        "arm64"
+      ],
+      "dev": true,
+      "license": "MIT",
+      "optional": true,
+      "os": [
+        "darwin"
+      ],
+      "engines": {
+        "node": ">=18"
+      }
+    },
+    "node_modules/@esbuild/darwin-x64": {
+      "version": "0.27.3",
+      "resolved": "https://registry.npmjs.org/@esbuild/darwin-x64/-/darwin-x64-0.27.3.tgz",
+      "integrity": "sha512-vHk/hA7/1AckjGzRqi6wbo+jaShzRowYip6rt6q7VYEDX4LEy1pZfDpdxCBnGtl+A5zq8iXDcyuxwtv3hNtHFg==",
+      "cpu": [
+        "x64"
+      ],
+      "dev": true,
+      "license": "MIT",
+      "optional": true,
+      "os": [
+        "darwin"
+      ],
+      "engines": {
+        "node": ">=18"
+      }
+    },
+    "node_modules/@esbuild/freebsd-arm64": {
+      "version": "0.27.3",
+      "resolved": "https://registry.npmjs.org/@esbuild/freebsd-arm64/-/freebsd-arm64-0.27.3.tgz",
+      "integrity": "sha512-ipTYM2fjt3kQAYOvo6vcxJx3nBYAzPjgTCk7QEgZG8AUO3ydUhvelmhrbOheMnGOlaSFUoHXB6un+A7q4ygY9w==",
+      "cpu": [
+        "arm64"
+      ],
+      "dev": true,
+      "license": "MIT",
+      "optional": true,
+      "os": [
+        "freebsd"
+      ],
+      "engines": {
+        "node": ">=18"
+      }
+    },
+    "node_modules/@esbuild/freebsd-x64": {
+      "version": "0.27.3",
+      "resolved": "https://registry.npmjs.org/@esbuild/freebsd-x64/-/freebsd-x64-0.27.3.tgz",
+      "integrity": "sha512-dDk0X87T7mI6U3K9VjWtHOXqwAMJBNN2r7bejDsc+j03SEjtD9HrOl8gVFByeM0aJksoUuUVU9TBaZa2rgj0oA==",
+      "cpu": [
+        "x64"
+      ],
+      "dev": true,
+      "license": "MIT",
+      "optional": true,
+      "os": [
+        "freebsd"
+      ],
+      "engines": {
+        "node": ">=18"
+      }
+    },
+    "node_modules/@esbuild/linux-arm": {
+      "version": "0.27.3",
+      "resolved": "https://registry.npmjs.org/@esbuild/linux-arm/-/linux-arm-0.27.3.tgz",
+      "integrity": "sha512-s6nPv2QkSupJwLYyfS+gwdirm0ukyTFNl3KTgZEAiJDd+iHZcbTPPcWCcRYH+WlNbwChgH2QkE9NSlNrMT8Gfw==",
+      "cpu": [
+        "arm"
+      ],
+      "dev": true,
+      "license": "MIT",
+      "optional": true,
+      "os": [
+        "linux"
+      ],
+      "engines": {
+        "node": ">=18"
+      }
+    },
+    "node_modules/@esbuild/linux-arm64": {
+      "version": "0.27.3",
+      "resolved": "https://registry.npmjs.org/@esbuild/linux-arm64/-/linux-arm64-0.27.3.tgz",
+      "integrity": "sha512-sZOuFz/xWnZ4KH3YfFrKCf1WyPZHakVzTiqji3WDc0BCl2kBwiJLCXpzLzUBLgmp4veFZdvN5ChW4Eq/8Fc2Fg==",
+      "cpu": [
+        "arm64"
+      ],
+      "dev": true,
+      "license": "MIT",
+      "optional": true,
+      "os": [
+        "linux"
+      ],
+      "engines": {
+        "node": ">=18"
+      }
+    },
+    "node_modules/@esbuild/linux-ia32": {
+      "version": "0.27.3",
+      "resolved": "https://registry.npmjs.org/@esbuild/linux-ia32/-/linux-ia32-0.27.3.tgz",
+      "integrity": "sha512-yGlQYjdxtLdh0a3jHjuwOrxQjOZYD/C9PfdbgJJF3TIZWnm/tMd/RcNiLngiu4iwcBAOezdnSLAwQDPqTmtTYg==",
+      "cpu": [
+        "ia32"
+      ],
+      "dev": true,
+      "license": "MIT",
+      "optional": true,
+      "os": [
+        "linux"
+      ],
+      "engines": {
+        "node": ">=18"
+      }
+    },
+    "node_modules/@esbuild/linux-loong64": {
+      "version": "0.27.3",
+      "resolved": "https://registry.npmjs.org/@esbuild/linux-loong64/-/linux-loong64-0.27.3.tgz",
+      "integrity": "sha512-WO60Sn8ly3gtzhyjATDgieJNet/KqsDlX5nRC5Y3oTFcS1l0KWba+SEa9Ja1GfDqSF1z6hif/SkpQJbL63cgOA==",
+      "cpu": [
+        "loong64"
+      ],
+      "dev": true,
+      "license": "MIT",
+      "optional": true,
+      "os": [
+        "linux"
+      ],
+      "engines": {
+        "node": ">=18"
+      }
+    },
+    "node_modules/@esbuild/linux-mips64el": {
+      "version": "0.27.3",
+      "resolved": "https://registry.npmjs.org/@esbuild/linux-mips64el/-/linux-mips64el-0.27.3.tgz",
+      "integrity": "sha512-APsymYA6sGcZ4pD6k+UxbDjOFSvPWyZhjaiPyl/f79xKxwTnrn5QUnXR5prvetuaSMsb4jgeHewIDCIWljrSxw==",
+      "cpu": [
+        "mips64el"
+      ],
+      "dev": true,
+      "license": "MIT",
+      "optional": true,
+      "os": [
+        "linux"
+      ],
+      "engines": {
+        "node": ">=18"
+      }
+    },
+    "node_modules/@esbuild/linux-ppc64": {
+      "version": "0.27.3",
+      "resolved": "https://registry.npmjs.org/@esbuild/linux-ppc64/-/linux-ppc64-0.27.3.tgz",
+      "integrity": "sha512-eizBnTeBefojtDb9nSh4vvVQ3V9Qf9Df01PfawPcRzJH4gFSgrObw+LveUyDoKU3kxi5+9RJTCWlj4FjYXVPEA==",
+      "cpu": [
+        "ppc64"
+      ],
+      "dev": true,
+      "license": "MIT",
+      "optional": true,
+      "os": [
+        "linux"
+      ],
+      "engines": {
+        "node": ">=18"
+      }
+    },
+    "node_modules/@esbuild/linux-riscv64": {
+      "version": "0.27.3",
+      "resolved": "https://registry.npmjs.org/@esbuild/linux-riscv64/-/linux-riscv64-0.27.3.tgz",
+      "integrity": "sha512-3Emwh0r5wmfm3ssTWRQSyVhbOHvqegUDRd0WhmXKX2mkHJe1SFCMJhagUleMq+Uci34wLSipf8Lagt4LlpRFWQ==",
+      "cpu": [
+        "riscv64"
+      ],
+      "dev": true,
+      "license": "MIT",
+      "optional": true,
+      "os": [
+        "linux"
+      ],
+      "engines": {
+        "node": ">=18"
+      }
+    },
+    "node_modules/@esbuild/linux-s390x": {
+      "version": "0.27.3",
+      "resolved": "https://registry.npmjs.org/@esbuild/linux-s390x/-/linux-s390x-0.27.3.tgz",
+      "integrity": "sha512-pBHUx9LzXWBc7MFIEEL0yD/ZVtNgLytvx60gES28GcWMqil8ElCYR4kvbV2BDqsHOvVDRrOxGySBM9Fcv744hw==",
+      "cpu": [
+        "s390x"
+      ],
+      "dev": true,
+      "license": "MIT",
+      "optional": true,
+      "os": [
+        "linux"
+      ],
+      "engines": {
+        "node": ">=18"
+      }
+    },
+    "node_modules/@esbuild/linux-x64": {
+      "version": "0.27.3",
+      "resolved": "https://registry.npmjs.org/@esbuild/linux-x64/-/linux-x64-0.27.3.tgz",
+      "integrity": "sha512-Czi8yzXUWIQYAtL/2y6vogER8pvcsOsk5cpwL4Gk5nJqH5UZiVByIY8Eorm5R13gq+DQKYg0+JyQoytLQas4dA==",
+      "cpu": [
+        "x64"
+      ],
+      "dev": true,
+      "license": "MIT",
+      "optional": true,
+      "os": [
+        "linux"
+      ],
+      "engines": {
+        "node": ">=18"
+      }
+    },
+    "node_modules/@esbuild/netbsd-arm64": {
+      "version": "0.27.3",
+      "resolved": "https://registry.npmjs.org/@esbuild/netbsd-arm64/-/netbsd-arm64-0.27.3.tgz",
+      "integrity": "sha512-sDpk0RgmTCR/5HguIZa9n9u+HVKf40fbEUt+iTzSnCaGvY9kFP0YKBWZtJaraonFnqef5SlJ8/TiPAxzyS+UoA==",
+      "cpu": [
+        "arm64"
+      ],
+      "dev": true,
+      "license": "MIT",
+      "optional": true,
+      "os": [
+        "netbsd"
+      ],
+      "engines": {
+        "node": ">=18"
+      }
+    },
+    "node_modules/@esbuild/netbsd-x64": {
+      "version": "0.27.3",
+      "resolved": "https://registry.npmjs.org/@esbuild/netbsd-x64/-/netbsd-x64-0.27.3.tgz",
+      "integrity": "sha512-P14lFKJl/DdaE00LItAukUdZO5iqNH7+PjoBm+fLQjtxfcfFE20Xf5CrLsmZdq5LFFZzb5JMZ9grUwvtVYzjiA==",
+      "cpu": [
+        "x64"
+      ],
+      "dev": true,
+      "license": "MIT",
+      "optional": true,
+      "os": [
+        "netbsd"
+      ],
+      "engines": {
+        "node": ">=18"
+      }
+    },
+    "node_modules/@esbuild/openbsd-arm64": {
+      "version": "0.27.3",
+      "resolved": "https://registry.npmjs.org/@esbuild/openbsd-arm64/-/openbsd-arm64-0.27.3.tgz",
+      "integrity": "sha512-AIcMP77AvirGbRl/UZFTq5hjXK+2wC7qFRGoHSDrZ5v5b8DK/GYpXW3CPRL53NkvDqb9D+alBiC/dV0Fb7eJcw==",
+      "cpu": [
+        "arm64"
+      ],
+      "dev": true,
+      "license": "MIT",
+      "optional": true,
+      "os": [
+        "openbsd"
+      ],
+      "engines": {
+        "node": ">=18"
+      }
+    },
+    "node_modules/@esbuild/openbsd-x64": {
+      "version": "0.27.3",
+      "resolved": "https://registry.npmjs.org/@esbuild/openbsd-x64/-/openbsd-x64-0.27.3.tgz",
+      "integrity": "sha512-DnW2sRrBzA+YnE70LKqnM3P+z8vehfJWHXECbwBmH/CU51z6FiqTQTHFenPlHmo3a8UgpLyH3PT+87OViOh1AQ==",
+      "cpu": [
+        "x64"
+      ],
+      "dev": true,
+      "license": "MIT",
+      "optional": true,
+      "os": [
+        "openbsd"
+      ],
+      "engines": {
+        "node": ">=18"
+      }
+    },
+    "node_modules/@esbuild/openharmony-arm64": {
+      "version": "0.27.3",
+      "resolved": "https://registry.npmjs.org/@esbuild/openharmony-arm64/-/openharmony-arm64-0.27.3.tgz",
+      "integrity": "sha512-NinAEgr/etERPTsZJ7aEZQvvg/A6IsZG/LgZy+81wON2huV7SrK3e63dU0XhyZP4RKGyTm7aOgmQk0bGp0fy2g==",
+      "cpu": [
+        "arm64"
+      ],
+      "dev": true,
+      "license": "MIT",
+      "optional": true,
+      "os": [
+        "openharmony"
+      ],
+      "engines": {
+        "node": ">=18"
+      }
+    },
+    "node_modules/@esbuild/sunos-x64": {
+      "version": "0.27.3",
+      "resolved": "https://registry.npmjs.org/@esbuild/sunos-x64/-/sunos-x64-0.27.3.tgz",
+      "integrity": "sha512-PanZ+nEz+eWoBJ8/f8HKxTTD172SKwdXebZ0ndd953gt1HRBbhMsaNqjTyYLGLPdoWHy4zLU7bDVJztF5f3BHA==",
+      "cpu": [
+        "x64"
+      ],
+      "dev": true,
+      "license": "MIT",
+      "optional": true,
+      "os": [
+        "sunos"
+      ],
+      "engines": {
+        "node": ">=18"
+      }
+    },
+    "node_modules/@esbuild/win32-arm64": {
+      "version": "0.27.3",
+      "resolved": "https://registry.npmjs.org/@esbuild/win32-arm64/-/win32-arm64-0.27.3.tgz",
+      "integrity": "sha512-B2t59lWWYrbRDw/tjiWOuzSsFh1Y/E95ofKz7rIVYSQkUYBjfSgf6oeYPNWHToFRr2zx52JKApIcAS/D5TUBnA==",
+      "cpu": [
+        "arm64"
+      ],
+      "dev": true,
+      "license": "MIT",
+      "optional": true,
+      "os": [
+        "win32"
+      ],
+      "engines": {
+        "node": ">=18"
+      }
+    },
+    "node_modules/@esbuild/win32-ia32": {
+      "version": "0.27.3",
+      "resolved": "https://registry.npmjs.org/@esbuild/win32-ia32/-/win32-ia32-0.27.3.tgz",
+      "integrity": "sha512-QLKSFeXNS8+tHW7tZpMtjlNb7HKau0QDpwm49u0vUp9y1WOF+PEzkU84y9GqYaAVW8aH8f3GcBck26jh54cX4Q==",
+      "cpu": [
+        "ia32"
+      ],
+      "dev": true,
+      "license": "MIT",
+      "optional": true,
+      "os": [
+        "win32"
+      ],
+      "engines": {
+        "node": ">=18"
+      }
+    },
+    "node_modules/@esbuild/win32-x64": {
+      "version": "0.27.3",
+      "resolved": "https://registry.npmjs.org/@esbuild/win32-x64/-/win32-x64-0.27.3.tgz",
+      "integrity": "sha512-4uJGhsxuptu3OcpVAzli+/gWusVGwZZHTlS63hh++ehExkVT8SgiEf7/uC/PclrPPkLhZqGgCTjd0VWLo6xMqA==",
+      "cpu": [
+        "x64"
+      ],
+      "dev": true,
+      "license": "MIT",
+      "optional": true,
+      "os": [
+        "win32"
+      ],
+      "engines": {
+        "node": ">=18"
+      }
+    },
+    "node_modules/@eslint-community/eslint-utils": {
+      "version": "4.9.1",
+      "resolved": "https://registry.npmjs.org/@eslint-community/eslint-utils/-/eslint-utils-4.9.1.tgz",
+      "integrity": "sha512-phrYmNiYppR7znFEdqgfWHXR6NCkZEK7hwWDHZUjit/2/U0r6XvkDl0SYnoM51Hq7FhCGdLDT6zxCCOY1hexsQ==",
+      "dev": true,
+      "license": "MIT",
+      "dependencies": {
+        "eslint-visitor-keys": "^3.4.3"
+      },
+      "engines": {
+        "node": "^12.22.0 || ^14.17.0 || >=16.0.0"
+      },
+      "funding": {
+        "url": "https://opencollective.com/eslint"
+      },
+      "peerDependencies": {
+        "eslint": "^6.0.0 || ^7.0.0 || >=8.0.0"
+      }
+    },
+    "node_modules/@eslint-community/eslint-utils/node_modules/eslint-visitor-keys": {
+      "version": "3.4.3",
+      "resolved": "https://registry.npmjs.org/eslint-visitor-keys/-/eslint-visitor-keys-3.4.3.tgz",
+      "integrity": "sha512-wpc+LXeiyiisxPlEkUzU6svyS1frIO3Mgxj1fdy7Pm8Ygzguax2N3Fa/D/ag1WqbOprdI+uY6wMUl8/a2G+iag==",
+      "dev": true,
+      "license": "Apache-2.0",
+      "engines": {
+        "node": "^12.22.0 || ^14.17.0 || >=16.0.0"
+      },
+      "funding": {
+        "url": "https://opencollective.com/eslint"
+      }
+    },
+    "node_modules/@eslint-community/regexpp": {
+      "version": "4.12.2",
+      "resolved": "https://registry.npmjs.org/@eslint-community/regexpp/-/regexpp-4.12.2.tgz",
+      "integrity": "sha512-EriSTlt5OC9/7SXkRSCAhfSxxoSUgBm33OH+IkwbdpgoqsSsUg7y3uh+IICI/Qg4BBWr3U2i39RpmycbxMq4ew==",
+      "dev": true,
+      "license": "MIT",
+      "engines": {
+        "node": "^12.0.0 || ^14.0.0 || >=16.0.0"
+      }
+    },
+    "node_modules/@eslint/config-array": {
+      "version": "0.21.2",
+      "resolved": "https://registry.npmjs.org/@eslint/config-array/-/config-array-0.21.2.tgz",
+      "integrity": "sha512-nJl2KGTlrf9GjLimgIru+V/mzgSK0ABCDQRvxw5BjURL7WfH5uoWmizbH7QB6MmnMBd8cIC9uceWnezL1VZWWw==",
+      "dev": true,
+      "license": "Apache-2.0",
+      "dependencies": {
+        "@eslint/object-schema": "^2.1.7",
+        "debug": "^4.3.1",
+        "minimatch": "^3.1.5"
+      },
+      "engines": {
+        "node": "^18.18.0 || ^20.9.0 || >=21.1.0"
+      }
+    },
+    "node_modules/@eslint/config-helpers": {
+      "version": "0.4.2",
+      "resolved": "https://registry.npmjs.org/@eslint/config-helpers/-/config-helpers-0.4.2.tgz",
+      "integrity": "sha512-gBrxN88gOIf3R7ja5K9slwNayVcZgK6SOUORm2uBzTeIEfeVaIhOpCtTox3P6R7o2jLFwLFTLnC7kU/RGcYEgw==",
+      "dev": true,
+      "license": "Apache-2.0",
+      "dependencies": {
+        "@eslint/core": "^0.17.0"
+      },
+      "engines": {
+        "node": "^18.18.0 || ^20.9.0 || >=21.1.0"
+      }
+    },
+    "node_modules/@eslint/core": {
+      "version": "0.17.0",
+      "resolved": "https://registry.npmjs.org/@eslint/core/-/core-0.17.0.tgz",
+      "integrity": "sha512-yL/sLrpmtDaFEiUj1osRP4TI2MDz1AddJL+jZ7KSqvBuliN4xqYY54IfdN8qD8Toa6g1iloph1fxQNkjOxrrpQ==",
+      "dev": true,
+      "license": "Apache-2.0",
+      "dependencies": {
+        "@types/json-schema": "^7.0.15"
+      },
+      "engines": {
+        "node": "^18.18.0 || ^20.9.0 || >=21.1.0"
+      }
+    },
+    "node_modules/@eslint/eslintrc": {
+      "version": "3.3.5",
+      "resolved": "https://registry.npmjs.org/@eslint/eslintrc/-/eslintrc-3.3.5.tgz",
+      "integrity": "sha512-4IlJx0X0qftVsN5E+/vGujTRIFtwuLbNsVUe7TO6zYPDR1O6nFwvwhIKEKSrl6dZchmYBITazxKoUYOjdtjlRg==",
+      "dev": true,
+      "license": "MIT",
+      "dependencies": {
+        "ajv": "^6.14.0",
+        "debug": "^4.3.2",
+        "espree": "^10.0.1",
+        "globals": "^14.0.0",
+        "ignore": "^5.2.0",
+        "import-fresh": "^3.2.1",
+        "js-yaml": "^4.1.1",
+        "minimatch": "^3.1.5",
+        "strip-json-comments": "^3.1.1"
+      },
+      "engines": {
+        "node": "^18.18.0 || ^20.9.0 || >=21.1.0"
+      },
+      "funding": {
+        "url": "https://opencollective.com/eslint"
+      }
+    },
+    "node_modules/@eslint/js": {
+      "version": "9.39.4",
+      "resolved": "https://registry.npmjs.org/@eslint/js/-/js-9.39.4.tgz",
+      "integrity": "sha512-nE7DEIchvtiFTwBw4Lfbu59PG+kCofhjsKaCWzxTpt4lfRjRMqG6uMBzKXuEcyXhOHoUp9riAm7/aWYGhXZ9cw==",
+      "dev": true,
+      "license": "MIT",
+      "engines": {
+        "node": "^18.18.0 || ^20.9.0 || >=21.1.0"
+      },
+      "funding": {
+        "url": "https://eslint.org/donate"
+      }
+    },
+    "node_modules/@eslint/markdown": {
+      "version": "7.5.1",
+      "resolved": "https://registry.npmjs.org/@eslint/markdown/-/markdown-7.5.1.tgz",
+      "integrity": "sha512-R8uZemG9dKTbru/DQRPblbJyXpObwKzo8rv1KYGGuPUPtjM4LXBYM9q5CIZAComzZupws3tWbDwam5AFpPLyJQ==",
+      "dev": true,
+      "license": "MIT",
+      "workspaces": [
+        "examples/*"
+      ],
+      "dependencies": {
+        "@eslint/core": "^0.17.0",
+        "@eslint/plugin-kit": "^0.4.1",
+        "github-slugger": "^2.0.0",
+        "mdast-util-from-markdown": "^2.0.2",
+        "mdast-util-frontmatter": "^2.0.1",
+        "mdast-util-gfm": "^3.1.0",
+        "micromark-extension-frontmatter": "^2.0.0",
+        "micromark-extension-gfm": "^3.0.0",
+        "micromark-util-normalize-identifier": "^2.0.1"
+      },
+      "engines": {
+        "node": "^18.18.0 || ^20.9.0 || >=21.1.0"
+      }
+    },
+    "node_modules/@eslint/object-schema": {
+      "version": "2.1.7",
+      "resolved": "https://registry.npmjs.org/@eslint/object-schema/-/object-schema-2.1.7.tgz",
+      "integrity": "sha512-VtAOaymWVfZcmZbp6E2mympDIHvyjXs/12LqWYjVw6qjrfF+VK+fyG33kChz3nnK+SU5/NeHOqrTEHS8sXO3OA==",
+      "dev": true,
+      "license": "Apache-2.0",
+      "engines": {
+        "node": "^18.18.0 || ^20.9.0 || >=21.1.0"
+      }
+    },
+    "node_modules/@eslint/plugin-kit": {
+      "version": "0.4.1",
+      "resolved": "https://registry.npmjs.org/@eslint/plugin-kit/-/plugin-kit-0.4.1.tgz",
+      "integrity": "sha512-43/qtrDUokr7LJqoF2c3+RInu/t4zfrpYdoSDfYyhg52rwLV6TnOvdG4fXm7IkSB3wErkcmJS9iEhjVtOSEjjA==",
+      "dev": true,
+      "license": "Apache-2.0",
+      "dependencies": {
+        "@eslint/core": "^0.17.0",
+        "levn": "^0.4.1"
+      },
+      "engines": {
+        "node": "^18.18.0 || ^20.9.0 || >=21.1.0"
+      }
+    },
+    "node_modules/@humanfs/core": {
+      "version": "0.19.1",
+      "resolved": "https://registry.npmjs.org/@humanfs/core/-/core-0.19.1.tgz",
+      "integrity": "sha512-5DyQ4+1JEUzejeK1JGICcideyfUbGixgS9jNgex5nqkW+cY7WZhxBigmieN5Qnw9ZosSNVC9KQKyb+GUaGyKUA==",
+      "dev": true,
+      "license": "Apache-2.0",
+      "engines": {
+        "node": ">=18.18.0"
+      }
+    },
+    "node_modules/@humanfs/node": {
+      "version": "0.16.7",
+      "resolved": "https://registry.npmjs.org/@humanfs/node/-/node-0.16.7.tgz",
+      "integrity": "sha512-/zUx+yOsIrG4Y43Eh2peDeKCxlRt/gET6aHfaKpuq267qXdYDFViVHfMaLyygZOnl0kGWxFIgsBy8QFuTLUXEQ==",
+      "dev": true,
+      "license": "Apache-2.0",
+      "dependencies": {
+        "@humanfs/core": "^0.19.1",
+        "@humanwhocodes/retry": "^0.4.0"
+      },
+      "engines": {
+        "node": ">=18.18.0"
+      }
+    },
+    "node_modules/@humanwhocodes/module-importer": {
+      "version": "1.0.1",
+      "resolved": "https://registry.npmjs.org/@humanwhocodes/module-importer/-/module-importer-1.0.1.tgz",
+      "integrity": "sha512-bxveV4V8v5Yb4ncFTT3rPSgZBOpCkjfK0y4oVVVJwIuDVBRMDXrPyXRL988i5ap9m9bnyEEjWfm5WkBmtffLfA==",
+      "dev": true,
+      "license": "Apache-2.0",
+      "engines": {
+        "node": ">=12.22"
+      },
+      "funding": {
+        "type": "github",
+        "url": "https://github.com/sponsors/nzakas"
+      }
+    },
+    "node_modules/@humanwhocodes/retry": {
+      "version": "0.4.3",
+      "resolved": "https://registry.npmjs.org/@humanwhocodes/retry/-/retry-0.4.3.tgz",
+      "integrity": "sha512-bV0Tgo9K4hfPCek+aMAn81RppFKv2ySDQeMoSZuvTASywNTnVJCArCZE2FWqpvIatKu7VMRLWlR1EazvVhDyhQ==",
+      "dev": true,
+      "license": "Apache-2.0",
+      "engines": {
+        "node": ">=18.18"
+      },
+      "funding": {
+        "type": "github",
+        "url": "https://github.com/sponsors/nzakas"
+      }
+    },
+    "node_modules/@jridgewell/gen-mapping": {
+      "version": "0.3.13",
+      "resolved": "https://registry.npmjs.org/@jridgewell/gen-mapping/-/gen-mapping-0.3.13.tgz",
+      "integrity": "sha512-2kkt/7niJ6MgEPxF0bYdQ6etZaA+fQvDcLKckhy1yIQOzaoKjBBjSj63/aLVjYE3qhRt5dvM+uUyfCg6UKCBbA==",
+      "dev": true,
+      "license": "MIT",
+      "dependencies": {
+        "@jridgewell/sourcemap-codec": "^1.5.0",
+        "@jridgewell/trace-mapping": "^0.3.24"
+      }
+    },
+    "node_modules/@jridgewell/resolve-uri": {
+      "version": "3.1.2",
+      "resolved": "https://registry.npmjs.org/@jridgewell/resolve-uri/-/resolve-uri-3.1.2.tgz",
+      "integrity": "sha512-bRISgCIjP20/tbWSPWMEi54QVPRZExkuD9lJL+UIxUKtwVJA8wW1Trb1jMs1RFXo1CBTNZ/5hpC9QvmKWdopKw==",
+      "dev": true,
+      "license": "MIT",
+      "engines": {
+        "node": ">=6.0.0"
+      }
+    },
+    "node_modules/@jridgewell/sourcemap-codec": {
+      "version": "1.5.5",
+      "resolved": "https://registry.npmjs.org/@jridgewell/sourcemap-codec/-/sourcemap-codec-1.5.5.tgz",
+      "integrity": "sha512-cYQ9310grqxueWbl+WuIUIaiUaDcj7WOq5fVhEljNVgRfOUhY9fy2zTvfoqWsnebh8Sl70VScFbICvJnLKB0Og==",
+      "dev": true,
+      "license": "MIT"
+    },
+    "node_modules/@jridgewell/trace-mapping": {
+      "version": "0.3.31",
+      "resolved": "https://registry.npmjs.org/@jridgewell/trace-mapping/-/trace-mapping-0.3.31.tgz",
+      "integrity": "sha512-zzNR+SdQSDJzc8joaeP8QQoCQr8NuYx2dIIytl1QeBEZHJ9uW6hebsrYgbz8hJwUQao3TWCMtmfV8Nu1twOLAw==",
+      "dev": true,
+      "license": "MIT",
+      "dependencies": {
+        "@jridgewell/resolve-uri": "^3.1.0",
+        "@jridgewell/sourcemap-codec": "^1.4.14"
+      }
+    },
+    "node_modules/@napi-rs/wasm-runtime": {
+      "version": "1.1.1",
+      "resolved": "https://registry.npmjs.org/@napi-rs/wasm-runtime/-/wasm-runtime-1.1.1.tgz",
+      "integrity": "sha512-p64ah1M1ld8xjWv3qbvFwHiFVWrq1yFvV4f7w+mzaqiR4IlSgkqhcRdHwsGgomwzBH51sRY4NEowLxnaBjcW/A==",
+      "dev": true,
+      "license": "MIT",
+      "optional": true,
+      "dependencies": {
+        "@emnapi/core": "^1.7.1",
+        "@emnapi/runtime": "^1.7.1",
+        "@tybys/wasm-util": "^0.10.1"
+      },
+      "funding": {
+        "type": "github",
+        "url": "https://github.com/sponsors/Brooooooklyn"
+      }
+    },
+    "node_modules/@oxc-project/types": {
+      "version": "0.115.0",
+      "resolved": "https://registry.npmjs.org/@oxc-project/types/-/types-0.115.0.tgz",
+      "integrity": "sha512-4n91DKnebUS4yjUHl2g3/b2T+IUdCfmoZGhmwsovZCDaJSs+QkVAM+0AqqTxHSsHfeiMuueT75cZaZcT/m0pSw==",
+      "dev": true,
+      "license": "MIT",
+      "funding": {
+        "url": "https://github.com/sponsors/Boshen"
+      }
+    },
+    "node_modules/@parcel/watcher": {
+      "version": "2.5.6",
+      "resolved": "https://registry.npmjs.org/@parcel/watcher/-/watcher-2.5.6.tgz",
+      "integrity": "sha512-tmmZ3lQxAe/k/+rNnXQRawJ4NjxO2hqiOLTHvWchtGZULp4RyFeh6aU4XdOYBFe2KE1oShQTv4AblOs2iOrNnQ==",
+      "dev": true,
+      "hasInstallScript": true,
+      "license": "MIT",
+      "dependencies": {
+        "detect-libc": "^2.0.3",
+        "is-glob": "^4.0.3",
+        "node-addon-api": "^7.0.0",
+        "picomatch": "^4.0.3"
+      },
+      "engines": {
+        "node": ">= 10.0.0"
+      },
+      "funding": {
+        "type": "opencollective",
+        "url": "https://opencollective.com/parcel"
+      },
+      "optionalDependencies": {
+        "@parcel/watcher-android-arm64": "2.5.6",
+        "@parcel/watcher-darwin-arm64": "2.5.6",
+        "@parcel/watcher-darwin-x64": "2.5.6",
+        "@parcel/watcher-freebsd-x64": "2.5.6",
+        "@parcel/watcher-linux-arm-glibc": "2.5.6",
+        "@parcel/watcher-linux-arm-musl": "2.5.6",
+        "@parcel/watcher-linux-arm64-glibc": "2.5.6",
+        "@parcel/watcher-linux-arm64-musl": "2.5.6",
+        "@parcel/watcher-linux-x64-glibc": "2.5.6",
+        "@parcel/watcher-linux-x64-musl": "2.5.6",
+        "@parcel/watcher-win32-arm64": "2.5.6",
+        "@parcel/watcher-win32-ia32": "2.5.6",
+        "@parcel/watcher-win32-x64": "2.5.6"
+      }
+    },
+    "node_modules/@parcel/watcher-android-arm64": {
+      "version": "2.5.6",
+      "resolved": "https://registry.npmjs.org/@parcel/watcher-android-arm64/-/watcher-android-arm64-2.5.6.tgz",
+      "integrity": "sha512-YQxSS34tPF/6ZG7r/Ih9xy+kP/WwediEUsqmtf0cuCV5TPPKw/PQHRhueUo6JdeFJaqV3pyjm0GdYjZotbRt/A==",
+      "cpu": [
+        "arm64"
+      ],
+      "dev": true,
+      "license": "MIT",
+      "optional": true,
+      "os": [
+        "android"
+      ],
+      "engines": {
+        "node": ">= 10.0.0"
+      },
+      "funding": {
+        "type": "opencollective",
+        "url": "https://opencollective.com/parcel"
+      }
+    },
+    "node_modules/@parcel/watcher-darwin-arm64": {
+      "version": "2.5.6",
+      "resolved": "https://registry.npmjs.org/@parcel/watcher-darwin-arm64/-/watcher-darwin-arm64-2.5.6.tgz",
+      "integrity": "sha512-Z2ZdrnwyXvvvdtRHLmM4knydIdU9adO3D4n/0cVipF3rRiwP+3/sfzpAwA/qKFL6i1ModaabkU7IbpeMBgiVEA==",
+      "cpu": [
+        "arm64"
+      ],
+      "dev": true,
+      "license": "MIT",
+      "optional": true,
+      "os": [
+        "darwin"
+      ],
+      "engines": {
+        "node": ">= 10.0.0"
+      },
+      "funding": {
+        "type": "opencollective",
+        "url": "https://opencollective.com/parcel"
+      }
+    },
+    "node_modules/@parcel/watcher-darwin-x64": {
+      "version": "2.5.6",
+      "resolved": "https://registry.npmjs.org/@parcel/watcher-darwin-x64/-/watcher-darwin-x64-2.5.6.tgz",
+      "integrity": "sha512-HgvOf3W9dhithcwOWX9uDZyn1lW9R+7tPZ4sug+NGrGIo4Rk1hAXLEbcH1TQSqxts0NYXXlOWqVpvS1SFS4fRg==",
+      "cpu": [
+        "x64"
+      ],
+      "dev": true,
+      "license": "MIT",
+      "optional": true,
+      "os": [
+        "darwin"
+      ],
+      "engines": {
+        "node": ">= 10.0.0"
+      },
+      "funding": {
+        "type": "opencollective",
+        "url": "https://opencollective.com/parcel"
+      }
+    },
+    "node_modules/@parcel/watcher-freebsd-x64": {
+      "version": "2.5.6",
+      "resolved": "https://registry.npmjs.org/@parcel/watcher-freebsd-x64/-/watcher-freebsd-x64-2.5.6.tgz",
+      "integrity": "sha512-vJVi8yd/qzJxEKHkeemh7w3YAn6RJCtYlE4HPMoVnCpIXEzSrxErBW5SJBgKLbXU3WdIpkjBTeUNtyBVn8TRng==",
+      "cpu": [
+        "x64"
+      ],
+      "dev": true,
+      "license": "MIT",
+      "optional": true,
+      "os": [
+        "freebsd"
+      ],
+      "engines": {
+        "node": ">= 10.0.0"
+      },
+      "funding": {
+        "type": "opencollective",
+        "url": "https://opencollective.com/parcel"
+      }
+    },
+    "node_modules/@parcel/watcher-linux-arm-glibc": {
+      "version": "2.5.6",
+      "resolved": "https://registry.npmjs.org/@parcel/watcher-linux-arm-glibc/-/watcher-linux-arm-glibc-2.5.6.tgz",
+      "integrity": "sha512-9JiYfB6h6BgV50CCfasfLf/uvOcJskMSwcdH1PHH9rvS1IrNy8zad6IUVPVUfmXr+u+Km9IxcfMLzgdOudz9EQ==",
+      "cpu": [
+        "arm"
+      ],
+      "dev": true,
+      "license": "MIT",
+      "optional": true,
+      "os": [
+        "linux"
+      ],
+      "engines": {
+        "node": ">= 10.0.0"
+      },
+      "funding": {
+        "type": "opencollective",
+        "url": "https://opencollective.com/parcel"
+      }
+    },
+    "node_modules/@parcel/watcher-linux-arm-musl": {
+      "version": "2.5.6",
+      "resolved": "https://registry.npmjs.org/@parcel/watcher-linux-arm-musl/-/watcher-linux-arm-musl-2.5.6.tgz",
+      "integrity": "sha512-Ve3gUCG57nuUUSyjBq/MAM0CzArtuIOxsBdQ+ftz6ho8n7s1i9E1Nmk/xmP323r2YL0SONs1EuwqBp2u1k5fxg==",
+      "cpu": [
+        "arm"
+      ],
+      "dev": true,
+      "license": "MIT",
+      "optional": true,
+      "os": [
+        "linux"
+      ],
+      "engines": {
+        "node": ">= 10.0.0"
+      },
+      "funding": {
+        "type": "opencollective",
+        "url": "https://opencollective.com/parcel"
+      }
+    },
+    "node_modules/@parcel/watcher-linux-arm64-glibc": {
+      "version": "2.5.6",
+      "resolved": "https://registry.npmjs.org/@parcel/watcher-linux-arm64-glibc/-/watcher-linux-arm64-glibc-2.5.6.tgz",
+      "integrity": "sha512-f2g/DT3NhGPdBmMWYoxixqYr3v/UXcmLOYy16Bx0TM20Tchduwr4EaCbmxh1321TABqPGDpS8D/ggOTaljijOA==",
+      "cpu": [
+        "arm64"
+      ],
+      "dev": true,
+      "license": "MIT",
+      "optional": true,
+      "os": [
+        "linux"
+      ],
+      "engines": {
+        "node": ">= 10.0.0"
+      },
+      "funding": {
+        "type": "opencollective",
+        "url": "https://opencollective.com/parcel"
+      }
+    },
+    "node_modules/@parcel/watcher-linux-arm64-musl": {
+      "version": "2.5.6",
+      "resolved": "https://registry.npmjs.org/@parcel/watcher-linux-arm64-musl/-/watcher-linux-arm64-musl-2.5.6.tgz",
+      "integrity": "sha512-qb6naMDGlbCwdhLj6hgoVKJl2odL34z2sqkC7Z6kzir8b5W65WYDpLB6R06KabvZdgoHI/zxke4b3zR0wAbDTA==",
+      "cpu": [
+        "arm64"
+      ],
+      "dev": true,
+      "license": "MIT",
+      "optional": true,
+      "os": [
+        "linux"
+      ],
+      "engines": {
+        "node": ">= 10.0.0"
+      },
+      "funding": {
+        "type": "opencollective",
+        "url": "https://opencollective.com/parcel"
+      }
+    },
+    "node_modules/@parcel/watcher-linux-x64-glibc": {
+      "version": "2.5.6",
+      "resolved": "https://registry.npmjs.org/@parcel/watcher-linux-x64-glibc/-/watcher-linux-x64-glibc-2.5.6.tgz",
+      "integrity": "sha512-kbT5wvNQlx7NaGjzPFu8nVIW1rWqV780O7ZtkjuWaPUgpv2NMFpjYERVi0UYj1msZNyCzGlaCWEtzc+exjMGbQ==",
+      "cpu": [
+        "x64"
+      ],
+      "dev": true,
+      "license": "MIT",
+      "optional": true,
+      "os": [
+        "linux"
+      ],
+      "engines": {
+        "node": ">= 10.0.0"
+      },
+      "funding": {
+        "type": "opencollective",
+        "url": "https://opencollective.com/parcel"
+      }
+    },
+    "node_modules/@parcel/watcher-linux-x64-musl": {
+      "version": "2.5.6",
+      "resolved": "https://registry.npmjs.org/@parcel/watcher-linux-x64-musl/-/watcher-linux-x64-musl-2.5.6.tgz",
+      "integrity": "sha512-1JRFeC+h7RdXwldHzTsmdtYR/Ku8SylLgTU/reMuqdVD7CtLwf0VR1FqeprZ0eHQkO0vqsbvFLXUmYm/uNKJBg==",
+      "cpu": [
+        "x64"
+      ],
+      "dev": true,
+      "license": "MIT",
+      "optional": true,
+      "os": [
+        "linux"
+      ],
+      "engines": {
+        "node": ">= 10.0.0"
+      },
+      "funding": {
+        "type": "opencollective",
+        "url": "https://opencollective.com/parcel"
+      }
+    },
+    "node_modules/@parcel/watcher-win32-arm64": {
+      "version": "2.5.6",
+      "resolved": "https://registry.npmjs.org/@parcel/watcher-win32-arm64/-/watcher-win32-arm64-2.5.6.tgz",
+      "integrity": "sha512-3ukyebjc6eGlw9yRt678DxVF7rjXatWiHvTXqphZLvo7aC5NdEgFufVwjFfY51ijYEWpXbqF5jtrK275z52D4Q==",
+      "cpu": [
+        "arm64"
+      ],
+      "dev": true,
+      "license": "MIT",
+      "optional": true,
+      "os": [
+        "win32"
+      ],
+      "engines": {
+        "node": ">= 10.0.0"
+      },
+      "funding": {
+        "type": "opencollective",
+        "url": "https://opencollective.com/parcel"
+      }
+    },
+    "node_modules/@parcel/watcher-win32-ia32": {
+      "version": "2.5.6",
+      "resolved": "https://registry.npmjs.org/@parcel/watcher-win32-ia32/-/watcher-win32-ia32-2.5.6.tgz",
+      "integrity": "sha512-k35yLp1ZMwwee3Ez/pxBi5cf4AoBKYXj00CZ80jUz5h8prpiaQsiRPKQMxoLstNuqe2vR4RNPEAEcjEFzhEz/g==",
+      "cpu": [
+        "ia32"
+      ],
+      "dev": true,
+      "license": "MIT",
+      "optional": true,
+      "os": [
+        "win32"
+      ],
+      "engines": {
+        "node": ">= 10.0.0"
+      },
+      "funding": {
+        "type": "opencollective",
+        "url": "https://opencollective.com/parcel"
+      }
+    },
+    "node_modules/@parcel/watcher-win32-x64": {
+      "version": "2.5.6",
+      "resolved": "https://registry.npmjs.org/@parcel/watcher-win32-x64/-/watcher-win32-x64-2.5.6.tgz",
+      "integrity": "sha512-hbQlYcCq5dlAX9Qx+kFb0FHue6vbjlf0FrNzSKdYK2APUf7tGfGxQCk2ihEREmbR6ZMc0MVAD5RIX/41gpUzTw==",
+      "cpu": [
+        "x64"
+      ],
+      "dev": true,
+      "license": "MIT",
+      "optional": true,
+      "os": [
+        "win32"
+      ],
+      "engines": {
+        "node": ">= 10.0.0"
+      },
+      "funding": {
+        "type": "opencollective",
+        "url": "https://opencollective.com/parcel"
+      }
+    },
+    "node_modules/@rolldown/binding-android-arm64": {
+      "version": "1.0.0-rc.8",
+      "resolved": "https://registry.npmjs.org/@rolldown/binding-android-arm64/-/binding-android-arm64-1.0.0-rc.8.tgz",
+      "integrity": "sha512-5bcmMQDWEfWUq3m79Mcf/kbO6e5Jr6YjKSsA1RnpXR6k73hQ9z1B17+4h93jXpzHvS18p7bQHM1HN/fSd+9zog==",
+      "cpu": [
+        "arm64"
+      ],
+      "dev": true,
+      "license": "MIT",
+      "optional": true,
+      "os": [
+        "android"
+      ],
+      "engines": {
+        "node": "^20.19.0 || >=22.12.0"
+      }
+    },
+    "node_modules/@rolldown/binding-darwin-arm64": {
+      "version": "1.0.0-rc.8",
+      "resolved": "https://registry.npmjs.org/@rolldown/binding-darwin-arm64/-/binding-darwin-arm64-1.0.0-rc.8.tgz",
+      "integrity": "sha512-dcHPd5N4g9w2iiPRJmAvO0fsIWzF2JPr9oSuTjxLL56qu+oML5aMbBMNwWbk58Mt3pc7vYs9CCScwLxdXPdRsg==",
+      "cpu": [
+        "arm64"
+      ],
+      "dev": true,
+      "license": "MIT",
+      "optional": true,
+      "os": [
+        "darwin"
+      ],
+      "engines": {
+        "node": "^20.19.0 || >=22.12.0"
+      }
+    },
+    "node_modules/@rolldown/binding-darwin-x64": {
+      "version": "1.0.0-rc.8",
+      "resolved": "https://registry.npmjs.org/@rolldown/binding-darwin-x64/-/binding-darwin-x64-1.0.0-rc.8.tgz",
+      "integrity": "sha512-mw0VzDvoj8AuR761QwpdCFN0sc/jspuc7eRYJetpLWd+XyansUrH3C7IgNw6swBOgQT9zBHNKsVCjzpfGJlhUA==",
+      "cpu": [
+        "x64"
+      ],
+      "dev": true,
+      "license": "MIT",
+      "optional": true,
+      "os": [
+        "darwin"
+      ],
+      "engines": {
+        "node": "^20.19.0 || >=22.12.0"
+      }
+    },
+    "node_modules/@rolldown/binding-freebsd-x64": {
+      "version": "1.0.0-rc.8",
+      "resolved": "https://registry.npmjs.org/@rolldown/binding-freebsd-x64/-/binding-freebsd-x64-1.0.0-rc.8.tgz",
+      "integrity": "sha512-xNrRa6mQ9NmMIJBdJtPMPG8Mso0OhM526pDzc/EKnRrIrrkHD1E0Z6tONZRmUeJElfsQ6h44lQQCcDilSNIvSQ==",
+      "cpu": [
+        "x64"
+      ],
+      "dev": true,
+      "license": "MIT",
+      "optional": true,
+      "os": [
+        "freebsd"
+      ],
+      "engines": {
+        "node": "^20.19.0 || >=22.12.0"
+      }
+    },
+    "node_modules/@rolldown/binding-linux-arm-gnueabihf": {
+      "version": "1.0.0-rc.8",
+      "resolved": "https://registry.npmjs.org/@rolldown/binding-linux-arm-gnueabihf/-/binding-linux-arm-gnueabihf-1.0.0-rc.8.tgz",
+      "integrity": "sha512-WgCKoO6O/rRUwimWfEJDeztwJJmuuX0N2bYLLRxmXDTtCwjToTOqk7Pashl/QpQn3H/jHjx0b5yCMbcTVYVpNg==",
+      "cpu": [
+        "arm"
+      ],
+      "dev": true,
+      "license": "MIT",
+      "optional": true,
+      "os": [
+        "linux"
+      ],
+      "engines": {
+        "node": "^20.19.0 || >=22.12.0"
+      }
+    },
+    "node_modules/@rolldown/binding-linux-arm64-gnu": {
+      "version": "1.0.0-rc.8",
+      "resolved": "https://registry.npmjs.org/@rolldown/binding-linux-arm64-gnu/-/binding-linux-arm64-gnu-1.0.0-rc.8.tgz",
+      "integrity": "sha512-tOHgTOQa8G4Z3ULj4G3NYOGGJEsqPHR91dT72u63OtVsZ7B6wFJKOx+ZKv+pvwzxWz92/I2ycaqi2/Ll4l+rlg==",
+      "cpu": [
+        "arm64"
+      ],
+      "dev": true,
+      "license": "MIT",
+      "optional": true,
+      "os": [
+        "linux"
+      ],
+      "engines": {
+        "node": "^20.19.0 || >=22.12.0"
+      }
+    },
+    "node_modules/@rolldown/binding-linux-arm64-musl": {
+      "version": "1.0.0-rc.8",
+      "resolved": "https://registry.npmjs.org/@rolldown/binding-linux-arm64-musl/-/binding-linux-arm64-musl-1.0.0-rc.8.tgz",
+      "integrity": "sha512-oRbxcgDujCi2Yp1GTxoUFsIFlZsuPHU4OV4AzNc3/6aUmR4lfm9FK0uwQu82PJsuUwnF2jFdop3Ep5c1uK7Uxg==",
+      "cpu": [
+        "arm64"
+      ],
+      "dev": true,
+      "license": "MIT",
+      "optional": true,
+      "os": [
+        "linux"
+      ],
+      "engines": {
+        "node": "^20.19.0 || >=22.12.0"
+      }
+    },
+    "node_modules/@rolldown/binding-linux-ppc64-gnu": {
+      "version": "1.0.0-rc.8",
+      "resolved": "https://registry.npmjs.org/@rolldown/binding-linux-ppc64-gnu/-/binding-linux-ppc64-gnu-1.0.0-rc.8.tgz",
+      "integrity": "sha512-oaLRyUHw8kQE5M89RqrDJZ10GdmGJcMeCo8tvaE4ukOofqgjV84AbqBSH6tTPjeT2BHv+xlKj678GBuIb47lKA==",
+      "cpu": [
+        "ppc64"
+      ],
+      "dev": true,
+      "license": "MIT",
+      "optional": true,
+      "os": [
+        "linux"
+      ],
+      "engines": {
+        "node": "^20.19.0 || >=22.12.0"
+      }
+    },
+    "node_modules/@rolldown/binding-linux-s390x-gnu": {
+      "version": "1.0.0-rc.8",
+      "resolved": "https://registry.npmjs.org/@rolldown/binding-linux-s390x-gnu/-/binding-linux-s390x-gnu-1.0.0-rc.8.tgz",
+      "integrity": "sha512-1hjSKFrod5MwBBdLOOA0zpUuSfSDkYIY+QqcMcIU1WOtswZtZdUkcFcZza9b2HcAb0bnpmmyo0LZcaxLb2ov1g==",
+      "cpu": [
+        "s390x"
+      ],
+      "dev": true,
+      "license": "MIT",
+      "optional": true,
+      "os": [
+        "linux"
+      ],
+      "engines": {
+        "node": "^20.19.0 || >=22.12.0"
+      }
+    },
+    "node_modules/@rolldown/binding-linux-x64-gnu": {
+      "version": "1.0.0-rc.8",
+      "resolved": "https://registry.npmjs.org/@rolldown/binding-linux-x64-gnu/-/binding-linux-x64-gnu-1.0.0-rc.8.tgz",
+      "integrity": "sha512-a1+F0aV4Wy9tT3o+cHl3XhOy6aFV+B8Ll+/JFj98oGkb6lGk3BNgrxd+80RwYRVd23oLGvj3LwluKYzlv1PEuw==",
+      "cpu": [
+        "x64"
+      ],
+      "dev": true,
+      "license": "MIT",
+      "optional": true,
+      "os": [
+        "linux"
+      ],
+      "engines": {
+        "node": "^20.19.0 || >=22.12.0"
+      }
+    },
+    "node_modules/@rolldown/binding-linux-x64-musl": {
+      "version": "1.0.0-rc.8",
+      "resolved": "https://registry.npmjs.org/@rolldown/binding-linux-x64-musl/-/binding-linux-x64-musl-1.0.0-rc.8.tgz",
+      "integrity": "sha512-bGyXCFU11seFrf7z8PcHSwGEiFVkZ9vs+auLacVOQrVsI8PFHJzzJROF3P6b0ODDmXr0m6Tj5FlDhcXVk0Jp8w==",
+      "cpu": [
+        "x64"
+      ],
+      "dev": true,
+      "license": "MIT",
+      "optional": true,
+      "os": [
+        "linux"
+      ],
+      "engines": {
+        "node": "^20.19.0 || >=22.12.0"
+      }
+    },
+    "node_modules/@rolldown/binding-openharmony-arm64": {
+      "version": "1.0.0-rc.8",
+      "resolved": "https://registry.npmjs.org/@rolldown/binding-openharmony-arm64/-/binding-openharmony-arm64-1.0.0-rc.8.tgz",
+      "integrity": "sha512-n8d+L2bKgf9G3+AM0bhHFWdlz9vYKNim39ujRTieukdRek0RAo2TfG2uEnV9spa4r4oHUfL9IjcY3M9SlqN1gw==",
+      "cpu": [
+        "arm64"
+      ],
+      "dev": true,
+      "license": "MIT",
+      "optional": true,
+      "os": [
+        "openharmony"
+      ],
+      "engines": {
+        "node": "^20.19.0 || >=22.12.0"
+      }
+    },
+    "node_modules/@rolldown/binding-wasm32-wasi": {
+      "version": "1.0.0-rc.8",
+      "resolved": "https://registry.npmjs.org/@rolldown/binding-wasm32-wasi/-/binding-wasm32-wasi-1.0.0-rc.8.tgz",
+      "integrity": "sha512-4R4iJDIk7BrJdteAbEAICXPoA7vZoY/M0OBfcRlQxzQvUYMcEp2GbC/C8UOgQJhu2TjGTpX1H8vVO1xHWcRqQA==",
+      "cpu": [
+        "wasm32"
+      ],
+      "dev": true,
+      "license": "MIT",
+      "optional": true,
+      "dependencies": {
+        "@napi-rs/wasm-runtime": "^1.1.1"
+      },
+      "engines": {
+        "node": ">=14.0.0"
+      }
+    },
+    "node_modules/@rolldown/binding-win32-arm64-msvc": {
+      "version": "1.0.0-rc.8",
+      "resolved": "https://registry.npmjs.org/@rolldown/binding-win32-arm64-msvc/-/binding-win32-arm64-msvc-1.0.0-rc.8.tgz",
+      "integrity": "sha512-3lwnklba9qQOpFnQ7EW+A1m4bZTWXZE4jtehsZ0YOl2ivW1FQqp5gY7X2DLuKITggesyuLwcmqS11fA7NtrmrA==",
+      "cpu": [
+        "arm64"
+      ],
+      "dev": true,
+      "license": "MIT",
+      "optional": true,
+      "os": [
+        "win32"
+      ],
+      "engines": {
+        "node": "^20.19.0 || >=22.12.0"
+      }
+    },
+    "node_modules/@rolldown/binding-win32-x64-msvc": {
+      "version": "1.0.0-rc.8",
+      "resolved": "https://registry.npmjs.org/@rolldown/binding-win32-x64-msvc/-/binding-win32-x64-msvc-1.0.0-rc.8.tgz",
+      "integrity": "sha512-VGjCx9Ha1P/r3tXGDZyG0Fcq7Q0Afnk64aaKzr1m40vbn1FL8R3W0V1ELDvPgzLXaaqK/9PnsqSaLWXfn6JtGQ==",
+      "cpu": [
+        "x64"
+      ],
+      "dev": true,
+      "license": "MIT",
+      "optional": true,
+      "os": [
+        "win32"
+      ],
+      "engines": {
+        "node": "^20.19.0 || >=22.12.0"
+      }
+    },
+    "node_modules/@rolldown/pluginutils": {
+      "version": "1.0.0-rc.8",
+      "resolved": "https://registry.npmjs.org/@rolldown/pluginutils/-/pluginutils-1.0.0-rc.8.tgz",
+      "integrity": "sha512-wzJwL82/arVfeSP3BLr1oTy40XddjtEdrdgtJ4lLRBu06mP3q/8HGM6K0JRlQuTA3XB0pNJx2so/nmpY4xyOew==",
+      "dev": true,
+      "license": "MIT"
+    },
+    "node_modules/@rollup/rollup-android-arm-eabi": {
+      "version": "4.59.0",
+      "resolved": "https://registry.npmjs.org/@rollup/rollup-android-arm-eabi/-/rollup-android-arm-eabi-4.59.0.tgz",
+      "integrity": "sha512-upnNBkA6ZH2VKGcBj9Fyl9IGNPULcjXRlg0LLeaioQWueH30p6IXtJEbKAgvyv+mJaMxSm1l6xwDXYjpEMiLMg==",
+      "cpu": [
+        "arm"
+      ],
+      "dev": true,
+      "license": "MIT",
+      "optional": true,
+      "os": [
+        "android"
+      ]
+    },
+    "node_modules/@rollup/rollup-android-arm64": {
+      "version": "4.59.0",
+      "resolved": "https://registry.npmjs.org/@rollup/rollup-android-arm64/-/rollup-android-arm64-4.59.0.tgz",
+      "integrity": "sha512-hZ+Zxj3SySm4A/DylsDKZAeVg0mvi++0PYVceVyX7hemkw7OreKdCvW2oQ3T1FMZvCaQXqOTHb8qmBShoqk69Q==",
+      "cpu": [
+        "arm64"
+      ],
+      "dev": true,
+      "license": "MIT",
+      "optional": true,
+      "os": [
+        "android"
+      ]
+    },
+    "node_modules/@rollup/rollup-darwin-arm64": {
+      "version": "4.59.0",
+      "resolved": "https://registry.npmjs.org/@rollup/rollup-darwin-arm64/-/rollup-darwin-arm64-4.59.0.tgz",
+      "integrity": "sha512-W2Psnbh1J8ZJw0xKAd8zdNgF9HRLkdWwwdWqubSVk0pUuQkoHnv7rx4GiF9rT4t5DIZGAsConRE3AxCdJ4m8rg==",
+      "cpu": [
+        "arm64"
+      ],
+      "dev": true,
+      "license": "MIT",
+      "optional": true,
+      "os": [
+        "darwin"
+      ]
+    },
+    "node_modules/@rollup/rollup-darwin-x64": {
+      "version": "4.59.0",
+      "resolved": "https://registry.npmjs.org/@rollup/rollup-darwin-x64/-/rollup-darwin-x64-4.59.0.tgz",
+      "integrity": "sha512-ZW2KkwlS4lwTv7ZVsYDiARfFCnSGhzYPdiOU4IM2fDbL+QGlyAbjgSFuqNRbSthybLbIJ915UtZBtmuLrQAT/w==",
+      "cpu": [
+        "x64"
+      ],
+      "dev": true,
+      "license": "MIT",
+      "optional": true,
+      "os": [
+        "darwin"
+      ]
+    },
+    "node_modules/@rollup/rollup-freebsd-arm64": {
+      "version": "4.59.0",
+      "resolved": "https://registry.npmjs.org/@rollup/rollup-freebsd-arm64/-/rollup-freebsd-arm64-4.59.0.tgz",
+      "integrity": "sha512-EsKaJ5ytAu9jI3lonzn3BgG8iRBjV4LxZexygcQbpiU0wU0ATxhNVEpXKfUa0pS05gTcSDMKpn3Sx+QB9RlTTA==",
+      "cpu": [
+        "arm64"
+      ],
+      "dev": true,
+      "license": "MIT",
+      "optional": true,
+      "os": [
+        "freebsd"
+      ]
+    },
+    "node_modules/@rollup/rollup-freebsd-x64": {
+      "version": "4.59.0",
+      "resolved": "https://registry.npmjs.org/@rollup/rollup-freebsd-x64/-/rollup-freebsd-x64-4.59.0.tgz",
+      "integrity": "sha512-d3DuZi2KzTMjImrxoHIAODUZYoUUMsuUiY4SRRcJy6NJoZ6iIqWnJu9IScV9jXysyGMVuW+KNzZvBLOcpdl3Vg==",
+      "cpu": [
+        "x64"
+      ],
+      "dev": true,
+      "license": "MIT",
+      "optional": true,
+      "os": [
+        "freebsd"
+      ]
+    },
+    "node_modules/@rollup/rollup-linux-arm-gnueabihf": {
+      "version": "4.59.0",
+      "resolved": "https://registry.npmjs.org/@rollup/rollup-linux-arm-gnueabihf/-/rollup-linux-arm-gnueabihf-4.59.0.tgz",
+      "integrity": "sha512-t4ONHboXi/3E0rT6OZl1pKbl2Vgxf9vJfWgmUoCEVQVxhW6Cw/c8I6hbbu7DAvgp82RKiH7TpLwxnJeKv2pbsw==",
+      "cpu": [
+        "arm"
+      ],
+      "dev": true,
+      "license": "MIT",
+      "optional": true,
+      "os": [
+        "linux"
+      ]
+    },
+    "node_modules/@rollup/rollup-linux-arm-musleabihf": {
+      "version": "4.59.0",
+      "resolved": "https://registry.npmjs.org/@rollup/rollup-linux-arm-musleabihf/-/rollup-linux-arm-musleabihf-4.59.0.tgz",
+      "integrity": "sha512-CikFT7aYPA2ufMD086cVORBYGHffBo4K8MQ4uPS/ZnY54GKj36i196u8U+aDVT2LX4eSMbyHtyOh7D7Zvk2VvA==",
+      "cpu": [
+        "arm"
+      ],
+      "dev": true,
+      "license": "MIT",
+      "optional": true,
+      "os": [
+        "linux"
+      ]
+    },
+    "node_modules/@rollup/rollup-linux-arm64-gnu": {
+      "version": "4.59.0",
+      "resolved": "https://registry.npmjs.org/@rollup/rollup-linux-arm64-gnu/-/rollup-linux-arm64-gnu-4.59.0.tgz",
+      "integrity": "sha512-jYgUGk5aLd1nUb1CtQ8E+t5JhLc9x5WdBKew9ZgAXg7DBk0ZHErLHdXM24rfX+bKrFe+Xp5YuJo54I5HFjGDAA==",
+      "cpu": [
+        "arm64"
+      ],
+      "dev": true,
+      "license": "MIT",
+      "optional": true,
+      "os": [
+        "linux"
+      ]
+    },
+    "node_modules/@rollup/rollup-linux-arm64-musl": {
+      "version": "4.59.0",
+      "resolved": "https://registry.npmjs.org/@rollup/rollup-linux-arm64-musl/-/rollup-linux-arm64-musl-4.59.0.tgz",
+      "integrity": "sha512-peZRVEdnFWZ5Bh2KeumKG9ty7aCXzzEsHShOZEFiCQlDEepP1dpUl/SrUNXNg13UmZl+gzVDPsiCwnV1uI0RUA==",
+      "cpu": [
+        "arm64"
+      ],
+      "dev": true,
+      "license": "MIT",
+      "optional": true,
+      "os": [
+        "linux"
+      ]
+    },
+    "node_modules/@rollup/rollup-linux-loong64-gnu": {
+      "version": "4.59.0",
+      "resolved": "https://registry.npmjs.org/@rollup/rollup-linux-loong64-gnu/-/rollup-linux-loong64-gnu-4.59.0.tgz",
+      "integrity": "sha512-gbUSW/97f7+r4gHy3Jlup8zDG190AuodsWnNiXErp9mT90iCy9NKKU0Xwx5k8VlRAIV2uU9CsMnEFg/xXaOfXg==",
+      "cpu": [
+        "loong64"
+      ],
+      "dev": true,
+      "license": "MIT",
+      "optional": true,
+      "os": [
+        "linux"
+      ]
+    },
+    "node_modules/@rollup/rollup-linux-loong64-musl": {
+      "version": "4.59.0",
+      "resolved": "https://registry.npmjs.org/@rollup/rollup-linux-loong64-musl/-/rollup-linux-loong64-musl-4.59.0.tgz",
+      "integrity": "sha512-yTRONe79E+o0FWFijasoTjtzG9EBedFXJMl888NBEDCDV9I2wGbFFfJQQe63OijbFCUZqxpHz1GzpbtSFikJ4Q==",
+      "cpu": [
+        "loong64"
+      ],
+      "dev": true,
+      "license": "MIT",
+      "optional": true,
+      "os": [
+        "linux"
+      ]
+    },
+    "node_modules/@rollup/rollup-linux-ppc64-gnu": {
+      "version": "4.59.0",
+      "resolved": "https://registry.npmjs.org/@rollup/rollup-linux-ppc64-gnu/-/rollup-linux-ppc64-gnu-4.59.0.tgz",
+      "integrity": "sha512-sw1o3tfyk12k3OEpRddF68a1unZ5VCN7zoTNtSn2KndUE+ea3m3ROOKRCZxEpmT9nsGnogpFP9x6mnLTCaoLkA==",
+      "cpu": [
+        "ppc64"
+      ],
+      "dev": true,
+      "license": "MIT",
+      "optional": true,
+      "os": [
+        "linux"
+      ]
+    },
+    "node_modules/@rollup/rollup-linux-ppc64-musl": {
+      "version": "4.59.0",
+      "resolved": "https://registry.npmjs.org/@rollup/rollup-linux-ppc64-musl/-/rollup-linux-ppc64-musl-4.59.0.tgz",
+      "integrity": "sha512-+2kLtQ4xT3AiIxkzFVFXfsmlZiG5FXYW7ZyIIvGA7Bdeuh9Z0aN4hVyXS/G1E9bTP/vqszNIN/pUKCk/BTHsKA==",
+      "cpu": [
+        "ppc64"
+      ],
+      "dev": true,
+      "license": "MIT",
+      "optional": true,
+      "os": [
+        "linux"
+      ]
+    },
+    "node_modules/@rollup/rollup-linux-riscv64-gnu": {
+      "version": "4.59.0",
+      "resolved": "https://registry.npmjs.org/@rollup/rollup-linux-riscv64-gnu/-/rollup-linux-riscv64-gnu-4.59.0.tgz",
+      "integrity": "sha512-NDYMpsXYJJaj+I7UdwIuHHNxXZ/b/N2hR15NyH3m2qAtb/hHPA4g4SuuvrdxetTdndfj9b1WOmy73kcPRoERUg==",
+      "cpu": [
+        "riscv64"
+      ],
+      "dev": true,
+      "license": "MIT",
+      "optional": true,
+      "os": [
+        "linux"
+      ]
+    },
+    "node_modules/@rollup/rollup-linux-riscv64-musl": {
+      "version": "4.59.0",
+      "resolved": "https://registry.npmjs.org/@rollup/rollup-linux-riscv64-musl/-/rollup-linux-riscv64-musl-4.59.0.tgz",
+      "integrity": "sha512-nLckB8WOqHIf1bhymk+oHxvM9D3tyPndZH8i8+35p/1YiVoVswPid2yLzgX7ZJP0KQvnkhM4H6QZ5m0LzbyIAg==",
+      "cpu": [
+        "riscv64"
+      ],
+      "dev": true,
+      "license": "MIT",
+      "optional": true,
+      "os": [
+        "linux"
+      ]
+    },
+    "node_modules/@rollup/rollup-linux-s390x-gnu": {
+      "version": "4.59.0",
+      "resolved": "https://registry.npmjs.org/@rollup/rollup-linux-s390x-gnu/-/rollup-linux-s390x-gnu-4.59.0.tgz",
+      "integrity": "sha512-oF87Ie3uAIvORFBpwnCvUzdeYUqi2wY6jRFWJAy1qus/udHFYIkplYRW+wo+GRUP4sKzYdmE1Y3+rY5Gc4ZO+w==",
+      "cpu": [
+        "s390x"
+      ],
+      "dev": true,
+      "license": "MIT",
+      "optional": true,
+      "os": [
+        "linux"
+      ]
+    },
+    "node_modules/@rollup/rollup-linux-x64-gnu": {
+      "version": "4.59.0",
+      "resolved": "https://registry.npmjs.org/@rollup/rollup-linux-x64-gnu/-/rollup-linux-x64-gnu-4.59.0.tgz",
+      "integrity": "sha512-3AHmtQq/ppNuUspKAlvA8HtLybkDflkMuLK4DPo77DfthRb71V84/c4MlWJXixZz4uruIH4uaa07IqoAkG64fg==",
+      "cpu": [
+        "x64"
+      ],
+      "dev": true,
+      "license": "MIT",
+      "optional": true,
+      "os": [
+        "linux"
+      ]
+    },
+    "node_modules/@rollup/rollup-linux-x64-musl": {
+      "version": "4.59.0",
+      "resolved": "https://registry.npmjs.org/@rollup/rollup-linux-x64-musl/-/rollup-linux-x64-musl-4.59.0.tgz",
+      "integrity": "sha512-2UdiwS/9cTAx7qIUZB/fWtToJwvt0Vbo0zmnYt7ED35KPg13Q0ym1g442THLC7VyI6JfYTP4PiSOWyoMdV2/xg==",
+      "cpu": [
+        "x64"
+      ],
+      "dev": true,
+      "license": "MIT",
+      "optional": true,
+      "os": [
+        "linux"
+      ]
+    },
+    "node_modules/@rollup/rollup-openbsd-x64": {
+      "version": "4.59.0",
+      "resolved": "https://registry.npmjs.org/@rollup/rollup-openbsd-x64/-/rollup-openbsd-x64-4.59.0.tgz",
+      "integrity": "sha512-M3bLRAVk6GOwFlPTIxVBSYKUaqfLrn8l0psKinkCFxl4lQvOSz8ZrKDz2gxcBwHFpci0B6rttydI4IpS4IS/jQ==",
+      "cpu": [
+        "x64"
+      ],
+      "dev": true,
+      "license": "MIT",
+      "optional": true,
+      "os": [
+        "openbsd"
+      ]
+    },
+    "node_modules/@rollup/rollup-openharmony-arm64": {
+      "version": "4.59.0",
+      "resolved": "https://registry.npmjs.org/@rollup/rollup-openharmony-arm64/-/rollup-openharmony-arm64-4.59.0.tgz",
+      "integrity": "sha512-tt9KBJqaqp5i5HUZzoafHZX8b5Q2Fe7UjYERADll83O4fGqJ49O1FsL6LpdzVFQcpwvnyd0i+K/VSwu/o/nWlA==",
+      "cpu": [
+        "arm64"
+      ],
+      "dev": true,
+      "license": "MIT",
+      "optional": true,
+      "os": [
+        "openharmony"
+      ]
+    },
+    "node_modules/@rollup/rollup-win32-arm64-msvc": {
+      "version": "4.59.0",
+      "resolved": "https://registry.npmjs.org/@rollup/rollup-win32-arm64-msvc/-/rollup-win32-arm64-msvc-4.59.0.tgz",
+      "integrity": "sha512-V5B6mG7OrGTwnxaNUzZTDTjDS7F75PO1ae6MJYdiMu60sq0CqN5CVeVsbhPxalupvTX8gXVSU9gq+Rx1/hvu6A==",
+      "cpu": [
+        "arm64"
+      ],
+      "dev": true,
+      "license": "MIT",
+      "optional": true,
+      "os": [
+        "win32"
+      ]
+    },
+    "node_modules/@rollup/rollup-win32-ia32-msvc": {
+      "version": "4.59.0",
+      "resolved": "https://registry.npmjs.org/@rollup/rollup-win32-ia32-msvc/-/rollup-win32-ia32-msvc-4.59.0.tgz",
+      "integrity": "sha512-UKFMHPuM9R0iBegwzKF4y0C4J9u8C6MEJgFuXTBerMk7EJ92GFVFYBfOZaSGLu6COf7FxpQNqhNS4c4icUPqxA==",
+      "cpu": [
+        "ia32"
+      ],
+      "dev": true,
+      "license": "MIT",
+      "optional": true,
+      "os": [
+        "win32"
+      ]
+    },
+    "node_modules/@rollup/rollup-win32-x64-gnu": {
+      "version": "4.59.0",
+      "resolved": "https://registry.npmjs.org/@rollup/rollup-win32-x64-gnu/-/rollup-win32-x64-gnu-4.59.0.tgz",
+      "integrity": "sha512-laBkYlSS1n2L8fSo1thDNGrCTQMmxjYY5G0WFWjFFYZkKPjsMBsgJfGf4TLxXrF6RyhI60L8TMOjBMvXiTcxeA==",
+      "cpu": [
+        "x64"
+      ],
+      "dev": true,
+      "license": "MIT",
+      "optional": true,
+      "os": [
+        "win32"
+      ]
+    },
+    "node_modules/@rollup/rollup-win32-x64-msvc": {
+      "version": "4.59.0",
+      "resolved": "https://registry.npmjs.org/@rollup/rollup-win32-x64-msvc/-/rollup-win32-x64-msvc-4.59.0.tgz",
+      "integrity": "sha512-2HRCml6OztYXyJXAvdDXPKcawukWY2GpR5/nxKp4iBgiO3wcoEGkAaqctIbZcNB6KlUQBIqt8VYkNSj2397EfA==",
+      "cpu": [
+        "x64"
+      ],
+      "dev": true,
+      "license": "MIT",
+      "optional": true,
+      "os": [
+        "win32"
+      ]
+    },
+    "node_modules/@standard-schema/spec": {
+      "version": "1.1.0",
+      "resolved": "https://registry.npmjs.org/@standard-schema/spec/-/spec-1.1.0.tgz",
+      "integrity": "sha512-l2aFy5jALhniG5HgqrD6jXLi/rUWrKvqN/qJx6yoJsgKhblVd+iqqU4RCXavm/jPityDo5TCvKMnpjKnOriy0w==",
+      "dev": true,
+      "license": "MIT"
+    },
+    "node_modules/@tybys/wasm-util": {
+      "version": "0.10.1",
+      "resolved": "https://registry.npmjs.org/@tybys/wasm-util/-/wasm-util-0.10.1.tgz",
+      "integrity": "sha512-9tTaPJLSiejZKx+Bmog4uSubteqTvFrVrURwkmHixBo0G4seD0zUxp98E1DzUBJxLQ3NPwXrGKDiVjwx/DpPsg==",
+      "dev": true,
+      "license": "MIT",
+      "optional": true,
+      "dependencies": {
+        "tslib": "^2.4.0"
+      }
+    },
+    "node_modules/@types/chai": {
+      "version": "5.2.3",
+      "resolved": "https://registry.npmjs.org/@types/chai/-/chai-5.2.3.tgz",
+      "integrity": "sha512-Mw558oeA9fFbv65/y4mHtXDs9bPnFMZAL/jxdPFUpOHHIXX91mcgEHbS5Lahr+pwZFR8A7GQleRWeI6cGFC2UA==",
+      "dev": true,
+      "license": "MIT",
+      "dependencies": {
+        "@types/deep-eql": "*",
+        "assertion-error": "^2.0.1"
+      }
+    },
+    "node_modules/@types/debug": {
+      "version": "4.1.12",
+      "resolved": "https://registry.npmjs.org/@types/debug/-/debug-4.1.12.tgz",
+      "integrity": "sha512-vIChWdVG3LG1SMxEvI/AK+FWJthlrqlTu7fbrlywTkkaONwk/UAGaULXRlf8vkzFBLVm0zkMdCquhL5aOjhXPQ==",
+      "dev": true,
+      "license": "MIT",
+      "dependencies": {
+        "@types/ms": "*"
+      }
+    },
+    "node_modules/@types/deep-eql": {
+      "version": "4.0.2",
+      "resolved": "https://registry.npmjs.org/@types/deep-eql/-/deep-eql-4.0.2.tgz",
+      "integrity": "sha512-c9h9dVVMigMPc4bwTvC5dxqtqJZwQPePsWjPlpSOnojbor6pGqdk541lfA7AqFQr5pB1BRdq0juY9db81BwyFw==",
+      "dev": true,
+      "license": "MIT"
+    },
+    "node_modules/@types/estree": {
+      "version": "1.0.8",
+      "resolved": "https://registry.npmjs.org/@types/estree/-/estree-1.0.8.tgz",
+      "integrity": "sha512-dWHzHa2WqEXI/O1E9OjrocMTKJl2mSrEolh1Iomrv6U+JuNwaHXsXx9bLu5gG7BUWFIN0skIQJQ/L1rIex4X6w==",
+      "dev": true,
+      "license": "MIT"
+    },
+    "node_modules/@types/jsesc": {
+      "version": "2.5.1",
+      "resolved": "https://registry.npmjs.org/@types/jsesc/-/jsesc-2.5.1.tgz",
+      "integrity": "sha512-9VN+6yxLOPLOav+7PwjZbxiID2bVaeq0ED4qSQmdQTdjnXJSaCVKTR58t15oqH1H5t8Ng2ZX1SabJVoN9Q34bw==",
+      "dev": true,
+      "license": "MIT"
+    },
+    "node_modules/@types/json-schema": {
+      "version": "7.0.15",
+      "resolved": "https://registry.npmjs.org/@types/json-schema/-/json-schema-7.0.15.tgz",
+      "integrity": "sha512-5+fP8P8MFNC+AyZCDxrB2pkZFPGzqQWUzpSeuuVLvm8VMcorNYavBqoFcxK8bQz4Qsbn4oUEEem4wDLfcysGHA==",
+      "dev": true,
+      "license": "MIT"
+    },
+    "node_modules/@types/mdast": {
+      "version": "4.0.4",
+      "resolved": "https://registry.npmjs.org/@types/mdast/-/mdast-4.0.4.tgz",
+      "integrity": "sha512-kGaNbPh1k7AFzgpud/gMdvIm5xuECykRR+JnWKQno9TAXVa6WIVCGTPvYGekIDL4uwCZQSYbUxNBSb1aUo79oA==",
+      "dev": true,
+      "license": "MIT",
+      "dependencies": {
+        "@types/unist": "*"
+      }
+    },
+    "node_modules/@types/ms": {
+      "version": "2.1.0",
+      "resolved": "https://registry.npmjs.org/@types/ms/-/ms-2.1.0.tgz",
+      "integrity": "sha512-GsCCIZDE/p3i96vtEqx+7dBUGXrc7zeSK3wwPHIaRThS+9OhWIXRqzs4d6k1SVU8g91DrNRWxWUGhp5KXQb2VA==",
+      "dev": true,
+      "license": "MIT"
+    },
+    "node_modules/@types/node": {
+      "version": "25.4.0",
+      "resolved": "https://registry.npmjs.org/@types/node/-/node-25.4.0.tgz",
+      "integrity": "sha512-9wLpoeWuBlcbBpOY3XmzSTG3oscB6xjBEEtn+pYXTfhyXhIxC5FsBer2KTopBlvKEiW9l13po9fq+SJY/5lkhw==",
+      "dev": true,
+      "license": "MIT",
+      "dependencies": {
+        "undici-types": "~7.18.0"
+      }
+    },
+    "node_modules/@types/unist": {
+      "version": "3.0.3",
+      "resolved": "https://registry.npmjs.org/@types/unist/-/unist-3.0.3.tgz",
+      "integrity": "sha512-ko/gIFJRv177XgZsZcBwnqJN5x/Gien8qNOn0D5bQU/zAzVf9Zt3BlcUiLqhV9y4ARk0GbT3tnUiPNgnTXzc/Q==",
+      "dev": true,
+      "license": "MIT"
+    },
+    "node_modules/@typescript-eslint/eslint-plugin": {
+      "version": "8.57.0",
+      "resolved": "https://registry.npmjs.org/@typescript-eslint/eslint-plugin/-/eslint-plugin-8.57.0.tgz",
+      "integrity": "sha512-qeu4rTHR3/IaFORbD16gmjq9+rEs9fGKdX0kF6BKSfi+gCuG3RCKLlSBYzn/bGsY9Tj7KE/DAQStbp8AHJGHEQ==",
+      "dev": true,
+      "license": "MIT",
+      "dependencies": {
+        "@eslint-community/regexpp": "^4.12.2",
+        "@typescript-eslint/scope-manager": "8.57.0",
+        "@typescript-eslint/type-utils": "8.57.0",
+        "@typescript-eslint/utils": "8.57.0",
+        "@typescript-eslint/visitor-keys": "8.57.0",
+        "ignore": "^7.0.5",
+        "natural-compare": "^1.4.0",
+        "ts-api-utils": "^2.4.0"
+      },
+      "engines": {
+        "node": "^18.18.0 || ^20.9.0 || >=21.1.0"
+      },
+      "funding": {
+        "type": "opencollective",
+        "url": "https://opencollective.com/typescript-eslint"
+      },
+      "peerDependencies": {
+        "@typescript-eslint/parser": "^8.57.0",
+        "eslint": "^8.57.0 || ^9.0.0 || ^10.0.0",
+        "typescript": ">=4.8.4 <6.0.0"
+      }
+    },
+    "node_modules/@typescript-eslint/eslint-plugin/node_modules/ignore": {
+      "version": "7.0.5",
+      "resolved": "https://registry.npmjs.org/ignore/-/ignore-7.0.5.tgz",
+      "integrity": "sha512-Hs59xBNfUIunMFgWAbGX5cq6893IbWg4KnrjbYwX3tx0ztorVgTDA6B2sxf8ejHJ4wz8BqGUMYlnzNBer5NvGg==",
+      "dev": true,
+      "license": "MIT",
+      "engines": {
+        "node": ">= 4"
+      }
+    },
+    "node_modules/@typescript-eslint/parser": {
+      "version": "8.57.0",
+      "resolved": "https://registry.npmjs.org/@typescript-eslint/parser/-/parser-8.57.0.tgz",
+      "integrity": "sha512-XZzOmihLIr8AD1b9hL9ccNMzEMWt/dE2u7NyTY9jJG6YNiNthaD5XtUHVF2uCXZ15ng+z2hT3MVuxnUYhq6k1g==",
+      "dev": true,
+      "license": "MIT",
+      "dependencies": {
+        "@typescript-eslint/scope-manager": "8.57.0",
+        "@typescript-eslint/types": "8.57.0",
+        "@typescript-eslint/typescript-estree": "8.57.0",
+        "@typescript-eslint/visitor-keys": "8.57.0",
+        "debug": "^4.4.3"
+      },
+      "engines": {
+        "node": "^18.18.0 || ^20.9.0 || >=21.1.0"
+      },
+      "funding": {
+        "type": "opencollective",
+        "url": "https://opencollective.com/typescript-eslint"
+      },
+      "peerDependencies": {
+        "eslint": "^8.57.0 || ^9.0.0 || ^10.0.0",
+        "typescript": ">=4.8.4 <6.0.0"
+      }
+    },
+    "node_modules/@typescript-eslint/project-service": {
+      "version": "8.57.0",
+      "resolved": "https://registry.npmjs.org/@typescript-eslint/project-service/-/project-service-8.57.0.tgz",
+      "integrity": "sha512-pR+dK0BlxCLxtWfaKQWtYr7MhKmzqZxuii+ZjuFlZlIGRZm22HnXFqa2eY+90MUz8/i80YJmzFGDUsi8dMOV5w==",
+      "dev": true,
+      "license": "MIT",
+      "dependencies": {
+        "@typescript-eslint/tsconfig-utils": "^8.57.0",
+        "@typescript-eslint/types": "^8.57.0",
+        "debug": "^4.4.3"
+      },
+      "engines": {
+        "node": "^18.18.0 || ^20.9.0 || >=21.1.0"
+      },
+      "funding": {
+        "type": "opencollective",
+        "url": "https://opencollective.com/typescript-eslint"
+      },
+      "peerDependencies": {
+        "typescript": ">=4.8.4 <6.0.0"
+      }
+    },
+    "node_modules/@typescript-eslint/scope-manager": {
+      "version": "8.57.0",
+      "resolved": "https://registry.npmjs.org/@typescript-eslint/scope-manager/-/scope-manager-8.57.0.tgz",
+      "integrity": "sha512-nvExQqAHF01lUM66MskSaZulpPL5pgy5hI5RfrxviLgzZVffB5yYzw27uK/ft8QnKXI2X0LBrHJFr1TaZtAibw==",
+      "dev": true,
+      "license": "MIT",
+      "dependencies": {
+        "@typescript-eslint/types": "8.57.0",
+        "@typescript-eslint/visitor-keys": "8.57.0"
+      },
+      "engines": {
+        "node": "^18.18.0 || ^20.9.0 || >=21.1.0"
+      },
+      "funding": {
+        "type": "opencollective",
+        "url": "https://opencollective.com/typescript-eslint"
+      }
+    },
+    "node_modules/@typescript-eslint/tsconfig-utils": {
+      "version": "8.57.0",
+      "resolved": "https://registry.npmjs.org/@typescript-eslint/tsconfig-utils/-/tsconfig-utils-8.57.0.tgz",
+      "integrity": "sha512-LtXRihc5ytjJIQEH+xqjB0+YgsV4/tW35XKX3GTZHpWtcC8SPkT/d4tqdf1cKtesryHm2bgp6l555NYcT2NLvA==",
+      "dev": true,
+      "license": "MIT",
+      "engines": {
+        "node": "^18.18.0 || ^20.9.0 || >=21.1.0"
+      },
+      "funding": {
+        "type": "opencollective",
+        "url": "https://opencollective.com/typescript-eslint"
+      },
+      "peerDependencies": {
+        "typescript": ">=4.8.4 <6.0.0"
+      }
+    },
+    "node_modules/@typescript-eslint/type-utils": {
+      "version": "8.57.0",
+      "resolved": "https://registry.npmjs.org/@typescript-eslint/type-utils/-/type-utils-8.57.0.tgz",
+      "integrity": "sha512-yjgh7gmDcJ1+TcEg8x3uWQmn8ifvSupnPfjP21twPKrDP/pTHlEQgmKcitzF/rzPSmv7QjJ90vRpN4U+zoUjwQ==",
+      "dev": true,
+      "license": "MIT",
+      "dependencies": {
+        "@typescript-eslint/types": "8.57.0",
+        "@typescript-eslint/typescript-estree": "8.57.0",
+        "@typescript-eslint/utils": "8.57.0",
+        "debug": "^4.4.3",
+        "ts-api-utils": "^2.4.0"
+      },
+      "engines": {
+        "node": "^18.18.0 || ^20.9.0 || >=21.1.0"
+      },
+      "funding": {
+        "type": "opencollective",
+        "url": "https://opencollective.com/typescript-eslint"
+      },
+      "peerDependencies": {
+        "eslint": "^8.57.0 || ^9.0.0 || ^10.0.0",
+        "typescript": ">=4.8.4 <6.0.0"
+      }
+    },
+    "node_modules/@typescript-eslint/types": {
+      "version": "8.57.0",
+      "resolved": "https://registry.npmjs.org/@typescript-eslint/types/-/types-8.57.0.tgz",
+      "integrity": "sha512-dTLI8PEXhjUC7B9Kre+u0XznO696BhXcTlOn0/6kf1fHaQW8+VjJAVHJ3eTI14ZapTxdkOmc80HblPQLaEeJdg==",
+      "dev": true,
+      "license": "MIT",
+      "engines": {
+        "node": "^18.18.0 || ^20.9.0 || >=21.1.0"
+      },
+      "funding": {
+        "type": "opencollective",
+        "url": "https://opencollective.com/typescript-eslint"
+      }
+    },
+    "node_modules/@typescript-eslint/typescript-estree": {
+      "version": "8.57.0",
+      "resolved": "https://registry.npmjs.org/@typescript-eslint/typescript-estree/-/typescript-estree-8.57.0.tgz",
+      "integrity": "sha512-m7faHcyVg0BT3VdYTlX8GdJEM7COexXxS6KqGopxdtkQRvBanK377QDHr4W/vIPAR+ah9+B/RclSW5ldVniO1Q==",
+      "dev": true,
+      "license": "MIT",
+      "dependencies": {
+        "@typescript-eslint/project-service": "8.57.0",
+        "@typescript-eslint/tsconfig-utils": "8.57.0",
+        "@typescript-eslint/types": "8.57.0",
+        "@typescript-eslint/visitor-keys": "8.57.0",
+        "debug": "^4.4.3",
+        "minimatch": "^10.2.2",
+        "semver": "^7.7.3",
+        "tinyglobby": "^0.2.15",
+        "ts-api-utils": "^2.4.0"
+      },
+      "engines": {
+        "node": "^18.18.0 || ^20.9.0 || >=21.1.0"
+      },
+      "funding": {
+        "type": "opencollective",
+        "url": "https://opencollective.com/typescript-eslint"
+      },
+      "peerDependencies": {
+        "typescript": ">=4.8.4 <6.0.0"
+      }
+    },
+    "node_modules/@typescript-eslint/typescript-estree/node_modules/balanced-match": {
+      "version": "4.0.4",
+      "resolved": "https://registry.npmjs.org/balanced-match/-/balanced-match-4.0.4.tgz",
+      "integrity": "sha512-BLrgEcRTwX2o6gGxGOCNyMvGSp35YofuYzw9h1IMTRmKqttAZZVU67bdb9Pr2vUHA8+j3i2tJfjO6C6+4myGTA==",
+      "dev": true,
+      "license": "MIT",
+      "engines": {
+        "node": "18 || 20 || >=22"
+      }
+    },
+    "node_modules/@typescript-eslint/typescript-estree/node_modules/brace-expansion": {
+      "version": "5.0.4",
+      "resolved": "https://registry.npmjs.org/brace-expansion/-/brace-expansion-5.0.4.tgz",
+      "integrity": "sha512-h+DEnpVvxmfVefa4jFbCf5HdH5YMDXRsmKflpf1pILZWRFlTbJpxeU55nJl4Smt5HQaGzg1o6RHFPJaOqnmBDg==",
+      "dev": true,
+      "license": "MIT",
+      "dependencies": {
+        "balanced-match": "^4.0.2"
+      },
+      "engines": {
+        "node": "18 || 20 || >=22"
+      }
+    },
+    "node_modules/@typescript-eslint/typescript-estree/node_modules/minimatch": {
+      "version": "10.2.4",
+      "resolved": "https://registry.npmjs.org/minimatch/-/minimatch-10.2.4.tgz",
+      "integrity": "sha512-oRjTw/97aTBN0RHbYCdtF1MQfvusSIBQM0IZEgzl6426+8jSC0nF1a/GmnVLpfB9yyr6g6FTqWqiZVbxrtaCIg==",
+      "dev": true,
+      "license": "BlueOak-1.0.0",
+      "dependencies": {
+        "brace-expansion": "^5.0.2"
+      },
+      "engines": {
+        "node": "18 || 20 || >=22"
+      },
+      "funding": {
+        "url": "https://github.com/sponsors/isaacs"
+      }
+    },
+    "node_modules/@typescript-eslint/utils": {
+      "version": "8.57.0",
+      "resolved": "https://registry.npmjs.org/@typescript-eslint/utils/-/utils-8.57.0.tgz",
+      "integrity": "sha512-5iIHvpD3CZe06riAsbNxxreP+MuYgVUsV0n4bwLH//VJmgtt54sQeY2GszntJ4BjYCpMzrfVh2SBnUQTtys2lQ==",
+      "dev": true,
+      "license": "MIT",
+      "dependencies": {
+        "@eslint-community/eslint-utils": "^4.9.1",
+        "@typescript-eslint/scope-manager": "8.57.0",
+        "@typescript-eslint/types": "8.57.0",
+        "@typescript-eslint/typescript-estree": "8.57.0"
+      },
+      "engines": {
+        "node": "^18.18.0 || ^20.9.0 || >=21.1.0"
+      },
+      "funding": {
+        "type": "opencollective",
+        "url": "https://opencollective.com/typescript-eslint"
+      },
+      "peerDependencies": {
+        "eslint": "^8.57.0 || ^9.0.0 || ^10.0.0",
+        "typescript": ">=4.8.4 <6.0.0"
+      }
+    },
+    "node_modules/@typescript-eslint/visitor-keys": {
+      "version": "8.57.0",
+      "resolved": "https://registry.npmjs.org/@typescript-eslint/visitor-keys/-/visitor-keys-8.57.0.tgz",
+      "integrity": "sha512-zm6xx8UT/Xy2oSr2ZXD0pZo7Jx2XsCoID2IUh9YSTFRu7z+WdwYTRk6LhUftm1crwqbuoF6I8zAFeCMw0YjwDg==",
+      "dev": true,
+      "license": "MIT",
+      "dependencies": {
+        "@typescript-eslint/types": "8.57.0",
+        "eslint-visitor-keys": "^5.0.0"
+      },
+      "engines": {
+        "node": "^18.18.0 || ^20.9.0 || >=21.1.0"
+      },
+      "funding": {
+        "type": "opencollective",
+        "url": "https://opencollective.com/typescript-eslint"
+      }
+    },
+    "node_modules/@typescript-eslint/visitor-keys/node_modules/eslint-visitor-keys": {
+      "version": "5.0.1",
+      "resolved": "https://registry.npmjs.org/eslint-visitor-keys/-/eslint-visitor-keys-5.0.1.tgz",
+      "integrity": "sha512-tD40eHxA35h0PEIZNeIjkHoDR4YjjJp34biM0mDvplBe//mB+IHCqHDGV7pxF+7MklTvighcCPPZC7ynWyjdTA==",
+      "dev": true,
+      "license": "Apache-2.0",
+      "engines": {
+        "node": "^20.19.0 || ^22.13.0 || >=24"
+      },
+      "funding": {
+        "url": "https://opencollective.com/eslint"
+      }
+    },
+    "node_modules/@vitest/coverage-v8": {
+      "version": "4.0.18",
+      "resolved": "https://registry.npmjs.org/@vitest/coverage-v8/-/coverage-v8-4.0.18.tgz",
+      "integrity": "sha512-7i+N2i0+ME+2JFZhfuz7Tg/FqKtilHjGyGvoHYQ6iLV0zahbsJ9sljC9OcFcPDbhYKCet+sG8SsVqlyGvPflZg==",
+      "dev": true,
+      "license": "MIT",
+      "dependencies": {
+        "@bcoe/v8-coverage": "^1.0.2",
+        "@vitest/utils": "4.0.18",
+        "ast-v8-to-istanbul": "^0.3.10",
+        "istanbul-lib-coverage": "^3.2.2",
+        "istanbul-lib-report": "^3.0.1",
+        "istanbul-reports": "^3.2.0",
+        "magicast": "^0.5.1",
+        "obug": "^2.1.1",
+        "std-env": "^3.10.0",
+        "tinyrainbow": "^3.0.3"
+      },
+      "funding": {
+        "url": "https://opencollective.com/vitest"
+      },
+      "peerDependencies": {
+        "@vitest/browser": "4.0.18",
+        "vitest": "4.0.18"
+      },
+      "peerDependenciesMeta": {
+        "@vitest/browser": {
+          "optional": true
+        }
+      }
+    },
+    "node_modules/@vitest/expect": {
+      "version": "4.0.18",
+      "resolved": "https://registry.npmjs.org/@vitest/expect/-/expect-4.0.18.tgz",
+      "integrity": "sha512-8sCWUyckXXYvx4opfzVY03EOiYVxyNrHS5QxX3DAIi5dpJAAkyJezHCP77VMX4HKA2LDT/Jpfo8i2r5BE3GnQQ==",
+      "dev": true,
+      "license": "MIT",
+      "dependencies": {
+        "@standard-schema/spec": "^1.0.0",
+        "@types/chai": "^5.2.2",
+        "@vitest/spy": "4.0.18",
+        "@vitest/utils": "4.0.18",
+        "chai": "^6.2.1",
+        "tinyrainbow": "^3.0.3"
+      },
+      "funding": {
+        "url": "https://opencollective.com/vitest"
+      }
+    },
+    "node_modules/@vitest/mocker": {
+      "version": "4.0.18",
+      "resolved": "https://registry.npmjs.org/@vitest/mocker/-/mocker-4.0.18.tgz",
+      "integrity": "sha512-HhVd0MDnzzsgevnOWCBj5Otnzobjy5wLBe4EdeeFGv8luMsGcYqDuFRMcttKWZA5vVO8RFjexVovXvAM4JoJDQ==",
+      "dev": true,
+      "license": "MIT",
+      "dependencies": {
+        "@vitest/spy": "4.0.18",
+        "estree-walker": "^3.0.3",
+        "magic-string": "^0.30.21"
+      },
+      "funding": {
+        "url": "https://opencollective.com/vitest"
+      },
+      "peerDependencies": {
+        "msw": "^2.4.9",
+        "vite": "^6.0.0 || ^7.0.0-0"
+      },
+      "peerDependenciesMeta": {
+        "msw": {
+          "optional": true
+        },
+        "vite": {
+          "optional": true
+        }
+      }
+    },
+    "node_modules/@vitest/pretty-format": {
+      "version": "4.0.18",
+      "resolved": "https://registry.npmjs.org/@vitest/pretty-format/-/pretty-format-4.0.18.tgz",
+      "integrity": "sha512-P24GK3GulZWC5tz87ux0m8OADrQIUVDPIjjj65vBXYG17ZeU3qD7r+MNZ1RNv4l8CGU2vtTRqixrOi9fYk/yKw==",
+      "dev": true,
+      "license": "MIT",
+      "dependencies": {
+        "tinyrainbow": "^3.0.3"
+      },
+      "funding": {
+        "url": "https://opencollective.com/vitest"
+      }
+    },
+    "node_modules/@vitest/runner": {
+      "version": "4.0.18",
+      "resolved": "https://registry.npmjs.org/@vitest/runner/-/runner-4.0.18.tgz",
+      "integrity": "sha512-rpk9y12PGa22Jg6g5M3UVVnTS7+zycIGk9ZNGN+m6tZHKQb7jrP7/77WfZy13Y/EUDd52NDsLRQhYKtv7XfPQw==",
+      "dev": true,
+      "license": "MIT",
+      "dependencies": {
+        "@vitest/utils": "4.0.18",
+        "pathe": "^2.0.3"
+      },
+      "funding": {
+        "url": "https://opencollective.com/vitest"
+      }
+    },
+    "node_modules/@vitest/snapshot": {
+      "version": "4.0.18",
+      "resolved": "https://registry.npmjs.org/@vitest/snapshot/-/snapshot-4.0.18.tgz",
+      "integrity": "sha512-PCiV0rcl7jKQjbgYqjtakly6T1uwv/5BQ9SwBLekVg/EaYeQFPiXcgrC2Y7vDMA8dM1SUEAEV82kgSQIlXNMvA==",
+      "dev": true,
+      "license": "MIT",
+      "dependencies": {
+        "@vitest/pretty-format": "4.0.18",
+        "magic-string": "^0.30.21",
+        "pathe": "^2.0.3"
+      },
+      "funding": {
+        "url": "https://opencollective.com/vitest"
+      }
+    },
+    "node_modules/@vitest/spy": {
+      "version": "4.0.18",
+      "resolved": "https://registry.npmjs.org/@vitest/spy/-/spy-4.0.18.tgz",
+      "integrity": "sha512-cbQt3PTSD7P2OARdVW3qWER5EGq7PHlvE+QfzSC0lbwO+xnt7+XH06ZzFjFRgzUX//JmpxrCu92VdwvEPlWSNw==",
+      "dev": true,
+      "license": "MIT",
+      "funding": {
+        "url": "https://opencollective.com/vitest"
+      }
+    },
+    "node_modules/@vitest/utils": {
+      "version": "4.0.18",
+      "resolved": "https://registry.npmjs.org/@vitest/utils/-/utils-4.0.18.tgz",
+      "integrity": "sha512-msMRKLMVLWygpK3u2Hybgi4MNjcYJvwTb0Ru09+fOyCXIgT5raYP041DRRdiJiI3k/2U6SEbAETB3YtBrUkCFA==",
+      "dev": true,
+      "license": "MIT",
+      "dependencies": {
+        "@vitest/pretty-format": "4.0.18",
+        "tinyrainbow": "^3.0.3"
+      },
+      "funding": {
+        "url": "https://opencollective.com/vitest"
+      }
+    },
+    "node_modules/acorn": {
+      "version": "8.16.0",
+      "resolved": "https://registry.npmjs.org/acorn/-/acorn-8.16.0.tgz",
+      "integrity": "sha512-UVJyE9MttOsBQIDKw1skb9nAwQuR5wuGD3+82K6JgJlm/Y+KI92oNsMNGZCYdDsVtRHSak0pcV5Dno5+4jh9sw==",
+      "dev": true,
+      "license": "MIT",
+      "bin": {
+        "acorn": "bin/acorn"
+      },
+      "engines": {
+        "node": ">=0.4.0"
+      }
+    },
+    "node_modules/acorn-jsx": {
+      "version": "5.3.2",
+      "resolved": "https://registry.npmjs.org/acorn-jsx/-/acorn-jsx-5.3.2.tgz",
+      "integrity": "sha512-rq9s+JNhf0IChjtDXxllJ7g41oZk5SlXtp0LHwyA5cejwn7vKmKp4pPri6YEePv2PU65sAsegbXtIinmDFDXgQ==",
+      "dev": true,
+      "license": "MIT",
+      "peerDependencies": {
+        "acorn": "^6.0.0 || ^7.0.0 || ^8.0.0"
+      }
+    },
+    "node_modules/ajv": {
+      "version": "6.14.0",
+      "resolved": "https://registry.npmjs.org/ajv/-/ajv-6.14.0.tgz",
+      "integrity": "sha512-IWrosm/yrn43eiKqkfkHis7QioDleaXQHdDVPKg0FSwwd/DuvyX79TZnFOnYpB7dcsFAMmtFztZuXPDvSePkFw==",
+      "dev": true,
+      "license": "MIT",
+      "dependencies": {
+        "fast-deep-equal": "^3.1.1",
+        "fast-json-stable-stringify": "^2.0.0",
+        "json-schema-traverse": "^0.4.1",
+        "uri-js": "^4.2.2"
+      },
+      "funding": {
+        "type": "github",
+        "url": "https://github.com/sponsors/epoberezkin"
+      }
+    },
+    "node_modules/ansi-styles": {
+      "version": "4.3.0",
+      "resolved": "https://registry.npmjs.org/ansi-styles/-/ansi-styles-4.3.0.tgz",
+      "integrity": "sha512-zbB9rCJAT1rbjiVDb2hqKFHNYLxgtk8NURxZ3IZwD3F6NtxbXZQCnnSi1Lkx+IDohdPlFp222wVALIheZJQSEg==",
+      "dev": true,
+      "license": "MIT",
+      "dependencies": {
+        "color-convert": "^2.0.1"
+      },
+      "engines": {
+        "node": ">=8"
+      },
+      "funding": {
+        "url": "https://github.com/chalk/ansi-styles?sponsor=1"
+      }
+    },
+    "node_modules/argparse": {
+      "version": "2.0.1",
+      "resolved": "https://registry.npmjs.org/argparse/-/argparse-2.0.1.tgz",
+      "integrity": "sha512-8+9WqebbFzpX9OR+Wa6O29asIogeRMzcGtAINdpMHHyAg10f05aSFVBbcEqGf/PXw1EjAZ+q2/bEBg3DvurK3Q==",
+      "dev": true,
+      "license": "Python-2.0"
+    },
+    "node_modules/array-find-index": {
+      "version": "1.0.2",
+      "resolved": "https://registry.npmjs.org/array-find-index/-/array-find-index-1.0.2.tgz",
+      "integrity": "sha512-M1HQyIXcBGtVywBt8WVdim+lrNaK7VHp99Qt5pSNziXznKHViIBbXWtfRTpEFpF/c4FdfxNAsCCwPp5phBYJtw==",
+      "dev": true,
+      "license": "MIT",
+      "engines": {
+        "node": ">=0.10.0"
+      }
+    },
+    "node_modules/assertion-error": {
+      "version": "2.0.1",
+      "resolved": "https://registry.npmjs.org/assertion-error/-/assertion-error-2.0.1.tgz",
+      "integrity": "sha512-Izi8RQcffqCeNVgFigKli1ssklIbpHnCYc6AknXGYoB6grJqyeby7jv12JUQgmTAnIDnbck1uxksT4dzN3PWBA==",
+      "dev": true,
+      "license": "MIT",
+      "engines": {
+        "node": ">=12"
+      }
+    },
+    "node_modules/ast-kit": {
+      "version": "3.0.0-beta.1",
+      "resolved": "https://registry.npmjs.org/ast-kit/-/ast-kit-3.0.0-beta.1.tgz",
+      "integrity": "sha512-trmleAnZ2PxN/loHWVhhx1qeOHSRXq4TDsBBxq3GqeJitfk3+jTQ+v/C1km/KYq9M7wKqCewMh+/NAvVH7m+bw==",
+      "dev": true,
+      "license": "MIT",
+      "dependencies": {
+        "@babel/parser": "^8.0.0-beta.4",
+        "estree-walker": "^3.0.3",
+        "pathe": "^2.0.3"
+      },
+      "engines": {
+        "node": ">=20.19.0"
+      },
+      "funding": {
+        "url": "https://github.com/sponsors/sxzz"
+      }
+    },
+    "node_modules/ast-kit/node_modules/@babel/helper-string-parser": {
+      "version": "8.0.0-rc.2",
+      "resolved": "https://registry.npmjs.org/@babel/helper-string-parser/-/helper-string-parser-8.0.0-rc.2.tgz",
+      "integrity": "sha512-noLx87RwlBEMrTzncWd/FvTxoJ9+ycHNg0n8yyYydIoDsLZuxknKgWRJUqcrVkNrJ74uGyhWQzQaS3q8xfGAhQ==",
+      "dev": true,
+      "license": "MIT",
+      "engines": {
+        "node": "^20.19.0 || >=22.12.0"
+      }
+    },
+    "node_modules/ast-kit/node_modules/@babel/helper-validator-identifier": {
+      "version": "8.0.0-rc.2",
+      "resolved": "https://registry.npmjs.org/@babel/helper-validator-identifier/-/helper-validator-identifier-8.0.0-rc.2.tgz",
+      "integrity": "sha512-xExUBkuXWJjVuIbO7z6q7/BA9bgfJDEhVL0ggrggLMbg0IzCUWGT1hZGE8qUH7Il7/RD/a6cZ3AAFrrlp1LF/A==",
+      "dev": true,
+      "license": "MIT",
+      "engines": {
+        "node": "^20.19.0 || >=22.12.0"
+      }
+    },
+    "node_modules/ast-kit/node_modules/@babel/parser": {
+      "version": "8.0.0-rc.2",
+      "resolved": "https://registry.npmjs.org/@babel/parser/-/parser-8.0.0-rc.2.tgz",
+      "integrity": "sha512-29AhEtcq4x8Dp3T72qvUMZHx0OMXCj4Jy/TEReQa+KWLln524Cj1fWb3QFi0l/xSpptQBR6y9RNEXuxpFvwiUQ==",
+      "dev": true,
+      "license": "MIT",
+      "dependencies": {
+        "@babel/types": "^8.0.0-rc.2"
+      },
+      "bin": {
+        "parser": "bin/babel-parser.js"
+      },
+      "engines": {
+        "node": "^20.19.0 || >=22.12.0"
+      }
+    },
+    "node_modules/ast-kit/node_modules/@babel/types": {
+      "version": "8.0.0-rc.2",
+      "resolved": "https://registry.npmjs.org/@babel/types/-/types-8.0.0-rc.2.tgz",
+      "integrity": "sha512-91gAaWRznDwSX4E2tZ1YjBuIfnQVOFDCQ2r0Toby0gu4XEbyF623kXLMA8d4ZbCu+fINcrudkmEcwSUHgDDkNw==",
+      "dev": true,
+      "license": "MIT",
+      "dependencies": {
+        "@babel/helper-string-parser": "^8.0.0-rc.2",
+        "@babel/helper-validator-identifier": "^8.0.0-rc.2"
+      },
+      "engines": {
+        "node": "^20.19.0 || >=22.12.0"
+      }
+    },
+    "node_modules/ast-v8-to-istanbul": {
+      "version": "0.3.12",
+      "resolved": "https://registry.npmjs.org/ast-v8-to-istanbul/-/ast-v8-to-istanbul-0.3.12.tgz",
+      "integrity": "sha512-BRRC8VRZY2R4Z4lFIL35MwNXmwVqBityvOIwETtsCSwvjl0IdgFsy9NhdaA6j74nUdtJJlIypeRhpDam19Wq3g==",
+      "dev": true,
+      "license": "MIT",
+      "dependencies": {
+        "@jridgewell/trace-mapping": "^0.3.31",
+        "estree-walker": "^3.0.3",
+        "js-tokens": "^10.0.0"
+      }
+    },
+    "node_modules/automd": {
+      "version": "0.4.3",
+      "resolved": "https://registry.npmjs.org/automd/-/automd-0.4.3.tgz",
+      "integrity": "sha512-5WJNEiaNpFm8h0OmQzhnESthadUQhJwQfka/TmmJpMudZ8qU9MZao9p0G1g7WYA9pVTz6FMMOSvxnfQ9g8q9vQ==",
+      "dev": true,
+      "license": "MIT",
+      "dependencies": {
+        "@parcel/watcher": "^2.5.6",
+        "c12": "^3.3.3",
+        "citty": "^0.2.0",
+        "consola": "^3.4.2",
+        "defu": "^6.1.4",
+        "destr": "^2.0.5",
+        "didyoumean2": "^7.0.4",
+        "magic-string": "^0.30.21",
+        "mdbox": "^0.1.1",
+        "mlly": "^1.8.0",
+        "ofetch": "^1.5.1",
+        "pathe": "^2.0.3",
+        "perfect-debounce": "^2.1.0",
+        "pkg-types": "^2.3.0",
+        "scule": "^1.3.0",
+        "tinyglobby": "^0.2.15",
+        "untyped": "^2.0.0"
+      },
+      "bin": {
+        "automd": "dist/cli.mjs"
+      }
+    },
+    "node_modules/balanced-match": {
+      "version": "1.0.2",
+      "resolved": "https://registry.npmjs.org/balanced-match/-/balanced-match-1.0.2.tgz",
+      "integrity": "sha512-3oSeUO0TMV67hN1AmbXsK4yaqU7tjiHlbxRDZOpH0KW9+CeX4bRAaX0Anxt0tx2MrpRpWwQaPwIlISEJhYU5Pw==",
+      "dev": true,
+      "license": "MIT"
+    },
+    "node_modules/baseline-browser-mapping": {
+      "version": "2.10.0",
+      "resolved": "https://registry.npmjs.org/baseline-browser-mapping/-/baseline-browser-mapping-2.10.0.tgz",
+      "integrity": "sha512-lIyg0szRfYbiy67j9KN8IyeD7q7hcmqnJ1ddWmNt19ItGpNN64mnllmxUNFIOdOm6by97jlL6wfpTTJrmnjWAA==",
+      "dev": true,
+      "license": "Apache-2.0",
+      "bin": {
+        "baseline-browser-mapping": "dist/cli.cjs"
+      },
+      "engines": {
+        "node": ">=6.0.0"
+      }
+    },
+    "node_modules/birpc": {
+      "version": "4.0.0",
+      "resolved": "https://registry.npmjs.org/birpc/-/birpc-4.0.0.tgz",
+      "integrity": "sha512-LShSxJP0KTmd101b6DRyGBj57LZxSDYWKitQNW/mi8GRMvZb078Uf9+pveax1DrVL89vm7mWe+TovdI/UDOuPw==",
+      "dev": true,
+      "license": "MIT",
+      "funding": {
+        "url": "https://github.com/sponsors/antfu"
+      }
+    },
+    "node_modules/brace-expansion": {
+      "version": "1.1.12",
+      "resolved": "https://registry.npmjs.org/brace-expansion/-/brace-expansion-1.1.12.tgz",
+      "integrity": "sha512-9T9UjW3r0UW5c1Q7GTwllptXwhvYmEzFhzMfZ9H7FQWt+uZePjZPjBP/W1ZEyZ1twGWom5/56TF4lPcqjnDHcg==",
+      "dev": true,
+      "license": "MIT",
+      "dependencies": {
+        "balanced-match": "^1.0.0",
+        "concat-map": "0.0.1"
+      }
+    },
+    "node_modules/browserslist": {
+      "version": "4.28.1",
+      "resolved": "https://registry.npmjs.org/browserslist/-/browserslist-4.28.1.tgz",
+      "integrity": "sha512-ZC5Bd0LgJXgwGqUknZY/vkUQ04r8NXnJZ3yYi4vDmSiZmC/pdSN0NbNRPxZpbtO4uAfDUAFffO8IZoM3Gj8IkA==",
+      "dev": true,
+      "funding": [
+        {
+          "type": "opencollective",
+          "url": "https://opencollective.com/browserslist"
+        },
+        {
+          "type": "tidelift",
+          "url": "https://tidelift.com/funding/github/npm/browserslist"
+        },
+        {
+          "type": "github",
+          "url": "https://github.com/sponsors/ai"
+        }
+      ],
+      "license": "MIT",
+      "dependencies": {
+        "baseline-browser-mapping": "^2.9.0",
+        "caniuse-lite": "^1.0.30001759",
+        "electron-to-chromium": "^1.5.263",
+        "node-releases": "^2.0.27",
+        "update-browserslist-db": "^1.2.0"
+      },
+      "bin": {
+        "browserslist": "cli.js"
+      },
+      "engines": {
+        "node": "^6 || ^7 || ^8 || ^9 || ^10 || ^11 || ^12 || >=13.7"
+      }
+    },
+    "node_modules/builtin-modules": {
+      "version": "5.0.0",
+      "resolved": "https://registry.npmjs.org/builtin-modules/-/builtin-modules-5.0.0.tgz",
+      "integrity": "sha512-bkXY9WsVpY7CvMhKSR6pZilZu9Ln5WDrKVBUXf2S443etkmEO4V58heTecXcUIsNsi4Rx8JUO4NfX1IcQl4deg==",
+      "dev": true,
+      "license": "MIT",
+      "engines": {
+        "node": ">=18.20"
+      },
+      "funding": {
+        "url": "https://github.com/sponsors/sindresorhus"
+      }
+    },
+    "node_modules/bundle-name": {
+      "version": "4.1.0",
+      "resolved": "https://registry.npmjs.org/bundle-name/-/bundle-name-4.1.0.tgz",
+      "integrity": "sha512-tjwM5exMg6BGRI+kNmTntNsvdZS1X8BFYS6tnJ2hdH0kVxM6/eVZ2xy+FqStSWvYmtfFMDLIxurorHwDKfDz5Q==",
+      "dev": true,
+      "license": "MIT",
+      "dependencies": {
+        "run-applescript": "^7.0.0"
+      },
+      "engines": {
+        "node": ">=18"
+      },
+      "funding": {
+        "url": "https://github.com/sponsors/sindresorhus"
+      }
+    },
+    "node_modules/c12": {
+      "version": "3.3.3",
+      "resolved": "https://registry.npmjs.org/c12/-/c12-3.3.3.tgz",
+      "integrity": "sha512-750hTRvgBy5kcMNPdh95Qo+XUBeGo8C7nsKSmedDmaQI+E0r82DwHeM6vBewDe4rGFbnxoa4V9pw+sPh5+Iz8Q==",
+      "dev": true,
+      "license": "MIT",
+      "dependencies": {
+        "chokidar": "^5.0.0",
+        "confbox": "^0.2.2",
+        "defu": "^6.1.4",
+        "dotenv": "^17.2.3",
+        "exsolve": "^1.0.8",
+        "giget": "^2.0.0",
+        "jiti": "^2.6.1",
+        "ohash": "^2.0.11",
+        "pathe": "^2.0.3",
+        "perfect-debounce": "^2.0.0",
+        "pkg-types": "^2.3.0",
+        "rc9": "^2.1.2"
+      },
+      "peerDependencies": {
+        "magicast": "*"
+      },
+      "peerDependenciesMeta": {
+        "magicast": {
+          "optional": true
+        }
+      }
+    },
+    "node_modules/callsites": {
+      "version": "3.1.0",
+      "resolved": "https://registry.npmjs.org/callsites/-/callsites-3.1.0.tgz",
+      "integrity": "sha512-P8BjAsXvZS+VIDUI11hHCQEv74YT67YUi5JJFNWIqL235sBmjX4+qx9Muvls5ivyNENctx46xQLQ3aTuE7ssaQ==",
+      "dev": true,
+      "license": "MIT",
+      "engines": {
+        "node": ">=6"
+      }
+    },
+    "node_modules/caniuse-lite": {
+      "version": "1.0.30001777",
+      "resolved": "https://registry.npmjs.org/caniuse-lite/-/caniuse-lite-1.0.30001777.tgz",
+      "integrity": "sha512-tmN+fJxroPndC74efCdp12j+0rk0RHwV5Jwa1zWaFVyw2ZxAuPeG8ZgWC3Wz7uSjT3qMRQ5XHZ4COgQmsCMJAQ==",
+      "dev": true,
+      "funding": [
+        {
+          "type": "opencollective",
+          "url": "https://opencollective.com/browserslist"
+        },
+        {
+          "type": "tidelift",
+          "url": "https://tidelift.com/funding/github/npm/caniuse-lite"
+        },
+        {
+          "type": "github",
+          "url": "https://github.com/sponsors/ai"
+        }
+      ],
+      "license": "CC-BY-4.0"
+    },
+    "node_modules/ccount": {
+      "version": "2.0.1",
+      "resolved": "https://registry.npmjs.org/ccount/-/ccount-2.0.1.tgz",
+      "integrity": "sha512-eyrF0jiFpY+3drT6383f1qhkbGsLSifNAjA61IUjZjmLCWjItY6LB9ft9YhoDgwfmclB2zhu51Lc7+95b8NRAg==",
+      "dev": true,
+      "license": "MIT",
+      "funding": {
+        "type": "github",
+        "url": "https://github.com/sponsors/wooorm"
+      }
+    },
+    "node_modules/chai": {
+      "version": "6.2.2",
+      "resolved": "https://registry.npmjs.org/chai/-/chai-6.2.2.tgz",
+      "integrity": "sha512-NUPRluOfOiTKBKvWPtSD4PhFvWCqOi0BGStNWs57X9js7XGTprSmFoz5F0tWhR4WPjNeR9jXqdC7/UpSJTnlRg==",
+      "dev": true,
+      "license": "MIT",
+      "engines": {
+        "node": ">=18"
+      }
+    },
+    "node_modules/chalk": {
+      "version": "4.1.2",
+      "resolved": "https://registry.npmjs.org/chalk/-/chalk-4.1.2.tgz",
+      "integrity": "sha512-oKnbhFyRIXpUuez8iBMmyEa4nbj4IOQyuhc/wy9kY7/WVPcwIO9VA668Pu8RkO7+0G76SLROeyw9CpQ061i4mA==",
+      "dev": true,
+      "license": "MIT",
+      "dependencies": {
+        "ansi-styles": "^4.1.0",
+        "supports-color": "^7.1.0"
+      },
+      "engines": {
+        "node": ">=10"
+      },
+      "funding": {
+        "url": "https://github.com/chalk/chalk?sponsor=1"
+      }
+    },
+    "node_modules/change-case": {
+      "version": "5.4.4",
+      "resolved": "https://registry.npmjs.org/change-case/-/change-case-5.4.4.tgz",
+      "integrity": "sha512-HRQyTk2/YPEkt9TnUPbOpr64Uw3KOicFWPVBb+xiHvd6eBx/qPr9xqfBFDT8P2vWsvvz4jbEkfDe71W3VyNu2w==",
+      "dev": true,
+      "license": "MIT"
+    },
+    "node_modules/changelogen": {
+      "version": "0.6.2",
+      "resolved": "https://registry.npmjs.org/changelogen/-/changelogen-0.6.2.tgz",
+      "integrity": "sha512-QtC7+r9BxoUm+XDAwhLbz3CgU134J1ytfE3iCpLpA4KFzX2P1e6s21RrWDwUBzfx66b1Rv+6lOA2nS2btprd+A==",
+      "dev": true,
+      "license": "MIT",
+      "dependencies": {
+        "c12": "^3.0.4",
+        "confbox": "^0.2.2",
+        "consola": "^3.4.2",
+        "convert-gitmoji": "^0.1.5",
+        "mri": "^1.2.0",
+        "node-fetch-native": "^1.6.6",
+        "ofetch": "^1.4.1",
+        "open": "^10.1.2",
+        "pathe": "^2.0.3",
+        "pkg-types": "^2.2.0",
+        "scule": "^1.3.0",
+        "semver": "^7.7.2",
+        "std-env": "^3.9.0"
+      },
+      "bin": {
+        "changelogen": "dist/cli.mjs"
+      }
+    },
+    "node_modules/character-entities": {
+      "version": "2.0.2",
+      "resolved": "https://registry.npmjs.org/character-entities/-/character-entities-2.0.2.tgz",
+      "integrity": "sha512-shx7oQ0Awen/BRIdkjkvz54PnEEI/EjwXDSIZp86/KKdbafHh1Df/RYGBhn4hbe2+uKC9FnT5UCEdyPz3ai9hQ==",
+      "dev": true,
+      "license": "MIT",
+      "funding": {
+        "type": "github",
+        "url": "https://github.com/sponsors/wooorm"
+      }
+    },
+    "node_modules/chokidar": {
+      "version": "5.0.0",
+      "resolved": "https://registry.npmjs.org/chokidar/-/chokidar-5.0.0.tgz",
+      "integrity": "sha512-TQMmc3w+5AxjpL8iIiwebF73dRDF4fBIieAqGn9RGCWaEVwQ6Fb2cGe31Yns0RRIzii5goJ1Y7xbMwo1TxMplw==",
+      "dev": true,
+      "license": "MIT",
+      "dependencies": {
+        "readdirp": "^5.0.0"
+      },
+      "engines": {
+        "node": ">= 20.19.0"
+      },
+      "funding": {
+        "url": "https://paulmillr.com/funding/"
+      }
+    },
+    "node_modules/ci-info": {
+      "version": "4.4.0",
+      "resolved": "https://registry.npmjs.org/ci-info/-/ci-info-4.4.0.tgz",
+      "integrity": "sha512-77PSwercCZU2Fc4sX94eF8k8Pxte6JAwL4/ICZLFjJLqegs7kCuAsqqj/70NQF6TvDpgFjkubQB2FW2ZZddvQg==",
+      "dev": true,
+      "funding": [
+        {
+          "type": "github",
+          "url": "https://github.com/sponsors/sibiraj-s"
+        }
+      ],
+      "license": "MIT",
+      "engines": {
+        "node": ">=8"
+      }
+    },
+    "node_modules/citty": {
+      "version": "0.2.1",
+      "resolved": "https://registry.npmjs.org/citty/-/citty-0.2.1.tgz",
+      "integrity": "sha512-kEV95lFBhQgtogAPlQfJJ0WGVSokvLr/UEoFPiKKOXF7pl98HfUVUD0ejsuTCld/9xH9vogSywZ5KqHzXrZpqg==",
+      "dev": true,
+      "license": "MIT"
+    },
+    "node_modules/clean-regexp": {
+      "version": "1.0.0",
+      "resolved": "https://registry.npmjs.org/clean-regexp/-/clean-regexp-1.0.0.tgz",
+      "integrity": "sha512-GfisEZEJvzKrmGWkvfhgzcz/BllN1USeqD2V6tg14OAOgaCD2Z/PUEuxnAZ/nPvmaHRG7a8y77p1T/IRQ4D1Hw==",
+      "dev": true,
+      "license": "MIT",
+      "dependencies": {
+        "escape-string-regexp": "^1.0.5"
+      },
+      "engines": {
+        "node": ">=4"
+      }
+    },
+    "node_modules/clean-regexp/node_modules/escape-string-regexp": {
+      "version": "1.0.5",
+      "resolved": "https://registry.npmjs.org/escape-string-regexp/-/escape-string-regexp-1.0.5.tgz",
+      "integrity": "sha512-vbRorB5FUQWvla16U8R/qgaFIya2qGzwDrNmCZuYKrbdSUMG6I1ZCGQRefkRVhuOkIGVne7BQ35DSfo1qvJqFg==",
+      "dev": true,
+      "license": "MIT",
+      "engines": {
+        "node": ">=0.8.0"
+      }
+    },
+    "node_modules/color-convert": {
+      "version": "2.0.1",
+      "resolved": "https://registry.npmjs.org/color-convert/-/color-convert-2.0.1.tgz",
+      "integrity": "sha512-RRECPsj7iu/xb5oKYcsFHSppFNnsj/52OVTRKb4zP5onXwVF3zVmmToNcOfGC+CRDpfK/U584fMg38ZHCaElKQ==",
+      "dev": true,
+      "license": "MIT",
+      "dependencies": {
+        "color-name": "~1.1.4"
+      },
+      "engines": {
+        "node": ">=7.0.0"
+      }
+    },
+    "node_modules/color-name": {
+      "version": "1.1.4",
+      "resolved": "https://registry.npmjs.org/color-name/-/color-name-1.1.4.tgz",
+      "integrity": "sha512-dOy+3AuW3a2wNbZHIuMZpTcgjGuLU/uBL/ubcZF9OXbDo8ff4O8yVp5Bf0efS8uEoYo5q4Fx7dY9OgQGXgAsQA==",
+      "dev": true,
+      "license": "MIT"
+    },
+    "node_modules/commenting": {
+      "version": "1.1.0",
+      "resolved": "https://registry.npmjs.org/commenting/-/commenting-1.1.0.tgz",
+      "integrity": "sha512-YeNK4tavZwtH7jEgK1ZINXzLKm6DZdEMfsaaieOsCAN0S8vsY7UeuO3Q7d/M018EFgE+IeUAuBOKkFccBZsUZA==",
+      "dev": true,
+      "license": "MIT"
+    },
+    "node_modules/concat-map": {
+      "version": "0.0.1",
+      "resolved": "https://registry.npmjs.org/concat-map/-/concat-map-0.0.1.tgz",
+      "integrity": "sha512-/Srv4dswyQNBfohGpz9o6Yb3Gz3SrUDqBH5rTuhGR7ahtlbYKnVxw2bCFMRljaA7EXHaXZ8wsHdodFvbkhKmqg==",
+      "dev": true,
+      "license": "MIT"
+    },
+    "node_modules/confbox": {
+      "version": "0.2.4",
+      "resolved": "https://registry.npmjs.org/confbox/-/confbox-0.2.4.tgz",
+      "integrity": "sha512-ysOGlgTFbN2/Y6Cg3Iye8YKulHw+R2fNXHrgSmXISQdMnomY6eNDprVdW9R5xBguEqI954+S6709UyiO7B+6OQ==",
+      "dev": true,
+      "license": "MIT"
+    },
+    "node_modules/consola": {
+      "version": "3.4.2",
+      "resolved": "https://registry.npmjs.org/consola/-/consola-3.4.2.tgz",
+      "integrity": "sha512-5IKcdX0nnYavi6G7TtOhwkYzyjfJlatbjMjuLSfE2kYT5pMDOilZ4OvMhi637CcDICTmz3wARPoyhqyX1Y+XvA==",
+      "dev": true,
+      "license": "MIT",
+      "engines": {
+        "node": "^14.18.0 || >=16.10.0"
+      }
+    },
+    "node_modules/convert-gitmoji": {
+      "version": "0.1.5",
+      "resolved": "https://registry.npmjs.org/convert-gitmoji/-/convert-gitmoji-0.1.5.tgz",
+      "integrity": "sha512-4wqOafJdk2tqZC++cjcbGcaJ13BZ3kwldf06PTiAQRAB76Z1KJwZNL1SaRZMi2w1FM9RYTgZ6QErS8NUl/GBmQ==",
+      "dev": true,
+      "license": "MIT"
+    },
+    "node_modules/core-js-compat": {
+      "version": "3.48.0",
+      "resolved": "https://registry.npmjs.org/core-js-compat/-/core-js-compat-3.48.0.tgz",
+      "integrity": "sha512-OM4cAF3D6VtH/WkLtWvyNC56EZVXsZdU3iqaMG2B4WvYrlqU831pc4UtG5yp0sE9z8Y02wVN7PjW5Zf9Gt0f1Q==",
+      "dev": true,
+      "license": "MIT",
+      "dependencies": {
+        "browserslist": "^4.28.1"
+      },
+      "funding": {
+        "type": "opencollective",
+        "url": "https://opencollective.com/core-js"
+      }
+    },
+    "node_modules/cross-spawn": {
+      "version": "7.0.6",
+      "resolved": "https://registry.npmjs.org/cross-spawn/-/cross-spawn-7.0.6.tgz",
+      "integrity": "sha512-uV2QOWP2nWzsy2aMp8aRibhi9dlzF5Hgh5SHaB9OiTGEyDTiJJyx0uy51QXdyWbtAHNua4XJzUKca3OzKUd3vA==",
+      "dev": true,
+      "license": "MIT",
+      "dependencies": {
+        "path-key": "^3.1.0",
+        "shebang-command": "^2.0.0",
+        "which": "^2.0.1"
+      },
+      "engines": {
+        "node": ">= 8"
+      }
+    },
+    "node_modules/debug": {
+      "version": "4.4.3",
+      "resolved": "https://registry.npmjs.org/debug/-/debug-4.4.3.tgz",
+      "integrity": "sha512-RGwwWnwQvkVfavKVt22FGLw+xYSdzARwm0ru6DhTVA3umU5hZc28V3kO4stgYryrTlLpuvgI9GiijltAjNbcqA==",
+      "dev": true,
+      "license": "MIT",
+      "dependencies": {
+        "ms": "^2.1.3"
+      },
+      "engines": {
+        "node": ">=6.0"
+      },
+      "peerDependenciesMeta": {
+        "supports-color": {
+          "optional": true
+        }
+      }
+    },
+    "node_modules/decode-named-character-reference": {
+      "version": "1.3.0",
+      "resolved": "https://registry.npmjs.org/decode-named-character-reference/-/decode-named-character-reference-1.3.0.tgz",
+      "integrity": "sha512-GtpQYB283KrPp6nRw50q3U9/VfOutZOe103qlN7BPP6Ad27xYnOIWv4lPzo8HCAL+mMZofJ9KEy30fq6MfaK6Q==",
+      "dev": true,
+      "license": "MIT",
+      "dependencies": {
+        "character-entities": "^2.0.0"
+      },
+      "funding": {
+        "type": "github",
+        "url": "https://github.com/sponsors/wooorm"
+      }
+    },
+    "node_modules/deep-is": {
+      "version": "0.1.4",
+      "resolved": "https://registry.npmjs.org/deep-is/-/deep-is-0.1.4.tgz",
+      "integrity": "sha512-oIPzksmTg4/MriiaYGO+okXDT7ztn/w3Eptv/+gSIdMdKsJo0u4CfYNFJPy+4SKMuCqGw2wxnA+URMg3t8a/bQ==",
+      "dev": true,
+      "license": "MIT"
+    },
+    "node_modules/default-browser": {
+      "version": "5.5.0",
+      "resolved": "https://registry.npmjs.org/default-browser/-/default-browser-5.5.0.tgz",
+      "integrity": "sha512-H9LMLr5zwIbSxrmvikGuI/5KGhZ8E2zH3stkMgM5LpOWDutGM2JZaj460Udnf1a+946zc7YBgrqEWwbk7zHvGw==",
+      "dev": true,
+      "license": "MIT",
+      "dependencies": {
+        "bundle-name": "^4.1.0",
+        "default-browser-id": "^5.0.0"
+      },
+      "engines": {
+        "node": ">=18"
+      },
+      "funding": {
+        "url": "https://github.com/sponsors/sindresorhus"
+      }
+    },
+    "node_modules/default-browser-id": {
+      "version": "5.0.1",
+      "resolved": "https://registry.npmjs.org/default-browser-id/-/default-browser-id-5.0.1.tgz",
+      "integrity": "sha512-x1VCxdX4t+8wVfd1so/9w+vQ4vx7lKd2Qp5tDRutErwmR85OgmfX7RlLRMWafRMY7hbEiXIbudNrjOAPa/hL8Q==",
+      "dev": true,
+      "license": "MIT",
+      "engines": {
+        "node": ">=18"
+      },
+      "funding": {
+        "url": "https://github.com/sponsors/sindresorhus"
+      }
+    },
+    "node_modules/define-lazy-prop": {
+      "version": "3.0.0",
+      "resolved": "https://registry.npmjs.org/define-lazy-prop/-/define-lazy-prop-3.0.0.tgz",
+      "integrity": "sha512-N+MeXYoqr3pOgn8xfyRPREN7gHakLYjhsHhWGT3fWAiL4IkAt0iDw14QiiEm2bE30c5XX5q0FtAA3CK5f9/BUg==",
+      "dev": true,
+      "license": "MIT",
+      "engines": {
+        "node": ">=12"
+      },
+      "funding": {
+        "url": "https://github.com/sponsors/sindresorhus"
+      }
+    },
+    "node_modules/defu": {
+      "version": "6.1.4",
+      "resolved": "https://registry.npmjs.org/defu/-/defu-6.1.4.tgz",
+      "integrity": "sha512-mEQCMmwJu317oSz8CwdIOdwf3xMif1ttiM8LTufzc3g6kR+9Pe236twL8j3IYT1F7GfRgGcW6MWxzZjLIkuHIg==",
+      "dev": true,
+      "license": "MIT"
+    },
+    "node_modules/dequal": {
+      "version": "2.0.3",
+      "resolved": "https://registry.npmjs.org/dequal/-/dequal-2.0.3.tgz",
+      "integrity": "sha512-0je+qPKHEMohvfRTCEo3CrPG6cAzAYgmzKyxRiYSSDkS6eGJdyVJm7WaYA5ECaAD9wLB2T4EEeymA5aFVcYXCA==",
+      "dev": true,
+      "license": "MIT",
+      "engines": {
+        "node": ">=6"
+      }
+    },
+    "node_modules/destr": {
+      "version": "2.0.5",
+      "resolved": "https://registry.npmjs.org/destr/-/destr-2.0.5.tgz",
+      "integrity": "sha512-ugFTXCtDZunbzasqBxrK93Ik/DRYsO6S/fedkWEMKqt04xZ4csmnmwGDBAb07QWNaGMAmnTIemsYZCksjATwsA==",
+      "dev": true,
+      "license": "MIT"
+    },
+    "node_modules/detect-libc": {
+      "version": "2.1.2",
+      "resolved": "https://registry.npmjs.org/detect-libc/-/detect-libc-2.1.2.tgz",
+      "integrity": "sha512-Btj2BOOO83o3WyH59e8MgXsxEQVcarkUOpEYrubB0urwnN10yQ364rsiByU11nZlqWYZm05i/of7io4mzihBtQ==",
+      "dev": true,
+      "license": "Apache-2.0",
+      "engines": {
+        "node": ">=8"
+      }
+    },
+    "node_modules/devlop": {
+      "version": "1.1.0",
+      "resolved": "https://registry.npmjs.org/devlop/-/devlop-1.1.0.tgz",
+      "integrity": "sha512-RWmIqhcFf1lRYBvNmr7qTNuyCt/7/ns2jbpp1+PalgE/rDQcBT0fioSMUpJ93irlUhC5hrg4cYqe6U+0ImW0rA==",
+      "dev": true,
+      "license": "MIT",
+      "dependencies": {
+        "dequal": "^2.0.0"
+      },
+      "funding": {
+        "type": "github",
+        "url": "https://github.com/sponsors/wooorm"
+      }
+    },
+    "node_modules/didyoumean2": {
+      "version": "7.0.4",
+      "resolved": "https://registry.npmjs.org/didyoumean2/-/didyoumean2-7.0.4.tgz",
+      "integrity": "sha512-+yW4SNY7W2DOWe2Jx5H4c2qMTFbLGM6wIyoDPkAPy66X+sD1KfYjBPAIWPVsYqMxelflaMQCloZDudELIPhLqA==",
+      "dev": true,
+      "license": "MIT",
+      "dependencies": {
+        "@babel/runtime": "^7.24.8",
+        "fastest-levenshtein": "^1.0.16",
+        "lodash.deburr": "^4.1.0"
+      },
+      "engines": {
+        "node": "^18.12.0 || >=20.9.0"
+      }
+    },
+    "node_modules/dotenv": {
+      "version": "17.3.1",
+      "resolved": "https://registry.npmjs.org/dotenv/-/dotenv-17.3.1.tgz",
+      "integrity": "sha512-IO8C/dzEb6O3F9/twg6ZLXz164a2fhTnEWb95H23Dm4OuN+92NmEAlTrupP9VW6Jm3sO26tQlqyvyi4CsnY9GA==",
+      "dev": true,
+      "license": "BSD-2-Clause",
+      "engines": {
+        "node": ">=12"
+      },
+      "funding": {
+        "url": "https://dotenvx.com"
+      }
+    },
+    "node_modules/dts-resolver": {
+      "version": "2.1.3",
+      "resolved": "https://registry.npmjs.org/dts-resolver/-/dts-resolver-2.1.3.tgz",
+      "integrity": "sha512-bihc7jPC90VrosXNzK0LTE2cuLP6jr0Ro8jk+kMugHReJVLIpHz/xadeq3MhuwyO4TD4OA3L1Q8pBBFRc08Tsw==",
+      "dev": true,
+      "license": "MIT",
+      "engines": {
+        "node": ">=20.19.0"
+      },
+      "funding": {
+        "url": "https://github.com/sponsors/sxzz"
+      },
+      "peerDependencies": {
+        "oxc-resolver": ">=11.0.0"
+      },
+      "peerDependenciesMeta": {
+        "oxc-resolver": {
+          "optional": true
+        }
+      }
+    },
+    "node_modules/electron-to-chromium": {
+      "version": "1.5.307",
+      "resolved": "https://registry.npmjs.org/electron-to-chromium/-/electron-to-chromium-1.5.307.tgz",
+      "integrity": "sha512-5z3uFKBWjiNR44nFcYdkcXjKMbg5KXNdciu7mhTPo9tB7NbqSNP2sSnGR+fqknZSCwKkBN+oxiiajWs4dT6ORg==",
+      "dev": true,
+      "license": "ISC"
+    },
+    "node_modules/es-module-lexer": {
+      "version": "1.7.0",
+      "resolved": "https://registry.npmjs.org/es-module-lexer/-/es-module-lexer-1.7.0.tgz",
+      "integrity": "sha512-jEQoCwk8hyb2AZziIOLhDqpm5+2ww5uIE6lkO/6jcOCusfk6LhMHpXXfBLXTZ7Ydyt0j4VoUQv6uGNYbdW+kBA==",
+      "dev": true,
+      "license": "MIT"
+    },
+    "node_modules/esbuild": {
+      "version": "0.27.3",
+      "resolved": "https://registry.npmjs.org/esbuild/-/esbuild-0.27.3.tgz",
+      "integrity": "sha512-8VwMnyGCONIs6cWue2IdpHxHnAjzxnw2Zr7MkVxB2vjmQ2ivqGFb4LEG3SMnv0Gb2F/G/2yA8zUaiL1gywDCCg==",
+      "dev": true,
+      "hasInstallScript": true,
+      "license": "MIT",
+      "bin": {
+        "esbuild": "bin/esbuild"
+      },
+      "engines": {
+        "node": ">=18"
+      },
+      "optionalDependencies": {
+        "@esbuild/aix-ppc64": "0.27.3",
+        "@esbuild/android-arm": "0.27.3",
+        "@esbuild/android-arm64": "0.27.3",
+        "@esbuild/android-x64": "0.27.3",
+        "@esbuild/darwin-arm64": "0.27.3",
+        "@esbuild/darwin-x64": "0.27.3",
+        "@esbuild/freebsd-arm64": "0.27.3",
+        "@esbuild/freebsd-x64": "0.27.3",
+        "@esbuild/linux-arm": "0.27.3",
+        "@esbuild/linux-arm64": "0.27.3",
+        "@esbuild/linux-ia32": "0.27.3",
+        "@esbuild/linux-loong64": "0.27.3",
+        "@esbuild/linux-mips64el": "0.27.3",
+        "@esbuild/linux-ppc64": "0.27.3",
+        "@esbuild/linux-riscv64": "0.27.3",
+        "@esbuild/linux-s390x": "0.27.3",
+        "@esbuild/linux-x64": "0.27.3",
+        "@esbuild/netbsd-arm64": "0.27.3",
+        "@esbuild/netbsd-x64": "0.27.3",
+        "@esbuild/openbsd-arm64": "0.27.3",
+        "@esbuild/openbsd-x64": "0.27.3",
+        "@esbuild/openharmony-arm64": "0.27.3",
+        "@esbuild/sunos-x64": "0.27.3",
+        "@esbuild/win32-arm64": "0.27.3",
+        "@esbuild/win32-ia32": "0.27.3",
+        "@esbuild/win32-x64": "0.27.3"
+      }
+    },
+    "node_modules/escalade": {
+      "version": "3.2.0",
+      "resolved": "https://registry.npmjs.org/escalade/-/escalade-3.2.0.tgz",
+      "integrity": "sha512-WUj2qlxaQtO4g6Pq5c29GTcWGDyd8itL8zTlipgECz3JesAiiOKotd8JU6otB3PACgG6xkJUyVhboMS+bje/jA==",
+      "dev": true,
+      "license": "MIT",
+      "engines": {
+        "node": ">=6"
+      }
+    },
+    "node_modules/escape-string-regexp": {
+      "version": "4.0.0",
+      "resolved": "https://registry.npmjs.org/escape-string-regexp/-/escape-string-regexp-4.0.0.tgz",
+      "integrity": "sha512-TtpcNJ3XAzx3Gq8sWRzJaVajRs0uVxA2YAkdb1jm2YkPz4G6egUFAyA3n5vtEIZefPk5Wa4UXbKuS5fKkJWdgA==",
+      "dev": true,
+      "license": "MIT",
+      "engines": {
+        "node": ">=10"
+      },
+      "funding": {
+        "url": "https://github.com/sponsors/sindresorhus"
+      }
+    },
+    "node_modules/eslint": {
+      "version": "9.39.4",
+      "resolved": "https://registry.npmjs.org/eslint/-/eslint-9.39.4.tgz",
+      "integrity": "sha512-XoMjdBOwe/esVgEvLmNsD3IRHkm7fbKIUGvrleloJXUZgDHig2IPWNniv+GwjyJXzuNqVjlr5+4yVUZjycJwfQ==",
+      "dev": true,
+      "license": "MIT",
+      "dependencies": {
+        "@eslint-community/eslint-utils": "^4.8.0",
+        "@eslint-community/regexpp": "^4.12.1",
+        "@eslint/config-array": "^0.21.2",
+        "@eslint/config-helpers": "^0.4.2",
+        "@eslint/core": "^0.17.0",
+        "@eslint/eslintrc": "^3.3.5",
+        "@eslint/js": "9.39.4",
+        "@eslint/plugin-kit": "^0.4.1",
+        "@humanfs/node": "^0.16.6",
+        "@humanwhocodes/module-importer": "^1.0.1",
+        "@humanwhocodes/retry": "^0.4.2",
+        "@types/estree": "^1.0.6",
+        "ajv": "^6.14.0",
+        "chalk": "^4.0.0",
+        "cross-spawn": "^7.0.6",
+        "debug": "^4.3.2",
+        "escape-string-regexp": "^4.0.0",
+        "eslint-scope": "^8.4.0",
+        "eslint-visitor-keys": "^4.2.1",
+        "espree": "^10.4.0",
+        "esquery": "^1.5.0",
+        "esutils": "^2.0.2",
+        "fast-deep-equal": "^3.1.3",
+        "file-entry-cache": "^8.0.0",
+        "find-up": "^5.0.0",
+        "glob-parent": "^6.0.2",
+        "ignore": "^5.2.0",
+        "imurmurhash": "^0.1.4",
+        "is-glob": "^4.0.0",
+        "json-stable-stringify-without-jsonify": "^1.0.1",
+        "lodash.merge": "^4.6.2",
+        "minimatch": "^3.1.5",
+        "natural-compare": "^1.4.0",
+        "optionator": "^0.9.3"
+      },
+      "bin": {
+        "eslint": "bin/eslint.js"
+      },
+      "engines": {
+        "node": "^18.18.0 || ^20.9.0 || >=21.1.0"
+      },
+      "funding": {
+        "url": "https://eslint.org/donate"
+      },
+      "peerDependencies": {
+        "jiti": "*"
+      },
+      "peerDependenciesMeta": {
+        "jiti": {
+          "optional": true
+        }
+      }
+    },
+    "node_modules/eslint-config-unjs": {
+      "version": "0.6.2",
+      "resolved": "https://registry.npmjs.org/eslint-config-unjs/-/eslint-config-unjs-0.6.2.tgz",
+      "integrity": "sha512-u80AQy4W4B+gbd/Ow1xlUrvNhTq/tqs1mpfYNzFfazp1wF5L0ZY4gizspfXgr8QDeugMfo+rno6Iwd89jKSJSw==",
+      "dev": true,
+      "license": "MIT",
+      "dependencies": {
+        "@eslint/js": "^9.39.2",
+        "@eslint/markdown": "^7.5.1",
+        "eslint-plugin-unicorn": "^62.0.0",
+        "globals": "^17.0.0",
+        "typescript-eslint": "^8.51.0"
+      },
+      "peerDependencies": {
+        "eslint": "*",
+        "typescript": "*"
+      }
+    },
+    "node_modules/eslint-config-unjs/node_modules/globals": {
+      "version": "17.4.0",
+      "resolved": "https://registry.npmjs.org/globals/-/globals-17.4.0.tgz",
+      "integrity": "sha512-hjrNztw/VajQwOLsMNT1cbJiH2muO3OROCHnbehc8eY5JyD2gqz4AcMHPqgaOR59DjgUjYAYLeH699g/eWi2jw==",
+      "dev": true,
+      "license": "MIT",
+      "engines": {
+        "node": ">=18"
+      },
+      "funding": {
+        "url": "https://github.com/sponsors/sindresorhus"
+      }
+    },
+    "node_modules/eslint-plugin-unicorn": {
+      "version": "62.0.0",
+      "resolved": "https://registry.npmjs.org/eslint-plugin-unicorn/-/eslint-plugin-unicorn-62.0.0.tgz",
+      "integrity": "sha512-HIlIkGLkvf29YEiS/ImuDZQbP12gWyx5i3C6XrRxMvVdqMroCI9qoVYCoIl17ChN+U89pn9sVwLxhIWj5nEc7g==",
+      "dev": true,
+      "license": "MIT",
+      "dependencies": {
+        "@babel/helper-validator-identifier": "^7.28.5",
+        "@eslint-community/eslint-utils": "^4.9.0",
+        "@eslint/plugin-kit": "^0.4.0",
+        "change-case": "^5.4.4",
+        "ci-info": "^4.3.1",
+        "clean-regexp": "^1.0.0",
+        "core-js-compat": "^3.46.0",
+        "esquery": "^1.6.0",
+        "find-up-simple": "^1.0.1",
+        "globals": "^16.4.0",
+        "indent-string": "^5.0.0",
+        "is-builtin-module": "^5.0.0",
+        "jsesc": "^3.1.0",
+        "pluralize": "^8.0.0",
+        "regexp-tree": "^0.1.27",
+        "regjsparser": "^0.13.0",
+        "semver": "^7.7.3",
+        "strip-indent": "^4.1.1"
+      },
+      "engines": {
+        "node": "^20.10.0 || >=21.0.0"
+      },
+      "funding": {
+        "url": "https://github.com/sindresorhus/eslint-plugin-unicorn?sponsor=1"
+      },
+      "peerDependencies": {
+        "eslint": ">=9.38.0"
+      }
+    },
+    "node_modules/eslint-plugin-unicorn/node_modules/globals": {
+      "version": "16.5.0",
+      "resolved": "https://registry.npmjs.org/globals/-/globals-16.5.0.tgz",
+      "integrity": "sha512-c/c15i26VrJ4IRt5Z89DnIzCGDn9EcebibhAOjw5ibqEHsE1wLUgkPn9RDmNcUKyU87GeaL633nyJ+pplFR2ZQ==",
+      "dev": true,
+      "license": "MIT",
+      "engines": {
+        "node": ">=18"
+      },
+      "funding": {
+        "url": "https://github.com/sponsors/sindresorhus"
+      }
+    },
+    "node_modules/eslint-scope": {
+      "version": "8.4.0",
+      "resolved": "https://registry.npmjs.org/eslint-scope/-/eslint-scope-8.4.0.tgz",
+      "integrity": "sha512-sNXOfKCn74rt8RICKMvJS7XKV/Xk9kA7DyJr8mJik3S7Cwgy3qlkkmyS2uQB3jiJg6VNdZd/pDBJu0nvG2NlTg==",
+      "dev": true,
+      "license": "BSD-2-Clause",
+      "dependencies": {
+        "esrecurse": "^4.3.0",
+        "estraverse": "^5.2.0"
+      },
+      "engines": {
+        "node": "^18.18.0 || ^20.9.0 || >=21.1.0"
+      },
+      "funding": {
+        "url": "https://opencollective.com/eslint"
+      }
+    },
+    "node_modules/eslint-visitor-keys": {
+      "version": "4.2.1",
+      "resolved": "https://registry.npmjs.org/eslint-visitor-keys/-/eslint-visitor-keys-4.2.1.tgz",
+      "integrity": "sha512-Uhdk5sfqcee/9H/rCOJikYz67o0a2Tw2hGRPOG2Y1R2dg7brRe1uG0yaNQDHu+TO/uQPF/5eCapvYSmHUjt7JQ==",
+      "dev": true,
+      "license": "Apache-2.0",
+      "engines": {
+        "node": "^18.18.0 || ^20.9.0 || >=21.1.0"
+      },
+      "funding": {
+        "url": "https://opencollective.com/eslint"
+      }
+    },
+    "node_modules/espree": {
+      "version": "10.4.0",
+      "resolved": "https://registry.npmjs.org/espree/-/espree-10.4.0.tgz",
+      "integrity": "sha512-j6PAQ2uUr79PZhBjP5C5fhl8e39FmRnOjsD5lGnWrFU8i2G776tBK7+nP8KuQUTTyAZUwfQqXAgrVH5MbH9CYQ==",
+      "dev": true,
+      "license": "BSD-2-Clause",
+      "dependencies": {
+        "acorn": "^8.15.0",
+        "acorn-jsx": "^5.3.2",
+        "eslint-visitor-keys": "^4.2.1"
+      },
+      "engines": {
+        "node": "^18.18.0 || ^20.9.0 || >=21.1.0"
+      },
+      "funding": {
+        "url": "https://opencollective.com/eslint"
+      }
+    },
+    "node_modules/esquery": {
+      "version": "1.7.0",
+      "resolved": "https://registry.npmjs.org/esquery/-/esquery-1.7.0.tgz",
+      "integrity": "sha512-Ap6G0WQwcU/LHsvLwON1fAQX9Zp0A2Y6Y/cJBl9r/JbW90Zyg4/zbG6zzKa2OTALELarYHmKu0GhpM5EO+7T0g==",
+      "dev": true,
+      "license": "BSD-3-Clause",
+      "dependencies": {
+        "estraverse": "^5.1.0"
+      },
+      "engines": {
+        "node": ">=0.10"
+      }
+    },
+    "node_modules/esrecurse": {
+      "version": "4.3.0",
+      "resolved": "https://registry.npmjs.org/esrecurse/-/esrecurse-4.3.0.tgz",
+      "integrity": "sha512-KmfKL3b6G+RXvP8N1vr3Tq1kL/oCFgn2NYXEtqP8/L3pKapUA4G8cFVaoF3SU323CD4XypR/ffioHmkti6/Tag==",
+      "dev": true,
+      "license": "BSD-2-Clause",
+      "dependencies": {
+        "estraverse": "^5.2.0"
+      },
+      "engines": {
+        "node": ">=4.0"
+      }
+    },
+    "node_modules/estraverse": {
+      "version": "5.3.0",
+      "resolved": "https://registry.npmjs.org/estraverse/-/estraverse-5.3.0.tgz",
+      "integrity": "sha512-MMdARuVEQziNTeJD8DgMqmhwR11BRQ/cBP+pLtYdSTnf3MIO8fFeiINEbX36ZdNlfU/7A9f3gUw49B3oQsvwBA==",
+      "dev": true,
+      "license": "BSD-2-Clause",
+      "engines": {
+        "node": ">=4.0"
+      }
+    },
+    "node_modules/estree-walker": {
+      "version": "3.0.3",
+      "resolved": "https://registry.npmjs.org/estree-walker/-/estree-walker-3.0.3.tgz",
+      "integrity": "sha512-7RUKfXgSMMkzt6ZuXmqapOurLGPPfgj6l9uRZ7lRGolvk0y2yocc35LdcxKC5PQZdn2DMqioAQ2NoWcrTKmm6g==",
+      "dev": true,
+      "license": "MIT",
+      "dependencies": {
+        "@types/estree": "^1.0.0"
+      }
+    },
+    "node_modules/esutils": {
+      "version": "2.0.3",
+      "resolved": "https://registry.npmjs.org/esutils/-/esutils-2.0.3.tgz",
+      "integrity": "sha512-kVscqXk4OCp68SZ0dkgEKVi6/8ij300KBWTJq32P/dYeWTSwK41WyTxalN1eRmA5Z9UU/LX9D7FWSmV9SAYx6g==",
+      "dev": true,
+      "license": "BSD-2-Clause",
+      "engines": {
+        "node": ">=0.10.0"
+      }
+    },
+    "node_modules/expect-type": {
+      "version": "1.3.0",
+      "resolved": "https://registry.npmjs.org/expect-type/-/expect-type-1.3.0.tgz",
+      "integrity": "sha512-knvyeauYhqjOYvQ66MznSMs83wmHrCycNEN6Ao+2AeYEfxUIkuiVxdEa1qlGEPK+We3n0THiDciYSsCcgW/DoA==",
+      "dev": true,
+      "license": "Apache-2.0",
+      "engines": {
+        "node": ">=12.0.0"
+      }
+    },
+    "node_modules/exsolve": {
+      "version": "1.0.8",
+      "resolved": "https://registry.npmjs.org/exsolve/-/exsolve-1.0.8.tgz",
+      "integrity": "sha512-LmDxfWXwcTArk8fUEnOfSZpHOJ6zOMUJKOtFLFqJLoKJetuQG874Uc7/Kki7zFLzYybmZhp1M7+98pfMqeX8yA==",
+      "dev": true,
+      "license": "MIT"
+    },
+    "node_modules/fast-deep-equal": {
+      "version": "3.1.3",
+      "resolved": "https://registry.npmjs.org/fast-deep-equal/-/fast-deep-equal-3.1.3.tgz",
+      "integrity": "sha512-f3qQ9oQy9j2AhBe/H9VC91wLmKBCCU/gDOnKNAYG5hswO7BLKj09Hc5HYNz9cGI++xlpDCIgDaitVs03ATR84Q==",
+      "dev": true,
+      "license": "MIT"
+    },
+    "node_modules/fast-json-stable-stringify": {
+      "version": "2.1.0",
+      "resolved": "https://registry.npmjs.org/fast-json-stable-stringify/-/fast-json-stable-stringify-2.1.0.tgz",
+      "integrity": "sha512-lhd/wF+Lk98HZoTCtlVraHtfh5XYijIjalXck7saUtuanSDyLMxnHhSXEDJqHxD7msR8D0uCmqlkwjCV8xvwHw==",
+      "dev": true,
+      "license": "MIT"
+    },
+    "node_modules/fast-levenshtein": {
+      "version": "2.0.6",
+      "resolved": "https://registry.npmjs.org/fast-levenshtein/-/fast-levenshtein-2.0.6.tgz",
+      "integrity": "sha512-DCXu6Ifhqcks7TZKY3Hxp3y6qphY5SJZmrWMDrKcERSOXWQdMhU9Ig/PYrzyw/ul9jOIyh0N4M0tbC5hodg8dw==",
+      "dev": true,
+      "license": "MIT"
+    },
+    "node_modules/fastest-levenshtein": {
+      "version": "1.0.16",
+      "resolved": "https://registry.npmjs.org/fastest-levenshtein/-/fastest-levenshtein-1.0.16.tgz",
+      "integrity": "sha512-eRnCtTTtGZFpQCwhJiUOuxPQWRXVKYDn0b2PeHfXL6/Zi53SLAzAHfVhVWK2AryC/WH05kGfxhFIPvTF0SXQzg==",
+      "dev": true,
+      "license": "MIT",
+      "engines": {
+        "node": ">= 4.9.1"
+      }
+    },
+    "node_modules/fault": {
+      "version": "2.0.1",
+      "resolved": "https://registry.npmjs.org/fault/-/fault-2.0.1.tgz",
+      "integrity": "sha512-WtySTkS4OKev5JtpHXnib4Gxiurzh5NCGvWrFaZ34m6JehfTUhKZvn9njTfw48t6JumVQOmrKqpmGcdwxnhqBQ==",
+      "dev": true,
+      "license": "MIT",
+      "dependencies": {
+        "format": "^0.2.0"
+      },
+      "funding": {
+        "type": "github",
+        "url": "https://github.com/sponsors/wooorm"
+      }
+    },
+    "node_modules/fdir": {
+      "version": "6.5.0",
+      "resolved": "https://registry.npmjs.org/fdir/-/fdir-6.5.0.tgz",
+      "integrity": "sha512-tIbYtZbucOs0BRGqPJkshJUYdL+SDH7dVM8gjy+ERp3WAUjLEFJE+02kanyHtwjWOnwrKYBiwAmM0p4kLJAnXg==",
+      "dev": true,
+      "license": "MIT",
+      "engines": {
+        "node": ">=12.0.0"
+      },
+      "peerDependencies": {
+        "picomatch": "^3 || ^4"
+      },
+      "peerDependenciesMeta": {
+        "picomatch": {
+          "optional": true
+        }
+      }
+    },
+    "node_modules/file-entry-cache": {
+      "version": "8.0.0",
+      "resolved": "https://registry.npmjs.org/file-entry-cache/-/file-entry-cache-8.0.0.tgz",
+      "integrity": "sha512-XXTUwCvisa5oacNGRP9SfNtYBNAMi+RPwBFmblZEF7N7swHYQS6/Zfk7SRwx4D5j3CH211YNRco1DEMNVfZCnQ==",
+      "dev": true,
+      "license": "MIT",
+      "dependencies": {
+        "flat-cache": "^4.0.0"
+      },
+      "engines": {
+        "node": ">=16.0.0"
+      }
+    },
+    "node_modules/find-up": {
+      "version": "5.0.0",
+      "resolved": "https://registry.npmjs.org/find-up/-/find-up-5.0.0.tgz",
+      "integrity": "sha512-78/PXT1wlLLDgTzDs7sjq9hzz0vXD+zn+7wypEe4fXQxCmdmqfGsEPQxmiCSQI3ajFV91bVSsvNtrJRiW6nGng==",
+      "dev": true,
+      "license": "MIT",
+      "dependencies": {
+        "locate-path": "^6.0.0",
+        "path-exists": "^4.0.0"
+      },
+      "engines": {
+        "node": ">=10"
+      },
+      "funding": {
+        "url": "https://github.com/sponsors/sindresorhus"
+      }
+    },
+    "node_modules/find-up-simple": {
+      "version": "1.0.1",
+      "resolved": "https://registry.npmjs.org/find-up-simple/-/find-up-simple-1.0.1.tgz",
+      "integrity": "sha512-afd4O7zpqHeRyg4PfDQsXmlDe2PfdHtJt6Akt8jOWaApLOZk5JXs6VMR29lz03pRe9mpykrRCYIYxaJYcfpncQ==",
+      "dev": true,
+      "license": "MIT",
+      "engines": {
+        "node": ">=18"
+      },
+      "funding": {
+        "url": "https://github.com/sponsors/sindresorhus"
+      }
+    },
+    "node_modules/flat-cache": {
+      "version": "4.0.1",
+      "resolved": "https://registry.npmjs.org/flat-cache/-/flat-cache-4.0.1.tgz",
+      "integrity": "sha512-f7ccFPK3SXFHpx15UIGyRJ/FJQctuKZ0zVuN3frBo4HnK3cay9VEW0R6yPYFHC0AgqhukPzKjq22t5DmAyqGyw==",
+      "dev": true,
+      "license": "MIT",
+      "dependencies": {
+        "flatted": "^3.2.9",
+        "keyv": "^4.5.4"
+      },
+      "engines": {
+        "node": ">=16"
+      }
+    },
+    "node_modules/flatted": {
+      "version": "3.4.1",
+      "resolved": "https://registry.npmjs.org/flatted/-/flatted-3.4.1.tgz",
+      "integrity": "sha512-IxfVbRFVlV8V/yRaGzk0UVIcsKKHMSfYw66T/u4nTwlWteQePsxe//LjudR1AMX4tZW3WFCh3Zqa/sjlqpbURQ==",
+      "dev": true,
+      "license": "ISC"
+    },
+    "node_modules/format": {
+      "version": "0.2.2",
+      "resolved": "https://registry.npmjs.org/format/-/format-0.2.2.tgz",
+      "integrity": "sha512-wzsgA6WOq+09wrU1tsJ09udeR/YZRaeArL9e1wPbFg3GG2yDnC2ldKpxs4xunpFF9DgqCqOIra3bc1HWrJ37Ww==",
+      "dev": true,
+      "engines": {
+        "node": ">=0.4.x"
+      }
+    },
+    "node_modules/fsevents": {
+      "version": "2.3.3",
+      "resolved": "https://registry.npmjs.org/fsevents/-/fsevents-2.3.3.tgz",
+      "integrity": "sha512-5xoDfX+fL7faATnagmWPpbFtwh/R77WmMMqqHGS65C3vvB0YHrgF+B1YmZ3441tMj5n63k0212XNoJwzlhffQw==",
+      "dev": true,
+      "hasInstallScript": true,
+      "license": "MIT",
+      "optional": true,
+      "os": [
+        "darwin"
+      ],
+      "engines": {
+        "node": "^8.16.0 || ^10.6.0 || >=11.0.0"
+      }
+    },
+    "node_modules/get-tsconfig": {
+      "version": "4.13.6",
+      "resolved": "https://registry.npmjs.org/get-tsconfig/-/get-tsconfig-4.13.6.tgz",
+      "integrity": "sha512-shZT/QMiSHc/YBLxxOkMtgSid5HFoauqCE3/exfsEcwg1WkeqjG+V40yBbBrsD+jW2HDXcs28xOfcbm2jI8Ddw==",
+      "dev": true,
+      "license": "MIT",
+      "dependencies": {
+        "resolve-pkg-maps": "^1.0.0"
+      },
+      "funding": {
+        "url": "https://github.com/privatenumber/get-tsconfig?sponsor=1"
+      }
+    },
+    "node_modules/giget": {
+      "version": "2.0.0",
+      "resolved": "https://registry.npmjs.org/giget/-/giget-2.0.0.tgz",
+      "integrity": "sha512-L5bGsVkxJbJgdnwyuheIunkGatUF/zssUoxxjACCseZYAVbaqdh9Tsmmlkl8vYan09H7sbvKt4pS8GqKLBrEzA==",
+      "dev": true,
+      "license": "MIT",
+      "dependencies": {
+        "citty": "^0.1.6",
+        "consola": "^3.4.0",
+        "defu": "^6.1.4",
+        "node-fetch-native": "^1.6.6",
+        "nypm": "^0.6.0",
+        "pathe": "^2.0.3"
+      },
+      "bin": {
+        "giget": "dist/cli.mjs"
+      }
+    },
+    "node_modules/giget/node_modules/citty": {
+      "version": "0.1.6",
+      "resolved": "https://registry.npmjs.org/citty/-/citty-0.1.6.tgz",
+      "integrity": "sha512-tskPPKEs8D2KPafUypv2gxwJP8h/OaJmC82QQGGDQcHvXX43xF2VDACcJVmZ0EuSxkpO9Kc4MlrA3q0+FG58AQ==",
+      "dev": true,
+      "license": "MIT",
+      "dependencies": {
+        "consola": "^3.2.3"
+      }
+    },
+    "node_modules/github-slugger": {
+      "version": "2.0.0",
+      "resolved": "https://registry.npmjs.org/github-slugger/-/github-slugger-2.0.0.tgz",
+      "integrity": "sha512-IaOQ9puYtjrkq7Y0Ygl9KDZnrf/aiUJYUpVf89y8kyaxbRG7Y1SrX/jaumrv81vc61+kiMempujsM3Yw7w5qcw==",
+      "dev": true,
+      "license": "ISC"
+    },
+    "node_modules/glob-parent": {
+      "version": "6.0.2",
+      "resolved": "https://registry.npmjs.org/glob-parent/-/glob-parent-6.0.2.tgz",
+      "integrity": "sha512-XxwI8EOhVQgWp6iDL+3b0r86f4d6AX6zSU55HfB4ydCEuXLXc5FcYeOu+nnGftS4TEju/11rt4KJPTMgbfmv4A==",
+      "dev": true,
+      "license": "ISC",
+      "dependencies": {
+        "is-glob": "^4.0.3"
+      },
+      "engines": {
+        "node": ">=10.13.0"
+      }
+    },
+    "node_modules/globals": {
+      "version": "14.0.0",
+      "resolved": "https://registry.npmjs.org/globals/-/globals-14.0.0.tgz",
+      "integrity": "sha512-oahGvuMGQlPw/ivIYBjVSrWAfWLBeku5tpPE2fOPLi+WHffIWbuh2tCjhyQhTBPMf5E9jDEH4FOmTYgYwbKwtQ==",
+      "dev": true,
+      "license": "MIT",
+      "engines": {
+        "node": ">=18"
+      },
+      "funding": {
+        "url": "https://github.com/sponsors/sindresorhus"
+      }
+    },
+    "node_modules/has-flag": {
+      "version": "4.0.0",
+      "resolved": "https://registry.npmjs.org/has-flag/-/has-flag-4.0.0.tgz",
+      "integrity": "sha512-EykJT/Q1KjTWctppgIAgfSO0tKVuZUjhgMr17kqTumMl6Afv3EISleU7qZUzoXDFTAHTDC4NOoG/ZxU3EvlMPQ==",
+      "dev": true,
+      "license": "MIT",
+      "engines": {
+        "node": ">=8"
+      }
+    },
+    "node_modules/html-escaper": {
+      "version": "2.0.2",
+      "resolved": "https://registry.npmjs.org/html-escaper/-/html-escaper-2.0.2.tgz",
+      "integrity": "sha512-H2iMtd0I4Mt5eYiapRdIDjp+XzelXQ0tFE4JS7YFwFevXXMmOp9myNrUvCg0D6ws8iqkRPBfKHgbwig1SmlLfg==",
+      "dev": true,
+      "license": "MIT"
+    },
+    "node_modules/ignore": {
+      "version": "5.3.2",
+      "resolved": "https://registry.npmjs.org/ignore/-/ignore-5.3.2.tgz",
+      "integrity": "sha512-hsBTNUqQTDwkWtcdYI2i06Y/nUBEsNEDJKjWdigLvegy8kDuJAS8uRlpkkcQpyEXL0Z/pjDy5HBmMjRCJ2gq+g==",
+      "dev": true,
+      "license": "MIT",
+      "engines": {
+        "node": ">= 4"
+      }
+    },
+    "node_modules/import-fresh": {
+      "version": "3.3.1",
+      "resolved": "https://registry.npmjs.org/import-fresh/-/import-fresh-3.3.1.tgz",
+      "integrity": "sha512-TR3KfrTZTYLPB6jUjfx6MF9WcWrHL9su5TObK4ZkYgBdWKPOFoSoQIdEuTuR82pmtxH2spWG9h6etwfr1pLBqQ==",
+      "dev": true,
+      "license": "MIT",
+      "dependencies": {
+        "parent-module": "^1.0.0",
+        "resolve-from": "^4.0.0"
+      },
+      "engines": {
+        "node": ">=6"
+      },
+      "funding": {
+        "url": "https://github.com/sponsors/sindresorhus"
+      }
+    },
+    "node_modules/imurmurhash": {
+      "version": "0.1.4",
+      "resolved": "https://registry.npmjs.org/imurmurhash/-/imurmurhash-0.1.4.tgz",
+      "integrity": "sha512-JmXMZ6wuvDmLiHEml9ykzqO6lwFbof0GG4IkcGaENdCRDDmMVnny7s5HsIgHCbaq0w2MyPhDqkhTUgS2LU2PHA==",
+      "dev": true,
+      "license": "MIT",
+      "engines": {
+        "node": ">=0.8.19"
+      }
+    },
+    "node_modules/indent-string": {
+      "version": "5.0.0",
+      "resolved": "https://registry.npmjs.org/indent-string/-/indent-string-5.0.0.tgz",
+      "integrity": "sha512-m6FAo/spmsW2Ab2fU35JTYwtOKa2yAwXSwgjSv1TJzh4Mh7mC3lzAOVLBprb72XsTrgkEIsl7YrFNAiDiRhIGg==",
+      "dev": true,
+      "license": "MIT",
+      "engines": {
+        "node": ">=12"
+      },
+      "funding": {
+        "url": "https://github.com/sponsors/sindresorhus"
+      }
+    },
+    "node_modules/is-builtin-module": {
+      "version": "5.0.0",
+      "resolved": "https://registry.npmjs.org/is-builtin-module/-/is-builtin-module-5.0.0.tgz",
+      "integrity": "sha512-f4RqJKBUe5rQkJ2eJEJBXSticB3hGbN9j0yxxMQFqIW89Jp9WYFtzfTcRlstDKVUTRzSOTLKRfO9vIztenwtxA==",
+      "dev": true,
+      "license": "MIT",
+      "dependencies": {
+        "builtin-modules": "^5.0.0"
+      },
+      "engines": {
+        "node": ">=18.20"
+      },
+      "funding": {
+        "url": "https://github.com/sponsors/sindresorhus"
+      }
+    },
+    "node_modules/is-docker": {
+      "version": "3.0.0",
+      "resolved": "https://registry.npmjs.org/is-docker/-/is-docker-3.0.0.tgz",
+      "integrity": "sha512-eljcgEDlEns/7AXFosB5K/2nCM4P7FQPkGc/DWLy5rmFEWvZayGrik1d9/QIY5nJ4f9YsVvBkA6kJpHn9rISdQ==",
+      "dev": true,
+      "license": "MIT",
+      "bin": {
+        "is-docker": "cli.js"
+      },
+      "engines": {
+        "node": "^12.20.0 || ^14.13.1 || >=16.0.0"
+      },
+      "funding": {
+        "url": "https://github.com/sponsors/sindresorhus"
+      }
+    },
+    "node_modules/is-extglob": {
+      "version": "2.1.1",
+      "resolved": "https://registry.npmjs.org/is-extglob/-/is-extglob-2.1.1.tgz",
+      "integrity": "sha512-SbKbANkN603Vi4jEZv49LeVJMn4yGwsbzZworEoyEiutsN3nJYdbO36zfhGJ6QEDpOZIFkDtnq5JRxmvl3jsoQ==",
+      "dev": true,
+      "license": "MIT",
+      "engines": {
+        "node": ">=0.10.0"
+      }
+    },
+    "node_modules/is-glob": {
+      "version": "4.0.3",
+      "resolved": "https://registry.npmjs.org/is-glob/-/is-glob-4.0.3.tgz",
+      "integrity": "sha512-xelSayHH36ZgE7ZWhli7pW34hNbNl8Ojv5KVmkJD4hBdD3th8Tfk9vYasLM+mXWOZhFkgZfxhLSnrwRr4elSSg==",
+      "dev": true,
+      "license": "MIT",
+      "dependencies": {
+        "is-extglob": "^2.1.1"
+      },
+      "engines": {
+        "node": ">=0.10.0"
+      }
+    },
+    "node_modules/is-inside-container": {
+      "version": "1.0.0",
+      "resolved": "https://registry.npmjs.org/is-inside-container/-/is-inside-container-1.0.0.tgz",
+      "integrity": "sha512-KIYLCCJghfHZxqjYBE7rEy0OBuTd5xCHS7tHVgvCLkx7StIoaxwNW3hCALgEUjFfeRk+MG/Qxmp/vtETEF3tRA==",
+      "dev": true,
+      "license": "MIT",
+      "dependencies": {
+        "is-docker": "^3.0.0"
+      },
+      "bin": {
+        "is-inside-container": "cli.js"
+      },
+      "engines": {
+        "node": ">=14.16"
+      },
+      "funding": {
+        "url": "https://github.com/sponsors/sindresorhus"
+      }
+    },
+    "node_modules/is-wsl": {
+      "version": "3.1.1",
+      "resolved": "https://registry.npmjs.org/is-wsl/-/is-wsl-3.1.1.tgz",
+      "integrity": "sha512-e6rvdUCiQCAuumZslxRJWR/Doq4VpPR82kqclvcS0efgt430SlGIk05vdCN58+VrzgtIcfNODjozVielycD4Sw==",
+      "dev": true,
+      "license": "MIT",
+      "dependencies": {
+        "is-inside-container": "^1.0.0"
+      },
+      "engines": {
+        "node": ">=16"
+      },
+      "funding": {
+        "url": "https://github.com/sponsors/sindresorhus"
+      }
+    },
+    "node_modules/isexe": {
+      "version": "2.0.0",
+      "resolved": "https://registry.npmjs.org/isexe/-/isexe-2.0.0.tgz",
+      "integrity": "sha512-RHxMLp9lnKHGHRng9QFhRCMbYAcVpn69smSGcq3f36xjgVVWThj4qqLbTLlq7Ssj8B+fIQ1EuCEGI2lKsyQeIw==",
+      "dev": true,
+      "license": "ISC"
+    },
+    "node_modules/istanbul-lib-coverage": {
+      "version": "3.2.2",
+      "resolved": "https://registry.npmjs.org/istanbul-lib-coverage/-/istanbul-lib-coverage-3.2.2.tgz",
+      "integrity": "sha512-O8dpsF+r0WV/8MNRKfnmrtCWhuKjxrq2w+jpzBL5UZKTi2LeVWnWOmWRxFlesJONmc+wLAGvKQZEOanko0LFTg==",
+      "dev": true,
+      "license": "BSD-3-Clause",
+      "engines": {
+        "node": ">=8"
+      }
+    },
+    "node_modules/istanbul-lib-report": {
+      "version": "3.0.1",
+      "resolved": "https://registry.npmjs.org/istanbul-lib-report/-/istanbul-lib-report-3.0.1.tgz",
+      "integrity": "sha512-GCfE1mtsHGOELCU8e/Z7YWzpmybrx/+dSTfLrvY8qRmaY6zXTKWn6WQIjaAFw069icm6GVMNkgu0NzI4iPZUNw==",
+      "dev": true,
+      "license": "BSD-3-Clause",
+      "dependencies": {
+        "istanbul-lib-coverage": "^3.0.0",
+        "make-dir": "^4.0.0",
+        "supports-color": "^7.1.0"
+      },
+      "engines": {
+        "node": ">=10"
+      }
+    },
+    "node_modules/istanbul-reports": {
+      "version": "3.2.0",
+      "resolved": "https://registry.npmjs.org/istanbul-reports/-/istanbul-reports-3.2.0.tgz",
+      "integrity": "sha512-HGYWWS/ehqTV3xN10i23tkPkpH46MLCIMFNCaaKNavAXTF1RkqxawEPtnjnGZ6XKSInBKkiOA5BKS+aZiY3AvA==",
+      "dev": true,
+      "license": "BSD-3-Clause",
+      "dependencies": {
+        "html-escaper": "^2.0.0",
+        "istanbul-lib-report": "^3.0.0"
+      },
+      "engines": {
+        "node": ">=8"
+      }
+    },
+    "node_modules/jiti": {
+      "version": "2.6.1",
+      "resolved": "https://registry.npmjs.org/jiti/-/jiti-2.6.1.tgz",
+      "integrity": "sha512-ekilCSN1jwRvIbgeg/57YFh8qQDNbwDb9xT/qu2DAHbFFZUicIl4ygVaAvzveMhMVr3LnpSKTNnwt8PoOfmKhQ==",
+      "dev": true,
+      "license": "MIT",
+      "bin": {
+        "jiti": "lib/jiti-cli.mjs"
+      }
+    },
+    "node_modules/js-tokens": {
+      "version": "10.0.0",
+      "resolved": "https://registry.npmjs.org/js-tokens/-/js-tokens-10.0.0.tgz",
+      "integrity": "sha512-lM/UBzQmfJRo9ABXbPWemivdCW8V2G8FHaHdypQaIy523snUjog0W71ayWXTjiR+ixeMyVHN2XcpnTd/liPg/Q==",
+      "dev": true,
+      "license": "MIT"
+    },
+    "node_modules/js-yaml": {
+      "version": "4.1.1",
+      "resolved": "https://registry.npmjs.org/js-yaml/-/js-yaml-4.1.1.tgz",
+      "integrity": "sha512-qQKT4zQxXl8lLwBtHMWwaTcGfFOZviOJet3Oy/xmGk2gZH677CJM9EvtfdSkgWcATZhj/55JZ0rmy3myCT5lsA==",
+      "dev": true,
+      "license": "MIT",
+      "dependencies": {
+        "argparse": "^2.0.1"
+      },
+      "bin": {
+        "js-yaml": "bin/js-yaml.js"
+      }
+    },
+    "node_modules/jsesc": {
+      "version": "3.1.0",
+      "resolved": "https://registry.npmjs.org/jsesc/-/jsesc-3.1.0.tgz",
+      "integrity": "sha512-/sM3dO2FOzXjKQhJuo0Q173wf2KOo8t4I8vHy6lF9poUp7bKT0/NHE8fPX23PwfhnykfqnC2xRxOnVw5XuGIaA==",
+      "dev": true,
+      "license": "MIT",
+      "bin": {
+        "jsesc": "bin/jsesc"
+      },
+      "engines": {
+        "node": ">=6"
+      }
+    },
+    "node_modules/json-buffer": {
+      "version": "3.0.1",
+      "resolved": "https://registry.npmjs.org/json-buffer/-/json-buffer-3.0.1.tgz",
+      "integrity": "sha512-4bV5BfR2mqfQTJm+V5tPPdf+ZpuhiIvTuAB5g8kcrXOZpTT/QwwVRWBywX1ozr6lEuPdbHxwaJlm9G6mI2sfSQ==",
+      "dev": true,
+      "license": "MIT"
+    },
+    "node_modules/json-schema-traverse": {
+      "version": "0.4.1",
+      "resolved": "https://registry.npmjs.org/json-schema-traverse/-/json-schema-traverse-0.4.1.tgz",
+      "integrity": "sha512-xbbCH5dCYU5T8LcEhhuh7HJ88HXuW3qsI3Y0zOZFKfZEHcpWiHU/Jxzk629Brsab/mMiHQti9wMP+845RPe3Vg==",
+      "dev": true,
+      "license": "MIT"
+    },
+    "node_modules/json-stable-stringify-without-jsonify": {
+      "version": "1.0.1",
+      "resolved": "https://registry.npmjs.org/json-stable-stringify-without-jsonify/-/json-stable-stringify-without-jsonify-1.0.1.tgz",
+      "integrity": "sha512-Bdboy+l7tA3OGW6FjyFHWkP5LuByj1Tk33Ljyq0axyzdk9//JSi2u3fP1QSmd1KNwq6VOKYGlAu87CisVir6Pw==",
+      "dev": true,
+      "license": "MIT"
+    },
+    "node_modules/keyv": {
+      "version": "4.5.4",
+      "resolved": "https://registry.npmjs.org/keyv/-/keyv-4.5.4.tgz",
+      "integrity": "sha512-oxVHkHR/EJf2CNXnWxRLW6mg7JyCCUcG0DtEGmL2ctUo1PNTin1PUil+r/+4r5MpVgC/fn1kjsx7mjSujKqIpw==",
+      "dev": true,
+      "license": "MIT",
+      "dependencies": {
+        "json-buffer": "3.0.1"
+      }
+    },
+    "node_modules/knitwork": {
+      "version": "1.3.0",
+      "resolved": "https://registry.npmjs.org/knitwork/-/knitwork-1.3.0.tgz",
+      "integrity": "sha512-4LqMNoONzR43B1W0ek0fhXMsDNW/zxa1NdFAVMY+k28pgZLovR4G3PB5MrpTxCy1QaZCqNoiaKPr5w5qZHfSNw==",
+      "dev": true,
+      "license": "MIT"
+    },
+    "node_modules/levn": {
+      "version": "0.4.1",
+      "resolved": "https://registry.npmjs.org/levn/-/levn-0.4.1.tgz",
+      "integrity": "sha512-+bT2uH4E5LGE7h/n3evcS/sQlJXCpIp6ym8OWJ5eV6+67Dsql/LaaT7qJBAt2rzfoa/5QBGBhxDix1dMt2kQKQ==",
+      "dev": true,
+      "license": "MIT",
+      "dependencies": {
+        "prelude-ls": "^1.2.1",
+        "type-check": "~0.4.0"
+      },
+      "engines": {
+        "node": ">= 0.8.0"
+      }
+    },
+    "node_modules/locate-path": {
+      "version": "6.0.0",
+      "resolved": "https://registry.npmjs.org/locate-path/-/locate-path-6.0.0.tgz",
+      "integrity": "sha512-iPZK6eYjbxRu3uB4/WZ3EsEIMJFMqAoopl3R+zuq0UjcAm/MO6KCweDgPfP3elTztoKP3KtnVHxTn2NHBSDVUw==",
+      "dev": true,
+      "license": "MIT",
+      "dependencies": {
+        "p-locate": "^5.0.0"
+      },
+      "engines": {
+        "node": ">=10"
+      },
+      "funding": {
+        "url": "https://github.com/sponsors/sindresorhus"
+      }
+    },
+    "node_modules/lodash": {
+      "version": "4.17.23",
+      "resolved": "https://registry.npmjs.org/lodash/-/lodash-4.17.23.tgz",
+      "integrity": "sha512-LgVTMpQtIopCi79SJeDiP0TfWi5CNEc/L/aRdTh3yIvmZXTnheWpKjSZhnvMl8iXbC1tFg9gdHHDMLoV7CnG+w==",
+      "dev": true,
+      "license": "MIT"
+    },
+    "node_modules/lodash.deburr": {
+      "version": "4.1.0",
+      "resolved": "https://registry.npmjs.org/lodash.deburr/-/lodash.deburr-4.1.0.tgz",
+      "integrity": "sha512-m/M1U1f3ddMCs6Hq2tAsYThTBDaAKFDX3dwDo97GEYzamXi9SqUpjWi/Rrj/gf3X2n8ktwgZrlP1z6E3v/IExQ==",
+      "dev": true,
+      "license": "MIT"
+    },
+    "node_modules/lodash.merge": {
+      "version": "4.6.2",
+      "resolved": "https://registry.npmjs.org/lodash.merge/-/lodash.merge-4.6.2.tgz",
+      "integrity": "sha512-0KpjqXRVvrYyCsX1swR/XTK0va6VQkQM6MNo7PqW77ByjAhoARA8EfrP1N4+KlKj8YS0ZUCtRT/YUuhyYDujIQ==",
+      "dev": true,
+      "license": "MIT"
+    },
+    "node_modules/longest-streak": {
+      "version": "3.1.0",
+      "resolved": "https://registry.npmjs.org/longest-streak/-/longest-streak-3.1.0.tgz",
+      "integrity": "sha512-9Ri+o0JYgehTaVBBDoMqIl8GXtbWg711O3srftcHhZ0dqnETqLaoIK0x17fUw9rFSlK/0NlsKe0Ahhyl5pXE2g==",
+      "dev": true,
+      "license": "MIT",
+      "funding": {
+        "type": "github",
+        "url": "https://github.com/sponsors/wooorm"
+      }
+    },
+    "node_modules/magic-string": {
+      "version": "0.30.21",
+      "resolved": "https://registry.npmjs.org/magic-string/-/magic-string-0.30.21.tgz",
+      "integrity": "sha512-vd2F4YUyEXKGcLHoq+TEyCjxueSeHnFxyyjNp80yg0XV4vUhnDer/lvvlqM/arB5bXQN5K2/3oinyCRyx8T2CQ==",
+      "dev": true,
+      "license": "MIT",
+      "dependencies": {
+        "@jridgewell/sourcemap-codec": "^1.5.5"
+      }
+    },
+    "node_modules/magicast": {
+      "version": "0.5.2",
+      "resolved": "https://registry.npmjs.org/magicast/-/magicast-0.5.2.tgz",
+      "integrity": "sha512-E3ZJh4J3S9KfwdjZhe2afj6R9lGIN5Pher1pF39UGrXRqq/VDaGVIGN13BjHd2u8B61hArAGOnso7nBOouW3TQ==",
+      "dev": true,
+      "license": "MIT",
+      "dependencies": {
+        "@babel/parser": "^7.29.0",
+        "@babel/types": "^7.29.0",
+        "source-map-js": "^1.2.1"
+      }
+    },
+    "node_modules/make-dir": {
+      "version": "4.0.0",
+      "resolved": "https://registry.npmjs.org/make-dir/-/make-dir-4.0.0.tgz",
+      "integrity": "sha512-hXdUTZYIVOt1Ex//jAQi+wTZZpUpwBj/0QsOzqegb3rGMMeJiSEu5xLHnYfBrRV4RH2+OCSOO95Is/7x1WJ4bw==",
+      "dev": true,
+      "license": "MIT",
+      "dependencies": {
+        "semver": "^7.5.3"
+      },
+      "engines": {
+        "node": ">=10"
+      },
+      "funding": {
+        "url": "https://github.com/sponsors/sindresorhus"
+      }
+    },
+    "node_modules/markdown-table": {
+      "version": "3.0.4",
+      "resolved": "https://registry.npmjs.org/markdown-table/-/markdown-table-3.0.4.tgz",
+      "integrity": "sha512-wiYz4+JrLyb/DqW2hkFJxP7Vd7JuTDm77fvbM8VfEQdmSMqcImWeeRbHwZjBjIFki/VaMK2BhFi7oUUZeM5bqw==",
+      "dev": true,
+      "license": "MIT",
+      "funding": {
+        "type": "github",
+        "url": "https://github.com/sponsors/wooorm"
+      }
+    },
+    "node_modules/md4w": {
+      "version": "0.2.7",
+      "resolved": "https://registry.npmjs.org/md4w/-/md4w-0.2.7.tgz",
+      "integrity": "sha512-lFM7vwk3d4MzkV2mija7aPkK6OjKXZDQsH2beX+e2cvccBoqc6RraheMtAO0Wcr/gjj5L+WS5zhb+06AmuGZrg==",
+      "dev": true,
+      "license": "MIT"
+    },
+    "node_modules/mdast-util-find-and-replace": {
+      "version": "3.0.2",
+      "resolved": "https://registry.npmjs.org/mdast-util-find-and-replace/-/mdast-util-find-and-replace-3.0.2.tgz",
+      "integrity": "sha512-Tmd1Vg/m3Xz43afeNxDIhWRtFZgM2VLyaf4vSTYwudTyeuTneoL3qtWMA5jeLyz/O1vDJmmV4QuScFCA2tBPwg==",
+      "dev": true,
+      "license": "MIT",
+      "dependencies": {
+        "@types/mdast": "^4.0.0",
+        "escape-string-regexp": "^5.0.0",
+        "unist-util-is": "^6.0.0",
+        "unist-util-visit-parents": "^6.0.0"
+      },
+      "funding": {
+        "type": "opencollective",
+        "url": "https://opencollective.com/unified"
+      }
+    },
+    "node_modules/mdast-util-find-and-replace/node_modules/escape-string-regexp": {
+      "version": "5.0.0",
+      "resolved": "https://registry.npmjs.org/escape-string-regexp/-/escape-string-regexp-5.0.0.tgz",
+      "integrity": "sha512-/veY75JbMK4j1yjvuUxuVsiS/hr/4iHs9FTT6cgTexxdE0Ly/glccBAkloH/DofkjRbZU3bnoj38mOmhkZ0lHw==",
+      "dev": true,
+      "license": "MIT",
+      "engines": {
+        "node": ">=12"
+      },
+      "funding": {
+        "url": "https://github.com/sponsors/sindresorhus"
+      }
+    },
+    "node_modules/mdast-util-from-markdown": {
+      "version": "2.0.3",
+      "resolved": "https://registry.npmjs.org/mdast-util-from-markdown/-/mdast-util-from-markdown-2.0.3.tgz",
+      "integrity": "sha512-W4mAWTvSlKvf8L6J+VN9yLSqQ9AOAAvHuoDAmPkz4dHf553m5gVj2ejadHJhoJmcmxEnOv6Pa8XJhpxE93kb8Q==",
+      "dev": true,
+      "license": "MIT",
+      "dependencies": {
+        "@types/mdast": "^4.0.0",
+        "@types/unist": "^3.0.0",
+        "decode-named-character-reference": "^1.0.0",
+        "devlop": "^1.0.0",
+        "mdast-util-to-string": "^4.0.0",
+        "micromark": "^4.0.0",
+        "micromark-util-decode-numeric-character-reference": "^2.0.0",
+        "micromark-util-decode-string": "^2.0.0",
+        "micromark-util-normalize-identifier": "^2.0.0",
+        "micromark-util-symbol": "^2.0.0",
+        "micromark-util-types": "^2.0.0",
+        "unist-util-stringify-position": "^4.0.0"
+      },
+      "funding": {
+        "type": "opencollective",
+        "url": "https://opencollective.com/unified"
+      }
+    },
+    "node_modules/mdast-util-frontmatter": {
+      "version": "2.0.1",
+      "resolved": "https://registry.npmjs.org/mdast-util-frontmatter/-/mdast-util-frontmatter-2.0.1.tgz",
+      "integrity": "sha512-LRqI9+wdgC25P0URIJY9vwocIzCcksduHQ9OF2joxQoyTNVduwLAFUzjoopuRJbJAReaKrNQKAZKL3uCMugWJA==",
+      "dev": true,
+      "license": "MIT",
+      "dependencies": {
+        "@types/mdast": "^4.0.0",
+        "devlop": "^1.0.0",
+        "escape-string-regexp": "^5.0.0",
+        "mdast-util-from-markdown": "^2.0.0",
+        "mdast-util-to-markdown": "^2.0.0",
+        "micromark-extension-frontmatter": "^2.0.0"
+      },
+      "funding": {
+        "type": "opencollective",
+        "url": "https://opencollective.com/unified"
+      }
+    },
+    "node_modules/mdast-util-frontmatter/node_modules/escape-string-regexp": {
+      "version": "5.0.0",
+      "resolved": "https://registry.npmjs.org/escape-string-regexp/-/escape-string-regexp-5.0.0.tgz",
+      "integrity": "sha512-/veY75JbMK4j1yjvuUxuVsiS/hr/4iHs9FTT6cgTexxdE0Ly/glccBAkloH/DofkjRbZU3bnoj38mOmhkZ0lHw==",
+      "dev": true,
+      "license": "MIT",
+      "engines": {
+        "node": ">=12"
+      },
+      "funding": {
+        "url": "https://github.com/sponsors/sindresorhus"
+      }
+    },
+    "node_modules/mdast-util-gfm": {
+      "version": "3.1.0",
+      "resolved": "https://registry.npmjs.org/mdast-util-gfm/-/mdast-util-gfm-3.1.0.tgz",
+      "integrity": "sha512-0ulfdQOM3ysHhCJ1p06l0b0VKlhU0wuQs3thxZQagjcjPrlFRqY215uZGHHJan9GEAXd9MbfPjFJz+qMkVR6zQ==",
+      "dev": true,
+      "license": "MIT",
+      "dependencies": {
+        "mdast-util-from-markdown": "^2.0.0",
+        "mdast-util-gfm-autolink-literal": "^2.0.0",
+        "mdast-util-gfm-footnote": "^2.0.0",
+        "mdast-util-gfm-strikethrough": "^2.0.0",
+        "mdast-util-gfm-table": "^2.0.0",
+        "mdast-util-gfm-task-list-item": "^2.0.0",
+        "mdast-util-to-markdown": "^2.0.0"
+      },
+      "funding": {
+        "type": "opencollective",
+        "url": "https://opencollective.com/unified"
+      }
+    },
+    "node_modules/mdast-util-gfm-autolink-literal": {
+      "version": "2.0.1",
+      "resolved": "https://registry.npmjs.org/mdast-util-gfm-autolink-literal/-/mdast-util-gfm-autolink-literal-2.0.1.tgz",
+      "integrity": "sha512-5HVP2MKaP6L+G6YaxPNjuL0BPrq9orG3TsrZ9YXbA3vDw/ACI4MEsnoDpn6ZNm7GnZgtAcONJyPhOP8tNJQavQ==",
+      "dev": true,
+      "license": "MIT",
+      "dependencies": {
+        "@types/mdast": "^4.0.0",
+        "ccount": "^2.0.0",
+        "devlop": "^1.0.0",
+        "mdast-util-find-and-replace": "^3.0.0",
+        "micromark-util-character": "^2.0.0"
+      },
+      "funding": {
+        "type": "opencollective",
+        "url": "https://opencollective.com/unified"
+      }
+    },
+    "node_modules/mdast-util-gfm-footnote": {
+      "version": "2.1.0",
+      "resolved": "https://registry.npmjs.org/mdast-util-gfm-footnote/-/mdast-util-gfm-footnote-2.1.0.tgz",
+      "integrity": "sha512-sqpDWlsHn7Ac9GNZQMeUzPQSMzR6Wv0WKRNvQRg0KqHh02fpTz69Qc1QSseNX29bhz1ROIyNyxExfawVKTm1GQ==",
+      "dev": true,
+      "license": "MIT",
+      "dependencies": {
+        "@types/mdast": "^4.0.0",
+        "devlop": "^1.1.0",
+        "mdast-util-from-markdown": "^2.0.0",
+        "mdast-util-to-markdown": "^2.0.0",
+        "micromark-util-normalize-identifier": "^2.0.0"
+      },
+      "funding": {
+        "type": "opencollective",
+        "url": "https://opencollective.com/unified"
+      }
+    },
+    "node_modules/mdast-util-gfm-strikethrough": {
+      "version": "2.0.0",
+      "resolved": "https://registry.npmjs.org/mdast-util-gfm-strikethrough/-/mdast-util-gfm-strikethrough-2.0.0.tgz",
+      "integrity": "sha512-mKKb915TF+OC5ptj5bJ7WFRPdYtuHv0yTRxK2tJvi+BDqbkiG7h7u/9SI89nRAYcmap2xHQL9D+QG/6wSrTtXg==",
+      "dev": true,
+      "license": "MIT",
+      "dependencies": {
+        "@types/mdast": "^4.0.0",
+        "mdast-util-from-markdown": "^2.0.0",
+        "mdast-util-to-markdown": "^2.0.0"
+      },
+      "funding": {
+        "type": "opencollective",
+        "url": "https://opencollective.com/unified"
+      }
+    },
+    "node_modules/mdast-util-gfm-table": {
+      "version": "2.0.0",
+      "resolved": "https://registry.npmjs.org/mdast-util-gfm-table/-/mdast-util-gfm-table-2.0.0.tgz",
+      "integrity": "sha512-78UEvebzz/rJIxLvE7ZtDd/vIQ0RHv+3Mh5DR96p7cS7HsBhYIICDBCu8csTNWNO6tBWfqXPWekRuj2FNOGOZg==",
+      "dev": true,
+      "license": "MIT",
+      "dependencies": {
+        "@types/mdast": "^4.0.0",
+        "devlop": "^1.0.0",
+        "markdown-table": "^3.0.0",
+        "mdast-util-from-markdown": "^2.0.0",
+        "mdast-util-to-markdown": "^2.0.0"
+      },
+      "funding": {
+        "type": "opencollective",
+        "url": "https://opencollective.com/unified"
+      }
+    },
+    "node_modules/mdast-util-gfm-task-list-item": {
+      "version": "2.0.0",
+      "resolved": "https://registry.npmjs.org/mdast-util-gfm-task-list-item/-/mdast-util-gfm-task-list-item-2.0.0.tgz",
+      "integrity": "sha512-IrtvNvjxC1o06taBAVJznEnkiHxLFTzgonUdy8hzFVeDun0uTjxxrRGVaNFqkU1wJR3RBPEfsxmU6jDWPofrTQ==",
+      "dev": true,
+      "license": "MIT",
+      "dependencies": {
+        "@types/mdast": "^4.0.0",
+        "devlop": "^1.0.0",
+        "mdast-util-from-markdown": "^2.0.0",
+        "mdast-util-to-markdown": "^2.0.0"
+      },
+      "funding": {
+        "type": "opencollective",
+        "url": "https://opencollective.com/unified"
+      }
+    },
+    "node_modules/mdast-util-phrasing": {
+      "version": "4.1.0",
+      "resolved": "https://registry.npmjs.org/mdast-util-phrasing/-/mdast-util-phrasing-4.1.0.tgz",
+      "integrity": "sha512-TqICwyvJJpBwvGAMZjj4J2n0X8QWp21b9l0o7eXyVJ25YNWYbJDVIyD1bZXE6WtV6RmKJVYmQAKWa0zWOABz2w==",
+      "dev": true,
+      "license": "MIT",
+      "dependencies": {
+        "@types/mdast": "^4.0.0",
+        "unist-util-is": "^6.0.0"
+      },
+      "funding": {
+        "type": "opencollective",
+        "url": "https://opencollective.com/unified"
+      }
+    },
+    "node_modules/mdast-util-to-markdown": {
+      "version": "2.1.2",
+      "resolved": "https://registry.npmjs.org/mdast-util-to-markdown/-/mdast-util-to-markdown-2.1.2.tgz",
+      "integrity": "sha512-xj68wMTvGXVOKonmog6LwyJKrYXZPvlwabaryTjLh9LuvovB/KAH+kvi8Gjj+7rJjsFi23nkUxRQv1KqSroMqA==",
+      "dev": true,
+      "license": "MIT",
+      "dependencies": {
+        "@types/mdast": "^4.0.0",
+        "@types/unist": "^3.0.0",
+        "longest-streak": "^3.0.0",
+        "mdast-util-phrasing": "^4.0.0",
+        "mdast-util-to-string": "^4.0.0",
+        "micromark-util-classify-character": "^2.0.0",
+        "micromark-util-decode-string": "^2.0.0",
+        "unist-util-visit": "^5.0.0",
+        "zwitch": "^2.0.0"
+      },
+      "funding": {
+        "type": "opencollective",
+        "url": "https://opencollective.com/unified"
+      }
+    },
+    "node_modules/mdast-util-to-string": {
+      "version": "4.0.0",
+      "resolved": "https://registry.npmjs.org/mdast-util-to-string/-/mdast-util-to-string-4.0.0.tgz",
+      "integrity": "sha512-0H44vDimn51F0YwvxSJSm0eCDOJTRlmN0R1yBh4HLj9wiV1Dn0QoXGbvFAWj2hSItVTlCmBF1hqKlIyUBVFLPg==",
+      "dev": true,
+      "license": "MIT",
+      "dependencies": {
+        "@types/mdast": "^4.0.0"
+      },
+      "funding": {
+        "type": "opencollective",
+        "url": "https://opencollective.com/unified"
+      }
+    },
+    "node_modules/mdbox": {
+      "version": "0.1.1",
+      "resolved": "https://registry.npmjs.org/mdbox/-/mdbox-0.1.1.tgz",
+      "integrity": "sha512-jvLISenzbLRPWWamTG3THlhTcMbKWzJQNyTi61AVXhCBOC+gsldNTUfUNH8d3Vay83zGehFw3wZpF3xChzkTIQ==",
+      "dev": true,
+      "license": "MIT",
+      "dependencies": {
+        "md4w": "^0.2.6"
+      }
+    },
+    "node_modules/micromark": {
+      "version": "4.0.2",
+      "resolved": "https://registry.npmjs.org/micromark/-/micromark-4.0.2.tgz",
+      "integrity": "sha512-zpe98Q6kvavpCr1NPVSCMebCKfD7CA2NqZ+rykeNhONIJBpc1tFKt9hucLGwha3jNTNI8lHpctWJWoimVF4PfA==",
+      "dev": true,
+      "funding": [
+        {
+          "type": "GitHub Sponsors",
+          "url": "https://github.com/sponsors/unifiedjs"
+        },
+        {
+          "type": "OpenCollective",
+          "url": "https://opencollective.com/unified"
+        }
+      ],
+      "license": "MIT",
+      "dependencies": {
+        "@types/debug": "^4.0.0",
+        "debug": "^4.0.0",
+        "decode-named-character-reference": "^1.0.0",
+        "devlop": "^1.0.0",
+        "micromark-core-commonmark": "^2.0.0",
+        "micromark-factory-space": "^2.0.0",
+        "micromark-util-character": "^2.0.0",
+        "micromark-util-chunked": "^2.0.0",
+        "micromark-util-combine-extensions": "^2.0.0",
+        "micromark-util-decode-numeric-character-reference": "^2.0.0",
+        "micromark-util-encode": "^2.0.0",
+        "micromark-util-normalize-identifier": "^2.0.0",
+        "micromark-util-resolve-all": "^2.0.0",
+        "micromark-util-sanitize-uri": "^2.0.0",
+        "micromark-util-subtokenize": "^2.0.0",
+        "micromark-util-symbol": "^2.0.0",
+        "micromark-util-types": "^2.0.0"
+      }
+    },
+    "node_modules/micromark-core-commonmark": {
+      "version": "2.0.3",
+      "resolved": "https://registry.npmjs.org/micromark-core-commonmark/-/micromark-core-commonmark-2.0.3.tgz",
+      "integrity": "sha512-RDBrHEMSxVFLg6xvnXmb1Ayr2WzLAWjeSATAoxwKYJV94TeNavgoIdA0a9ytzDSVzBy2YKFK+emCPOEibLeCrg==",
+      "dev": true,
+      "funding": [
+        {
+          "type": "GitHub Sponsors",
+          "url": "https://github.com/sponsors/unifiedjs"
+        },
+        {
+          "type": "OpenCollective",
+          "url": "https://opencollective.com/unified"
+        }
+      ],
+      "license": "MIT",
+      "dependencies": {
+        "decode-named-character-reference": "^1.0.0",
+        "devlop": "^1.0.0",
+        "micromark-factory-destination": "^2.0.0",
+        "micromark-factory-label": "^2.0.0",
+        "micromark-factory-space": "^2.0.0",
+        "micromark-factory-title": "^2.0.0",
+        "micromark-factory-whitespace": "^2.0.0",
+        "micromark-util-character": "^2.0.0",
+        "micromark-util-chunked": "^2.0.0",
+        "micromark-util-classify-character": "^2.0.0",
+        "micromark-util-html-tag-name": "^2.0.0",
+        "micromark-util-normalize-identifier": "^2.0.0",
+        "micromark-util-resolve-all": "^2.0.0",
+        "micromark-util-subtokenize": "^2.0.0",
+        "micromark-util-symbol": "^2.0.0",
+        "micromark-util-types": "^2.0.0"
+      }
+    },
+    "node_modules/micromark-extension-frontmatter": {
+      "version": "2.0.0",
+      "resolved": "https://registry.npmjs.org/micromark-extension-frontmatter/-/micromark-extension-frontmatter-2.0.0.tgz",
+      "integrity": "sha512-C4AkuM3dA58cgZha7zVnuVxBhDsbttIMiytjgsM2XbHAB2faRVaHRle40558FBN+DJcrLNCoqG5mlrpdU4cRtg==",
+      "dev": true,
+      "license": "MIT",
+      "dependencies": {
+        "fault": "^2.0.0",
+        "micromark-util-character": "^2.0.0",
+        "micromark-util-symbol": "^2.0.0",
+        "micromark-util-types": "^2.0.0"
+      },
+      "funding": {
+        "type": "opencollective",
+        "url": "https://opencollective.com/unified"
+      }
+    },
+    "node_modules/micromark-extension-gfm": {
+      "version": "3.0.0",
+      "resolved": "https://registry.npmjs.org/micromark-extension-gfm/-/micromark-extension-gfm-3.0.0.tgz",
+      "integrity": "sha512-vsKArQsicm7t0z2GugkCKtZehqUm31oeGBV/KVSorWSy8ZlNAv7ytjFhvaryUiCUJYqs+NoE6AFhpQvBTM6Q4w==",
+      "dev": true,
+      "license": "MIT",
+      "dependencies": {
+        "micromark-extension-gfm-autolink-literal": "^2.0.0",
+        "micromark-extension-gfm-footnote": "^2.0.0",
+        "micromark-extension-gfm-strikethrough": "^2.0.0",
+        "micromark-extension-gfm-table": "^2.0.0",
+        "micromark-extension-gfm-tagfilter": "^2.0.0",
+        "micromark-extension-gfm-task-list-item": "^2.0.0",
+        "micromark-util-combine-extensions": "^2.0.0",
+        "micromark-util-types": "^2.0.0"
+      },
+      "funding": {
+        "type": "opencollective",
+        "url": "https://opencollective.com/unified"
+      }
+    },
+    "node_modules/micromark-extension-gfm-autolink-literal": {
+      "version": "2.1.0",
+      "resolved": "https://registry.npmjs.org/micromark-extension-gfm-autolink-literal/-/micromark-extension-gfm-autolink-literal-2.1.0.tgz",
+      "integrity": "sha512-oOg7knzhicgQ3t4QCjCWgTmfNhvQbDDnJeVu9v81r7NltNCVmhPy1fJRX27pISafdjL+SVc4d3l48Gb6pbRypw==",
+      "dev": true,
+      "license": "MIT",
+      "dependencies": {
+        "micromark-util-character": "^2.0.0",
+        "micromark-util-sanitize-uri": "^2.0.0",
+        "micromark-util-symbol": "^2.0.0",
+        "micromark-util-types": "^2.0.0"
+      },
+      "funding": {
+        "type": "opencollective",
+        "url": "https://opencollective.com/unified"
+      }
+    },
+    "node_modules/micromark-extension-gfm-footnote": {
+      "version": "2.1.0",
+      "resolved": "https://registry.npmjs.org/micromark-extension-gfm-footnote/-/micromark-extension-gfm-footnote-2.1.0.tgz",
+      "integrity": "sha512-/yPhxI1ntnDNsiHtzLKYnE3vf9JZ6cAisqVDauhp4CEHxlb4uoOTxOCJ+9s51bIB8U1N1FJ1RXOKTIlD5B/gqw==",
+      "dev": true,
+      "license": "MIT",
+      "dependencies": {
+        "devlop": "^1.0.0",
+        "micromark-core-commonmark": "^2.0.0",
+        "micromark-factory-space": "^2.0.0",
+        "micromark-util-character": "^2.0.0",
+        "micromark-util-normalize-identifier": "^2.0.0",
+        "micromark-util-sanitize-uri": "^2.0.0",
+        "micromark-util-symbol": "^2.0.0",
+        "micromark-util-types": "^2.0.0"
+      },
+      "funding": {
+        "type": "opencollective",
+        "url": "https://opencollective.com/unified"
+      }
+    },
+    "node_modules/micromark-extension-gfm-strikethrough": {
+      "version": "2.1.0",
+      "resolved": "https://registry.npmjs.org/micromark-extension-gfm-strikethrough/-/micromark-extension-gfm-strikethrough-2.1.0.tgz",
+      "integrity": "sha512-ADVjpOOkjz1hhkZLlBiYA9cR2Anf8F4HqZUO6e5eDcPQd0Txw5fxLzzxnEkSkfnD0wziSGiv7sYhk/ktvbf1uw==",
+      "dev": true,
+      "license": "MIT",
+      "dependencies": {
+        "devlop": "^1.0.0",
+        "micromark-util-chunked": "^2.0.0",
+        "micromark-util-classify-character": "^2.0.0",
+        "micromark-util-resolve-all": "^2.0.0",
+        "micromark-util-symbol": "^2.0.0",
+        "micromark-util-types": "^2.0.0"
+      },
+      "funding": {
+        "type": "opencollective",
+        "url": "https://opencollective.com/unified"
+      }
+    },
+    "node_modules/micromark-extension-gfm-table": {
+      "version": "2.1.1",
+      "resolved": "https://registry.npmjs.org/micromark-extension-gfm-table/-/micromark-extension-gfm-table-2.1.1.tgz",
+      "integrity": "sha512-t2OU/dXXioARrC6yWfJ4hqB7rct14e8f7m0cbI5hUmDyyIlwv5vEtooptH8INkbLzOatzKuVbQmAYcbWoyz6Dg==",
+      "dev": true,
+      "license": "MIT",
+      "dependencies": {
+        "devlop": "^1.0.0",
+        "micromark-factory-space": "^2.0.0",
+        "micromark-util-character": "^2.0.0",
+        "micromark-util-symbol": "^2.0.0",
+        "micromark-util-types": "^2.0.0"
+      },
+      "funding": {
+        "type": "opencollective",
+        "url": "https://opencollective.com/unified"
+      }
+    },
+    "node_modules/micromark-extension-gfm-tagfilter": {
+      "version": "2.0.0",
+      "resolved": "https://registry.npmjs.org/micromark-extension-gfm-tagfilter/-/micromark-extension-gfm-tagfilter-2.0.0.tgz",
+      "integrity": "sha512-xHlTOmuCSotIA8TW1mDIM6X2O1SiX5P9IuDtqGonFhEK0qgRI4yeC6vMxEV2dgyr2TiD+2PQ10o+cOhdVAcwfg==",
+      "dev": true,
+      "license": "MIT",
+      "dependencies": {
+        "micromark-util-types": "^2.0.0"
+      },
+      "funding": {
+        "type": "opencollective",
+        "url": "https://opencollective.com/unified"
+      }
+    },
+    "node_modules/micromark-extension-gfm-task-list-item": {
+      "version": "2.1.0",
+      "resolved": "https://registry.npmjs.org/micromark-extension-gfm-task-list-item/-/micromark-extension-gfm-task-list-item-2.1.0.tgz",
+      "integrity": "sha512-qIBZhqxqI6fjLDYFTBIa4eivDMnP+OZqsNwmQ3xNLE4Cxwc+zfQEfbs6tzAo2Hjq+bh6q5F+Z8/cksrLFYWQQw==",
+      "dev": true,
+      "license": "MIT",
+      "dependencies": {
+        "devlop": "^1.0.0",
+        "micromark-factory-space": "^2.0.0",
+        "micromark-util-character": "^2.0.0",
+        "micromark-util-symbol": "^2.0.0",
+        "micromark-util-types": "^2.0.0"
+      },
+      "funding": {
+        "type": "opencollective",
+        "url": "https://opencollective.com/unified"
+      }
+    },
+    "node_modules/micromark-factory-destination": {
+      "version": "2.0.1",
+      "resolved": "https://registry.npmjs.org/micromark-factory-destination/-/micromark-factory-destination-2.0.1.tgz",
+      "integrity": "sha512-Xe6rDdJlkmbFRExpTOmRj9N3MaWmbAgdpSrBQvCFqhezUn4AHqJHbaEnfbVYYiexVSs//tqOdY/DxhjdCiJnIA==",
+      "dev": true,
+      "funding": [
+        {
+          "type": "GitHub Sponsors",
+          "url": "https://github.com/sponsors/unifiedjs"
+        },
+        {
+          "type": "OpenCollective",
+          "url": "https://opencollective.com/unified"
+        }
+      ],
+      "license": "MIT",
+      "dependencies": {
+        "micromark-util-character": "^2.0.0",
+        "micromark-util-symbol": "^2.0.0",
+        "micromark-util-types": "^2.0.0"
+      }
+    },
+    "node_modules/micromark-factory-label": {
+      "version": "2.0.1",
+      "resolved": "https://registry.npmjs.org/micromark-factory-label/-/micromark-factory-label-2.0.1.tgz",
+      "integrity": "sha512-VFMekyQExqIW7xIChcXn4ok29YE3rnuyveW3wZQWWqF4Nv9Wk5rgJ99KzPvHjkmPXF93FXIbBp6YdW3t71/7Vg==",
+      "dev": true,
+      "funding": [
+        {
+          "type": "GitHub Sponsors",
+          "url": "https://github.com/sponsors/unifiedjs"
+        },
+        {
+          "type": "OpenCollective",
+          "url": "https://opencollective.com/unified"
+        }
+      ],
+      "license": "MIT",
+      "dependencies": {
+        "devlop": "^1.0.0",
+        "micromark-util-character": "^2.0.0",
+        "micromark-util-symbol": "^2.0.0",
+        "micromark-util-types": "^2.0.0"
+      }
+    },
+    "node_modules/micromark-factory-space": {
+      "version": "2.0.1",
+      "resolved": "https://registry.npmjs.org/micromark-factory-space/-/micromark-factory-space-2.0.1.tgz",
+      "integrity": "sha512-zRkxjtBxxLd2Sc0d+fbnEunsTj46SWXgXciZmHq0kDYGnck/ZSGj9/wULTV95uoeYiK5hRXP2mJ98Uo4cq/LQg==",
+      "dev": true,
+      "funding": [
+        {
+          "type": "GitHub Sponsors",
+          "url": "https://github.com/sponsors/unifiedjs"
+        },
+        {
+          "type": "OpenCollective",
+          "url": "https://opencollective.com/unified"
+        }
+      ],
+      "license": "MIT",
+      "dependencies": {
+        "micromark-util-character": "^2.0.0",
+        "micromark-util-types": "^2.0.0"
+      }
+    },
+    "node_modules/micromark-factory-title": {
+      "version": "2.0.1",
+      "resolved": "https://registry.npmjs.org/micromark-factory-title/-/micromark-factory-title-2.0.1.tgz",
+      "integrity": "sha512-5bZ+3CjhAd9eChYTHsjy6TGxpOFSKgKKJPJxr293jTbfry2KDoWkhBb6TcPVB4NmzaPhMs1Frm9AZH7OD4Cjzw==",
+      "dev": true,
+      "funding": [
+        {
+          "type": "GitHub Sponsors",
+          "url": "https://github.com/sponsors/unifiedjs"
+        },
+        {
+          "type": "OpenCollective",
+          "url": "https://opencollective.com/unified"
+        }
+      ],
+      "license": "MIT",
+      "dependencies": {
+        "micromark-factory-space": "^2.0.0",
+        "micromark-util-character": "^2.0.0",
+        "micromark-util-symbol": "^2.0.0",
+        "micromark-util-types": "^2.0.0"
+      }
+    },
+    "node_modules/micromark-factory-whitespace": {
+      "version": "2.0.1",
+      "resolved": "https://registry.npmjs.org/micromark-factory-whitespace/-/micromark-factory-whitespace-2.0.1.tgz",
+      "integrity": "sha512-Ob0nuZ3PKt/n0hORHyvoD9uZhr+Za8sFoP+OnMcnWK5lngSzALgQYKMr9RJVOWLqQYuyn6ulqGWSXdwf6F80lQ==",
+      "dev": true,
+      "funding": [
+        {
+          "type": "GitHub Sponsors",
+          "url": "https://github.com/sponsors/unifiedjs"
+        },
+        {
+          "type": "OpenCollective",
+          "url": "https://opencollective.com/unified"
+        }
+      ],
+      "license": "MIT",
+      "dependencies": {
+        "micromark-factory-space": "^2.0.0",
+        "micromark-util-character": "^2.0.0",
+        "micromark-util-symbol": "^2.0.0",
+        "micromark-util-types": "^2.0.0"
+      }
+    },
+    "node_modules/micromark-util-character": {
+      "version": "2.1.1",
+      "resolved": "https://registry.npmjs.org/micromark-util-character/-/micromark-util-character-2.1.1.tgz",
+      "integrity": "sha512-wv8tdUTJ3thSFFFJKtpYKOYiGP2+v96Hvk4Tu8KpCAsTMs6yi+nVmGh1syvSCsaxz45J6Jbw+9DD6g97+NV67Q==",
+      "dev": true,
+      "funding": [
+        {
+          "type": "GitHub Sponsors",
+          "url": "https://github.com/sponsors/unifiedjs"
+        },
+        {
+          "type": "OpenCollective",
+          "url": "https://opencollective.com/unified"
+        }
+      ],
+      "license": "MIT",
+      "dependencies": {
+        "micromark-util-symbol": "^2.0.0",
+        "micromark-util-types": "^2.0.0"
+      }
+    },
+    "node_modules/micromark-util-chunked": {
+      "version": "2.0.1",
+      "resolved": "https://registry.npmjs.org/micromark-util-chunked/-/micromark-util-chunked-2.0.1.tgz",
+      "integrity": "sha512-QUNFEOPELfmvv+4xiNg2sRYeS/P84pTW0TCgP5zc9FpXetHY0ab7SxKyAQCNCc1eK0459uoLI1y5oO5Vc1dbhA==",
+      "dev": true,
+      "funding": [
+        {
+          "type": "GitHub Sponsors",
+          "url": "https://github.com/sponsors/unifiedjs"
+        },
+        {
+          "type": "OpenCollective",
+          "url": "https://opencollective.com/unified"
+        }
+      ],
+      "license": "MIT",
+      "dependencies": {
+        "micromark-util-symbol": "^2.0.0"
+      }
+    },
+    "node_modules/micromark-util-classify-character": {
+      "version": "2.0.1",
+      "resolved": "https://registry.npmjs.org/micromark-util-classify-character/-/micromark-util-classify-character-2.0.1.tgz",
+      "integrity": "sha512-K0kHzM6afW/MbeWYWLjoHQv1sgg2Q9EccHEDzSkxiP/EaagNzCm7T/WMKZ3rjMbvIpvBiZgwR3dKMygtA4mG1Q==",
+      "dev": true,
+      "funding": [
+        {
+          "type": "GitHub Sponsors",
+          "url": "https://github.com/sponsors/unifiedjs"
+        },
+        {
+          "type": "OpenCollective",
+          "url": "https://opencollective.com/unified"
+        }
+      ],
+      "license": "MIT",
+      "dependencies": {
+        "micromark-util-character": "^2.0.0",
+        "micromark-util-symbol": "^2.0.0",
+        "micromark-util-types": "^2.0.0"
+      }
+    },
+    "node_modules/micromark-util-combine-extensions": {
+      "version": "2.0.1",
+      "resolved": "https://registry.npmjs.org/micromark-util-combine-extensions/-/micromark-util-combine-extensions-2.0.1.tgz",
+      "integrity": "sha512-OnAnH8Ujmy59JcyZw8JSbK9cGpdVY44NKgSM7E9Eh7DiLS2E9RNQf0dONaGDzEG9yjEl5hcqeIsj4hfRkLH/Bg==",
+      "dev": true,
+      "funding": [
+        {
+          "type": "GitHub Sponsors",
+          "url": "https://github.com/sponsors/unifiedjs"
+        },
+        {
+          "type": "OpenCollective",
+          "url": "https://opencollective.com/unified"
+        }
+      ],
+      "license": "MIT",
+      "dependencies": {
+        "micromark-util-chunked": "^2.0.0",
+        "micromark-util-types": "^2.0.0"
+      }
+    },
+    "node_modules/micromark-util-decode-numeric-character-reference": {
+      "version": "2.0.2",
+      "resolved": "https://registry.npmjs.org/micromark-util-decode-numeric-character-reference/-/micromark-util-decode-numeric-character-reference-2.0.2.tgz",
+      "integrity": "sha512-ccUbYk6CwVdkmCQMyr64dXz42EfHGkPQlBj5p7YVGzq8I7CtjXZJrubAYezf7Rp+bjPseiROqe7G6foFd+lEuw==",
+      "dev": true,
+      "funding": [
+        {
+          "type": "GitHub Sponsors",
+          "url": "https://github.com/sponsors/unifiedjs"
+        },
+        {
+          "type": "OpenCollective",
+          "url": "https://opencollective.com/unified"
+        }
+      ],
+      "license": "MIT",
+      "dependencies": {
+        "micromark-util-symbol": "^2.0.0"
+      }
+    },
+    "node_modules/micromark-util-decode-string": {
+      "version": "2.0.1",
+      "resolved": "https://registry.npmjs.org/micromark-util-decode-string/-/micromark-util-decode-string-2.0.1.tgz",
+      "integrity": "sha512-nDV/77Fj6eH1ynwscYTOsbK7rR//Uj0bZXBwJZRfaLEJ1iGBR6kIfNmlNqaqJf649EP0F3NWNdeJi03elllNUQ==",
+      "dev": true,
+      "funding": [
+        {
+          "type": "GitHub Sponsors",
+          "url": "https://github.com/sponsors/unifiedjs"
+        },
+        {
+          "type": "OpenCollective",
+          "url": "https://opencollective.com/unified"
+        }
+      ],
+      "license": "MIT",
+      "dependencies": {
+        "decode-named-character-reference": "^1.0.0",
+        "micromark-util-character": "^2.0.0",
+        "micromark-util-decode-numeric-character-reference": "^2.0.0",
+        "micromark-util-symbol": "^2.0.0"
+      }
+    },
+    "node_modules/micromark-util-encode": {
+      "version": "2.0.1",
+      "resolved": "https://registry.npmjs.org/micromark-util-encode/-/micromark-util-encode-2.0.1.tgz",
+      "integrity": "sha512-c3cVx2y4KqUnwopcO9b/SCdo2O67LwJJ/UyqGfbigahfegL9myoEFoDYZgkT7f36T0bLrM9hZTAaAyH+PCAXjw==",
+      "dev": true,
+      "funding": [
+        {
+          "type": "GitHub Sponsors",
+          "url": "https://github.com/sponsors/unifiedjs"
+        },
+        {
+          "type": "OpenCollective",
+          "url": "https://opencollective.com/unified"
+        }
+      ],
+      "license": "MIT"
+    },
+    "node_modules/micromark-util-html-tag-name": {
+      "version": "2.0.1",
+      "resolved": "https://registry.npmjs.org/micromark-util-html-tag-name/-/micromark-util-html-tag-name-2.0.1.tgz",
+      "integrity": "sha512-2cNEiYDhCWKI+Gs9T0Tiysk136SnR13hhO8yW6BGNyhOC4qYFnwF1nKfD3HFAIXA5c45RrIG1ub11GiXeYd1xA==",
+      "dev": true,
+      "funding": [
+        {
+          "type": "GitHub Sponsors",
+          "url": "https://github.com/sponsors/unifiedjs"
+        },
+        {
+          "type": "OpenCollective",
+          "url": "https://opencollective.com/unified"
+        }
+      ],
+      "license": "MIT"
+    },
+    "node_modules/micromark-util-normalize-identifier": {
+      "version": "2.0.1",
+      "resolved": "https://registry.npmjs.org/micromark-util-normalize-identifier/-/micromark-util-normalize-identifier-2.0.1.tgz",
+      "integrity": "sha512-sxPqmo70LyARJs0w2UclACPUUEqltCkJ6PhKdMIDuJ3gSf/Q+/GIe3WKl0Ijb/GyH9lOpUkRAO2wp0GVkLvS9Q==",
+      "dev": true,
+      "funding": [
+        {
+          "type": "GitHub Sponsors",
+          "url": "https://github.com/sponsors/unifiedjs"
+        },
+        {
+          "type": "OpenCollective",
+          "url": "https://opencollective.com/unified"
+        }
+      ],
+      "license": "MIT",
+      "dependencies": {
+        "micromark-util-symbol": "^2.0.0"
+      }
+    },
+    "node_modules/micromark-util-resolve-all": {
+      "version": "2.0.1",
+      "resolved": "https://registry.npmjs.org/micromark-util-resolve-all/-/micromark-util-resolve-all-2.0.1.tgz",
+      "integrity": "sha512-VdQyxFWFT2/FGJgwQnJYbe1jjQoNTS4RjglmSjTUlpUMa95Htx9NHeYW4rGDJzbjvCsl9eLjMQwGeElsqmzcHg==",
+      "dev": true,
+      "funding": [
+        {
+          "type": "GitHub Sponsors",
+          "url": "https://github.com/sponsors/unifiedjs"
+        },
+        {
+          "type": "OpenCollective",
+          "url": "https://opencollective.com/unified"
+        }
+      ],
+      "license": "MIT",
+      "dependencies": {
+        "micromark-util-types": "^2.0.0"
+      }
+    },
+    "node_modules/micromark-util-sanitize-uri": {
+      "version": "2.0.1",
+      "resolved": "https://registry.npmjs.org/micromark-util-sanitize-uri/-/micromark-util-sanitize-uri-2.0.1.tgz",
+      "integrity": "sha512-9N9IomZ/YuGGZZmQec1MbgxtlgougxTodVwDzzEouPKo3qFWvymFHWcnDi2vzV1ff6kas9ucW+o3yzJK9YB1AQ==",
+      "dev": true,
+      "funding": [
+        {
+          "type": "GitHub Sponsors",
+          "url": "https://github.com/sponsors/unifiedjs"
+        },
+        {
+          "type": "OpenCollective",
+          "url": "https://opencollective.com/unified"
+        }
+      ],
+      "license": "MIT",
+      "dependencies": {
+        "micromark-util-character": "^2.0.0",
+        "micromark-util-encode": "^2.0.0",
+        "micromark-util-symbol": "^2.0.0"
+      }
+    },
+    "node_modules/micromark-util-subtokenize": {
+      "version": "2.1.0",
+      "resolved": "https://registry.npmjs.org/micromark-util-subtokenize/-/micromark-util-subtokenize-2.1.0.tgz",
+      "integrity": "sha512-XQLu552iSctvnEcgXw6+Sx75GflAPNED1qx7eBJ+wydBb2KCbRZe+NwvIEEMM83uml1+2WSXpBAcp9IUCgCYWA==",
+      "dev": true,
+      "funding": [
+        {
+          "type": "GitHub Sponsors",
+          "url": "https://github.com/sponsors/unifiedjs"
+        },
+        {
+          "type": "OpenCollective",
+          "url": "https://opencollective.com/unified"
+        }
+      ],
+      "license": "MIT",
+      "dependencies": {
+        "devlop": "^1.0.0",
+        "micromark-util-chunked": "^2.0.0",
+        "micromark-util-symbol": "^2.0.0",
+        "micromark-util-types": "^2.0.0"
+      }
+    },
+    "node_modules/micromark-util-symbol": {
+      "version": "2.0.1",
+      "resolved": "https://registry.npmjs.org/micromark-util-symbol/-/micromark-util-symbol-2.0.1.tgz",
+      "integrity": "sha512-vs5t8Apaud9N28kgCrRUdEed4UJ+wWNvicHLPxCa9ENlYuAY31M0ETy5y1vA33YoNPDFTghEbnh6efaE8h4x0Q==",
+      "dev": true,
+      "funding": [
+        {
+          "type": "GitHub Sponsors",
+          "url": "https://github.com/sponsors/unifiedjs"
+        },
+        {
+          "type": "OpenCollective",
+          "url": "https://opencollective.com/unified"
+        }
+      ],
+      "license": "MIT"
+    },
+    "node_modules/micromark-util-types": {
+      "version": "2.0.2",
+      "resolved": "https://registry.npmjs.org/micromark-util-types/-/micromark-util-types-2.0.2.tgz",
+      "integrity": "sha512-Yw0ECSpJoViF1qTU4DC6NwtC4aWGt1EkzaQB8KPPyCRR8z9TWeV0HbEFGTO+ZY1wB22zmxnJqhPyTpOVCpeHTA==",
+      "dev": true,
+      "funding": [
+        {
+          "type": "GitHub Sponsors",
+          "url": "https://github.com/sponsors/unifiedjs"
+        },
+        {
+          "type": "OpenCollective",
+          "url": "https://opencollective.com/unified"
+        }
+      ],
+      "license": "MIT"
+    },
+    "node_modules/minimatch": {
+      "version": "3.1.5",
+      "resolved": "https://registry.npmjs.org/minimatch/-/minimatch-3.1.5.tgz",
+      "integrity": "sha512-VgjWUsnnT6n+NUk6eZq77zeFdpW2LWDzP6zFGrCbHXiYNul5Dzqk2HHQ5uFH2DNW5Xbp8+jVzaeNt94ssEEl4w==",
+      "dev": true,
+      "license": "ISC",
+      "dependencies": {
+        "brace-expansion": "^1.1.7"
+      },
+      "engines": {
+        "node": "*"
+      }
+    },
+    "node_modules/mlly": {
+      "version": "1.8.1",
+      "resolved": "https://registry.npmjs.org/mlly/-/mlly-1.8.1.tgz",
+      "integrity": "sha512-SnL6sNutTwRWWR/vcmCYHSADjiEesp5TGQQ0pXyLhW5IoeibRlF/CbSLailbB3CNqJUk9cVJ9dUDnbD7GrcHBQ==",
+      "dev": true,
+      "license": "MIT",
+      "dependencies": {
+        "acorn": "^8.16.0",
+        "pathe": "^2.0.3",
+        "pkg-types": "^1.3.1",
+        "ufo": "^1.6.3"
+      }
+    },
+    "node_modules/mlly/node_modules/confbox": {
+      "version": "0.1.8",
+      "resolved": "https://registry.npmjs.org/confbox/-/confbox-0.1.8.tgz",
+      "integrity": "sha512-RMtmw0iFkeR4YV+fUOSucriAQNb9g8zFR52MWCtl+cCZOFRNL6zeB395vPzFhEjjn4fMxXudmELnl/KF/WrK6w==",
+      "dev": true,
+      "license": "MIT"
+    },
+    "node_modules/mlly/node_modules/pkg-types": {
+      "version": "1.3.1",
+      "resolved": "https://registry.npmjs.org/pkg-types/-/pkg-types-1.3.1.tgz",
+      "integrity": "sha512-/Jm5M4RvtBFVkKWRu2BLUTNP8/M2a+UwuAX+ae4770q1qVGtfjG+WTCupoZixokjmHiry8uI+dlY8KXYV5HVVQ==",
+      "dev": true,
+      "license": "MIT",
+      "dependencies": {
+        "confbox": "^0.1.8",
+        "mlly": "^1.7.4",
+        "pathe": "^2.0.1"
+      }
+    },
+    "node_modules/moment": {
+      "version": "2.30.1",
+      "resolved": "https://registry.npmjs.org/moment/-/moment-2.30.1.tgz",
+      "integrity": "sha512-uEmtNhbDOrWPFS+hdjFCBfy9f2YoyzRpwcl+DqpC6taX21FzsTLQVbMV/W7PzNSX6x/bhC1zA3c2UQ5NzH6how==",
+      "dev": true,
+      "license": "MIT",
+      "engines": {
+        "node": "*"
+      }
+    },
+    "node_modules/mri": {
+      "version": "1.2.0",
+      "resolved": "https://registry.npmjs.org/mri/-/mri-1.2.0.tgz",
+      "integrity": "sha512-tzzskb3bG8LvYGFF/mDTpq3jpI6Q9wc3LEmBaghu+DdCssd1FakN7Bc0hVNmEyGq1bq3RgfkCb3cmQLpNPOroA==",
+      "dev": true,
+      "license": "MIT",
+      "engines": {
+        "node": ">=4"
+      }
+    },
+    "node_modules/ms": {
+      "version": "2.1.3",
+      "resolved": "https://registry.npmjs.org/ms/-/ms-2.1.3.tgz",
+      "integrity": "sha512-6FlzubTLZG3J2a/NVCAleEhjzq5oxgHyaCU9yYXvcLsvoVaHJq/s5xXI6/XXP6tz7R9xAOtHnSO/tXtF3WRTlA==",
+      "dev": true,
+      "license": "MIT"
+    },
+    "node_modules/nanoid": {
+      "version": "3.3.11",
+      "resolved": "https://registry.npmjs.org/nanoid/-/nanoid-3.3.11.tgz",
+      "integrity": "sha512-N8SpfPUnUp1bK+PMYW8qSWdl9U+wwNWI4QKxOYDy9JAro3WMX7p2OeVRF9v+347pnakNevPmiHhNmZ2HbFA76w==",
+      "dev": true,
+      "funding": [
+        {
+          "type": "github",
+          "url": "https://github.com/sponsors/ai"
+        }
+      ],
+      "license": "MIT",
+      "bin": {
+        "nanoid": "bin/nanoid.cjs"
+      },
+      "engines": {
+        "node": "^10 || ^12 || ^13.7 || ^14 || >=15.0.1"
+      }
+    },
+    "node_modules/natural-compare": {
+      "version": "1.4.0",
+      "resolved": "https://registry.npmjs.org/natural-compare/-/natural-compare-1.4.0.tgz",
+      "integrity": "sha512-OWND8ei3VtNC9h7V60qff3SVobHr996CTwgxubgyQYEpg290h9J0buyECNNJexkFm5sOajh5G116RYA1c8ZMSw==",
+      "dev": true,
+      "license": "MIT"
+    },
+    "node_modules/node-addon-api": {
+      "version": "7.1.1",
+      "resolved": "https://registry.npmjs.org/node-addon-api/-/node-addon-api-7.1.1.tgz",
+      "integrity": "sha512-5m3bsyrjFWE1xf7nz7YXdN4udnVtXK6/Yfgn5qnahL6bCkf2yKt4k3nuTKAtT4r3IG8JNR2ncsIMdZuAzJjHQQ==",
+      "dev": true,
+      "license": "MIT"
+    },
+    "node_modules/node-fetch-native": {
+      "version": "1.6.7",
+      "resolved": "https://registry.npmjs.org/node-fetch-native/-/node-fetch-native-1.6.7.tgz",
+      "integrity": "sha512-g9yhqoedzIUm0nTnTqAQvueMPVOuIY16bqgAJJC8XOOubYFNwz6IER9qs0Gq2Xd0+CecCKFjtdDTMA4u4xG06Q==",
+      "dev": true,
+      "license": "MIT"
+    },
+    "node_modules/node-releases": {
+      "version": "2.0.36",
+      "resolved": "https://registry.npmjs.org/node-releases/-/node-releases-2.0.36.tgz",
+      "integrity": "sha512-TdC8FSgHz8Mwtw9g5L4gR/Sh9XhSP/0DEkQxfEFXOpiul5IiHgHan2VhYYb6agDSfp4KuvltmGApc8HMgUrIkA==",
+      "dev": true,
+      "license": "MIT"
+    },
+    "node_modules/nypm": {
+      "version": "0.6.5",
+      "resolved": "https://registry.npmjs.org/nypm/-/nypm-0.6.5.tgz",
+      "integrity": "sha512-K6AJy1GMVyfyMXRVB88700BJqNUkByijGJM8kEHpLdcAt+vSQAVfkWWHYzuRXHSY6xA2sNc5RjTj0p9rE2izVQ==",
+      "dev": true,
+      "license": "MIT",
+      "dependencies": {
+        "citty": "^0.2.0",
+        "pathe": "^2.0.3",
+        "tinyexec": "^1.0.2"
+      },
+      "bin": {
+        "nypm": "dist/cli.mjs"
+      },
+      "engines": {
+        "node": ">=18"
+      }
+    },
+    "node_modules/obug": {
+      "version": "2.1.1",
+      "resolved": "https://registry.npmjs.org/obug/-/obug-2.1.1.tgz",
+      "integrity": "sha512-uTqF9MuPraAQ+IsnPf366RG4cP9RtUi7MLO1N3KEc+wb0a6yKpeL0lmk2IB1jY5KHPAlTc6T/JRdC/YqxHNwkQ==",
+      "dev": true,
+      "funding": [
+        "https://github.com/sponsors/sxzz",
+        "https://opencollective.com/debug"
+      ],
+      "license": "MIT"
+    },
+    "node_modules/obuild": {
+      "version": "0.4.32",
+      "resolved": "https://registry.npmjs.org/obuild/-/obuild-0.4.32.tgz",
+      "integrity": "sha512-U6NNt0ZL6lCE6B6aU+qcBwbFU4g2DwrO/68aKhyXNUXgq0jf+fyHg9s9+ETZOLmlfJvQwub5mHF3aohGgWGYKQ==",
+      "dev": true,
+      "license": "MIT",
+      "dependencies": {
+        "c12": "^4.0.0-beta.3",
+        "consola": "^3.4.2",
+        "defu": "^6.1.4",
+        "exsolve": "^1.0.8",
+        "magic-string": "^0.30.21",
+        "pathe": "^2.0.3",
+        "pretty-bytes": "^7.1.0",
+        "rolldown": "^1.0.0-rc.7",
+        "rolldown-plugin-dts": "^0.22.4",
+        "rollup-plugin-license": "^3.7.0",
+        "tinyglobby": "^0.2.15"
+      },
+      "bin": {
+        "obuild": "dist/cli.mjs"
+      }
+    },
+    "node_modules/obuild/node_modules/c12": {
+      "version": "4.0.0-beta.3",
+      "resolved": "https://registry.npmjs.org/c12/-/c12-4.0.0-beta.3.tgz",
+      "integrity": "sha512-15pHxeM4kKKnF1zPgvXq/ETPhf9DRLGg0Id3GAyQhQqYxt8WkUCbo0NipHjQUUNC5VPP8TQA32pGPDUZmAi/3g==",
+      "dev": true,
+      "license": "MIT",
+      "dependencies": {
+        "confbox": "^0.2.4",
+        "defu": "^6.1.4",
+        "exsolve": "^1.0.8",
+        "pathe": "^2.0.3",
+        "pkg-types": "^2.3.0",
+        "rc9": "^3.0.0"
+      },
+      "peerDependencies": {
+        "chokidar": "^5",
+        "dotenv": "*",
+        "giget": "*",
+        "jiti": "*",
+        "magicast": "*"
+      },
+      "peerDependenciesMeta": {
+        "chokidar": {
+          "optional": true
+        },
+        "dotenv": {
+          "optional": true
+        },
+        "giget": {
+          "optional": true
+        },
+        "jiti": {
+          "optional": true
+        },
+        "magicast": {
+          "optional": true
+        }
+      }
+    },
+    "node_modules/obuild/node_modules/rc9": {
+      "version": "3.0.0",
+      "resolved": "https://registry.npmjs.org/rc9/-/rc9-3.0.0.tgz",
+      "integrity": "sha512-MGOue0VqscKWQ104udASX/3GYDcKyPI4j4F8gu/jHHzglpmy9a/anZK3PNe8ug6aZFl+9GxLtdhe3kVZuMaQbA==",
+      "dev": true,
+      "license": "MIT",
+      "dependencies": {
+        "defu": "^6.1.4",
+        "destr": "^2.0.5"
+      }
+    },
+    "node_modules/ofetch": {
+      "version": "1.5.1",
+      "resolved": "https://registry.npmjs.org/ofetch/-/ofetch-1.5.1.tgz",
+      "integrity": "sha512-2W4oUZlVaqAPAil6FUg/difl6YhqhUR7x2eZY4bQCko22UXg3hptq9KLQdqFClV+Wu85UX7hNtdGTngi/1BxcA==",
+      "dev": true,
+      "license": "MIT",
+      "dependencies": {
+        "destr": "^2.0.5",
+        "node-fetch-native": "^1.6.7",
+        "ufo": "^1.6.1"
+      }
+    },
+    "node_modules/ohash": {
+      "version": "2.0.11",
+      "resolved": "https://registry.npmjs.org/ohash/-/ohash-2.0.11.tgz",
+      "integrity": "sha512-RdR9FQrFwNBNXAr4GixM8YaRZRJ5PUWbKYbE5eOsrwAjJW0q2REGcf79oYPsLyskQCZG1PLN+S/K1V00joZAoQ==",
+      "dev": true,
+      "license": "MIT"
+    },
+    "node_modules/open": {
+      "version": "10.2.0",
+      "resolved": "https://registry.npmjs.org/open/-/open-10.2.0.tgz",
+      "integrity": "sha512-YgBpdJHPyQ2UE5x+hlSXcnejzAvD0b22U2OuAP+8OnlJT+PjWPxtgmGqKKc+RgTM63U9gN0YzrYc71R2WT/hTA==",
+      "dev": true,
+      "license": "MIT",
+      "dependencies": {
+        "default-browser": "^5.2.1",
+        "define-lazy-prop": "^3.0.0",
+        "is-inside-container": "^1.0.0",
+        "wsl-utils": "^0.1.0"
+      },
+      "engines": {
+        "node": ">=18"
+      },
+      "funding": {
+        "url": "https://github.com/sponsors/sindresorhus"
+      }
+    },
+    "node_modules/optionator": {
+      "version": "0.9.4",
+      "resolved": "https://registry.npmjs.org/optionator/-/optionator-0.9.4.tgz",
+      "integrity": "sha512-6IpQ7mKUxRcZNLIObR0hz7lxsapSSIYNZJwXPGeF0mTVqGKFIXj1DQcMoT22S3ROcLyY/rz0PWaWZ9ayWmad9g==",
+      "dev": true,
+      "license": "MIT",
+      "dependencies": {
+        "deep-is": "^0.1.3",
+        "fast-levenshtein": "^2.0.6",
+        "levn": "^0.4.1",
+        "prelude-ls": "^1.2.1",
+        "type-check": "^0.4.0",
+        "word-wrap": "^1.2.5"
+      },
+      "engines": {
+        "node": ">= 0.8.0"
+      }
+    },
+    "node_modules/p-limit": {
+      "version": "3.1.0",
+      "resolved": "https://registry.npmjs.org/p-limit/-/p-limit-3.1.0.tgz",
+      "integrity": "sha512-TYOanM3wGwNGsZN2cVTYPArw454xnXj5qmWF1bEoAc4+cU/ol7GVh7odevjp1FNHduHc3KZMcFduxU5Xc6uJRQ==",
+      "dev": true,
+      "license": "MIT",
+      "dependencies": {
+        "yocto-queue": "^0.1.0"
+      },
+      "engines": {
+        "node": ">=10"
+      },
+      "funding": {
+        "url": "https://github.com/sponsors/sindresorhus"
+      }
+    },
+    "node_modules/p-locate": {
+      "version": "5.0.0",
+      "resolved": "https://registry.npmjs.org/p-locate/-/p-locate-5.0.0.tgz",
+      "integrity": "sha512-LaNjtRWUBY++zB5nE/NwcaoMylSPk+S+ZHNB1TzdbMJMny6dynpAGt7X/tl/QYq3TIeE6nxHppbo2LGymrG5Pw==",
+      "dev": true,
+      "license": "MIT",
+      "dependencies": {
+        "p-limit": "^3.0.2"
+      },
+      "engines": {
+        "node": ">=10"
+      },
+      "funding": {
+        "url": "https://github.com/sponsors/sindresorhus"
+      }
+    },
+    "node_modules/package-name-regex": {
+      "version": "2.0.6",
+      "resolved": "https://registry.npmjs.org/package-name-regex/-/package-name-regex-2.0.6.tgz",
+      "integrity": "sha512-gFL35q7kbE/zBaPA3UKhp2vSzcPYx2ecbYuwv1ucE9Il6IIgBDweBlH8D68UFGZic2MkllKa2KHCfC1IQBQUYA==",
+      "dev": true,
+      "license": "MIT",
+      "engines": {
+        "node": ">=12"
+      },
+      "funding": {
+        "url": "https://github.com/sponsors/dword-design"
+      }
+    },
+    "node_modules/parent-module": {
+      "version": "1.0.1",
+      "resolved": "https://registry.npmjs.org/parent-module/-/parent-module-1.0.1.tgz",
+      "integrity": "sha512-GQ2EWRpQV8/o+Aw8YqtfZZPfNRWZYkbidE9k5rpl/hC3vtHHBfGm2Ifi6qWV+coDGkrUKZAxE3Lot5kcsRlh+g==",
+      "dev": true,
+      "license": "MIT",
+      "dependencies": {
+        "callsites": "^3.0.0"
+      },
+      "engines": {
+        "node": ">=6"
+      }
+    },
+    "node_modules/path-exists": {
+      "version": "4.0.0",
+      "resolved": "https://registry.npmjs.org/path-exists/-/path-exists-4.0.0.tgz",
+      "integrity": "sha512-ak9Qy5Q7jYb2Wwcey5Fpvg2KoAc/ZIhLSLOSBmRmygPsGwkVVt0fZa0qrtMz+m6tJTAHfZQ8FnmB4MG4LWy7/w==",
+      "dev": true,
+      "license": "MIT",
+      "engines": {
+        "node": ">=8"
+      }
+    },
+    "node_modules/path-key": {
+      "version": "3.1.1",
+      "resolved": "https://registry.npmjs.org/path-key/-/path-key-3.1.1.tgz",
+      "integrity": "sha512-ojmeN0qd+y0jszEtoY48r0Peq5dwMEkIlCOu6Q5f41lfkswXuKtYrhgoTpLnyIcHm24Uhqx+5Tqm2InSwLhE6Q==",
+      "dev": true,
+      "license": "MIT",
+      "engines": {
+        "node": ">=8"
+      }
+    },
+    "node_modules/pathe": {
+      "version": "2.0.3",
+      "resolved": "https://registry.npmjs.org/pathe/-/pathe-2.0.3.tgz",
+      "integrity": "sha512-WUjGcAqP1gQacoQe+OBJsFA7Ld4DyXuUIjZ5cc75cLHvJ7dtNsTugphxIADwspS+AraAUePCKrSVtPLFj/F88w==",
+      "dev": true,
+      "license": "MIT"
+    },
+    "node_modules/perfect-debounce": {
+      "version": "2.1.0",
+      "resolved": "https://registry.npmjs.org/perfect-debounce/-/perfect-debounce-2.1.0.tgz",
+      "integrity": "sha512-LjgdTytVFXeUgtHZr9WYViYSM/g8MkcTPYDlPa3cDqMirHjKiSZPYd6DoL7pK8AJQr+uWkQvCjHNdiMqsrJs+g==",
+      "dev": true,
+      "license": "MIT"
+    },
+    "node_modules/picocolors": {
+      "version": "1.1.1",
+      "resolved": "https://registry.npmjs.org/picocolors/-/picocolors-1.1.1.tgz",
+      "integrity": "sha512-xceH2snhtb5M9liqDsmEw56le376mTZkEX/jEb/RxNFyegNul7eNslCXP9FDj/Lcu0X8KEyMceP2ntpaHrDEVA==",
+      "dev": true,
+      "license": "ISC"
+    },
+    "node_modules/picomatch": {
+      "version": "4.0.3",
+      "resolved": "https://registry.npmjs.org/picomatch/-/picomatch-4.0.3.tgz",
+      "integrity": "sha512-5gTmgEY/sqK6gFXLIsQNH19lWb4ebPDLA4SdLP7dsWkIXHWlG66oPuVvXSGFPppYZz8ZDZq0dYYrbHfBCVUb1Q==",
+      "dev": true,
+      "license": "MIT",
+      "engines": {
+        "node": ">=12"
+      },
+      "funding": {
+        "url": "https://github.com/sponsors/jonschlinkert"
+      }
+    },
+    "node_modules/pkg-types": {
+      "version": "2.3.0",
+      "resolved": "https://registry.npmjs.org/pkg-types/-/pkg-types-2.3.0.tgz",
+      "integrity": "sha512-SIqCzDRg0s9npO5XQ3tNZioRY1uK06lA41ynBC1YmFTmnY6FjUjVt6s4LoADmwoig1qqD0oK8h1p/8mlMx8Oig==",
+      "dev": true,
+      "license": "MIT",
+      "dependencies": {
+        "confbox": "^0.2.2",
+        "exsolve": "^1.0.7",
+        "pathe": "^2.0.3"
+      }
+    },
+    "node_modules/pluralize": {
+      "version": "8.0.0",
+      "resolved": "https://registry.npmjs.org/pluralize/-/pluralize-8.0.0.tgz",
+      "integrity": "sha512-Nc3IT5yHzflTfbjgqWcCPpo7DaKy4FnpB0l/zCAW0Tc7jxAiuqSxHasntB3D7887LSrA93kDJ9IXovxJYxyLCA==",
+      "dev": true,
+      "license": "MIT",
+      "engines": {
+        "node": ">=4"
+      }
+    },
+    "node_modules/postcss": {
+      "version": "8.5.8",
+      "resolved": "https://registry.npmjs.org/postcss/-/postcss-8.5.8.tgz",
+      "integrity": "sha512-OW/rX8O/jXnm82Ey1k44pObPtdblfiuWnrd8X7GJ7emImCOstunGbXUpp7HdBrFQX6rJzn3sPT397Wp5aCwCHg==",
+      "dev": true,
+      "funding": [
+        {
+          "type": "opencollective",
+          "url": "https://opencollective.com/postcss/"
+        },
+        {
+          "type": "tidelift",
+          "url": "https://tidelift.com/funding/github/npm/postcss"
+        },
+        {
+          "type": "github",
+          "url": "https://github.com/sponsors/ai"
+        }
+      ],
+      "license": "MIT",
+      "dependencies": {
+        "nanoid": "^3.3.11",
+        "picocolors": "^1.1.1",
+        "source-map-js": "^1.2.1"
+      },
+      "engines": {
+        "node": "^10 || ^12 || >=14"
+      }
+    },
+    "node_modules/prelude-ls": {
+      "version": "1.2.1",
+      "resolved": "https://registry.npmjs.org/prelude-ls/-/prelude-ls-1.2.1.tgz",
+      "integrity": "sha512-vkcDPrRZo1QZLbn5RLGPpg/WmIQ65qoWWhcGKf/b5eplkkarX0m9z8ppCat4mlOqUsWpyNuYgO3VRyrYHSzX5g==",
+      "dev": true,
+      "license": "MIT",
+      "engines": {
+        "node": ">= 0.8.0"
+      }
+    },
+    "node_modules/prettier": {
+      "version": "3.8.1",
+      "resolved": "https://registry.npmjs.org/prettier/-/prettier-3.8.1.tgz",
+      "integrity": "sha512-UOnG6LftzbdaHZcKoPFtOcCKztrQ57WkHDeRD9t/PTQtmT0NHSeWWepj6pS0z/N7+08BHFDQVUrfmfMRcZwbMg==",
+      "dev": true,
+      "license": "MIT",
+      "bin": {
+        "prettier": "bin/prettier.cjs"
+      },
+      "engines": {
+        "node": ">=14"
+      },
+      "funding": {
+        "url": "https://github.com/prettier/prettier?sponsor=1"
+      }
+    },
+    "node_modules/pretty-bytes": {
+      "version": "7.1.0",
+      "resolved": "https://registry.npmjs.org/pretty-bytes/-/pretty-bytes-7.1.0.tgz",
+      "integrity": "sha512-nODzvTiYVRGRqAOvE84Vk5JDPyyxsVk0/fbA/bq7RqlnhksGpset09XTxbpvLTIjoaF7K8Z8DG8yHtKGTPSYRw==",
+      "dev": true,
+      "license": "MIT",
+      "engines": {
+        "node": ">=20"
+      },
+      "funding": {
+        "url": "https://github.com/sponsors/sindresorhus"
+      }
+    },
+    "node_modules/punycode": {
+      "version": "2.3.1",
+      "resolved": "https://registry.npmjs.org/punycode/-/punycode-2.3.1.tgz",
+      "integrity": "sha512-vYt7UD1U9Wg6138shLtLOvdAu+8DsC/ilFtEVHcH+wydcSpNE20AfSOduf6MkRFahL5FY7X1oU7nKVZFtfq8Fg==",
+      "dev": true,
+      "license": "MIT",
+      "engines": {
+        "node": ">=6"
+      }
+    },
+    "node_modules/rc9": {
+      "version": "2.1.2",
+      "resolved": "https://registry.npmjs.org/rc9/-/rc9-2.1.2.tgz",
+      "integrity": "sha512-btXCnMmRIBINM2LDZoEmOogIZU7Qe7zn4BpomSKZ/ykbLObuBdvG+mFq11DL6fjH1DRwHhrlgtYWG96bJiC7Cg==",
+      "dev": true,
+      "license": "MIT",
+      "dependencies": {
+        "defu": "^6.1.4",
+        "destr": "^2.0.3"
+      }
+    },
+    "node_modules/readdirp": {
+      "version": "5.0.0",
+      "resolved": "https://registry.npmjs.org/readdirp/-/readdirp-5.0.0.tgz",
+      "integrity": "sha512-9u/XQ1pvrQtYyMpZe7DXKv2p5CNvyVwzUB6uhLAnQwHMSgKMBR62lc7AHljaeteeHXn11XTAaLLUVZYVZyuRBQ==",
+      "dev": true,
+      "license": "MIT",
+      "engines": {
+        "node": ">= 20.19.0"
+      },
+      "funding": {
+        "type": "individual",
+        "url": "https://paulmillr.com/funding/"
+      }
+    },
+    "node_modules/regexp-tree": {
+      "version": "0.1.27",
+      "resolved": "https://registry.npmjs.org/regexp-tree/-/regexp-tree-0.1.27.tgz",
+      "integrity": "sha512-iETxpjK6YoRWJG5o6hXLwvjYAoW+FEZn9os0PD/b6AP6xQwsa/Y7lCVgIixBbUPMfhu+i2LtdeAqVTgGlQarfA==",
+      "dev": true,
+      "license": "MIT",
+      "bin": {
+        "regexp-tree": "bin/regexp-tree"
+      }
+    },
+    "node_modules/regjsparser": {
+      "version": "0.13.0",
+      "resolved": "https://registry.npmjs.org/regjsparser/-/regjsparser-0.13.0.tgz",
+      "integrity": "sha512-NZQZdC5wOE/H3UT28fVGL+ikOZcEzfMGk/c3iN9UGxzWHMa1op7274oyiUVrAG4B2EuFhus8SvkaYnhvW92p9Q==",
+      "dev": true,
+      "license": "BSD-2-Clause",
+      "dependencies": {
+        "jsesc": "~3.1.0"
+      },
+      "bin": {
+        "regjsparser": "bin/parser"
+      }
+    },
+    "node_modules/resolve-from": {
+      "version": "4.0.0",
+      "resolved": "https://registry.npmjs.org/resolve-from/-/resolve-from-4.0.0.tgz",
+      "integrity": "sha512-pb/MYmXstAkysRFx8piNI1tGFNQIFA3vkE3Gq4EuA1dF6gHp/+vgZqsCGJapvy8N3Q+4o7FwvquPJcnZ7RYy4g==",
+      "dev": true,
+      "license": "MIT",
+      "engines": {
+        "node": ">=4"
+      }
+    },
+    "node_modules/resolve-pkg-maps": {
+      "version": "1.0.0",
+      "resolved": "https://registry.npmjs.org/resolve-pkg-maps/-/resolve-pkg-maps-1.0.0.tgz",
+      "integrity": "sha512-seS2Tj26TBVOC2NIc2rOe2y2ZO7efxITtLZcGSOnHHNOQ7CkiUBfw0Iw2ck6xkIhPwLhKNLS8BO+hEpngQlqzw==",
+      "dev": true,
+      "license": "MIT",
+      "funding": {
+        "url": "https://github.com/privatenumber/resolve-pkg-maps?sponsor=1"
+      }
+    },
+    "node_modules/rolldown": {
+      "version": "1.0.0-rc.8",
+      "resolved": "https://registry.npmjs.org/rolldown/-/rolldown-1.0.0-rc.8.tgz",
+      "integrity": "sha512-RGOL7mz/aoQpy/y+/XS9iePBfeNRDUdozrhCEJxdpJyimW8v6yp4c30q6OviUU5AnUJVLRL9GP//HUs6N3ALrQ==",
+      "dev": true,
+      "license": "MIT",
+      "dependencies": {
+        "@oxc-project/types": "=0.115.0",
+        "@rolldown/pluginutils": "1.0.0-rc.8"
+      },
+      "bin": {
+        "rolldown": "bin/cli.mjs"
+      },
+      "engines": {
+        "node": "^20.19.0 || >=22.12.0"
+      },
+      "optionalDependencies": {
+        "@rolldown/binding-android-arm64": "1.0.0-rc.8",
+        "@rolldown/binding-darwin-arm64": "1.0.0-rc.8",
+        "@rolldown/binding-darwin-x64": "1.0.0-rc.8",
+        "@rolldown/binding-freebsd-x64": "1.0.0-rc.8",
+        "@rolldown/binding-linux-arm-gnueabihf": "1.0.0-rc.8",
+        "@rolldown/binding-linux-arm64-gnu": "1.0.0-rc.8",
+        "@rolldown/binding-linux-arm64-musl": "1.0.0-rc.8",
+        "@rolldown/binding-linux-ppc64-gnu": "1.0.0-rc.8",
+        "@rolldown/binding-linux-s390x-gnu": "1.0.0-rc.8",
+        "@rolldown/binding-linux-x64-gnu": "1.0.0-rc.8",
+        "@rolldown/binding-linux-x64-musl": "1.0.0-rc.8",
+        "@rolldown/binding-openharmony-arm64": "1.0.0-rc.8",
+        "@rolldown/binding-wasm32-wasi": "1.0.0-rc.8",
+        "@rolldown/binding-win32-arm64-msvc": "1.0.0-rc.8",
+        "@rolldown/binding-win32-x64-msvc": "1.0.0-rc.8"
+      }
+    },
+    "node_modules/rolldown-plugin-dts": {
+      "version": "0.22.4",
+      "resolved": "https://registry.npmjs.org/rolldown-plugin-dts/-/rolldown-plugin-dts-0.22.4.tgz",
+      "integrity": "sha512-pueqTPyN1N6lWYivyDGad+j+GO3DT67pzpct8s8e6KGVIezvnrDjejuw1AXFeyDRas3xTq4Ja6Lj5R5/04C5GQ==",
+      "dev": true,
+      "license": "MIT",
+      "dependencies": {
+        "@babel/generator": "8.0.0-rc.2",
+        "@babel/helper-validator-identifier": "8.0.0-rc.2",
+        "@babel/parser": "8.0.0-rc.2",
+        "@babel/types": "8.0.0-rc.2",
+        "ast-kit": "^3.0.0-beta.1",
+        "birpc": "^4.0.0",
+        "dts-resolver": "^2.1.3",
+        "get-tsconfig": "^4.13.6",
+        "obug": "^2.1.1"
+      },
+      "engines": {
+        "node": ">=20.19.0"
+      },
+      "funding": {
+        "url": "https://github.com/sponsors/sxzz"
+      },
+      "peerDependencies": {
+        "@ts-macro/tsc": "^0.3.6",
+        "@typescript/native-preview": ">=7.0.0-dev.20250601.1",
+        "rolldown": "^1.0.0-rc.3",
+        "typescript": "^5.0.0 || ^6.0.0-beta",
+        "vue-tsc": "~3.2.0"
+      },
+      "peerDependenciesMeta": {
+        "@ts-macro/tsc": {
+          "optional": true
+        },
+        "@typescript/native-preview": {
+          "optional": true
+        },
+        "typescript": {
+          "optional": true
+        },
+        "vue-tsc": {
+          "optional": true
+        }
+      }
+    },
+    "node_modules/rolldown-plugin-dts/node_modules/@babel/helper-string-parser": {
+      "version": "8.0.0-rc.2",
+      "resolved": "https://registry.npmjs.org/@babel/helper-string-parser/-/helper-string-parser-8.0.0-rc.2.tgz",
+      "integrity": "sha512-noLx87RwlBEMrTzncWd/FvTxoJ9+ycHNg0n8yyYydIoDsLZuxknKgWRJUqcrVkNrJ74uGyhWQzQaS3q8xfGAhQ==",
+      "dev": true,
+      "license": "MIT",
+      "engines": {
+        "node": "^20.19.0 || >=22.12.0"
+      }
+    },
+    "node_modules/rolldown-plugin-dts/node_modules/@babel/helper-validator-identifier": {
+      "version": "8.0.0-rc.2",
+      "resolved": "https://registry.npmjs.org/@babel/helper-validator-identifier/-/helper-validator-identifier-8.0.0-rc.2.tgz",
+      "integrity": "sha512-xExUBkuXWJjVuIbO7z6q7/BA9bgfJDEhVL0ggrggLMbg0IzCUWGT1hZGE8qUH7Il7/RD/a6cZ3AAFrrlp1LF/A==",
+      "dev": true,
+      "license": "MIT",
+      "engines": {
+        "node": "^20.19.0 || >=22.12.0"
+      }
+    },
+    "node_modules/rolldown-plugin-dts/node_modules/@babel/parser": {
+      "version": "8.0.0-rc.2",
+      "resolved": "https://registry.npmjs.org/@babel/parser/-/parser-8.0.0-rc.2.tgz",
+      "integrity": "sha512-29AhEtcq4x8Dp3T72qvUMZHx0OMXCj4Jy/TEReQa+KWLln524Cj1fWb3QFi0l/xSpptQBR6y9RNEXuxpFvwiUQ==",
+      "dev": true,
+      "license": "MIT",
+      "dependencies": {
+        "@babel/types": "^8.0.0-rc.2"
+      },
+      "bin": {
+        "parser": "bin/babel-parser.js"
+      },
+      "engines": {
+        "node": "^20.19.0 || >=22.12.0"
+      }
+    },
+    "node_modules/rolldown-plugin-dts/node_modules/@babel/types": {
+      "version": "8.0.0-rc.2",
+      "resolved": "https://registry.npmjs.org/@babel/types/-/types-8.0.0-rc.2.tgz",
+      "integrity": "sha512-91gAaWRznDwSX4E2tZ1YjBuIfnQVOFDCQ2r0Toby0gu4XEbyF623kXLMA8d4ZbCu+fINcrudkmEcwSUHgDDkNw==",
+      "dev": true,
+      "license": "MIT",
+      "dependencies": {
+        "@babel/helper-string-parser": "^8.0.0-rc.2",
+        "@babel/helper-validator-identifier": "^8.0.0-rc.2"
+      },
+      "engines": {
+        "node": "^20.19.0 || >=22.12.0"
+      }
+    },
+    "node_modules/rollup": {
+      "version": "4.59.0",
+      "resolved": "https://registry.npmjs.org/rollup/-/rollup-4.59.0.tgz",
+      "integrity": "sha512-2oMpl67a3zCH9H79LeMcbDhXW/UmWG/y2zuqnF2jQq5uq9TbM9TVyXvA4+t+ne2IIkBdrLpAaRQAvo7YI/Yyeg==",
+      "dev": true,
+      "license": "MIT",
+      "dependencies": {
+        "@types/estree": "1.0.8"
+      },
+      "bin": {
+        "rollup": "dist/bin/rollup"
+      },
+      "engines": {
+        "node": ">=18.0.0",
+        "npm": ">=8.0.0"
+      },
+      "optionalDependencies": {
+        "@rollup/rollup-android-arm-eabi": "4.59.0",
+        "@rollup/rollup-android-arm64": "4.59.0",
+        "@rollup/rollup-darwin-arm64": "4.59.0",
+        "@rollup/rollup-darwin-x64": "4.59.0",
+        "@rollup/rollup-freebsd-arm64": "4.59.0",
+        "@rollup/rollup-freebsd-x64": "4.59.0",
+        "@rollup/rollup-linux-arm-gnueabihf": "4.59.0",
+        "@rollup/rollup-linux-arm-musleabihf": "4.59.0",
+        "@rollup/rollup-linux-arm64-gnu": "4.59.0",
+        "@rollup/rollup-linux-arm64-musl": "4.59.0",
+        "@rollup/rollup-linux-loong64-gnu": "4.59.0",
+        "@rollup/rollup-linux-loong64-musl": "4.59.0",
+        "@rollup/rollup-linux-ppc64-gnu": "4.59.0",
+        "@rollup/rollup-linux-ppc64-musl": "4.59.0",
+        "@rollup/rollup-linux-riscv64-gnu": "4.59.0",
+        "@rollup/rollup-linux-riscv64-musl": "4.59.0",
+        "@rollup/rollup-linux-s390x-gnu": "4.59.0",
+        "@rollup/rollup-linux-x64-gnu": "4.59.0",
+        "@rollup/rollup-linux-x64-musl": "4.59.0",
+        "@rollup/rollup-openbsd-x64": "4.59.0",
+        "@rollup/rollup-openharmony-arm64": "4.59.0",
+        "@rollup/rollup-win32-arm64-msvc": "4.59.0",
+        "@rollup/rollup-win32-ia32-msvc": "4.59.0",
+        "@rollup/rollup-win32-x64-gnu": "4.59.0",
+        "@rollup/rollup-win32-x64-msvc": "4.59.0",
+        "fsevents": "~2.3.2"
+      }
+    },
+    "node_modules/rollup-plugin-license": {
+      "version": "3.7.0",
+      "resolved": "https://registry.npmjs.org/rollup-plugin-license/-/rollup-plugin-license-3.7.0.tgz",
+      "integrity": "sha512-RvvOIF+GH3fBR3wffgc/vmjQn6qOn72WjppWVDp/v+CLpT0BbcRBdSkPeeIOL6U5XccdYgSIMjUyXgxlKEEFcw==",
+      "dev": true,
+      "license": "MIT",
+      "dependencies": {
+        "commenting": "^1.1.0",
+        "fdir": "^6.4.3",
+        "lodash": "^4.17.21",
+        "magic-string": "^0.30.0",
+        "moment": "^2.30.1",
+        "package-name-regex": "^2.0.6",
+        "spdx-expression-validate": "^2.0.0",
+        "spdx-satisfies": "^5.0.1"
+      },
+      "engines": {
+        "node": ">=14.0.0"
+      },
+      "peerDependencies": {
+        "rollup": "^1.0.0 || ^2.0.0 || ^3.0.0 || ^4.0.0"
+      }
+    },
+    "node_modules/run-applescript": {
+      "version": "7.1.0",
+      "resolved": "https://registry.npmjs.org/run-applescript/-/run-applescript-7.1.0.tgz",
+      "integrity": "sha512-DPe5pVFaAsinSaV6QjQ6gdiedWDcRCbUuiQfQa2wmWV7+xC9bGulGI8+TdRmoFkAPaBXk8CrAbnlY2ISniJ47Q==",
+      "dev": true,
+      "license": "MIT",
+      "engines": {
+        "node": ">=18"
+      },
+      "funding": {
+        "url": "https://github.com/sponsors/sindresorhus"
+      }
+    },
+    "node_modules/scule": {
+      "version": "1.3.0",
+      "resolved": "https://registry.npmjs.org/scule/-/scule-1.3.0.tgz",
+      "integrity": "sha512-6FtHJEvt+pVMIB9IBY+IcCJ6Z5f1iQnytgyfKMhDKgmzYG+TeH/wx1y3l27rshSbLiSanrR9ffZDrEsmjlQF2g==",
+      "dev": true,
+      "license": "MIT"
+    },
+    "node_modules/semver": {
+      "version": "7.7.4",
+      "resolved": "https://registry.npmjs.org/semver/-/semver-7.7.4.tgz",
+      "integrity": "sha512-vFKC2IEtQnVhpT78h1Yp8wzwrf8CM+MzKMHGJZfBtzhZNycRFnXsHk6E5TxIkkMsgNS7mdX3AGB7x2QM2di4lA==",
+      "dev": true,
+      "license": "ISC",
+      "bin": {
+        "semver": "bin/semver.js"
+      },
+      "engines": {
+        "node": ">=10"
+      }
+    },
+    "node_modules/shebang-command": {
+      "version": "2.0.0",
+      "resolved": "https://registry.npmjs.org/shebang-command/-/shebang-command-2.0.0.tgz",
+      "integrity": "sha512-kHxr2zZpYtdmrN1qDjrrX/Z1rR1kG8Dx+gkpK1G4eXmvXswmcE1hTWBWYUzlraYw1/yZp6YuDY77YtvbN0dmDA==",
+      "dev": true,
+      "license": "MIT",
+      "dependencies": {
+        "shebang-regex": "^3.0.0"
+      },
+      "engines": {
+        "node": ">=8"
+      }
+    },
+    "node_modules/shebang-regex": {
+      "version": "3.0.0",
+      "resolved": "https://registry.npmjs.org/shebang-regex/-/shebang-regex-3.0.0.tgz",
+      "integrity": "sha512-7++dFhtcx3353uBaq8DDR4NuxBetBzC7ZQOhmTQInHEd6bSrXdiEyzCvG07Z44UYdLShWUyXt5M/yhz8ekcb1A==",
+      "dev": true,
+      "license": "MIT",
+      "engines": {
+        "node": ">=8"
+      }
+    },
+    "node_modules/siginfo": {
+      "version": "2.0.0",
+      "resolved": "https://registry.npmjs.org/siginfo/-/siginfo-2.0.0.tgz",
+      "integrity": "sha512-ybx0WO1/8bSBLEWXZvEd7gMW3Sn3JFlW3TvX1nREbDLRNQNaeNN8WK0meBwPdAaOI7TtRRRJn/Es1zhrrCHu7g==",
+      "dev": true,
+      "license": "ISC"
+    },
+    "node_modules/source-map-js": {
+      "version": "1.2.1",
+      "resolved": "https://registry.npmjs.org/source-map-js/-/source-map-js-1.2.1.tgz",
+      "integrity": "sha512-UXWMKhLOwVKb728IUtQPXxfYU+usdybtUrK/8uGE8CQMvrhOpwvzDBwj0QhSL7MQc7vIsISBG8VQ8+IDQxpfQA==",
+      "dev": true,
+      "license": "BSD-3-Clause",
+      "engines": {
+        "node": ">=0.10.0"
+      }
+    },
+    "node_modules/spdx-compare": {
+      "version": "1.0.0",
+      "resolved": "https://registry.npmjs.org/spdx-compare/-/spdx-compare-1.0.0.tgz",
+      "integrity": "sha512-C1mDZOX0hnu0ep9dfmuoi03+eOdDoz2yvK79RxbcrVEG1NO1Ph35yW102DHWKN4pk80nwCgeMmSY5L25VE4D9A==",
+      "dev": true,
+      "license": "MIT",
+      "dependencies": {
+        "array-find-index": "^1.0.2",
+        "spdx-expression-parse": "^3.0.0",
+        "spdx-ranges": "^2.0.0"
+      }
+    },
+    "node_modules/spdx-exceptions": {
+      "version": "2.5.0",
+      "resolved": "https://registry.npmjs.org/spdx-exceptions/-/spdx-exceptions-2.5.0.tgz",
+      "integrity": "sha512-PiU42r+xO4UbUS1buo3LPJkjlO7430Xn5SVAhdpzzsPHsjbYVflnnFdATgabnLude+Cqu25p6N+g2lw/PFsa4w==",
+      "dev": true,
+      "license": "CC-BY-3.0"
+    },
+    "node_modules/spdx-expression-parse": {
+      "version": "3.0.1",
+      "resolved": "https://registry.npmjs.org/spdx-expression-parse/-/spdx-expression-parse-3.0.1.tgz",
+      "integrity": "sha512-cbqHunsQWnJNE6KhVSMsMeH5H/L9EpymbzqTQ3uLwNCLZ1Q481oWaofqH7nO6V07xlXwY6PhQdQ2IedWx/ZK4Q==",
+      "dev": true,
+      "license": "MIT",
+      "dependencies": {
+        "spdx-exceptions": "^2.1.0",
+        "spdx-license-ids": "^3.0.0"
+      }
+    },
+    "node_modules/spdx-expression-validate": {
+      "version": "2.0.0",
+      "resolved": "https://registry.npmjs.org/spdx-expression-validate/-/spdx-expression-validate-2.0.0.tgz",
+      "integrity": "sha512-b3wydZLM+Tc6CFvaRDBOF9d76oGIHNCLYFeHbftFXUWjnfZWganmDmvtM5sm1cRwJc/VDBMLyGGrsLFd1vOxbg==",
+      "dev": true,
+      "license": "(MIT AND CC-BY-3.0)",
+      "dependencies": {
+        "spdx-expression-parse": "^3.0.0"
+      }
+    },
+    "node_modules/spdx-license-ids": {
+      "version": "3.0.23",
+      "resolved": "https://registry.npmjs.org/spdx-license-ids/-/spdx-license-ids-3.0.23.tgz",
+      "integrity": "sha512-CWLcCCH7VLu13TgOH+r8p1O/Znwhqv/dbb6lqWy67G+pT1kHmeD/+V36AVb/vq8QMIQwVShJ6Ssl5FPh0fuSdw==",
+      "dev": true,
+      "license": "CC0-1.0"
+    },
+    "node_modules/spdx-ranges": {
+      "version": "2.1.1",
+      "resolved": "https://registry.npmjs.org/spdx-ranges/-/spdx-ranges-2.1.1.tgz",
+      "integrity": "sha512-mcdpQFV7UDAgLpXEE/jOMqvK4LBoO0uTQg0uvXUewmEFhpiZx5yJSZITHB8w1ZahKdhfZqP5GPEOKLyEq5p8XA==",
+      "dev": true,
+      "license": "(MIT AND CC-BY-3.0)"
+    },
+    "node_modules/spdx-satisfies": {
+      "version": "5.0.1",
+      "resolved": "https://registry.npmjs.org/spdx-satisfies/-/spdx-satisfies-5.0.1.tgz",
+      "integrity": "sha512-Nwor6W6gzFp8XX4neaKQ7ChV4wmpSh2sSDemMFSzHxpTw460jxFYeOn+jq4ybnSSw/5sc3pjka9MQPouksQNpw==",
+      "dev": true,
+      "license": "MIT",
+      "dependencies": {
+        "spdx-compare": "^1.0.0",
+        "spdx-expression-parse": "^3.0.0",
+        "spdx-ranges": "^2.0.0"
+      }
+    },
+    "node_modules/stackback": {
+      "version": "0.0.2",
+      "resolved": "https://registry.npmjs.org/stackback/-/stackback-0.0.2.tgz",
+      "integrity": "sha512-1XMJE5fQo1jGH6Y/7ebnwPOBEkIEnT4QF32d5R1+VXdXveM0IBMJt8zfaxX1P3QhVwrYe+576+jkANtSS2mBbw==",
+      "dev": true,
+      "license": "MIT"
+    },
+    "node_modules/std-env": {
+      "version": "3.10.0",
+      "resolved": "https://registry.npmjs.org/std-env/-/std-env-3.10.0.tgz",
+      "integrity": "sha512-5GS12FdOZNliM5mAOxFRg7Ir0pWz8MdpYm6AY6VPkGpbA7ZzmbzNcBJQ0GPvvyWgcY7QAhCgf9Uy89I03faLkg==",
+      "dev": true,
+      "license": "MIT"
+    },
+    "node_modules/strip-indent": {
+      "version": "4.1.1",
+      "resolved": "https://registry.npmjs.org/strip-indent/-/strip-indent-4.1.1.tgz",
+      "integrity": "sha512-SlyRoSkdh1dYP0PzclLE7r0M9sgbFKKMFXpFRUMNuKhQSbC6VQIGzq3E0qsfvGJaUFJPGv6Ws1NZ/haTAjfbMA==",
+      "dev": true,
+      "license": "MIT",
+      "engines": {
+        "node": ">=12"
+      },
+      "funding": {
+        "url": "https://github.com/sponsors/sindresorhus"
+      }
+    },
+    "node_modules/strip-json-comments": {
+      "version": "3.1.1",
+      "resolved": "https://registry.npmjs.org/strip-json-comments/-/strip-json-comments-3.1.1.tgz",
+      "integrity": "sha512-6fPc+R4ihwqP6N/aIv2f1gMH8lOVtWQHoqC4yK6oSDVVocumAsfCqjkXnqiYMhmMwS/mEHLp7Vehlt3ql6lEig==",
+      "dev": true,
+      "license": "MIT",
+      "engines": {
+        "node": ">=8"
+      },
+      "funding": {
+        "url": "https://github.com/sponsors/sindresorhus"
+      }
+    },
+    "node_modules/supports-color": {
+      "version": "7.2.0",
+      "resolved": "https://registry.npmjs.org/supports-color/-/supports-color-7.2.0.tgz",
+      "integrity": "sha512-qpCAvRl9stuOHveKsn7HncJRvv501qIacKzQlO/+Lwxc9+0q2wLyv4Dfvt80/DPn2pqOBsJdDiogXGR9+OvwRw==",
+      "dev": true,
+      "license": "MIT",
+      "dependencies": {
+        "has-flag": "^4.0.0"
+      },
+      "engines": {
+        "node": ">=8"
+      }
+    },
+    "node_modules/tinybench": {
+      "version": "2.9.0",
+      "resolved": "https://registry.npmjs.org/tinybench/-/tinybench-2.9.0.tgz",
+      "integrity": "sha512-0+DUvqWMValLmha6lr4kD8iAMK1HzV0/aKnCtWb9v9641TnP/MFb7Pc2bxoxQjTXAErryXVgUOfv2YqNllqGeg==",
+      "dev": true,
+      "license": "MIT"
+    },
+    "node_modules/tinyexec": {
+      "version": "1.0.2",
+      "resolved": "https://registry.npmjs.org/tinyexec/-/tinyexec-1.0.2.tgz",
+      "integrity": "sha512-W/KYk+NFhkmsYpuHq5JykngiOCnxeVL8v8dFnqxSD8qEEdRfXk1SDM6JzNqcERbcGYj9tMrDQBYV9cjgnunFIg==",
+      "dev": true,
+      "license": "MIT",
+      "engines": {
+        "node": ">=18"
+      }
+    },
+    "node_modules/tinyglobby": {
+      "version": "0.2.15",
+      "resolved": "https://registry.npmjs.org/tinyglobby/-/tinyglobby-0.2.15.tgz",
+      "integrity": "sha512-j2Zq4NyQYG5XMST4cbs02Ak8iJUdxRM0XI5QyxXuZOzKOINmWurp3smXu3y5wDcJrptwpSjgXHzIQxR0omXljQ==",
+      "dev": true,
+      "license": "MIT",
+      "dependencies": {
+        "fdir": "^6.5.0",
+        "picomatch": "^4.0.3"
+      },
+      "engines": {
+        "node": ">=12.0.0"
+      },
+      "funding": {
+        "url": "https://github.com/sponsors/SuperchupuDev"
+      }
+    },
+    "node_modules/tinyrainbow": {
+      "version": "3.0.3",
+      "resolved": "https://registry.npmjs.org/tinyrainbow/-/tinyrainbow-3.0.3.tgz",
+      "integrity": "sha512-PSkbLUoxOFRzJYjjxHJt9xro7D+iilgMX/C9lawzVuYiIdcihh9DXmVibBe8lmcFrRi/VzlPjBxbN7rH24q8/Q==",
+      "dev": true,
+      "license": "MIT",
+      "engines": {
+        "node": ">=14.0.0"
+      }
+    },
+    "node_modules/ts-api-utils": {
+      "version": "2.4.0",
+      "resolved": "https://registry.npmjs.org/ts-api-utils/-/ts-api-utils-2.4.0.tgz",
+      "integrity": "sha512-3TaVTaAv2gTiMB35i3FiGJaRfwb3Pyn/j3m/bfAvGe8FB7CF6u+LMYqYlDh7reQf7UNvoTvdfAqHGmPGOSsPmA==",
+      "dev": true,
+      "license": "MIT",
+      "engines": {
+        "node": ">=18.12"
+      },
+      "peerDependencies": {
+        "typescript": ">=4.8.4"
+      }
+    },
+    "node_modules/tslib": {
+      "version": "2.8.1",
+      "resolved": "https://registry.npmjs.org/tslib/-/tslib-2.8.1.tgz",
+      "integrity": "sha512-oJFu94HQb+KVduSUQL7wnpmqnfmLsOA/nAh6b6EH0wCEoK0/mPeXU6c3wKDV83MkOuHPRHtSXKKU99IBazS/2w==",
+      "dev": true,
+      "license": "0BSD",
+      "optional": true
+    },
+    "node_modules/type-check": {
+      "version": "0.4.0",
+      "resolved": "https://registry.npmjs.org/type-check/-/type-check-0.4.0.tgz",
+      "integrity": "sha512-XleUoc9uwGXqjWwXaUTZAmzMcFZ5858QA2vvx1Ur5xIcixXIP+8LnFDgRplU30us6teqdlskFfu+ae4K79Ooew==",
+      "dev": true,
+      "license": "MIT",
+      "dependencies": {
+        "prelude-ls": "^1.2.1"
+      },
+      "engines": {
+        "node": ">= 0.8.0"
+      }
+    },
+    "node_modules/typescript": {
+      "version": "5.9.3",
+      "resolved": "https://registry.npmjs.org/typescript/-/typescript-5.9.3.tgz",
+      "integrity": "sha512-jl1vZzPDinLr9eUt3J/t7V6FgNEw9QjvBPdysz9KfQDD41fQrC2Y4vKQdiaUpFT4bXlb1RHhLpp8wtm6M5TgSw==",
+      "dev": true,
+      "license": "Apache-2.0",
+      "bin": {
+        "tsc": "bin/tsc",
+        "tsserver": "bin/tsserver"
+      },
+      "engines": {
+        "node": ">=14.17"
+      }
+    },
+    "node_modules/typescript-eslint": {
+      "version": "8.57.0",
+      "resolved": "https://registry.npmjs.org/typescript-eslint/-/typescript-eslint-8.57.0.tgz",
+      "integrity": "sha512-W8GcigEMEeB07xEZol8oJ26rigm3+bfPHxHvwbYUlu1fUDsGuQ7Hiskx5xGW/xM4USc9Ephe3jtv7ZYPQntHeA==",
+      "dev": true,
+      "license": "MIT",
+      "dependencies": {
+        "@typescript-eslint/eslint-plugin": "8.57.0",
+        "@typescript-eslint/parser": "8.57.0",
+        "@typescript-eslint/typescript-estree": "8.57.0",
+        "@typescript-eslint/utils": "8.57.0"
+      },
+      "engines": {
+        "node": "^18.18.0 || ^20.9.0 || >=21.1.0"
+      },
+      "funding": {
+        "type": "opencollective",
+        "url": "https://opencollective.com/typescript-eslint"
+      },
+      "peerDependencies": {
+        "eslint": "^8.57.0 || ^9.0.0 || ^10.0.0",
+        "typescript": ">=4.8.4 <6.0.0"
+      }
+    },
+    "node_modules/ufo": {
+      "version": "1.6.3",
+      "resolved": "https://registry.npmjs.org/ufo/-/ufo-1.6.3.tgz",
+      "integrity": "sha512-yDJTmhydvl5lJzBmy/hyOAA0d+aqCBuwl818haVdYCRrWV84o7YyeVm4QlVHStqNrrJSTb6jKuFAVqAFsr+K3Q==",
+      "dev": true,
+      "license": "MIT"
+    },
+    "node_modules/undici-types": {
+      "version": "7.18.2",
+      "resolved": "https://registry.npmjs.org/undici-types/-/undici-types-7.18.2.tgz",
+      "integrity": "sha512-AsuCzffGHJybSaRrmr5eHr81mwJU3kjw6M+uprWvCXiNeN9SOGwQ3Jn8jb8m3Z6izVgknn1R0FTCEAP2QrLY/w==",
+      "dev": true,
+      "license": "MIT"
+    },
+    "node_modules/unist-util-is": {
+      "version": "6.0.1",
+      "resolved": "https://registry.npmjs.org/unist-util-is/-/unist-util-is-6.0.1.tgz",
+      "integrity": "sha512-LsiILbtBETkDz8I9p1dQ0uyRUWuaQzd/cuEeS1hoRSyW5E5XGmTzlwY1OrNzzakGowI9Dr/I8HVaw4hTtnxy8g==",
+      "dev": true,
+      "license": "MIT",
+      "dependencies": {
+        "@types/unist": "^3.0.0"
+      },
+      "funding": {
+        "type": "opencollective",
+        "url": "https://opencollective.com/unified"
+      }
+    },
+    "node_modules/unist-util-stringify-position": {
+      "version": "4.0.0",
+      "resolved": "https://registry.npmjs.org/unist-util-stringify-position/-/unist-util-stringify-position-4.0.0.tgz",
+      "integrity": "sha512-0ASV06AAoKCDkS2+xw5RXJywruurpbC4JZSm7nr7MOt1ojAzvyyaO+UxZf18j8FCF6kmzCZKcAgN/yu2gm2XgQ==",
+      "dev": true,
+      "license": "MIT",
+      "dependencies": {
+        "@types/unist": "^3.0.0"
+      },
+      "funding": {
+        "type": "opencollective",
+        "url": "https://opencollective.com/unified"
+      }
+    },
+    "node_modules/unist-util-visit": {
+      "version": "5.1.0",
+      "resolved": "https://registry.npmjs.org/unist-util-visit/-/unist-util-visit-5.1.0.tgz",
+      "integrity": "sha512-m+vIdyeCOpdr/QeQCu2EzxX/ohgS8KbnPDgFni4dQsfSCtpz8UqDyY5GjRru8PDKuYn7Fq19j1CQ+nJSsGKOzg==",
+      "dev": true,
+      "license": "MIT",
+      "dependencies": {
+        "@types/unist": "^3.0.0",
+        "unist-util-is": "^6.0.0",
+        "unist-util-visit-parents": "^6.0.0"
+      },
+      "funding": {
+        "type": "opencollective",
+        "url": "https://opencollective.com/unified"
+      }
+    },
+    "node_modules/unist-util-visit-parents": {
+      "version": "6.0.2",
+      "resolved": "https://registry.npmjs.org/unist-util-visit-parents/-/unist-util-visit-parents-6.0.2.tgz",
+      "integrity": "sha512-goh1s1TBrqSqukSc8wrjwWhL0hiJxgA8m4kFxGlQ+8FYQ3C/m11FcTs4YYem7V664AhHVvgoQLk890Ssdsr2IQ==",
+      "dev": true,
+      "license": "MIT",
+      "dependencies": {
+        "@types/unist": "^3.0.0",
+        "unist-util-is": "^6.0.0"
+      },
+      "funding": {
+        "type": "opencollective",
+        "url": "https://opencollective.com/unified"
+      }
+    },
+    "node_modules/untyped": {
+      "version": "2.0.0",
+      "resolved": "https://registry.npmjs.org/untyped/-/untyped-2.0.0.tgz",
+      "integrity": "sha512-nwNCjxJTjNuLCgFr42fEak5OcLuB3ecca+9ksPFNvtfYSLpjf+iJqSIaSnIile6ZPbKYxI5k2AfXqeopGudK/g==",
+      "dev": true,
+      "license": "MIT",
+      "dependencies": {
+        "citty": "^0.1.6",
+        "defu": "^6.1.4",
+        "jiti": "^2.4.2",
+        "knitwork": "^1.2.0",
+        "scule": "^1.3.0"
+      },
+      "bin": {
+        "untyped": "dist/cli.mjs"
+      }
+    },
+    "node_modules/untyped/node_modules/citty": {
+      "version": "0.1.6",
+      "resolved": "https://registry.npmjs.org/citty/-/citty-0.1.6.tgz",
+      "integrity": "sha512-tskPPKEs8D2KPafUypv2gxwJP8h/OaJmC82QQGGDQcHvXX43xF2VDACcJVmZ0EuSxkpO9Kc4MlrA3q0+FG58AQ==",
+      "dev": true,
+      "license": "MIT",
+      "dependencies": {
+        "consola": "^3.2.3"
+      }
+    },
+    "node_modules/update-browserslist-db": {
+      "version": "1.2.3",
+      "resolved": "https://registry.npmjs.org/update-browserslist-db/-/update-browserslist-db-1.2.3.tgz",
+      "integrity": "sha512-Js0m9cx+qOgDxo0eMiFGEueWztz+d4+M3rGlmKPT+T4IS/jP4ylw3Nwpu6cpTTP8R1MAC1kF4VbdLt3ARf209w==",
+      "dev": true,
+      "funding": [
+        {
+          "type": "opencollective",
+          "url": "https://opencollective.com/browserslist"
+        },
+        {
+          "type": "tidelift",
+          "url": "https://tidelift.com/funding/github/npm/browserslist"
+        },
+        {
+          "type": "github",
+          "url": "https://github.com/sponsors/ai"
+        }
+      ],
+      "license": "MIT",
+      "dependencies": {
+        "escalade": "^3.2.0",
+        "picocolors": "^1.1.1"
+      },
+      "bin": {
+        "update-browserslist-db": "cli.js"
+      },
+      "peerDependencies": {
+        "browserslist": ">= 4.21.0"
+      }
+    },
+    "node_modules/uri-js": {
+      "version": "4.4.1",
+      "resolved": "https://registry.npmjs.org/uri-js/-/uri-js-4.4.1.tgz",
+      "integrity": "sha512-7rKUyy33Q1yc98pQ1DAmLtwX109F7TIfWlW1Ydo8Wl1ii1SeHieeh0HHfPeL2fMXK6z0s8ecKs9frCuLJvndBg==",
+      "dev": true,
+      "license": "BSD-2-Clause",
+      "dependencies": {
+        "punycode": "^2.1.0"
+      }
+    },
+    "node_modules/vite": {
+      "version": "7.3.1",
+      "resolved": "https://registry.npmjs.org/vite/-/vite-7.3.1.tgz",
+      "integrity": "sha512-w+N7Hifpc3gRjZ63vYBXA56dvvRlNWRczTdmCBBa+CotUzAPf5b7YMdMR/8CQoeYE5LX3W4wj6RYTgonm1b9DA==",
+      "dev": true,
+      "license": "MIT",
+      "dependencies": {
+        "esbuild": "^0.27.0",
+        "fdir": "^6.5.0",
+        "picomatch": "^4.0.3",
+        "postcss": "^8.5.6",
+        "rollup": "^4.43.0",
+        "tinyglobby": "^0.2.15"
+      },
+      "bin": {
+        "vite": "bin/vite.js"
+      },
+      "engines": {
+        "node": "^20.19.0 || >=22.12.0"
+      },
+      "funding": {
+        "url": "https://github.com/vitejs/vite?sponsor=1"
+      },
+      "optionalDependencies": {
+        "fsevents": "~2.3.3"
+      },
+      "peerDependencies": {
+        "@types/node": "^20.19.0 || >=22.12.0",
+        "jiti": ">=1.21.0",
+        "less": "^4.0.0",
+        "lightningcss": "^1.21.0",
+        "sass": "^1.70.0",
+        "sass-embedded": "^1.70.0",
+        "stylus": ">=0.54.8",
+        "sugarss": "^5.0.0",
+        "terser": "^5.16.0",
+        "tsx": "^4.8.1",
+        "yaml": "^2.4.2"
+      },
+      "peerDependenciesMeta": {
+        "@types/node": {
+          "optional": true
+        },
+        "jiti": {
+          "optional": true
+        },
+        "less": {
+          "optional": true
+        },
+        "lightningcss": {
+          "optional": true
+        },
+        "sass": {
+          "optional": true
+        },
+        "sass-embedded": {
+          "optional": true
+        },
+        "stylus": {
+          "optional": true
+        },
+        "sugarss": {
+          "optional": true
+        },
+        "terser": {
+          "optional": true
+        },
+        "tsx": {
+          "optional": true
+        },
+        "yaml": {
+          "optional": true
+        }
+      }
+    },
+    "node_modules/vitest": {
+      "version": "4.0.18",
+      "resolved": "https://registry.npmjs.org/vitest/-/vitest-4.0.18.tgz",
+      "integrity": "sha512-hOQuK7h0FGKgBAas7v0mSAsnvrIgAvWmRFjmzpJ7SwFHH3g1k2u37JtYwOwmEKhK6ZO3v9ggDBBm0La1LCK4uQ==",
+      "dev": true,
+      "license": "MIT",
+      "dependencies": {
+        "@vitest/expect": "4.0.18",
+        "@vitest/mocker": "4.0.18",
+        "@vitest/pretty-format": "4.0.18",
+        "@vitest/runner": "4.0.18",
+        "@vitest/snapshot": "4.0.18",
+        "@vitest/spy": "4.0.18",
+        "@vitest/utils": "4.0.18",
+        "es-module-lexer": "^1.7.0",
+        "expect-type": "^1.2.2",
+        "magic-string": "^0.30.21",
+        "obug": "^2.1.1",
+        "pathe": "^2.0.3",
+        "picomatch": "^4.0.3",
+        "std-env": "^3.10.0",
+        "tinybench": "^2.9.0",
+        "tinyexec": "^1.0.2",
+        "tinyglobby": "^0.2.15",
+        "tinyrainbow": "^3.0.3",
+        "vite": "^6.0.0 || ^7.0.0",
+        "why-is-node-running": "^2.3.0"
+      },
+      "bin": {
+        "vitest": "vitest.mjs"
+      },
+      "engines": {
+        "node": "^20.0.0 || ^22.0.0 || >=24.0.0"
+      },
+      "funding": {
+        "url": "https://opencollective.com/vitest"
+      },
+      "peerDependencies": {
+        "@edge-runtime/vm": "*",
+        "@opentelemetry/api": "^1.9.0",
+        "@types/node": "^20.0.0 || ^22.0.0 || >=24.0.0",
+        "@vitest/browser-playwright": "4.0.18",
+        "@vitest/browser-preview": "4.0.18",
+        "@vitest/browser-webdriverio": "4.0.18",
+        "@vitest/ui": "4.0.18",
+        "happy-dom": "*",
+        "jsdom": "*"
+      },
+      "peerDependenciesMeta": {
+        "@edge-runtime/vm": {
+          "optional": true
+        },
+        "@opentelemetry/api": {
+          "optional": true
+        },
+        "@types/node": {
+          "optional": true
+        },
+        "@vitest/browser-playwright": {
+          "optional": true
+        },
+        "@vitest/browser-preview": {
+          "optional": true
+        },
+        "@vitest/browser-webdriverio": {
+          "optional": true
+        },
+        "@vitest/ui": {
+          "optional": true
+        },
+        "happy-dom": {
+          "optional": true
+        },
+        "jsdom": {
+          "optional": true
+        }
+      }
+    },
+    "node_modules/which": {
+      "version": "2.0.2",
+      "resolved": "https://registry.npmjs.org/which/-/which-2.0.2.tgz",
+      "integrity": "sha512-BLI3Tl1TW3Pvl70l3yq3Y64i+awpwXqsGBYWkkqMtnbXgrMD+yj7rhW0kuEDxzJaYXGjEW5ogapKNMEKNMjibA==",
+      "dev": true,
+      "license": "ISC",
+      "dependencies": {
+        "isexe": "^2.0.0"
+      },
+      "bin": {
+        "node-which": "bin/node-which"
+      },
+      "engines": {
+        "node": ">= 8"
+      }
+    },
+    "node_modules/why-is-node-running": {
+      "version": "2.3.0",
+      "resolved": "https://registry.npmjs.org/why-is-node-running/-/why-is-node-running-2.3.0.tgz",
+      "integrity": "sha512-hUrmaWBdVDcxvYqnyh09zunKzROWjbZTiNy8dBEjkS7ehEDQibXJ7XvlmtbwuTclUiIyN+CyXQD4Vmko8fNm8w==",
+      "dev": true,
+      "license": "MIT",
+      "dependencies": {
+        "siginfo": "^2.0.0",
+        "stackback": "0.0.2"
+      },
+      "bin": {
+        "why-is-node-running": "cli.js"
+      },
+      "engines": {
+        "node": ">=8"
+      }
+    },
+    "node_modules/word-wrap": {
+      "version": "1.2.5",
+      "resolved": "https://registry.npmjs.org/word-wrap/-/word-wrap-1.2.5.tgz",
+      "integrity": "sha512-BN22B5eaMMI9UMtjrGd5g5eCYPpCPDUy0FJXbYsaT5zYxjFOckS53SQDE3pWkVoWpHXVb3BrYcEN4Twa55B5cA==",
+      "dev": true,
+      "license": "MIT",
+      "engines": {
+        "node": ">=0.10.0"
+      }
+    },
+    "node_modules/wsl-utils": {
+      "version": "0.1.0",
+      "resolved": "https://registry.npmjs.org/wsl-utils/-/wsl-utils-0.1.0.tgz",
+      "integrity": "sha512-h3Fbisa2nKGPxCpm89Hk33lBLsnaGBvctQopaBSOW/uIs6FTe1ATyAnKFJrzVs9vpGdsTe73WF3V4lIsk4Gacw==",
+      "dev": true,
+      "license": "MIT",
+      "dependencies": {
+        "is-wsl": "^3.1.0"
+      },
+      "engines": {
+        "node": ">=18"
+      },
+      "funding": {
+        "url": "https://github.com/sponsors/sindresorhus"
+      }
+    },
+    "node_modules/yocto-queue": {
+      "version": "0.1.0",
+      "resolved": "https://registry.npmjs.org/yocto-queue/-/yocto-queue-0.1.0.tgz",
+      "integrity": "sha512-rVksvsnNCdJ/ohGc6xgPwyN8eheCxsiLM8mxuE/t/mOVqJewPuO1miLpTHQiRgTKCLexL4MeAFVagts7HmNZ2Q==",
+      "dev": true,
+      "license": "MIT",
+      "engines": {
+        "node": ">=10"
+      },
+      "funding": {
+        "url": "https://github.com/sponsors/sindresorhus"
+      }
+    },
+    "node_modules/zwitch": {
+      "version": "2.0.4",
+      "resolved": "https://registry.npmjs.org/zwitch/-/zwitch-2.0.4.tgz",
+      "integrity": "sha512-bXE4cR/kVZhKZX/RjPEflHaKVhUVl85noU3v6b8apfQEc1x4A+zBxjZ4lN8LqGd6WZ3dl98pY4o717VFmoPp+A==",
+      "dev": true,
+      "license": "MIT",
+      "funding": {
+        "type": "github",
+        "url": "https://github.com/sponsors/wooorm"
+      }
+    }
+  }
+}
diff --git a/src/_parser.ts b/src/_parser.ts
index 5483e44..d9244c2 100644
--- a/src/_parser.ts
+++ b/src/_parser.ts
@@ -142,9 +142,20 @@ export function parseRawArgs<T = Record<string, any>>(
     (out as any)[key] = value;
   }
 
-  // Apply negated flags
+  // Apply negated flags and propagate to main option and all aliases
   for (const [name] of Object.entries(negatedFlags)) {
     (out as any)[name] = false;
+    // If name is an alias, also set the main option
+    const main = aliasToMain.get(name);
+    if (main) {
+      (out as any)[main] = false;
+    }
+    // Set all aliases of either this name or its main option to false
+    const mainName = main || name;
+    const aliases = mainToAliases.get(mainName) || [];
+    for (const alias of aliases) {
+      (out as any)[alias] = false;
+    }
   }
 
   // Propagate values between aliases
```


