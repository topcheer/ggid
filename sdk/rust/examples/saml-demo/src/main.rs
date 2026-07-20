use std::env;
use std::io::Read;

fn main() {
    let ggid_url = env::var("GGID_URL").expect("GGID_URL required");
    let acs_url = env::var("SP_ACS_URL").unwrap_or_else(|_| "http://localhost:9091/acs".to_string());
    let entity_id = env::var("SP_ENTITY_ID").unwrap_or_else(|_| format!("{}/saml", ggid_url));
    let listen = env::var("LISTEN_ADDR").unwrap_or_else(|_| "0.0.0.0:9091".to_string());

    println!("SAML Demo (Rust) on http://{}", listen);
    println!("GGID URL: {}", ggid_url);
    println!("Entity ID: {}", entity_id);
    println!("ACS URL: {}", acs_url);

    let server = tiny_http::Server::http(&listen).unwrap();

    for request in server.incoming_requests() {
        let url = request.url().to_string();
        match url.as_str() {
            "/" => {
                let html = format!(r#"<!DOCTYPE html><html><head><title>SAML Demo (Rust)</title></head>
<body style="font-family:system-ui;max-width:600px;margin:50px auto">
<h1>SAML Demo App (Rust)</h1>
<p>SP-initiated SAML SSO with GGID IAM.</p>
<a href="/login" style="background:#4f46e5;color:white;padding:10px 20px;border-radius:6px;text-decoration:none">Login with GGID SSO</a>
</body></html>"#);
                let _ = request.respond(tiny_http::Response::from_string(html));
            }
            "/login" => {
                let sso_url = format!("{}/saml/sso?relay_state={}", ggid_url, acs_url);
                let _ = request.respond(tiny_http::Response::empty(302)
                    .with_header(tiny_http::Header::from_bytes(&b"Location"[..], sso_url.as_bytes()).unwrap()));
            }
            "/acs" => {
                let mut body = String::new();
                let _ = request.as_reader().read_to_string(&mut body);
                let html = format!(r#"<!DOCTYPE html><html><head><title>SAML ACS</title></head>
<body style="font-family:system-ui;max-width:600px;margin:50px auto">
<h1>SAML Login Success</h1>
<p>Received SAML response at ACS endpoint.</p>
<p>ACS URL: {}</p>
<p>Entity ID: {}</p>
<a href="/">Back to Home</a>
</body></html>"#, acs_url, entity_id);
                let _ = request.respond(tiny_http::Response::from_string(html));
            }
            "/saml/metadata" => {
                let meta = format!(r#"<?xml version="1.0" encoding="UTF-8"?>
<EntityDescriptor xmlns="urn:oasis:names:tc:SAML:2.0:metadata" entityID="{}">
  <SPSSODescriptor protocolSupportEnumeration="urn:oasis:names:tc:SAML:2.0:protocol">
    <NameIDFormat>urn:oasis:names:tc:SAML:1.1:nameid-format:emailAddress</NameIDFormat>
    <AssertionConsumerService Binding="urn:oasis:names:tc:SAML:2.0:bindings:HTTP-POST" Location="{}" index="0"/>
  </SPSSODescriptor>
</EntityDescriptor>"#, entity_id, acs_url);
                let _ = request.respond(tiny_http::Response::from_string(meta)
                    .with_header(tiny_http::Header::from_bytes(&b"Content-Type"[..], &b"application/xml"[..]).unwrap()));
            }
            _ => {
                let _ = request.respond(tiny_http::Response::from_string("404 Not Found").with_status_code(404));
            }
        }
    }
}
