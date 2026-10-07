package mtgban

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/hashicorp/go-retryablehttp"
)

// The slowest fetch a scraper makes, MTGBAN's 120 MB Magic price dump, takes
// about 11s to its headers and 100s in all; these bound a stalled one.
const (
	defaultHeaderTimeout = 2 * time.Minute
	defaultTimeout       = 10 * time.Minute
)

// HTTPOption configures a client from NewHTTPClient.
type HTTPOption func(*httpConfig)

type httpConfig struct {
	headerTimeout time.Duration
	timeout       time.Duration
	retries       int
	waitMin       time.Duration
	waitMax       time.Duration
	backoff       retryablehttp.Backoff
	retryPosts    bool
	errorHandler  retryablehttp.ErrorHandler
	checkRedirect func(*http.Request, []*http.Request) error
	wraps         []func(http.RoundTripper) http.RoundTripper
}

type logKey struct{}

// ContextWithLogCallback returns ctx carrying fn, which a client from
// NewHTTPClient reports each retry of a request made with ctx through: the
// method and the address without its query, which can carry a signature.
func ContextWithLogCallback(ctx context.Context, fn LogCallbackFunc) context.Context {
	return context.WithValue(ctx, logKey{}, fn)
}

// WithHTTPTimeout bounds each attempt, body included.
func WithHTTPTimeout(d time.Duration) HTTPOption {
	return func(c *httpConfig) { c.timeout = d }
}

// WithHTTPHeaderTimeout bounds the wait for each attempt's response headers.
func WithHTTPHeaderTimeout(d time.Duration) HTTPOption {
	return func(c *httpConfig) { c.headerTimeout = d }
}

// WithHTTPRetries sets how many times a failed request is retried.
func WithHTTPRetries(n int) HTTPOption {
	return func(c *httpConfig) { c.retries = n }
}

// WithHTTPRetryWait sets the shortest and longest wait between attempts.
func WithHTTPRetryWait(minWait, maxWait time.Duration) HTTPOption {
	return func(c *httpConfig) { c.waitMin, c.waitMax = minWait, maxWait }
}

// WithHTTPBackoff replaces how long to wait before each retry.
func WithHTTPBackoff(fn retryablehttp.Backoff) HTTPOption {
	return func(c *httpConfig) { c.backoff = fn }
}

// WithHTTPRetryPosts retries a POST or PATCH as it does a GET, for a client
// whose POSTs only read, such as a search sent as a body.
func WithHTTPRetryPosts() HTTPOption {
	return func(c *httpConfig) { c.retryPosts = true }
}

// WithHTTPErrorHandler replaces what a request answers once its retries run
// out.
func WithHTTPErrorHandler(fn retryablehttp.ErrorHandler) HTTPOption {
	return func(c *httpConfig) { c.errorHandler = fn }
}

// WithHTTPCheckRedirect replaces the redirect policy, on the client returned
// and on the one each attempt goes through, which would otherwise follow the
// redirect itself: return http.ErrUseLastResponse to read a 3xx as it is.
func WithHTTPCheckRedirect(fn func(*http.Request, []*http.Request) error) HTTPOption {
	return func(c *httpConfig) { c.checkRedirect = fn }
}

// WithHTTPTransport wraps the transport every attempt goes through, inside
// the retries, so a wrapper that signs a request signs each attempt. Wraps
// apply in the order given, the last one outermost.
func WithHTTPTransport(wrap func(http.RoundTripper) http.RoundTripper) HTTPOption {
	return func(c *httpConfig) { c.wraps = append(c.wraps, wrap) }
}

// NewHTTPClient returns the client scrapers fetch with: it retries a failed
// request with retryablehttp's policy, logging each retry through the
// request context's ContextWithLogCallback, and bounds every attempt, so a
// server that stops answering costs a retry instead of the whole run. A POST
// or PATCH, which the server may have acted on, is retried only when it was
// never read, unless WithHTTPRetryPosts says the client's POSTs only read.
func NewHTTPClient(opts ...HTTPOption) *http.Client {
	client := retryablehttp.NewClient()
	client.Logger = nil

	config := httpConfig{
		headerTimeout: defaultHeaderTimeout,
		timeout:       defaultTimeout,
		retries:       client.RetryMax,
		waitMin:       client.RetryWaitMin,
		waitMax:       client.RetryWaitMax,
	}
	for _, opt := range opts {
		opt(&config)
	}

	transport, ok := client.HTTPClient.Transport.(*http.Transport)
	if ok {
		transport.ResponseHeaderTimeout = config.headerTimeout
	}
	for _, wrap := range config.wraps {
		client.HTTPClient.Transport = wrap(client.HTTPClient.Transport)
	}
	client.HTTPClient.Timeout = config.timeout
	client.HTTPClient.CheckRedirect = config.checkRedirect

	client.RetryMax = config.retries
	client.RetryWaitMin = config.waitMin
	client.RetryWaitMax = config.waitMax
	if config.backoff != nil {
		client.Backoff = config.backoff
	}
	if !config.retryPosts {
		client.CheckRetry = retryUnreadWrites
	}
	if config.errorHandler != nil {
		client.ErrorHandler = config.errorHandler
	}
	client.RequestLogHook = func(_ retryablehttp.Logger, req *http.Request, attempt int) {
		logf, ok := req.Context().Value(logKey{}).(LogCallbackFunc)
		if attempt == 0 || !ok || logf == nil {
			return
		}
		logf("%s %s://%s%s: retry %d of %d", req.Method, req.URL.Scheme, req.URL.Host, req.URL.Path, attempt, config.retries)
	}
	standard := client.StandardClient()
	standard.CheckRedirect = config.checkRedirect
	return standard
}

// retryUnreadWrites is retryablehttp's policy, except that a POST or PATCH is
// retried only when the server never read it: its dial failed, or it was
// answered 429.
func retryUnreadWrites(ctx context.Context, resp *http.Response, err error) (bool, error) {
	retry, checkErr := retryablehttp.DefaultRetryPolicy(ctx, resp, err)
	if !retry || !isWrite(resp, err) {
		return retry, checkErr
	}
	if resp != nil {
		if resp.StatusCode == http.StatusTooManyRequests {
			return true, checkErr
		}
		// Still a failure, so the caller does not take it for a success
		return false, fmt.Errorf("unexpected HTTP status %s, not retried: the server may have acted on it", resp.Status)
	}
	var opErr *net.OpError
	return errors.As(err, &opErr) && opErr.Op == "dial", checkErr
}

// isWrite reads the method off the response or, with none, off the error
// net/http names it in ("Post").
func isWrite(resp *http.Response, err error) bool {
	var method string
	var urlErr *url.Error
	if resp != nil && resp.Request != nil {
		method = resp.Request.Method
	} else if errors.As(err, &urlErr) {
		method = strings.ToUpper(urlErr.Op)
	}
	return method == http.MethodPost || method == http.MethodPatch
}
