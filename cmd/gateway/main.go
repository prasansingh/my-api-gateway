package main

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/singhprasan/my-api-gateway/internal/config"
	"github.com/singhprasan/my-api-gateway/internal/health"
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
	handler := middleware.Chain(p, middleware.Logging(), middleware.Metrics())

	srv := &http.Server{
		Addr:         fmt.Sprintf(":%d", cfg.Server.Port),
		Handler:      handler,
		ReadTimeout:  cfg.Server.ReadTimeout.Std(),
		WriteTimeout: cfg.Server.WriteTimeout.Std(),
	}

	checker := health.New(cfg.Routes, cfg.Health.CheckInterval.Std(), cfg.Health.CheckTimeout.Std())
	checker.Start()

	metricsMux := http.NewServeMux()
	metricsMux.Handle("/metrics", promhttp.Handler())
	metricsMux.HandleFunc("/livez", checker.LivezHandler())
	metricsMux.HandleFunc("/readyz", checker.ReadyzHandler())
	metricsSrv := &http.Server{
		Addr:    ":9090",
		Handler: metricsMux,
	}

	go func() {
		slog.Info("gateway listening", "port", cfg.Server.Port)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			slog.Error("server error", "error", err)
			os.Exit(1)
		}
	}()

	go func() {
		slog.Info("metrics server listening", "port", 9090)
		if err := metricsSrv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			slog.Error("metrics server error", "error", err)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	slog.Info("shutting down gateway")

	checker.MarkUnhealthy()

	if d := cfg.Server.DrainWindow.Std(); d > 0 {
		slog.Info("draining", "window", d)
		time.Sleep(d)
	}

	ctx, cancel := context.WithTimeout(context.Background(), cfg.Server.Shutdown.Std())
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		slog.Error("shutdown error", "error", err)
	}
	if err := metricsSrv.Shutdown(ctx); err != nil {
		slog.Error("metrics server shutdown error", "error", err)
	}

	checker.Stop()

	slog.Info("gateway stopped")
}
