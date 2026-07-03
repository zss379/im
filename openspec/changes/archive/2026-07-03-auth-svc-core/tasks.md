## 1. Project Setup

- [x] 1.1 Create project directory structure and go.mod with bcrypt dependency
- [x] 1.2 Create Makefile, config.yaml, and config loading

## 2. Data Layer

- [x] 2.1 Define User, UserStatus models and all request/response types
- [x] 2.2 Implement MySQLRepo with user CRUD, status management, batch queries
- [x] 2.3 Implement Redis Cache for status and login attempt tracking

## 3. Business Logic

- [x] 3.1 Implement login with bcrypt verify, JWT generation, attempt tracking
- [x] 3.2 Implement token refresh
- [x] 3.3 Implement user CRUD and password change
- [x] 3.4 Implement status management with Redis+MySQL dual storage

## 4. HTTP Layer

- [x] 4.1 Create unified response helpers
- [x] 4.2 Register public auth routes and protected user routes
- [x] 4.3 Create JWT auth, logger, and recovery middleware

## 5. Service Entrypoint

- [x] 5.1 Create cmd/main.go with public routes + protected route groups
- [x] 5.2 Add Prometheus metrics
- [x] 5.3 Implement graceful shutdown

## 6. Verification

- [x] 6.1 Verify type consistency and code completeness
