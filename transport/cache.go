package transport

import (
	"bufio"
	"bytes"
	"net/http"
	"net/http/httputil"

	"github.com/konradasb/github_exporter/cache"
)

// CacheTransportOptions are options for CacheTransport
type CacheTransportOptions struct {
}

// CacheTransport implements http.RoundTripper
type CacheTransport struct {
	Next http.RoundTripper
	Opts *CacheTransportOptions

	Cache cache.Cache
}

// NewCacheTransport initializes a new *CacheTransport instance
func NewCacheTransport(next http.RoundTripper, opts *CacheTransportOptions) *CacheTransport {
	if opts == nil {
		opts = &CacheTransportOptions{}
	}

	cache := cache.NewMemoryCache()

	t := &CacheTransport{
		Cache: cache,
		Next:  next,
		Opts:  opts,
	}

	return t
}

// RoundTrip implements http.RoundTripper
func (t *CacheTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	cacheable := IsRequestCacheable(req)

	var err error
	var cached *http.Response
	if cacheable {
		var req2 *http.Request
		cached, err = t.GetCachedResponse(req)
		if cached != nil && err == nil {
			etag := cached.Header.Get("ETag")
			if etag != "" {
				req2 = CloneRequest(req)
				req2.Header.Set("If-None-Match", etag)
			}
			lastModified := cached.Header.Get("Last-Modified")
			if lastModified != "" {
				if req2 == nil {
					req2 = CloneRequest(req)
				}
				req2.Header.Set("If-Modified-Since", lastModified)
			}
			if req2 != nil {
				req = req2
			}
		}
	}

	resp, err := t.Next.RoundTrip(req)
	if err == nil && req.Method == "GET" && resp.StatusCode == http.StatusNotModified {
		return cached, nil
	} else if err != nil || (cached != nil && resp.StatusCode >= 500) && req.Method == "GET" {
		return cached, nil
	} else {
		if err != nil || resp.StatusCode != http.StatusOK {
			t.Cache.Delete(t.GetCacheKey(req))
		}
		if err != nil {
			return nil, err
		}
	}

	if cacheable {
		buf, err := httputil.DumpResponse(resp, true)
		if err == nil {
			t.Cache.Set(t.GetCacheKey(req), buf)
		}
	}

	return resp, nil
}

func (t *CacheTransport) GetCachedResponse(req *http.Request) (*http.Response, error) {
	item, ok := t.Cache.Get(t.GetCacheKey(req))
	if !ok {
		return nil, nil
	}

	buf := bytes.NewBuffer(item)
	return http.ReadResponse(bufio.NewReader(buf), req)
}

func (t *CacheTransport) GetCacheKey(req *http.Request) string {
	if req.Method == http.MethodGet {
		return req.URL.String()
	}
	return req.Method + " " + req.URL.String()
}
