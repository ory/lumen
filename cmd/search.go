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

package cmd

import (
	"fmt"
	"io"
	"time"
)

type traceSpan struct {
	label    string
	duration time.Duration
	detail   string
}

type tracer struct {
	enabled bool
	start   time.Time
	last    time.Time
	spans   []traceSpan
}

func (t *tracer) record(label, detail string) {
	if !t.enabled {
		return
	}
	now := time.Now()
	t.spans = append(t.spans, traceSpan{
		label:    label,
		duration: now.Sub(t.last),
		detail:   detail,
	})
	t.last = now
}

func (t *tracer) print(w io.Writer) {
	if !t.enabled {
		return
	}
	const sep = "───────────────────────────────────────────────────────────────────────"
	for _, s := range t.spans {
		ms := s.duration.Milliseconds()
		fmt.Fprintf(w, "[%4dms] %-22s → %s\n", ms, s.label, s.detail)
	}
	fmt.Fprintln(w, sep)
	total := time.Since(t.start)
	fmt.Fprintf(w, "[%4dms] total\n", total.Milliseconds())
}
