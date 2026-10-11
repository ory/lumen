package auth

// HTTP2Credentials carries the bearer token presented on h2 connections.
type HTTP2Credentials struct {
	Token  string
	Expiry int64
}
