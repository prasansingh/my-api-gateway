package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"github.com/singhprasan/my-api-gateway/internal/config"
	"github.com/singhprasan/my-api-gateway/internal/proxy"
)

func main() {
	cfg, err := config.Load("config.yaml")
	if err != nil {
		fmt.Printf("failed to load config: %v\n", err)
		os.Exit(1)
	}

	p := proxy.New(cfg.Routes)

	srv := &http.Server{
		Addr:         fmt.Sprintf(":%d", cfg.Server.Port),
		Handler:      p,
		ReadTimeout:  cfg.Server.ReadTimeout.Std(),
		WriteTimeout: cfg.Server.WriteTimeout.Std(),
	}

	go func() {
		fmt.Printf("gateway listening on :%d\n", cfg.Server.Port)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			fmt.Printf("server error: %v\n", err)
			os.Exit(1)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	fmt.Println("shutting down gateway...")

	ctx, cancel := context.WithTimeout(context.Background(), cfg.Server.Shutdown.Std())
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		fmt.Printf("shutdown error: %v\n", err)
	}

	fmt.Println("gateway stopped")
}
