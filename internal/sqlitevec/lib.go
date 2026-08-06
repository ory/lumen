// Package sqlitevec registers the bundled sqlite-vec v0.1.9 extension and
// contains the small amount of vector serialization needed by the store.
package sqlitevec

// #cgo CFLAGS: -DSQLITE_CORE
// #cgo linux LDFLAGS: -lm
// #include "sqlite-vec.h"
import "C"

import (
	"bytes"
	"encoding/binary"
)

// Auto registers sqlite-vec for every SQLite connection opened afterward.
func Auto() {
	C.sqlite3_auto_extension((*[0]byte)(C.sqlite3_vec_init))
}

// Cancel cancels the automatic sqlite-vec extension registration.
func Cancel() {
	C.sqlite3_cancel_auto_extension((*[0]byte)(C.sqlite3_vec_init))
}

// SerializeFloat32 encodes a vector as sqlite-vec's little-endian float BLOB.
func SerializeFloat32(vector []float32) ([]byte, error) {
	buf := new(bytes.Buffer)
	if err := binary.Write(buf, binary.LittleEndian, vector); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}
