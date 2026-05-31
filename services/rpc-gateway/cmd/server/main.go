package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/crypto-exchange-lab/chainstore"
	"github.com/crypto-exchange-lab/go-common/config"
	"github.com/crypto-exchange-lab/go-common/httputil"
	"github.com/crypto-exchange-lab/go-common/logger"
	"github.com/crypto-exchange-lab/go-common/metrics"
	"github.com/crypto-exchange-lab/rpc-gateway/internal/handler"
	apperrors "github.com/crypto-exchange-lab/go-common/errors"
	"go.uber.org/zap"
)

func main() {
	cfg := config.Load("rpc-gateway")
	port := 8089
	if v := os.Getenv("RPC_GATEWAY_HTTP_PORT"); v != "" {
		fmt.Sscanf(v, "%d", &port) //nolint:errcheck
	}
	cfg.HTTPPort = port

	log, _ := logger.New(cfg.ServiceName, cfg.LogLevel)
	defer log.Sync() //nolint:errcheck

	ctx := context.Background()
	st, err := chainstore.New(ctx, cfg.PostgresURL)
	if err != nil {
		log.Fatal("database", zap.Error(err))
	}
	defer st.Close()

	mux := http.NewServeMux()
	mux.HandleFunc("GET /health", func(w http.ResponseWriter, _ *http.Request) {
		httputil.OK(w, map[string]string{"status": "ok", "service": cfg.ServiceName, "phase": "5"})
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
		Handler:      metrics.Wrap(cfg.ServiceName, cors(mux)),
		ReadTimeout:  config.HTTPReadTimeout(),
		WriteTimeout: config.HTTPWriteTimeout(),
	}
	go func() {
		log.Info("listening", zap.Int("port", cfg.HTTPPort))
		_ = srv.ListenAndServe()
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	_ = srv.Shutdown(shutdownCtx)
}

func cors(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		next.ServeHTTP(w, r)
	})
}
