package validate

import (
	"fmt"
	"net/mail"
	"strings"
	"unicode/utf8"
)

type Errors map[string]string

func (e Errors) HasErrors() bool {
	return len(e) > 0
}

func (e Errors) Error() string {
	parts := make([]string, 0, len(e))
	for field, msg := range e {
		parts = append(parts, fmt.Sprintf("%s: %s", field, msg))
	}
	return strings.Join(parts, "; ")
}

func Required(errs Errors, field, value string) {
	if strings.TrimSpace(value) == "" {
		errs[field] = "is required"
	}
}

func MinLength(errs Errors, field, value string, min int) {
	if utf8.RuneCountInString(value) < min {
		errs[field] = fmt.Sprintf("must be at least %d characters", min)
	}
}

func MaxLength(errs Errors, field, value string, max int) {
	if utf8.RuneCountInString(value) > max {
		errs[field] = fmt.Sprintf("must be at most %d characters", max)
	}
}

func Email(errs Errors, field, value string) {
	if value == "" {
		return
	}
	if _, err := mail.ParseAddress(value); err != nil {
		errs[field] = "is not a valid email"
	}
}
