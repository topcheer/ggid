//! Framework integration: Axum middleware and extractors for GGID auth.
//!
//! ## Usage with Axum
//! ```no_run
//! use axum::{routing::get, Router};
//! use ggid::middleware::{GGIDLayer, RequirePermission};
//!
//! let app = Router::new()
//!     .route("/api/products", get(|| async { "products" }))
//!     .layer(GGIDLayer::new("https://ggid.iot2.win", "tenant-uuid"));
//! ```

use std::sync::Arc;

use axum::{
    extract::{Request, State},
    http::{header, StatusCode},
    middleware::Next,
    response::{IntoResponse, Response},
};
use crate::client::GGIDClient;
use crate::error::GGIDError;

/// Shared state holding the GGID client.
#[derive(Clone)]
pub struct GGIDState {
    pub client: Arc<GGIDClient>,
}

impl GGIDState {
    pub fn new(base_url: &str, tenant_id: &str) -> Self {
        Self {
            client: Arc::new(GGIDClient::new(base_url, tenant_id)),
        }
    }
}

/// Extract the authenticated user's token from the request.
pub fn extract_token(req: &Request) -> Result<String, Response> {
    let auth_header = req
        .headers()
        .get(header::AUTHORIZATION)
        .and_then(|v| v.to_str().ok())
        .ok_or_else(|| {
            (
                StatusCode::UNAUTHORIZED,
                "missing Authorization header",
            )
                .into_response()
        })?;

    if !auth_header.starts_with("Bearer ") {
        return Err((
            StatusCode::UNAUTHORIZED,
            "invalid Authorization format",
        )
            .into_response());
    }

    Ok(auth_header[7..].to_string())
}

/// Auth middleware — verifies JWT and attaches claims to request extensions.
pub async fn auth_middleware(
    State(state): State<GGIDState>,
    mut req: Request,
    next: Next,
) -> Response {
    let token = match extract_token(&req) {
        Ok(t) => t,
        Err(resp) => return resp,
    };

    match state.client.verify_token(&token).await {
        Ok(claims) => {
            req.extensions_mut().insert(claims);
            req.extensions_mut().insert(token);
            next.run(req).await
        }
        Err(e) => (
            StatusCode::UNAUTHORIZED,
            format!("invalid token: {}", e),
        )
            .into_response(),
    }
}

/// Permission guard — checks RBAC permission before allowing the request.
///
/// Use with `from_fn_with_state`:
/// ```ignore
/// use axum::middleware::from_fn_with_state;
/// use ggid::middleware::{GGIDState, require_permission};
///
/// Router::new()
///     .route("/api/products", get(handler))
///     .route_layer(from_fn_with_state(state.clone(), require_permission("products", "read")))
/// ```
pub fn require_permission(resource: &'static str, action: &'static str) -> impl Fn(State<GGIDState>, Request, Next) -> std::pin::Pin<Box<dyn std::future::Future<Output = Response> + Send>> + Clone + Send + Sync + 'static {
    move |state: State<GGIDState>, req: Request, next: Next| {
        let resource = resource;
        let action = action;
        Box::pin(async move {
            let token = match extract_token(&req) {
                Ok(t) => t,
                Err(resp) => return resp,
            };

            match state.client.check_permission(&token, resource, action).await {
                Ok(true) => next.run(req).await,
                Ok(false) => (
                    StatusCode::FORBIDDEN,
                    format!("permission denied: {}:{}", resource, action),
                )
                    .into_response(),
                Err(e) => (
                    StatusCode::INTERNAL_SERVER_ERROR,
                    format!("authorization error: {}", e),
                )
                    .into_response(),
            }
        })
    }
}

/// Role guard — checks if the user has one of the required roles.
pub fn require_role(roles: Vec<String>) -> impl Fn(State<GGIDState>, Request, Next) -> std::pin::Pin<Box<dyn std::future::Future<Output = Response> + Send>> + Clone + Send + Sync + 'static {
    move |state: State<GGIDState>, req: Request, next: Next| {
        let roles = roles.clone();
        Box::pin(async move {
            let token = match extract_token(&req) {
                Ok(t) => t,
                Err(resp) => return resp,
            };

            match state.client.verify_token(&token).await {
                Ok(claims) => {
                    let has_role = claims.roles.iter().any(|r| roles.contains(r) || r == "admin");
                    if has_role {
                        next.run(req).await
                    } else {
                        (
                            StatusCode::FORBIDDEN,
                            format!("insufficient role: required {:?}", roles),
                        )
                            .into_response()
                    }
                }
                Err(e) => (
                    StatusCode::UNAUTHORIZED,
                    format!("invalid token: {}", e),
                )
                    .into_response(),
            }
        })
    }
}
