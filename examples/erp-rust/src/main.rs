# Cross-Board ERP Demo — Rust implementation
// Tests all GGID core features via Rust SDK
// Run: GGID_URL=http://localhost:8080 cargo run

use axum::{
    extract::{Path, State},
    http::{HeaderMap, StatusCode},
    response::Json,
    routing::{get, post, put, delete},
    Router,
};
use serde::{Deserialize, Serialize};
use serde_json::{json, Value};
use std::collections::HashMap;
use std::sync::Arc;
use std::time::{SystemTime, UNIX_EPOCH};
use tokio::sync::RwLock;

// --- GGID config ---
const GGID_URL: &str = "http://localhost:8080";
const TENANT_ID: &str = "00000000-0000-0000-0000-000000000001";

// --- State ---
type Store = Arc<RwLock<AppState>>;

struct AppState {
    products: HashMap<String, Product>,
    orders: HashMap<String, Order>,
    audit_log: Vec<AuditEntry>,
    product_seq: u32,
    order_seq: u32,
    ggid_client: ggid_sdk::GGIDClient,
}

#[derive(Clone, Serialize, Deserialize)]
struct Product {
    id: String,
    name: String,
    sku: String,
    price: f64,
    stock: u32,
    category: String,
}

#[derive(Clone, Serialize, Deserialize)]
struct Order {
    id: String,
    customer: String,
    quantity: u32,
    amount: f64,
    status: String,
    created_by: String,
}

#[derive(Clone, Serialize)]
struct AuditEntry {
    id: String,
    action: String,
    resource: String,
    result: String,
    actor_id: String,
}

// --- Auth context ---
struct AuthContext {
    user_id: String,
    permissions: Vec<String>,
    scopes: Vec<String>,
}

async fn extract_auth(headers: &HeaderMap, client: &ggid_sdk::GGIDClient) -> Option<AuthContext> {
    let auth = headers.get("authorization")?.to_str().ok()?;
    let token = auth.strip_prefix("Bearer ")?;
    let claims = client.verify_token(token).await.ok()?;
    Some(AuthContext {
        user_id: claims.sub.clone(),
        permissions: claims.permissions.clone(),
        scopes: claims.scope.split_whitespace().map(String::from).collect(),
    })
}

fn check_perm(auth: &AuthContext, perm: &str) -> bool {
    auth.permissions.iter().any(|p| p == perm || p == "admin")
        || auth.scopes.iter().any(|s| s == perm || s == "admin")
}

fn now_str() -> String {
    format!("{:?}", SystemTime::now().duration_since(UNIX_EPOCH).unwrap().as_secs())
}

// --- Handlers ---

async fn list_inventory(State(state): State<Store>, headers: HeaderMap) -> Result<Json<Value>, StatusCode> {
    let auth = extract_auth(&headers, &state.read().await.ggid_client).await.ok_or(StatusCode::UNAUTHORIZED)?;
    if !check_perm(&auth, "inventory:read") { return Err(StatusCode::FORBIDDEN); }
    let s = state.read().await;
    Ok(Json(json!({ "items": s.products.values().collect::<Vec<_>>(), "total": s.products.len() })))
}

async fn create_inventory(State(state): State<Store>, headers: HeaderMap, body: Json<Product>) -> Result<Json<Value>, StatusCode> {
    let auth = extract_auth(&headers, &state.read().await.ggid_client).await.ok_or(StatusCode::UNAUTHORIZED)?;
    if !check_perm(&auth, "inventory:write") { return Err(StatusCode::FORBIDDEN); }
    let mut s = state.write().await;
    s.product_seq += 1;
    let id = format!("PROD-{:04}", s.product_seq);
    let mut product = body.0;
    product.id = id.clone();
    s.products.insert(id.clone(), product.clone());
    s.audit_log.push(AuditEntry { id: format!("AUD-{}", s.audit_log.len()+1), action: "inventory.create".into(), resource: "product".into(), result: "success".into(), actor_id: auth.user_id });
    Ok((StatusCode::CREATED, Json(json!(product))).into())
}

