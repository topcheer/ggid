# GGID SDK for Rust

Simplest, most flexible IAM integration for Rust applications.

## Quick Start

Add to `Cargo.toml`:
```toml
[dependencies]
ggid = "0.1"
```

### Verify a JWT
```rust
use ggid::GGIDClient;

# #[tokio::main]
# async fn main() -> Result<(), ggid::GGIDError> {
let ggid = GGIDClient::new("https://ggid.iot2.win", "tenant-uuid");
let claims = ggid.verify_token("eyJ...").await?;
println!("user_id={}, roles={:?}", claims.sub, claims.roles);
# Ok(())
# }
```

### Check Permission
```rust
let allowed = ggid.check_permission(token, "products", "read").await?;
if !allowed {
    return Err("forbidden".into());
}
```

### OAuth Authorization Code Flow
```rust
// 1. Redirect user to authorize URL
let url = ggid.get_authorize_url(
    "client-id",
    "https://app.com/callback",
    Some("openid profile email"),
    Some("random-state"),
);

// 2. Exchange code for tokens
let tokens = ggid.exchange_code(
    code,
    "https://app.com/callback",
    "client-id",
    "client-secret",
).await?;
```

### Axum Middleware
```rust
use axum::{routing::get, Router};
use ggid::middleware::{GGIDState, auth_middleware, require_permission};
use axum::middleware::from_fn_with_state;

let state = GGIDState::new("https://ggid.iot2.win", "tenant-uuid");

let app = Router::new()
    .route("/api/products", get(list_products))
    .route_layer(from_fn_with_state(
        state.clone(),
        require_permission("products", "read"),
    ))
    .layer(from_fn_with_state(
        state.clone(),
        auth_middleware,
    ))
    .with_state(state);
```

## API Reference

| Method | Description |
|--------|-------------|
| `verify_token(token)` | Verify JWT via JWKS, return claims |
| `get_user_info(token)` | Get user info from userinfo endpoint |
| `get_authorize_url(...)` | Build OAuth authorize URL |
| `exchange_code(...)` | Exchange auth code for tokens |
| `refresh_token(...)` | Refresh access token |
| `revoke_token(...)` | Revoke a token |
| `check_permission(token, resource, action)` | RBAC permission check |
| `assign_role(token, user_id, role_id)` | Assign role to user |
| `revoke_role(token, user_id, role_id)` | Revoke role from user |
| `get_user_roles(token, user_id)` | Get user's roles |
| `list_roles(token)` | List all tenant roles |
| `list_permissions(token)` | List permission tree |
| `evaluate_abac(token, request)` | Evaluate ABAC conditions |
| `check_policy(token, request)` | Full policy check |

## License

Apache-2.0
