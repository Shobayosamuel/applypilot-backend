package job

import (
	"net/url"
	"slices"
	"strconv"
	"strings"

	"github.com/shobayosamuel/applypilot-backend/internal/platform/apperr"
)

const (
	StatusSaved        = "saved"
	StatusApplied      = "applied"
	StatusInterviewing = "interviewing"
	StatusOffer        = "offer"
	StatusRejected     = "rejected"
	StatusArchived     = "archived"
)

var Statuses = []string{StatusSaved, StatusApplied, StatusInterviewing, StatusOffer, StatusRejected, StatusArchived}

func validStatus(s string) bool { return slices.Contains(Statuses, s) }

// CreateRequest saves a job (upserts by URL).
type CreateRequest struct {
	Title       string `json:"title"`
	Company     string `json:"company"`
	Location    string `json:"location"`
	URL         string `json:"url"`
	Description string `json:"description"`
	Salary      string `json:"salary"`
	Source      string `json:"source"`
	Status      string `json:"status"`
	Notes       string `json:"notes"`
}

func (r *CreateRequest) Validate() error {
	r.URL = strings.TrimSpace(r.URL)
	r.Title = strings.TrimSpace(r.Title)
	if r.Status == "" {
		r.Status = StatusSaved
	}
	if r.Source == "" {
		r.Source = "manual"
	}
	if !validStatus(r.Status) {
		return apperr.Invalid("invalid status")
	}
	if r.Title == "" && r.URL == "" {
		return apperr.Invalid("a job needs at least a title or a URL")
	}
	if len(r.Description) > 100_000 {
		r.Description = r.Description[:100_000]
	}
	return nil
}

// UpdateRequest patches a job; omitted fields are left unchanged.
type UpdateRequest struct {
	Title       *string `json:"title"`
	Company     *string `json:"company"`
	Location    *string `json:"location"`
	URL         *string `json:"url"`
	Description *string `json:"description"`
	Salary      *string `json:"salary"`
	Status      *string `json:"status"`
	Notes       *string `json:"notes"`
}

func (r *UpdateRequest) Validate() error {
	if r.Status != nil && !validStatus(*r.Status) {
		return apperr.Invalid("invalid status")
	}
	return nil
}

// ListQuery is parsed from ?status=&q=&limit=.
type ListQuery struct {
	Status string
	Query  string
	Limit  int
}

func ParseListQuery(v url.Values) (ListQuery, error) {
	q := ListQuery{Status: v.Get("status"), Query: strings.TrimSpace(v.Get("q"))}
	q.Limit, _ = strconv.Atoi(v.Get("limit"))
	if q.Limit <= 0 || q.Limit > 500 {
		q.Limit = 500
	}
	if q.Status != "" && !validStatus(q.Status) {
		return q, apperr.Invalid("invalid status")
	}
	return q, nil
}
