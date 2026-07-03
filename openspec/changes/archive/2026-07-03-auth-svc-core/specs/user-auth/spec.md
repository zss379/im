## ADDED Requirements

### Requirement: User SHALL authenticate with account and password
The system SHALL verify user credentials and issue a JWT token.

#### Scenario: Successful login
- **WHEN** user provides correct account and password
- **THEN** system returns a JWT token and user brief (user_id, account, name, avatar, status)

#### Scenario: Invalid credentials
- **WHEN** user provides wrong password
- **THEN** system returns 401 and increments login attempt counter

#### Scenario: Account locked after 5 failed attempts
- **WHEN** user exceeds 5 failed login attempts within 15 minutes
- **THEN** system locks the account and returns lockout error

#### Scenario: Disabled account login
- **WHEN** user account is disabled
- **THEN** system returns 401 with "account disabled" error

#### Scenario: Token refresh
- **WHEN** user sends a valid token to refresh endpoint
- **THEN** system returns a new token with extended expiry

### Requirement: Admin SHALL manage user accounts
The system SHALL support creating, reading, and updating user accounts.

#### Scenario: Create user
- **WHEN** admin creates a user with account, password, name
- **THEN** system stores the user with bcrypt-hashed password

#### Scenario: Duplicate account
- **WHEN** admin creates a user with an existing account name
- **THEN** system returns an error

#### Scenario: Get user detail
- **WHEN** client requests user detail
- **THEN** system returns user_id, account, name, avatar, phone, email, status, timestamps

#### Scenario: Update user profile
- **WHEN** user updates name/avatar/phone/email
- **THEN** system applies the changes

#### Scenario: Change password
- **WHEN** user provides correct old password and a new password
- **THEN** system updates the password hash

### Requirement: System SHALL track online status
The system SHALL track and expose user online/offline/busy/DND status.

#### Scenario: Set status
- **WHEN** user sets their status to online/busy/DND
- **THEN** system persists status in MySQL and caches in Redis

#### Scenario: Get status (cached)
- **WHEN** client requests a user's status
- **THEN** system returns from Redis cache if available, else MySQL

#### Scenario: Batch get status
- **WHEN** client requests status for multiple user IDs
- **THEN** system returns status for all requested users (default offline)
