package auth

import (
	"context"
	"errors"
	"log"

	"github.com/shobayosamuel/applypilot-backend/internal/platform/apperr"
	"github.com/shobayosamuel/applypilot-backend/internal/platform/security"
	"google.golang.org/api/idtoken"
)

// ProfileSeeder creates a new user's starter profile (implemented by the profile app).
type ProfileSeeder interface {
	Seed(ctx context.Context, userID, name, email string) error
}

type Service struct {
	repo           *Repository
	tokens         *security.Tokens
	profile        ProfileSeeder
	googleClientID string // empty disables Google sign-in
}

func NewService(repo *Repository, tokens *security.Tokens, profile ProfileSeeder, googleClientID string) *Service {
	return &Service{repo: repo, tokens: tokens, profile: profile, googleClientID: googleClientID}
}

func (s *Service) Signup(ctx context.Context, req SignupRequest) (*AuthResource, error) {
	hash, err := security.HashPassword(req.Password)
	if err != nil {
		return nil, err
	}
	u, err := s.create(ctx, req.Email, req.Name, hash)
	if err != nil {
		return nil, err
	}
	return s.issue(u)
}

// Google signs in (or registers) the owner of a Google ID token. Google-only
// accounts have an empty password hash, which CheckPassword never matches.
func (s *Service) Google(ctx context.Context, req GoogleRequest) (*AuthResource, error) {
	if s.googleClientID == "" {
		return nil, apperr.Unavailable("google sign-in is not configured", nil)
	}
	payload, err := idtoken.Validate(ctx, req.IDToken, s.googleClientID)
	if err != nil {
		return nil, apperr.Unauthorized("invalid google token")
	}
	if verified, _ := payload.Claims["email_verified"].(bool); !verified {
		return nil, apperr.Unauthorized("google email is not verified")
	}
	email, _ := payload.Claims["email"].(string)
	name, _ := payload.Claims["name"].(string)

	u, err := s.repo.ByEmail(ctx, email)
	if errors.Is(err, errNotFound) {
		u, err = s.create(ctx, email, name, "")
	}
	if err != nil {
		return nil, err
	}
	return s.issue(u)
}

func (s *Service) create(ctx context.Context, email, name, hash string) (*User, error) {
	u, err := s.repo.Create(ctx, email, name, hash)
	if errors.Is(err, errDuplicate) {
		return nil, apperr.Conflict("an account with this email already exists")
	} else if err != nil {
		return nil, err
	}
	if err := s.profile.Seed(ctx, u.ID, u.Name, u.Email); err != nil {
		log.Printf("seed profile for %s: %v", u.ID, err) // not fatal: profile is created on first save
	}
	return u, nil
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
