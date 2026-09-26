// Package server wires the apps together and builds the HTTP handler.
package server

import (
	"database/sql"
	"net/http"
	"time"

	"github.com/shobayosamuel/applypilot-backend/internal/apps/ai"
	"github.com/shobayosamuel/applypilot-backend/internal/apps/auth"
	"github.com/shobayosamuel/applypilot-backend/internal/apps/job"
	"github.com/shobayosamuel/applypilot-backend/internal/apps/profile"
	"github.com/shobayosamuel/applypilot-backend/internal/apps/resume"
	"github.com/shobayosamuel/applypilot-backend/internal/platform/claude"
	"github.com/shobayosamuel/applypilot-backend/internal/platform/config"
	"github.com/shobayosamuel/applypilot-backend/internal/platform/httpx"
	"github.com/shobayosamuel/applypilot-backend/internal/platform/middleware"
	"github.com/shobayosamuel/applypilot-backend/internal/platform/security"
)

const tokenTTL = 30 * 24 * time.Hour

func New(cfg config.Config, db *sql.DB) http.Handler {
	tokens := security.NewTokens(cfg.JWTSecret, tokenTTL)
	authed := middleware.Auth(tokens)
	limit := middleware.RateLimit(10, time.Minute, cfg.TrustProxy)

	// services
	profileSvc := profile.NewService(profile.NewRepository(db))
	authSvc := auth.NewService(auth.NewRepository(db), tokens, profileSvc)
	resumeSvc := resume.NewService(resume.NewRepository(db))
	jobSvc := job.NewService(job.NewRepository(db))
	aiSvc := ai.NewService(ai.NewRepository(db), claude.NewClient(cfg.AnthropicAPIKey), ai.Config{
		SmartModel: cfg.ModelSmart, FastModel: cfg.ModelFast, DailyLimit: cfg.AIDailyLimit,
	}, profileSvc, jobSvc, resumeSvc)
	jobSvc.SetAIUsage(aiSvc)

	// routes
	mux := http.NewServeMux()
	mux.HandleFunc("GET /healthz", func(w http.ResponseWriter, r *http.Request) {
		httpx.JSON(w, http.StatusOK, map[string]string{"status": "ok"})
	})
	auth.NewHandler(authSvc).Routes(mux, authed, limit)
	profile.NewHandler(profileSvc).Routes(mux, authed)
	resume.NewHandler(resumeSvc).Routes(mux, authed)
	job.NewHandler(jobSvc).Routes(mux, authed)
	ai.NewHandler(aiSvc).Routes(mux, authed)

	return middleware.Recover(middleware.Logger(middleware.CORS(cfg.AllowedOrigins)(mux)))
}
