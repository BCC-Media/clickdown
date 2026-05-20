package main

import (
	"context"
	"embed"
	"errors"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"

	"github.com/bcc-media/clickdown/internal/clickup"
	"github.com/bcc-media/clickdown/internal/config"
	"github.com/bcc-media/clickdown/internal/db"
	"github.com/bcc-media/clickdown/internal/server"
	syncworker "github.com/bcc-media/clickdown/internal/sync"
)

//go:embed all:web/dist
var webFS embed.FS

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("config: %v", err)
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	store, err := db.Open(ctx, cfg.DBPath)
	if err != nil {
		log.Fatalf("db: %v", err)
	}
	defer store.Close()

	// Load persisted sync interval (overrides cfg if set).
	interval := cfg.SyncInterval
	if v, err := store.Q.GetSetting(ctx, "sync_interval_seconds"); err == nil && v != "" {
		if secs, perr := strconv.ParseInt(v, 10, 64); perr == nil && secs > 0 {
			interval = time.Duration(secs) * time.Second
		}
	}

	client := clickup.New(cfg.Token)
	worker := syncworker.NewWorker(store, client, interval)
	go worker.Run(ctx)

	srv := server.New(store, client, worker, webFS)
	httpServer := &http.Server{Addr: cfg.ListenAddr, Handler: srv.Handler()}
	go func() {
		log.Printf("listening on http://%s", cfg.ListenAddr)
		if err := httpServer.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Fatalf("http: %v", err)
		}
	}()

	<-ctx.Done()
	log.Printf("shutting down")
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	_ = httpServer.Shutdown(shutdownCtx)
}
