## Context

The address book/contacts module (IM-MOD-005) requires department hierarchy management, member profile caching, and multi-dimensional search. Since auth-svc does not exist yet, contact-svc maintains its own contact profile cache synced from the HR system.

## Goals / Non-Goals

**Goals:**
- Department tree with recursive parent-child hierarchy
- Member search by name and pinyin across the org
- Member profile with phone masking
- User-department multi-membership
- HR batch sync endpoint

**Non-Goals:**
- User authentication or token management (handled by auth-svc)
- Real-time online status tracking
- Group management or "my groups" features (handled by group-svc)

## Decisions

1. **Three-table design**: `contact_department` (tree), `contact_profile` (user cache), `contact_user_dept` (M:N membership) — clean separation of concerns.
2. **Recursive tree**: Departments use parent_id adjacency list. Full list fetched and built into a tree in application memory — fine for typical org sizes (<5000 departments).
3. **Pinyin search**: `name_py` field stored in profile for pinyin initial matching, populated during HR sync.
4. **Phone masking**: Done at the service layer before returning responses.
5. **HR sync is upsert-based**: Uses `Save` (INSERT ON DUPLICATE KEY UPDATE) for idempotent sync.

## Risks / Trade-offs

- [Tree performance] Loading all depts into memory for tree building is fine for <5000 nodes; beyond that, consider lazy-loading per level
- [Profile staleness] Contact profiles are synced from HR — lag between HR update and sync could show stale data
