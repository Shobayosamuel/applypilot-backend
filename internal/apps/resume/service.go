package resume

import (
	"context"
	"errors"

	"github.com/shobayosamuel/applypilot-backend/internal/platform/apperr"
)

type Service struct{ repo *Repository }

func NewService(repo *Repository) *Service { return &Service{repo: repo} }

func mapErr(err error) error {
	if errors.Is(err, errNotFound) {
		return apperr.NotFound("resume not found")
	}
	return err
}

func (s *Service) Upload(ctx context.Context, userID string, req *UploadRequest) (*Resource, error) {
	r, err := s.repo.Create(ctx, userID, req.Name, req.Filename, req.ContentType, req.Data)
	if err != nil {
		return nil, err
	}
	res := NewResource(r)
	return &res, nil
}

func (s *Service) List(ctx context.Context, userID string) ([]Resource, error) {
	list, err := s.repo.List(ctx, userID)
	if err != nil {
		return nil, err
	}
	return NewResources(list), nil
}

// File returns metadata and bytes; also used by the AI app to parse resumes.
func (s *Service) File(ctx context.Context, userID, id string) (*Resource, []byte, error) {
	r, data, err := s.repo.File(ctx, userID, id)
	if err != nil {
		return nil, nil, mapErr(err)
	}
	res := NewResource(r)
	return &res, data, nil
}

func (s *Service) SetDefault(ctx context.Context, userID, id string) error {
	return mapErr(s.repo.SetDefault(ctx, userID, id))
}

func (s *Service) Delete(ctx context.Context, userID, id string) error {
	return mapErr(s.repo.Delete(ctx, userID, id))
}
