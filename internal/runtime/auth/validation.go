package auth

import (
	"regexp"
	"strings"
)

var emailPattern = regexp.MustCompile("^[A-Za-z0-9.!#$%&'*+/=?^_`{|}~-]+@[A-Za-z0-9](?:[A-Za-z0-9-]*[A-Za-z0-9])?(?:\\.[A-Za-z0-9](?:[A-Za-z0-9-]*[A-Za-z0-9])?)+$")
var uuidPattern = regexp.MustCompile(`^[0-9a-f]{8}-[0-9a-f]{4}-4[0-9a-f]{3}-[89ab][0-9a-f]{3}-[0-9a-f]{12}$`)

func NormalizeEmail(email string) (string, string, error) {
	email = strings.TrimSpace(email)
	if len(email) > 254 || !emailPattern.MatchString(email) {
		return "", "", ErrInvalidRequest
	}
	local, _, _ := strings.Cut(email, "@")
	if len(local) > 64 || strings.HasPrefix(local, ".") || strings.HasSuffix(local, ".") || strings.Contains(local, "..") {
		return "", "", ErrInvalidRequest
	}
	return email, strings.ToLower(email), nil
}
