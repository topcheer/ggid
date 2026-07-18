// Package router contains static HTML/JSON templates served by the Gateway.
// Extracting them into a separate file keeps the Go routing logic testable.
package router

const swaggerHTML = `<!DOCTYPE html>
<html lang="en">
<head>
  <meta charset="utf-8">
  <meta name="viewport" content="width=device-width, initial-scale=1">
  <title>GGID API Documentation</title>
  <link rel="stylesheet" href="https://unpkg.com/swagger-ui-dist@5/swagger-ui.css">
  <style>body{margin:0}</style>
</head>
<body>
  <div id="swagger-ui"></div>
  <script src="https://unpkg.com/swagger-ui-dist@5/swagger-ui-bundle.js"></script>
  <script>
    window.onload=function(){
      SwaggerUIBundle({
        url:'/api-docs',
        dom_id:'#swagger-ui',
        deepLinking:true,
        presets:[SwaggerUIBundle.presets.apis],
        layout:'BaseLayout',
        requestInterceptor:function(req){
          req.headers['X-Tenant-ID']='00000000-0000-0000-0000-000000000001';
          return req;
        }
      })
    }
  </script>
</body>
</html>`

const openAPISpec = `{
  "openapi": "3.0.3",
  "info": {"title": "GGID IAM API", "version": "1.0.0", "license": {"name": "Apache 2.0"}},
  "servers": [{"url": "http://localhost:8080"}],
  "components": {
    "securitySchemes": {
      "BearerAuth": {"type": "http", "scheme": "bearer"},
      "TenantID": {"type": "apiKey", "in": "header", "name": "X-Tenant-ID"}
    },
    "schemas": {
      "LoginRequest": {
        "type": "object",
        "required": ["username", "password"],
        "properties": {
          "username": {"type": "string", "example": "admin"},
          "password": {"type": "string", "format": "password", "example": "Admin@123456"}
        }
      },
      "LoginResponse": {
        "type": "object",
        "properties": {
          "access_token": {"type": "string"},
          "refresh_token": {"type": "string"},
          "token_type": {"type": "string", "example": "Bearer"},
          "expires_in": {"type": "integer", "example": 900}
        }
      },
      "RegisterRequest": {
        "type": "object",
        "required": ["username", "email", "password"],
        "properties": {
          "username": {"type": "string", "minLength": 3, "example": "newuser"},
          "email": {"type": "string", "format": "email", "example": "user@example.com"},
          "password": {"type": "string", "format": "password", "minLength": 8, "example": "SecurePass@123"}
        }
      },
      "User": {
        "type": "object",
        "properties": {
          "id": {"type": "string", "format": "uuid"},
          "tenant_id": {"type": "string", "format": "uuid"},
          "username": {"type": "string"},
          "email": {"type": "string", "format": "email"},
          "name": {"type": "string"},
          "status": {"type": "string", "enum": ["active", "suspended", "deleted"]},
          "mfa_enabled": {"type": "boolean"},
          "created_at": {"type": "string", "format": "date-time"},
          "updated_at": {"type": "string", "format": "date-time"}
        }
      },
      "UserList": {
        "type": "object",
        "properties": {
          "users": {"type": "array", "items": {"$ref": "#/components/schemas/User"}},
          "total": {"type": "integer"},
          "limit": {"type": "integer"},
          "offset": {"type": "integer"}
        }
      },
      "Role": {
        "type": "object",
        "properties": {
          "id": {"type": "string", "format": "uuid"},
          "name": {"type": "string", "example": "admin"},
          "key": {"type": "string", "example": "admin"},
          "description": {"type": "string"}
        }
      },
      "AssignRoleRequest": {
        "type": "object",
        "required": ["user_id", "role_id"],
        "properties": {
          "user_id": {"type": "string", "format": "uuid"},
          "role_id": {"type": "string", "format": "uuid"}
        }
      },
      "AuditEvent": {
        "type": "object",
        "properties": {
          "id": {"type": "string", "format": "uuid"},
          "action": {"type": "string", "example": "user.login"},
          "result": {"type": "string", "enum": ["success", "failure", "denied"]},
          "tenant_id": {"type": "string", "format": "uuid"},
          "resource_type": {"type": "string"},
          "resource_id": {"type": "string"},
          "created_at": {"type": "string", "format": "date-time"}
        }
      },
      "TokenRequest": {
        "type": "object",
        "required": ["grant_type"],
        "properties": {
          "grant_type": {"type": "string", "enum": ["authorization_code", "refresh_token", "client_credentials", "urn:ietf:params:oauth:grant-type:token-exchange", "urn:ietf:params:oauth:grant-type:device_code"]},
          "code": {"type": "string"},
          "refresh_token": {"type": "string"},
          "client_id": {"type": "string"},
          "client_secret": {"type": "string", "format": "password"},
          "redirect_uri": {"type": "string"},
          "scope": {"type": "string", "example": "openid profile email"}
        }
      },
      "TokenResponse": {
        "type": "object",
        "properties": {
          "access_token": {"type": "string"},
          "token_type": {"type": "string", "example": "Bearer"},
          "expires_in": {"type": "integer", "example": 900},
          "refresh_token": {"type": "string"},
          "id_token": {"type": "string"},
          "scope": {"type": "string"}
        }
      },
      "AgentScopeUpdate": {
        "type": "object",
        "required": ["scopes"],
        "properties": {
          "scopes": {
            "type": "array",
            "items": {"type": "string"},
            "example": ["users:read", "audit:read"],
            "description": "API scopes the agent is allowed to use. Valid scopes: users:read, users:write, roles:read, roles:write, policies:read, policies:write, audit:read, oauth:read, oauth:admin, agents:read, agents:write, org:read, org:write"
          }
        }
      },
      "Error": {
        "type": "object",
        "properties": {
          "error": {"type": "string"},
          "error_description": {"type": "string"}
        }
      }
    }
  },
  "security": [{"BearerAuth": [], "TenantID": []}],
  "tags": [
    {"name": "Auth", "description": "Authentication endpoints"},
    {"name": "Users", "description": "User management"},
    {"name": "Roles", "description": "Roles and permissions"},
    {"name": "Organizations", "description": "Organization management"},
    {"name": "Audit", "description": "Audit logs"},
    {"name": "OAuth2", "description": "OAuth2/OIDC endpoints"},
    {"name": "Agents", "description": "AI Agent identity and scoping"}
  ],
  "paths": {
    "/api/v1/auth/register": {
      "post": {"tags":["Auth"],"summary":"Register new user","security":[],"requestBody":{"required":true,"content":{"application/json":{"schema":{"$ref":"#/components/schemas/RegisterRequest"}}}},"responses":{"201":{"description":"User created","content":{"application/json":{"schema":{"$ref":"#/components/schemas/User"}}}},"409":{"description":"User already exists","content":{"application/json":{"schema":{"$ref":"#/components/schemas/Error"}}}}}}
    },
    "/api/v1/auth/login": {
      "post": {"tags":["Auth"],"summary":"Authenticate and obtain tokens","security":[],"requestBody":{"required":true,"content":{"application/json":{"schema":{"$ref":"#/components/schemas/LoginRequest"}}}},"responses":{"200":{"description":"Authentication successful","content":{"application/json":{"schema":{"$ref":"#/components/schemas/LoginResponse"}}}},"401":{"description":"Invalid credentials","content":{"application/json":{"schema":{"$ref":"#/components/schemas/Error"}}}}}}
    },
    "/api/v1/auth/refresh": {
      "post": {"tags":["Auth"],"summary":"Refresh access token","security":[],"requestBody":{"content":{"application/json":{"schema":{"type":"object","properties":{"refresh_token":{"type":"string"}}}}}},"responses":{"200":{"description":"New token pair","content":{"application/json":{"schema":{"$ref":"#/components/schemas/LoginResponse"}}}},"401":{"description":"Invalid refresh token"}}}
    },
    "/api/v1/auth/password/forgot": {
      "post": {"tags":["Auth"],"summary":"Request password reset","security":[],"requestBody":{"content":{"application/json":{"schema":{"type":"object","required":["email"],"properties":{"email":{"type":"string","format":"email"}}}}}},"responses":{"200":{"description":"Reset email sent if account exists"}}}
    },
    "/api/v1/auth/password/reset": {
      "post": {"tags":["Auth"],"summary":"Reset password with token","security":[],"requestBody":{"content":{"application/json":{"schema":{"type":"object","required":["token","password"],"properties":{"token":{"type":"string"},"password":{"type":"string","format":"password","minLength":8}}}}}},"responses":{"200":{"description":"Password reset"},"400":{"description":"Invalid token"}}}
    },
    "/api/v1/auth/social/{provider}": {
      "get": {"tags":["Auth"],"summary":"Begin social login","security":[],"parameters":[{"name":"provider","in":"path","required":true,"schema":{"type":"string","enum":["google","github","oidc"]}}],"responses":{"200":{"description":"Auth URL"}}}
    },
    "/api/v1/users": {
      "get": {"tags":["Users"],"summary":"List users","parameters":[{"name":"limit","in":"query","schema":{"type":"integer","default":50,"maximum":100}},{"name":"offset","in":"query","schema":{"type":"integer","default":0}},{"name":"status","in":"query","schema":{"type":"string","enum":["active","suspended","deleted"]}}],"responses":{"200":{"description":"User list","content":{"application/json":{"schema":{"$ref":"#/components/schemas/UserList"}}}}},"post": {"tags":["Users"],"summary":"Create user","requestBody":{"required":true,"content":{"application/json":{"schema":{"$ref":"#/components/schemas/RegisterRequest"}}}},"responses":{"201":{"description":"Created","content":{"application/json":{"schema":{"$ref":"#/components/schemas/User"}}}},"409":{"description":"Conflict"}}}
    },
    "/api/v1/users/{id}": {
      "get": {"tags":["Users"],"summary":"Get user by ID","parameters":[{"name":"id","in":"path","required":true,"schema":{"type":"string","format":"uuid"}}],"responses":{"200":{"description":"OK","content":{"application/json":{"schema":{"$ref":"#/components/schemas/User"}}}},"404":{"description":"Not found"}}},
      "put": {"tags":["Users"],"summary":"Update user","parameters":[{"name":"id","in":"path","required":true,"schema":{"type":"string","format":"uuid"}}],"requestBody":{"content":{"application/json":{"schema":{"type":"object","properties":{"name":{"type":"string"},"status":{"type":"string","enum":["active","suspended"]}}}}}},"responses":{"200":{"description":"Updated","content":{"application/json":{"schema":{"$ref":"#/components/schemas/User"}}}}}},
      "delete": {"tags":["Users"],"summary":"Delete user","parameters":[{"name":"id","in":"path","required":true,"schema":{"type":"string","format":"uuid"}}],"responses":{"204":{"description":"Deleted"}}}
    },
    "/api/v1/roles": {
      "get": {"tags":["Roles"],"summary":"List roles","responses":{"200":{"description":"OK","content":{"application/json":{"schema":{"type":"object","properties":{"roles":{"type":"array","items":{"$ref":"#/components/schemas/Role"}}}}}}}},"post": {"tags":["Roles"],"summary":"Create role","requestBody":{"required":true,"content":{"application/json":{"schema":{"$ref":"#/components/schemas/Role"}}}},"responses":{"201":{"description":"Created","content":{"application/json":{"schema":{"$ref":"#/components/schemas/Role"}}}}}}
    },
    "/api/v1/roles/assign": {
      "post": {"tags":["Roles"],"summary":"Assign role to user","requestBody":{"required":true,"content":{"application/json":{"schema":{"$ref":"#/components/schemas/AssignRoleRequest"}}}},"responses":{"200":{"description":"Assigned","content":{"application/json":{"schema":{"type":"object","properties":{"status":{"type":"string","example":"assigned"}}}}}},"403":{"description":"Admin required"}}}
    },
    "/api/v1/orgs": {
      "get": {"tags":["Organizations"],"summary":"List organizations","responses":{"200":{"description":"OK"}}},"post": {"tags":["Organizations"],"summary":"Create organization","requestBody":{"content":{"application/json":{"schema":{"type":"object","properties":{"name":{"type":"string"},"slug":{"type":"string"}}}}}},"responses":{"201":{"description":"Created"}}}
    },
    "/api/v1/orgs/{id}/tree": {
      "get": {"tags":["Organizations"],"summary":"Get org tree","parameters":[{"name":"id","in":"path","required":true,"schema":{"type":"string","format":"uuid"}}],"responses":{"200":{"description":"OK"}}}
    },
    "/api/v1/audit": {
      "get": {"tags":["Audit"],"summary":"Query audit events","parameters":[{"name":"limit","in":"query","schema":{"type":"integer","default":50,"maximum":100}},{"name":"offset","in":"query","schema":{"type":"integer","default":0}},{"name":"action","in":"query","schema":{"type":"string","example":"user.login"}},{"name":"start_time","in":"query","schema":{"type":"string","format":"date-time"}},{"name":"end_time","in":"query","schema":{"type":"string","format":"date-time"}}],"responses":{"200":{"description":"OK","content":{"application/json":{"schema":{"type":"object","properties":{"events":{"type":"array","items":{"$ref":"#/components/schemas/AuditEvent"}},"total":{"type":"integer"}}}}}}}}
    },
    "/oauth/authorize": {
      "get": {"tags":["OAuth2"],"summary":"OAuth2 authorize","security":[],"parameters":[{"name":"client_id","in":"query","required":true,"schema":{"type":"string"}},{"name":"redirect_uri","in":"query","required":true,"schema":{"type":"string"}},{"name":"response_type","in":"query","required":true,"schema":{"type":"string"}},{"name":"scope","in":"query","schema":{"type":"string"}},{"name":"state","in":"query","schema":{"type":"string"}}],"responses":{"302":{"description":"Redirect with code"}}}
    },
    "/oauth/token": {
      "post": {"tags":["OAuth2"],"summary":"OAuth2 token exchange","security":[],"requestBody":{"required":true,"content":{"application/x-www-form-urlencoded":{"schema":{"$ref":"#/components/schemas/TokenRequest"}}}},"responses":{"200":{"description":"Token response","content":{"application/json":{"schema":{"$ref":"#/components/schemas/TokenResponse"}}}},"400":{"description":"Invalid request","content":{"application/json":{"schema":{"$ref":"#/components/schemas/Error"}}}},"401":{"description":"Client authentication failed"}}}
    },
    "/api/v1/agents/{id}/scopes": {
      "get": {"tags":["Agents"],"summary":"Get agent permission scopes","parameters":[{"name":"id","in":"path","required":true,"schema":{"type":"string","format":"uuid"}}],"responses":{"200":{"description":"Agent scopes","content":{"application/json":{"schema":{"type":"object","properties":{"agent_id":{"type":"string"},"scopes":{"type":"array","items":{"type":"string"}},"available":{"type":"array","items":{"type":"string"}}}}}}}}},
      "post": {"tags":["Agents"],"summary":"Set agent permission scopes","parameters":[{"name":"id","in":"path","required":true,"schema":{"type":"string","format":"uuid"}}],"requestBody":{"required":true,"content":{"application/json":{"schema":{"$ref":"#/components/schemas/AgentScopeUpdate"}}}},"responses":{"200":{"description":"Scopes updated","content":{"application/json":{"schema":{"type":"object","properties":{"status":{"type":"string","example":"updated"},"agent_id":{"type":"string"},"scopes":{"type":"array","items":{"type":"string"}}}}}}},"404":{"description":"Agent not found"}}}
    },
    "/oauth/.well-known/openid-configuration": {
      "get": {"tags":["OAuth2"],"summary":"OIDC discovery","security":[],"responses":{"200":{"description":"Discovery document"}}}
    },
    "/healthz": {
      "get": {"tags":["Auth"],"summary":"Health check","security":[],"responses":{"200":{"description":"OK"}}}
    }
  }
}`

