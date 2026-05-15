# Enterprise CBT SaaS Kampus

Production-grade foundation for a multi-tenant campus CBT SaaS platform.

## Stack

- Backend: Golang, Fiber, PostgreSQL, Redis, RabbitMQ, WebSocket, JWT, Clean Architecture.
- Frontend: React, Vite, Tailwind CSS, TanStack Query, Zustand, React Hook Form, Zod.
- Infrastructure: Docker, Kubernetes, Nginx, Prometheus, Grafana, Loki, Sentry-ready, MinIO/S3.

## Quick Start

```sh
docker compose up -d --build
```

Default seeded admin for local smoke testing:

- Email: `admin@example.edu`
- Password: `ChangeMe123!`
- Tenant ID: `10000000-0000-0000-0000-000000000001`

For production-style config validation you can still use:

```sh
docker compose -f docker-compose.production.yml --env-file .env config --quiet
```

## Backend Verification

```sh
cd backend
go test ./...
go build ./cmd/api
go build ./cmd/worker
```

## Documentation

- `docs/ARCHITECTURE.md`
- `docs/ERD.md`
- `docs/API.md`
- `docs/openapi.yaml`
- `docs/DATABASE_SCHEMA.md`
- `docs/KUBERNETES.md`
- `docs/DEPLOYMENT.md`
- `docs/RECOVERY_FLOW.md`
- `docs/WEBSOCKET.md`
- `docs/SCALABILITY.md`
- `docs/SECURITY.md`
- `docs/PRODUCTION_CHECKLIST.md`
