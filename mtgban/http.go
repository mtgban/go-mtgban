package mtgban

import (
	"context"
	"net/http"
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
	checkRetry    retryablehttp.CheckRetry
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

// WithHTTPCheckRetry replaces the decision whether a response or an error is
// retried.
func WithHTTPCheckRetry(fn retryablehttp.CheckRetry) HTTPOption {
	return func(c *httpConfig) { c.checkRetry = fn }
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
// server that stops answering costs a retry instead of the whole run.
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
	if config.checkRetry != nil {
		client.CheckRetry = config.checkRetry
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
