package transport

import (
	"bufio"
	"bytes"
	"net/http"
	"net/http/httputil"
	"strconv"

	"github.com/konradasb/github_exporter/cache"
)

type cacheTransport struct {
	Next  http.RoundTripper
	Cache cache.Cache
}

// NewCacheTransport creates a new CacheTransport instance
//
// CacheTransport is responsible for caching responses to the configured
// Cache implementation.
//
// If the response to the request is found in cache, it sets headers
// If-None-Match, If-Modified-Since, X-Cache, X-Cache-Age
//
// If the response received has X-Revalidated headers, the cached
// response age is refreshed to the current time
func NewCacheTransport(rt http.RoundTripper, c cache.Cache) *cacheTransport {
	if rt == nil {
		rt = http.DefaultTransport
	}

	if c == nil {
		c = cache.NewMemoryCache()
	}

	t := &cacheTransport{
		Next:  rt,
		Cache: c,
	}

	return t
}

// RoundTrip implements http.RoundTripper
func (t *cacheTransport) RoundTrip(req *http.Request) (resp *http.Response, err error) {
	cacheable := (req.Method == "GET" || req.Method == "HEAD")

	var item *cache.Item
	var cached *http.Response
	if cacheable {
		item = t.Cache.Get(req.URL.String())
		if item != nil {
			cached, err = http.ReadResponse(bufio.NewReader(bytes.NewBuffer(item.Object.([]byte))), req)
		}
		if cached != nil && err == nil {
			req2 := CloneRequest(req)
			etag := cached.Header.Get("ETag")
			if etag != "" {
				req2.Header.Set("If-None-Match", etag)
			}
			lastModified := cached.Header.Get("Last-Modified")
			if lastModified != "" {
				req2.Header.Set("If-Modified-Since", lastModified)
			}
			req2.Header.Set("X-Cache", "1")
			req2.Header.Set("X-Cache-Age", strconv.Itoa(int(item.GetAge().Seconds())))
			req = req2
		}
	}

	resp, err = t.Next.RoundTrip(req)
	if err == nil && req.Method == "GET" && resp.StatusCode == http.StatusNotModified {
		// Refresh the cache item age, if the 304 (Not Modified) response was not from RevalidationTransport.
		// Meaning the cache item age has expired in RevalidationTransport, the request was still sent, but
		// the received response was still 304 (Not Modified).
		//
		// In this case it's safe to refresh the age of it.
		if resp.Header.Get("X-Revalidated") == "" {
			item.RefreshAge()
		}
		return cached, nil
	} else if err != nil || (cached != nil && resp.StatusCode >= 500) && req.Method == "GET" {
		return cached, nil
	} else {
		if err != nil || resp.StatusCode != http.StatusOK {
			t.Cache.Delete(req.URL.String())
		}
		if err != nil {
			return nil, err
		}
	}

	if cacheable {
		buf, err := httputil.DumpResponse(resp, true)
		if err == nil {
			t.Cache.Set(req.URL.String(), buf)
		}
	}

	return resp, nil
}
