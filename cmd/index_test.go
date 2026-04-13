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
	"testing"

	"github.com/spf13/cobra"
)

func TestApplyModelFlag_AcceptsModelAlias(t *testing.T) {
	cmd := &cobra.Command{}
	cmd.Flags().StringP("model", "m", "", "")
	if err := cmd.Flags().Set("model", "text-embedding-nomic-embed-code"); err != nil {
		t.Fatalf("failed to set flag: %v", err)
	}
	if err := applyModelFlag(cmd); err != nil {
		t.Errorf("expected alias to be accepted, got error: %v", err)
	}
}

func TestApplyModelFlag_AcceptsCanonicalModel(t *testing.T) {
	cmd := &cobra.Command{}
	cmd.Flags().StringP("model", "m", "", "")
	if err := cmd.Flags().Set("model", "nomic-ai/nomic-embed-code-GGUF"); err != nil {
		t.Fatalf("failed to set flag: %v", err)
	}
	if err := applyModelFlag(cmd); err != nil {
		t.Errorf("expected canonical model to be accepted, got error: %v", err)
	}
}

func TestApplyModelFlag_RejectsUnknownModel(t *testing.T) {
	cmd := &cobra.Command{}
	cmd.Flags().StringP("model", "m", "", "")
	if err := cmd.Flags().Set("model", "not-a-real-model"); err != nil {
		t.Fatalf("failed to set flag: %v", err)
	}
	if err := applyModelFlag(cmd); err == nil {
		t.Error("expected error for unknown model, got nil")
	}
}
