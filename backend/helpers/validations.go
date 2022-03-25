package helpers

import (
	"net"
	"regexp"
	"strings"
)

// IsEmailValid checks if the email provided passes the required structure
// and length test. It also checks the domain has a valid MX record.
func IsEmailValid(e string) (bool, error) {
	emailRegex, err := regexp.Compile(
		"^[a-zA-Z0-9.!#$%&'*+\\/=?^_`{|}~-]+@[a-zA-Z0-9](?:[a-zA-Z0-9-]{0,61}[a-zA-Z0-9])?(?:\\.[a-zA-Z0-9](?:[a-zA-Z0-9-]{0,61}[a-zA-Z0-9])?)*$",
	)
	if err != nil {
		return false, err
	}

	if len(e) < 3 && len(e) > 254 {
		return false, nil
	}
	if !emailRegex.MatchString(e) {
		return false, nil
	}
	parts := strings.Split(e, "@")
	mx, err := net.LookupMX(parts[1])
	if err != nil || len(mx) == 0 {
		return false, err
	}

	return true, nil
}

// StringSliceContains returns whether a slice of strings
// contains the param passed string
func StringSliceContains(s []string, v string) bool {
	res := false
	for _, el := range s {
		if el == v {
			res = true
		}
	}

	return res
}
