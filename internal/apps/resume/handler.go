package resume

import (
	"mime"
	"net/http"

	"github.com/shobayosamuel/applypilot-backend/internal/platform/httpx"
	"github.com/shobayosamuel/applypilot-backend/internal/platform/middleware"
)

type Handler struct{ svc *Service }

func NewHandler(svc *Service) *Handler { return &Handler{svc: svc} }

func (h *Handler) Routes(mux *http.ServeMux, authed middleware.Func) {
	mux.HandleFunc("GET /api/resumes", authed(h.List))
	mux.HandleFunc("POST /api/resumes", authed(h.Upload))
	mux.HandleFunc("GET /api/resumes/{id}/file", authed(h.File))
	mux.HandleFunc("POST /api/resumes/{id}/default", authed(h.SetDefault))
	mux.HandleFunc("DELETE /api/resumes/{id}", authed(h.Delete))
}

func (h *Handler) List(w http.ResponseWriter, r *http.Request) {
	list, err := h.svc.List(r.Context(), httpx.UserID(r))
	if err != nil {
		httpx.Error(w, r, err)
		return
	}
	httpx.JSON(w, http.StatusOK, list)
}

func (h *Handler) Upload(w http.ResponseWriter, r *http.Request) {
	req, err := ParseUpload(w, r)
	if err != nil {
		httpx.Error(w, r, err)
		return
	}
	res, err := h.svc.Upload(r.Context(), httpx.UserID(r), req)
	if err != nil {
		httpx.Error(w, r, err)
		return
	}
	httpx.JSON(w, http.StatusCreated, res)
}

// File streams the file ("default" selects the default resume).
func (h *Handler) File(w http.ResponseWriter, r *http.Request) {
	res, data, err := h.svc.File(r.Context(), httpx.UserID(r), r.PathValue("id"))
	if err != nil {
		httpx.Error(w, r, err)
		return
	}
	w.Header().Set("Content-Type", res.ContentType)
	w.Header().Set("Content-Disposition", mime.FormatMediaType("inline", map[string]string{"filename": res.Filename}))
	w.Header().Set("X-Resume-Filename", res.Filename)
	w.Write(data)
}

func (h *Handler) SetDefault(w http.ResponseWriter, r *http.Request) {
	if err := h.svc.SetDefault(r.Context(), httpx.UserID(r), r.PathValue("id")); err != nil {
		httpx.Error(w, r, err)
		return
	}
	httpx.NoContent(w)
}

func (h *Handler) Delete(w http.ResponseWriter, r *http.Request) {
	if err := h.svc.Delete(r.Context(), httpx.UserID(r), r.PathValue("id")); err != nil {
		httpx.Error(w, r, err)
		return
	}
	httpx.NoContent(w)
}
