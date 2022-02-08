package validators

import (
	"net/url"
	"regexp"
	"time"
)

// RegexpValidator validates a map of regular expressions (URL Paths)
// and their durations
type regexpValidator struct {
	Regexps map[*regexp.Regexp]time.Duration
}

// NewRegexpValidator creates a new *regexpValidator instance
func NewRegexpValidator(regexps map[*regexp.Regexp]time.Duration) *regexpValidator {
	v := &regexpValidator{
		Regexps: regexps,
	}

	return v
}

// Valid loops through a map of regular expressions.
// Upon successful match, returns true if the URL cache age is less than the maximum allowed
func (v *regexpValidator) Valid(url *url.URL, age time.Duration) bool {
	for re, maxAge := range v.Regexps {
		if re.MatchString(url.Path) {
			return age <= maxAge
		}
	}
	return false
}
