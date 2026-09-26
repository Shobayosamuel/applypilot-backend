package ai

import (
	"context"
	"net/http"

	"github.com/shobayosamuel/applypilot-backend/internal/platform/httpx"
	"github.com/shobayosamuel/applypilot-backend/internal/platform/middleware"
)

type Handler struct{ svc *Service }

func NewHandler(svc *Service) *Handler { return &Handler{svc: svc} }

func (h *Handler) Routes(mux *http.ServeMux, authed middleware.Func) {
	mux.HandleFunc("POST /api/ai/parse-resume", authed(handle(h.svc.ParseResume, 4<<10)))
	mux.HandleFunc("POST /api/ai/match", authed(handle(h.svc.Match, 1<<20)))
	mux.HandleFunc("POST /api/ai/cover-letter", authed(handle(h.svc.CoverLetter, 1<<20)))
	mux.HandleFunc("POST /api/ai/tailor-resume", authed(handle(h.svc.TailorResume, 1<<20)))
	mux.HandleFunc("POST /api/ai/answer", authed(handle(h.svc.Answer, 1<<20)))
	mux.HandleFunc("POST /api/ai/map-fields", authed(handle(h.svc.MapFields, 1<<20)))
	mux.HandleFunc("POST /api/ai/extract-job", authed(handle(h.svc.ExtractJob, 2<<20)))
	mux.HandleFunc("GET /api/ai/usage", authed(h.Usage))
	mux.HandleFunc("GET /api/ai/generations", authed(h.Generations))
}

// validatable is implemented by every request type (pointer receiver).
type validatable[T any] interface {
	*T
	Validate() error
}

// handle adapts a service method to an HTTP handler: decode, validate, call, respond.
func handle[Req any, PReq validatable[Req], Res any](fn func(context.Context, string, Req) (*Res, error), maxBytes int64) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req Req
		if err := httpx.Decode(w, r, &req, maxBytes); err != nil {
			httpx.Error(w, r, err)
			return
		}
		if err := PReq(&req).Validate(); err != nil {
			httpx.Error(w, r, err)
			return
		}
		res, err := fn(r.Context(), httpx.UserID(r), req)
		if err != nil {
			httpx.Error(w, r, err)
			return
		}
		httpx.JSON(w, http.StatusOK, res)
	}
}

func (h *Handler) Usage(w http.ResponseWriter, r *http.Request) {
	res, err := h.svc.Usage(r.Context(), httpx.UserID(r))
	if err != nil {
		httpx.Error(w, r, err)
		return
	}
	httpx.JSON(w, http.StatusOK, res)
}

func (h *Handler) Generations(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	res, err := h.svc.Generations(r.Context(), httpx.UserID(r), q.Get("job_id"), q.Get("kind"))
	if err != nil {
		httpx.Error(w, r, err)
		return
	}
	httpx.JSON(w, http.StatusOK, res)
}
