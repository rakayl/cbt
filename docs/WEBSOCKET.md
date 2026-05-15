# WebSocket Architecture

WebSocket endpoint exists per module at `/api/v1/{module}/ws`. Authentication uses JWT before upgrade through the protected route; browser clients may pass the token as `?token=` because native WebSocket cannot set arbitrary authorization headers. Horizontal scaling uses Redis for shared timer state, active session state, autosave state, and pub/sub fan-out. Backend pods remain stateless; session truth lives in PostgreSQL and Redis.

The exam session WebSocket publishes and subscribes to Redis channel `exam_session:{session_id}:events`, emits heartbeat/timer ticks, and supports reconnecting browser clients with exponential backoff in the React client.

Exam timer is server authoritative. Frontend only renders countdown returned by API/WebSocket and persists answer draft locally for offline replay.
