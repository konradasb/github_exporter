package transport

import "net/http"

// TransportOpts are options for creating new Transport instances
type TransportOpts struct {
	RatelimitTransportEnabled bool
	RatelimitTransportOpts    *RatelimitTransportOpts

	ThrottleTransportEnabled bool
	ThrottleTransportOpts    *ThrottleTransportOptions

	CacheTransportEnabled bool
	CacheTransportOpts    *CacheTransportOptions
}

// NewTransport initializes Transport instances
func NewTransport(rt http.RoundTripper, opts *TransportOpts) http.RoundTripper {
	if opts == nil {
		opts = &TransportOpts{
			RatelimitTransportEnabled: true,
			RatelimitTransportOpts:    nil,
			ThrottleTransportEnabled:  true,
			ThrottleTransportOpts:     nil,
			CacheTransportEnabled:     true,
			CacheTransportOpts:        nil,
		}
	}

	if rt == nil {
		rt = http.DefaultTransport
	}

	if opts.RatelimitTransportEnabled {
		rt = NewRatelimitTransport(rt, opts.RatelimitTransportOpts)
	}
	if opts.ThrottleTransportEnabled {
		rt = NewThrottleTransport(rt, opts.ThrottleTransportOpts)
	}
	if opts.CacheTransportEnabled {
		rt = NewCacheTransport(rt, opts.CacheTransportOpts)
	}

	return rt
}
