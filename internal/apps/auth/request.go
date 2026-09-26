package auth

import (
	"net/mail"
	"strings"

	"github.com/shobayosamuel/applypilot-backend/internal/platform/apperr"
)

type SignupRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
	Name     string `json:"name"`
}

func (r *SignupRequest) Validate() error {
	r.Email = strings.TrimSpace(r.Email)
	r.Name = strings.TrimSpace(r.Name)
	if _, err := mail.ParseAddress(r.Email); err != nil {
		return apperr.Invalid("enter a valid email address")
	}
	if len(r.Password) < 8 {
		return apperr.Invalid("password must be at least 8 characters")
	}
	if len(r.Password) > 256 || len(r.Name) > 200 {
		return apperr.Invalid("input too long")
	}
	return nil
}

type LoginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

func (r *LoginRequest) Validate() error {
	r.Email = strings.TrimSpace(r.Email)
	if r.Email == "" || r.Password == "" {
		return apperr.Invalid("email and password are required")
	}
	return nil
}
