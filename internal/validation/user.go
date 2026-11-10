package validation

import (
	"net/mail"
	"regexp"
	"strings"
	"unicode/utf8"

	"github.com/Salim-nuhu/CRUD/internal/models"
)

const (
	minNameLen       = 2
	maxNameLen       = 100
	maxEmailLen      = 254
	minPasswordLen   = 8
	maxPasswordBytes = 72
)

// Errors maps a field name to what is wrong with it.
type Errors map[string]string

// User cleans and validates the fields shared by create and update.
// It returns the cleaned values plus any problems found (empty = valid).
func User(name, email string) (string, string, Errors) {
	errs := Errors{}

	name = strings.TrimSpace(name)
	email = strings.ToLower(strings.TrimSpace(email))

	// Count runes, not bytes, so names like "Ama Ñ" or "Kwàmé" count fairly.
	switch n := utf8.RuneCountInString(name); {
	case n == 0:
		errs["name"] = "name is required"
	case n < minNameLen:
		errs["name"] = "name must be at least 2 characters"
	case n > maxNameLen:
		errs["name"] = "name must be at most 100 characters"
	}

	switch {
	case email == "":
		errs["email"] = "email is required"
	case len(email) > maxEmailLen:
		errs["email"] = "email is too long"
	case !validEmail(email):
		errs["email"] = "email is not valid"
	}

	return name, email, errs
}

func validEmail(email string) bool {
	addr, err := mail.ParseAddress(email)
	// ParseAddress also accepts things like "Bob <bob@x.com>".
	// Requiring the parsed address to equal the input rejects those.
	if err != nil || addr.Address != email {
		return false
	}
	// "a@b" is technically parseable but is never a real public address.
	at := strings.LastIndex(email, "@")
	return strings.Contains(email[at+1:], ".")
}

// Password returns a problem description, or "" if the password is fine.
// We do NOT trim it: spaces are legitimate password characters.
func Password(password string) string {
	switch {
	case utf8.RuneCountInString(password) < minPasswordLen:
		return "password must be at least 8 characters"
	case len(password) > maxPasswordBytes:
		return "password must be at most 72 bytes"
	}
	return ""
}

// Role accepts only the roles a person may pick for themselves.
// Admin is deliberately NOT allowed here.
func Role(role string) (string, string) {
	role = strings.ToLower(strings.TrimSpace(role))
	switch role {
	case models.RoleBuyer, models.RoleOwner:
		return role, ""
	case "":
		return "", "role is required (buyer or owner)"
	default:
		return "", "role must be buyer or owner"
	}
}

// phoneRe: an optional +, then 7 to 15 digits (15 is the international maximum).
var phoneRe = regexp.MustCompile(`^\+?[0-9]{7,15}$`)

// phoneCleaner removes the separators people like to type.
var phoneCleaner = strings.NewReplacer(" ", "", "-", "", "(", "", ")", "", ".", "")

// Phone cleans a phone number and checks its shape. It returns the cleaned
// number and a problem description ("" if fine).
func Phone(phone string, required bool) (string, string) {
	phone = phoneCleaner.Replace(strings.TrimSpace(phone))

	if phone == "" {
		if required {
			return "", "phone is required"
		}
		return "", ""
	}
	if !phoneRe.MatchString(phone) {
		return "", "phone must be 7 to 15 digits, optionally starting with +"
	}
	return phone, ""
}
