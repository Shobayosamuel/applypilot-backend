package main

import (
	"context"
	"errors"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/shobayosamuel/applypilot-backend/internal/platform/claude"
	"github.com/shobayosamuel/applypilot-backend/internal/platform/config"
	"github.com/shobayosamuel/applypilot-backend/internal/platform/database"
	"github.com/shobayosamuel/applypilot-backend/internal/server"
)

func main() {
	cfg := config.Load()
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	db, err := database.Open(ctx, cfg.DatabaseURL)
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()
	if err := database.CheckSchema(ctx, db); err != nil {
		log.Fatal(err)
	}
	if u := os.Getenv("ANTHROPIC_BASE_URL"); u != "" {
		claude.SetBaseURL(u)
	}

	srv := &http.Server{
		Addr:              cfg.Addr,
		Handler:           server.New(cfg, db),
		ReadHeaderTimeout: 10 * time.Second,
		WriteTimeout:      4 * time.Minute, // AI calls can be slow
		IdleTimeout:       2 * time.Minute,
	}
	go func() {
		log.Printf("ApplyPilot API listening on %s (smart=%s fast=%s)", cfg.Addr, cfg.ModelSmart, cfg.ModelFast)
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Fatal(err)
		}
	}()
	<-ctx.Done()
	shutdown, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	srv.Shutdown(shutdown)
}
