use std::sync::Arc;
use std::time::Duration;

use reqwest::Client as HttpClient;
use serde_json::json;

use crate::auth::AuthService;
use crate::error::GGIDError;
use crate::rbac::RBACService;
use crate::abac::ABACService;
use crate::types::*;

/// GGID IAM client — the single entry point for all SDK operations.
///
/// Create with [`GGIDClient::new`] and use for JWT verification,
/// permission checks, OAuth flows, and policy evaluation.
///
/// # Example
/// ```no_run
/// use ggid::GGIDClient;
/// # #[tokio::main]
/// # async fn main() -> Result<(), ggid::GGIDError> {
/// let ggid = GGIDClient::new("https://ggid.iot2.win", "tenant-uuid");
/// let claims = ggid.verify_token("eyJ...").await?;
/// # Ok(())
/// # }
/// ```
#[derive(Clone)]
pub struct GGIDClient {
    pub base_url: String,
    pub tenant_id: String,
    pub(crate) http: HttpClient,
}

impl GGIDClient {
    /// Create a new GGID client with default settings.
    pub fn new(base_url: impl Into<String>, tenant_id: impl Into<String>) -> Self {
        let http = HttpClient::builder()
            .timeout(Duration::from_secs(10))
            .build()
            .expect("failed to build HTTP client");
        Self {
            base_url: base_url.into().trim_end_matches('/').to_string(),
            tenant_id: tenant_id.into(),
            http,
        }
    }

    /// Create a builder for custom configuration.
    pub fn builder() -> GGIDClientBuilder {
        GGIDClientBuilder::default()
    }

    /// Verify a JWT token and return claims.
    ///
    /// Fetches JWKS from GGID and verifies signature + expiry.
    pub async fn verify_token(&self, token: &str) -> Result<Claims, GGIDError> {
        AuthService::new(self.clone()).verify_token(token).await
    }

    /// Get user info from the userinfo endpoint.
    pub async fn get_user_info(&self, token: &str) -> Result<UserInfo, GGIDError> {
        AuthService::new(self.clone()).get_user_info(token).await
    }

    // --- OAuth/OIDC ---

    /// Build an authorize URL for the authorization code flow.
    pub fn get_authorize_url(
        &self,
        client_id: &str,
        redirect_uri: &str,
        scope: Option<&str>,
        state: Option<&str>,
    ) -> String {
        let scope_val = scope.unwrap_or("openid profile email");
        let mut query = url::form_urlencoded::Serializer::new(String::new());
        query.append_pair("response_type", "code");
        query.append_pair("client_id", client_id);
        query.append_pair("redirect_uri", redirect_uri);
        query.append_pair("tenant_id", &self.tenant_id);
        query.append_pair("scope", scope_val);
        if let Some(s) = state {
            if !s.is_empty() {
                query.append_pair("state", s);
            }
        }
        format!(
            "{}/api/v1/oauth/authorize?{}",
            self.base_url,
            query.finish()
        )
    }

    /// Exchange an authorization code for tokens.
    pub async fn exchange_code(
        &self,
        code: &str,
        redirect_uri: &str,
        client_id: &str,
        client_secret: &str,
    ) -> Result<TokenResponse, GGIDError> {
        let resp = self
            .http
            .post(format!("{}/api/v1/oauth/token", self.base_url))
            .header("X-Tenant-ID", &self.tenant_id)
            .json(&json!({
                "grant_type": "authorization_code",
                "code": code,
                "redirect_uri": redirect_uri,
                "client_id": client_id,
                "client_secret": client_secret,
            }))
            .send()
            .await?;

        if !resp.status().is_success() {
            let status = resp.status().as_u16();
            let body = resp.text().await.unwrap_or_default();
            return Err(GGIDError::Api { status, body });
        }

        Ok(resp.json().await?)
    }

    /// Refresh an access token.
    pub async fn refresh_token(
        &self,
        refresh_token: &str,
        client_id: &str,
        client_secret: &str,
    ) -> Result<TokenResponse, GGIDError> {
        let resp = self
            .http
            .post(format!("{}/api/v1/oauth/token", self.base_url))
            .header("X-Tenant-ID", &self.tenant_id)
            .json(&json!({
                "grant_type": "refresh_token",
                "refresh_token": refresh_token,
                "client_id": client_id,
                "client_secret": client_secret,
            }))
            .send()
            .await?;

        if !resp.status().is_success() {
            let status = resp.status().as_u16();
            let body = resp.text().await.unwrap_or_default();
            return Err(GGIDError::Api { status, body });
        }

        Ok(resp.json().await?)
    }

