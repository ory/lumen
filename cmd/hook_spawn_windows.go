//go:build windows

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

// spawnBackgroundIndexer is a no-op on Windows.
// Windows background indexing is not supported (requires flock + Setsid, both Unix-only).
// To add Windows support, use CREATE_NEW_PROCESS_GROUP | DETACHED_PROCESS via
// syscall.SysProcAttr and replace flock with a Windows mutex or named pipe.
func spawnBackgroundIndexer(_ string) {}
