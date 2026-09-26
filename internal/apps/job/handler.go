package job

import (
	"net/http"

	"github.com/shobayosamuel/applypilot-backend/internal/platform/httpx"
	"github.com/shobayosamuel/applypilot-backend/internal/platform/middleware"
)

type Handler struct{ svc *Service }

func NewHandler(svc *Service) *Handler { return &Handler{svc: svc} }

func (h *Handler) Routes(mux *http.ServeMux, authed middleware.Func) {
	mux.HandleFunc("GET /api/jobs", authed(h.List))
	mux.HandleFunc("POST /api/jobs", authed(h.Create))
	mux.HandleFunc("GET /api/jobs/lookup", authed(h.Lookup))
	mux.HandleFunc("GET /api/jobs/{id}", authed(h.Get))
	mux.HandleFunc("PATCH /api/jobs/{id}", authed(h.Update))
	mux.HandleFunc("DELETE /api/jobs/{id}", authed(h.Delete))
	mux.HandleFunc("GET /api/stats", authed(h.Stats))
}

func (h *Handler) List(w http.ResponseWriter, r *http.Request) {
	q, err := ParseListQuery(r.URL.Query())
	if err != nil {
		httpx.Error(w, r, err)
		return
	}
	list, err := h.svc.List(r.Context(), httpx.UserID(r), q)
	if err != nil {
		httpx.Error(w, r, err)
		return
	}
	httpx.JSON(w, http.StatusOK, list)
}

// Create upserts by URL: 201 when new, 200 when merged into an existing job.
func (h *Handler) Create(w http.ResponseWriter, r *http.Request) {
	var req CreateRequest
	if err := httpx.Decode(w, r, &req, 1<<20); err != nil {
		httpx.Error(w, r, err)
		return
	}
	if err := req.Validate(); err != nil {
		httpx.Error(w, r, err)
		return
	}
	res, created, err := h.svc.Save(r.Context(), httpx.UserID(r), req)
	if err != nil {
		httpx.Error(w, r, err)
		return
	}
	status := http.StatusOK
	if created {
		status = http.StatusCreated
	}
	httpx.JSON(w, status, res)
}

func (h *Handler) Lookup(w http.ResponseWriter, r *http.Request) {
	res, err := h.svc.GetByURL(r.Context(), httpx.UserID(r), r.URL.Query().Get("url"))
	if err != nil {
		httpx.Error(w, r, err)
		return
	}
	httpx.JSON(w, http.StatusOK, res)
}

func (h *Handler) Get(w http.ResponseWriter, r *http.Request) {
	res, err := h.svc.Get(r.Context(), httpx.UserID(r), r.PathValue("id"))
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
	res, err := h.svc.Update(r.Context(), httpx.UserID(r), r.PathValue("id"), req)
	if err != nil {
		httpx.Error(w, r, err)
		return
	}
	httpx.JSON(w, http.StatusOK, res)
}

func (h *Handler) Delete(w http.ResponseWriter, r *http.Request) {
	if err := h.svc.Delete(r.Context(), httpx.UserID(r), r.PathValue("id")); err != nil {
		httpx.Error(w, r, err)
		return
	}
	httpx.NoContent(w)
}

func (h *Handler) Stats(w http.ResponseWriter, r *http.Request) {
	res, err := h.svc.Stats(r.Context(), httpx.UserID(r))
	if err != nil {
		httpx.Error(w, r, err)
		return
	}
	httpx.JSON(w, http.StatusOK, res)
}
