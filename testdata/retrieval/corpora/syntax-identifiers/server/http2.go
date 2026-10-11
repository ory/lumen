package server

import (
	"crypto/tls"
	"net/http"

	"golang.org/x/net/http2"
)

// HTTP2Server serves h2 over TLS with tuned stream limits.
type HTTP2Server struct {
	Addr                 string
	MaxConcurrentStreams uint32
	TLS                  *tls.Config
}

// ListenAndServe configures h2 on a standard library server and starts it.
func (s *HTTP2Server) ListenAndServe(h http.Handler) error {
	srv := &http.Server{Addr: s.Addr, Handler: h, TLSConfig: s.TLS}
	if err := http2.ConfigureServer(srv, &http2.Server{MaxConcurrentStreams: s.MaxConcurrentStreams}); err != nil {
		return err
	}
	return srv.ListenAndServeTLS("", "")
}
