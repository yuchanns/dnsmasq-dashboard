package main

import (
	"context"
	"errors"
	"log/slog"
	"net"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/yuchanns/dnsmasq-dashboard/internal/config"
	"github.com/yuchanns/dnsmasq-dashboard/internal/dashboard"
	"github.com/yuchanns/dnsmasq-dashboard/internal/neighbor"
	"github.com/yuchanns/dnsmasq-dashboard/internal/server"
)

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	}))

	cfg, err := config.Load()
	if err != nil {
		logger.Error("invalid configuration", "error", err)
		os.Exit(1)
	}

	var neighborReader neighbor.Reader = neighbor.CommandReader{
		Command:   cfg.IPCommand,
		Interface: cfg.Interface,
	}
	if cfg.NeighborFile != "" {
		neighborReader = neighbor.FileReader{Path: cfg.NeighborFile}
	}

	dashboardService := dashboard.NewService(cfg, neighborReader, logger)
	httpServer, err := server.New(dashboardService, logger)
	if err != nil {
		logger.Error("initialize server", "error", err)
		os.Exit(1)
	}

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()
	go dashboardService.Run(ctx)

	instance := &http.Server{
		Addr:              cfg.ListenAddress,
		Handler:           httpServer.Handler(),
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       10 * time.Second,
		WriteTimeout:      0,
		IdleTimeout:       90 * time.Second,
		BaseContext: func(net.Listener) context.Context {
			return ctx
		},
	}

	go func() {
		logger.Info("leaseboard listening",
			"address", cfg.ListenAddress,
			"leaseFile", cfg.LeaseFile,
			"interface", cfg.Interface,
		)
		if err := instance.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			logger.Error("serve dashboard", "error", err)
			stop()
		}
	}()

	<-ctx.Done()
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := instance.Shutdown(shutdownCtx); err != nil {
		logger.Error("shutdown dashboard", "error", err)
	}
}
