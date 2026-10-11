package uber

import "net/http"

// ÜberClient talks to the partner API for the über-shipping integration.
type ÜberClient struct {
	base string
	http *http.Client
}

// Send posts a shipment to the partner API.
func (c *ÜberClient) Send(path string) (*http.Response, error) {
	return c.http.Post(c.base+path, "application/json", nil)
}

// Close releases idle connections held by the ÜberClient transport.
func (c *ÜberClient) Close() {
	c.http.CloseIdleConnections()
}
