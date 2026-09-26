package auth

import (
	"context"
	"errors"
	"log"

	"github.com/shobayosamuel/applypilot-backend/internal/platform/apperr"
	"github.com/shobayosamuel/applypilot-backend/internal/platform/security"
)

// ProfileSeeder creates a new user's starter profile (implemented by the profile app).
type ProfileSeeder interface {
	Seed(ctx context.Context, userID, name, email string) error
}

type Service struct {
	repo    *Repository
	tokens  *security.Tokens
	profile ProfileSeeder
}

func NewService(repo *Repository, tokens *security.Tokens, profile ProfileSeeder) *Service {
	return &Service{repo: repo, tokens: tokens, profile: profile}
}

func (s *Service) Signup(ctx context.Context, req SignupRequest) (*AuthResource, error) {
	hash, err := security.HashPassword(req.Password)
	if err != nil {
		return nil, err
	}
	u, err := s.repo.Create(ctx, req.Email, req.Name, hash)
	if errors.Is(err, errDuplicate) {
		return nil, apperr.Conflict("an account with this email already exists")
	} else if err != nil {
		return nil, err
	}
	if err := s.profile.Seed(ctx, u.ID, u.Name, u.Email); err != nil {
		log.Printf("seed profile for %s: %v", u.ID, err) // not fatal: profile is created on first save
	}
	return s.issue(u)
}

func (s *Service) Login(ctx context.Context, req LoginRequest) (*AuthResource, error) {
	u, err := s.repo.ByEmail(ctx, req.Email)
	if err != nil && !errors.Is(err, errNotFound) {
		return nil, err
	}
	if u == nil || !security.CheckPassword(u.PasswordHash, req.Password) {
		return nil, apperr.Unauthorized("incorrect email or password")
	}
	return s.issue(u)
}

func (s *Service) Me(ctx context.Context, userID string) (*UserResource, error) {
	u, err := s.repo.ByID(ctx, userID)
	if errors.Is(err, errNotFound) {
		return nil, apperr.NotFound("user not found")
	} else if err != nil {
		return nil, err
	}
	res := NewUserResource(u)
	return &res, nil
}

func (s *Service) issue(u *User) (*AuthResource, error) {
	tok, exp, err := s.tokens.Issue(u.ID)
	if err != nil {
		return nil, err
	}
	return &AuthResource{Token: tok, ExpiresAt: exp, User: NewUserResource(u)}, nil
}
