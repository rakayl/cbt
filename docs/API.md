# API Documentation

Base path: `/api/v1`.

Public auth endpoints:
- `POST /auth/login`
- `POST /auth/register`
- `POST /auth/forgot-password`
- `POST /auth/refresh`

Protected module endpoints follow the same production CRUD contract:
- `GET /{module}?page=1&limit=20&search=keyword`
- `POST /{module}`
- `GET /{module}/{id}`
- `PUT /{module}/{id}`
- `DELETE /{module}/{id}`
- `GET /{module}/ws` for authenticated WebSocket.

Required headers: `Authorization: Bearer <access_token>`, `X-Tenant-ID`, optional request signing headers in production for non-GET requests.

Exam engine endpoints:
- `POST /exam-sessions/start` validates exam token, creates session, randomizes questions/options, and stores server timer state.
- `POST /exam-sessions/autosave` upserts answer deltas and preserves recovery state.
- `POST /exam-sessions/{id}/reconnect` validates session state against server time and returns synchronized countdown/status.
- `POST /exam-sessions/auto-submit-expired` submits expired sessions from worker/ops automation.

Proctoring endpoints:
- `POST /proctoring/events` records anti-cheat and AI proctoring events, writes browser/face activity logs, scores suspicious activity, and publishes `proctoring_queue` plus `analytics_queue`.

Billing endpoints:
- `GET /billing/usage` returns tenant subscription, feature flags, and quota consumption.
- `POST /billing/checkout-intent` creates a payment-provider-ready checkout intent.
- `POST /billing/change-plan` updates tenant subscription plan after payment/provider confirmation.

Reporting endpoints:
- `POST /reports/export` generates downloadable `pdf`, `excel`, or `csv` reports for `tenant_analytics`, `lecturer_report`, and `transcript`.

Question bank endpoints:
- `POST /question-banks/{id}/import` imports CSV, XLSX, or DOCX into questions/options.
- `POST /question-banks/{id}/media` uploads image/audio/video assets to S3/MinIO and returns a presigned URL.
