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
	"github.com/shopspring/decimal"
	"go.uber.org/zap"
)

// Demo funding interval (production: 8h).
const defaultFundingRate = "0.0001"

func main() {
	cfg := config.Load("funding-engine")
	port := 8088
	if v := os.Getenv("FUNDING_ENGINE_HTTP_PORT"); v != "" {
		fmt.Sscanf(v, "%d", &port) //nolint:errcheck
	}
	cfg.HTTPPort = port

	interval := 5 * time.Minute
	if v := os.Getenv("FUNDING_INTERVAL"); v != "" {
		if d, err := time.ParseDuration(v); err == nil {
			interval = d
		}
	}

	log, _ := logger.New(cfg.ServiceName, cfg.LogLevel)
	defer log.Sync() //nolint:errcheck

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	st, err := perpstore.New(ctx, cfg.PostgresURL)
	if err != nil {
		log.Fatal("database", zap.Error(err))
	}
	defer st.Close()

	engine := &perpservice.Engine{
		Store: st,
		Account: tradeclients.NewAccountClient(env("ACCOUNT_SERVICE_URL", "http://localhost:8081")),
	}
	rate, _ := decimal.NewFromString(env("FUNDING_RATE", defaultFundingRate))

	settler := &Settler{Engine: engine, Store: st, Rate: rate}

	mux := http.NewServeMux()
	mux.HandleFunc("GET /health", func(w http.ResponseWriter, _ *http.Request) {
		httputil.OK(w, map[string]string{"status": "ok", "service": cfg.ServiceName, "phase": "3"})
	})
	mux.HandleFunc("GET /api/v1/funding/rates", func(w http.ResponseWriter, r *http.Request) {
		symbol := r.URL.Query().Get("symbol")
		if symbol == "" {
			httputil.OK(w, map[string]string{"default_rate": rate.String()})
			return
		}
		rate, err := st.LatestFundingRate(r.Context(), symbol)
		if err != nil {
			httputil.OK(w, map[string]string{"symbol": symbol, "rate": settler.Rate.String()})
			return
		}
		httputil.OK(w, map[string]string{"symbol": symbol, "rate": rate.String()})
	})
	mux.HandleFunc("POST /api/v1/funding/settle", func(w http.ResponseWriter, r *http.Request) {
		n, err := settler.RunOnce(r.Context())
		if err != nil {
			httputil.Fail(w, apperrors.New(apperrors.CodeInternal, err.Error()))
			return
		}
		httputil.OK(w, map[string]any{"payments": n, "rate": rate.String()})
	})
	metrics.Register(mux, cfg.ServiceName)

	go func() {
		ticker := time.NewTicker(interval)
		defer ticker.Stop()
		for {
			n, err := settler.RunOnce(ctx)
			if err != nil {
				log.Warn("funding failed", zap.Error(err))
			} else {
				log.Info("funding settled", zap.Int("payments", n))
			}
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
			}
		}
	}()

	srv := &http.Server{Addr: fmt.Sprintf(":%d", cfg.HTTPPort), Handler: metrics.Wrap(cfg.ServiceName, cors(mux))}
	go func() {
		log.Info("listening", zap.Int("port", cfg.HTTPPort), zap.Duration("interval", interval))
		_ = srv.ListenAndServe()
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	cancel()
}

// Settler applies funding payments to all open positions.
type Settler struct {
	Engine *perpservice.Engine
	Store  *perpstore.Store
	Rate   decimal.Decimal
}

func (s *Settler) RunOnce(ctx context.Context) (int, error) {
	markets, err := s.Store.ListMarkets(ctx)
	if err != nil {
		return 0, err
	}
	payments := 0
	for _, m := range markets {
		_ = s.Store.InsertFundingRate(ctx, m.Symbol, s.Rate)
	}
	positions, err := s.Store.ListAllPositions(ctx)
	if err != nil {
		return 0, err
	}
	for i := range positions {
		if err := s.Engine.ApplyFunding(ctx, &positions[i], s.Rate); err != nil {
			continue
		}
		payments++
	}
	return payments, nil
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
