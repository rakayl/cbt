# Deployment

1. Rotate all values in `.env` or Kubernetes `Secret`.
2. Run PostgreSQL migration from `backend/migrations/202605130001_enterprise_cbt_schema.sql`.
   For a new local environment, migrations also seed `admin@example.edu` / `ChangeMe123!`; rotate or delete that account before public exposure.
3. Build images: `docker build -t enterprise-cbt-backend ./backend` and `docker build -t enterprise-cbt-frontend ./frontend`.
4. Build worker image with `docker build -f backend/Dockerfile.worker -t enterprise-cbt-worker ./backend`.
5. Deploy local production stack with `docker compose -f docker-compose.production.yml --env-file .env up -d --build`.
6. Deploy Kubernetes with `kubectl apply -f k8s/`.
7. Configure PostgreSQL HA externally with Patroni/Cloud SQL/RDS Multi-AZ, Redis Cluster, RabbitMQ quorum queues, and S3 lifecycle policy.
