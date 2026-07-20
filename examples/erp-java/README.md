# ERP Java Demo

Cross-board ERP demo using GGID Java SDK.

## Modules
- Auth: POST /api/auth/login
- Inventory: GET/POST (inventory:read/write)
- Orders: GET/POST + approve (orders:read/write/approve)
- Users: GET (users:read)
- Roles: GET (roles:read)
- Audit: GET (audit:read)
- My Permissions: GET

## Run
```bash
GGID_URL=https://ggid.iot2.win mvn exec:java
```
