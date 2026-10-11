// Package example provides test fixtures.
package example

import "fmt"

// Hello prints a greeting.
func Hello(name string) {
	fmt.Println("hello", name)
}

// Greeter defines a greeting interface.
type Greeter interface {
	Greet(name string) string
}

// Server handles requests.
type Server struct {
	Port int
	Host string
}

// Start launches the server.
func (s *Server) Start() error {
	return nil
}

// MaxRetries is the max retry count.
const MaxRetries = 3

// DefaultHost is the default hostname.
var DefaultHost = "localhost"
