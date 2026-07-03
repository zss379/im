## ADDED Requirements

### Requirement: Rate limit rule management
The system SHALL allow administrators to manage rate limit rules through CRUD operations.

#### Scenario: List rate limit rules
- **WHEN** admin sends GET /api/v1/rate-limit/rules
- **THEN** system returns all configured rate limit rules

#### Scenario: Create a rate limit rule
- **WHEN** admin sends POST /api/v1/rate-limit/rules with target_type, max_count, and time_window_seconds
- **THEN** system creates the rule

#### Scenario: Update a rate limit rule
- **WHEN** admin sends PUT /api/v1/rate-limit/rules/:rule_id with fields to update
- **THEN** system updates the rule

#### Scenario: Delete a rate limit rule
- **WHEN** admin sends DELETE /api/v1/rate-limit/rules/:rule_id
- **THEN** system deletes the rule

### Requirement: Sliding window rate limiting
The system SHALL enforce rate limits using Redis ZSET-based sliding windows.

#### Scenario: Under rate limit returns passed
- **WHEN** user sends POST /api/v1/rate-limit/check within allowed count
- **THEN** system returns passed=true with remaining count

#### Scenario: Over rate limit returns blocked
- **WHEN** user exceeds the max count within the time window
- **THEN** system returns passed=false with remaining=0

#### Scenario: Old entries expire from window
- **WHEN** entries outside the time window are cleaned up
- **THEN** they no longer count toward the rate limit

#### Scenario: No rule configured returns passed
- **WHEN** no rate limit rule exists for the target type
- **THEN** system returns passed=true with remaining=-1

### Requirement: Target-type based rate limiting
The system SHALL support different rate limits for different target types (user/bot).

#### Scenario: User and bot have separate limits
- **WHEN** a user and a bot are checked against their respective rules
- **THEN** each target type uses its own configured limit

#### Scenario: Default seed rules exist
- **WHEN** the service starts with an empty database
- **THEN** system seeds default rules: user=5/s, bot=10/s
