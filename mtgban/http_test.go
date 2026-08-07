package mtgban

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/hashicorp/go-cleanhttp"
)

func TestSetTimeouts(t *testing.T) {
	client := SetTimeouts(cleanhttp.DefaultClient())

	if client.Timeout != requestTimeout {
		t.Errorf("request timeout is %v, expected %v", client.Timeout, requestTimeout)
	}

	transport, ok := client.Transport.(*http.Transport)
	if !ok {
		t.Fatalf("transport is %T, expected *http.Transport", client.Transport)
	}
	if transport.ResponseHeaderTimeout != responseHeaderTimeout {
		t.Errorf("header timeout is %v, expected %v", transport.ResponseHeaderTimeout, responseHeaderTimeout)
	}

	if SetTimeouts(nil) != nil {
		t.Error("a nil client should pass through")
	}
}

// A peer that accepts the connection and then never replies is the case these
// timeouts exist for: without one the request never returns. The bound is
// lowered here so the test does not wait for the real one.
func TestSetTimeoutsEndsAStalledRequest(t *testing.T) {
	release := make(chan struct{})
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		<-release
	}))
	defer func() {
		close(release)
		server.Close()
	}()

	client := SetTimeouts(cleanhttp.DefaultClient())
	client.Transport.(*http.Transport).ResponseHeaderTimeout = 100 * time.Millisecond

	done := make(chan error, 1)
	go func() {
		resp, err := client.Get(server.URL)
		if resp != nil {
			resp.Body.Close()
		}
		done <- err
	}()

	select {
	case err := <-done:
		if err == nil {
			t.Error("expected the stalled request to fail")
		}
	case <-time.After(10 * time.Second):
		t.Error("the request outlived its header timeout")
	}
}
