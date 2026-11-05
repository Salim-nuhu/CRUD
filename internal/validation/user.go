package validation

import (
	// mail parses and checks the syntax of email addresses.
	"net/mail"
	// strings normalizes and inspects text values.
	"strings"
	// utf8 counts Unicode characters and validates password length.
	"unicode/utf8"
)

// These limits define acceptable input sizes for user fields.
const (
	// minNameLen is the fewest Unicode characters allowed in a name.
	minNameLen = 2
	// maxNameLen is the most Unicode characters allowed in a name.
	maxNameLen = 100
	// maxEmailLen follows the maximum size accepted for an email address.
	maxEmailLen = 254
	// minPasswordLen is the fewest Unicode characters allowed in a password.
	minPasswordLen = 8
	// maxPasswordBytes matches bcrypt's maximum input size.
	maxPasswordBytes = 72
)

// Errors maps a field name to what is wrong with it.
type Errors map[string]string

// User cleans and validates the fields shared by create and update.
// It returns the cleaned values plus any problems found (empty = valid).
func User(name, email string) (string, string, Errors) {
	// Start with an empty map that will collect any field-specific failures.
	errs := Errors{}

	// Remove surrounding whitespace from a name and canonicalize email casing.
	name = strings.TrimSpace(name)
	email = strings.ToLower(strings.TrimSpace(email))

	// Count runes, not bytes, so names like "Ama Ñ" or "Kwàmé" count fairly.
	switch n := utf8.RuneCountInString(name); {
	case n == 0:
		// Require a non-empty name after trimming.
		errs["name"] = "name is required"
	case n < minNameLen:
		// Reject names shorter than the minimum Unicode-character count.
		errs["name"] = "name must be at least 2 characters"
	case n > maxNameLen:
		// Reject names longer than the configured maximum.
		errs["name"] = "name must be at most 100 characters"
	}

	// Validate the normalized email in order from required to format checks.
	switch {
	case email == "":
		// Require an email value.
		errs["email"] = "email is required"
	case len(email) > maxEmailLen:
		// Reject an address that exceeds the maximum byte length.
		errs["email"] = "email is too long"
	case !validEmail(email):
		// Reject values that do not match the application's email rules.
		errs["email"] = "email is not valid"
	}

	// Return the cleaned values even when errors were found for caller reporting.
	return name, email, errs
}

// validEmail accepts a simple address only when parsing returns the same text.
func validEmail(email string) bool {
	// ParseAddress checks syntax but may also accept a display name wrapper.
	addr, err := mail.ParseAddress(email)
	// ParseAddress also accepts things like "Bob <bob@x.com>".
	// Requiring the parsed address to equal the input rejects those.
	if err != nil || addr.Address != email {
		// Reject malformed addresses and display-name forms.
		return false
	}
	// "a@b" is technically parseable but is never a real public address.
	at := strings.LastIndex(email, "@")
	// Require a dot somewhere after the at-sign as a basic domain check.
	return strings.Contains(email[at+1:], ".")
}

// Password returns a problem description, or "" if the password is fine.
// We do NOT trim it: spaces are legitimate password characters.
func Password(password string) string {
	// Check the Unicode character minimum and bcrypt byte maximum.
	switch {
	case utf8.RuneCountInString(password) < minPasswordLen:
		// Explain the minimum password length requirement.
		return "password must be at least 8 characters"
	case len(password) > maxPasswordBytes:
		// Explain the maximum accepted password byte length.
		return "password must be at most 72 bytes"
	}
	// An empty message means the password passed these checks.
	return ""
}
