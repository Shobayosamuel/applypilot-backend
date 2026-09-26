package ai

import (
	"strings"

	"github.com/shobayosamuel/applypilot-backend/internal/platform/apperr"
)

// JobInput is job context sent inline (e.g. by the extension for an unsaved posting).
type JobInput struct {
	Title       string `json:"title"`
	Company     string `json:"company"`
	Location    string `json:"location"`
	Description string `json:"description"`
	URL         string `json:"url"`
}

// JobRef points at a saved job (job_id) or carries the job inline (job).
type JobRef struct {
	JobID string    `json:"job_id"`
	Job   *JobInput `json:"job"`
}

func (r *JobRef) validate(optional bool) error {
	if r.JobID != "" {
		return nil
	}
	if r.Job == nil && optional {
		r.Job = &JobInput{Title: "(unknown role)"}
	}
	if r.Job == nil || (strings.TrimSpace(r.Job.Description) == "" && strings.TrimSpace(r.Job.Title) == "") {
		return apperr.Invalid("provide job_id or a job with at least a title or description")
	}
	return nil
}

type ParseResumeRequest struct {
	ResumeID string `json:"resume_id"` // empty = default resume
}

func (r *ParseResumeRequest) Validate() error {
	if r.ResumeID == "" {
		r.ResumeID = "default"
	}
	return nil
}

type MatchRequest struct{ JobRef }

func (r *MatchRequest) Validate() error { return r.validate(false) }

type CoverLetterRequest struct {
	JobRef
	Tone         string `json:"tone"`
	Instructions string `json:"instructions"`
}

func (r *CoverLetterRequest) Validate() error {
	if len(r.Instructions) > 2000 {
		return apperr.Invalid("instructions too long (max 2000 characters)")
	}
	if r.Tone == "" {
		r.Tone = "professional and warm"
	}
	return r.validate(false)
}

type TailorRequest struct{ JobRef }

func (r *TailorRequest) Validate() error { return r.validate(false) }

type AnswerRequest struct {
	JobRef
	Question string `json:"question"`
	MaxChars int    `json:"max_chars"`
}

func (r *AnswerRequest) Validate() error {
	r.Question = strings.TrimSpace(r.Question)
	if r.Question == "" {
		return apperr.Invalid("question is required")
	}
	if len(r.Question) > 4000 {
		return apperr.Invalid("question too long")
	}
	if r.MaxChars <= 0 || r.MaxChars > 5000 {
		r.MaxChars = 1200
	}
	return r.validate(true)
}

// FormField is one application-form field the extension couldn't fill by rules.
type FormField struct {
	ID          string   `json:"id"`
	Label       string   `json:"label"`
	Type        string   `json:"type"`
	Name        string   `json:"name,omitempty"`
	Placeholder string   `json:"placeholder,omitempty"`
	Options     []string `json:"options,omitempty"`
	Required    bool     `json:"required,omitempty"`
}

type MapFieldsRequest struct {
	JobRef
	Fields []FormField `json:"fields"`
}

func (r *MapFieldsRequest) Validate() error {
	if len(r.Fields) > 150 {
		r.Fields = r.Fields[:150]
	}
	return r.validate(true)
}

type ExtractJobRequest struct {
	URL      string `json:"url"`
	PageText string `json:"page_text"`
}

func (r *ExtractJobRequest) Validate() error {
	if strings.TrimSpace(r.PageText) == "" {
		return apperr.Invalid("page_text is required")
	}
	if len(r.PageText) > 40000 {
		r.PageText = r.PageText[:40000]
	}
	return nil
}
