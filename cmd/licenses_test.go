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
	"bytes"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLicensesCommand_PrintsLlamaCppNotice(t *testing.T) {
	notice, err := os.ReadFile(filepath.Join("..", "internal", "llama", "csrc", "LICENSE"))
	require.NoError(t, err)

	buf := new(bytes.Buffer)
	rootCmd.SetOut(buf)
	rootCmd.SetErr(buf)
	rootCmd.SetArgs([]string{"licenses"})
	require.NoError(t, rootCmd.Execute())

	assert.Contains(t, buf.String(), "llama.cpp")
	assert.Contains(t, buf.String(), string(notice))
}
