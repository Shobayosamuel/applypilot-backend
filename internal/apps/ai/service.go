package ai

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"strings"

	"github.com/shobayosamuel/applypilot-backend/internal/apps/job"
	"github.com/shobayosamuel/applypilot-backend/internal/apps/resume"
	"github.com/shobayosamuel/applypilot-backend/internal/platform/apperr"
	c "github.com/shobayosamuel/applypilot-backend/internal/platform/claude"
)

// Dependencies on other apps, defined here (consumer side) to keep apps decoupled.
type (
	ProfileReader interface {
		Raw(ctx context.Context, userID string) (json.RawMessage, error)
	}
	JobStore interface {
		Get(ctx context.Context, userID, id string) (*job.Resource, error)
		GetByURL(ctx context.Context, userID, url string) (*job.Resource, error)
		SaveMatch(ctx context.Context, userID, id string, score int, details any) error
	}
	ResumeReader interface {
		File(ctx context.Context, userID, id string) (*resume.Resource, []byte, error)
	}
)

type Config struct {
	SmartModel string // resumes, cover letters, match, answers
	FastModel  string // form mapping, job extraction
	DailyLimit int    // calls per user per 24h; 0 = unlimited
}

type Service struct {
	repo    *Repository
	claude  *c.Client
	cfg     Config
	profile ProfileReader
	jobs    JobStore
	resumes ResumeReader
}

func NewService(repo *Repository, client *c.Client, cfg Config, profile ProfileReader, jobs JobStore, resumes ResumeReader) *Service {
	return &Service{repo: repo, claude: client, cfg: cfg, profile: profile, jobs: jobs, resumes: resumes}
}

// CallsToday implements job.AIUsage.
func (s *Service) CallsToday(ctx context.Context, userID string) (int, error) {
	return s.repo.CallsToday(ctx, userID)
}

// ---------- shared plumbing ----------

func (s *Service) checkQuota(ctx context.Context, userID string) error {
	if s.cfg.DailyLimit <= 0 {
		return nil
	}
	n, err := s.repo.CallsToday(ctx, userID)
	if err != nil {
		return err
	}
	if n >= s.cfg.DailyLimit {
		return apperr.RateLimited("daily AI limit reached; try again tomorrow")
	}
	return nil
}

// resolveJob loads a saved job by id, or uses the inline job (linking it to a
// saved job with the same URL when there is one).
func (s *Service) resolveJob(ctx context.Context, userID string, ref JobRef) (JobInput, *string, error) {
	if ref.JobID != "" {
		j, err := s.jobs.Get(ctx, userID, ref.JobID)
		if err != nil {
			return JobInput{}, nil, err
		}
		id := j.ID
		return JobInput{Title: j.Title, Company: j.Company, Location: j.Location, Description: j.Description, URL: j.URL}, &id, nil
	}
	in := *ref.Job
	var jobID *string
	if in.URL != "" {
		if j, err := s.jobs.GetByURL(ctx, userID, in.URL); err == nil {
			jobID = &j.ID
			if in.Description == "" {
				in.Description = j.Description
			}
		}
	}
	return in, jobID, nil
}

func (s *Service) record(ctx context.Context, userID string, jobID *string, kind, input string, output any, r *c.Response) {
	if len(input) > 500 {
		input = input[:500]
	}
	g := &Generation{UserID: userID, JobID: jobID, Kind: kind, InputSummary: input, Output: outputString(output)}
	if r != nil {
		g.Model = r.Model
		g.InputTokens = r.Usage.InputTokens + r.Usage.CacheCreationInputTokens
		g.OutputTokens = r.Usage.OutputTokens
		g.CacheRead = r.Usage.CacheReadInputTokens
	}
	if err := s.repo.Log(ctx, g); err != nil {
		log.Printf("log generation: %v", err)
	}
}

// claudeErr converts client errors into client-safe app errors.
func claudeErr(err error) error {
	if errors.Is(err, c.ErrNoAPIKey) {
		return apperr.Unavailable("AI is not configured on the server (ANTHROPIC_API_KEY missing)", err)
	}
	var apiErr *c.APIError
	if errors.As(err, &apiErr) && apiErr.Status == 429 {
		return apperr.Unavailable("AI provider is rate limited, please retry shortly", err)
	}
	return apperr.Upstream("AI request failed, please retry", err)
}

