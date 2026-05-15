# Deployments

This folder mirrors production deployment artifacts in the structure requested by the enterprise blueprint.

- `docker/` contains Docker Compose and Nginx reverse proxy configuration.
- `kubernetes/` contains Kubernetes namespace, deployments, services, ingress, HPA, PVC, secrets, configmaps, worker, and backup CronJob manifests.

Root-level `docker-compose.production.yml`, `nginx/`, and `k8s/` are kept as convenience entrypoints for local commands and CI compatibility.
