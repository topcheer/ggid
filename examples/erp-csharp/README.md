# ERP C# Demo

Cross-board ERP demo using GGID C# SDK.

## Modules
- Auth: POST /api/auth/login
- Inventory: GET/POST /api/inventory (inventory:read/write)
- Orders: GET/POST /api/orders (orders:read/write), POST /api/orders/{id}/approve (orders:approve)
- Users: GET /api/users (users:read)
- Roles: GET /api/roles (roles:read)
- Audit: GET /api/audit (audit:read)
- My Permissions: GET /api/my-permissions

## Run
```bash
GGID_URL=https://ggid.iot2.win dotnet run
```
