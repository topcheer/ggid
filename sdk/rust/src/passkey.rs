//! Passkey/WebAuthn registration and authentication API calls.

use serde_json::Value;
use super::client::GGIDClient;
use super::error::GGIDError;

async fn parse_resp(resp: reqwest::Response) -> Result<Value, GGIDError> {
    let status = resp.status();
    let text = resp.text().await.unwrap_or_default();
    if !status.is_success() {
        return Err(GGIDError::Api { status: status.as_u16(), body: text });
    }
    Ok(serde_json::from_str(&text)?)
}

impl GGIDClient {
    /// Begin WebAuthn/Passkey registration. Returns server challenge options.
    pub async fn begin_passkey_registration(
        &self, token: &str, device_name: &str,
    ) -> Result<Value, GGIDError> {
        let body = serde_json::json!({"type": "webauthn", "name": device_name});
        let resp = self.http
            .post(format!("{}/api/v1/auth/mfa/enroll", self.base_url))
            .header("Authorization", format!("Bearer {}", token))
            .header("X-Tenant-ID", &self.tenant_id)
            .json(&body)
            .send().await?;
        parse_resp(resp).await
    }

    /// Finish WebAuthn/Passkey registration.
    pub async fn finish_passkey_registration(
        &self, token: &str, device_id: &str, attestation: &str,
    ) -> Result<Value, GGIDError> {
        let body = serde_json::json!({"device_id": device_id, "code": attestation});
        let resp = self.http
            .post(format!("{}/api/v1/auth/mfa/verify", self.base_url))
            .header("Authorization", format!("Bearer {}", token))
            .header("X-Tenant-ID", &self.tenant_id)
            .json(&body)
            .send().await?;
        parse_resp(resp).await
    }

    /// Begin WebAuthn/Passkey login.
    pub async fn begin_passkey_login(&self, username: &str) -> Result<Value, GGIDError> {
        let body = serde_json::json!({"username": username});
        let resp = self.http
            .post(format!("{}/api/v1/auth/webauthn/login/begin", self.base_url))
            .header("X-Tenant-ID", &self.tenant_id)
            .json(&body)
            .send().await?;
        parse_resp(resp).await
    }

    /// Finish WebAuthn/Passkey login.
    pub async fn finish_passkey_login(&self, assertion: &str) -> Result<Value, GGIDError> {
        let body = serde_json::json!({"assertion": assertion});
        let resp = self.http
            .post(format!("{}/api/v1/auth/webauthn/login/finish", self.base_url))
            .header("X-Tenant-ID", &self.tenant_id)
            .json(&body)
            .send().await?;
        parse_resp(resp).await
    }
}
