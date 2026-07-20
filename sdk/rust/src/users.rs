//! User management CRUD — typed wrappers.

use serde::{Deserialize, Serialize};
use serde_json::Value;
use super::client::GGIDClient;
use super::error::GGIDError;

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct User {
    pub id: String,
    pub username: String,
    pub email: String,
    #[serde(default)]
    pub display_name: String,
    #[serde(default)]
    pub phone: String,
    #[serde(default)]
    pub status: String,
}

#[derive(Debug, Clone, Serialize)]
pub struct CreateUserRequest {
    pub username: String,
    pub email: String,
    pub password: String,
    #[serde(skip_serializing_if = "String::is_empty", default)]
    pub display_name: String,
}

async fn parse_resp(resp: reqwest::Response) -> Result<Value, GGIDError> {
    let status = resp.status();
    let text = resp.text().await.unwrap_or_default();
    if !status.is_success() {
        return Err(GGIDError::Api { status: status.as_u16(), body: text });
    }
    Ok(serde_json::from_str(&text)?)
}

impl GGIDClient {
    /// Create a new user (typed).
    pub async fn create_user_typed(
        &self, token: &str, req: &CreateUserRequest,
    ) -> Result<User, GGIDError> {
        let v = serde_json::to_value(req)?;
        let resp = self.http
            .post(format!("{}/api/v1/users", self.base_url))
            .header("Authorization", format!("Bearer {}", token))
            .header("X-Tenant-ID", &self.tenant_id)
            .json(&v)
            .send().await?;
        let result: Value = parse_resp(resp).await?;
        Ok(serde_json::from_value(result)?)
    }

    /// List users (typed).
    pub async fn list_users_typed(
        &self, token: &str, page: u32, page_size: u32,
    ) -> Result<Vec<User>, GGIDError> {
        let path = format!("/api/v1/users?page={}&page_size={}", page, page_size);
        let resp = self.http
            .get(format!("{}{}", self.base_url, path))
            .header("Authorization", format!("Bearer {}", token))
            .header("X-Tenant-ID", &self.tenant_id)
            .send().await?;
        let v: Value = parse_resp(resp).await?;
        let users_val = v.get("users").unwrap_or(&v);
        Ok(serde_json::from_value(users_val.clone())?)
    }
}
