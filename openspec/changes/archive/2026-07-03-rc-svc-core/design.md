## Context

The Risk Control Service (rc-svc) centralizes content safety and rate limiting for the IM platform. Currently, sensitive word filtering, rate limiting, and file size validation are handled in individual services without a consistent policy layer. A dedicated service provides unified policy management and enforcement across all channels.

## Goals / Non-Goals

**Goals:**
- Centralized sensitive word detection with DFA engine (O(n) matching)
- Redis-based sliding window rate limiting per target (user/bot)
- Configurable file upload limits per file type
- REST API for CRUD operations on all rule/config types
- Combined check chain endpoint for message-svc integration
- Prometheus metrics for observability

**Non-Goals:**
- Real-time policy hot-reload (requires full DFA rebuild; acceptable for low update frequency)
- Distributed rate limiting across multiple rc-svc instances (single Redis instance handles this)
- ML-based content moderation (rule-based only)

## Decisions

- **Go + Gin**: Consistent with existing message-svc tech stack; shared patterns reduce maintenance burden
- **DFA Trie for sensitive words**: O(n) matching independent of dictionary size; case-insensitive via rune normalization. Alternatives considered: regex (slow for large dictionaries), Aho-Corasick (faster but more complex; DFA is sufficient)
- **Redis ZSET for rate limiting**: Atomic sliding window with ZREMRANGEBYSCORE cleanup. Alternatives considered: fixed window counters (allows bursts at window boundary), token bucket (more complex; ZSET sliding window is simpler and sufficient)
- **GORM for MySQL**: Consistent with message-svc; AutoMigrate eliminates manual schema management for initial development
- **Interface-based handler design**: Rate limit and file limit handlers use interface injection rather than concrete service types, enabling isolated unit testing
- **Async DFA refresh**: Word CRUD triggers goroutine-based DFA rebuild (10s timeout); avoids blocking the API response while ensuring eventual consistency

## Risks / Trade-offs

- DFA rebuild on every word CRUD → Mitigation: acceptable for low-frequency updates; batch import supported
- Single Redis instance as rate limiter → Mitigation: Redis is local to the deployment; can be clustered if needed
- GORM AutoMigrate in production → Mitigation: safe for additive changes; schema migrations for destructive changes require manual review
