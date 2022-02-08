package transport

import (
	"bytes"
	"io/ioutil"
	"net/http"
	"time"

	"github.com/konradasb/github_exporter/validators"
)

type revalidationTransport struct {
	Validator validators.Validator
	Next      http.RoundTripper
}

// NewRevalidationTransport creates a new RevalidationTransport instance
//
// RevalidationTransport is responsible for revalidating requests from cache
// based on their configured maximum age
//
// On each request, it checks for X-Cache-Age headers. Upon success,
// determines the current age of the request and validates it
// against a Validator implementation
//
// If the request is still considered valid by the Validator,
// returns a new empty response with status 304
//
// The response will have X-Revalidated headers, indicating
// that the response was sent from this Transport
func NewRevalidationTransport(rt http.RoundTripper, v validators.Validator) *revalidationTransport {
	if rt == nil {
		rt = http.DefaultTransport
	}

	if v == nil {
		v = validators.NewEmptyValidator()
	}

	t := &revalidationTransport{
		Next:      rt,
		Validator: v,
	}

	return t
}

// RoundTrip implements http.RoundTripper
func (t *revalidationTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	x := req.Header.Get("X-Cache-Age")
	if x != "" {
		age, err := time.ParseDuration(x + "s")
		if err == nil && t.Validator.Valid(req.URL, age) {
			resp := &http.Response{
				Request:          req,
				TransferEncoding: req.TransferEncoding,
				StatusCode:       http.StatusNotModified,
				Status:           http.StatusText(http.StatusNotModified),
				Body:             ioutil.NopCloser(bytes.NewReader([]byte(""))),
				Header:           http.Header{},
			}
			resp.Header.Set("X-Revalidated", "1")
			return resp, nil
		}
	}
	return t.Next.RoundTrip(req)
}
