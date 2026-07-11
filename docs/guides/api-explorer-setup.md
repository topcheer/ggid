# API Explorer Setup

> Deploy Swagger UI to browse and test the GGID OpenAPI specification.

---

## Option A: Built-in Swagger UI (Recommended)

GGID Gateway includes a built-in Swagger UI at `/docs`:

```bash
curl http://localhost:8080/docs
# Returns interactive Swagger UI
```

The spec is served at:
```
http://localhost:8080/docs/openapi.yaml
```

---

## Option B: Standalone Swagger UI (Docker)

### docker-compose.yml

```yaml
services:
  swagger-ui:
    image: swaggerapi/swagger-ui:latest
    ports:
      - "8081:8080"
    environment:
      SWAGGER_JSON: /spec/openapi.yaml
    volumes:
      - ./docs/openapi.yaml:/spec/openapi.yaml:ro
```

```bash
docker compose up -d swagger-ui
# Open http://localhost:8081
```

---

## Option C: Nginx Static Hosting

```nginx
server {
    listen 80;
    server_name api-docs.example.com;

    location / {
        root /usr/share/nginx/html/swagger-ui-dist;
        index index.html;
    }

    location /openapi.yaml {
        alias /data/openapi.yaml;
        add_header Content-Type application/yaml;
    }
}
```

### index.html

```html
<!DOCTYPE html>
<html>
<head>
  <title>GGID API Explorer</title>
  <link rel="stylesheet" href="https://unpkg.com/swagger-ui-dist@5/swagger-ui.css">
</head>
<body>
  <div id="swagger-ui"></div>
  <script src="https://unpkg.com/swagger-ui-dist@5/swagger-ui-bundle.js"></script>
  <script>
    window.onload = () => {
      SwaggerUIBundle({
        url: '/openapi.yaml',
        dom_id: '#swagger-ui',
        deepLinking: true,
        tryItOutEnabled: true,
      });
    };
  </script>
</body>
</html>
```

---

## Testing Endpoints

In Swagger UI:
1. Click **Authorize** button
2. Enter JWT: `Bearer eyJ...`
3. Enter `X-Tenant-ID`: `00000000-0000-0000-0000-000000000001`
4. Click any endpoint → **Try it out** → **Execute**

---

*See: [REST API Reference](../api/rest-api.md) | [OpenAPI Spec](../openapi.yaml)*

*Last updated: 2025-07-11*
