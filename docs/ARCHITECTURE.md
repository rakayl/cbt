# Enterprise CBT SaaS Kampus Architecture

Layer backend mengikuti alur Handler / Controller -> Service -> Repository -> Model -> Database. Setiap module di `backend/internal/modules` memiliki `model.go`, `repository.go`, `service.go`, `handler.go`, `dto.go`, `route.go`, `validation.go`, `middleware.go`, `websocket.go`, dan `jobs.go`.

Runtime utama: Go Fiber API, PostgreSQL, Redis, RabbitMQ, MinIO/S3, WebSocket, React Vite, TanStack Query, Zustand, React Hook Form, Zod, Docker, Kubernetes, Prometheus, Grafana, Loki, dan Sentry.

Microservice readiness dicapai dengan module boundary, event queue per domain, DTO separation, shared middleware, dan database schema yang dapat dipisah menjadi dedicated database per tenant atau per service.
