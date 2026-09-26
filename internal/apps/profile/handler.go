package profile

import (
	"net/http"

	"github.com/shobayosamuel/applypilot-backend/internal/platform/httpx"
	"github.com/shobayosamuel/applypilot-backend/internal/platform/middleware"
)

type Handler struct{ svc *Service }

func NewHandler(svc *Service) *Handler { return &Handler{svc: svc} }

func (h *Handler) Routes(mux *http.ServeMux, authed middleware.Func) {
	mux.HandleFunc("GET /api/profile", authed(h.Get))
	mux.HandleFunc("PUT /api/profile", authed(h.Update))
}

func (h *Handler) Get(w http.ResponseWriter, r *http.Request) {
	res, err := h.svc.Get(r.Context(), httpx.UserID(r))
	if err != nil {
		httpx.Error(w, r, err)
		return
	}
	httpx.JSON(w, http.StatusOK, res)
}

func (h *Handler) Update(w http.ResponseWriter, r *http.Request) {
	var req UpdateRequest
	if err := httpx.Decode(w, r, &req, 1<<20); err != nil {
		httpx.Error(w, r, err)
		return
	}
	if err := req.Validate(); err != nil {
		httpx.Error(w, r, err)
		return
	}
	res, err := h.svc.Update(r.Context(), httpx.UserID(r), req)
	if err != nil {
		httpx.Error(w, r, err)
		return
	}
	httpx.JSON(w, http.StatusOK, res)
}
