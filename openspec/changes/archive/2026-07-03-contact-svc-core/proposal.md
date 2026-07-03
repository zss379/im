## Why

The contact/address book module is a core IM feature providing organization hierarchy, member search, and profile cards. A dedicated contact-svc is needed to manage department tree, member profiles, and HR sync — replacing client-side static data.

## What Changes

- New microservice: `contact-svc` (port 8084)
- Department hierarchy CRUD with recursive tree building
- Member profile management with name/pinyin search
- User-department multi-membership support
- HR batch sync endpoint for org data integration
- Phone number masking for privacy

## Capabilities

### New Capabilities
- `org-directory`: Department tree, member profiles, multi-dimensional search, HR sync

### Modified Capabilities

<!-- No existing specs modified -->

## Impact

- New service `contact-svc` under `services/contact-svc/`
- New DB tables: `contact_department`, `contact_profile`, `contact_user_dept`
- Redis DB 4 (reserved, minimal usage)
- No breaking changes to existing services
