//! # GGID SDK for Rust
//!
//! Simplest, most flexible IAM integration for Rust applications.
//!
//! ## Quick Start
//! ```no_run
//! use ggid::GGIDClient;
//!
//! # #[tokio::main]
//! # async fn main() -> Result<(), Box<dyn std::error::Error>> {
//! let ggid = GGIDClient::new("https://ggid.iot2.win", "00000000-0000-0000-0000-000000000001");
//! let claims = ggid.verify_token("eyJ...").await?;
//! let allowed = ggid.check_permission("eyJ...", "products", "read").await?;
//! println!("allowed={}, user={}", allowed, claims.sub);
//! # Ok(())
//! # }
//! ```

pub mod client;
pub mod auth;
pub mod rbac;
pub mod abac;
#[cfg(feature = "middleware")]
pub mod middleware;
pub mod types;
pub mod error;

pub use client::GGIDClient;
pub use types::{Claims, UserInfo, TokenResponse, Role, Permission, ABACEvalRequest, ABACEvalResult, ABACCondition, PolicyCheckRequest, PolicyCheckResult, IntrospectionResult, Webhook, Agent, AgentRegistration, AgentTokenResponse, AgentTokenClaims, AccessRequest, AccessRequestResponse};
pub use error::GGIDError;
