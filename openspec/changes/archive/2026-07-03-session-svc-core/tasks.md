## 1. Project Setup

- [x] 1.1 Create project directory structure and go.mod
- [x] 1.2 Create Makefile, config.yaml, and config loading

## 2. Data Layer

- [x] 2.1 Define Session model and request/response types
- [x] 2.2 Implement MySQLRepo with full CRUD and list queries
- [x] 2.3 Implement Redis Cache for unread count

## 3. Business Logic

- [x] 3.1 Implement session creation with dedup and reactivation
- [x] 3.2 Implement session list with type/unread filtering
- [x] 3.3 Implement pin/unpin with max 5 enforcement
- [x] 3.4 Implement mute/unmute
- [x] 3.5 Implement mark read/unread and batch update
- [x] 3.6 Implement soft delete

## 4. HTTP Layer

- [x] 4.1 Create unified response helpers
- [x] 4.2 Register all 9 REST endpoints in SessionHandler
- [x] 4.3 Create JWT auth, logger, and recovery middleware

## 5. Service Entrypoint

- [x] 5.1 Create cmd/main.go with full service wiring
- [x] 5.2 Add Prometheus metrics
- [x] 5.3 Implement graceful shutdown

## 6. Verification

- [x] 6.1 Verify type consistency and code completeness
