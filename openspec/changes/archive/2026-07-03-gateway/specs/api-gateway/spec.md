## ADDED Requirements

### Requirement: Route-based reverse proxy
The API Gateway SHALL forward incoming HTTP requests to backend microservices based on the request path prefix.

#### Scenario: Forward auth-svc requests
- **WHEN** a request arrives at `/api/v1/auth/*`
- **THEN** the gateway SHALL proxy the request to `auth-svc:8080`

#### Scenario: Forward message-svc requests
- **WHEN** a request arrives at `/api/v1/messages/*`
- **THEN** the gateway SHALL proxy the request to `message-svc:8081`

#### Scenario: Forward bot-svc requests
- **WHEN** a request arrives at `/api/v1/bots/*`
- **THEN** the gateway SHALL proxy the request to `bot-svc:8082`

#### Scenario: Forward session-svc requests
- **WHEN** a request arrives at `/api/v1/sessions/*`
- **THEN** the gateway SHALL proxy the request to `session-svc:8083`

#### Scenario: Forward contact-svc requests
- **WHEN** a request arrives at `/api/v1/contacts/*`
- **THEN** the gateway SHALL proxy the request to `contact-svc:8084`

#### Scenario: Forward audit-svc requests
- **WHEN** a request arrives at `/api/v1/audit/*`
- **THEN** the gateway SHALL proxy the request to `audit-svc:8085`

#### Scenario: Forward group-svc requests
- **WHEN** a request arrives at `/api/v1/groups/*`
- **THEN** the gateway SHALL proxy the request to `group-svc:8086`

#### Scenario: Forward file-svc requests
- **WHEN** a request arrives at `/api/v1/files/*`
- **THEN** the gateway SHALL proxy the request to `file-svc:8087`

#### Scenario: Forward rc-svc requests
- **WHEN** a request arrives at `/api/v1/rc/*`
- **THEN** the gateway SHALL proxy the request to `rc-svc:8088`

### Requirement: Health check endpoint
The gateway SHALL expose a `/health` endpoint returning upstream service health status.

#### Scenario: All upstreams healthy
- **WHEN** all configured backend services respond to health checks
- **THEN** the gateway SHALL return HTTP 200 with JSON `{"status":"ok","upstreams":{...}}`

#### Scenario: Upstream unhealthy
- **WHEN** one or more backend services are unreachable
- **THEN** the gateway SHALL return HTTP 503 with JSON indicating which upstreams are down

### Requirement: Prometheus metrics
The gateway SHALL expose Prometheus metrics at `/metrics` including request count, latency histogram, and status code distribution per route.

#### Scenario: Metrics collection
- **WHEN** a request passes through the gateway
- **THEN** the gateway SHALL record request duration, status code, and route label

### Requirement: Route configuration
The route table SHALL be configurable via YAML without code changes.

#### Scenario: Load routes from config
- **WHEN** the gateway starts
- **THEN** it SHALL read the route table from its config file
