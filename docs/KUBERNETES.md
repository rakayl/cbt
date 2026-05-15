# Kubernetes

Kubernetes manifests are available in both `k8s/` and `deployments/kubernetes/`.

Included resources:

- Namespace
- ConfigMap
- Secret
- Backend Deployment
- Worker Deployment
- Frontend Deployment
- Services
- Ingress
- HPA
- PVC
- PostgreSQL backup CronJob

The backend and frontend are horizontally scalable. The worker is independently scalable for recovery, grading, reporting, analytics, proctoring, and notification queue workloads.
