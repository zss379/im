## 1. Project Scaffolding

- [x] 1.1 Create `services/gateway-svc/` with `go.mod`, `cmd/main.go`, `internal/config/config.go`
- [x] 1.2 Add config YAML with route table, JWT secret, rate limit, CORS settings
- [x] 1.3 Implement `config.Load()` and `DefaultConfig()`

## 2. Proxy Core

- [x] 2.1 Implement URL rewriting and reverse proxy creation per route
- [x] 2.2 Build router: match request path prefix → proxy to backend
- [x] 2.3 Add request/response header manipulation (X-Forwarded-*, X-Real-IP)
- [x] 2.4 Implement error handling: backend timeout, connection refused

## 3. Auth Middleware

- [x] 3.1 JWT token extraction and HS256 validation
- [x] 3.2 Public path whitelist bypass
- [x] 3.3 Inject `X-User-ID`, `X-Tenant-ID` headers on valid auth
- [x] 3.4 Return 401 on missing/invalid token

## 4. Rate Limiter

- [x] 4.1 Implement in-memory token bucket per client IP
- [x] 4.2 Configurable refill rate and bucket size
- [x] 4.3 Return 429 with Retry-After on limit exceeded

## 5. CORS & Metrics

- [x] 5.1 CORS middleware: handle OPTIONS preflight, add headers
- [x] 5.2 Prometheus middleware: request count, latency histogram, status codes
- [x] 5.3 Expose `/metrics` endpoint

## 6. Health Check

- [x] 6.1 Implement upstream health check on startup and periodic refresh
- [x] 6.2 Expose `/health` returning aggregated upstream status (200/503)

## 7. Docker & Integration

- [x] 7.1 Create `services/gateway-svc/Dockerfile`
- [x] 7.2 Create `deploy/docker/config/gateway-svc.yaml`
- [x] 7.3 Update `docker-compose.yml`: add gateway service, remove public port mappings from backends
