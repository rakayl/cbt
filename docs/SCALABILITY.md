# Scalability for 100,000 Concurrent Users

- API pods scale with HPA from 6 to 200 replicas.
- WebSocket state is distributed through Redis Cluster.
- PostgreSQL uses HA primary, read replicas, PgBouncer, indexes, and partitioning for logs/answers/proctoring data.
- RabbitMQ queues isolate grading, analytics, reporting, recovery, proctoring, and notification workloads.
- Static frontend assets are CDN ready.
- Object uploads go directly to S3/MinIO with presigned URLs.