    /// Revoke a token.
    pub async fn revoke_token(
        &self,
        token: &str,
        client_id: &str,
        client_secret: &str,
    ) -> Result<(), GGIDError> {
        let _ = self
            .http
            .post(format!("{}/api/v1/oauth/revoke", self.base_url))
            .header("X-Tenant-ID", &self.tenant_id)
            .json(&json!({
                "token": token,
                "client_id": client_id,
                "client_secret": client_secret,
            }))
            .send()
            .await?;
        Ok(())
    }

    // --- Auth Login ---

    /// Login with username/password and get tokens.
    ///
    /// Uses the /api/v1/auth/login endpoint.
    pub async fn login(
        &self,
        username: &str,
        password: &str,
    ) -> Result<TokenResponse, GGIDError> {
        let resp = self
            .http
            .post(format!("{}/api/v1/auth/login", self.base_url))
            .header("X-Tenant-ID", &self.tenant_id)
            .json(&json!({
                "username": username,
                "password": password,
            }))
            .send()
            .await?;

        if !resp.status().is_success() {
            let status = resp.status().as_u16();
            let body = resp.text().await.unwrap_or_default();
            return Err(GGIDError::Api { status, body });
        }

        Ok(resp.json().await?)
    }

    // --- OAuth Introspect ---

    /// Introspect a token to check its status.
    ///
    /// Returns token metadata including active status, expiry, and scope.
    pub async fn introspect_token(
        &self,
        token: &str,
        client_id: &str,
        client_secret: &str,
    ) -> Result<IntrospectionResult, GGIDError> {
        let resp = self
            .http
            .post(format!("{}/api/v1/oauth/introspect", self.base_url))
            .header("X-Tenant-ID", &self.tenant_id)
            .header("Content-Type", "application/x-www-form-urlencoded")
            .body(format!(
                "token={}&client_id={}&client_secret={}",
                urlencoding::encode(token),
                urlencoding::encode(client_id),
                urlencoding::encode(client_secret),
            ))
            .send()
            .await?;

        if !resp.status().is_success() {
            let status = resp.status().as_u16();
            let body = resp.text().await.unwrap_or_default();
            return Err(GGIDError::Api { status, body });
        }

        Ok(resp.json().await?)
    }

    // --- Webhooks ---

    /// List all webhooks for the current tenant.
    pub async fn list_webhooks(&self, token: &str) -> Result<Vec<Webhook>, GGIDError> {
        let resp = self
            .http
            .get(format!("{}/api/v1/webhooks", self.base_url))
            .header("Authorization", format!("Bearer {}", token))
            .header("X-Tenant-ID", &self.tenant_id)
            .send()
            .await?;

        if !resp.status().is_success() {
            let status = resp.status().as_u16();
            let body = resp.text().await.unwrap_or_default();
            return Err(GGIDError::Api { status, body });
        }

        Ok(resp.json().await?)
    }

    /// Create a new webhook.
    pub async fn create_webhook(
        &self,
        token: &str,
        url: &str,
        events: &[&str],
    ) -> Result<Webhook, GGIDError> {
        let resp = self
            .http
            .post(format!("{}/api/v1/webhooks", self.base_url))
            .header("Authorization", format!("Bearer {}", token))
            .header("X-Tenant-ID", &self.tenant_id)
            .json(&json!({
                "url": url,
                "events": events,
            }))
            .send()
            .await?;

        if !resp.status().is_success() {
            let status = resp.status().as_u16();
            let body = resp.text().await.unwrap_or_default();
            return Err(GGIDError::Api { status, body });
        }

        Ok(resp.json().await?)
    }

    /// Delete a webhook by ID.
    pub async fn delete_webhook(&self, token: &str, webhook_id: &str) -> Result<(), GGIDError> {
        let resp = self
            .http
            .delete(format!("{}/api/v1/webhooks/{}", self.base_url, webhook_id))
            .header("Authorization", format!("Bearer {}", token))
            .header("X-Tenant-ID", &self.tenant_id)
            .send()
            .await?;

        if !resp.status().is_success() {
            let status = resp.status().as_u16();
            let body = resp.text().await.unwrap_or_default();
            return Err(GGIDError::Api { status, body });
        }

        Ok(())
    }

