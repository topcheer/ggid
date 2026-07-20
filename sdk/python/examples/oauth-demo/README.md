# OAuth Demo — Python SDK

OAuth 2.0 Authorization Code flow with GGID IAM.

## Setup

```bash
export GGID_URL=https://ggid.iot2.win
export CLIENT_ID=gcid_xxx
export CLIENT_SECRET=gcs_xxx
export REDIRECT_URI=http://localhost:9098/callback
export TENANT_ID=00000000-0000-0000-0000-000000000001
```

## Run

```bash
pip install flask requests
python oauth-demo.py
```

Open http://localhost:9098
