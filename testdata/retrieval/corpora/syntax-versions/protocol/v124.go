package protocol

import "net"

// handshakeV124 negotiates framing with compressed headers.
func handshakeV124(conn net.Conn) error {
	_, err := conn.Write([]byte("HELLO protocol1.2.4\n"))
	return err
}
