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

package embedder

import (
	"fmt"
	"os"
	"unsafe"

	"golang.org/x/sys/windows"
)

// fileBasicInfo is the Win32 FILE_BASIC_INFO structure.
type fileBasicInfo struct {
	CreationTime, LastAccessTime, LastWriteTime, ChangeTime int64
	FileAttributes                                          uint32
	_                                                       uint32
}

// fileVersion identifies the content version of an open file by volume,
// file index, size, and last-write and change times. Renaming another file
// into place changes the file index, and rewriting a file changes its change
// time even when its last-write time is restored.
func fileVersion(f *os.File) (string, bool) {
	h := windows.Handle(f.Fd())
	var id windows.ByHandleFileInformation
	if err := windows.GetFileInformationByHandle(h, &id); err != nil {
		return "", false
	}
	var bi fileBasicInfo
	if err := windows.GetFileInformationByHandleEx(h, windows.FileBasicInfo, (*byte)(unsafe.Pointer(&bi)), uint32(unsafe.Sizeof(bi))); err != nil {
		return "", false
	}
	size := int64(id.FileSizeHigh)<<32 | int64(id.FileSizeLow)
	return fmt.Sprintf("w%x:%x%08x:%d:%d:%d", id.VolumeSerialNumber, id.FileIndexHigh, id.FileIndexLow,
		size, bi.LastWriteTime, bi.ChangeTime), true
}
