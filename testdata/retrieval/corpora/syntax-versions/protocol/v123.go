package protocol

import "net"

// handshakeV123 negotiates the original framing with length-prefixed headers.
func handshakeV123(conn net.Conn) error {
	_, err := conn.Write([]byte("HELLO protocol1.2.3\n"))
	return err
}
