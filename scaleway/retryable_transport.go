package scaleway

import (
	"bytes"
	"context"
	"io"
	"io/ioutil"
	"net/http"
	"time"

	"github.com/hashicorp/go-retryablehttp"
)

// TODO Retry logic should be moved in the SDK
// newRetryableTransport creates a http transport with retry capability.
func newRetryableTransport(defaultTransport http.RoundTripper) http.RoundTripper {
	c := retryablehttp.NewClient()
	c.HTTPClient = &http.Client{Transport: defaultTransport}

	c.RetryMax = 3
	c.RetryWaitMax = 2 * time.Minute
	c.Logger = l
	c.RetryWaitMin = time.Second * 2
	c.CheckRetry = func(ctx context.Context, resp *http.Response, err error) (bool, error) {
		if resp == nil || resp.StatusCode == http.StatusTooManyRequests {
			return true, err
		}
		return retryablehttp.DefaultRetryPolicy(ctx, resp, err)
	}
	c.ErrorHandler = func(resp *http.Response, err error, numTries int) (*http.Response, error) {
		// Do not return error as response will be handled by scaleway sdk-go
		return resp, nil
	}

	return &retryableTransport{c}
}

// client is a bridge between scw.httpClient interface and retryablehttp.Client
type retryableTransport struct {
	*retryablehttp.Client
}

// RoundTrip wraps calling an HTTP method with retries.
func (c *retryableTransport) RoundTrip(r *http.Request) (*http.Response, error) {
	var body io.ReadSeeker
	if r.Body != nil {
		bs, err := ioutil.ReadAll(r.Body)
		if err != nil {
			return nil, err
		}
		body = bytes.NewReader(bs)
	}
	req, err := retryablehttp.NewRequest(r.Method, r.URL.String(), body)
	if err != nil {
		return nil, err
	}
	for key, val := range r.Header {
		req.Header.Set(key, val[0])
	}
	req.GetBody = func() (io.ReadCloser, error) {
		b, err := req.BodyBytes()
		if err != nil {
			return nil, err
		}
		return io.NopCloser(bytes.NewReader(b)), err
	}
	return c.Client.Do(req)
}
