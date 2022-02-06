package ratelimit

import (
	"bufio"
	"bytes"
	"log"
	"net/http"
	"net/http/httputil"
	"time"

	"github.com/google/go-github/v42/github"
	"github.com/patrickmn/go-cache"
	"go.uber.org/ratelimit"
)

const (
	// DefaultRequestsPerSecond default value used for rate limiting RPS
	defaultRequestsPerSecond = 50
	// DefaultCacheExpiresTime default value for cache expires time
	defaultCacheExpiresTime = 60 * time.Second
	// DefaultCacheCleanupTime default value for cache cleanup time
	defaultCacheCleanupTime = 60 * time.Second
)

// RatelimitTransportOption option for RateLimitTransport
type RatelimitTransportOption func(*ratelimitTransport)

type ratelimitTransport struct {
	next  http.RoundTripper
	rtl   ratelimit.Limiter
	cache *cache.Cache
}

// NewRatelimitTransport initializes a new *ratelimitTransport instance
func NewRatelimitTransport(next http.RoundTripper, opts ...RatelimitTransportOption) *ratelimitTransport {
	t := &ratelimitTransport{
		cache: cache.New(defaultCacheExpiresTime, defaultCacheCleanupTime),
		rtl:   ratelimit.New(defaultRequestsPerSecond),
		next:  next,
	}

	for _, opt := range opts {
		opt(t)
	}

	return t
}

// WithCache allows to specify cache instance for *ratelimitTransport
func WithCache(cache *cache.Cache) RatelimitTransportOption {
	return func(rt *ratelimitTransport) {
		rt.cache = cache
	}
}

// WithLimiter allows to specify limiter instance for *ratelimitTransport
func WithLimiter(rtl ratelimit.Limiter) RatelimitTransportOption {
	return func(rt *ratelimitTransport) {
		rt.rtl = rtl
	}
}

// RoundTrip implements http.RoundTripper
func (t *ratelimitTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	cache, _ := t.cache.Get(req.URL.String())
	if cache != nil {
		buf := bytes.NewBuffer(cache.([]byte))
		return http.ReadResponse(bufio.NewReader(buf), nil)
	}

	t.rtl.Take()

	resp, err := t.next.RoundTrip(req)
	if err != nil {
		return resp, err
	}

	buf, err := httputil.DumpResponse(resp, true)
	if err != nil {
		return resp, err
	}

	t.cache.Set(req.URL.String(), buf, 0)

	err = github.CheckResponse(resp)
	switch e := err.(type) {
	case *github.AbuseRateLimitError:
		log.Printf("[DEBUG>ABUSERATELIMIT] Sleeping %s before performing other requests", e.GetRetryAfter())
		time.Sleep(e.GetRetryAfter())
		return t.next.RoundTrip(req)
	case *github.RateLimitError:
		log.Printf("[DEBUG>RATELIMIT] Sleeping %s before performing other requests", time.Until(e.Rate.Reset.Time))
		time.Sleep(time.Until(e.Rate.Reset.Time))
		return t.next.RoundTrip(req)
	}

	return resp, nil
}
