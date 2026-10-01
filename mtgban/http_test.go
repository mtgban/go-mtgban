package mtgban

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/hashicorp/go-retryablehttp"
)

// The header bound is set through a type assertion on retryablehttp's default
// transport, which would drop it silently if that transport ever changed.
func TestNewHTTPClientBoundsEveryAttempt(t *testing.T) {
	retrying := NewHTTPClient()
	rt, ok := retrying.Transport.(*retryablehttp.RoundTripper)
	if !ok {
		t.Fatalf("transport is %T, want the retrying round tripper", retrying.Transport)
	}
	inner := rt.Client.HTTPClient
	if inner.Timeout != defaultTimeout {
		t.Errorf("attempt timeout is %v, want %v", inner.Timeout, defaultTimeout)
	}
	transport, ok := inner.Transport.(*http.Transport)
	if !ok {
		t.Fatalf("transport is %T, which takes no response header timeout", inner.Transport)
	}
	if transport.ResponseHeaderTimeout != defaultHeaderTimeout {
		t.Errorf("header timeout is %v, want %v", transport.ResponseHeaderTimeout, defaultHeaderTimeout)
	}
}

func TestNewHTTPClientRetriesThroughTheWrapper(t *testing.T) {
	var served atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if served.Add(1) == 1 {
			w.WriteHeader(http.StatusServiceUnavailable)
		}
	}))
	defer srv.Close()

	var wrapped atomic.Int32
	var logged []string
	client := NewHTTPClient(
		WithHTTPRetries(2),
		WithHTTPRetryWait(time.Millisecond, time.Millisecond),
		WithHTTPTransport(func(rt http.RoundTripper) http.RoundTripper {
			return roundTripFunc(func(req *http.Request) (*http.Response, error) {
				wrapped.Add(1)
				return rt.RoundTrip(req)
			})
		}),
		WithHTTPLogCallback(func(format string, a ...any) {
			logged = append(logged, fmt.Sprintf(format, a...))
		}),
	)
	resp, err := client.Get(srv.URL + "/prices?sig=secret")
	if err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()
	if resp.StatusCode != http.StatusOK || wrapped.Load() != 2 {
		t.Errorf("got %d after %d wrapped attempts, want 200 after 2", resp.StatusCode, wrapped.Load())
	}
	if len(logged) != 1 || strings.Contains(logged[0], "secret") {
		t.Errorf("retry log %q, want one line without the query", logged)
	}
}

// Options are collected before the client is built, so a wrapper given first
// does not hide the transport the header bound is set on.
func TestNewHTTPClientBoundsAWrappedTransport(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		<-r.Context().Done()
	}))
	defer srv.Close()

	client := NewHTTPClient(
		WithHTTPTransport(func(rt http.RoundTripper) http.RoundTripper {
			return roundTripFunc(rt.RoundTrip)
		}),
		WithHTTPHeaderTimeout(50*time.Millisecond),
		WithHTTPTimeout(5*time.Second),
		WithHTTPRetries(0),
	)
	_, err := client.Get(srv.URL)
	if err == nil || !strings.Contains(err.Error(), "timeout awaiting response headers") {
		t.Errorf("a stalled server answered %v, want a header timeout", err)
	}
}

func TestNewHTTPClientReadsARedirectAsItIs(t *testing.T) {
	var followed atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/missing" {
			http.Redirect(w, r, "/home", http.StatusFound)
			return
		}
		followed.Add(1)
	}))
	defer srv.Close()

	client := NewHTTPClient(WithHTTPCheckRedirect(func(*http.Request, []*http.Request) error {
		return http.ErrUseLastResponse
	}))
	resp, err := client.Get(srv.URL + "/missing")
	if err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()
	if resp.StatusCode != http.StatusFound || followed.Load() != 0 {
		t.Errorf("got %d with the redirect followed %d times, want 302 unfollowed", resp.StatusCode, followed.Load())
	}
}

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return f(req)
}
