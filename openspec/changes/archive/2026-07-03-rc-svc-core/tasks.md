## 1. Project Setup

- [x] 1.1 Create go.mod with dependencies
- [x] 1.2 Create config.yaml and config loader
- [x] 1.3 Create Makefile

## 2. Models

- [x] 2.1 Define GORM models (SensitiveWord, FrequencyControlRule, FileLimitConfig)
- [x] 2.2 Define request/response types

## 3. DFA Engine

- [x] 3.1 Implement DFA trie with Build/Check/Replace
- [x] 3.2 Support case-insensitive matching and longest-match semantics
- [x] 3.3 Support three strategies: replace, block, log
- [x] 3.4 Add goroutine-safe RWMutex for concurrent read/hot-reload
- [x] 3.5 Write DFA engine unit tests

## 4. Repository Layer

- [x] 4.1 Implement MySQL repo with CRUD for all three tables
- [x] 4.2 Implement AutoMigrate and SeedDefaults
- [x] 4.3 Implement Redis sliding window rate limiter
- [x] 4.4 Write cache unit tests with miniredis

## 5. Service Layer

- [x] 5.1 Implement RCService with CheckSensitive, CheckRateLimit, CheckFileLimit
- [x] 5.2 Implement CheckChain combined pipeline
- [x] 5.3 Implement CRUD methods for all three domains
- [x] 5.4 Implement async DFA refresh on word CRUD
- [x] 5.5 Write service tests

## 6. Handler Layer

- [x] 6.1 Create response helpers
- [x] 6.2 Implement sensitive word handler (7 endpoints)
- [x] 6.3 Implement rate limit handler (5 endpoints)
- [x] 6.4 Implement file limit handler (5 endpoints)
- [x] 6.5 Write handler tests for all endpoints

## 7. Middleware

- [x] 7.1 Implement JWT auth middleware
- [x] 7.2 Implement logger and recovery middleware

## 8. Metrics

- [x] 8.1 Create Prometheus metrics for sensitive/rate/file checks

## 9. Main Entry Point

- [x] 9.1 Create cmd/main.go with full service wiring
