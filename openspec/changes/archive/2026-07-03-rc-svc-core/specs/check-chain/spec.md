## ADDED Requirements

### Requirement: Combined check chain
The system SHALL provide a combined endpoint that runs sensitive word, rate limit, and file limit checks in sequence.

#### Scenario: All checks pass returns passed
- **WHEN** content passes all three checks (sensitive, rate, file)
- **THEN** system returns passed=true with individual check results

#### Scenario: Sensitive word block fails the chain
- **WHEN** content contains a blocked sensitive word
- **THEN** system returns passed=false with sensitive_check showing blocked

#### Scenario: Rate limit failure fails the chain
- **WHEN** sender exceeds rate limit
- **THEN** system returns passed=false with rate_limit_check showing not passed

#### Scenario: File limit failure fails the chain
- **WHEN** file exceeds size limit
- **THEN** system returns passed=false with file_limit_check showing not passed

#### Scenario: Individual check failures are non-blocking
- **WHEN** a check encounters an error
- **THEN** system logs the warning and continues to the next check

#### Scenario: Empty fields skip checks
- **WHEN** content is empty, sender_id is 0, or file_type is empty
- **THEN** system skips the corresponding check
