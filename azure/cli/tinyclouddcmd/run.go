package tinyclouddcmd

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"

	"tinycloud/internal/app"
	"tinycloud/internal/config"
	"tinycloud/internal/state"
	"tinycloud/internal/telemetry"
)

func Main() {
	cfg := config.FromEnv()
	logger := telemetry.NewJSONLogger(os.Stdout)

	store, err := state.NewStore(cfg.DataRoot)
	if err != nil {
		log.Fatalf("init state store: %v", err)
	}

	server := app.NewServer(cfg, store, logger)

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	if err := server.Run(ctx); err != nil {
		log.Fatalf("run server: %v", err)
	}
}
