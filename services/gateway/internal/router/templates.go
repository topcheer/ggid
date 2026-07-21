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

const hostedDeviceApproveHTML = `<!DOCTYPE html>
<html lang="en">
<head>
  <meta charset="utf-8">
  <meta name="viewport" content="width=device-width, initial-scale=1">
  <title>Authorize Device — GGID</title>
  <style>
    *{margin:0;padding:0;box-sizing:border-box}
    body{font-family:-apple-system,BlinkMacSystemFont,'Segoe UI',Roboto,sans-serif;background:#f0f2f5;display:flex;align-items:center;justify-content:center;min-height:100vh}
    .card{background:#fff;border-radius:12px;box-shadow:0 2px 10px rgba(0,0,0,.08);width:440px;max-width:90vw;padding:36px}
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
    .error{background:#fef2f2;border:1px solid #fecaca;color:#dc2626;padding:10px 12px;border-radius:8px;font-size:13px;margin-bottom:16px;display:none}
    .success{background:#f0fdf4;border:1px solid #bbf7d0;color:#16a34a;padding:10px 12px;border-radius:8px;font-size:13px;margin-bottom:16px;display:none}
    .code-input{text-align:center;font-size:28px;letter-spacing:4px;font-weight:700;text-transform:uppercase}
    .hint{background:#eff6ff;border:1px solid #bfdbfe;color:#1e40af;padding:8px 12px;border-radius:8px;font-size:12px;margin-top:16px;text-align:center}
    .mfa{display:none}
  </style>
</head>
<body>
<div class="card">
  <div class="logo">G<span>GID</span></div>
  <div class="subtitle">Authorize a device or CLI</div>

  <div class="error" id="err"></div>
  <div class="success" id="ok"></div>

  <!-- Login Form (shown if not authenticated) -->
  <div id="login-form">
    <div class="subtitle" style="margin-bottom:20px">Sign in first to approve this device</div>
    <div class="field">
      <label>Username</label>
      <input type="text" id="username" placeholder="admin" autocomplete="username">
    </div>
    <div class="field">
      <label>Password</label>
      <input type="password" id="password" placeholder="Your password" autocomplete="current-password">
    </div>
    <button class="btn btn-primary" id="login-btn" onclick="doLogin()">Sign In</button>
  </div>

  <!-- MFA Form -->
  <div id="mfa-form" style="display:none">
    <div class="field">
      <label>Authentication Code</label>
      <input type="text" id="mfa-code" placeholder="123456" maxlength="6" onkeyup="if(event.keyCode===13)doMfa()">
    </div>
    <button class="btn btn-primary" onclick="doMfa()">Verify</button>
  </div>

  <!-- Device Code Approval (shown after authentication) -->
  <div id="approve-form" style="display:none">
    <div class="hint">Enter the code displayed on your device or CLI</div>
    <div style="height:16px"></div>
    <div class="field">
      <label>Device Code</label>
      <input type="text" id="user-code" class="code-input" placeholder="XXXX-XXXX" maxlength="9" onkeyup="if(event.keyCode===13)doApprove()">
    </div>
    <button class="btn btn-primary" id="approve-btn" onclick="doApprove()">Authorize Device</button>
  </div>
</div>

<script>
const T="00000000-0000-0000-0000-000000000001";
const params=new URLSearchParams(location.search);
var token=localStorage.getItem("ggid_access_token");

function showErr(m){const e=document.getElementById("err");e.textContent=m;e.style.display="block";setTimeout(()=>e.style.display="none",5000)}
function showOk(m){const e=document.getElementById("ok");e.textContent=m;e.style.display="block"}

// Pre-fill code from URL param (verification_uri_complete)
if(params.get("user_code")){
  document.getElementById("user-code").value=params.get("user_code");
}

// Check if already logged in
if(token){
  showApproveForm();
} else {
  document.getElementById("login-form").style.display="block";
}

function showApproveForm(){
  document.getElementById("login-form").style.display="none";
  document.getElementById("mfa-form").style.display="none";
  document.getElementById("approve-form").style.display="block";
}

async function doLogin(){
  const u=document.getElementById("username").value;
  const p=document.getElementById("password").value;
  if(!u||!p){showErr("Please enter username and password");return}
  const btn=document.getElementById("login-btn");
  btn.disabled=true;btn.textContent="Signing in...";
  try{
    const r=await fetch("/api/v1/auth/login",{method:"POST",headers:{"Content-Type":"application/json","X-Tenant-ID":T},body:JSON.stringify({username:u,password:p})});
    const d=await r.json();
    if(!r.ok){showErr(d.error||d.detail||"Login failed");return}
    if(d.mfa_required){
      document.getElementById("login-form").style.display="none";
      document.getElementById("mfa-form").style.display="block";
      window._session=d.session_id;
      return;
    }
    token=d.access_token;
    localStorage.setItem("ggid_access_token",d.access_token);
    showApproveForm();
  }catch(e){showErr("Network error — is the API running?")}
  finally{btn.disabled=false;btn.textContent="Sign In"}
}

async function doMfa(){
  const code=document.getElementById("mfa-code").value;
  try{
    const r=await fetch("/api/v1/auth/mfa/login",{method:"POST",headers:{"Content-Type":"application/json","X-Tenant-ID":T},body:JSON.stringify({session_id:window._session,code:code})});
    const d=await r.json();
    if(!r.ok){showErr(d.error||d.detail||"Invalid code");return}
    token=d.access_token;
    localStorage.setItem("ggid_access_token",d.access_token);
    showApproveForm();
  }catch(e){showErr("Network error")}
}

async function doApprove(){
  const code=document.getElementById("user-code").value.trim();
  if(!code){showErr("Please enter the device code");return}
  const btn=document.getElementById("approve-btn");
  btn.disabled=true;btn.textContent="Authorizing...";
  try{
    // Decode JWT to get user_id for the approve endpoint
    var userId="";
    try{
      const payload=JSON.parse(atob(token.split(".")[1]));
      userId=payload.sub||"";
    }catch(e){}
    const r=await fetch("/api/v1/oauth/device/approve",{
      method:"POST",
      headers:{"Content-Type":"application/x-www-form-urlencoded","X-Tenant-ID":T,"X-User-ID":userId},
      body:"user_code="+encodeURIComponent(code)+"&user_id="+encodeURIComponent(userId)
    });
    const d=await r.json();
    if(!r.ok){showErr(d.error||d.detail||"Approval failed");return}
    showOk("Device authorized! You can close this page.");
    document.getElementById("approve-form").style.display="none";
  }catch(e){showErr("Network error")}
  finally{btn.disabled=false;btn.textContent="Authorize Device"}
}
</script>
</body>
</html>`
