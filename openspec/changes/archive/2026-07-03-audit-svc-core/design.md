## Context

Audit logging (IM-MOD-008) requires write throughput >1000 TPS and million-record queries <5s. Two log types: admin operations (lower volume, 2yr retention) and message audit (higher volume, 6mo retention).

## Goals / Non-Goals

**Goals:**
- Structured admin op and message audit logs
- Multi-condition search (time, operator, type, keyword, sensitive flag)
- Retention cleanup
- Batch message log writes

**Non-Goals:**
- Log modification or deletion (enforced at DB level)
- Real-time log streaming

## Decisions

1. **Single MySQL with indexes**: Composite index on (tenant_id, created_at) for range queries. Separate indexes on operator_id, op_type, sender_id, session_id for filtered queries.
2. **Batch writes**: Message audit logs batch at 100/insert for write efficiency.
3. **Soft cleanup**: Cleanup endpoint is manual/triggered—no scheduler built-in. Can be called by cron job.
4. **No Redis**: Logs are write-heavy and read-rare for compliance. Redis cache adds complexity without benefit.

## Risks / Trade-offs

- [MySQL write pressure] At >1000 TPS, MySQL may become a bottleneck. Future: switch to Kafka + batch consumer for message audit logs.
- [Table size] Message audit logs grow fast. Partitioning by month recommended for production.