// prepare runs the common steps: quota, job lookup, profile load.
func (s *Service) prepare(ctx context.Context, userID string, ref JobRef) (JobInput, *string, json.RawMessage, error) {
	if err := s.checkQuota(ctx, userID); err != nil {
		return JobInput{}, nil, nil, err
	}
	j, jobID, err := s.resolveJob(ctx, userID, ref)
	if err != nil {
		return JobInput{}, nil, nil, err
	}
	prof, err := s.profile.Raw(ctx, userID)
	return j, jobID, prof, err
}

// ---------- features ----------

func (s *Service) ParseResume(ctx context.Context, userID string, req ParseResumeRequest) (*ParsedProfileResource, error) {
	if err := s.checkQuota(ctx, userID); err != nil {
		return nil, err
	}
	res, data, err := s.resumes.File(ctx, userID, req.ResumeID)
	if err != nil {
		return nil, err
	}
	var content []c.Block
	switch res.ContentType {
	case "application/pdf":
		content = append(content, c.Block{Type: "document", Source: &c.Source{Type: "base64", MediaType: "application/pdf", Data: base64.StdEncoding.EncodeToString(data)}})
	case "text/plain":
		content = append(content, c.Block{Type: "text", Text: "<resume>\n" + string(data) + "\n</resume>"})
	default:
		return nil, apperr.Invalid("resume import supports PDF or TXT files; upload a PDF version to import")
	}
	content = append(content, c.Block{Type: "text", Text: promptParseResume})

	var out map[string]any
	r, err := s.claude.Structured(ctx, c.Request{
		Model: s.cfg.SmartModel, MaxTokens: 8000,
		System:   []c.Block{{Type: "text", Text: "You extract structured data from resumes with perfect fidelity. Never invent information."}},
		Messages: []c.Message{{Role: "user", Content: content}},
	}, toolSaveProfile, &out)
	if err != nil {
		return nil, claudeErr(err)
	}
	s.record(ctx, userID, nil, "parse_resume", res.Filename, out, r)
	return &ParsedProfileResource{Profile: out}, nil
}

func (s *Service) Match(ctx context.Context, userID string, req MatchRequest) (*MatchResource, error) {
	j, jobID, prof, err := s.prepare(ctx, userID, req.JobRef)
	if err != nil {
		return nil, err
	}
	var out MatchResource
	r, err := s.claude.Structured(ctx, c.Request{
		Model: s.cfg.SmartModel, MaxTokens: 2000, System: systemWithProfile(prof), Messages: userMsg(jobBlock(j) + promptMatch),
	}, toolReportMatch, &out)
	if err != nil {
		return nil, claudeErr(err)
	}
	out.Score = max(0, min(100, out.Score))
	if jobID != nil {
		if err := s.jobs.SaveMatch(ctx, userID, *jobID, out.Score, out); err != nil {
			log.Printf("save match: %v", err)
		}
	}
	s.record(ctx, userID, jobID, "match", j.Title+" @ "+j.Company, out, r)
	return &out, nil
}

func (s *Service) CoverLetter(ctx context.Context, userID string, req CoverLetterRequest) (*CoverLetterResource, error) {
	j, jobID, prof, err := s.prepare(ctx, userID, req.JobRef)
	if err != nil {
		return nil, err
	}
	prompt := jobBlock(j) + fmt.Sprintf(promptCoverLetter, req.Tone)
	if ins := strings.TrimSpace(req.Instructions); ins != "" {
		prompt += "\n\nAdditional instructions from the candidate:\n" + ins
	}
	r, err := s.claude.Create(ctx, c.Request{Model: s.cfg.SmartModel, MaxTokens: 1500, System: systemWithProfile(prof), Messages: userMsg(prompt)})
	if err != nil {
		return nil, claudeErr(err)
	}
	out := strings.TrimSpace(r.Text())
	s.record(ctx, userID, jobID, "cover_letter", j.Title+" @ "+j.Company, out, r)
	return &CoverLetterResource{CoverLetter: out}, nil
}

func (s *Service) TailorResume(ctx context.Context, userID string, req TailorRequest) (*TailoredResumeResource, error) {
	j, jobID, prof, err := s.prepare(ctx, userID, req.JobRef)
	if err != nil {
		return nil, err
	}
	var out TailoredResumeResource
	r, err := s.claude.Structured(ctx, c.Request{
		Model: s.cfg.SmartModel, MaxTokens: 4000, System: systemWithProfile(prof), Messages: userMsg(jobBlock(j) + promptTailor),
	}, toolTailor, &out)
	if err != nil {
		return nil, claudeErr(err)
	}
	s.record(ctx, userID, jobID, "tailor_resume", j.Title+" @ "+j.Company, out, r)
	return &out, nil
}

