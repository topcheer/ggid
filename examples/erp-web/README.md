# ERP Web — Next.js + Ant Design with GGID IAM

Replaces cross-erp/web with permission-aware frontend.

## Features
- OAuth login via GGID (authorization code flow)
- Role extraction from JWT scopes (Sales Manager / Warehouse Manager / Finance Officer / Administrator)
- Menu visibility based on permissions
- Button-level permission control (Create/Approve/Delete/Ship)
- 403 page for unauthorized access
- Inventory, Orders, Reports, Admin pages

## Config (env vars)
```
GGID_URL=https://ggid.iot2.win
CLIENT_ID=gcid_xxx
CLIENT_SECRET=gcs_xxx
REDIRECT_URI=https://erp.iot2.win/callback
TENANT_ID=00000000-0000-0000-0000-000000000001
```

## Permission Matrix
| Resource | Sales Manager | Warehouse Manager | Finance Officer |
|----------|--------------|-------------------|----------------|
| Dashboard | read | read | read |
| Orders | read, write, approve | read, write | read |
| Inventory | read | read, write, delete | — |
| Reports | read | read | read, write |

## Run
```bash
npm install && npm run dev
```

## Docker
```bash
docker build -t erp-web .
```
