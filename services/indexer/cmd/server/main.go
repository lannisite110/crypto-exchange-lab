package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/crypto-exchange-lab/chainrpc"
	"github.com/crypto-exchange-lab/chainstore"
	apperrors "github.com/crypto-exchange-lab/go-common/errors"
	"github.com/crypto-exchange-lab/go-common/config"
	"github.com/crypto-exchange-lab/go-common/httputil"
	"github.com/crypto-exchange-lab/go-common/logger"
	"github.com/crypto-exchange-lab/go-common/metrics"
	"go.uber.org/zap"
)

func main() {
	cfg := config.Load("indexer")
	port := 8090
	if v := os.Getenv("INDEXER_HTTP_PORT"); v != "" {
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

	chainKey := env("INDEXER_CHAIN", "sepolia")
	ch, err := st.GetChain(ctx, chainKey)
	if err != nil {
		log.Fatal("chain config", zap.Error(err))
	}
	rpcURL := env("SEPOLIA_RPC_URL", ch.RPCURL)
	if rpcURL == "" {
		log.Fatal("missing SEPOLIA_RPC_URL or chain rpc_url")
	}

	watch := parseWatchList()
	for _, w := range watch {
		label := w.label
		if label == "" {
			label = w.addr
		}
		if err := st.UpsertWatchedContract(ctx, chainKey, w.addr, label); err != nil {
			log.Warn("watched contract", zap.Error(err))
		}
	}

	addresses := make([]string, len(watch))
	for i, w := range watch {
		addresses[i] = strings.ToLower(w.addr)
	}

	syncer := &chainstore.Syncer{
		Store:    st,
		RPC:      chainrpc.NewClient(rpcURL),
		ChainID:  chainKey,
		Watch:    addresses,
		BatchMax: envInt("INDEXER_BATCH_SIZE", 8),
	}

	interval := envDuration("INDEXER_POLL_INTERVAL", 12*time.Second)
	stop := make(chan struct{})
	go func() {
		ticker := time.NewTicker(interval)
		defer ticker.Stop()
		for {
			select {
			case <-stop:
				return
			case <-ticker.C:
				n, err := syncer.RunOnce(ctx)
				if err != nil {
					log.Warn("sync", zap.Error(err))
				} else if n > 0 {
					log.Info("indexed blocks", zap.Int("count", n))
				}
			}
		}
	}()

	mux := http.NewServeMux()
	mux.HandleFunc("GET /health", func(w http.ResponseWriter, _ *http.Request) {
		httputil.OK(w, map[string]string{"status": "ok", "service": cfg.ServiceName, "phase": "5"})
	})
	mux.HandleFunc("GET /ready", func(w http.ResponseWriter, r *http.Request) {
		if err := st.Pool().Ping(r.Context()); err != nil {
			httputil.Fail(w, apperrors.New(apperrors.CodeInternal, "database not ready"))
			return
		}
		stt, _ := st.GetSyncState(r.Context(), chainKey)
		httputil.OK(w, map[string]any{"status": "ready", "sync": stt})
	})
	metrics.Register(mux, cfg.ServiceName)

	srv := &http.Server{Addr: fmt.Sprintf(":%d", cfg.HTTPPort), Handler: metrics.Wrap(cfg.ServiceName, cors(mux))}
	go func() {
		log.Info("listening", zap.Int("port", cfg.HTTPPort), zap.String("chain", chainKey))
		_ = srv.ListenAndServe()
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	close(stop)
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	_ = srv.Shutdown(shutdownCtx)
}

type watchEntry struct {
	addr  string
	label string
}

func parseWatchList() []watchEntry {
	var out []watchEntry
	add := func(envKey, label string) {
		if a := os.Getenv(envKey); a != "" {
			out = append(out, watchEntry{addr: a, label: label})
		}
	}
	add("INDEXER_AMM_PAIR", "AMM Pair")
	add("INDEXER_AMM_ROUTER", "AMM Router")
	add("INDEXER_LAB_TOKEN", "LAB")
	add("INDEXER_LAB_USD", "LUSD")
	add("INDEXER_AMM_FACTORY", "AMM Factory")
	return out
}

func env(k, def string) string {
	if v := os.Getenv(k); v != "" {
		return v
	}
	return def
}

func envInt(k string, def int) int {
	if v := os.Getenv(k); v != "" {
		var n int
		if _, err := fmt.Sscanf(v, "%d", &n); err == nil {
			return n
		}
	}
	return def
}

func envDuration(k string, def time.Duration) time.Duration {
	if v := os.Getenv(k); v != "" {
		if d, err := time.ParseDuration(v); err == nil {
			return d
		}
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
