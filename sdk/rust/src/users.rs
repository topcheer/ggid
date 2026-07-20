//! User management CRUD — typed wrappers around existing client methods.
//! The client already has get_user/delete_user returning Value; this adds typed versions.

use serde::{Deserialize, Serialize};

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

impl super::client::GGIDClient {
    /// Create a new user (typed).
    pub async fn create_user_typed(
        &self,
        token: &str,
        req: &CreateUserRequest,
    ) -> Result<User, super::error::GGIDError> {
        let v = serde_json::to_value(req).unwrap();
        let result = self.api_post("/api/v1/users", token, &v).await?;
        serde_json::from_value(result).map_err(|e| super::error::GGIDError::Api(format!("parse: {e}")))
    }

    /// List users (typed).
    pub async fn list_users_typed(
        &self,
        token: &str,
        page: u32,
        page_size: u32,
    ) -> Result<Vec<User>, super::error::GGIDError> {
        let path = format!("/api/v1/users?page={}&page_size={}", page, page_size);
        let v = self.api_get(&path, token).await?;
        let users_val = v.get("users").unwrap_or(&v);
        serde_json::from_value(users_val.clone()).map_err(|e| super::error::GGIDError::Api(format!("parse: {e}")))
    }

    /// Update user (typed).
    pub async fn update_user_typed(
        &self,
        token: &str,
        user_id: &str,
        display_name: Option<&str>,
        phone: Option<&str>,
    ) -> Result<User, super::error::GGIDError> {
        let mut body = serde_json::Map::new();
        if let Some(n) = display_name { body.insert("display_name".into(), n.into()); }
        if let Some(p) = phone { body.insert("phone".into(), p.into()); }
        let v = serde_json::Value::Object(body);
        let result = self.api_put(&format!("/api/v1/users/{}", user_id), token, &v).await?;
        serde_json::from_value(result).map_err(|e| super::error::GGIDError::Api(format!("parse: {e}")))
    }
}