func (s *Service) Answer(ctx context.Context, userID string, req AnswerRequest) (*AnswerResource, error) {
	j, jobID, prof, err := s.prepare(ctx, userID, req.JobRef)
	if err != nil {
		return nil, err
	}
	prompt := jobBlock(j) + fmt.Sprintf(promptAnswer, req.Question, req.MaxChars)
	r, err := s.claude.Create(ctx, c.Request{Model: s.cfg.SmartModel, MaxTokens: 1200, System: systemWithProfile(prof), Messages: userMsg(prompt)})
	if err != nil {
		return nil, claudeErr(err)
	}
	out := strings.TrimSpace(r.Text())
	if len(out) > req.MaxChars {
		out = out[:req.MaxChars]
	}
	s.record(ctx, userID, jobID, "answer", req.Question, out, r)
	return &AnswerResource{Answer: out}, nil
}

func (s *Service) MapFields(ctx context.Context, userID string, req MapFieldsRequest) (*MapFieldsResource, error) {
	if len(req.Fields) == 0 {
		return &MapFieldsResource{Values: []FieldValue{}}, nil
	}
	j, jobID, prof, err := s.prepare(ctx, userID, req.JobRef)
	if err != nil {
		return nil, err
	}
	fj, _ := json.Marshal(req.Fields)
	var raw struct {
		Values []FieldValue `json:"values"`
	}
	r, err := s.claude.Structured(ctx, c.Request{
		Model: s.cfg.FastModel, MaxTokens: 4000, System: systemWithProfile(prof),
		Messages: userMsg(jobBlock(j) + "\n\n<form_fields>\n" + string(fj) + promptMapFields),
	}, toolFillFields, &raw)
	if err != nil {
		return nil, claudeErr(err)
	}
	out := &MapFieldsResource{Values: cleanFieldValues(req.Fields, raw.Values)}
	labels := make([]string, 0, len(req.Fields))
	for _, f := range req.Fields {
		labels = append(labels, f.Label)
	}
	s.record(ctx, userID, jobID, "map_fields", strings.Join(labels, "; "), out.Values, r)
	return out, nil
}

// cleanFieldValues drops empty values, unknown ids, and choices that aren't one
// of the field's real options (normalising case to the option's spelling).
func cleanFieldValues(fields []FormField, values []FieldValue) []FieldValue {
	byID := make(map[string]FormField, len(fields))
	for _, f := range fields {
		byID[f.ID] = f
	}
	out := make([]FieldValue, 0, len(values))
	for _, v := range values {
		f, ok := byID[v.ID]
		if !ok || v.Value == "" {
			continue
		}
		if len(f.Options) > 0 && f.Type != "checkbox" {
			matched := false
			for _, o := range f.Options {
				if strings.EqualFold(strings.TrimSpace(o), strings.TrimSpace(v.Value)) {
					v.Value, matched = o, true
					break
				}
			}
			if !matched {
				continue
			}
		}
		out = append(out, v)
	}
	return out
}

func (s *Service) ExtractJob(ctx context.Context, userID string, req ExtractJobRequest) (*ExtractedJobResource, error) {
	if err := s.checkQuota(ctx, userID); err != nil {
		return nil, err
	}
	var out ExtractedJobResource
	r, err := s.claude.Structured(ctx, c.Request{
		Model: s.cfg.FastModel, MaxTokens: 6000,
		System:   []c.Block{{Type: "text", Text: "You extract job postings from web page text accurately. Never invent details."}},
		Messages: userMsg("URL: " + req.URL + "\n\n<page_text>\n" + req.PageText + "\n</page_text>" + promptExtractJob),
	}, toolSaveJob, &out)
	if err != nil {
		return nil, claudeErr(err)
	}
	out.URL = req.URL
	s.record(ctx, userID, nil, "extract_job", req.URL, out.Title+" @ "+out.Company, r)
	return &out, nil
}

func (s *Service) Usage(ctx context.Context, userID string) (*UsageResource, error) {
	u, err := s.repo.Usage(ctx, userID)
	if err != nil {
		return nil, err
	}
	return &UsageResource{
		CallsToday: u.CallsToday, DailyLimit: s.cfg.DailyLimit, Calls30d: u.Calls30d,
		InputTokens: u.InputTokens, OutputTokens: u.OutputTokens, CacheRead: u.CacheTokens,
	}, nil
}

func (s *Service) Generations(ctx context.Context, userID, jobID, kind string) ([]GenerationResource, error) {
	list, err := s.repo.List(ctx, userID, jobID, kind)
	if err != nil {
		return nil, err
	}
	return NewGenerationResources(list), nil
}
