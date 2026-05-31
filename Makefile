.PHONY: up down logs ps migrate migrate-sh test-go test-contracts test-all build lint tidy monitoring-up monitoring-down deploy-demo-up deploy-demo-down run-account run-order run-matching

COMPOSE := docker compose -f infra/docker/docker-compose.yml
COMPOSE_MONITORING := $(COMPOSE) -f infra/docker/docker-compose.monitoring.yml --profile monitoring
COMPOSE_DEMO := $(COMPOSE) -f infra/docker/docker-compose.demo.yml
DATABASE_URL ?= postgres://lab:lab@localhost:5432/crypto_exchange_lab?sslmode=disable

up:
	$(COMPOSE) up -d

down:
	$(COMPOSE) down

logs:
	$(COMPOSE) logs -f

ps:
	$(COMPOSE) ps

monitoring-up:
	$(COMPOSE_MONITORING) up -d prometheus grafana

monitoring-down:
	$(COMPOSE_MONITORING) down

deploy-demo-up:
	$(COMPOSE_DEMO) up -d --build

deploy-demo-down:
	$(COMPOSE_DEMO) down

migrate-sh:
	chmod +x scripts/migrate.sh 2>/dev/null || true
	./scripts/migrate.sh

migrate:
	docker exec -i cel-postgres psql -U lab -d crypto_exchange_lab < infra/postgres/migrations/001_phase1_schema.up.sql
	docker exec -i cel-postgres psql -U lab -d crypto_exchange_lab < infra/postgres/migrations/002_phase1_seed.up.sql
	docker exec -i cel-postgres psql -U lab -d crypto_exchange_lab < infra/postgres/migrations/003_phase2_venue.up.sql
	docker exec -i cel-postgres psql -U lab -d crypto_exchange_lab < infra/postgres/migrations/004_phase3_perps.up.sql
	docker exec -i cel-postgres psql -U lab -d crypto_exchange_lab < infra/postgres/migrations/005_phase5_chain_indexer.up.sql
	docker exec -i cel-postgres psql -U lab -d crypto_exchange_lab < infra/postgres/migrations/006_phase7_markets.up.sql

test-go:
	cd packages/matching && go test ./...
	cd packages/go-common && go test ./...
	cd packages/orderflow && go test ./...
	cd services/account-service && go test ./...
	cd services/matching-engine && go test ./...
	cd services/order-service && go test ./...
	cd services/orderbook-dex && go test ./...
	cd packages/perps && go test ./...
	cd services/hyperliquid-engine && go test ./...
	cd services/risk-engine && go test ./...
	cd services/liquidation-engine && go test ./...
	cd services/funding-engine && go test ./...
	cd packages/chainrpc && go test ./...
	cd packages/chainstore && go test ./...
	cd services/rpc-gateway && go test ./...
	cd services/indexer && go test ./...

test-contracts:
	pnpm contracts:test

deploy-amm-sepolia:
	pnpm contracts:deploy:sepolia

test-all: test-go test-contracts

build:
	pnpm build

lint:
	pnpm lint

tidy:
	cd packages/go-common && go mod tidy
	cd packages/matching && go mod tidy
	cd packages/orderstore && go mod tidy
	cd packages/tradeclients && go mod tidy
	cd packages/orderflow && go mod tidy
	cd packages/orderapi && go mod tidy
	cd services/account-service && go mod tidy
	cd services/matching-engine && go mod tidy
	cd services/order-service && go mod tidy
	cd services/orderbook-dex && go mod tidy
	cd packages/perps && go mod tidy
	cd packages/perpstore && go mod tidy
	cd packages/perpservice && go mod tidy
	cd services/hyperliquid-engine && go mod tidy
	cd services/risk-engine && go mod tidy
	cd services/liquidation-engine && go mod tidy
	cd services/funding-engine && go mod tidy
	cd packages/chainrpc && go mod tidy
	cd packages/chainstore && go mod tidy
	cd services/rpc-gateway && go mod tidy
	cd services/indexer && go mod tidy

install:
	pnpm install

run-account:
	go run ./services/account-service/cmd/server

run-order:
	ACCOUNT_SERVICE_URL=http://localhost:8081 MATCHING_ENGINE_URL=http://localhost:8083 \
		go run ./services/order-service/cmd/server

run-matching:
	go run ./services/matching-engine/cmd/server

run-orderbook-dex:
	ACCOUNT_SERVICE_URL=http://localhost:8081 MATCHING_ENGINE_URL=http://localhost:8083 \
		go run ./services/orderbook-dex/cmd/server

run-hyperliquid:
	ACCOUNT_SERVICE_URL=http://localhost:8081 MATCHING_ENGINE_URL=http://localhost:8083 \
		go run ./services/hyperliquid-engine/cmd/server

run-risk:
	go run ./services/risk-engine/cmd/server

run-liquidation:
	HYPERLIQUID_ENGINE_URL=http://localhost:8085 go run ./services/liquidation-engine/cmd/server

run-funding:
	ACCOUNT_SERVICE_URL=http://localhost:8081 go run ./services/funding-engine/cmd/server

run-rpc-gateway:
	go run ./services/rpc-gateway/cmd/server

run-indexer:
	go run ./services/indexer/cmd/server
