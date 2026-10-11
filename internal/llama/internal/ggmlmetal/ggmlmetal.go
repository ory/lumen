// Copyright 2026 Aeneas Rekkas
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

//go:build darwin && arm64

// Package ggmlmetal compiles the vendored ggml Metal backend with its kernel
// sources embedded in the binary (see metallib_darwin_arm64.S).
package ggmlmetal

// #cgo CPPFLAGS: -I${SRCDIR}/../../csrc/ggml/include -I${SRCDIR}/../../csrc/ggml/src
// #cgo CPPFLAGS: -I${SRCDIR}/../../csrc/ggml/src/ggml-metal
// #cgo CPPFLAGS: -DNDEBUG -DGGML_SCHED_MAX_COPIES=4 -D_XOPEN_SOURCE=600 -D_DARWIN_C_SOURCE -DGGML_METAL_EMBED_LIBRARY
// #cgo CFLAGS: -std=gnu11 -O3 -w
// #cgo CXXFLAGS: -std=gnu++17 -O3 -w
// #cgo LDFLAGS: -framework Foundation -framework Metal -framework MetalKit
import "C"
