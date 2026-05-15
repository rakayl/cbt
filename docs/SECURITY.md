# Security Hardening

Implemented foundations: JWT access/refresh token separation, refresh session storage in Redis, RBAC middleware, tenant middleware, rate limiting, secure headers, CORS, CSRF protection for cookie flows, optional HMAC request signing, optional IP whitelist, audit request logging, MFA/SSO-ready schema, IP/device/fingerprint session schema, and soft delete.

Production edge should add WAF, Cloudflare or equivalent DDoS protection, KMS-backed encryption at rest, external secret manager, and mTLS for service-to-service traffic. Request signing can be enforced with `REQUEST_SIGNING_REQUIRED=true`; IP allowlisting can be enforced with `IP_WHITELIST=10.0.0.0/8,203.0.113.10`.
