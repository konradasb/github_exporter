package transport

import (
	"net/http"
	"time"

	"github.com/google/go-github/v42/github"
)

// RatelimitTransport implements http.RoundTripper
type RatelimitTransport struct {
	Next    http.RoundTripper
	Options *RatelimitTransportOpts
}

// RatelimitTransportOpts are options for RatelimitTransport
type RatelimitTransportOpts struct{}

// NewRatelimitTransport initializes a new *RatelimitTransport instance
func NewRatelimitTransport(next http.RoundTripper, opts *RatelimitTransportOpts) *RatelimitTransport {
	if opts == nil {
		opts = &RatelimitTransportOpts{}
	}

	t := &RatelimitTransport{
		Next:    next,
		Options: opts,
	}

	return t
}

// RoundTrip implements http.RoundTripper
func (t *RatelimitTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	resp, err := t.Next.RoundTrip(req)
	if err != nil {
		return resp, err
	}

	// Bug: https://github.com/google/go-github/pull/986
	r1, r2, err := DrainBody(resp.Body)
	if err != nil {
		return nil, err
	}
	resp.Body = r1
	err = github.CheckResponse(resp)
	resp.Body = r2

	switch e := err.(type) {
	case *github.AbuseRateLimitError:
		time.Sleep(e.GetRetryAfter())
		return t.Next.RoundTrip(req)
	case *github.RateLimitError:
		time.Sleep(time.Until(e.Rate.Reset.Time))
		return t.Next.RoundTrip(req)
	}

	return resp, nil
}
