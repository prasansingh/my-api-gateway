package main

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"github.com/singhprasan/my-api-gateway/internal/config"
	"github.com/singhprasan/my-api-gateway/internal/middleware"
	"github.com/singhprasan/my-api-gateway/internal/proxy"
)

func main() {
	slog.SetDefault(slog.New(slog.NewJSONHandler(os.Stdout, nil)))

	cfg, err := config.Load("config.yaml")
	if err != nil {
		slog.Error("failed to load config", "error", err)
		os.Exit(1)
	}

	p := proxy.New(cfg.Routes)
	handler := middleware.Chain(p, middleware.Logging())

	srv := &http.Server{
		Addr:         fmt.Sprintf(":%d", cfg.Server.Port),
		Handler:      handler,
		ReadTimeout:  cfg.Server.ReadTimeout.Std(),
		WriteTimeout: cfg.Server.WriteTimeout.Std(),
	}

	go func() {
		slog.Info("gateway listening", "port", cfg.Server.Port)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			slog.Error("server error", "error", err)
			os.Exit(1)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	slog.Info("shutting down gateway")

	ctx, cancel := context.WithTimeout(context.Background(), cfg.Server.Shutdown.Std())
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		slog.Error("shutdown error", "error", err)
	}

	slog.Info("gateway stopped")
}
