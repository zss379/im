## Why

The platform needs a dedicated group management service to handle group CRUD, member management, role-based permissions, mute controls, and join request workflows. Currently group operations lack a centralized service layer with proper permission enforcement and capacity limits.

## What Changes

- New `group-svc` microservice for group and member management
- Group CRUD: create, dismiss, exit, transfer ownership, update profile
- Member management: batch add/remove, role assignment (owner/admin/member), member search
- Permission hierarchy: owner > admin > member, strict enforcement
- Mute controls: single user mute and global group mute
- Join request system: verification toggle, request approval
- Capacity limits: 2000 members/group, 500 groups/user, 200 created groups/user, 5000 char notice

## Capabilities

### New Capabilities
- `group-management`: Group CRUD (create, dismiss, exit, transfer, update info) with capacity limits
- `member-management`: Batch add/remove members, role-based permissions, member listing and search
- `group-mute`: Single user mute and global group mute with duration control
- `join-request`: Join verification toggle and request approval workflow

### Modified Capabilities
- None (new service)

## Impact

- New Go service at `services/group-svc/` with MySQL for group/member/request data
- REST API under `/api/v1/` with JWT authentication (shared middleware pattern)
- Internal gRPC/HTTP interfaces for message-svc (mute check on send) and contact-svc (member list)
- Capacity limit enforcement at the service layer
