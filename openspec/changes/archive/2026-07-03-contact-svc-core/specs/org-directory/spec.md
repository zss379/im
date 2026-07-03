## ADDED Requirements

### Requirement: System SHALL expose org hierarchy as a tree
The system SHALL return the full department tree for a tenant.

#### Scenario: Get department tree
- **WHEN** client requests the department tree
- **THEN** system returns nested tree structure rooted at parent_id=0, ordered by sort_order and dept_id

#### Scenario: Empty org
- **WHEN** tenant has no departments
- **THEN** system returns an empty array

### Requirement: System SHALL support department CRUD
The system SHALL allow creating, reading, updating, and soft-deleting departments.

#### Scenario: Create department
- **WHEN** admin creates a department with name and optional parent_id
- **THEN** system persists the department and returns its detail

#### Scenario: Create with invalid parent
- **WHEN** admin creates a department with a non-existent parent_id
- **THEN** system returns an error

#### Scenario: Update department
- **WHEN** admin updates department name or sort_order
- **THEN** system applies the changes

#### Scenario: Delete department with children
- **WHEN** admin deletes a department that has active children
- **THEN** system rejects with an error

#### Scenario: Delete department with no children
- **WHEN** admin deletes a leaf department
- **THEN** system soft-deletes it (sets status=0)

#### Scenario: Get department detail
- **WHEN** client requests department detail
- **THEN** system returns name, member_count, parent_id, timestamps

### Requirement: System SHALL support member profile management
The system SHALL manage user contact profiles including name, avatar, phone, position, and department memberships.

#### Scenario: Get member detail
- **WHEN** client requests member profile
- **THEN** system returns name, avatar, masked phone, position, and department list

#### Scenario: Get non-existent member
- **WHEN** client requests a member from a different tenant
- **THEN** system returns 404

#### Scenario: Update member profile
- **WHEN** admin updates member name/avatar/phone/position/dept_ids
- **THEN** system upserts the profile and updates department associations

#### Scenario: Phone masking
- **WHEN** system returns member phone numbers
- **THEN** phone is masked as 138****1234 (keep first 3 and last 4 digits)

### Requirement: System SHALL support member search
The system SHALL search members by keyword (name or pinyin) across the org, optionally filtered by department.

#### Scenario: Search by name
- **WHEN** client searches with a name keyword
- **THEN** system returns matching members with name, avatar, position, primary dept name

#### Scenario: Search by pinyin initials
- **WHEN** client searches with pinyin initials
- **THEN** system returns members matching name_py field

#### Scenario: Filter by department
- **WHEN** client searches with dept_id filter
- **THEN** system returns only members in that department

#### Scenario: Empty search
- **WHEN** keyword has no matches
- **THEN** system returns empty list with total=0

#### Scenario: Paginated results
- **WHEN** client specifies page and page_size
- **THEN** system returns paginated results (default 50, max 100)

### Requirement: System SHALL list department members
The system SHALL return paginated member list for a given department.

#### Scenario: List department members
- **WHEN** client requests members of a department
- **THEN** system returns members with profile info, ordered by user_id

#### Scenario: Get member's departments
- **WHEN** client requests the departments a member belongs to
- **THEN** system returns department list with primary dept first

### Requirement: System SHALL support HR batch sync
The system SHALL accept batch org data from the HR system to upsert departments, members, and their associations.

#### Scenario: Full sync
- **WHEN** HR system sends departments and members arrays
- **THEN** system upserts all departments, upserts all member profiles, and updates department associations

#### Scenario: Partial sync
- **WHEN** HR system sends only member updates
- **THEN** system updates member profiles without affecting departments
