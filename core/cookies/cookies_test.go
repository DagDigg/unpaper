package cookies

import (
	"fmt"
	"strings"
	"testing"
	"time"
)

func TestCookies(t *testing.T) {
	tests := []struct {
		attributes string
		key        string
		wantValue  string
		wantBool   bool
	}{
		{
			attributes: "a=first; b=second;",
			key:        "a",
			wantValue:  "first",
			wantBool:   true,
		},
		{
			attributes: "a=first; b=second;",
			key:        "b",
			wantValue:  "second",
			wantBool:   true,
		},
		{
			attributes: "a=first;",
			key:        "a",
			wantValue:  "first",
			wantBool:   true,
		},
		{
			attributes: "a=first;",
			key:        "b",
			wantValue:  "",
			wantBool:   false,
		},
		{
			attributes: "",
			key:        "b",
			wantValue:  "",
			wantBool:   false,
		},
		{
			attributes: "b",
			key:        "b",
			wantValue:  "",
			wantBool:   false,
		},
		{
			attributes: "b=",
			key:        "b",
			wantValue:  "",
			wantBool:   false,
		},
		{
			attributes: "b=",
			key:        "b",
			wantValue:  "",
			wantBool:   false,
		},
		{
			attributes: "a=first;    b=second    ",
			key:        "b",
			wantValue:  "second",
			wantBool:   true,
		},
	}

	for _, tt := range tests {
		testName := fmt.Sprintf("cookies: %q, key: %q", tt.attributes, tt.key)
		t.Run(testName, func(t *testing.T) {
			ansCookie, ansBool := FindAttribute(tt.attributes, tt.key)
			if ansCookie != tt.wantValue {
				t.Errorf("got: %v, want: %v", ansCookie, tt.wantValue)
			}
			if ansBool != tt.wantBool {
				t.Errorf("got: %v, want: %v", ansBool, tt.wantBool)
			}
		})
	}
}

// Map of timezones
var countryTz = map[string]string{
	"Hungary": "Europe/Budapest",
	"Egypt":   "Africa/Cairo",
}

func TestGetValue(t *testing.T) {
	tests := []struct {
		domain     string
		cookieKey  string
		cookieVal  string
		expiry     time.Duration
		wantStr    string
		wantErrStr string
	}{
		{domain: "foo", cookieKey: "key", cookieVal: "val", expiry: 1 * time.Second, wantStr: "key=val; Domain=foo; Path=/; Max-Age=1; Secure; HttpOnly", wantErrStr: ""},
		{domain: "bar", cookieKey: "", cookieVal: "val", expiry: 2 * time.Minute, wantStr: "", wantErrStr: "missing cookie key or value"},
		{domain: "foo", cookieKey: "key", cookieVal: "", expiry: 2 * time.Minute, wantStr: "", wantErrStr: "missing cookie key or value"},
		{domain: "baz.io", cookieKey: "", cookieVal: "", expiry: 2 * time.Minute, wantStr: "", wantErrStr: "missing cookie key or value"},
	}

	for _, tt := range tests {
		m := &Manager{tt.domain}
		actual, actualErr := m.GetValue(tt.cookieKey, tt.cookieVal, tt.expiry)
		if !errorContains(actualErr, tt.wantErrStr) {
			t.Errorf("error during error validation. Got: '%s', want: '%s'", actualErr.Error(), tt.wantErrStr)
		}
		if tt.wantStr != actual {
			t.Errorf("got: '%v', want: '%v'", actual, tt.wantStr)
		}
	}
}

func TestDeleteCookie(t *testing.T) {
	tests := []struct {
		domain     string
		cookieKey  string
		wantStr    string
		wantErrStr string
	}{
		{domain: "foo", cookieKey: "key", wantStr: "key=deleted; Domain=foo; Path=/; Expires=Thu, 01 Jan 1970 00:00:00 GMT", wantErrStr: ""},
		{domain: "foo.io", cookieKey: "", wantStr: "", wantErrStr: "missing cookie key"},
	}

	for _, tt := range tests {
		m := &Manager{tt.domain}
		actualStr, actualErr := m.DeleteCookie(tt.cookieKey)
		if !errorContains(actualErr, tt.wantErrStr) {
			t.Errorf("error during error validation. Got: '%s', want: '%s'", actualErr.Error(), tt.wantErrStr)
		}
		if tt.wantStr != actualStr {
			t.Errorf("got: '%v', want: '%v'", actualStr, tt.wantStr)
		}
	}
}

// errorContains checks if the error message in out contains the text in want.
// This is safe when out is nil. Use an empty string for want if you want to
// test that err is nil.
func errorContains(out error, want string) bool {
	if out == nil {
		return want == ""
	}
	if want == "" {
		return false
	}
	return strings.Contains(out.Error(), want)
}
