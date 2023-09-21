// Package stringutil provides helper functions for string
package stringutil

import "unicode/utf8"

// Truncate truncates a string to the specified length.
func Truncate(str string, length int) string {
	if length <= 0 {
		return ""
	}

	if utf8.RuneCountInString(str) <= length {
		return str
	}

	return string([]rune(str)[:length])
}
