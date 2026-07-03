## Why

Audit logging is required for compliance and security (PRD §7). All admin operations and messages must be recorded with retention policies. This service provides the centralized audit store.

## What Changes

- New microservice: `audit-svc` (port 8085)
- Admin operation log: record all admin actions with operator, type, target, result, IP
- Message audit log: record all sent messages with sender, session, content, sensitive flag
- Multi-condition search with time range, operator, type, keyword filters
- Retention: admin logs 2 years, message logs 6 months (with cleanup endpoint)

## Capabilities

### New Capabilities
- `audit-logging`: Admin operation logs and message audit logs with search and retention

### Modified Capabilities

<!-- No existing specs modified -->

## Impact

- New service `audit-svc` under `services/audit-svc/`
- New DB tables: `audit_admin_op_log`, `audit_msg_log`
- Internal API for other services to write audit logs
- No breaking changes
