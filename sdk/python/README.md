# GGID Python SDK

[![PyPI version](https://badge.fury.io/py/ggid.svg)](https://badge.fury.io/py/ggid)
[![License: Apache 2.0](https://img.shields.io/badge/License-Apache_2.0-blue.svg)](https://opensource.org/licenses/Apache-2.0)

Python SDK for GGID IAM Platform — JWT verification, user management, and RBAC permission checking.

## Features

- **JWT Verification**: Verify RS256 JWTs from GGID Gateway with JWKS caching
- **FastAPI Middleware**: Drop-in authentication for FastAPI apps
- **Django Middleware**: Authentication decorator for Django views
- **Flask Decorator**: `@requires_auth` decorator for Flask routes
- **Permission Checking**: Check user roles and permissions via GGID Policy API
- **User Management**: CRUD operations via GGID Identity API
- **Async Support**: Full async/await support with httpx

## Installation

```bash
pip install ggid
```

## Quick Start

### FastAPI

```python
from fastapi import FastAPI, Depends
from ggid import GGIDMiddleware, get_current_user

app = FastAPI()
app.add_middleware(
    GGIDMiddleware,
    gateway_url="https://iam.example.com",
    jwks_url="https://iam.example.com/.well-known/jwks.json",
    tenant_id="00000000-0000-0000-0000-000000000001",
)

@app.get("/profile")
async def profile(user = Depends(get_current_user)):
    return {"user": user}
```

### Flask

```python
from flask import Flask, jsonify
from ggid.flask import requires_auth

app = Flask(__name__)
app.config["GGID_GATEWAY_URL"] = "https://iam.example.com"
app.config["GGID_TENANT_ID"] = "00000000-0000-0000-0000-000000000001"

@app.route("/profile")
@requires_auth
def profile(user):
    return jsonify({"user": user})
```

### Django

```python
# settings.py
GGID_GATEWAY_URL = "https://iam.example.com"
GGID_TENANT_ID = "00000000-0000-0000-0000-000000000001"

# views.py
from ggid.django import ggid_login_required

@ggid_login_required
def profile(request):
    return JsonResponse({"user": request.ggid_user})
```

### Permission Check

```python
from ggid import GGIDClient

client = GGIDClient(
    gateway_url="https://iam.example.com",
    tenant_id="00000000-0000-0000-0000-000000000001",
)

# Check if user can perform action on resource
allowed = await client.check_permission(
    user_id="abc-123",
    resource="documents:sensitive",
    action="read",
)
```

## License

Apache 2.0
