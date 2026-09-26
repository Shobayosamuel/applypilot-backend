package job

import (
	"context"
	"encoding/json"
	"errors"

	"github.com/shobayosamuel/applypilot-backend/internal/platform/apperr"
)

// AIUsage reports a user's AI calls in the last 24h (implemented by the ai app).
type AIUsage interface {
	CallsToday(ctx context.Context, userID string) (int, error)
}

type Service struct {
	repo    *Repository
	aiUsage AIUsage
}

func NewService(repo *Repository) *Service { return &Service{repo: repo} }

// SetAIUsage breaks the job <-> ai dependency cycle at wiring time.
func (s *Service) SetAIUsage(u AIUsage) { s.aiUsage = u }

func mapErr(err error) error {
	switch {
	case errors.Is(err, errNotFound):
		return apperr.NotFound("job not found")
	case errors.Is(err, errDuplicate):
		return apperr.Conflict("another job already uses this URL")
	}
	return err
}

func (s *Service) List(ctx context.Context, userID string, q ListQuery) ([]Resource, error) {
	list, err := s.repo.List(ctx, userID, Filter{Status: q.Status, Query: q.Query, Limit: q.Limit})
	if err != nil {
		return nil, err
	}
	return NewResources(list), nil
}

// Save creates or merges a job; created is true for a new row.
func (s *Service) Save(ctx context.Context, userID string, req CreateRequest) (*Resource, bool, error) {
	j, created, err := s.repo.Upsert(ctx, &Job{
		UserID: userID, Title: req.Title, Company: req.Company, Location: req.Location, URL: req.URL,
		Description: req.Description, Salary: req.Salary, Source: req.Source, Status: req.Status, Notes: req.Notes,
	})
	if err != nil {
		return nil, false, err
	}
	res := NewResource(j)
	return &res, created, nil
}

func (s *Service) Get(ctx context.Context, userID, id string) (*Resource, error) {
	j, err := s.repo.Get(ctx, userID, id)
	if err != nil {
		return nil, mapErr(err)
	}
	res := NewResource(j)
	return &res, nil
}

func (s *Service) GetByURL(ctx context.Context, userID, url string) (*Resource, error) {
	if url == "" {
		return nil, apperr.Invalid("url is required")
	}
	j, err := s.repo.GetByURL(ctx, userID, url)
	if err != nil {
		return nil, mapErr(err)
	}
	res := NewResource(j)
	return &res, nil
}

func (s *Service) Update(ctx context.Context, userID, id string, req UpdateRequest) (*Resource, error) {
	j, err := s.repo.Patch(ctx, userID, id, Patch{
		Title: req.Title, Company: req.Company, Location: req.Location, URL: req.URL,
		Description: req.Description, Salary: req.Salary, Status: req.Status, Notes: req.Notes,
	})
	if err != nil {
		return nil, mapErr(err)
	}
	res := NewResource(j)
	return &res, nil
}

func (s *Service) Delete(ctx context.Context, userID, id string) error {
	return mapErr(s.repo.Delete(ctx, userID, id))
}

// SaveMatch stores an AI match result on the job (used by the ai app).
func (s *Service) SaveMatch(ctx context.Context, userID, id string, score int, details any) error {
	b, err := json.Marshal(details)
	if err != nil {
		return err
	}
	return s.repo.SetMatch(ctx, userID, id, score, b)
}

func (s *Service) Stats(ctx context.Context, userID string) (*StatsResource, error) {
	counts, err := s.repo.StatusCounts(ctx, userID)
	if err != nil {
		return nil, err
	}
	st := &StatsResource{ByStatus: map[string]int{}}
	for _, k := range Statuses {
		st.ByStatus[k] = counts[k]
		st.Total += counts[k]
	}
	if st.AppliedTotal, st.AppliedWeek, err = s.repo.AppliedCounts(ctx, userID); err != nil {
		return nil, err
	}
	if s.aiUsage != nil {
		if st.AICallsToday, err = s.aiUsage.CallsToday(ctx, userID); err != nil {
			return nil, err
		}
	}
	return st, nil
}
