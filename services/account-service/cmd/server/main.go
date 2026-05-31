package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/crypto-exchange-lab/account-service/internal/handler"
	"github.com/crypto-exchange-lab/account-service/internal/store"
	apperrors "github.com/crypto-exchange-lab/go-common/errors"
	"github.com/crypto-exchange-lab/go-common/config"
	"github.com/crypto-exchange-lab/go-common/httputil"
	"github.com/crypto-exchange-lab/go-common/logger"
	"github.com/crypto-exchange-lab/go-common/metrics"
	"go.uber.org/zap"
)

func main() {
	cfg := config.Load("account-service")
	cfg.HTTPPort = envPort("ACCOUNT_SERVICE_HTTP_PORT", 8081)

	log, err := logger.New(cfg.ServiceName, cfg.LogLevel)
	if err != nil {
		panic(err)
	}
	defer log.Sync() //nolint:errcheck

	ctx := context.Background()
	st, err := store.New(ctx, cfg.PostgresURL)
	if err != nil {
		log.Fatal("database", zap.Error(err))
	}
	defer st.Close()

	mux := http.NewServeMux()
	mux.HandleFunc("GET /health", func(w http.ResponseWriter, _ *http.Request) {
		httputil.OK(w, map[string]string{"status": "ok", "service": cfg.ServiceName, "phase": "1"})
	})
	mux.HandleFunc("GET /ready", func(w http.ResponseWriter, r *http.Request) {
		if err := st.Pool().Ping(r.Context()); err != nil {
			httputil.Fail(w, apperrors.New(apperrors.CodeInternal, "database not ready"))
			return
		}
		httputil.OK(w, map[string]string{"status": "ready"})
	})
	handler.New(st).Register(mux)
	metrics.Register(mux, cfg.ServiceName)

	srv := &http.Server{
		Addr:         fmt.Sprintf(":%d", cfg.HTTPPort),
		Handler:      metrics.Wrap(cfg.ServiceName, withCORS(mux)),
		ReadTimeout:  config.HTTPReadTimeout(),
		WriteTimeout: config.HTTPWriteTimeout(),
	}

	go func() {
		log.Info("listening", zap.Int("port", cfg.HTTPPort))
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatal("server failed", zap.Error(err))
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := srv.Shutdown(shutdownCtx); err != nil {
		log.Error("shutdown failed", zap.Error(err))
	}
	log.Info("stopped")
}

func withCORS(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func envPort(key string, def int) int {
	if v := os.Getenv(key); v != "" {
		var p int
		if _, err := fmt.Sscanf(v, "%d", &p); err == nil {
			return p
		}
	}
	return def
}
