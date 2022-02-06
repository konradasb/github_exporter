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
	defaultCacheExpiresTime = 600 * time.Second
	// DefaultCacheCleanupTime default value for cache cleanup time
	defaultCacheCleanupTime = 60 * time.Second
)

// TransportOptions are options for Transport
type TransportOptions struct {
	RatelimitEnabled bool
	RatelimitRPS     int
	CacheExpiresTime time.Duration
	CacheCleanupTime time.Duration
	CacheEnabled     bool
}

// Transport implements http.RoundTripper
//
// It serves as a wrapper for requests, which implements caching
// of responses and rate limiting total amount of requests/s
type Transport struct {
	Next    http.RoundTripper
	Options *TransportOptions

	cache *cache.Cache
	rtl   ratelimit.Limiter
}

// NewRatelimitTransport initializes a new *ratelimitTransport instance
func NewTransport(next http.RoundTripper, options *TransportOptions) *Transport {
	t := &Transport{}

	if options == nil {
		options = &TransportOptions{
			RatelimitRPS:     defaultRequestsPerSecond,
			RatelimitEnabled: true,
			CacheExpiresTime: defaultCacheExpiresTime,
			CacheCleanupTime: defaultCacheCleanupTime,
			CacheEnabled:     true,
		}
	}
	t.Options = options

	if next == nil {
		next = http.DefaultTransport
	}
	t.Next = next

	if options.CacheEnabled {
		t.cache = cache.New(t.Options.CacheExpiresTime, t.Options.CacheCleanupTime)
	}

	if options.RatelimitEnabled {
		t.rtl = ratelimit.New(t.Options.RatelimitRPS)
	}

	return t
}

// RoundTrip implements http.RoundTripper
func (t *Transport) RoundTrip(req *http.Request) (*http.Response, error) {
	if t.Options.CacheEnabled {
		cache, _ := t.cache.Get(req.URL.String())
		if cache != nil {
			buf := bytes.NewBuffer(cache.([]byte))
			return http.ReadResponse(bufio.NewReader(buf), nil)
		}
	}

	if t.Options.RatelimitEnabled {
		t.rtl.Take()
	}

	resp, err := t.Next.RoundTrip(req)
	if err != nil {
		return resp, err
	}

	if t.Options.CacheEnabled {
		buf, err := httputil.DumpResponse(resp, true)
		if err != nil {
			return resp, err
		}

		t.cache.Set(req.URL.String(), buf, 0)
	}

	err = github.CheckResponse(resp)
	switch e := err.(type) {
	case *github.AbuseRateLimitError:
		log.Printf("[DEBUG>ABUSERATELIMIT] Sleeping %s before performing other requests", e.GetRetryAfter())
		time.Sleep(e.GetRetryAfter())
		return t.Next.RoundTrip(req)
	case *github.RateLimitError:
		log.Printf("[DEBUG>RATELIMIT] Sleeping %s before performing other requests", time.Until(e.Rate.Reset.Time))
		time.Sleep(time.Until(e.Rate.Reset.Time))
		return t.Next.RoundTrip(req)
	}

	return resp, nil
}
