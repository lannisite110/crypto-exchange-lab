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
	"github.com/crypto-exchange-lab/go-common/httputil"
	"github.com/crypto-exchange-lab/go-common/logger"
	"github.com/crypto-exchange-lab/go-common/metrics"
	"github.com/crypto-exchange-lab/hyperliquid-engine/internal/handler"
	"github.com/crypto-exchange-lab/perpservice"
	"github.com/crypto-exchange-lab/perpstore"
	"github.com/crypto-exchange-lab/tradeclients"
	"go.uber.org/zap"
)

func main() {
	cfg := config.Load("hyperliquid-engine")
	port := 8085
	if v := os.Getenv("HYPERLIQUID_ENGINE_HTTP_PORT"); v != "" {
		fmt.Sscanf(v, "%d", &port) //nolint:errcheck
	}
	cfg.HTTPPort = port

	accountURL := env("ACCOUNT_SERVICE_URL", "http://localhost:8081")
	matchURL := env("MATCHING_ENGINE_URL", "http://localhost:8083")

	log, err := logger.New(cfg.ServiceName, cfg.LogLevel)
	if err != nil {
		panic(err)
	}
	defer log.Sync() //nolint:errcheck

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	st, err := perpstore.New(ctx, cfg.PostgresURL)
	if err != nil {
		log.Fatal("database", zap.Error(err))
	}
	defer st.Close()

	engine := &perpservice.Engine{Store: st, Account: tradeclients.NewAccountClient(accountURL)}
	h := &handler.Handler{Engine: engine, Store: st}

	feed := &perpservice.MarkFeed{Store: st, MatchingURL: matchURL}
	go feed.Run(ctx)

	mux := http.NewServeMux()
	mux.HandleFunc("GET /health", func(w http.ResponseWriter, _ *http.Request) {
		httputil.OK(w, map[string]string{"status": "ok", "service": cfg.ServiceName, "phase": "3"})
	})
	h.Register(mux)
	metrics.Register(mux, cfg.ServiceName)

	srv := &http.Server{
		Addr: fmt.Sprintf(":%d", cfg.HTTPPort), Handler: metrics.Wrap(cfg.ServiceName, cors(mux)),
		ReadTimeout: config.HTTPReadTimeout(), WriteTimeout: config.HTTPWriteTimeout(),
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
	cancel()

	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer shutdownCancel()
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
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		next.ServeHTTP(w, r)
	})
}