const hostedLoginHTML = `<!DOCTYPE html>
<html lang="en">
<head>
  <meta charset="utf-8">
  <meta name="viewport" content="width=device-width, initial-scale=1">
  <title>Sign In — GGID</title>
  <style>
    *{margin:0;padding:0;box-sizing:border-box}
    body{font-family:-apple-system,BlinkMacSystemFont,'Segoe UI',Roboto,sans-serif;background:#f0f2f5;display:flex;align-items:center;justify-content:center;min-height:100vh}
    .card{background:#fff;border-radius:12px;box-shadow:0 2px 10px rgba(0,0,0,.08);width:400px;max-width:90vw;padding:36px}
    .logo{font-size:28px;font-weight:800;text-align:center;margin-bottom:6px;color:#1a1a1a}
    .logo span{color:#4f46e5}
    .subtitle{text-align:center;color:#6b7280;font-size:14px;margin-bottom:24px}
    .field{margin-bottom:16px}
    label{display:block;font-size:13px;font-weight:600;color:#374151;margin-bottom:5px}
    input{width:100%;padding:10px 12px;border:1px solid #d1d5db;border-radius:8px;font-size:14px;outline:none;transition:border-color .2s}
    input:focus{border-color:#4f46e5;box-shadow:0 0 0 3px rgba(79,70,229,.1)}
    .btn{width:100%;padding:11px;border:none;border-radius:8px;font-size:14px;font-weight:600;cursor:pointer;transition:background .2s}
    .btn-primary{background:#4f46e5;color:#fff}
    .btn-primary:hover{background:#4338ca}
    .btn-primary:disabled{opacity:.5;cursor:not-allowed}
    .divider{display:flex;align-items:center;margin:20px 0;color:#9ca3af;font-size:12px}
    .divider::before,.divider::after{content:'';flex:1;height:1px;background:#e5e7eb}
    .divider span{padding:0 12px}
    .social{display:grid;grid-template-columns:1fr 1fr 1fr;gap:8px}
    .btn-social{padding:10px;border:1px solid #d1d5db;border-radius:8px;background:#fff;font-size:13px;font-weight:500;cursor:pointer;text-align:center;transition:background .2s}
    .btn-social:hover{background:#f9fafb}
    .error{background:#fef2f2;border:1px solid #fecaca;color:#dc2626;padding:10px 12px;border-radius:8px;font-size:13px;margin-bottom:16px;display:none}
    .success{background:#f0fdf4;border:1px solid #bbf7d0;color:#16a34a;padding:10px 12px;border-radius:8px;font-size:13px;margin-bottom:16px;display:none}
    .footer{text-align:center;margin-top:20px;font-size:12px;color:#9ca3af}
    .footer a{color:#4f46e5;text-decoration:none}
    .mfa{display:none}
    .hint{background:#eff6ff;border:1px solid #bfdbfe;color:#1e40af;padding:8px 12px;border-radius:8px;font-size:12px;margin-top:16px;text-align:center}
  </style>
</head>
<body>
<div class="card">
  <div class="logo">G<span>GID</span></div>
  <div class="subtitle" id="subtitle">Sign in to your account</div>

  <div class="error" id="err"></div>
  <div class="success" id="ok"></div>

  <!-- Login Form -->
  <div id="login-form">
    <div class="field">
      <label>Username</label>
      <input type="text" id="username" placeholder="admin" autocomplete="username">
    </div>
    <div class="field">
      <label>Password</label>
      <input type="password" id="password" placeholder="Your password" autocomplete="current-password">
    </div>
    <button class="btn btn-primary" id="login-btn" onclick="doLogin()">Sign In</button>

    <div class="divider"><span>or continue with</span></div>
    <div class="social">
      <button class="btn-social" onclick="socialLogin('google')">Google</button>
      <button class="btn-social" onclick="socialLogin('github')">GitHub</button>
      <button class="btn-social" onclick="socialLogin('oidc')">SSO</button>
    </div>

    <div class="hint">Use the credentials you set up during registration</div>
    <div class="footer">
      <a href="/forgot-password">Forgot password?</a> ·
      <a href="/register">Create account</a>
    </div>
  </div>

  <!-- MFA Form -->
  <div id="mfa-form" style="display:none">
    <div class="field">
      <label>Authentication Code</label>
      <input type="text" id="mfa-code" placeholder="123456" maxlength="6" onkeyup="if(event.keyCode===13)doMfa()">
    </div>
    <button class="btn btn-primary" onclick="doMfa()">Verify</button>
  </div>
</div>

<script>
const T="00000000-0000-0000-0000-000000000001";
const params=new URLSearchParams(location.search);
const redirectUri=params.get("redirect_uri")||"/";

function showErr(m){const e=document.getElementById("err");e.textContent=m;e.style.display="block";setTimeout(()=>e.style.display="none",5000)}
function showOk(m){const e=document.getElementById("ok");e.textContent=m;e.style.display="block"}

async function doLogin(){
  const u=document.getElementById("username").value;
  const p=document.getElementById("password").value;
  if(!u||!p){showErr("Please enter username and password");return}
  const btn=document.getElementById("login-btn");
  btn.disabled=true;btn.textContent="Signing in...";
  try{
    const r=await fetch("/api/v1/auth/login",{method:"POST",headers:{"Content-Type":"application/json","X-Tenant-ID":T},body:JSON.stringify({username:u,password:p})});
    const d=await r.json();
    if(!r.ok){showErr(d.error||"Login failed");return}
    if(d.mfa_required){document.getElementById("login-form").style.display="none";document.getElementById("mfa-form").style.display="block";document.getElementById("subtitle").textContent="Enter your verification code";window._session=d.session_id;return}
    localStorage.setItem("ggid_access_token",d.access_token);
    localStorage.setItem("ggid_refresh_token",d.refresh_token);
    showOk("Success! Redirecting...");
    setTimeout(()=>window.location.href=redirectUri,500);
  }catch(e){showErr("Network error — is the API running?")}
  finally{btn.disabled=false;btn.textContent="Sign In"}
}

async function doMfa(){
  const code=document.getElementById("mfa-code").value;
  try{
    const r=await fetch("/api/v1/auth/mfa/login",{method:"POST",headers:{"Content-Type":"application/json","X-Tenant-ID":T},body:JSON.stringify({session_id:window._session,code:code})});
    const d=await r.json();
    if(!r.ok){showErr(d.error||"Invalid code");return}
    localStorage.setItem("ggid_access_token",d.access_token);
    showOk("Verified! Redirecting...");
    setTimeout(()=>window.location.href=redirectUri,500);
  }catch(e){showErr("Network error")}
}

function socialLogin(p){
  fetch("/api/v1/auth/social/"+p+"?redirect_uri="+encodeURIComponent(redirectUri),{headers:{"X-Tenant-ID":T}})
    .then(r=>r.json()).then(d=>{if(d.auth_url)window.location.href=d.auth_url}).catch(()=>showErr(p+" login not configured"));
}
</script>
</body>
</html>`

