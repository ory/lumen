// Package sqlitevec registers the bundled sqlite-vec v0.1.9 extension and
// contains the small amount of vector serialization needed by the store.
package sqlitevec

// #cgo CFLAGS: -DSQLITE_CORE
// #cgo linux LDFLAGS: -lm
// /* SQLITE_CORE intentionally resolves sqlite-vec's SQLite symbols against
// the same mattn/go-sqlite3 amalgamation linked into this process. This matches
// the upstream sqlite-vec Go bindings and avoids loading a second SQLite ABI. */
// #include "sqlite-vec.h"
import "C"

import (
	"bytes"
	"encoding/binary"
	"fmt"
)

// Auto registers sqlite-vec for every SQLite connection opened afterward.
func Auto() error {
	if rc := C.sqlite3_auto_extension((*[0]byte)(C.sqlite3_vec_init)); rc != C.SQLITE_OK {
		return fmt.Errorf("register sqlite-vec auto extension: sqlite error %d", int(rc))
	}
	return nil
}

// Cancel cancels the automatic sqlite-vec extension registration.
func Cancel() {
	// Cancellation is best-effort cleanup; registration may already be absent.
	_ = C.sqlite3_cancel_auto_extension((*[0]byte)(C.sqlite3_vec_init))
}

// SerializeFloat32 encodes a vector as sqlite-vec's little-endian float BLOB.
func SerializeFloat32(vector []float32) ([]byte, error) {
	buf := new(bytes.Buffer)
	if err := binary.Write(buf, binary.LittleEndian, vector); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}
