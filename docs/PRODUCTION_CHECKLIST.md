# Production Checklist

- Rotate every secret in `.env`, Kubernetes `Secret`, registry, and CI/CD.
- Set `APP_ENV=production`, `REQUEST_SIGNING_REQUIRED=true` for signed external integrations, and `IP_WHITELIST` for restricted admin networks.
- Run migrations in order from `backend/migrations`.
- Use PostgreSQL HA with WAL archiving, read replicas, PgBouncer, daily base backup, and tested PITR.
- Use Redis Cluster for sessions, timers, locks, autosave state, and WebSocket fan-out.
- Use RabbitMQ quorum queues for `grading_queue`, `analytics_queue`, `report_queue`, `recovery_queue`, `proctoring_queue`, and `notification_queue`.
- Put Nginx/Ingress behind WAF/CDN/DDoS protection.
- Enable Sentry DSN, Prometheus scraping, Grafana dashboards, and Loki log shipping.
- Run `go test ./...`, `go build ./cmd/api`, `go build ./cmd/worker`, `docker compose -f docker-compose.production.yml config --quiet`, and `scripts/smoke-test.ps1`.
