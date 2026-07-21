use std::collections::HashMap;

use jsonwebtoken::{decode, decode_header, Algorithm, DecodingKey, Validation};
use serde_json::Value;

use crate::client::GGIDClient;
use crate::error::GGIDError;
use crate::types::*;

pub struct AuthService {
    client: GGIDClient,
}

impl AuthService {
    pub(crate) fn new(client: GGIDClient) -> Self {
        Self { client }
    }

    /// Verify a JWT token by fetching JWKS and checking signature.
    pub async fn verify_token(&self, token: &str) -> Result<Claims, GGIDError> {
        // Decode header to get kid
        let header = decode_header(token)?;
        let kid = header.kid.ok_or_else(|| {
            GGIDError::InvalidToken("missing kid in token header".into())
        })?;

        // Fetch JWKS
        let jwks: Value = self
            .client
            .http
            .get(format!("{}/.well-known/jwks.json", self.client.base_url))
            .header("X-Tenant-ID", &self.client.tenant_id)
            .send()
            .await?
            .json()
            .await?;

        // Find matching key
        let keys = jwks["keys"]
            .as_array()
            .ok_or_else(|| GGIDError::InvalidToken("invalid JWKS format".into()))?;

        let key_obj = keys
            .iter()
            .find(|k| k["kid"].as_str() == Some(kid.as_str()))
            .ok_or_else(|| GGIDError::InvalidToken("no matching key in JWKS".into()))?;

        let n = key_obj["n"].as_str().ok_or_else(|| {
            GGIDError::InvalidToken("invalid key: missing n".into())
        })?;
        let e = key_obj["e"].as_str().ok_or_else(|| {
            GGIDError::InvalidToken("invalid key: missing e".into())
        })?;

        // Build RSA public key from JWK
        let decoding_key = DecodingKey::from_rsa_components(n, e)
            .map_err(|e| GGIDError::InvalidToken(format!("invalid RSA key: {}", e)))?;

        let mut validation = Validation::new(Algorithm::RS256);
        validation.validate_exp = true;
        validation.validate_aud = false;

        let token_data = decode::<Claims>(token, &decoding_key, &validation)?;
        Ok(token_data.claims)
    }

    /// Get user info from the userinfo endpoint.
    pub async fn get_user_info(&self, token: &str) -> Result<UserInfo, GGIDError> {
 let resp = self
            .client
            .http
            .get(format!("{}/api/v1/oauth/userinfo", self.client.base_url))
            .header("Authorization", format!("Bearer {}", token))
            .header("X-Tenant-ID", &self.client.tenant_id)
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
