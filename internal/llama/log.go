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

package llama

import "C"

import (
	"log/slog"
	"strings"
	"sync/atomic"
)

var logger atomic.Pointer[slog.Logger]

// SetLogger routes llama.cpp warnings and errors to l. Without a logger,
// native log output is discarded so it never reaches stdout or stderr.
func SetLogger(l *slog.Logger) {
	logger.Store(l)
}

//export lumenLlamaLog
func lumenLlamaLog(level C.int, text *C.char) {
	l := logger.Load()
	if l == nil {
		return
	}
	msg := strings.TrimSpace(C.GoString(text))
	if msg == "" {
		return
	}
	if level == C.int(4) { // GGML_LOG_LEVEL_ERROR
		l.Error("llama.cpp", "msg", msg)
		return
	}
	l.Warn("llama.cpp", "msg", msg)
}
