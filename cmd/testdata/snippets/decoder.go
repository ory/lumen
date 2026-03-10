package decoder

import "fmt"

// Decoder handles binary protocol decoding.
type Decoder struct {
	buf    []byte
	offset int
}

// NewDecoder creates a Decoder from raw bytes.
func NewDecoder(buf []byte) *Decoder {
	return &Decoder{buf: buf}
}

// decodeStruct reads a struct from the buffer.
// It processes each field sequentially.
func (d *Decoder) decodeStruct() (map[string]interface{}, error) {
	result := make(map[string]interface{})

	fieldCount, err := d.readVarInt()
	if err != nil {
		return nil, fmt.Errorf("read field count: %w", err)
	}

	for i := 0; i < fieldCount; i++ {
		name, err := d.readString()
		if err != nil {
			return nil, fmt.Errorf("read field name: %w", err)
		}

		value, err := d.readValue()
		if err != nil {
			return nil, fmt.Errorf("read field %s: %w", name, err)
		}

		result[name] = value
	}

	return result, nil
}

// readVarInt reads a variable-length integer.
func (d *Decoder) readVarInt() (int, error) {
	if d.offset >= len(d.buf) {
		return 0, fmt.Errorf("unexpected EOF")
	}
	val := int(d.buf[d.offset])
	d.offset++
	return val, nil
}

// readString reads a length-prefixed string.
func (d *Decoder) readString() (string, error) {
	length, err := d.readVarInt()
	if err != nil {
		return "", err
	}
	if d.offset+length > len(d.buf) {
		return "", fmt.Errorf("string exceeds buffer")
	}
	s := string(d.buf[d.offset : d.offset+length])
	d.offset += length
	return s, nil
}

// readValue reads a typed value from the buffer.
func (d *Decoder) readValue() (interface{}, error) {
	typ, err := d.readVarInt()
	if err != nil {
		return nil, err
	}
	switch typ {
	case 0:
		return d.readVarInt()
	case 1:
		return d.readString()
	default:
		return nil, fmt.Errorf("unknown type %d", typ)
	}
}
