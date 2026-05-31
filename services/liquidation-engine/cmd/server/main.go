package main

import (
	"bytes"
	"context"
	"encoding/json"
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
	"github.com/crypto-exchange-lab/perps"
	"github.com/crypto-exchange-lab/perpstore"
	"github.com/shopspring/decimal"
	"go.uber.org/zap"
)

func main() {
	cfg := config.Load("liquidation-engine")
	port := 8087
	if v := os.Getenv("LIQUIDATION_ENGINE_HTTP_PORT"); v != "" {
		fmt.Sscanf(v, "%d", &port) //nolint:errcheck
	}
	cfg.HTTPPort = port

	hyperURL := env("HYPERLIQUID_ENGINE_URL", "http://localhost:8085")
	scanInterval := 15 * time.Second

	log, _ := logger.New(cfg.ServiceName, cfg.LogLevel)
	defer log.Sync() //nolint:errcheck

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	st, err := perpstore.New(ctx, cfg.PostgresURL)
	if err != nil {
		log.Fatal("database", zap.Error(err))
	}
	defer st.Close()

	scanner := &Scanner{Store: st, HyperURL: hyperURL, Client: &http.Client{Timeout: 10 * time.Second}}

	mux := http.NewServeMux()
	mux.HandleFunc("GET /health", func(w http.ResponseWriter, _ *http.Request) {
		httputil.OK(w, map[string]string{"status": "ok", "service": cfg.ServiceName, "phase": "3"})
	})
	mux.HandleFunc("POST /api/v1/scan", func(w http.ResponseWriter, r *http.Request) {
		n, err := scanner.RunOnce(r.Context())
		if err != nil {
			httputil.Fail(w, apperrors.New(apperrors.CodeInternal, err.Error()))
			return
		}
		httputil.OK(w, map[string]any{"liquidated": n})
	})
	metrics.Register(mux, cfg.ServiceName)

	go func() {
		ticker := time.NewTicker(scanInterval)
		defer ticker.Stop()
		for {
			n, err := scanner.RunOnce(ctx)
			if err != nil {
				log.Warn("scan failed", zap.Error(err))
			} else if n > 0 {
				log.Info("liquidations", zap.Int("count", n))
			}
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
			}
		}
	}()

	srv := &http.Server{Addr: fmt.Sprintf(":%d", cfg.HTTPPort), Handler: metrics.Wrap(cfg.ServiceName, mux)}
	go func() {
		log.Info("listening", zap.Int("port", cfg.HTTPPort))
		_ = srv.ListenAndServe()
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	cancel()
}

// Scanner finds underwater positions and calls hyperliquid-engine.
type Scanner struct {
	Store    *perpstore.Store
	HyperURL string
	Client   *http.Client
}

func (s *Scanner) RunOnce(ctx context.Context) (int, error) {
	positions, err := s.Store.ListAllPositions(ctx)
	if err != nil {
		return 0, err
	}
	liquidated := 0
	for _, pos := range positions {
		mkt, err := s.Store.GetMarket(ctx, pos.Symbol)
		if err != nil {
			continue
		}
		mark, err := s.Store.GetMarkPrice(ctx, pos.Symbol)
		if err != nil {
			continue
		}
		notional := perps.Notional(pos.Size, mark)
		upnl := perps.UnrealizedPnL(string(pos.Side), pos.Size, pos.EntryPrice, mark)
		equity := perps.Equity(pos.Margin, upnl)
		maint := perps.MaintenanceMargin(notional, mkt.MaintMarginRate)
		if perps.MarginRatio(equity, maint).GreaterThanOrEqual(decimal.NewFromInt(1)) {
			continue
		}
		if err := s.callLiquidate(ctx, pos.UserID, pos.Symbol); err != nil {
			continue
		}
		liquidated++
	}
	return liquidated, nil
}

func (s *Scanner) callLiquidate(ctx context.Context, userID, symbol string) error {
	body, _ := json.Marshal(map[string]string{"user_id": userID, "symbol": symbol})
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, s.HyperURL+"/api/v1/internal/liquidate", bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := s.Client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	var env httputil.Envelope
	if err := json.NewDecoder(resp.Body).Decode(&env); err != nil {
		return err
	}
	if !env.OK {
		return fmt.Errorf("liquidate failed")
	}
	return nil
}

func env(k, def string) string {
	if v := os.Getenv(k); v != "" {
		return v
	}
	return def
}