const hostedRegisterHTML = `<!DOCTYPE html>
<html lang="en"><head><meta charset="utf-8"><meta name="viewport" content="width=device-width,initial-scale=1">
<title>Create Account — GGID</title>
<style>*{margin:0;padding:0;box-sizing:border-box}body{font-family:-apple-system,BlinkMacSystemFont,sans-serif;background:#f0f2f5;display:flex;align-items:center;justify-content:center;min-height:100vh}.card{background:#fff;border-radius:12px;box-shadow:0 2px 10px rgba(0,0,0,.08);width:400px;max-width:90vw;padding:36px}.logo{font-size:28px;font-weight:800;text-align:center;margin-bottom:6px}.logo span{color:#4f46e5}.sub{text-align:center;color:#6b7280;font-size:14px;margin-bottom:24px}.field{margin-bottom:16px}label{display:block;font-size:13px;font-weight:600;margin-bottom:5px}input{width:100%;padding:10px 12px;border:1px solid #d1d5db;border-radius:8px;font-size:14px}.btn{width:100%;padding:11px;border:none;border-radius:8px;background:#4f46e5;color:#fff;font-size:14px;font-weight:600;cursor:pointer}.btn:hover{background:#4338ca}.err{background:#fef2f2;border:1px solid #fecaca;color:#dc2626;padding:10px;border-radius:8px;font-size:13px;margin-bottom:16px;display:none}.ok{background:#f0fdf4;border:1px solid #bbf7d0;color:#16a34a;padding:10px;border-radius:8px;font-size:13px;margin-bottom:16px;display:none}.footer{text-align:center;margin-top:20px;font-size:12px;color:#9ca3af}.footer a{color:#4f46e5;text-decoration:none}</style>
</head><body><div class="card">
<div class="logo">G<span>GID</span></div><div class="sub">Create your account</div>
<div class="err" id="err"></div><div class="ok" id="ok"></div>
<div class="field"><label>Username</label><input type="text" id="username" placeholder="Choose a username"></div>
<div class="field"><label>Email</label><input type="email" id="email" placeholder="you@example.com"></div>
<div class="field"><label>Password</label><input type="password" id="password" placeholder="At least 8 characters"></div>
<div class="field"><label>Full Name</label><input type="text" id="name" placeholder="Your name"></div>
<button class="btn" id="btn" onclick="doRegister()">Create Account</button>
<div class="footer">Already have an account? <a href="/login">Sign in</a></div>
</div><script>
const T="00000000-0000-0000-0000-000000000001";
function showErr(m){const e=document.getElementById("err");e.textContent=m;e.style.display="block";setTimeout(()=>e.style.display="none",5000)}
function showOk(m){const e=document.getElementById("ok");e.textContent=m;e.style.display="block"}
async function doRegister(){
  const u=document.getElementById("username").value,p=document.getElementById("password").value,e=document.getElementById("email").value,n=document.getElementById("name").value;
  if(!u||!p||!e){showErr("Username, email, and password are required");return}
  const btn=document.getElementById("btn");btn.disabled=true;btn.textContent="Creating...";
  try{
    const r=await fetch("/api/v1/auth/register",{method:"POST",headers:{"Content-Type":"application/json","X-Tenant-ID":T},body:JSON.stringify({username:u,email:e,password:p,name:n})});
    const d=await r.json();
    if(!r.ok){showErr(d.error||"Registration failed");return}
    showOk("Account created! Redirecting to login...");
    setTimeout(()=>window.location.href="/login",1500);
  }catch(err){showErr("Network error")}
  finally{btn.disabled=false;btn.textContent="Create Account"}
}
</script></body></html>`

