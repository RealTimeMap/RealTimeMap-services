package utils

import (
	"errors"
	"strings"
)

func SliceString(src string, limit int) string {
	tmp := []rune(src)

	if len(tmp) > limit {
		return string(tmp[:limit])
	}
	return string(tmp)
}

func ExtractEmailDomain(email string) (string, error) {
	parts := strings.Split(email, "@")
	if len(parts) != 2 {
		return "", errors.New("invalid email address")
	}
	domain := parts[1]

	return domain, nil
}
