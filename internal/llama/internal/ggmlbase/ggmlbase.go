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

// Package ggmlbase compiles the vendored ggml core, GGUF reader, and static
// backend registry. It exports no Go API; internal/llama links it in.
package ggmlbase

// #cgo CPPFLAGS: -I${SRCDIR}/../../csrc/ggml/include -I${SRCDIR}/../../csrc/ggml/src
// #cgo CPPFLAGS: -DNDEBUG -DGGML_SCHED_MAX_COPIES=4 -D_XOPEN_SOURCE=600 -DGGML_USE_CPU
// #cgo darwin CPPFLAGS: -D_DARWIN_C_SOURCE
// #cgo darwin,arm64 CPPFLAGS: -DGGML_USE_METAL
// #cgo linux CPPFLAGS: -D_GNU_SOURCE
// #cgo windows CPPFLAGS: -D_CRT_SECURE_NO_WARNINGS
// #cgo CFLAGS: -std=gnu11 -O3 -w
// #cgo CXXFLAGS: -std=gnu++17 -O3 -w
// #cgo linux LDFLAGS: -lm -ldl -lpthread -static-libstdc++ -static-libgcc
// #cgo windows LDFLAGS: -static
import "C"
