//! Passkey/WebAuthn registration and authentication API calls.
//!
//! # Usage
//!
//! ```no_run
//! use ggid::client::GGIDClient;
//!
//! # async fn example() {
//! let client = GGIDClient::builder()
//!     .base_url("https://ggid.example.com")
//!     .build()
//!     .unwrap();
//! let options = client.begin_passkey_registration("token", "my-key").await.unwrap();
//! # }
//! ```

use serde_json::Value;

impl super::client::GGIDClient {
    /// Begin WebAuthn/Passkey registration. Returns server challenge options.
    pub async fn begin_passkey_registration(
        &self,
        token: &str,
        device_name: &str,
    ) -> Result<Value, super::error::GGIDError> {
        self.post_authenticated(
            "/api/v1/auth/mfa/enroll",
            token,
            &serde_json::json!({"type": "webauthn", "name": device_name}),
        )
        .await
    }

    /// Finish WebAuthn/Passkey registration by verifying the attestation.
    pub async fn finish_passkey_registration(
        &self,
        token: &str,
        device_id: &str,
        attestation: &str,
    ) -> Result<Value, super::error::GGIDError> {
        self.post_authenticated(
            "/api/v1/auth/mfa/verify",
            token,
            &serde_json::json!({"device_id": device_id, "code": attestation}),
        )
        .await
    }

    /// Begin WebAuthn/Passkey login. Returns server challenge options.
    pub async fn begin_passkey_login(
        &self,
        username: &str,
    ) -> Result<Value, super::error::GGIDError> {
        self.post_public(
            "/api/v1/auth/webauthn/login/begin",
            &serde_json::json!({"username": username}),
        )
        .await
    }

    /// Finish WebAuthn/Passkey login by verifying the assertion.
    pub async fn finish_passkey_login(
        &self,
        assertion: &str,
    ) -> Result<Value, super::error::GGIDError> {
        self.post_public(
            "/api/v1/auth/webauthn/login/finish",
            &serde_json::json!({"assertion": assertion}),
        )
        .await
    }
}
