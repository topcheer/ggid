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
    /// Login via OAuth2 password grant to /api/v1/oauth/token.
    pub async fn login(
        &self,
        username: &str,
        password: &str,
    ) -> Result<TokenResponse, GGIDError> {
        let resp = self
            .http
            .post(format!("{}/api/v1/oauth/token", self.base_url))
            .header("X-Tenant-ID", &self.tenant_id)
            .form(&[
                ("grant_type", "password"),
                ("username", username),
                ("password", password),
            ])
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

    /// Exchange a token using RFC 8693 Token Exchange.
    ///
    /// Trades a subject token (e.g., access token) for a new access token
    /// with potentially different audience/scope. Used for delegation.
    pub async fn exchange_token(
        &self,
        client_id: &str,
        subject_token: &str,
        subject_token_type: &str,
        scope: Option<&str>,
    ) -> Result<TokenResponse, GGIDError> {
        let mut form = vec![
            ("grant_type", "urn:ietf:params:oauth:grant-type:token-exchange".to_string()),
            ("client_id", client_id.to_string()),
            ("subject_token", subject_token.to_string()),
            ("subject_token_type", subject_token_type.to_string()),
        ];
        if let Some(s) = scope {
            form.push(("scope", s.to_string()));
        }

        let resp = self
            .http
            .post(format!("{}/api/v1/oauth/token", self.base_url))
            .header("X-Tenant-ID", &self.tenant_id)
            .form(&form)
            .send()
            .await?;

        if !resp.status().is_success() {
            let status = resp.status().as_u16();
            let body = resp.text().await.unwrap_or_default();
            return Err(GGIDError::Api { status, body });
        }
        Ok(resp.json().await?)
    }

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

    // --- User Management CRUD ---

    /// Create a new user.
    pub async fn create_user(
        &self,
        token: &str,
        username: &str,
        email: &str,
        password: &str,
    ) -> Result<serde_json::Value, GGIDError> {
        let resp = self.http
            .post(format!("{}/api/v1/users", self.base_url))
            .header("Authorization", format!("Bearer {}", token))
            .header("X-Tenant-ID", &self.tenant_id)
            .json(&json!({"username": username, "email": email, "password": password}))
            .send().await
            .map_err(|e| GGIDError::HttpMsg(e.to_string()))?;
        parse_json(resp).await
    }

    /// Get a user by ID.
    pub async fn get_user(&self, token: &str, user_id: &str) -> Result<serde_json::Value, GGIDError> {
        let resp = self.http
            .get(format!("{}/api/v1/users/{}", self.base_url, user_id))
            .header("Authorization", format!("Bearer {}", token))
            .header("X-Tenant-ID", &self.tenant_id)
            .send().await
            .map_err(|e| GGIDError::HttpMsg(e.to_string()))?;
        parse_json(resp).await
    }

    /// List users with optional pagination.
    pub async fn list_users(&self, token: &str, limit: u32, offset: u32) -> Result<serde_json::Value, GGIDError> {
        let resp = self.http
            .get(format!("{}/api/v1/users?limit={}&offset={}", self.base_url, limit, offset))
            .header("Authorization", format!("Bearer {}", token))
            .header("X-Tenant-ID", &self.tenant_id)
            .send().await
            .map_err(|e| GGIDError::HttpMsg(e.to_string()))?;
        parse_json(resp).await
    }

    /// Update a user by ID.
    pub async fn update_user(
        &self,
        token: &str,
        user_id: &str,
        data: serde_json::Value,
    ) -> Result<serde_json::Value, GGIDError> {
        let resp = self.http
            .put(format!("{}/api/v1/users/{}", self.base_url, user_id))
            .header("Authorization", format!("Bearer {}", token))
            .header("X-Tenant-ID", &self.tenant_id)
            .json(&data)
            .send().await
            .map_err(|e| GGIDError::HttpMsg(e.to_string()))?;
        parse_json(resp).await
    }

    /// Delete a user by ID.
    pub async fn delete_user(&self, token: &str, user_id: &str) -> Result<(), GGIDError> {
        let resp = self.http
            .delete(format!("{}/api/v1/users/{}", self.base_url, user_id))
            .header("Authorization", format!("Bearer {}", token))
            .header("X-Tenant-ID", &self.tenant_id)
            .send().await
            .map_err(|e| GGIDError::HttpMsg(e.to_string()))?;
        if !resp.status().is_success() {
            return Err(GGIDError::HttpMsg(format!("delete failed: {}", resp.status())));
        }
        Ok(())
    }

    /// Create a role.
    pub async fn create_role(
        &self,
        token: &str,
        name: &str,
        key: &str,
    ) -> Result<serde_json::Value, GGIDError> {
        let resp = self.http
            .post(format!("{}/api/v1/roles", self.base_url))
            .header("Authorization", format!("Bearer {}", token))
            .header("X-Tenant-ID", &self.tenant_id)
            .json(&json!({"name": name, "key": key}))
            .send().await
            .map_err(|e| GGIDError::HttpMsg(e.to_string()))?;
        parse_json(resp).await
    }

    // --- Passkey/WebAuthn ---

    /// Start passkey registration (returns challenge + RP ID).
    pub async fn passkey_register_begin(
        &self,
        token: &str,
        user_id: &str,
    ) -> Result<serde_json::Value, GGIDError> {
        let resp = self.http
            .post(format!("{}/api/v1/auth/webauthn/register/begin", self.base_url))
            .header("Authorization", format!("Bearer {}", token))
            .header("X-Tenant-ID", &self.tenant_id)
            .json(&json!({"user_id": user_id}))
            .send().await
            .map_err(|e| GGIDError::HttpMsg(e.to_string()))?;
        parse_json(resp).await
    }

    /// Start passkey authentication (returns challenge).
    pub async fn passkey_auth_begin(
        &self,
        token: &str,
    ) -> Result<serde_json::Value, GGIDError> {
        let resp = self.http
            .post(format!("{}/api/v1/auth/webauthn/login/begin", self.base_url))
            .header("Authorization", format!("Bearer {}", token))
            .header("X-Tenant-ID", &self.tenant_id)
            .send().await
            .map_err(|e| GGIDError::HttpMsg(e.to_string()))?;
        parse_json(resp).await
    }

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

    // --- Agent Identity ---

    /// Register a new AI agent.
    pub async fn register_agent(
        &self,
        token: &str,
        reg: AgentRegistration,
    ) -> Result<Agent, GGIDError> {
        let resp = self
            .http
            .post(format!("{}/api/v1/agents/register", self.base_url))
            .header("Authorization", format!("Bearer {}", token))
            .header("X-Tenant-ID", &self.tenant_id)
            .json(&reg)
            .send()
            .await?;

        if !resp.status().is_success() {
            let status = resp.status().as_u16();
            let body = resp.text().await.unwrap_or_default();
            return Err(GGIDError::Api { status, body });
        }
        Ok(resp.json().await?)
    }

    /// List all agents for the current tenant.
    pub async fn list_agents(&self, token: &str) -> Result<Vec<Agent>, GGIDError> {
        use serde::Deserialize;
        #[derive(Deserialize)]
        struct AgentList {
            agents: Vec<Agent>,
        }
        let resp = self
            .http
            .get(format!("{}/api/v1/agents", self.base_url))
            .header("Authorization", format!("Bearer {}", token))
            .header("X-Tenant-ID", &self.tenant_id)
            .send()
            .await?;

        if !resp.status().is_success() {
            let status = resp.status().as_u16();
            let body = resp.text().await.unwrap_or_default();
            return Err(GGIDError::Api { status, body });
        }
        let result: AgentList = resp.json().await?;
        Ok(result.agents)
    }

    /// Exchange a user token for an agent-scoped token.
    pub async fn exchange_agent_token(
        &self,
        agent_id: &str,
        subject_token: &str,
        scopes: &[&str],
    ) -> Result<AgentTokenResponse, GGIDError> {
        let resp = self
            .http
            .post(format!("{}/api/v1/agents/token", self.base_url))
            .header("X-Tenant-ID", &self.tenant_id)
            .json(&json!({
                "agent_id": agent_id,
                "subject_token": subject_token,
                "scope": scopes,
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

    /// Verify an agent token and return its claims.
    pub async fn verify_agent_token(&self, token: &str) -> Result<AgentTokenClaims, GGIDError> {
        let resp = self
            .http
            .post(format!("{}/api/v1/agents/verify", self.base_url))
            .header("X-Tenant-ID", &self.tenant_id)
            .json(&json!({"token": token}))
            .send()
            .await?;

        if !resp.status().is_success() {
            let status = resp.status().as_u16();
            let body = resp.text().await.unwrap_or_default();
            return Err(GGIDError::Api { status, body });
        }
        Ok(resp.json().await?)
    }

    // --- Access Request (IGA) ---

    /// Create an access request.
    pub async fn create_access_request(
        &self,
        token: &str,
        req: AccessRequest,
    ) -> Result<AccessRequestResponse, GGIDError> {
        let resp = self
            .http
            .post(format!("{}/api/v1/access-requests", self.base_url))
            .header("Authorization", format!("Bearer {}", token))
            .header("X-Tenant-ID", &self.tenant_id)
            .json(&req)
            .send()
            .await?;

        if !resp.status().is_success() {
            let status = resp.status().as_u16();
            let body = resp.text().await.unwrap_or_default();
            return Err(GGIDError::Api { status, body });
        }
        Ok(resp.json().await?)
    }

    /// List access requests for the current tenant.
    pub async fn list_access_requests(
        &self, token: &str
    ) -> Result<Vec<AccessRequestResponse>, GGIDError> {
        use serde::Deserialize;
        #[derive(Deserialize)]
        struct RequestList {
            #[serde(default)]
            requests: Vec<AccessRequestResponse>,
        }
        let resp = self
            .http
            .get(format!("{}/api/v1/access-requests", self.base_url))
            .header("Authorization", format!("Bearer {}", token))
            .header("X-Tenant-ID", &self.tenant_id)
            .send()
            .await?;

        if !resp.status().is_success() {
            let status = resp.status().as_u16();
            let body = resp.text().await.unwrap_or_default();
            return Err(GGIDError::Api { status, body });
        }
        let result: RequestList = resp.json().await?;
        Ok(result.requests)
    }

    /// Approve an access request.
    pub async fn approve_access_request(
        &self,
        token: &str,
        request_id: &str,
        comment: &str,
    ) -> Result<AccessRequestResponse, GGIDError> {
        let resp = self
            .http
            .post(format!("{}/api/v1/access-requests/{}/approve", self.base_url, request_id))
            .header("Authorization", format!("Bearer {}", token))
            .header("X-Tenant-ID", &self.tenant_id)
            .json(&json!({"comment": comment}))
            .send()
            .await?;

        if !resp.status().is_success() {
            let status = resp.status().as_u16();
            let body = resp.text().await.unwrap_or_default();
            return Err(GGIDError::Api { status, body });
        }
        Ok(resp.json().await?)
    }

    /// Reject an access request.
    pub async fn reject_access_request(
        &self,
        token: &str,
        request_id: &str,
        comment: &str,
    ) -> Result<AccessRequestResponse, GGIDError> {
        let resp = self
            .http
            .post(format!("{}/api/v1/access-requests/{}/reject", self.base_url, request_id))
            .header("Authorization", format!("Bearer {}", token))
            .header("X-Tenant-ID", &self.tenant_id)
            .json(&json!({"comment": comment}))
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

    /// Obtain a token using OAuth2 client_credentials grant (M2M).
    pub async fn client_credentials(&self, client_id: &str, client_secret: &str, scope: &str) -> Result<TokenResponse, GGIDError> {
        let form = vec![
            ("grant_type", "client_credentials"),
            ("client_id", client_id),
            ("client_secret", client_secret),
            ("scope", scope),
        ];
        let resp = self.http
            .post(format!("{}/api/v1/oauth/token", self.base_url))
            .header("X-Tenant-ID", &self.tenant_id)
            .form(&form)
            .send()
            .await?;
        if !resp.status().is_success() {
            let status = resp.status().as_u16();
            let body = resp.text().await.unwrap_or_default();
            return Err(GGIDError::Api { status, body });
        }
        Ok(resp.json().await?)
    }
}

/// Standalone helper: parse HTTP response to JSON or error.
async fn parse_json(resp: reqwest::Response) -> Result<serde_json::Value, GGIDError> {
    let status = resp.status();
    let body: serde_json::Value = resp.json().await
        .map_err(GGIDError::from)?;
    if !status.is_success() {
        return Err(GGIDError::HttpMsg(format!("{}: {}", status, body)));
    }
    Ok(body)
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
