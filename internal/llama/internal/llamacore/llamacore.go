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

// Package llamacore compiles the vendored llama.cpp runtime (model loading,
// tokenization, contexts, and graph execution).
package llamacore

// #cgo CPPFLAGS: -I${SRCDIR}/../../csrc/include -I${SRCDIR}/../../csrc/src -I${SRCDIR}/../../csrc/ggml/include
// #cgo CPPFLAGS: -DNDEBUG
// #cgo windows CPPFLAGS: -D_CRT_SECURE_NO_WARNINGS
// #cgo CXXFLAGS: -std=gnu++17 -O3 -w
import "C"
