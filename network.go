package spectest

import "net/http"

// network is used to enable/disable networking for the test
type network struct {
	// Client is the http client used when networking is enabled.
	// By default, http.DefaultClient is used.
	*http.Client
	// enabled will enable networking for provided clients
	enabled bool
}

// newNetwork creates a new network setting
func newNetwork() *network {
	return &network{
		Client: http.DefaultClient,
	}
}

// isEnable returns true if networking is enabled
func (n *network) isEnable() bool {
	return n.enabled
}
