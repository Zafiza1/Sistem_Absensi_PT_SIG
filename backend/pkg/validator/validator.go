// Package validator provides one shared go-playground/validator instance
// and turns its errors into the field->message maps used by
// pkg/response.Fail, so every handler formats validation errors the same
// way for Flutter/Next.js clients.
package validator

import (
	"strings"

	"github.com/go-playground/validator/v10"
)

// Validate is the single validator instance used across all request DTOs.
var Validate = validator.New(validator.WithRequiredStructEnabled())

// FormatErrors turns a validator.ValidationErrors into a
// {"field": "human-readable message"} map. If err is not a
// ValidationErrors (e.g. it came from JSON decoding instead), it is
// returned as a single "_" entry so callers never lose the detail.
func FormatErrors(err error) map[string]string {
	out := map[string]string{}

	verrs, ok := err.(validator.ValidationErrors)
	if !ok {
		out["_"] = err.Error()
		return out
	}

	for _, fe := range verrs {
		field := strings.ToLower(fe.Field())
		out[field] = messageFor(fe)
	}
	return out
}

func messageFor(fe validator.FieldError) string {
	switch fe.Tag() {
	case "required":
		return "wajib diisi"
	case "email":
		return "format email tidak valid"
	case "min":
		return "minimal " + fe.Param() + " karakter"
	case "max":
		return "maksimal " + fe.Param() + " karakter"
	default:
		return "tidak valid"
	}
}
