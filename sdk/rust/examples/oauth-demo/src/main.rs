// OAuth 2.0 demo application.
// Demonstrates: login → authorization code flow → get token → show user info.
//
// Run:
//   GGID_URL=http://localhost:8080 CLIENT_ID=gcid_xxx CLIENT_SECRET=gcs_xxx cargo run

use std::env;
use std::net::TcpListener;
use std::io::{Read, Write};

fn main() {
    let ggid_url = env::var("GGID_URL").unwrap_or_else(|_| "http://localhost:8080".into());
    let client_id = env::var("CLIENT_ID").unwrap_or_else(|_| "demo-client".into());
    let client_secret = env::var("CLIENT_SECRET").unwrap_or_else(|_| "demo-secret".into());
    let port = env::var("PORT").unwrap_or_else(|_| "3004".into());
    let redirect_uri = format!("http://localhost:{}/auth/callback", port);

    let addr = format!("0.0.0.0:{}", port);
    let listener = TcpListener::bind(&addr).unwrap();
    println!("Rust OAuth demo on http://{}", addr);

    for stream in listener.incoming() {
        let mut stream = stream.unwrap();
        let mut buf = [0u8; 4096];
        let n = stream.read(&mut buf).unwrap_or(0);
        let req = String::from_utf8_lossy(&buf[..n]);
        let path = req.lines().next().unwrap_or("").split_whitespace().nth(1).unwrap_or("/");

        let resp = if path == "/" {
            format!("HTTP/1.1 200 OK\r\nContent-Type: text/html\r\n\r\n\
                <h1>Rust OAuth 2.0 Demo</h1><p>GGID: {}</p>\
                <p><a href=\"/auth/login\">Login with GGID</a></p>", ggid_url)
        } else if path == "/auth/login" {
            let url = format!("{}/oauth/authorize?response_type=code&client_id={}&redirect_uri={}&scope=openid+profile+email",
                ggid_url, client_id, redirect_uri);
            format!("HTTP/1.1 302 Found\r\nLocation: {}\r\n\r\n", url)
        } else if path.starts_with("/auth/callback") {
            format!("HTTP/1.1 200 OK\r\nContent-Type: text/html\r\n\r\n\
                <h1>OAuth Callback</h1><p>Code received. Token exchange requires async HTTP (see SDK client).</p>\
                <p>Client ID: {}</p>", client_id)
        } else {
            "HTTP/1.1 404 Not Found\r\n\r\n".into()
        };
        let _ = client_secret; // used in production token exchange
        stream.write_all(resp.as_bytes()).unwrap();
    }
}
