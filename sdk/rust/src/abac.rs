use serde_json::json;

use crate::client::GGIDClient;
use crate::error::GGIDError;
use crate::types::*;

pub struct ABACService {
    client: GGIDClient,
}

impl ABACService {
    pub(crate) fn new(client: GGIDClient) -> Self {
        Self { client }
    }

    /// Evaluate ABAC conditions against GGID policy engine.
    pub async fn evaluate_abac(
        &self,
        token: &str,
        request: ABACEvalRequest,
    ) -> Result<ABACEvalResult, GGIDError> {
        let resp = self
            .client
            .http
            .post(format!("{}/api/v1/policies/abac/evaluate", self.client.base_url))
            .header("Authorization", format!("Bearer {}", token))
            .header("X-Tenant-ID", &self.client.tenant_id)
            .json(&request)
            .send()
            .await?;

        if !resp.status().is_success() {
            let status = resp.status().as_u16();
            let body = resp.text().await.unwrap_or_default();
            return Err(GGIDError::Api { status, body });
        }

        Ok(resp.json().await?)
    }

    /// Check a policy with full subject and condition context.
    pub async fn check_policy(
        &self,
        token: &str,
        request: PolicyCheckRequest,
    ) -> Result<PolicyCheckResult, GGIDError> {
        let resp = self
            .client
            .http
            .post(format!("{}/api/v1/policies/check", self.client.base_url))
            .header("Authorization", format!("Bearer {}", token))
            .header("X-Tenant-ID", &self.client.tenant_id)
            .json(&request)
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
