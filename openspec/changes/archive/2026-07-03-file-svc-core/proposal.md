## Why

File service (IM-MOD-009) provides file upload, download, and S3-compatible storage for all IM file operations (images, videos, documents, voice messages). All other services depend on file-svc for media handling.

## What Changes

- New microservice: `file-svc` (port 8087)
- Single-file upload via multipart form
- Multipart (chunked) upload for large files (init, part, complete)
- File metadata management
- Presigned download URL (1hr expiry)
- Soft-delete with S3 object cleanup
- MinIO/S3-compatible storage backend
- File type classification (image, video, audio, file) with size limits per PRD §4.9.5

## Capabilities

### New Capabilities
- `file-storage`: File upload/download, multipart upload, S3 storage, metadata management

### Modified Capabilities

<!-- No existing specs modified -->

## Impact

- New service `file-svc` under `services/file-svc/`
- New DB table: `file_resource`
- MinIO bucket: `im-files` (auto-created on startup)
- New dependency: `github.com/minio/minio-go/v7`
- File size limits: image 10MB, video 100MB, file 50MB
