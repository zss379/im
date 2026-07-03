## ADDED Requirements

### Requirement: Mute member
The system SHALL allow group owners and admins to mute individual members for a specified duration.

#### Scenario: Owner mutes a member
- **WHEN** the owner sends POST /api/v1/groups/:group_id/mute/members with user_id and duration
- **THEN** system mutes the member for the specified duration

#### Scenario: Muted member cannot send messages
- **WHEN** message-svc calls GET /api/v1/groups/:group_id/mute/check for a muted member
- **THEN** system returns muted=true with remaining duration

#### Scenario: Non-muted member can send
- **WHEN** message-svc checks a non-muted member
- **THEN** system returns muted=false

#### Scenario: Admin cannot mute owner
- **WHEN** an admin attempts to mute the owner
- **THEN** system returns forbidden error

### Requirement: Unmute member
The system SHALL allow owners and admins to unmute a member before the duration expires.

#### Scenario: Owner unmutes a member
- **WHEN** the owner sends DELETE /api/v1/groups/:group_id/mute/members/:user_id
- **THEN** system removes the mute

### Requirement: Global group mute
The system SHALL allow owners and admins to enable global group mute.

#### Scenario: Enable global mute
- **WHEN** the owner sends POST /api/v1/groups/:group_id/mute/global with duration
- **THEN** system mutes all non-admin members

#### Scenario: Admin bypasses global mute
- **WHEN** message-svc checks an admin during global mute
- **THEN** system returns muted=false for the admin

#### Scenario: Disable global mute
- **WHEN** the owner sends DELETE /api/v1/groups/:group_id/mute/global
- **THEN** system removes the global mute
