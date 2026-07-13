use serde_json::json;

use crate::client::GGIDClient;
use crate::error::GGIDError;
use crate::types::*;

pub struct RBACService {
    client: GGIDClient,
}

impl RBACService {
    pub(crate) fn new(client: GGIDClient) -> Self {
        Self { client }
    }

    /// Check permission via GGID policy engine.
    /// Returns true if allowed, false if denied.
    pub async fn check_permission(
        &self,
        token: &str,
        resource: &str,
        action: &str,
    ) -> Result<bool, GGIDError> {
        // First decode the token to get user_id (without verification for speed)
        let user_id = extract_user_id(token).unwrap_or_default();

        let resp = self
            .client
            .http
            .post(format!("{}/api/v1/policies/check", self.client.base_url))
            .header("Authorization", format!("Bearer {}", token))
            .header("X-Tenant-ID", &self.client.tenant_id)
            .json(&json!({
                "user_id": user_id,
                "resource": resource,
                "action": action,
            }))
            .send()
            .await?;

        if resp.status().is_success() {
            let result: serde_json::Value = resp.json().await?;
            Ok(result.get("allowed").and_then(|v| v.as_bool()).unwrap_or(false))
        } else {
            Ok(false)
        }
    }

    /// Assign a role to a user.
    pub async fn assign_role(
        &self,
        token: &str,
        user_id: &str,
        role_id: &str,
    ) -> Result<(), GGIDError> {
        let resp = self
            .client
            .http
            .post(format!("{}/api/v1/roles/{}/assign", self.client.base_url, role_id))
            .header("Authorization", format!("Bearer {}", token))
            .header("X-Tenant-ID", &self.client.tenant_id)
            .json(&json!({ "user_id": user_id }))
            .send()
            .await?;

        if !resp.status().is_success() {
            let status = resp.status().as_u16();
            let body = resp.text().await.unwrap_or_default();
            return Err(GGIDError::Api { status, body });
        }
        Ok(())
    }

    /// Revoke a role from a user.
    pub async fn revoke_role(
        &self,
        token: &str,
        user_id: &str,
        role_id: &str,
    ) -> Result<(), GGIDError> {
        let resp = self
            .client
            .http
            .post(format!("{}/api/v1/roles/{}/revoke", self.client.base_url, role_id))
            .header("Authorization", format!("Bearer {}", token))
            .header("X-Tenant-ID", &self.client.tenant_id)
            .json(&json!({ "user_id": user_id }))
            .send()
            .await?;

        if !resp.status().is_success() {
            let status = resp.status().as_u16();
            let body = resp.text().await.unwrap_or_default();
            return Err(GGIDError::Api { status, body });
        }
        Ok(())
    }

    /// Get roles for a user.
    pub async fn get_user_roles(
        &self,
        token: &str,
        user_id: &str,
    ) -> Result<Vec<Role>, GGIDError> {
        let resp = self
            .client
            .http
            .get(format!("{}/api/v1/users/{}/roles", self.client.base_url, user_id))
            .header("Authorization", format!("Bearer {}", token))
            .header("X-Tenant-ID", &self.client.tenant_id)
            .send()
            .await?;

        if !resp.status().is_success() {
            let status = resp.status().as_u16();
            let body = resp.text().await.unwrap_or_default();
            return Err(GGIDError::Api { status, body });
        }

        let val: serde_json::Value = resp.json().await?;
        // Handle both array and {data: [...]} responses
        let roles = if let Some(data) = val.get("data").and_then(|v| v.as_array()) {
            data.clone()
        } else if val.is_array() {
            val.as_array().unwrap().clone()
        } else {
            vec![]
        };

        Ok(serde_json::from_value(serde_json::Value::Array(roles))?)
    }

    /// List all roles in the tenant.
    pub async fn list_roles(&self, token: &str) -> Result<Vec<Role>, GGIDError> {
        let resp = self
            .client
            .http
            .get(format!("{}/api/v1/roles", self.client.base_url))
            .header("Authorization", format!("Bearer {}", token))
            .header("X-Tenant-ID", &self.client.tenant_id)
            .send()
            .await?;

        if !resp.status().is_success() {
            let status = resp.status().as_u16();
            let body = resp.text().await.unwrap_or_default();
            return Err(GGIDError::Api { status, body });
        }

        let val: serde_json::Value = resp.json().await?;
        let roles = if let Some(data) = val.get("data").and_then(|v| v.as_array()) {
            data.clone()
        } else if val.is_array() {
            val.as_array().unwrap().clone()
        } else {
            vec![]
        };

        Ok(serde_json::from_value(serde_json::Value::Array(roles))?)
    }

    /// List all permissions (permission tree).
    pub async fn list_permissions(&self, token: &str) -> Result<Vec<Permission>, GGIDError> {
        let resp = self
            .client
            .http
            .get(format!("{}/api/v1/policies/permissions/tree", self.client.base_url))
            .header("Authorization", format!("Bearer {}", token))
            .header("X-Tenant-ID", &self.client.tenant_id)
            .send()
            .await?;

        if !resp.status().is_success() {
            let status = resp.status().as_u16();
            let body = resp.text().await.unwrap_or_default();
            return Err(GGIDError::Api { status, body });
        }

        let val: serde_json::Value = resp.json().await?;
        let perms = if let Some(data) = val.get("data").and_then(|v| v.as_array()) {
            data.clone()
        } else if val.is_array() {
            val.as_array().unwrap().clone()
        } else {
            vec![]
        };

        Ok(serde_json::from_value(serde_json::Value::Array(perms))?)
    }
}

/// Extract user_id from JWT payload without verification (for policy check).
fn extract_user_id(token: &str) -> Option<String> {
    let parts: Vec<&str> = token.split('.').collect();
    if parts.len() != 3 {
        return None;
    }
    use base64ct::{Base64UrlUnpadded, Encoding};
    let payload = Base64UrlUnpadded::decode_vec(parts[1]).ok()?;
    let claims: serde_json::Value = serde_json::from_slice(&payload).ok()?;
    claims.get("sub").and_then(|v| v.as_str()).map(|s| s.to_string())
}
