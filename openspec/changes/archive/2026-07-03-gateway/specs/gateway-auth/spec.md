## ADDED Requirements

### Requirement: Edge JWT validation
The gateway SHALL validate JWT tokens on all incoming requests except whitelisted public paths.

#### Scenario: Valid token forwarded
- **WHEN** a request has a valid JWT in the `Authorization: Bearer <token>` header
- **THEN** the gateway SHALL extract user_id, tenant_id from the token claims
- **THEN** the gateway SHALL add `X-User-ID` and `X-Tenant-ID` headers before proxying

#### Scenario: Missing token on protected route
- **WHEN** a protected route is accessed without an `Authorization` header
- **THEN** the gateway SHALL return HTTP 401

#### Scenario: Invalid token on protected route
- **WHEN** a protected route is accessed with an expired or invalid JWT
- **THEN** the gateway SHALL return HTTP 401

#### Scenario: Public routes bypass auth
- **WHEN** a request hits `POST /api/v1/auth/login` or `POST /api/v1/auth/refresh`
- **THEN** the gateway SHALL proxy the request WITHOUT JWT validation

### Requirement: Rate limiting
The gateway SHALL limit request rate per client IP using a configurable token bucket.

#### Scenario: Under rate limit
- **WHEN** a client sends requests within the allowed rate
- **THEN** the gateway SHALL proxy the request normally

#### Scenario: Exceeded rate limit
- **WHEN** a client exceeds the configured rate limit
- **THEN** the gateway SHALL return HTTP 429 with `Retry-After` header

### Requirement: CORS handling
The gateway SHALL handle CORS preflight requests and set appropriate CORS headers on all responses.

#### Scenario: CORS preflight
- **WHEN** a browser sends an `OPTIONS` request
- **THEN** the gateway SHALL respond with appropriate CORS headers and HTTP 200

#### Scenario: CORS headers on proxied responses
- **WHEN** a proxied response passes through the gateway
- **THEN** the gateway SHALL add CORS headers to the response
