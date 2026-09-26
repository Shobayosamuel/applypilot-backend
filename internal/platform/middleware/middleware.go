// Package middleware contains the HTTP middleware shared by all apps.
package middleware

import (
	"log"
	"net"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/shobayosamuel/applypilot-backend/internal/platform/httpx"
	"github.com/shobayosamuel/applypilot-backend/internal/platform/security"
)

// Func wraps a handler func (used for per-route middleware such as Auth).
type Func func(http.HandlerFunc) http.HandlerFunc

// Auth requires a valid bearer token and stores the user id in the context.
func Auth(tokens *security.Tokens) Func {
	return func(next http.HandlerFunc) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			tok, ok := strings.CutPrefix(r.Header.Get("Authorization"), "Bearer ")
			if !ok || tok == "" {
				httpx.ErrorMsg(w, http.StatusUnauthorized, "missing bearer token")
				return
			}
			uid, err := tokens.Verify(tok)
			if err != nil {
				httpx.ErrorMsg(w, http.StatusUnauthorized, "invalid or expired token")
				return
			}
			next(w, r.WithContext(httpx.WithUserID(r.Context(), uid)))
		}
	}
}

// CORS allows the configured dashboard origins and any chrome-extension:// origin.
func CORS(allowed []string) func(http.Handler) http.Handler {
	ok := func(o string) bool {
		if strings.HasPrefix(o, "chrome-extension://") {
			return true
		}
		for _, a := range allowed {
			if a == "*" || a == o {
				return true
			}
		}
		return false
	}
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if o := r.Header.Get("Origin"); o != "" && ok(o) {
				h := w.Header()
				h.Set("Access-Control-Allow-Origin", o)
				h.Set("Vary", "Origin")
				h.Set("Access-Control-Allow-Headers", "Authorization, Content-Type")
				h.Set("Access-Control-Allow-Methods", "GET, POST, PUT, PATCH, DELETE, OPTIONS")
				h.Set("Access-Control-Expose-Headers", "X-Resume-Filename, Content-Disposition")
				h.Set("Access-Control-Max-Age", "600")
			}
			if r.Method == http.MethodOptions {
				w.WriteHeader(http.StatusNoContent)
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

type statusWriter struct {
	http.ResponseWriter
	status int
}

func (w *statusWriter) WriteHeader(c int) { w.status = c; w.ResponseWriter.WriteHeader(c) }

func Logger(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		sw := &statusWriter{ResponseWriter: w, status: 200}
		next.ServeHTTP(sw, r)
		log.Printf("%s %s %d %s", r.Method, r.URL.Path, sw.status, time.Since(start).Round(time.Millisecond))
	})
}

func Recover(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if v := recover(); v != nil {
				log.Printf("panic: %v", v)
				httpx.ErrorMsg(w, http.StatusInternalServerError, "internal error")
			}
		}()
		next.ServeHTTP(w, r)
	})
}

// RateLimit is a fixed-window per-IP limiter (in memory; fine for one instance).
func RateLimit(max int, window time.Duration, trustProxy bool) Func {
	var mu sync.Mutex
	type win struct {
		start time.Time
		n     int
	}
	hits := map[string]*win{}
	return func(next http.HandlerFunc) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			ip, _, _ := net.SplitHostPort(r.RemoteAddr)
			if fwd := r.Header.Get("X-Forwarded-For"); trustProxy && fwd != "" {
				ip = strings.TrimSpace(strings.Split(fwd, ",")[0])
			}
			now := time.Now()
			mu.Lock()
			if len(hits) > 10000 {
				for k, v := range hits {
					if now.Sub(v.start) > window {
						delete(hits, k)
					}
				}
			}
			h, ok := hits[ip]
			if !ok || now.Sub(h.start) > window {
				h = &win{start: now}
				hits[ip] = h
			}
			h.n++
			allowed := h.n <= max
			mu.Unlock()
			if !allowed {
				httpx.ErrorMsg(w, http.StatusTooManyRequests, "too many attempts, try again in a minute")
				return
			}
			next(w, r)
		}
	}
}
