## Purpose

群成员管理能力，包括成员列表、搜索、角色管理、禁言等。

## Requirements

### Requirement: Group member keyword search
group-svc SHALL support keyword-based search of group members by user_id.

#### Scenario: Search by numeric user_id
- **GIVEN** a group with members whose user_id includes 10001
- **WHEN** user calls GET /api/v1/groups/:group_id/members/search?keyword=10001
- **THEN** only members with user_id 10001 SHALL be returned

#### Scenario: Empty keyword returns all members
- **GIVEN** a group with multiple members
- **WHEN** user calls GET /api/v1/groups/:group_id/members/search?keyword=
- **THEN** all members SHALL be returned (same as ListMembers)

#### Scenario: Non-numeric keyword
- **GIVEN** a group with members
- **WHEN** user calls GET /api/v1/groups/:group_id/members/search?keyword=abc
- **THEN** empty list SHALL be returned
- **AND** total SHALL be 0
