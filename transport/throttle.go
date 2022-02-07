package transport

import (
	"net/http"

	"go.uber.org/ratelimit"
)

// ThrottleTransportOptions are options for ThrottleTransport
type ThrottleTransportOptions struct {
	RequestsPerSecond int
}

// ThrottleTransport implements http.RoundTripper
type ThrottleTransport struct {
	Next    http.RoundTripper
	Opts    *ThrottleTransportOptions
	Limiter ratelimit.Limiter
}

// NewThrottleTransport initializes a new *ThrottleTransport instance
func NewThrottleTransport(next http.RoundTripper, opts *ThrottleTransportOptions) *ThrottleTransport {
	if opts == nil {
		opts = &ThrottleTransportOptions{
			RequestsPerSecond: 100,
		}
	}

	rtl := ratelimit.New(opts.RequestsPerSecond)

	t := &ThrottleTransport{
		Opts:    opts,
		Next:    next,
		Limiter: rtl,
	}

	return t
}

// RoundTrip implements http.RoundTripper
func (t *ThrottleTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	t.Limiter.Take()

	return t.Next.RoundTrip(req)
}
