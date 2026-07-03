## ADDED Requirements

### Requirement: User SHALL upload files
The system SHALL accept file uploads via multipart form and store in S3-compatible storage.

#### Scenario: Upload image
- **WHEN** user uploads an image file under 10MB
- **THEN** system stores the file in S3 and returns file metadata with presigned URL

#### Scenario: Upload exceeds size limit
- **WHEN** user uploads an image over 10MB or file over 50MB
- **THEN** system rejects with an error

#### Scenario: Upload video
- **WHEN** user uploads a video file under 100MB
- **THEN** system stores the file and classifies as video type

### Requirement: System SHALL support multipart upload for large files
The system SHALL support chunked upload for large files via S3 multipart upload API.

#### Scenario: Initiate multipart upload
- **WHEN** client initiates multipart upload with filename and size
- **THEN** system returns upload ID and object key

#### Scenario: Upload part
- **WHEN** client uploads a part with part number and upload ID
- **THEN** system stores the part and returns ETag

#### Scenario: Complete multipart upload
- **WHEN** client completes with all part ETags
- **THEN** system finalizes the file and returns file metadata with URL

### Requirement: User SHALL download files
The system SHALL provide file download via presigned URL.

#### Scenario: Get download URL
- **WHEN** user requests file download
- **THEN** system returns a 302 redirect to a presigned S3 URL (1hr expiry)

#### Scenario: Get file metadata
- **WHEN** user requests file info
- **THEN** system returns file name, size, type, timestamps

#### Scenario: File not found
- **WHEN** user requests a non-existent or deleted file
- **THEN** system returns 404

### Requirement: User SHALL delete files
The system SHALL soft-delete file metadata and remove the S3 object.

#### Scenario: Delete file
- **WHEN** user deletes a file
- **THEN** system sets status=0 in DB and removes the object from S3

### Requirement: System SHALL list files
The system SHALL support paginated file listing with optional type filter.

#### Scenario: List files by type
- **WHEN** user lists files with file_type=image
- **THEN** system returns only image files, ordered by created_at DESC
