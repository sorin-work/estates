package httpapi

import (
	"net/http"
	"net/mail"
	"strings"
	"unicode/utf8"
)

// validator collects field errors so the client can show them all at once.
type validator struct {
	fields map[string]string
}

func (v *validator) check(ok bool, field, msg string) {
	if ok {
		return
	}
	if v.fields == nil {
		v.fields = map[string]string{}
	}
	if _, exists := v.fields[field]; !exists {
		v.fields[field] = msg
	}
}

func (v *validator) required(value, field string) {
	v.check(strings.TrimSpace(value) != "", field, "is required")
}

func (v *validator) maxLen(value, field string, n int) {
	v.check(utf8.RuneCountInString(value) <= n, field, "is too long")
}

func (v *validator) email(value, field string) {
	if value == "" {
		return
	}
	addr, err := mail.ParseAddress(value)
	v.check(err == nil && addr.Address == value, field, "is not a valid email address")
}

func (v *validator) err() error {
	if len(v.fields) == 0 {
		return nil
	}
	return &apiError{
		Status:  http.StatusUnprocessableEntity,
		Code:    "validation_failed",
		Message: "some fields are invalid",
		Fields:  v.fields,
	}
}

// trim trims surrounding whitespace from every string pointer given.
func trim(fields ...*string) {
	for _, f := range fields {
		*f = strings.TrimSpace(*f)
	}
}
