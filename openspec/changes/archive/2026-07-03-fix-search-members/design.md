## Context

`SearchMembers` in `repo.go` currently delegates to `ListMembers`, ignoring the `keyword` parameter entirely. The `GroupMember` model stores only `user_id` (int64) as a member identifier — no display name or nickname is stored in group-svc.

## Goals / Non-Goals

**Goals:**
- `SearchMembers` filters by `user_id` when keyword is a numeric string
- `SearchMembers` returns all members when keyword is empty (preserving existing behavior)
- Non-numeric keyword returns empty results (no name-based search possible without user-svc join)

**Non-Goals:**
- No user name / display name search (requires user-svc integration — out of scope)
- No API changes (request/response unchanged)
- No index changes (existing `idx_group_user` covers the query)

## Decisions

| Decision | Choice | Rationale |
|---|---|---|
| Keyword type | Numeric user_id only | `GroupMember` lacks name fields; joining user-svc adds complexity beyond this fix |
| Empty keyword | Fall through to `ListMembers` | Preserves backward compatibility for callers that don't need filtering |
| Match mode | Exact match on user_id | Index-friendly; IN clause supports potential batch operations later |

## Risks / Trade-offs

- Non-numeric keyword returns empty results — acceptable as the endpoint's purpose is ID-based search. Future name search would require user-svc integration or denormalizing display names into group_member.
