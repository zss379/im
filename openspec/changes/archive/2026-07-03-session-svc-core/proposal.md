## Why

IM session management (conversation list, pin, mute, unread sync) is a core feature that currently has no dedicated service. The session-svc provides a centralized session metadata service to support the session list UI on all clients, replacing ad-hoc client-side session management.

## What Changes

- New microservice: `session-svc` (port 8083)
- Session CRUD (create, list, get, soft-delete)
- Pin/unpin sessions (max 5 pinned)
- Mute/unmute session-level do-not-disturb
- Unread count tracking and batch sync across devices
- Redis cache for unread count fast access
- Follows same architecture as group-svc: Gin + GORM + JWT + Prometheus

## Capabilities

### New Capabilities
- `session-management`: Session lifecycle, list, pin, mute, soft-delete, unread tracking

### Modified Capabilities

<!-- No existing specs are being modified -->

## Impact

- New service `session-svc` under `services/session-svc/`
- New DB tables: `session`
- Redis DB 3 for unread cache
- No breaking changes to existing services