async fn list_orders(State(state): State<Store>, headers: HeaderMap) -> Result<Json<Value>, StatusCode> {
    let auth = extract_auth(&headers, &state.read().await.ggid_client).await.ok_or(StatusCode::UNAUTHORIZED)?;
    if !check_perm(&auth, "orders:read") { return Err(StatusCode::FORBIDDEN); }
    let show_all = check_perm(&auth, "orders:read:all");
    let uid = auth.user_id.clone();
    let s = state.read().await;
    let list: Vec<&Order> = s.orders.values().filter(|o| show_all || o.created_by == uid).collect();
    Ok(Json(json!({ "items": list, "total": list.len() })))
}

async fn create_order(State(state): State<Store>, headers: HeaderMap, body: Json<Order>) -> Result<Json<Value>, StatusCode> {
    let auth = extract_auth(&headers, &state.read().await.ggid_client).await.ok_or(StatusCode::UNAUTHORIZED)?;
    if !check_perm(&auth, "orders:write") { return Err(StatusCode::FORBIDDEN); }
    let mut s = state.write().await;
    s.order_seq += 1;
    let id = format!("ORD-{:04}", s.order_seq);
    let mut order = body.0;
    order.id = id.clone();
    order.status = "pending".into();
    order.created_by = auth.user_id.clone();
    s.orders.insert(id.clone(), order.clone());
    s.audit_log.push(AuditEntry { id: format!("AUD-{}", s.audit_log.len()+1), action: "orders.create".into(), resource: "order".into(), result: "success".into(), actor_id: auth.user_id });
    Ok((StatusCode::CREATED, Json(json!(order))).into())
}

async fn approve_order(State(state): State<Store>, headers: HeaderMap, Path(id): Path<String>) -> Result<Json<Value>, StatusCode> {
    let auth = extract_auth(&headers, &state.read().await.ggid_client).await.ok_or(StatusCode::UNAUTHORIZED)?;
    if !check_perm(&auth, "orders:approve") { return Err(StatusCode::FORBIDDEN); }
    let mut s = state.write().await;
    let order = s.orders.get_mut(&id).ok_or(StatusCode::NOT_FOUND)?;
    order.status = "approved".into();
    Ok(Json(json!(order)))
}

async fn get_audit(State(state): State<Store>, headers: HeaderMap) -> Result<Json<Value>, StatusCode> {
    let auth = extract_auth(&headers, &state.read().await.ggid_client).await.ok_or(StatusCode::UNAUTHORIZED)?;
    if !check_perm(&auth, "audit:read") { return Err(StatusCode::FORBIDDEN); }
    let s = state.read().await;
    Ok(Json(json!({ "items": s.audit_log, "total": s.audit_log.len() })))
}

async fn dashboard(State(state): State<Store>, headers: HeaderMap) -> Result<Json<Value>, StatusCode> {
    let auth = extract_auth(&headers, &state.read().await.ggid_client).await.ok_or(StatusCode::UNAUTHORIZED)?;
    if !check_perm(&auth, "dashboard:read") { return Err(StatusCode::FORBIDDEN); }
    let s = state.read().await;
    let pending = s.orders.values().filter(|o| o.status == "pending").count();
    let approved = s.orders.values().filter(|o| o.status == "approved").count();
    Ok(Json(json!({ "products": s.products.len(), "orders": s.orders.len(), "pending": pending, "approved": approved })))
}

async fn health() -> Json<Value> {
    Json(json!({ "status": "ok" }))
}

#[tokio::main]
async fn main() {
    let state: Store = Arc::new(RwLock::new(AppState {
        products: HashMap::new(), orders: HashMap::new(), audit_log: Vec::new(),
        product_seq: 0, order_seq: 0,
        ggid_client: ggid_sdk::GGIDClient::new(GGID_URL, TENANT_ID),
    }));

    let app = Router::new()
        .route("/health", get(health))
        .route("/api/inventory", get(list_inventory).post(create_inventory))
        .route("/api/orders", get(list_orders).post(create_order))
        .route("/api/orders/:id/approve", put(approve_order))
        .route("/api/audit", get(get_audit))
        .route("/api/dashboard", get(dashboard))
        .with_state(state);

    let listener = tokio::net::TcpListener::bind("0.0.0.0:9092").await.unwrap();
    println!("ERP Rust Demo on :9092");
    axum::serve(listener, app).await.unwrap();
}
