// Package httpx holds JSON request and response helpers shared by all handlers.
// Error bodies follow the Error schema in docs/api/openapi.yaml.
package httpx

import (
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net/http"
)

type errorBody struct {
	Error errorDetail `json:"error"`
}

type errorDetail struct {
	Code    string            `json:"code"`
	Message string            `json:"message"`
	Fields  map[string]string `json:"fields,omitempty"`
}

// JSON writes v as a JSON response with the given status.
func JSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(v); err != nil {
		slog.Error("encode response", "err", err)
	}
}

// Error writes an error response with a machine-readable code.
func Error(w http.ResponseWriter, status int, code, message string) {
	JSON(w, status, errorBody{Error: errorDetail{Code: code, Message: message}})
}

// ValidationError writes a 400 validation_error with per-field messages.
func ValidationError(w http.ResponseWriter, message string, fields map[string]string) {
	JSON(w, http.StatusBadRequest, errorBody{Error: errorDetail{
		Code: "validation_error", Message: message, Fields: fields,
	}})
}

// Internal logs err and writes a generic 500 that leaks no detail.
func Internal(w http.ResponseWriter, err error) {
	slog.Error("internal error", "err", err)
	Error(w, http.StatusInternalServerError, "internal_error", "internal server error")
}

// DecodeJSON reads exactly one JSON value from the body, rejecting unknown fields,
// trailing data and bodies larger than 1 MiB.
func DecodeJSON(w http.ResponseWriter, r *http.Request, dst any) error {
	dec := json.NewDecoder(http.MaxBytesReader(w, r.Body, 1<<20))
	dec.DisallowUnknownFields()
	if err := dec.Decode(dst); err != nil {
		return err
	}
	if err := dec.Decode(&struct{}{}); !errors.Is(err, io.EOF) {
		return errors.New("request body must contain a single JSON value")
	}
	return nil
}

// IsUUID reports whether s is a canonical 8-4-4-4-12 hexadecimal UUID. Handlers use it
// to turn a malformed id into a 400 instead of a database cast error.
func IsUUID(s string) bool {
	if len(s) != 36 {
		return false
	}
	for i := 0; i < len(s); i++ {
		switch i {
		case 8, 13, 18, 23:
			if s[i] != '-' {
				return false
			}
		default:
			c := s[i]
			if (c < '0' || c > '9') && (c < 'a' || c > 'f') && (c < 'A' || c > 'F') {
				return false
			}
		}
	}
	return true
}
