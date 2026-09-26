package resume

import (
	"io"
	"net/http"
	"path/filepath"
	"strings"

	"github.com/shobayosamuel/applypilot-backend/internal/platform/apperr"
)

const MaxBytes = 5 << 20

var allowedTypes = map[string]string{
	".pdf":  "application/pdf",
	".docx": "application/vnd.openxmlformats-officedocument.wordprocessingml.document",
	".doc":  "application/msword",
	".txt":  "text/plain",
}

// UploadRequest is a parsed multipart upload (field "file", optional "name").
type UploadRequest struct {
	Name        string
	Filename    string
	ContentType string
	Data        []byte
}

func ParseUpload(w http.ResponseWriter, r *http.Request) (*UploadRequest, error) {
	r.Body = http.MaxBytesReader(w, r.Body, MaxBytes+64<<10)
	if err := r.ParseMultipartForm(MaxBytes); err != nil {
		return nil, apperr.Invalid("file too large (max 5 MB) or invalid upload")
	}
	f, hdr, err := r.FormFile("file")
	if err != nil {
		return nil, apperr.Invalid("missing file field")
	}
	defer f.Close()
	data, err := io.ReadAll(io.LimitReader(f, MaxBytes+1))
	if err != nil {
		return nil, apperr.Invalid("could not read upload")
	}
	req := &UploadRequest{Name: strings.TrimSpace(r.FormValue("name")), Filename: filepath.Base(hdr.Filename), Data: data}
	return req, req.Validate()
}

func (r *UploadRequest) Validate() error {
	ext := strings.ToLower(filepath.Ext(r.Filename))
	ct, ok := allowedTypes[ext]
	if !ok {
		return apperr.Invalid("resume must be PDF, DOCX, DOC or TXT")
	}
	if len(r.Data) == 0 || len(r.Data) > MaxBytes {
		return apperr.Invalid("file must be between 1 byte and 5 MB")
	}
	if ct == "application/pdf" && !strings.HasPrefix(string(r.Data[:min(5, len(r.Data))]), "%PDF-") {
		return apperr.Invalid("file is not a valid PDF")
	}
	r.ContentType = ct
	if r.Name == "" {
		r.Name = strings.TrimSuffix(r.Filename, filepath.Ext(r.Filename))
	}
	return nil
}
