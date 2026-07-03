## ADDED Requirements

### Requirement: User can create a session
The system SHALL create a new session for a user when they initiate a conversation.

#### Scenario: Create single chat session
- **WHEN** user creates a session with type=single and target_id=another_user
- **THEN** system creates a session with deterministic conversation_id `s_{minUID}_{maxUID}`

#### Scenario: Create group chat session
- **WHEN** user creates a session with type=group and target_id=group_id
- **THEN** system creates a session with conversation_id `g_{groupID}`

#### Scenario: Duplicate session returns existing
- **WHEN** user creates a session with same conversation_id as an existing non-deleted session
- **THEN** system returns the existing session without creating a duplicate

#### Scenario: Reactivate deleted session
- **WHEN** user creates a session for a previously deleted conversation
- **THEN** system reactivates the session (sets is_deleted=false)

#### Scenario: Session limit exceeded
- **WHEN** user already has 2000 active sessions
- **THEN** system rejects creation with an error

### Requirement: User can list sessions
The system SHALL return a paginated list of user's active sessions ordered by pinned status and last message time.

#### Scenario: List all active sessions
- **WHEN** user requests session list
- **THEN** system returns sessions with is_deleted=false, ordered by is_pinned DESC, pinned_at DESC, last_message_at DESC, created_at DESC

#### Scenario: Filter by session type
- **WHEN** user requests session list with type=1
- **THEN** system returns only single-chat sessions

#### Scenario: Filter unread only
- **WHEN** user requests session list with unread=true
- **THEN** system returns only sessions with unread_count > 0

#### Scenario: Pagination
- **WHEN** user requests session list with page=2, page_size=20
- **THEN** system returns the second page of 20 sessions

### Requirement: User can get session detail
The system SHALL return full session details for a given session ID.

#### Scenario: Get existing session
- **WHEN** user requests a session they own
- **THEN** system returns session details including type, target_id, pin/mute/unread status

#### Scenario: Get non-existent or deleted session
- **WHEN** user requests a session that doesn't exist, is deleted, or belongs to another user
- **THEN** system returns 404

### Requirement: User can delete a session
The system SHALL soft-delete a session (local deletion, cloud messages retained).

#### Scenario: Soft delete
- **WHEN** user deletes a session
- **THEN** system sets is_deleted=true, the session is hidden from list but restorable

#### Scenario: Delete non-existent session
- **WHEN** user deletes a session they don't own
- **THEN** system returns an error

### Requirement: User can pin sessions
The system SHALL allow users to pin sessions (up to 5).

#### Scenario: Pin a session
- **WHEN** user pins a session with pinned=true
- **THEN** system sets is_pinned=true and pinned_at=now

#### Scenario: Unpin a session
- **WHEN** user unpins a session with pinned=false
- **THEN** system sets is_pinned=false and pinned_at=null

#### Scenario: Pin limit exceeded
- **WHEN** user already has 5 pinned sessions and tries to pin another
- **THEN** system returns a max pinned error

### Requirement: User can mute sessions
The system SHALL allow users to mute/unmute session notifications.

#### Scenario: Mute a session
- **WHEN** user mutes a session with muted=true
- **THEN** system sets is_muted=true

#### Scenario: Unmute a session
- **WHEN** user unmutes a session with muted=false
- **THEN** system sets is_muted=false

### Requirement: User can mark sessions as read/unread
The system SHALL support marking individual sessions as read or unread.

#### Scenario: Mark as read
- **WHEN** user marks a session as read
- **THEN** system sets unread_count=0 and updates Redis cache

#### Scenario: Mark as unread
- **WHEN** user marks a session as unread
- **THEN** system sets unread_count=1 and updates Redis cache

#### Scenario: Batch update unread counts
- **WHEN** user sends batch unread update with multiple session IDs
- **THEN** system updates all specified sessions' unread_count and Redis cache
