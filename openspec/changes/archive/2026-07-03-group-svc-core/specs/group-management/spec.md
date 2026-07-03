## ADDED Requirements

### Requirement: Create group
The system SHALL allow any user to create a group, becoming the group owner.

#### Scenario: Successful group creation
- **WHEN** user sends POST /api/v1/groups with group name and optional member list
- **THEN** system creates the group and sets the creator as owner, returns group info

#### Scenario: Capacity limit enforced
- **WHEN** user has reached the maximum of 200 created groups
- **THEN** system rejects creation with an error

#### Scenario: Group name validation
- **WHEN** group name is empty or exceeds max length
- **THEN** system rejects with validation error

### Requirement: Dismiss group
The system SHALL allow only the group owner to dismiss a group.

#### Scenario: Owner dismisses group
- **WHEN** the group owner sends DELETE /api/v1/groups/:group_id
- **THEN** system deletes the group and all member associations

#### Scenario: Non-owner cannot dismiss
- **WHEN** an admin or member attempts to dismiss the group
- **THEN** system returns forbidden error

#### Scenario: Irreversible confirmation
- **WHEN** owner requests dismiss
- **THEN** system requires secondary confirmation parameter

### Requirement: Transfer group ownership
The system SHALL allow the owner to transfer ownership to another member.

#### Scenario: Successful transfer
- **WHEN** the owner sends PUT /api/v1/groups/:group_id/transfer with target member ID
- **THEN** system transfers ownership, demotes former owner to member

#### Scenario: Non-owner cannot transfer
- **WHEN** an admin or member attempts to transfer
- **THEN** system returns forbidden error

### Requirement: Exit group
The system SHALL allow members to voluntarily exit a group.

#### Scenario: Member exits group
- **WHEN** a member sends POST /api/v1/groups/:group_id/exit
- **THEN** system removes them from the group

#### Scenario: Owner must transfer before exit
- **WHEN** the owner attempts to exit
- **THEN** system rejects with error to transfer ownership first

### Requirement: Update group info
The system SHALL allow group owners and admins to update group information.

#### Scenario: Owner updates group name
- **WHEN** the owner sends PUT /api/v1/groups/:group_id with name
- **THEN** system updates the group name

#### Scenario: Admin updates group notice
- **WHEN** an admin sends PUT /api/v1/groups/:group_id with notice
- **THEN** system updates the group notice, limited to 5000 characters

#### Scenario: Member cannot update group info
- **WHEN** a member attempts to update group info
- **THEN** system returns forbidden error
