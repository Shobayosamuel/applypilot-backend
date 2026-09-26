package profile

import (
	"context"
	"encoding/json"
	"strings"
)

type Service struct{ repo *Repository }

func NewService(repo *Repository) *Service { return &Service{repo: repo} }

func (s *Service) Get(ctx context.Context, userID string) (*Resource, error) {
	data, at, err := s.repo.Get(ctx, userID)
	if err != nil {
		return nil, err
	}
	return &Resource{Profile: data, UpdatedAt: at}, nil
}

// Raw returns just the profile JSON (used by the AI app as prompt context).
func (s *Service) Raw(ctx context.Context, userID string) (json.RawMessage, error) {
	data, _, err := s.repo.Get(ctx, userID)
	return data, err
}

func (s *Service) Update(ctx context.Context, userID string, req UpdateRequest) (*Resource, error) {
	at, err := s.repo.Upsert(ctx, userID, req.Profile)
	if err != nil {
		return nil, err
	}
	return &Resource{Profile: req.Profile, UpdatedAt: at}, nil
}

// Seed creates the starter profile for a new account (implements auth.ProfileSeeder).
func (s *Service) Seed(ctx context.Context, userID, name, email string) error {
	first, last, _ := strings.Cut(strings.TrimSpace(name), " ")
	data, _ := json.Marshal(map[string]string{"first_name": first, "last_name": strings.TrimSpace(last), "email": email})
	return s.repo.InsertIfMissing(ctx, userID, data)
}
