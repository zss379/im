## ADDED Requirements

### Requirement: File upload limit management
The system SHALL allow administrators to manage file upload limits through CRUD operations.

#### Scenario: List file limit configs
- **WHEN** admin sends GET /api/v1/file-limit/configs
- **THEN** system returns all configured file limits

#### Scenario: Create a file limit config
- **WHEN** admin sends POST /api/v1/file-limit/configs with file_type, max_size_mb, and optional allowed_extensions
- **THEN** system creates the config

#### Scenario: Update a file limit config
- **WHEN** admin sends PUT /api/v1/file-limit/configs/:config_id with fields to update
- **THEN** system updates the config

#### Scenario: Delete a file limit config
- **WHEN** admin sends DELETE /api/v1/file-limit/configs/:config_id
- **THEN** system deletes the config

### Requirement: File upload size validation
The system SHALL validate file upload sizes against configured limits per file type.

#### Scenario: File within size limit returns passed
- **WHEN** a file size is within the configured max_size_mb for its type
- **THEN** system returns passed=true with the max size info

#### Scenario: File exceeds size limit returns blocked
- **WHEN** a file size exceeds the configured max_size_mb for its type
- **THEN** system returns passed=false with the max size info

#### Scenario: No limit configured returns passed
- **WHEN** no file limit exists for the file type
- **THEN** system returns passed=true

### Requirement: Default file limits
The system SHALL provide default file upload limits for common file types.

#### Scenario: Default limits exist on fresh start
- **WHEN** the service starts with an empty database
- **THEN** system seeds defaults: image=10MB, video=100MB, document=50MB
