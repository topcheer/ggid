# GGID SDK Quickstart Examples

These examples demonstrate the **5-minute integration** promise:
minimal code to get JWT authentication working in your application.

## Examples

| Language | File | Framework | Lines | What it does |
|----------|------|-----------|-------|-------------|
| Go | `go-quickstart/main.go` | net/http | ~40 | Login + JWT middleware + protected route |
| Node | `node-quickstart/index.ts` | Express | ~35 | Login + Express middleware + protected route |
| Python | `python-quickstart/app.py` | Flask | ~35 | Login + middleware + protected route |
| Java | `java-quickstart/QuickstartApp.java` | Servlet | ~45 | Login + JWT filter + protected route |

## Prerequisites

1. GGID running: `cd deploy && docker compose up -d`
2. Default tenant: `00000000-0000-0000-0000-000000000001`
3. Default admin: `admin / Admin@123456`

## Run

### Go
```bash
cd go-quickstart && go run main.go
```

### Node
```bash
cd node-quickstart && npm install && npm start
```

### Python
```bash
cd python-quickstart && pip install flask requests && python app.py
```

## The 3-Line Promise

Every SDK should enable authentication in 3 lines:

**Go:**
```go
client := ggid.NewClient("http://localhost:8080")
tokens, _ := client.Login(ctx, "admin", "Admin@123456")
handler := ggidmw.Auth("http://localhost:8080", ggidmw.Options{})(mux)
```

**Node:**
```ts
const client = new GGIDClient({ gatewayUrl: 'http://localhost:8080' });
const tokens = await client.login({ username: 'admin', password: 'Admin@123456' });
app.use(expressAuth({ gatewayUrl: 'http://localhost:8080' }));
```

**Python:**
```python
client = GGIDClient("http://localhost:8080")
tokens = client.login("admin", "Admin@123456")
app.wsgi_app = GGIDMiddleware(app.wsgi_app, gateway_url="http://localhost:8080")
```
