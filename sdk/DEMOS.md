# OAuth + SAML Demos

## Go

### OAuth Demo
```bash
cd sdk/go/examples/oauth-demo
go run main.go
# Visit http://localhost:3001/auth/login
```

### SAML Demo
```bash
cd sdk/go/examples/saml-demo
# Already exists in examples/saml-demo/
```

## Python

### OAuth Demo
```bash
cd sdk/python/examples/oauth-demo
pip install -r requirements.txt
python app.py
# Visit http://localhost:3003/auth/login
```

### SAML Demo
```bash
cd sdk/python/examples/saml-demo
python app.py
```

## Rust

### OAuth Demo
```bash
cd sdk/rust/examples/oauth-demo
cargo run
# Visit http://localhost:3004/auth/login
```

## Java

### OAuth Demo
```bash
cd sdk/java/examples/oauth-demo
mvn compile exec:java
# Visit http://localhost:3002/auth/login
```

## Configuration
All demos read from environment variables:
- `GGID_URL` — GGID instance URL (default: http://localhost:8080)
- `CLIENT_ID` — OAuth client ID
- `CLIENT_SECRET` — OAuth client secret
- `SP_ENTITY_ID` — SAML SP entity ID
- `ACS_URL` — SAML ACS URL
- `REDIRECT_URI` — OAuth redirect URI

No hardcoded domains.
