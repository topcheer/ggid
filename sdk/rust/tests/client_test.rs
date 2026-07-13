use ggid::*;
use ggid::client::GGIDClientBuilder;

#[test]
fn test_client_creation() {
    let client = GGIDClient::new("https://ggid.iot2.win", "tenant-123");
    assert_eq!(client.base_url, "https://ggid.iot2.win");
    assert_eq!(client.tenant_id, "tenant-123");
}

#[test]
fn test_client_trims_trailing_slash() {
    let client = GGIDClient::new("https://ggid.iot2.win/", "tenant-123");
    assert_eq!(client.base_url, "https://ggid.iot2.win");
}

#[test]
fn test_builder_defaults() {
    let client = GGIDClient::builder()
        .base_url("https://ggid.example.com")
        .tenant_id("t1")
        .build();
    assert!(client.is_ok());
    let client = client.unwrap();
    assert_eq!(client.base_url, "https://ggid.example.com");
}

#[test]
fn test_builder_missing_url() {
    let result = GGIDClient::builder()
        .tenant_id("t1")
        .build();
    assert!(result.is_err());
}

#[test]
fn test_authorize_url() {
    let client = GGIDClient::new("https://ggid.iot2.win", "tenant-uuid");
    let url = client.get_authorize_url(
        "client-123",
        "https://app.example.com/callback",
        Some("openid profile"),
        Some("xyz"),
    );
    assert!(url.contains("response_type=code"));
    assert!(url.contains("client_id=client-123"));
    assert!(url.contains("redirect_uri="));
    assert!(url.contains("state=xyz"));
    assert!(url.contains("scope=openid+profile"));
}

#[test]
fn test_authorize_url_no_state() {
    let client = GGIDClient::new("https://ggid.iot2.win", "tenant-uuid");
    let url = client.get_authorize_url(
        "client-123",
        "https://app.example.com/callback",
        None,
        None,
    );
    assert!(url.contains("scope=openid+profile+email"));
    assert!(!url.contains("state="));
}

#[test]
fn test_claims_deserialize() {
    let json = r#"{
        "sub": "user-123",
        "tenant_id": "tenant-456",
        "roles": ["admin", "editor"],
        "scope": "openid profile",
        "exp": 9999999999,
        "iat": 1000000000,
        "iss": "https://ggid.iot2.win"
    }"#;
    let claims: Claims = serde_json::from_str(json).unwrap();
    assert_eq!(claims.sub, "user-123");
    assert_eq!(claims.roles, vec!["admin", "editor"]);
}

#[test]
fn test_token_response_deserialize() {
    let json = r#"{
        "access_token": "atk123",
        "refresh_token": "rtk456",
        "expires_in": 3600,
        "token_type": "Bearer"
    }"#;
    let tr: TokenResponse = serde_json::from_str(json).unwrap();
    assert_eq!(tr.access_token, "atk123");
    assert_eq!(tr.refresh_token, Some("rtk456".into()));
    assert_eq!(tr.expires_in, 3600);
}

#[test]
fn test_role_deserialize() {
    let json = r#"{
        "id": "r1",
        "key": "admin",
        "name": "Administrator",
        "permissions": ["users:read", "users:write"]
    }"#;
    let role: Role = serde_json::from_str(json).unwrap();
    assert_eq!(role.key, "admin");
    assert_eq!(role.permissions.len(), 2);
}

#[test]
fn test_error_display() {
    let err = GGIDError::PermissionDenied("products".into(), "delete".into());
    assert!(err.to_string().contains("products"));
    assert!(err.to_string().contains("delete"));
}

#[test]
fn test_abac_request_serialize() {
    let req = ABACEvalRequest {
        action: "read".into(),
        resource: "documents".into(),
        conditions: vec![ABACCondition {
            field: "department".into(),
            operator: "eq".into(),
            value: serde_json::Value::String("finance".into()),
        }],
    };
    let json = serde_json::to_string(&req).unwrap();
    assert!(json.contains("documents"));
    assert!(json.contains("department"));
}
