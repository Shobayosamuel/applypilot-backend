// Package config loads settings from the environment.
package config

import (
	"log"
	"os"
	"strconv"
	"strings"
)

type Config struct {
	Addr            string
	DatabaseURL     string
	JWTSecret       string
	AnthropicAPIKey string
	ModelSmart      string // resumes, cover letters, match analysis
	ModelFast       string // field mapping, job extraction, short answers
	AllowedOrigins  []string
	AIDailyLimit    int
	WebURL          string
	TrustProxy      bool // honour X-Forwarded-For (only behind a trusted reverse proxy)
}

func Load() Config {
	c := Config{
		Addr:            env("ADDR", ":8080"),
		DatabaseURL:     env("DATABASE_URL", "postgres://applypilot:applypilot@localhost:5432/applypilot?sslmode=disable"),
		JWTSecret:       env("JWT_SECRET", ""),
		AnthropicAPIKey: env("ANTHROPIC_API_KEY", ""),
		ModelSmart:      env("CLAUDE_MODEL_SMART", "claude-sonnet-5"),
		ModelFast:       env("CLAUDE_MODEL_FAST", "claude-haiku-4-5"),
		AIDailyLimit:    envInt("AI_DAILY_LIMIT", 100),
		WebURL:          env("WEB_URL", "http://localhost:3000"),
		TrustProxy:      env("TRUST_PROXY", "") == "true",
	}
	for _, o := range strings.Split(env("ALLOWED_ORIGINS", "http://localhost:3000"), ",") {
		if o = strings.TrimSpace(o); o != "" {
			c.AllowedOrigins = append(c.AllowedOrigins, o)
		}
	}
	if len(c.JWTSecret) < 32 {
		log.Fatal("JWT_SECRET must be set and at least 32 characters")
	}
	if c.AnthropicAPIKey == "" {
		log.Println("WARNING: ANTHROPIC_API_KEY is not set; AI endpoints will return errors")
	}
	return c
}

func env(k, def string) string {
	if v := os.Getenv(k); v != "" {
		return v
	}
	return def
}

func envInt(k string, def int) int {
	if v := os.Getenv(k); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			return n
		}
	}
	return def
}
