## Context

File service (IM-MOD-009) is the last microservice needed to complete the IM backend. It provides S3-compatible object storage for all file types. Files are stored in MinIO, metadata in MySQL, and access via presigned URLs.

## Goals / Non-Goals

**Goals:**
- Simple and multipart uploads
- S3 presigned URL for download
- File type detection and size enforcement
- Soft delete with S3 cleanup

**Non-Goals:**
- File transcoding or thumbnail generation (future)
- CDN integration (future)

## Decisions

1. **MinIO SDK directly**: Using minio-go for S3 operations instead of AWS SDK. Lighter, and our storage is MinIO.
2. **Object key format**: `{tenantID}/{date}/{xid}.{ext}` — tenant-isolated, date-partitioned, unique per file.
3. **Presigned URLs**: 1-hour expiry. The client uses the URL to download directly from MinIO, reducing load on the service.
4. **Soft delete with S3 cleanup**: Metadata is soft-deleted for recoverability. S3 object is deleted immediately.

## Risks / Trade-offs

- [MinIO availability] If MinIO is down, upload/download fails. Consider multi-node MinIO deployment.
- [URL expiry] Presigned URLs expire after 1 hour. Long-lived sharing requires a different mechanism.