const hostedForgotPasswordHTML = `<!DOCTYPE html>
<html lang="en"><head><meta charset="utf-8"><meta name="viewport" content="width=device-width,initial-scale=1">
<title>Reset Password — GGID</title>
<style>*{margin:0;padding:0;box-sizing:border-box}body{font-family:-apple-system,BlinkMacSystemFont,sans-serif;background:#f0f2f5;display:flex;align-items:center;justify-content:center;min-height:100vh}.card{background:#fff;border-radius:12px;box-shadow:0 2px 10px rgba(0,0,0,.08);width:400px;max-width:90vw;padding:36px}.logo{font-size:28px;font-weight:800;text-align:center;margin-bottom:6px}.logo span{color:#4f46e5}.sub{text-align:center;color:#6b7280;font-size:14px;margin-bottom:24px}.field{margin-bottom:16px}label{display:block;font-size:13px;font-weight:600;margin-bottom:5px}input{width:100%;padding:10px 12px;border:1px solid #d1d5db;border-radius:8px;font-size:14px}.btn{width:100%;padding:11px;border:none;border-radius:8px;background:#4f46e5;color:#fff;font-size:14px;font-weight:600;cursor:pointer}.ok{background:#f0fdf4;border:1px solid #bbf7d0;color:#16a34a;padding:10px;border-radius:8px;font-size:13px;margin-bottom:16px;display:none}.footer{text-align:center;margin-top:20px;font-size:12px;color:#9ca3af}.footer a{color:#4f46e5;text-decoration:none}</style>
</head><body><div class="card">
<div class="logo">G<span>GID</span></div><div class="sub">Reset your password</div>
<div class="ok" id="ok"></div>
<div class="field"><label>Email or Username</label><input type="text" id="email" placeholder="you@example.com"></div>
<button class="btn" onclick="doReset()">Send Reset Link</button>
<div class="footer"><a href="/login">Back to sign in</a></div>
</div><script>
const T="00000000-0000-0000-0000-000000000001";
function showOk(m){const e=document.getElementById("ok");e.textContent=m;e.style.display="block"}
async function doReset(){
  const e=document.getElementById("email").value;
  if(!e)return;
  try{
    await fetch("/api/v1/auth/password/forgot",{method:"POST",headers:{"Content-Type":"application/json","X-Tenant-ID":T},body:JSON.stringify({email:e})});
  }catch(err){}
  showOk("If that email exists, a reset link has been sent.");
}
</script></body></html>`
