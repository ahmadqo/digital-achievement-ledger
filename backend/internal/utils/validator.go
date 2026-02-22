package utils

import (
	"encoding/json"
	"net/http"
	"regexp"
	"strings"
)

// DecodeJSON decode request body ke struct
func DecodeJSON(r *http.Request, dst interface{}) error {
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	return decoder.Decode(dst)
}

// ValidationErrors map field -> pesan error
type ValidationErrors map[string]string

func (ve ValidationErrors) HasErrors() bool {
	return len(ve) > 0
}

var emailRegex = regexp.MustCompile(`^[a-zA-Z0-9._%+\-]+@[a-zA-Z0-9.\-]+\.[a-zA-Z]{2,}$`)

func IsValidEmail(email string) bool {
	return emailRegex.MatchString(email)
}

func IsValidPassword(password string) bool {
	// Minimal 8 karakter, ada huruf dan angka
	if len(password) < 8 {
		return false
	}
	hasLetter := regexp.MustCompile(`[a-zA-Z]`).MatchString(password)
	hasDigit := regexp.MustCompile(`[0-9]`).MatchString(password)
	return hasLetter && hasDigit
}

func SanitizeString(s string) string {
	return strings.TrimSpace(s)
}