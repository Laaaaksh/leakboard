// Command leakboard runs the Leakboard server: the scheduled gitleaks
// scanner, the JSON API, and the dashboard's static frontend, all in one
// binary backed by a single Postgres database.
package main

import (
	"context"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/Laaaaksh/leakboard/internal/alert"
	"github.com/Laaaaksh/leakboard/internal/api"
	"github.com/Laaaaksh/leakboard/internal/config"
	"github.com/Laaaaksh/leakboard/internal/gitleaks"
	"github.com/Laaaaksh/leakboard/internal/scanner"
	"github.com/Laaaaksh/leakboard/internal/store"
	"github.com/Laaaaksh/leakboard/internal/webui"
)

const shutdownTimeout = 10 * time.Second

func main() {
	log := slog.New(slog.NewTextHandler(os.Stdout, nil))

	if err := run(log); err != nil {
		log.Error("fatal", "error", err)
		os.Exit(1)
	}
}

func run(log *slog.Logger) error {
	cfg, err := config.Load()
	if err != nil {
		return err
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	db, err := store.Open(ctx, cfg.DatabaseURL)
	if err != nil {
		return err
	}
	defer func() { _ = db.Close() }()

	gl := gitleaks.New(cfg.GitleaksPath)
	if err := gl.CheckAvailable(ctx); err != nil {
		return err
	}

	if err := os.MkdirAll(cfg.WorkDir, 0o755); err != nil {
		return err
	}

	sc := &scanner.Scanner{
		Store:    db,
		Gitleaks: gl,
		Alerts:   alert.New(),
		WorkDir:  cfg.WorkDir,
		Log:      log,
		BaseURL:  cfg.BaseURL,
	}
	go sc.Run(ctx, cfg.ScanInterval)

	frontend, err := webui.FS()
	if err != nil {
		return err
	}

	srv := &api.Server{
		Store:   db,
		Scanner: sc,
		Log:     log,
		WorkDir: cfg.WorkDir,
		Secure:  strings.HasPrefix(cfg.BaseURL, "https://"),
	}

	httpServer := &http.Server{
		Addr:    cfg.Addr,
		Handler: srv.Routes(frontend),
	}

	go func() {
		<-ctx.Done()
		shutdownCtx, cancel := context.WithTimeout(context.Background(), shutdownTimeout)
		defer cancel()
		_ = httpServer.Shutdown(shutdownCtx)
	}()

	log.Info("leakboard listening", "addr", cfg.Addr)
	if err := httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		return err
	}
	return nil
}
