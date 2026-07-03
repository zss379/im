## Why

Currently 9 microservices expose their HTTP ports directly. Clients must know 9 different addresses, each service duplicates JWT auth logic, and there's no centralized observability (CORS, rate limiting, request logging). An API Gateway provides a single entry point that handles cross-cutting concerns at the edge.

## What Changes

- New `gateway-svc` in `services/gateway-svc/` — a stateless reverse proxy
- Route-based forwarding: path prefix → backend service URL
- JWT validation at edge; pass `X-User-ID`, `X-Tenant-ID` to backends via headers
- Rate limiting per client IP (token bucket, configurable)
- Unified CORS, Prometheus metrics (request count, latency, status), access log
- Public routes whitelist (no JWT required): `POST /api/v1/auth/login`, `POST /api/v1/auth/refresh`
- Gateway listens on port 8080; backends become internal-only (remove their port exposure)
- `/health` endpoint returning overall upstream health

## Capabilities

### New Capabilities
- `api-gateway`: Route-based HTTP reverse proxy forwarding to backend microservices
- `gateway-auth`: Edge JWT validation and user context propagation via headers

### Modified Capabilities

None — no existing specs to change.

## Impact

- New module `services/gateway-svc/` (no DB, no Redis — pure stateless proxy)
- Backend services no longer need to handle JWT individually (but keep their auth middleware for defense-in-depth)
- Updates docker-compose: add gateway, remove public port mappings from backends, add `gateway` network alias
- All client traffic changes from `http://host:{svc-port}` to `http://host:8080/api/v1/{service}`
