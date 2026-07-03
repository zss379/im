## ADDED Requirements

### Requirement: Join verification toggle
The system SHALL allow group owners and admins to configure the join verification mode.

#### Scenario: Set to no verification
- **WHEN** the owner sends PUT /api/v1/groups/:group_id/join-config with verification_mode=open
- **THEN** any user can join without approval

#### Scenario: Set to admin approval
- **WHEN** the owner sends PUT with verification_mode=approval_required
- **THEN** join requests require admin approval

### Requirement: Request to join
The system SHALL allow users to request to join a group requiring approval.

#### Scenario: User requests to join
- **WHEN** a user sends POST /api/v1/groups/:group_id/join-requests
- **THEN** system creates a pending join request

#### Scenario: Duplicate request
- **WHEN** user already has a pending request
- **THEN** system rejects duplicate

#### Scenario: Already a member
- **WHEN** user is already a member
- **THEN** system rejects with existing membership error

### Requirement: Approve join request
The system SHALL allow group owners and admins to approve or reject join requests.

#### Scenario: Admin approves request
- **WHEN** an admin sends POST /api/v1/groups/:group_id/join-requests/:request_id/approve
- **THEN** system adds the user as a member and marks request as approved

#### Scenario: Admin rejects request
- **WHEN** an admin sends POST /api/v1/groups/:group_id/join-requests/:request_id/reject
- **THEN** system marks request as rejected

#### Scenario: Member capacity check
- **WHEN** approving would exceed 2000 member limit
- **THEN** system rejects the approval
