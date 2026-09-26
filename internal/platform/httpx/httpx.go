// Package httpx holds small HTTP helpers shared by every app's handler.
package httpx

import (
	"context"
	"encoding/json"
	"log"
	"net/http"

	"github.com/shobayosamuel/applypilot-backend/internal/platform/apperr"
)

func JSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(v)
}

func NoContent(w http.ResponseWriter) { w.WriteHeader(http.StatusNoContent) }

func ErrorMsg(w http.ResponseWriter, status int, msg string) {
	JSON(w, status, map[string]string{"error": msg})
}

var statusByKind = map[apperr.Kind]int{
	apperr.KindInvalid:      http.StatusBadRequest,
	apperr.KindUnauthorized: http.StatusUnauthorized,
	apperr.KindNotFound:     http.StatusNotFound,
	apperr.KindConflict:     http.StatusConflict,
	apperr.KindRateLimited:  http.StatusTooManyRequests,
	apperr.KindUnavailable:  http.StatusServiceUnavailable,
	apperr.KindUpstream:     http.StatusBadGateway,
}

// Error writes err as {"error": "..."} with the status for its kind.
// Untyped errors are logged and returned as a generic 500.
func Error(w http.ResponseWriter, r *http.Request, err error) {
	kind := apperr.KindOf(err)
	status, ok := statusByKind[kind]
	if !ok {
		log.Printf("%s %s: %v", r.Method, r.URL.Path, err)
		ErrorMsg(w, http.StatusInternalServerError, "internal error")
		return
	}
	if kind == apperr.KindUpstream || kind == apperr.KindUnavailable {
		log.Printf("%s %s: %v", r.Method, r.URL.Path, err)
	}
	ErrorMsg(w, status, apperr.Message(err))
}

// Decode reads a JSON body of at most maxBytes into v.
func Decode(w http.ResponseWriter, r *http.Request, v any, maxBytes int64) error {
	r.Body = http.MaxBytesReader(w, r.Body, maxBytes)
	if err := json.NewDecoder(r.Body).Decode(v); err != nil {
		return apperr.Invalid("invalid JSON body: " + err.Error())
	}
	return nil
}

type ctxKey struct{}

func WithUserID(ctx context.Context, id string) context.Context {
	return context.WithValue(ctx, ctxKey{}, id)
}

// UserID returns the authenticated user's id (set by middleware.Auth).
func UserID(r *http.Request) string {
	id, _ := r.Context().Value(ctxKey{}).(string)
	return id
}
