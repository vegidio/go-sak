package fetch

import (
	"context"
	"crypto/tls"
	"fmt"
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

var http11Transport = &http.Transport{
	ForceAttemptHTTP2: false,
	TLSNextProto:      make(map[string]func(string, *tls.Conn) http.RoundTripper),
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
	f.SetRedirectPolicy(resty.FlexibleRedirectPolicy(maxRedirects), resty.RedirectPolicyFunc(func(req *http.Request, via []*http.Request) error {
		if len(via) > 0 && via[0].URL.Scheme == "https" && req.URL.Scheme == "http" {
			return fmt.Errorf("refusing redirect from https to http: %s", req.URL)
		}

		return nil
	}))

	if disableHttp2 {
		f.SetTransport(http11Transport)
	}

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
			AddRetryCondition(
				func(r *resty.Response, err error) bool {
					if (err != nil || r.IsError()) && r.Request.Attempt <= retries {
						sleep := time.Duration(fibonacci(r.Request.Attempt+1)) * time.Second

						log.WithFields(log.Fields{
							"attempt": r.Request.Attempt,
							"error":   r.Error(),
							"status":  r.StatusCode(),
							"url":     r.Request.URL,
						}).Warn("failed to get data; retrying in ", sleep)

						time.Sleep(sleep)
						return true
					}

					return false
				},
			),

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
