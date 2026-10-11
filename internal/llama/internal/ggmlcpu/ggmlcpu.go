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

// Package ggmlcpu compiles the vendored ggml CPU backend.
//
// x86-64 kernels target x86-64-v3 (AVX2, FMA, F16C, BMI2). internal/llama
// checks those CPU features before calling into native code so older CPUs get
// an error instead of an illegal instruction. arm64 uses the platform baseline.
package ggmlcpu

// #cgo CPPFLAGS: -I${SRCDIR}/../../csrc/ggml/include -I${SRCDIR}/../../csrc/ggml/src
// #cgo CPPFLAGS: -I${SRCDIR}/../../csrc/ggml/src/ggml-cpu -I${SRCDIR}/../../csrc/ggml
// #cgo CPPFLAGS: -DNDEBUG -DGGML_SCHED_MAX_COPIES=4 -D_XOPEN_SOURCE=600
// #cgo CPPFLAGS: -DGGML_USE_CPU_REPACK -DGGML_USE_LLAMAFILE
// #cgo darwin CPPFLAGS: -D_DARWIN_C_SOURCE -DGGML_USE_ACCELERATE -DACCELERATE_NEW_LAPACK -DACCELERATE_LAPACK_ILP64
// #cgo linux CPPFLAGS: -D_GNU_SOURCE
// #cgo windows CPPFLAGS: -D_CRT_SECURE_NO_WARNINGS
// #cgo CFLAGS: -std=gnu11 -O3 -w
// #cgo CXXFLAGS: -std=gnu++17 -O3 -w
// #cgo amd64 CFLAGS: -march=x86-64-v3
// #cgo amd64 CXXFLAGS: -march=x86-64-v3
// #cgo darwin LDFLAGS: -framework Accelerate
import "C"
