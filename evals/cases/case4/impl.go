//go:build ignore

package validation

import "strings"

func IsValidEmail(email string) bool {
	return strings.Contains(email, "@")
}

func IsPositive(num int) bool {
	return num >= 0
}
