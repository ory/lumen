package server

import (
	"net/http"
	"time"
)

// HTTPServer serves plain HTTP/1.1 for health checks and redirects.
type HTTPServer struct {
	Addr        string
	ReadTimeout time.Duration
}

// Serve starts the plain listener.
func (s *HTTPServer) Serve(h http.Handler) error {
	srv := &http.Server{Addr: s.Addr, Handler: h, ReadTimeout: s.ReadTimeout}
	return srv.ListenAndServe()
}
