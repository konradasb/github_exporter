package validators

import (
	"net/url"
	"time"
)

// Validators are used to validate if the cache entry for the given URL is still
// valid at a certain age.
type Validator interface {
	Valid(url *url.URL, age time.Duration) bool
}
