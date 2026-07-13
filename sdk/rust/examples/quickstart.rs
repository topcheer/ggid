//! Quickstart example: minimal HTTP server with GGID auth.
//!
//! Run: `cargo run --example quickstart`
//!
//! This example shows the simplest way to use the GGID SDK.
//! For a full Axum integration example, enable the `middleware` feature:
//! ```toml
//! ggid = { version = "0.1", features = ["middleware"] }
//! ```

use ggid::GGIDClient;

#[tokio::main]
async fn main() {
    let ggid = GGIDClient::new(
        "https://ggid.iot2.win",
        "00000000-0000-0000-0000-000000000001",
    );

    // Example: verify a token
    let fake_token = "eyJhbGciOiJSUzI1NiJ9.eyJzdWIiOiJ1c2VyLTEyMyJ9.invalid";
    match ggid.verify_token(fake_token).await {
        Ok(claims) => println!("Verified! User: {}", claims.sub),
        Err(e) => println!("Token verification failed (expected with fake token): {}", e),
    }

    // Example: build an authorize URL
    let url = ggid.get_authorize_url(
        "my-client-id",
        "https://myapp.com/callback",
        Some("openid profile email"),
        Some("random-state"),
    );
    println!("Authorize URL: {}", url);

    // Example: check permission (would work with a real token)
    // let allowed = ggid.check_permission(real_token, "products", "read").await?;
    // println!("Allowed: {}", allowed);

    println!("\nGGID Rust SDK quickstart complete!");
}
