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

/// AI Agent registration request.
#[derive(Debug, Clone, Serialize)]
pub struct AgentRegistration {
    pub name: String,
    #[serde(rename = "type")]
    pub agent_type: String,
    #[serde(skip_serializing_if = "String::is_empty")]
    pub owner_user_id: String,
    #[serde(skip_serializing_if = "String::is_empty")]
    pub description: String,
    pub allowed_scopes: Vec<String>,
    #[serde(skip_serializing_if = "Vec::is_empty")]
    pub allowed_mcp_servers: Vec<String>,
    #[serde(skip_serializing_if = "is_zero")]
    pub max_delegation_depth: i32,
    #[serde(skip_serializing_if = "is_zero")]
    pub rate_limit_per_min: i32,
}

fn is_zero(v: &i32) -> bool {
    *v == 0
}

/// AI Agent.
#[derive(Debug, Clone, Deserialize)]
pub struct Agent {
    pub id: String,
    pub tenant_id: String,
    pub name: String,
    #[serde(rename = "type")]
    pub agent_type: String,
    pub owner_user_id: String,
    pub client_id: String,
    pub status: String,
    pub allowed_scopes: Vec<String>,
    pub max_delegation_depth: i32,
}

/// Agent token exchange response.
#[derive(Debug, Clone, Deserialize)]
pub struct AgentTokenResponse {
    pub access_token: String,
    pub token_type: String,
    pub expires_in: i64,
    pub scope: String,
    pub agent_id: String,
    #[serde(rename = "delegation_depth_remaining")]
    pub delegation_depth: i32,
}

/// Agent token claims.
#[derive(Debug, Clone, Deserialize)]
pub struct AgentTokenClaims {
    pub sub: String,
    pub iss: String,
    pub exp: i64,
    pub iat: i64,
    pub agent_id: String,
    #[serde(rename = "agent_type")]
    pub agent_type: String,
    #[serde(rename = "is_agent_token")]
    pub is_agent_token: bool,
    #[serde(rename = "max_delegation_depth")]
    pub max_delegation_depth: i32,
}

/// Access request (IGA) request.
#[derive(Debug, Clone, Serialize)]
pub struct AccessRequest {
    pub user_id: String,
    pub resource: String,
    pub action: String,
    #[serde(skip_serializing_if = "String::is_empty")]
    pub reason: String,
}

/// Access request response.
#[derive(Debug, Clone, Deserialize)]
pub struct AccessRequestResponse {
    pub id: String,
    pub user_id: String,
    pub resource: String,
    pub action: String,
    pub status: String,
    pub reason: Option<String>,
}
