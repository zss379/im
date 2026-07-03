## Why

The platform needs a centralized risk control service (rc-svc) to enforce content safety, rate limiting, and file upload policies across all IM channels. Currently, sensitive word filtering, rate limiting, and file size validation are handled ad-hoc in different services, leading to inconsistent enforcement and duplicated logic.

## What Changes

- New `rc-svc` microservice for centralized risk control
- DFA-based sensitive word engine with configurable strategies (replace/block/log)
- Redis sliding window rate limiter per target (user/bot)
- File upload limit configuration per file type (image/video/document)
- Full CRUD management APIs for sensitive words, rate limit rules, and file limit configs
- Combined check chain endpoint for message-svc integration
- Prometheus metrics for all check operations

## Capabilities

### New Capabilities
- `sensitive-word-filter`: DFA-based sensitive word detection with replace/block/log strategies, case-insensitive matching, longest-match semantics
- `rate-limiter`: Redis sliding window rate limiting per target type (user/bot), configurable max count and time window
- `file-upload-limit`: File type-based upload size and extension restrictions
- `check-chain`: Combined validation pipeline that runs sensitive word, rate limit, and file limit checks in sequence

### Modified Capabilities
- None (new service, no existing specs modified)

## Impact

- New Go service at `services/rc-svc/` with its own config, models, repos, engine, service, handler layers
- Requires MySQL (sensitive words, rules configs) and Redis (rate limiter state)
- Exposes REST API under `/api/v1/` with JWT authentication
- message-svc will call the check chain endpoint for message validation
- New Prometheus metrics: `rc_svc_sensitive_check_total`, `rc_svc_rate_limit_check_total`, `rc_svc_file_limit_check_total`
