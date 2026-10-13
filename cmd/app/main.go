package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"go.uber.org/zap"

	"pc/internal/config"
	"pc/internal/db"
	apphttp "pc/internal/http"
	"pc/internal/jobs"
	"pc/internal/logger"
	"pc/internal/migrate"
	"pc/internal/storage"
	"pc/web"
)

func main() {
	if len(os.Args) < 2 {
		usage()
		os.Exit(2)
	}
	cfg, err := config.Load()
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	log, err := logger.New(cfg.IsProd())
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	defer log.Sync()

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	switch os.Args[1] {
	case "serve":
		err = serve(ctx, cfg, log)
	case "migrate":
		if len(os.Args) < 3 {
			usage()
			os.Exit(2)
		}
		err = migrate.Run(ctx, cfg.DatabaseURL, os.Args[2])
	case "createadmin":
		err = createAdmin(ctx, cfg, os.Args[2:])
	case "resetpassword":
		err = resetPassword(ctx, cfg, os.Args[2:])
	case "seeddemo":
		err = seedDemo(ctx, cfg, os.Args[2:])
	case "seed":
		err = seed(ctx, cfg)
	default:
		usage()
		os.Exit(2)
	}
	if err != nil {
		log.Error("command failed", zap.Error(err))
		os.Exit(1)
	}
}

func usage() {
	fmt.Fprintln(os.Stderr, "usage: app <serve|migrate up|down|status|createadmin --username X|resetpassword --username X [--keep]|seed|seeddemo [--days N] [--reset]>")
}

func serve(ctx context.Context, cfg *config.Config, log *zap.Logger) error {
	pool, err := pgxpool.New(ctx, cfg.DatabaseURL)
	if err != nil {
		return err
	}
	defer pool.Close()

	dist, err := web.Dist()
	if err != nil {
		return err
	}
	go jobs.RunSessionCleanup(ctx, db.New(pool), log, time.Hour)
	go jobs.RunSelfieCleanup(ctx, db.New(pool), storage.NewLocal(cfg.UploadDir), log, cfg.SelfieRetentionDays, 24*time.Hour)

	app := apphttp.NewServer(apphttp.Deps{Cfg: cfg, Log: log, Pool: pool, Dist: dist})

	errCh := make(chan error, 1)
	go func() { errCh <- app.Listen(cfg.HTTPAddr) }()
	log.Info("server listening", zap.String("addr", cfg.HTTPAddr))

	select {
	case err := <-errCh:
		return err
	case <-ctx.Done():
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		return app.ShutdownWithContext(shutdownCtx)
	}
}
