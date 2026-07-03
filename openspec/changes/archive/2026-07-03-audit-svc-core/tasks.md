## 1. Project Setup

- [x] 1.1 Create project directory structure and go.mod
- [x] 1.2 Create Makefile, config.yaml, and config loading

## 2. Data Layer

- [x] 2.1 Define AdminOpLog, MsgAuditLog models with all request/response types
- [x] 2.2 Implement MySQLRepo with create, list (multi-filter), get, cleanup

## 3. Business Logic

- [x] 3.1 Implement admin op log write and search
- [x] 3.2 Implement message audit log write, batch write, and search
- [x] 3.3 Implement retention cleanup

## 4. HTTP Layer

- [x] 4.1 Create unified response helpers
- [x] 4.2 Register all 9 audit endpoints
- [x] 4.3 Reuse JWT auth, logger, and recovery middleware

## 5. Service Entrypoint

- [x] 5.1 Create cmd/main.go with full service wiring
- [x] 5.2 Add Prometheus metrics
- [x] 5.3 Implement graceful shutdown

## 6. Verification

- [x] 6.1 Verify code consistency
