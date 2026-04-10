package flux

import (
	"html"
	"regexp"
	"strings"
)

// Pre-compiled regular expressions — compiled once at startup, never again.
var (
	emailRegexpV = regexp.MustCompile(`^[a-zA-Z0-9._%+\-]+@[a-zA-Z0-9.\-]+\.[a-zA-Z]{2,}$`)
	urlRegexpV   = regexp.MustCompile(`^https?://[^\s/$.?#].[^\s]*$`)
	uuidRegexpV  = regexp.MustCompile(`^[a-f0-9]{8}-[a-f0-9]{4}-4[a-f0-9]{3}-[89ab][a-f0-9]{3}-[a-f0-9]{12}$`)
)

// IsValidEmail reports whether email is a properly formatted email address.
func IsValidEmail(email string) bool {
	return emailRegexpV.MatchString(email)
}

// IsValidURL reports whether url begins with http:// or https:// and is
// syntactically plausible.
func IsValidURL(url string) bool {
	return urlRegexpV.MatchString(url)
}

// IsValidUUID reports whether uuid is a canonical UUID v4 string
// (case-insensitive).
func IsValidUUID(uuid string) bool {
	return uuidRegexpV.MatchString(strings.ToLower(uuid))
}

// Sanitize HTML-escapes input to prevent XSS when the value is rendered in
// an HTML context. It converts <, >, &, ', and " to their HTML entities
// using the Go standard library — safer than a hand-rolled blocklist.
func Sanitize(input string) string {
	return html.EscapeString(input)
}

// TrimInput removes leading and trailing whitespace.
func TrimInput(input string) string {
	return strings.TrimSpace(input)
}

// ValidateStringLength reports whether the trimmed length of input falls
// within [minLen, maxLen] inclusive.
func ValidateStringLength(input string, minLen, maxLen int) bool {
	length := len(strings.TrimSpace(input))
	return length >= minLen && length <= maxLen
}

// ValidateRequired reports whether input is non-empty after trimming.
func ValidateRequired(input string) bool {
	return strings.TrimSpace(input) != ""
}
