# Observability (Phase 6)

Prometheus scrapes `/metrics` from every Go service. Grafana ships with a pre-built overview dashboard.

## Metrics

| Metric | Description |
|--------|-------------|
| `cel_service_up{service}` | Gauge set to 1 at startup |
| `cel_http_requests_total{service,method,route,status}` | Request counter |
| `cel_http_request_duration_seconds` | Latency histogram |

Implemented in `packages/go-common/metrics`.

## Local stack

1. Start Postgres/Redis: `make up`
2. Start Go backends (each exposes `:PORT/metrics`):

```bash
make run-account & make run-matching & make run-order &
# … other services as needed
```

3. Start monitoring:

```bash
make monitoring-up
```

| URL | Default login |
|-----|----------------|
| Prometheus | http://localhost:9090 |
| Grafana | http://localhost:3000 — `admin` / `lab` |

Open dashboard **Crypto Exchange Lab — Overview** (folder CEL).

Stop monitoring: `make monitoring-down`

## Troubleshooting

- **Empty targets:** backends must run on the host; Prometheus in Docker reaches them via `host.docker.internal`.
- **WSL2:** `host-gateway` in `docker-compose.monitoring.yml` maps the host; if scrape fails, replace targets in `infra/prometheus/prometheus.yml` with your WSL IP.
- Reload Prometheus config: `curl -X POST http://localhost:9090/-/reload` (lifecycle enabled in compose).
