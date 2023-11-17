package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"github.com/khulnasoft-lab/harbor-scanner-tunnel/pkg/etc"
	"github.com/khulnasoft-lab/harbor-scanner-tunnel/pkg/ext"
	"github.com/khulnasoft-lab/harbor-scanner-tunnel/pkg/http/api"
	v1 "github.com/khulnasoft-lab/harbor-scanner-tunnel/pkg/http/api/v1"
	"github.com/khulnasoft-lab/harbor-scanner-tunnel/pkg/persistence/redis"
	"github.com/khulnasoft-lab/harbor-scanner-tunnel/pkg/queue"
	"github.com/khulnasoft-lab/harbor-scanner-tunnel/pkg/redisx"
	"github.com/khulnasoft-lab/harbor-scanner-tunnel/pkg/scan"
	"github.com/khulnasoft-lab/harbor-scanner-tunnel/pkg/tunnel"
)

var (
	// Default wise GoReleaser sets three ldflags:
	version = "dev"
	commit  = "none"
	date    = "unknown"
)

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: etc.LogLevel(),
	}))
	slog.SetDefault(logger)

	info := etc.BuildInfo{
		Version: version,
		Commit:  commit,
		Date:    date,
	}

	ctx := context.Background()
	if err := run(ctx, info); err != nil {
		slog.Error("Error: %v", err)
		os.Exit(1)
	}
}

func run(ctx context.Context, info etc.BuildInfo) error {
	slog.Info("Starting harbor-scanner-tunnel", slog.String("version", info.Version),
		slog.String("commit", info.Commit), slog.String("built_at", info.Date),
	)

	config, err := etc.GetConfig()
	if err != nil {
		return fmt.Errorf("getting config: %w", err)
	}
	if err = etc.Check(config); err != nil {
		return fmt.Errorf("checking config: %w", err)
	}

	rdb, err := redisx.NewClient(config.RedisPool)
	if err != nil {
		return fmt.Errorf("constructing connection pool: %w", err)
	}

	wrapper := tunnel.NewWrapper(config.Tunnel, ext.DefaultAmbassador)
	store := redis.NewStore(config.RedisStore, rdb)
	controller := scan.NewController(store, wrapper, scan.NewTransformer(&scan.SystemClock{}))
	enqueuer := queue.NewEnqueuer(config.JobQueue, rdb, store)
	worker := queue.NewWorker(config.JobQueue, rdb, controller)

	apiHandler := v1.NewAPIHandler(info, config, enqueuer, store, wrapper)
	apiServer, err := api.NewServer(config.API, apiHandler)
	if err != nil {
		return fmt.Errorf("new api server: %w", err)
	}

	shutdownComplete := make(chan struct{})
	go func() {
		sigint := make(chan os.Signal, 1)
		signal.Notify(sigint, syscall.SIGINT, syscall.SIGTERM)
		captured := <-sigint
		slog.Debug("Trapped os signal", slog.String("signal", captured.String()))

		apiServer.Shutdown()
		worker.Stop()
		_ = rdb.Close()

		close(shutdownComplete)
	}()

	worker.Start(ctx)
	apiServer.ListenAndServe()

	<-shutdownComplete
	return nil
}