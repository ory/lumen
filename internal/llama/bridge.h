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

#ifndef LUMEN_LLAMA_BRIDGE_H
#define LUMEN_LLAMA_BRIDGE_H

#include <stdint.h>

#include "llama.h"

// lumen_llama_init initializes the llama.cpp backends once per process and
// routes native log output to the Go logger instead of stderr.
void lumen_llama_init(void);

// lumen_llama_load_model loads a GGUF model. cancel points to a flag that
// aborts loading when set to a non-zero value. When cpu_only is non-zero no
// accelerator device is used.
struct llama_model * lumen_llama_load_model(const char * path, int cpu_only, volatile int32_t * cancel);

// lumen_llama_new_context creates an embedding context with the given pooling
// type. cancel aborts in-flight CPU graph computation when non-zero.
struct llama_context * lumen_llama_new_context(struct llama_model * model, int32_t n_ctx, int32_t n_seq_max,
                                               int32_t n_threads, int32_t pooling, int cpu_only,
                                               volatile int32_t * cancel);

// lumen_llama_embed decodes n_seq token sequences laid out back to back in
// tokens (lengths in lens) as one batch and copies each pooled sequence
// embedding (n_embd floats) into out in input order. Returns 0 on success,
// the llama_decode status on decode failure, or -1000 - i when sequence i
// produced no embedding.
int32_t lumen_llama_embed(struct llama_context * ctx, const int32_t * tokens, const int32_t * lens, int32_t n_seq,
                          float * out, int32_t n_embd);

#endif
