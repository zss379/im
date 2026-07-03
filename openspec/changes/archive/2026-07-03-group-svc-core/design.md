## Context

The group-svc provides centralized group management for the IM platform. Groups are core collaboration containers with hierarchical role permissions (owner > admin > member), supporting up to 2000 members per group. The service handles group lifecycle, member administration, mute controls, and join requests, with strict enforcement of capacity limits and role-based authorization.

## Goals / Non-Goals

**Goals:**
- Group CRUD with capacity limit enforcement
- Member management with batch operations and role-based permissions
- Mute controls (single user and global)
- Join request system with verification toggle
- Strict role hierarchy enforcement (owner > admin > member)
- Performance: 2000-member group load < 1s, member changes sync < 2s

**Non-Goals:**
- Real-time notification of group changes (handled by message-svc via Kafka)
- Group chat message routing (handled by OpenIM)
- Group file/sharing storage (handled by file-svc)
- Department/all-company groups (special group types, future scope)

## Decisions

- **Role hierarchy enforced in service layer**: All mutating operations check role permissions before execution. Owner has absolute authority; admin cannot remove owner or other admins; members have minimal write permissions.
- **MySQL for group/member data**: Relational data with clear ownership and membership relationships. GORM with auto-migration for schema management.
- **Redis for mute state**: Mute flags are time-sensitive and frequently read (checked on every message send). Redis TTL handles automatic mute expiration.
- **Batch member operations use transactions**: Add/remove members within a single DB transaction to maintain consistency. Maximum 10 per batch per PRD constraint.
- **Internal API for message-svc integration**: Group-svc exposes an internal check endpoint for message-svc to verify mute status and send permissions before message delivery.
- **Capacity limits as configurable constants**: 2000 members/group, 500 groups/user, 200 created groups/user, 5000 chars notice — enforced at service layer.

## Risks / Trade-offs

- [Permission enforcement] Critical for data safety → all mutations check role in a single transaction; never trust client-provided role values
- [Capacity limits] Hard limits may need adjustment per tenant → stored as configurable service constants, not hardcoded magic numbers
- [Admin escalation] Admin could attempt privilege escalation → admin creation requires owner permission; demotion only by owner
