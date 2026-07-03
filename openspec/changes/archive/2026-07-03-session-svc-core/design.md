## Context

Session management (conversation list, pin, mute, unread counts) is currently unhandled by any IM service. The OpenIM engine handles message routing but does not provide business-level session metadata management. The session-svc fills this gap, supporting the session list UI on all clients.

## Goals / Non-Goals

**Goals:**
- Session CRUD with soft-delete (local deletion preserves cloud messages)
- Pin/unpin with max 5 pinned enforcement
- Session-level mute (do-not-disturb)
- Unread count tracking and batch sync across devices
- Follow same architecture as group-svc for consistency

**Non-Goals:**
- Message storage or retrieval (handled by OpenIM)
- Real-time push or WebSocket (handled by OpenIM)
- Session sorting or merging logic for cross-device reconciliation

## Decisions

1. **Standalone service (port 8083)**: Follows the same pattern as group-svc, msg-svc, rc-svc for operational consistency.
2. **Soft delete**: Session deletion uses `is_deleted` flag instead of hard delete, allowing reactivation.
3. **Deterministic conversation_id**: Single chat uses `s_{minUID}_{maxUID}`, group uses `g_{groupID}`, bot uses `b_{botID}` — ensures consistent ID regardless of which user creates the session.
4. **MySQL + Redis**: Session metadata in MySQL (source of truth), unread counts cached in Redis for fast cross-device sync.
5. **Unique constraint on (user_id, conversation_id)**: Prevents duplicate sessions per user per conversation.

## Risks / Trade-offs

- [Unread sync latency] Redis cache may briefly diverge from MySQL — acceptable since unread counts are eventually consistent and corrected on next full load
- [Session list performance] MySQL index on `(user_id, is_deleted, is_pinned, last_message_at)` is critical for list query performance