    // --- RBAC ---

    /// Check if a token has permission for a resource+action.
    ///
    /// Returns `true` if allowed, `false` if denied.
    pub async fn check_permission(
        &self,
        token: &str,
        resource: &str,
        action: &str,
    ) -> Result<bool, GGIDError> {
        RBACService::new(self.clone()).check_permission(token, resource, action).await
    }

    /// Assign a role to a user.
    pub async fn assign_role(
        &self,
        token: &str,
        user_id: &str,
        role_id: &str,
    ) -> Result<(), GGIDError> {
        RBACService::new(self.clone()).assign_role(token, user_id, role_id).await
    }

    /// Revoke a role from a user.
    pub async fn revoke_role(
        &self,
        token: &str,
        user_id: &str,
        role_id: &str,
    ) -> Result<(), GGIDError> {
        RBACService::new(self.clone()).revoke_role(token, user_id, role_id).await
    }

    /// Get all roles for a user.
    pub async fn get_user_roles(&self, token: &str, user_id: &str) -> Result<Vec<Role>, GGIDError> {
        RBACService::new(self.clone()).get_user_roles(token, user_id).await
    }

    /// List all roles in the tenant.
    pub async fn list_roles(&self, token: &str) -> Result<Vec<Role>, GGIDError> {
        RBACService::new(self.clone()).list_roles(token).await
    }

    /// List all permissions (permission tree).
    pub async fn list_permissions(&self, token: &str) -> Result<Vec<Permission>, GGIDError> {
        RBACService::new(self.clone()).list_permissions(token).await
    }

    // --- OAuth Discovery ---

    /// Get OIDC discovery document.
    pub async fn get_discovery(&self) -> Result<serde_json::Value, GGIDError> {
        let resp = self
            .http
            .get(format!("{}/.well-known/openid-configuration", self.base_url))
            .send()
            .await?;

        if !resp.status().is_success() {
            let status = resp.status().as_u16();
            let body = resp.text().await.unwrap_or_default();
            return Err(GGIDError::Api { status, body });
        }

        Ok(resp.json().await?)
    }

    // --- ABAC ---

    /// Evaluate ABAC conditions.
    pub async fn evaluate_abac(
        &self,
        token: &str,
        request: ABACEvalRequest,
    ) -> Result<ABACEvalResult, GGIDError> {
        ABACService::new(self.clone()).evaluate_abac(token, request).await
    }

    /// Check a policy with full context.
    pub async fn check_policy(
        &self,
        token: &str,
        request: PolicyCheckRequest,
    ) -> Result<PolicyCheckResult, GGIDError> {
        ABACService::new(self.clone()).check_policy(token, request).await
    }
}

/// Builder for custom GGIDClient configuration.
#[derive(Default)]
pub struct GGIDClientBuilder {
    base_url: Option<String>,
    tenant_id: Option<String>,
    timeout: Option<Duration>,
}

impl GGIDClientBuilder {
    pub fn base_url(mut self, url: impl Into<String>) -> Self {
        self.base_url = Some(url.into());
        self
    }

    pub fn tenant_id(mut self, id: impl Into<String>) -> Self {
        self.tenant_id = Some(id.into());
        self
    }

    pub fn timeout(mut self, dur: Duration) -> Self {
        self.timeout = Some(dur);
        self
    }

    pub fn build(self) -> Result<GGIDClient, GGIDError> {
        let base_url = self
            .base_url
            .ok_or_else(|| GGIDError::Other("base_url is required".into()))?;
        let tenant_id = self
            .tenant_id
            .ok_or_else(|| GGIDError::Other("tenant_id is required".into()))?;

        let mut builder = HttpClient::builder().timeout(self.timeout.unwrap_or(Duration::from_secs(10)));
        let http = builder.build().map_err(|e| GGIDError::Other(e.to_string()))?;

        Ok(GGIDClient {
            base_url: base_url.trim_end_matches('/').to_string(),
            tenant_id,
            http,
        })
    }
}
