package fetch

import (
	"context"
	"crypto/tls"
	"fmt"
	"maps"
	"net/http"
	"time"

	"github.com/go-resty/resty/v2"
	log "github.com/sirupsen/logrus"
)

type Fetch struct {
	restClient *resty.Client
	httpClient *http.Client
	headers    map[string]string
	retries    int
}

var http11Transport = newHTTP11Transport()

// newHTTP11Transport clones http.DefaultTransport and disables HTTP/2 on the copy, so that opting out of HTTP/2 does
// not also opt out of proxy support and the default connection timeouts.
func newHTTP11Transport() *http.Transport {
	t := http.DefaultTransport.(*http.Transport).Clone()
	t.ForceAttemptHTTP2 = false
	t.TLSNextProto = make(map[string]func(string, *tls.Conn) http.RoundTripper)

	return t
}

var userAgent = "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) " +
	"Chrome/133.0.0.0 Safari/537.36"

// New creates a new Fetch instance with specified headers and retry settings.
//
// Parameters:
//   - headers: a map of headers to be set on each request.
//   - retries: the number of retry attempts for failed requests.
//   - disableHttp2: a boolean flag to disable HTTP/2.
//
// Returns a new Fetch instance.
func New(headers map[string]string, retries int, disableHttp2 bool) *Fetch {
	logger := log.New()

	f := resty.New()

	// Reuse the same policy as the download client, so the two cannot drift apart.
	f.SetRedirectPolicy(resty.FlexibleRedirectPolicy(maxRedirects), resty.RedirectPolicyFunc(safeCheckRedirect))

	if disableHttp2 {
		f.SetTransport(http11Transport)
	}

	// Copy rather than defaulting into the caller's map: the map is retained in the Fetch and read by NewRequest
	// from arbitrary goroutines, so sharing it with the caller is both surprising and racy.
	headers = maps.Clone(headers)
	if headers == nil {
		headers = make(map[string]string)
	}

	if _, exists := headers["User-Agent"]; !exists {
		headers["User-Agent"] = userAgent
	}
	if _, exists := headers["Content-Type"]; !exists {
		headers["Content-Type"] = "application/json"
	}

	return &Fetch{
		restClient: f.
			SetLogger(logger).
			SetHeaders(headers).
			SetRetryCount(retries).
			SetRetryWaitTime(0).
			SetRetryMaxWaitTime(maxBackoff).
			SetRetryAfter(retryAfter).
			AddRetryCondition(shouldRetry),

		httpClient: newIdleTimeoutClient(30 * time.Second),
		headers:    headers,
		retries:    retries,
	}
}

// GetText performs a GET request to the specified URL and returns the response body as a string.
//
// Parameters:
//   - ctx: context for cancellation and timeouts.
//   - url: the URL to send the GET request to.
//
// Returns the response body as a string and an error if the request fails.
func (f *Fetch) GetText(ctx context.Context, url string) (string, error) {
	resp, err := f.restClient.R().
		SetContext(ctx).
		Get(url)

	if err != nil {
		status := 0
		if resp != nil {
			status = resp.StatusCode()
		}

		log.WithFields(log.Fields{
			"error":  err,
			"status": status,
			"url":    url,
		}).Error("Error getting text")

		return "", err
	}

	if resp.IsError() {
		log.WithFields(log.Fields{
			"status": resp.StatusCode(),
			"url":    url,
		}).Error("Error getting text")

		return "", fmt.Errorf("%s", resp.Status())
	}

	return resp.String(), nil
}

// GetResult performs a GET request to the specified URL and unmarshals the response body into the provided result
// interface.
//
// Parameters:
//   - ctx: context for cancellation and timeouts.
//   - url: the URL to send the GET request to.
//   - headers: per-request headers to set in addition to the client defaults.
//   - result: a pointer to the variable where the response body will be unmarshalled.
//
// Returns:
//   - *resty.Response: the response from the GET request.
//   - error: an error if the request fails or the response indicates an error.
func (f *Fetch) GetResult(ctx context.Context, url string, headers map[string]string, result any) (*resty.Response, error) {
	return f.doRequest(ctx, url, headers, nil, result, "GET")
}

// PostResult performs a POST request to the specified URL and unmarshals the response body into the provided result
// interface.
//
// Parameters:
//   - ctx: context for cancellation and timeouts.
//   - url: the URL to send the POST request to.
//   - headers: per-request headers to set in addition to the client defaults.
//   - body: optional request body to be marshalled as JSON.
//   - result: a pointer to the variable where the response body will be unmarshalled.
//
// Returns:
//   - *resty.Response: the response from the POST request.
//   - error: an error if the request fails or the response indicates an error.
func (f *Fetch) PostResult(ctx context.Context, url string, headers map[string]string, body any, result any) (*resty.Response, error) {
	return f.doRequest(ctx, url, headers, body, result, "POST")
}

// doRequest performs an HTTP request with the specified method and handles common error logging
func (f *Fetch) doRequest(ctx context.Context, url string, headers map[string]string, body any, result any, method string) (*resty.Response, error) {
	req := f.restClient.R().
		SetContext(ctx).
		SetHeaders(headers).
		ForceContentType("application/json").
		SetResult(result)

	if body != nil {
		req.SetBody(body)
	}

	var resp *resty.Response
	var err error

	switch method {
	case "GET":
		resp, err = req.Get(url)
	case "POST":
		resp, err = req.Post(url)
	default:
		return nil, fmt.Errorf("unsupported HTTP method: %s", method)
	}

	if err != nil {
		status := 0
		if resp != nil {
			status = resp.StatusCode()
		}

		log.WithFields(log.Fields{
			"error":  err,
			"status": status,
			"url":    url,
		}).Error("error getting result")

		return resp, err
	}

	if resp.IsError() {
		log.WithFields(log.Fields{
			"status": resp.StatusCode(),
			"url":    url,
		}).Error("Error getting result")

		return resp, fmt.Errorf("%s", resp.Status())
	}

	return resp, nil
}

// region - Retry policy

// shouldRetry reports whether a failed attempt is worth repeating.
//
// resty invokes a retry condition with a nil *Response when the failure happened before a request could even be sent,
// such as an unparsable URL, so every field access here has to be guarded.
func shouldRetry(r *resty.Response, err error) bool {
	if err != nil {
		return true
	}

	if r == nil {
		return false
	}

	// A 403 or a 404 answers the same way ten times in a row; only retry what the server said was temporary.
	return r.IsError() && isRetryableStatus(r.StatusCode())
}

// retryAfter returns how long to wait before the next attempt, capped at maxBackoff.
//
// The wait is returned rather than slept for, so that resty owns the delay: sleeping inside the retry condition made
// the wait uncancellable by the request context, and resty then slept again on top of it.
func retryAfter(_ *resty.Client, r *resty.Response) (time.Duration, error) {
	attempt := 1
	if r != nil && r.Request != nil {
		attempt = r.Request.Attempt
	}

	wait := time.Duration(fibonacci(attempt+1)) * time.Second
	wait = min(wait, maxBackoff)

	fields := log.Fields{"attempt": attempt, "retry_in": wait}
	if r != nil {
		fields["status"] = r.StatusCode()
		if r.Request != nil {
			fields["url"] = r.Request.URL
		}
	}
	log.WithFields(fields).Warn("failed to get data; retrying")

	return wait, nil
}

// endregion
