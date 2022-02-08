package transport

import (
	"net/http"

	"github.com/konradasb/github_exporter/cache"
	"github.com/konradasb/github_exporter/validators"
	"go.uber.org/ratelimit"
)

type transport struct {
	Next http.RoundTripper
}

// NewTransport initializes Transport instances
func NewTransport(rt http.RoundTripper) *transport {
	if rt == nil {
		rt = http.DefaultTransport
	}

	t := &transport{
		Next: rt,
	}

	return t
}

func (t *transport) RoundTrip(req *http.Request) (*http.Response, error) {
	return t.Next.RoundTrip(req)
}

func (t *transport) WithRevalidation(v validators.Validator) *transport {
	t.Next = NewRevalidationTransport(t.Next, v)
	return t
}

func (t *transport) WithThrottle(rtl ratelimit.Limiter) *transport {
	t.Next = NewThrottleTransport(t.Next, rtl)
	return t
}

func (t *transport) WithCache(cache cache.Cache) *transport {
	t.Next = NewCacheTransport(t.Next, cache)
	return t
}

func (t *transport) WithRatelimit() *transport {
	t.Next = NewRatelimitTransport(t.Next)
	return t
}
