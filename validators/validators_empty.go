package validators

import (
	"net/url"
	"time"
)

// EmptyValidator implements validators.Validator interface. Generally it is
// used for tests and default values
type emptyValidator struct{}

// NewEmptyValidator creates a new *emptyValidator instance
func NewEmptyValidator() *emptyValidator {
	return &emptyValidator{}
}

// Valid doesn't do anything and always returns true
func (v *emptyValidator) Valid(url *url.URL, age time.Duration) bool {
	return true
}
