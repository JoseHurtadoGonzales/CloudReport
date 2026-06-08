package main

import (
	"context"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/cloudreport/api/internal/api"
	"github.com/cloudreport/api/internal/auth"
	"github.com/cloudreport/api/internal/blob"
	"github.com/cloudreport/api/internal/config"
	"github.com/cloudreport/api/internal/queue"
	"github.com/cloudreport/api/internal/render"
	"github.com/cloudreport/api/internal/scheduler"
	"github.com/cloudreport/api/internal/server"
	"github.com/cloudreport/api/internal/store"
	"github.com/cloudreport/api/internal/ws"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

func main() {
	cfg := config.Load()

	// Logger
	zerolog.TimeFieldFormat = time.RFC3339
	lvl, _ := zerolog.ParseLevel(cfg.LogLevel)
	zerolog.SetGlobalLevel(lvl)
	log.Logger = log.Output(zerolog.ConsoleWriter{Out: os.Stderr, TimeFormat: time.RFC3339})

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// DB
	db, err := store.New(ctx, cfg.DatabaseURL)
	if err != nil {
		log.Fatal().Err(err).Msg("connect db")
	}
	defer db.Close()

	if err := db.Migrate(ctx); err != nil {
		log.Fatal().Err(err).Msg("migrate")
	}

	// Blob (SeaweedFS via S3 API)
	bs, err := blob.New(ctx, cfg)
	if err != nil {
		log.Fatal().Err(err).Msg("init seaweedfs s3")
	}

	// Queue
	q, err := queue.New(ctx, cfg)
	if err != nil {
		log.Fatal().Err(err).Msg("init redis")
	}
	defer q.Close()

	// Auth
	authSvc := auth.New(cfg, db)

	// Bootstrap an initial admin user if one is configured and no users exist yet.
	if cfg.InitialAdminUsername != "" && cfg.InitialAdminPassword != "" {
		if n, err := db.CountUsers(ctx); err == nil && n == 0 {
			u, regErr := authSvc.Register(ctx, cfg.InitialAdminUsername, cfg.InitialAdminPassword, cfg.InitialAdminEmail)
			if regErr != nil {
				log.Warn().Err(regErr).Msg("bootstrap admin: register failed")
			} else {
				log.Info().Str("username", u.Username).Bool("admin", u.IsAdmin).Msg("bootstrap admin created")
			}
		}
	}

	// Render orchestrator
	renderer := render.New(cfg, db, bs, q)

	// Scheduler
	sched := scheduler.New(db, renderer)
	if err := sched.Start(ctx); err != nil {
		log.Error().Err(err).Msg("scheduler start")
	}
	defer sched.Stop()

	// Report-retention sweeper: every hour, delete reports whose per-template
	// retention window has elapsed (blob + DB row). Runs once on boot too so a
	// long-stopped instance catches up immediately.
	go func() {
		sweep := func() {
			n, err := renderer.SweepExpiredReports(ctx)
			if err != nil {
				log.Error().Err(err).Msg("report sweep")
			} else if n > 0 {
				log.Info().Int("deleted", n).Msg("report retention sweep")
			}
			// Hard-cap reports + profiles to the newest MaxHistoryRows.
			renderer.EnforceHistoryCap(ctx)
		}
		sweep()
		ticker := time.NewTicker(time.Hour)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				sweep()
			}
		}
	}()

	// HTTP server
	hub := ws.NewHub()

	app := server.New(cfg)
	api.Register(app, &api.Deps{
		Cfg:      cfg,
		DB:       db,
		Blob:     bs,
		Queue:    q,
		Auth:     authSvc,
		Renderer: renderer,
		Hub:      hub,
	})

	// Graceful shutdown
	go func() {
		sigs := make(chan os.Signal, 1)
		signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)
		<-sigs
		log.Info().Msg("shutting down")
		_ = app.ShutdownWithTimeout(10 * time.Second)
		cancel()
	}()

	log.Info().Str("port", cfg.HTTPPort).Msg("cloud-report API listening")
	if err := app.Listen(":" + cfg.HTTPPort); err != nil {
		log.Fatal().Err(err).Msg("listen")
	}
}
