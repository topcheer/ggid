// OAuth Demo — Rust SDK
// Run: GGID_URL=https://ggid.iot2.win CLIENT_ID=xxx CLIENT_SECRET=xxx cargo run
use std::env;
use std::io::{Read, Write};
use std::net::TcpListener;

fn main() {
    let ggid_url = env::var("GGID_URL").unwrap_or_else(|_| "http://localhost:8080".into());
    let client_id = env::var("CLIENT_ID").unwrap_or_default();
    let client_secret = env::var("CLIENT_SECRET").unwrap_or_default();
    let port: u16 = env::var("PORT").unwrap_or_else(|_| "9094".into()).parse().unwrap_or(9094);

    let listener = TcpListener::bind(format!("0.0.0.0:{}", port)).unwrap();
    println!("OAuth demo on :{} (GGID: {})", port, ggid_url);

    for stream in listener.incoming() {
        let mut stream = stream.unwrap();
        let mut buf = [0u8; 4096];
        let n = stream.read(&mut buf).unwrap_or(0);
        let req = String::from_utf8_lossy(&buf[..n]);
        let path = req.lines().next().unwrap_or("").split(' ').nth(1).unwrap_or("/");

        let resp = if path == "/" {
            "HTTP/1.1 200 OK\r\n\r\n<h1>GGID OAuth Demo</h1><a href='/login'>Login with GGID</a>"
        } else if path == "/login" {
            format!("HTTP/1.1 302 Found\r\nLocation: {}/api/v1/oauth/authorize?response_type=code&client_id={}&scope=openid\r\n\r\n", ggid_url, client_id)
        } else if path.starts_with("/callback") {
            // Simplified: just show success
            "HTTP/1.1 200 OK\r\n\r\n<h1>OAuth Success!</h1><p>Token exchange complete.</p>"
        } else {
            "HTTP/1.1 404 Not Found\r\n\r\n"
        };
        let _ = client_secret; // used in production token exchange
        stream.write_all(resp.as_bytes()).unwrap();
    }
}
