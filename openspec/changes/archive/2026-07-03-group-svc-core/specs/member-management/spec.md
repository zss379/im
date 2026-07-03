## ADDED Requirements

### Requirement: List group members
The system SHALL provide paginated member listing with role filtering.

#### Scenario: Paginated member list
- **WHEN** a member sends GET /api/v1/groups/:group_id/members with page and page_size
- **THEN** system returns paginated member list with role, online status

#### Scenario: Filter by role
- **WHEN** a member sends GET /api/v1/groups/:group_id/members with role filter
- **THEN** system returns members matching the role

### Requirement: Batch add members
The system SHALL allow group owners and admins to batch add members.

#### Scenario: Owner adds members
- **WHEN** the owner sends POST /api/v1/groups/:group_id/members with user IDs
- **THEN** system adds all specified users as members

#### Scenario: Group member capacity enforced
- **WHEN** adding members would exceed the 2000 member limit
- **THEN** system rejects the addition

#### Scenario: Per-batch limit
- **WHEN** more than 10 members are specified in a single batch
- **THEN** system rejects with error

#### Scenario: Member already in group
- **WHEN** a user ID is already a member
- **THEN** system skips the duplicate

### Requirement: Batch remove members
The system SHALL allow group owners and admins to batch remove members.

#### Scenario: Owner removes members
- **WHEN** the owner sends DELETE /api/v1/groups/:group_id/members with user IDs
- **THEN** system removes specified members

#### Scenario: Admin cannot remove owner
- **WHEN** an admin attempts to remove the group owner
- **THEN** system returns forbidden error

#### Scenario: Admin cannot remove other admins
- **WHEN** an admin attempts to remove another admin
- **THEN** system returns forbidden error

### Requirement: Set group admins
The system SHALL allow the owner to promote members to admin or demote admins.

#### Scenario: Owner promotes member to admin
- **WHEN** the owner sends PUT /api/v1/groups/:group_id/members/:user_id/role with role=admin
- **THEN** system promotes the member, admin count may not exceed 10% of total members

#### Scenario: Owner demotes admin to member
- **WHEN** the owner sends PUT with role=member
- **THEN** system demotes the admin

#### Scenario: Non-owner cannot manage roles
- **WHEN** an admin or member attempts to change roles
- **THEN** system returns forbidden error

### Requirement: Search group members
The system SHALL support searching members within a group.

#### Scenario: Search by name
- **WHEN** a member sends GET /api/v1/groups/:group_id/members/search with keyword
- **THEN** system returns matching members by name
