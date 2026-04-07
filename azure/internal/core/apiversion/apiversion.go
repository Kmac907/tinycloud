package apiversion

import (
	"errors"
	"regexp"
	"strings"
)

var versionPattern = regexp.MustCompile(`^\d{4}-\d{2}-\d{2}(-preview)?$`)

var ErrMissing = errors.New("missing api-version")
var ErrInvalid = errors.New("invalid api-version")

func Parse(raw string) (string, error) {
	value := strings.TrimSpace(raw)
	if value == "" {
		return "", ErrMissing
	}
	if !versionPattern.MatchString(value) {
		return "", ErrInvalid
	}
	return value, nil
}
