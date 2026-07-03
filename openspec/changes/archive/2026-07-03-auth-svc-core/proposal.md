## Why

Authentication and user management is the foundational service that all other IM services depend on. The auth-svc provides login/logout, JWT token generation, user account management, and online status tracking.

## What Changes

- New microservice: `auth-svc` (port 8081)
- Login with bcrypt password verification and JWT token generation
- Token refresh endpoint
- User CRUD (create, read, update, change password)
- Online status management (online, busy, DND, offline) with Redis cache
- Login attempt rate limiting (5 attempts lockout)

## Capabilities

### New Capabilities
- `user-auth`: Login, token management, user account CRUD, online status

### Modified Capabilities

<!-- No existing specs modified -->

## Impact

- New service `auth-svc` under `services/auth-svc/`
- New DB tables: `auth_user`, `auth_user_status`
- Redis DB 1 for status cache and login attempt tracking
- Login and refresh endpoints are public (no JWT required)
- All other endpoints require JWT auth
