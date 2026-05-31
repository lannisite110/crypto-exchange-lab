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
	"github.com/crypto-exchange-lab/go-common/exchange"
	"github.com/crypto-exchange-lab/go-common/httputil"
	"github.com/crypto-exchange-lab/go-common/logger"
	"github.com/crypto-exchange-lab/go-common/metrics"
	"github.com/crypto-exchange-lab/orderapi"
	"github.com/crypto-exchange-lab/orderflow"
	"github.com/crypto-exchange-lab/orderstore"
	"github.com/crypto-exchange-lab/tradeclients"
	"go.uber.org/zap"
)

func main() {
	cfg := config.Load("order-service")
	port := 8082
	if v := os.Getenv("ORDER_SERVICE_HTTP_PORT"); v != "" {
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

	ctx := context.Background()
	st, err := orderstore.New(ctx, cfg.PostgresURL)
	if err != nil {
		log.Fatal("database", zap.Error(err))
	}
	defer st.Close()

	engine := &orderflow.Engine{
		Venue: exchange.VenueCEX, RefType: "cex_order",
		Store: st, Account: tradeclients.NewAccountClient(accountURL),
		Matching: tradeclients.NewMatchingClient(matchURL),
	}

	h := &orderapi.Handler{Engine: engine, Store: st, Service: cfg.ServiceName, Phase: "1"}

	mux := http.NewServeMux()
	mux.HandleFunc("GET /health", func(w http.ResponseWriter, _ *http.Request) {
		httputil.OK(w, map[string]string{"status": "ok", "service": cfg.ServiceName, "venue": "CEX"})
	})
	h.Register(mux)
	metrics.Register(mux, cfg.ServiceName)

	srv := &http.Server{
		Addr: fmt.Sprintf(":%d", cfg.HTTPPort), Handler: metrics.Wrap(cfg.ServiceName, cors(mux)),
		ReadTimeout: config.HTTPReadTimeout(), WriteTimeout: config.HTTPWriteTimeout(),
	}

	go func() {
		log.Info("listening", zap.Int("port", cfg.HTTPPort), zap.String("venue", "CEX"))
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatal("server failed", zap.Error(err))
		}
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
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		next.ServeHTTP(w, r)
	})
}
