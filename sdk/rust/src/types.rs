use serde::{Deserialize, Serialize};

/// JWT claims extracted from a verified token.
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct Claims {
    pub sub: String,
    pub tenant_id: String,
    pub roles: Vec<String>,
    pub scope: String,
    pub exp: u64,
    pub iat: u64,
    pub iss: String,
}

/// User info from GGID userinfo endpoint.
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct UserInfo {
    pub sub: String,
    pub name: Option<String>,
    pub email: Option<String>,
    pub roles: Vec<String>,
    pub picture: Option<String>,
}

/// OAuth token exchange response.
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct TokenResponse {
    pub access_token: String,
    pub refresh_token: Option<String>,
    pub id_token: Option<String>,
    pub expires_in: u64,
    pub token_type: String,
}

/// Role definition.
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct Role {
    pub id: String,
    pub key: String,
    pub name: String,
    pub permissions: Vec<String>,
}

/// Permission entry.
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct Permission {
    pub resource: String,
    pub actions: Vec<String>,
    pub description: Option<String>,
}

/// ABAC evaluation request.
#[derive(Debug, Clone, Serialize)]
pub struct ABACEvalRequest {
    pub action: String,
    pub resource: String,
    pub conditions: Vec<ABACCondition>,
}

/// ABAC condition.
#[derive(Debug, Clone, Serialize)]
pub struct ABACCondition {
    pub field: String,
    pub operator: String,
    pub value: serde_json::Value,
}

/// ABAC evaluation result.
#[derive(Debug, Clone, Deserialize)]
pub struct ABACEvalResult {
    pub matched: bool,
    pub matched_rules: Vec<String>,
}

/// Policy check request.
#[derive(Debug, Clone, Serialize)]
pub struct PolicyCheckRequest {
    pub action: String,
    pub resource: String,
    pub user_id: String,
    pub conditions: Option<std::collections::HashMap<String, serde_json::Value>>,
}

/// Policy check result.
#[derive(Debug, Clone, Deserialize)]
pub struct PolicyCheckResult {
    pub allowed: bool,
    pub matched_rules: Vec<String>,
    pub reason: Option<String>,
}

/// Token introspection result (RFC 7662).
#[derive(Debug, Clone, Deserialize)]
pub struct IntrospectionResult {
    pub active: bool,
    #[serde(default)]
    pub scope: Option<String>,
    #[serde(default)]
    pub client_id: Option<String>,
    #[serde(default)]
    pub username: Option<String>,
    #[serde(default)]
    pub token_type: Option<String>,
    #[serde(default)]
    pub exp: Option<u64>,
    #[serde(default)]
    pub iat: Option<u64>,
    #[serde(default)]
    pub sub: Option<String>,
    #[serde(default)]
    pub aud: Option<Vec<String>>,
    #[serde(default)]
    pub iss: Option<String>,
    #[serde(default)]
    pub jti: Option<String>,
}

/// Webhook configuration.
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct Webhook {
    pub id: String,
    pub url: String,
    pub events: Vec<String>,
    pub secret: Option<String>,
    pub active: bool,
    #[serde(default)]
    pub created_at: Option<String>,
}
