package validators

import (
	"net/url"
	"regexp"
	"testing"
	"time"
)

func TestRegexpValidator(t *testing.T) {
	v := NewRegexpValidator(
		map[*regexp.Regexp]time.Duration{
			regexp.MustCompile(`^/foobar$`): 5 * time.Second,
		},
	)

	tests := []struct {
		url   string
		age   time.Duration
		valid bool
	}{
		{"/foobar", 10 * time.Second, false},
		{"/foobar", 5 * time.Second, true},
	}

	for _, test := range tests {
		x := v.Valid(&url.URL{Path: test.url}, test.age)
		if x != test.valid {
			t.Errorf("url %s age %s : want == %v got == %v", test.url, test.age, test.valid, x)
		}
	}
}

func TestEmptyValidator(t *testing.T) {
	v := NewEmptyValidator()

	tests := []struct {
		url   string
		age   time.Duration
		valid bool
	}{
		{"/foobar", 10 * time.Second, true},
	}

	for _, test := range tests {
		x := v.Valid(&url.URL{Path: test.url}, test.age)
		if x != test.valid {
			t.Errorf("url %s age %s : want == %v got == %v", test.url, test.age, test.valid, x)
		}
	}
}
