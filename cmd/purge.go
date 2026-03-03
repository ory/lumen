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
	"os"
	"path/filepath"

	"github.com/aeneasr/lumen/internal/config"
	"github.com/spf13/cobra"
)

func init() {
	rootCmd.AddCommand(purgeCmd)
}

var purgeCmd = &cobra.Command{
	Use:   "purge",
	Short: "Remove all lumen index data",
	Long:  "Deletes all lumen index databases from ~/.local/share/lumen/. This is irreversible — indexes will be rebuilt on the next search.",
	Args:  cobra.NoArgs,
	RunE:  runPurge,
}

func runPurge(_ *cobra.Command, _ []string) error {
	dataDir := filepath.Join(config.XDGDataDir(), "lumen")

	info, err := os.Stat(dataDir)
	if err != nil {
		if os.IsNotExist(err) {
			fmt.Fprintln(os.Stderr, "No index data found — nothing to purge.")
			return nil
		}
		return fmt.Errorf("stat data directory: %w", err)
	}
	if !info.IsDir() {
		return fmt.Errorf("%s is not a directory", dataDir)
	}

	if err := os.RemoveAll(dataDir); err != nil {
		return fmt.Errorf("remove index data: %w", err)
	}
	fmt.Fprintf(os.Stderr, "Removed all index data (%s)\n", dataDir)
	return nil
}
