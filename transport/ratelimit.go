package transport

import (
	"net/http"
	"time"

	"github.com/google/go-github/v42/github"
)

type ratelimitTransport struct {
	Next http.RoundTripper
}

// NewRatelimitTransport creates a new RatelimitTransport instance
//
// RatelimitTransport is responsible for checking responses from
// requests for any Github related ratelimit errors
//
// Upon ratelimit error, the request is retried after the
// recommend time by Github
//
// This only considers Github API secondary limits, also considered
// as abuse rate limits
func NewRatelimitTransport(rt http.RoundTripper) *ratelimitTransport {
	if rt == nil {
		rt = http.DefaultTransport
	}

	t := &ratelimitTransport{
		Next: rt,
	}

	return t
}

// RoundTrip implements http.RoundTripper
func (t *ratelimitTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	resp, err := t.Next.RoundTrip(req)
	if err != nil {
		return resp, err
	}

	r1, r2, err := DrainBody(resp.Body)
	if err != nil {
		return nil, err
	}
	resp.Body = r1
	err = github.CheckResponse(resp)
	resp.Body = r2

	if err != nil {
		e, ok := err.(*github.AbuseRateLimitError)
		if ok {
			<-time.After(e.GetRetryAfter())
			return t.RoundTrip(req)
		}
	}

	return resp, nil
}
