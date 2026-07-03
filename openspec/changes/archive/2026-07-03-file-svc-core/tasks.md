## 1. Project Setup

- [x] 1.1 Create project directory structure with minio-go dependency
- [x] 1.2 Create Makefile, config.yaml (with MinIO config), config loading

## 2. Data Layer

- [x] 2.1 Define FileResource model with all request/response types
- [x] 2.2 Implement MySQLRepo with file CRUD
- [x] 2.3 Implement Storage (MinIO/S3) with upload, multipart, presigned URL

## 3. Business Logic

- [x] 3.1 Implement single file upload with size check and type classification
- [x] 3.2 Implement multipart upload (init, part, complete)
- [x] 3.3 Implement download with presigned URL
- [x] 3.4 Implement soft delete with S3 cleanup
- [x] 3.5 Implement file listing

## 4. HTTP Layer

- [x] 4.1 Create unified response helpers
- [x] 4.2 Register all 8 file endpoints
- [x] 4.3 Reuse JWT auth, logger, and recovery middleware

## 5. Service Entrypoint

- [x] 5.1 Create cmd/main.go with MinIO client init and bucket setup
- [x] 5.2 Add Prometheus metrics
- [x] 5.3 Implement graceful shutdown

## 6. Verification

- [x] 6.1 Verify code consistency
