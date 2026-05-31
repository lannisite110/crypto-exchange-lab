package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/crypto-exchange-lab/go-common/config"
	"github.com/crypto-exchange-lab/go-common/cors"
	"github.com/crypto-exchange-lab/go-common/httputil"
	"github.com/crypto-exchange-lab/go-common/logger"
	"github.com/crypto-exchange-lab/go-common/metrics"
	"github.com/crypto-exchange-lab/go-common/exchange"
	"github.com/crypto-exchange-lab/matching-engine/internal/engine"
	"github.com/crypto-exchange-lab/matching-engine/internal/handler"
	"github.com/crypto-exchange-lab/matching-engine/internal/seed"
	"go.uber.org/zap"
)

func main() {
	cfg := config.Load("matching-engine")
	port := 8083
	if v := os.Getenv("MATCHING_ENGINE_HTTP_PORT"); v != "" {
		fmt.Sscanf(v, "%d", &port) //nolint:errcheck
	}
	cfg.HTTPPort = port

	log, err := logger.New(cfg.ServiceName, cfg.LogLevel)
	if err != nil {
		panic(err)
	}
	defer log.Sync() //nolint:errcheck

	hub := engine.NewHub(exchange.DefaultSpotSymbols)
	if seed.Enabled() {
		seed.Liquidity(hub)
	}
	h := handler.New(hub)

	mux := http.NewServeMux()
	mux.HandleFunc("GET /health", func(w http.ResponseWriter, _ *http.Request) {
		httputil.OK(w, map[string]string{"status": "ok", "service": cfg.ServiceName, "phase": "1"})
	})
	h.Register(mux)
	metrics.Register(mux, cfg.ServiceName)

	srv := &http.Server{
		Addr:         fmt.Sprintf(":%d", cfg.HTTPPort),
		Handler:      metrics.Wrap(cfg.ServiceName, cors.Middleware(mux)),
		ReadTimeout:  config.HTTPReadTimeout(),
		WriteTimeout: 0, // WebSocket /ws/v1/market must stay open
	}

	go func() {
		log.Info("listening", zap.Int("port", cfg.HTTPPort), zap.String("markets", seed.LogSummary(hub)))
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatal("server failed", zap.Error(err))
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	_ = srv.Shutdown(ctx)
	log.Info("stopped")
}
