## Context

Auth-svc is the authentication and user management service. It generates JWT tokens that all other IM services validate. Password security uses bcrypt. Login rate limiting uses Redis TTL counters.

## Goals / Non-Goals

**Goals:**
- bcrypt password hashing
- JWT token generation and refresh (HS256)
- 5-attempt login lockout with 15-minute window
- Online status with Redis + MySQL dual storage
- Multi-tenant user isolation

**Non-Goals:**
- OAuth/Social login (future)
- Role-based access control (beyond basic user status)

## Decisions

1. **bcrypt for passwords**: Industry standard for password hashing. Default cost factor for reasonable performance.
2. **HS256 JWT**: Symmetric signing using shared secret. Consistent with existing services. Token expiry configured via config (default 24h).
3. **Dual status storage**: Redis for fast reads (2h TTL), MySQL for persistence. Cache-first on read, fallback to DB.
4. **Login attempts in Redis**: INCR with TTL. Simple counter approach - no persistent storage needed.
5. **Public login/refresh routes**: Auth endpoints must be accessible without JWT. All other endpoints require auth middleware.

## Risks / Trade-offs

- [Shared secret] All services share the same JWT secret. Secret rotation requires coordinated deployment.
- [Login lockout] Redis-based lockout may be lost on Redis restart - acceptable as it resets the lockout window.
