## ADDED Requirements

### Requirement: System SHALL record admin operations
The system SHALL persist all admin operations with full context for compliance auditing.

#### Scenario: Create admin op log
- **WHEN** any service calls create admin op log with op_type, operator_id, result, IP
- **THEN** system persists the log with timestamp

#### Scenario: Search admin logs by time range
- **WHEN** auditor queries with start_time and end_time
- **THEN** system returns logs within that range ordered by created_at DESC

#### Scenario: Search admin logs by operator
- **WHEN** auditor filters by operator_id
- **THEN** system returns only logs from that operator

#### Scenario: Search admin logs by type
- **WHEN** auditor filters by op_type
- **THEN** system returns only logs of that operation type

#### Scenario: Get admin log detail
- **WHEN** auditor requests a specific log by ID
- **THEN** system returns the full log including detail field

### Requirement: System SHALL record message audit logs
The system SHALL persist all sent messages for compliance review.

#### Scenario: Create message audit log
- **WHEN** message-svc sends a message audit log with sender, session, content, type
- **THEN** system persists the audit log

#### Scenario: Batch create message audit logs
- **WHEN** message-svc sends batch of audit logs
- **THEN** system persists all logs efficiently

#### Scenario: Search message logs by keyword
- **WHEN** auditor searches with a content keyword
- **THEN** system returns messages containing that keyword

#### Scenario: Filter by sensitive flag
- **WHEN** auditor filters by has_sensitive=true
- **THEN** system returns only messages that hit sensitive words

#### Scenario: Filter by sender
- **WHEN** auditor filters by sender_id
- **THEN** system returns only messages from that sender

### Requirement: System SHALL support log retention cleanup
The system SHALL delete logs older than retention period.

#### Scenario: Trigger cleanup
- **WHEN** cleanup endpoint is called
- **THEN** system deletes admin logs older than 2 years and msg logs older than 6 months
