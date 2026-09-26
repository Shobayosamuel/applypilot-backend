package auth

import (
	"net/http"

	"github.com/shobayosamuel/applypilot-backend/internal/platform/httpx"
	"github.com/shobayosamuel/applypilot-backend/internal/platform/middleware"
)

type Handler struct{ svc *Service }

func NewHandler(svc *Service) *Handler { return &Handler{svc: svc} }

// Routes registers the auth endpoints. limit throttles signup/login per IP.
func (h *Handler) Routes(mux *http.ServeMux, authed, limit middleware.Func) {
	mux.HandleFunc("POST /api/auth/signup", limit(h.Signup))
	mux.HandleFunc("POST /api/auth/login", limit(h.Login))
	mux.HandleFunc("GET /api/me", authed(h.Me))
}

func (h *Handler) Signup(w http.ResponseWriter, r *http.Request) {
	var req SignupRequest
	if err := httpx.Decode(w, r, &req, 16<<10); err != nil {
		httpx.Error(w, r, err)
		return
	}
	if err := req.Validate(); err != nil {
		httpx.Error(w, r, err)
		return
	}
	res, err := h.svc.Signup(r.Context(), req)
	if err != nil {
		httpx.Error(w, r, err)
		return
	}
	httpx.JSON(w, http.StatusCreated, res)
}

func (h *Handler) Login(w http.ResponseWriter, r *http.Request) {
	var req LoginRequest
	if err := httpx.Decode(w, r, &req, 16<<10); err != nil {
		httpx.Error(w, r, err)
		return
	}
	if err := req.Validate(); err != nil {
		httpx.Error(w, r, err)
		return
	}
	res, err := h.svc.Login(r.Context(), req)
	if err != nil {
		httpx.Error(w, r, err)
		return
	}
	httpx.JSON(w, http.StatusOK, res)
}

func (h *Handler) Me(w http.ResponseWriter, r *http.Request) {
	res, err := h.svc.Me(r.Context(), httpx.UserID(r))
	if err != nil {
		httpx.Error(w, r, err)
		return
	}
	httpx.JSON(w, http.StatusOK, res)
}
