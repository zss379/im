## Context

9 independent Go microservices each expose their own HTTP port with duplicated JWT auth, CORS, and metrics. Clients need to know 9 addresses. The gateway consolidates these cross-cutting concerns at a single entry point.

## Goals / Non-Goals

**Goals:**
- Single HTTP entry point on port 8080
- Path-based routing to all 9 backends
- Edge JWT validation with user context propagation
- Rate limiting per client IP
- Unified CORS, Prometheus metrics, access logging
- Health endpoint aggregating upstream status

**Non-Goals:**
- API aggregation / response transformation (pass-through proxy)
- TLS termination (handled by upstream LB in production)
- Service discovery beyond static config
- WebSocket proxying (handled by OpenIM directly)

## Decisions

**1. Standard library reverse proxy over custom forwarding**
`net/http/httputil.ReverseProxy` handles connection pooling, retries, and buffering. No need for a third-party proxy library.

**2. In-memory token bucket over Redis-backed rate limiting**
The gateway is stateless; Redis adds latency on every request. Per-IP rate limits with a configurable refill rate and bucket size, reset on restart. Acceptable for dev; production can plug in Redis.

**3. JWT validation at edge only (defense-in-depth optional)**
Backend services keep their auth middleware but can be configured to trust the `X-User-ID` header from the gateway. This allows backends to be called directly for testing while gateway is the default path.

**4. Static route table in YAML**
Routes map path prefix → backend URL. No service discovery needed for 9 services. Config reload via SIGHUP signal.

**Architecture:**

```
Client → :8080 → Gateway
                   ├── /api/v1/auth/*      → auth-svc:8080
                   ├── /api/v1/messages/*  → message-svc:8081
                   ├── /api/v1/bots/*      → bot-svc:8082
                   ├── /api/v1/sessions/*  → session-svc:8083
                   ├── /api/v1/contacts/*  → contact-svc:8084
                   ├── /api/v1/audit/*     → audit-svc:8085
                   ├── /api/v1/groups/*    → group-svc:8086
                   ├── /api/v1/files/*     → file-svc:8087
                   ├── /api/v1/rc/*        → rc-svc:8088
                   ├── /health             → local
                   └── /metrics            → local
```

## Risks / Trade-offs

- **Single point of failure** → Mitigated by stateless design; scale horizontally behind a load balancer
- **Rate limiter state lost on restart** → Acceptable for dev; production can add Redis-backed limiter
- **Backend auth header trust** → If a backend is exposed directly, headers can be forged. Keep auth middleware enabled on backends behind gateway.
