## 1. Project Setup

- [x] 1.1 Create go.mod with dependencies
- [x] 1.2 Create config.yaml and config loader
- [x] 1.3 Create Makefile

## 2. Models

- [x] 2.1 Define Group model (id, tenant_id, name, avatar, notice, owner_id, status, created_at, updated_at)
- [x] 2.2 Define GroupMember model (id, group_id, user_id, role, muted_until, joined_at)
- [x] 2.3 Define JoinRequest model (id, group_id, user_id, status, created_at)
- [x] 2.4 Define request/response types for all endpoints

## 3. Repository Layer

- [x] 3.1 Implement GroupRepo (Create, Get, Update, Delete, ListByUser, Search)
- [x] 3.2 Implement MemberRepo (BatchAdd, BatchRemove, List, Search, SetRole, GetByGroupAndUser)
- [x] 3.3 Implement JoinRequestRepo (Create, Approve, Reject, List, GetPending)
- [x] 3.4 Implement AutoMigrate and capacity constants
- [x] 3.5 Implement Redis cache for mute state

## 4. Service Layer

- [x] 4.1 Implement group CRUD with capacity enforcement
- [x] 4.2 Implement member management with role authorization
- [x] 4.3 Implement mute controls (single + global) with Redis TTL
- [x] 4.4 Implement join request workflow

## 5. Handler Layer

- [x] 5.1 Create response helpers (reuse existing pattern)
- [x] 5.2 Implement group handler (create, dismiss, transfer, exit, update, get, list)
- [x] 5.3 Implement member handler (list, add, remove, set role, search)
- [x] 5.4 Implement mute handler (mute member, unmute, global mute, check)
- [x] 5.5 Implement join request handler (request, approve, reject, list, config)

## 6. Middleware

- [x] 6.1 Implement JWT auth middleware
- [x] 6.2 Implement logger and recovery middleware
- [x] 6.3 Implement role-check middleware

## 7. Metrics

- [x] 7.1 Create Prometheus metrics for group operations

## 8. Main Entry Point

- [x] 8.1 Create cmd/main.go with full service wiring
