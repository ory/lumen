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

#include "bridge.h"

#include <string.h>

#include "_cgo_export.h"

static void lumen_llama_log(enum ggml_log_level level, const char * text, void * user_data) {
    (void) user_data;
    if (level == GGML_LOG_LEVEL_WARN || level == GGML_LOG_LEVEL_ERROR) {
        lumenLlamaLog((int) level, (char *) text);
    }
}

void lumen_llama_init(void) {
    llama_log_set(lumen_llama_log, NULL);
    llama_backend_init();
}

static bool lumen_llama_load_progress(float progress, void * data) {
    (void) progress;
    return *(volatile int32_t *) data == 0;
}

static bool lumen_llama_abort(void * data) {
    return *(volatile int32_t *) data != 0;
}

struct llama_model * lumen_llama_load_model(const char * path, int cpu_only, volatile int32_t * cancel) {
    static ggml_backend_dev_t no_devices[1] = { NULL };

    struct llama_model_params params = llama_model_default_params();
    params.progress_callback = lumen_llama_load_progress;
    params.progress_callback_user_data = (void *) cancel;
    if (cpu_only) {
        params.devices = no_devices;
        params.n_gpu_layers = 0;
    }
    return llama_model_load_from_file(path, params);
}

struct llama_context * lumen_llama_new_context(struct llama_model * model, int32_t n_ctx, int32_t n_seq_max,
                                               int32_t n_threads, int32_t pooling, int cpu_only,
                                               volatile int32_t * cancel) {
    struct llama_context_params params = llama_context_default_params();
    params.n_ctx = (uint32_t) n_ctx;
    params.n_batch = (uint32_t) n_ctx;
    params.n_ubatch = (uint32_t) n_ctx;
    params.n_seq_max = (uint32_t) n_seq_max;
    params.n_threads = n_threads;
    params.n_threads_batch = n_threads;
    params.embeddings = true;
    params.pooling_type = (enum llama_pooling_type) pooling;
    params.kv_unified = true;
    params.abort_callback = lumen_llama_abort;
    params.abort_callback_data = (void *) cancel;
    if (cpu_only) {
        params.offload_kqv = false;
        params.op_offload = false;
    }
    return llama_init_from_model(model, params);
}

int32_t lumen_llama_embed(struct llama_context * ctx, const int32_t * tokens, const int32_t * lens, int32_t n_seq,
                          float * out, int32_t n_embd) {
    int32_t n_tokens = 0;
    for (int32_t s = 0; s < n_seq; s++) {
        n_tokens += lens[s];
    }

    struct llama_batch batch = llama_batch_init(n_tokens, 0, 1);
    int32_t k = 0;
    for (int32_t s = 0; s < n_seq; s++) {
        for (int32_t i = 0; i < lens[s]; i++, k++) {
            batch.token[k] = tokens[k];
            batch.pos[k] = i;
            batch.n_seq_id[k] = 1;
            batch.seq_id[k][0] = s;
            batch.logits[k] = 1;
        }
    }
    batch.n_tokens = n_tokens;

    // Every call is independent: drop state left by the previous batch.
    llama_memory_clear(llama_get_memory(ctx), true);
    int32_t rc = llama_decode(ctx, batch);
    if (rc == 0) {
        for (int32_t s = 0; s < n_seq; s++) {
            const float * embd = llama_get_embeddings_seq(ctx, s);
            if (embd == NULL) {
                rc = -1000 - s;
                break;
            }
            memcpy(out + (size_t) s * (size_t) n_embd, embd, (size_t) n_embd * sizeof(float));
        }
    }
    llama_batch_free(batch);
    return rc;
}
