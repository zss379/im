## 1. Project Setup

- [x] 1.1 Create project directory structure and go.mod
- [x] 1.2 Create Makefile, config.yaml, and config loading

## 2. Data Layer

- [x] 2.1 Define Department, ContactProfile, UserDept models and request/response types
- [x] 2.2 Implement MySQLRepo with department CRUD, member search, user-dept associations
- [x] 2.3 Implement batch sync (BatchUpsertProfiles, BatchUpsertDepartments)

## 3. Business Logic

- [x] 3.1 Implement recursive department tree builder
- [x] 3.2 Implement department CRUD with children check on delete
- [x] 3.3 Implement member search with keyword + dept_id filter
- [x] 3.4 Implement member detail with phone masking
- [x] 3.5 Implement member update with dept association
- [x] 3.6 Implement HR batch sync

## 4. HTTP Layer

- [x] 4.1 Create unified response helpers
- [x] 4.2 Register all 11 REST endpoints in ContactHandler
- [x] 4.3 Create JWT auth, logger, and recovery middleware

## 5. Service Entrypoint

- [x] 5.1 Create cmd/main.go with full service wiring
- [x] 5.2 Add Prometheus metrics
- [x] 5.3 Implement graceful shutdown

## 6. Verification

- [x] 6.1 Verify type consistency and code completeness
