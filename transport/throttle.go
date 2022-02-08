package transport

import (
	"net/http"

	"go.uber.org/ratelimit"
)

type throttleTransport struct {
	Next    http.RoundTripper
	Limiter ratelimit.Limiter
}

// NewThrottleTransport creates a new ThrottleTransport instance
//
// ThrottleTransport is responsible for throttling requests to the
// configured amount of requests/s
func NewThrottleTransport(rt http.RoundTripper, rtl ratelimit.Limiter) *throttleTransport {
	if rt == nil {
		rt = http.DefaultTransport
	}

	if rtl == nil {
		rtl = ratelimit.New(100)
	}

	t := &throttleTransport{
		Next:    rt,
		Limiter: rtl,
	}

	return t
}

// RoundTrip implements http.RoundTripper
func (t *throttleTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	t.Limiter.Take()

	return t.Next.RoundTrip(req)
}
