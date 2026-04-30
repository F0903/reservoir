package main

import (
	"context"
	"log/slog"
	"os/signal"
	"reservoir/config"
	"reservoir/logging"
	"syscall"
)

func main() {
	cfg, err := config.LoadOrDefault("var/config.json")
	if err != nil {
		slog.Error("Failed to load configuration", "error", err)
		panic(err)
	}

	config.OverrideFromFlags(cfg)
	logging.Init(cfg)

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	rt, err := NewRuntime(cfg, ctx)
	if err != nil {
		slog.Error("Failed to initialize runtime", "error", err)
		panic(err)
	}
	defer rt.Close()

	if err := rt.Run(ctx); err != nil {
		slog.Error("Runtime stopped with error", "error", err)
		panic(err)
	}

	slog.Info("Runtime stopped")
}
