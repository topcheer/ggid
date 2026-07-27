package server

import (
	"fmt"
	"net/http"

	"github.com/google/uuid"
)

// renderDynamicLoginPage generates an HTML login page based on the client's
// configured auth methods. Shows only the forms for enabled methods.
func renderDynamicLoginPage(w http.ResponseWriter, r *http.Request, tenantID uuid.UUID, authMethods []string, apiBase string) {
	hasPassword := contains(authMethods, "password")
	hasPasskey := contains(authMethods, "passkey")
	hasSMS := contains(authMethods, "sms_otp")
	hasEmail := contains(authMethods, "email_otp")

	// Default to password if nothing configured
	if !hasPassword && !hasPasskey && !hasSMS && !hasEmail {
		hasPassword = true
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(http.StatusOK)

	// Build tab buttons
	var tabsHTML string
	tabIdx := 0
	if hasPassword {
		tabsHTML += tabBtn(tabIdx, "密码登录"); tabIdx++
	}
	if hasPasskey {
		tabsHTML += tabBtn(tabIdx, "Passkey"); tabIdx++
	}
	if hasSMS {
		tabsHTML += tabBtn(tabIdx, "短信验证码"); tabIdx++
	}
	if hasEmail {
		tabsHTML += tabBtn(tabIdx, "邮箱验证码"); tabIdx++
	}

	// Build tab panels
	var panelsHTML string
	panelIdx := 0
	if hasPassword {
		panelsHTML += passwordPanel(apiBase, tenantID.String(), panelIdx)
		panelIdx++
	}
	if hasPasskey {
		panelsHTML += passkeyPanel(apiBase, tenantID.String(), panelIdx)
		panelIdx++
	}
	if hasSMS {
		panelsHTML += otpPanel(apiBase, tenantID.String(), panelIdx, "sms")
		panelIdx++
	}
	if hasEmail {
		panelsHTML += otpPanel(apiBase, tenantID.String(), panelIdx, "email")
		panelIdx++
	}

	fmt.Fprintf(w, `<!DOCTYPE html>
<html lang="zh-CN">
<head>
<meta charset="utf-8"><meta name="viewport" content="width=device-width,initial-scale=1">
<title>GGID 登录</title>
<style>
*{margin:0;padding:0;box-sizing:border-box}
body{font-family:-apple-system,BlinkMacSystemFont,'Segoe UI',Roboto,sans-serif;background:linear-gradient(135deg,#667eea 0%%,#764ba2 100%%);min-height:100vh;display:flex;align-items:center;justify-content:center}
.card{background:#fff;border-radius:12px;padding:40px;width:400px;max-width:90vw;box-shadow:0 20px 60px rgba(0,0,0,.15)}
h1{text-align:center;color:#1677ff;margin-bottom:8px;font-size:24px}
.sub{text-align:center;color:#999;margin-bottom:24px;font-size:14px}
.err{color:#e74c3c;font-size:13px;margin:8px 0;display:none}
label{display:block;margin-bottom:6px;font-weight:600;font-size:14px;color:#333}
input{width:100%%;padding:12px;border:1px solid #ddd;border-radius:8px;font-size:14px;margin-bottom:16px}
input:focus{outline:none;border-color:#1677ff}
button{width:100%%;padding:12px;background:#1677ff;color:#fff;border:none;border-radius:8px;font-size:14px;cursor:pointer}
button:hover{background:#0958d9}
button:disabled{opacity:.6;cursor:not-allowed}
.tabs{display:flex;gap:4px;margin-bottom:20px;border-bottom:2px solid #eee}
.tab{padding:10px 16px;font-size:14px;cursor:pointer;border-bottom:2px solid transparent;margin-bottom:-2px;color:#999}
.tab.active{color:#1677ff;border-color:#1677ff;font-weight:600}
.panel{display:none}
.panel.active{display:block}
.otp-row{display:flex;gap:8px}
.otp-row input{flex:1;margin-bottom:0}
.otp-btn{width:auto;padding:12px 16px;white-space:nowrap;background:#f0f5ff;color:#1677ff;border:1px solid #91caff}
.redirect-info{margin-top:16px;text-align:center;font-size:12px;color:#bbb}
</style>
</head>
<body>
<div class="card">
<h1>🔐 GGID</h1>
<p class="sub">登录以继续</p>
<div class="tabs">%s</div>
<div id="err" class="err"></div>
%s
</div>
<script>
const API_BASE='%s',TENANT_ID='%s';
function showTab(i){document.querySelectorAll('.tab').forEach((t,idx)=>t.classList.toggle('active',idx===i));document.querySelectorAll('.panel').forEach((p,idx)=>p.classList.toggle('active',idx===i));document.getElementById('err').style.display='none';}
document.querySelectorAll('.tab').forEach((t,idx)=>t.onclick=()=>showTab(idx));
function redirectWithParam(key,val){const u=new URL(window.location.href),a=u.pathname+u.search,s=a.includes('?')?'&':'?';window.location.href=a+s+key+'='+val;}

// Password login
document.getElementById('pwForm')&&document.getElementById('pwForm').addEventListener('submit',async(e)=>{e.preventDefault();const b=document.getElementById('pwBtn');const er=document.getElementById('err');b.disabled=true;b.textContent='登录中...';er.style.display='none';try{const r=await fetch(API_BASE+'/api/v1/auth/verify',{method:'POST',headers:{'Content-Type':'application/json','X-Tenant-ID':TENANT_ID},body:JSON.stringify({username:document.getElementById('pwUser').value,password:document.getElementById('pwPass').value,tenant_id:TENANT_ID})});const d=await r.json();if(!r.ok){er.textContent=d.error?.message||d.error||'登录失败';er.style.display='block';b.disabled=false;b.textContent='登录';return;}redirectWithParam('user_id',d.user_id);}catch(err){er.textContent='网络错误';er.style.display='block';b.disabled=false;b.textContent='登录';}});

// Passkey login
async function passkeyLogin(){const btn=document.getElementById('pkBtn');if(!btn)return;btn.disabled=true;btn.textContent='请稍候...';const er=document.getElementById('err');er.style.display='none';try{const br=await fetch(API_BASE+'/api/v1/auth/webauthn/login/begin',{method:'POST',headers:{'Content-Type':'application/json','X-Tenant-ID':TENANT_ID}});if(!br.ok)throw new Error('无法启动Passkey认证');const opts=await br.json();const cred=await navigator.credentials.get({publicKey:opts.response});const body={id:cred.id,rawId:btoa(String.fromCharCode(...new Uint8Array(cred.rawId))),type:cred.type,response:{authenticatorData:btoa(String.fromCharCode(...new Uint8Array(cred.response.authenticatorData))),clientDataJSON:btoa(String.fromCharCode(...new Uint8Array(cred.response.clientDataJSON))),signature:btoa(String.fromCharCode(...new Uint8Array(cred.response.signature))),userHandle:cred.response.userHandle?btoa(String.fromCharCode(...new Uint8Array(cred.response.userHandle))):null}};const fr=await fetch(API_BASE+'/api/v1/auth/webauthn/login/finish',{method:'POST',headers:{'Content-Type':'application/json','X-Tenant-ID':TENANT_ID},body:JSON.stringify(body)});const res=await fr.json();if(!fr.ok)throw new Error(res.error||'Passkey验证失败');if(res.auth_ticket){redirectWithParam('auth_ticket',res.auth_ticket);}else throw new Error('未收到认证票据');}catch(e){if(e.name==='NotAllowedError')er.textContent='Passkey认证被取消';else er.textContent=e.message;er.style.display='block';btn.disabled=false;btn.textContent='使用Passkey登录';}}
document.getElementById('pkBtn')&&document.getElementById('pkBtn').addEventListener('click',passkeyLogin);

// OTP (SMS/Email) login
function setupOTP(prefix,channel){const sendBtn=document.getElementById(prefix+'SendBtn');if(!sendBtn)return;const verifyBtn=document.getElementById(prefix+'VerifyBtn');const er=document.getElementById('err');let countdown=0;sendBtn.addEventListener('click',async()=>{const id=document.getElementById(prefix+'Id').value;if(!id){er.textContent='请输入标识';er.style.display='block';return;}sendBtn.disabled=true;er.style.display='none';try{const r=await fetch(API_BASE+'/api/v1/auth/otp/send',{method:'POST',headers:{'Content-Type':'application/json','X-Tenant-ID':TENANT_ID},body:JSON.stringify({identifier:id,channel:channel})});const d=await r.json();if(!r.ok)throw new Error(d.error||'发送失败');er.textContent='验证码已发送';er.style.color='#52c41a';er.style.display='block';document.getElementById(prefix+'CodeWrap').style.display='block';countdown=60;const tick=()=>{if(countdown>0){sendBtn.textContent=countdown+'s';sendBtn.disabled=true;countdown--;setTimeout(tick,1000);}else{sendBtn.textContent='重新发送';sendBtn.disabled=false;}};tick();}catch(e){er.textContent=e.message;er.style.display='block';sendBtn.disabled=false;}});verifyBtn.addEventListener('click',async()=>{const id=document.getElementById(prefix+'Id').value;const code=document.getElementById(prefix+'Code').value;if(!code){er.textContent='请输入验证码';er.style.display='block';return;}verifyBtn.disabled=true;er.style.display='none';try{const r=await fetch(API_BASE+'/api/v1/auth/otp/verify',{method:'POST',headers:{'Content-Type':'application/json','X-Tenant-ID':TENANT_ID},body:JSON.stringify({identifier:id,code:code,channel:channel})});const d=await r.json();if(!r.ok)throw new Error(d.error||'验证失败');if(d.auth_ticket){redirectWithParam('auth_ticket',d.auth_ticket);}else throw new Error('未收到认证票据');}catch(e){er.textContent=e.message;er.style.display='block';verifyBtn.disabled=false;}});}
setupOTP('sms','sms');
setupOTP('email','email');
</script>
</body>
</html>`, tabsHTML, panelsHTML, apiBase, tenantID.String())
}
func tabBtn(idx int, label string) string {
	active := ""
	if idx == 0 {
		active = " active"
	}
	return fmt.Sprintf(`<div class="tab%s" data-idx="%d">%s</div>`, active, idx, label)
}

func passwordPanel(apiBase, tenantID string, idx int) string {
	active := ""
	if idx == 0 {
		active = " active"
	}
	return fmt.Sprintf(`<div class="panel%s" id="panel-pw">
<form id="pwForm">
<label>用户名</label>
<input id="pwUser" type="text" required autocomplete="username" placeholder="输入用户名">
<label>密码</label>
<input id="pwPass" type="password" required autocomplete="current-password" placeholder="输入密码">
<button type="submit" id="pwBtn">登录</button>
</form>
</div>`, active)
}

func passkeyPanel(apiBase, tenantID string, idx int) string {
	active := ""
	if idx == 0 {
		active = " active"
	}
	return fmt.Sprintf(`<div class="panel%s" id="panel-pk">
<div style="text-align:center;padding:20px 0">
<p style="color:#999;margin-bottom:16px">使用指纹、面容或安全密钥登录</p>
<button id="pkBtn" type="button">使用 Passkey 登录</button>
</div>
</div>`, active)
}

func otpPanel(apiBase, tenantID string, idx int, channel string) string {
	active := ""
	if idx == 0 {
		active = " active"
	}
	prefix := channel
	label := "手机号"
	placeholder := "+86 138 0000 0000"
	if channel == "email" {
		label = "邮箱地址"
		placeholder = "user@example.com"
	}
	return fmt.Sprintf(`<div class="panel%s" id="panel-%s">
<label>%s</label>
<input id="%sId" type="text" placeholder="%s">
<div id="%sCodeWrap" style="display:none">
<label>验证码</label>
<div class="otp-row">
<input id="%sCode" type="text" maxlength="6" placeholder="6位数字">
<button class="otp-btn" id="%sVerifyBtn" type="button">验证</button>
</div>
</div>
<button id="%sSendBtn" type="button" style="margin-top:12px">发送验证码</button>
</div>`, active, prefix, label, prefix, placeholder, prefix, prefix, prefix, prefix)
}
