package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	apperrors "github.com/crypto-exchange-lab/go-common/errors"
	"github.com/crypto-exchange-lab/go-common/config"
	"github.com/crypto-exchange-lab/go-common/httputil"
	"github.com/crypto-exchange-lab/go-common/logger"
	"github.com/crypto-exchange-lab/go-common/metrics"
	"github.com/crypto-exchange-lab/perpservice"
	"github.com/crypto-exchange-lab/perpstore"
	"github.com/crypto-exchange-lab/tradeclients"
	"go.uber.org/zap"
)

func main() {
	cfg := config.Load("risk-engine")
	port := 8086
	if v := os.Getenv("RISK_ENGINE_HTTP_PORT"); v != "" {
		fmt.Sscanf(v, "%d", &port) //nolint:errcheck
	}
	cfg.HTTPPort = port

	log, _ := logger.New(cfg.ServiceName, cfg.LogLevel)
	defer log.Sync() //nolint:errcheck

	ctx := context.Background()
	st, err := perpstore.New(ctx, cfg.PostgresURL)
	if err != nil {
		log.Fatal("database", zap.Error(err))
	}
	defer st.Close()

	engine := &perpservice.Engine{
		Store: st,
		Account: tradeclients.NewAccountClient(env("ACCOUNT_SERVICE_URL", "http://localhost:8081")),
	}

	mux := http.NewServeMux()
	mux.HandleFunc("GET /health", func(w http.ResponseWriter, _ *http.Request) {
		httputil.OK(w, map[string]string{"status": "ok", "service": cfg.ServiceName, "phase": "3"})
	})
	mux.HandleFunc("GET /api/v1/users/{id}/risk", func(w http.ResponseWriter, r *http.Request) {
		risks, err := engine.ListRisk(r.Context(), r.PathValue("id"))
		if err != nil {
			fail(w, err)
			return
		}
		httputil.OK(w, map[string]any{"user_id": r.PathValue("id"), "positions": risks})
	})
	mux.HandleFunc("GET /api/v1/users/{id}/positions/{symbol}/risk", func(w http.ResponseWriter, r *http.Request) {
		rsk, err := engine.RiskForPosition(r.Context(), r.PathValue("id"), r.PathValue("symbol"))
		if err != nil {
			fail(w, err)
			return
		}
		httputil.OK(w, rsk)
	})
	metrics.Register(mux, cfg.ServiceName)

	srv := &http.Server{Addr: fmt.Sprintf(":%d", cfg.HTTPPort), Handler: metrics.Wrap(cfg.ServiceName, cors(mux))}
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

func env(k, def string) string {
	if v := os.Getenv(k); v != "" {
		return v
	}
	return def
}

func cors(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func fail(w http.ResponseWriter, err error) {
	if ae, ok := err.(*apperrors.AppError); ok {
		httputil.Fail(w, ae)
		return
	}
	httputil.Fail(w, apperrors.New(apperrors.CodeInternal, err.Error()))
}
