// OAuth demo with fine-grained permission control.
//
// Run: GGID_URL=http://localhost:8080 CLIENT_ID=xxx CLIENT_SECRET=xxx cargo run

use std::env;
use std::net::TcpListener;
use std::io::{Read, Write};
use std::collections::HashMap;

fn main() {
    let ggid_url = env::var("GGID_URL").unwrap_or_else(|_| "http://localhost:8080".into());
    let client_id = env::var("CLIENT_ID").unwrap_or_else(|_| "demo-client".into());
    let client_secret = env::var("CLIENT_SECRET").unwrap_or_else(|_| "demo-secret".into());
    let port = env::var("PORT").unwrap_or_else(|_| "3004"..into());
    let redirect_uri = format!("http://localhost:{}/auth/callback", port);
    let _ = client_secret;

    let addr = format!("0.0.0.0:{}", port);
    let listener = TcpListener::bind(&addr).unwrap();
    println!("Rust OAuth demo on http://{}", addr);

    for stream in listener.incoming() {
        let mut stream = stream.unwrap();
        let mut buf = [0u8; 4096];
        let n = stream.read(&mut buf).unwrap_or(0);
        let req = String::from_utf8_lossy(&buf[..n]);
        let path = req.lines().next().unwrap_or("").split_whitespace().nth(1).unwrap_or("/");

        // Parse cookie for session (simplified: no real session store)
        let has_session = req.lines().any(|l| l.contains("ggid_session="));

        let resp = if path == "/" {
            if has_session { redirect("/dashboard") }
            else { html("<h1>Rust OAuth Demo</h1><p><a href=\"/auth/login\">Login with GGID</a></p>") }
        } else if path == "/auth/login" {
            redirect(&format!("{}/oauth/authorize?response_type=code&client_id={}&redirect_uri={}&scope=openid+profile+email",
                ggid_url, client_id, redirect_uri))
        } else if path.starts_with("/auth/callback") {
            // In production: exchange code for token, decode JWT, set session cookie
            html("<h1>OAuth Callback</h1><p>Token exchange happens server-side. <a href=\"/dashboard\">Continue</a></p>")
        } else if path == "/dashboard" {
            if !has_session { return redirect("/auth/login"); }
            let perms = ["inventory:read", "inventory:write", "orders:read", "orders:write", "admin"];
            let perm_list: String = perms.iter().map(|p| {
                let has = has_permission_demo(p);
                format!("<li style='color:{}'>{} {}</li>", if has {"green"} else {"red"}, if has {"YES"} else {"NO"}, p)
            }).collect::<Vec<_>>().join("");
            html(&format!("<h1>Dashboard</h1><p>Permission demo (all granted in demo mode)</p><ul>{}</ul>{}",
                perm_list, menu()))
        } else if path == "/inventory" {
            html(&format!("<h1>Inventory</h1><table border=1><tr><th>SKU</th><th>Name</th><th>Stock</th></tr><tr><td>SKU-001</td><td>Widget A</td><td>150</td></tr></table>{}", menu()))
        } else if path == "/orders" {
            html(&format!("<h1>Orders</h1><button>New Order</button><table border=1><tr><th>Order#</th><th>Customer</th><th>Status</th></tr><tr><td>ORD-001</td><td>Acme</td><td>Pending</td></tr></table>{}", menu()))
        } else if path == "/admin" {
            html(&format!("<h1>Admin Panel</h1><p>Welcome, administrator.</p>{}", menu()))
        } else {
            "HTTP/1.1 404 Not Found\r\n\r\n<h1>404</h1>".into()
        };
        stream.write_all(resp.as_bytes()).unwrap();
    }
}

fn has_permission_demo(_perm: &str) -> bool { true } // demo: all permissions granted

fn menu() -> &'static str {
    "<hr><div style='padding:8px'><a href=\"/dashboard\">Dashboard</a> | <a href=\"/orders\">Orders</a> | <a href=\"/inventory\">Inventory</a> | <a href=\"/admin\">Admin</a></div>"
}

fn html(body: &str) -> String {
    format!("HTTP/1.1 200 OK\r\nContent-Type: text/html\r\n\r\n<!DOCTYPE html><html><body style='font-family:sans-serif;max-width:800px;margin:40px'>{}</body></html>", body)
}

fn redirect(url: &str) -> String {
    format!("HTTP/1.1 302 Found\r\nLocation: {}\r\n\r\n", url)
}
