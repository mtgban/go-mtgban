package mtgban

import (
	"net/http"
	"time"
)

const (
	// How long a server has to start replying before the request is
	// abandoned. This deliberately leaves the body alone: several feeds are
	// tens of megabytes and legitimately stream for minutes, while a server
	// that accepts a connection and then says nothing is the case worth
	// cutting short.
	responseHeaderTimeout = 2 * time.Minute

	// Ceiling for a whole request, body included. Set well above the slowest
	// feed observed so that no real download is at risk; it exists to end a
	// stalled transfer eventually rather than to keep one brief.
	requestTimeout = 30 * time.Minute
)

// SetTimeouts bounds how long a single request made with this client may take
// and returns the client, so it can wrap a constructor call. Without it a
// request has no deadline of its own: a peer that accepts a connection and
// then stops talking holds the calling goroutine until the process is killed.
func SetTimeouts(client *http.Client) *http.Client {
	if client == nil {
		return nil
	}

	client.Timeout = requestTimeout

	// Only the stock transport exposes the header timeout, and a client built
	// by cleanhttp always carries one
	transport, ok := client.Transport.(*http.Transport)
	if ok {
		transport.ResponseHeaderTimeout = responseHeaderTimeout
	}

	return client
}
