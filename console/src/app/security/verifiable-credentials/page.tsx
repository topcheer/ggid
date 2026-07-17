"use client";

import { useState, useCallback, useEffect } from "react";
import {
  Shield, Loader2, AlertCircle, X, RefreshCw, Plus, Trash2, Check,
  CheckCircle, XCircle, KeyRound, FileJson, Eye, Code, Download,
  ArrowRight, ChevronRight, ShieldCheck, ShieldAlert, Clock,
  QrCode, List, Award, Smartphone,
} from "lucide-react";
import { useTranslations } from "@/lib/i18n";
import { authHeader } from "@/lib/auth-helpers";

const TENANT_ID = "00000000-0000-0000-0000-000000000001";

interface VCTemplate {
  id: string;
  name: string;
  type: string[];
  schema_url: string;
  claims: { name: string; type: string; required: boolean }[];
  signing_algorithm: "EdDSA" | "ES256K" | "RS256";
  status_list_type: "StatusList2021" | "RevocationList2020" | "none";
  created_at: string;
  issued_count: number;
}

interface IssuedVC {
  id: string;
  template_name: string;
  user_id: string;
  username: string;
  issued_at: string;
  expires_at: string;
  status: "valid" | "revoked" | "expired";
  vc_id: string;
}

interface DIDDocument {
  did: string;
  verification_method: string;
  key_type: string;
  created: string;
  rotated_at: string | null;
  well_known_url: string;
}

interface VerifyResult {
  valid: boolean;
  checks: { name: string; passed: boolean; detail: string }[];
  claims: Record<string, string>;
}

type Tab = "templates" | "issue" | "oid4vci" | "verify" | "issued" | "statuslist" | "did" | "trust";

const claimTypes = ["string", "number", "boolean", "date", "email", "url"];
const signAlgos = [
  { id: "EdDSA", name: "EdDSA (Ed25519)", desc: "Recommended. Compact + fast." },
  { id: "ES256K", name: "ES256K (secp256k1)", desc: "Blockchain-compatible." },
  { id: "RS256", name: "RS256 (RSA)", desc: "Legacy compatibility." },
];

const statusColors: Record<string, string> = {
  valid: "bg-green-100 text-green-700 dark:bg-green-900/30 dark:text-green-400",
  revoked: "bg-red-100 text-red-700 dark:bg-red-900/30 dark:text-red-400",
  expired: "bg-gray-100 text-gray-500 dark:bg-gray-800 dark:text-gray-400",
};

export default function VerifiableCredentialsPage() {
  const t = useTranslations();
  const [tab, setTab] = useState<Tab>("templates");
  const [templates, setTemplates] = useState<VCTemplate[]>([]);
  const [issuedVCs, setIssuedVCs] = useState<IssuedVC[]>([]);
  const [didDoc, setDidDoc] = useState<DIDDocument | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  // Template editor
  const [showEditor, setShowEditor] = useState(false);
  const [editTemplate, setEditTemplate] = useState<Partial<VCTemplate> | null>(null);
  const [claims, setClaims] = useState<VCTemplate["claims"]>([]);
  const [saving, setSaving] = useState(false);
  // Issue
  const [issueTemplateId, setIssueTemplateId] = useState("");
  const [issueUserId, setIssueUserId] = useState("");
  const [issueClaims, setIssueClaims] = useState("{}");
  const [issuing, setIssuing] = useState(false);
  // Verify
  const [verifyInput, setVerifyInput] = useState("");
  const [verifyResult, setVerifyResult] = useState<VerifyResult | null>(null);
  const [verifying, setVerifying] = useState(false);
  // Actions
  const [revokingId, setRevokingId] = useState<string | null>(null);
  const [rotatingKey, setRotatingKey] = useState(false);

  const loadData = useCallback(async () => {
    setLoading(true);
    setError(null);
    try {
      const h = { ...authHeader(), "X-Tenant-ID": TENANT_ID };
      const [tmplRes, issuedRes, didRes] = await Promise.all([
        fetch("/api/v1/identity/vc/templates", { headers: h }).catch(() => null),
        fetch("/api/v1/identity/vc/issued?page_size=50", { headers: h }).catch(() => null),
        fetch("/api/v1/identity/did/document", { headers: h }).catch(() => null),
      ]);
      if (tmplRes?.ok) { const d = await tmplRes.json(); setTemplates(d.templates || d.items || []); }
      if (issuedRes?.ok) { const d = await issuedRes.json(); setIssuedVCs(d.credentials || d.items || []); }
      if (didRes?.ok) setDidDoc(await didRes.json());
    } catch { setError("Failed to load VC data"); }
    finally { setLoading(false); }
  }, []);

  useEffect(() => { loadData(); }, [loadData]);

  const saveTemplate = async () => {
    if (!editTemplate?.name) return;
    setSaving(true);
    try {
      const method = editTemplate.id ? "PUT" : "POST";
      const url = editTemplate.id ? `/api/v1/identity/vc/templates/${editTemplate.id}` : "/api/v1/identity/vc/templates";
      await fetch(url, { method, headers: { ...authHeader(), "Content-Type": "application/json", "X-Tenant-ID": TENANT_ID }, body: JSON.stringify({ ...editTemplate, claims }) });
      setShowEditor(false); setEditTemplate(null); setClaims([]);
      loadData();
    } catch { setError("Failed to save template"); }
    finally { setSaving(false); }
  };

  const issueVC = async () => {
    if (!issueTemplateId || !issueUserId) return;
    setIssuing(true);
    try {
      const res = await fetch("/api/v1/identity/vc/issue", {
        method: "POST",
        headers: { ...authHeader(), "Content-Type": "application/json", "X-Tenant-ID": TENANT_ID },
        body: JSON.stringify({ template_id: issueTemplateId, user_id: issueUserId, claims: JSON.parse(issueClaims) }),
      });
      if (res.ok) { setIssueUserId(""); setIssueClaims("{}"); loadData(); }
      else { setError("Failed to issue VC"); }
    } catch { setError("Network error"); }
    finally { setIssuing(false); }
  };

  const verifyVP = async () => {
    if (!verifyInput) return;
    setVerifying(true);
    setVerifyResult(null);
    try {
      const res = await fetch("/api/v1/identity/vc/verify", {
        method: "POST",
        headers: { ...authHeader(), "Content-Type": "application/json", "X-Tenant-ID": TENANT_ID },
        body: JSON.stringify({ presentation: JSON.parse(verifyInput) }),
      });
      if (res.ok) setVerifyResult(await res.json());
      else { const d = await res.json().catch(() => ({})); setError(d.error || "Verification failed"); }
    } catch { setError("Invalid JSON or network error"); }
    finally { setVerifying(false); }
  };

  const revokeVC = async (id: string) => {
    setRevokingId(id);
    try {
      await fetch(`/api/v1/identity/vc/issued/${id}/revoke`, { method: "POST", headers: { ...authHeader(), "X-Tenant-ID": TENANT_ID } });
      setIssuedVCs(prev => prev.map(v => v.id === id ? { ...v, status: "revoked" } : v));
    } catch { setError("Failed to revoke"); }
    finally { setRevokingId(null); }
  };

  const rotateKey = async () => {
    setRotatingKey(true);
    try {
      const res = await fetch("/api/v1/identity/did/rotate-key", { method: "POST", headers: { ...authHeader(), "X-Tenant-ID": TENANT_ID } });
      if (res.ok) loadData();
    } catch { setError("Key rotation failed"); }
    finally { setRotatingKey(false); }
  };

  const cardCls = "rounded-xl border border-gray-200 bg-white p-5 shadow-sm dark:border-gray-700 dark:bg-gray-800";

  return (
    <div className="space-y-6">
      <div className="flex items-center justify-between">
        <div>
          <h1 className="flex items-center gap-2 text-2xl font-bold text-gray-900 dark:text-white"><Shield className="h-6 w-6 text-purple-500" /> Verifiable Credentials</h1>
          <p className="mt-1 text-sm text-gray-500 dark:text-gray-400">W3C VC management — issue, verify, revoke credentials with DID:web + JSON-LD schemas.</p>
        </div>
        <button onClick={loadData} disabled={loading} aria-label="Refresh" className="flex items-center gap-2 rounded-lg border border-gray-300 px-3 py-2 text-sm text-gray-600 hover:bg-gray-50 dark:border-gray-700 dark:text-gray-300 dark:hover:bg-gray-800"><RefreshCw className={"h-4 w-4 " + (loading ? "animate-spin" : "")} /> Refresh</button>
      </div>

      {error && <div role="alert" className="flex items-center gap-2 rounded-lg bg-red-50 px-4 py-3 text-sm text-red-700 dark:bg-red-900/20 dark:text-red-400"><AlertCircle className="h-4 w-4 shrink-0" />{error}<button onClick={() => setError(null)} aria-label="Dismiss" className="ml-auto"><X className="h-4 w-4" /></button></div>}

      <div className="flex gap-1 border-b border-gray-200 dark:border-gray-700 overflow-x-auto">
        {([
          { id: "templates" as Tab, label: "Templates", icon: FileJson },
          { id: "issue" as Tab, label: "Issue", icon: ShieldCheck },
          { id: "oid4vci" as Tab, label: "OID4VCI", icon: QrCode },
          { id: "verify" as Tab, label: "Verify", icon: Eye },
          { id: "issued" as Tab, label: "Issued VCs", icon: KeyRound },
          { id: "statuslist" as Tab, label: "Status List", icon: List },
          { id: "did" as Tab, label: "DID Management", icon: Shield },
          { id: "trust" as Tab, label: "Trust Registry", icon: Award },
        ]).map(tb => { const Icon = tb.icon; return (
          <button key={tb.id} onClick={() => setTab(tb.id)} aria-pressed={tab === tb.id} className={"flex items-center gap-1.5 border-b-2 px-4 py-2 text-sm font-medium transition whitespace-nowrap " + (tab === tb.id ? "border-purple-600 text-purple-600 dark:text-purple-400" : "border-transparent text-gray-500 hover:text-gray-700 dark:hover:text-gray-300")}><Icon className="h-4 w-4" /> {tb.label}</button>
        ); })}
      </div>

      {loading ? <div className="flex justify-center py-12"><Loader2 className="h-8 w-8 animate-spin text-purple-500" /></div> : (<>

      {/* TEMPLATES */}
      {tab === "templates" && (<>
        <div className="flex justify-end"><button onClick={() => { setEditTemplate({ name: "", type: [], signing_algorithm: "EdDSA", status_list_type: "StatusList2021" }); setClaims([{ name: "name", type: "string", required: true }]); setShowEditor(true); }} className="flex items-center gap-2 rounded-lg bg-purple-600 px-3 py-2 text-sm font-medium text-white hover:bg-purple-700"><Plus className="h-4 w-4" /> New Template</button></div>
        {templates.length === 0 ? <div className={cardCls}><div className="py-12 text-center"><FileJson className="mx-auto h-12 w-12 text-gray-300" /><p className="mt-4 text-sm text-gray-400">No VC templates defined.</p></div></div> : (
          <div className="space-y-3">{templates.map(tmpl => (
            <div key={tmpl.id} className={cardCls}>
              <div className="flex items-start justify-between">
                <div className="flex-1">
                  <div className="flex items-center gap-2"><span className="font-medium text-gray-900 dark:text-white">{tmpl.name}</span>{tmpl.type?.map(ty => <span key={ty} className="px-1.5 py-0.5 rounded bg-purple-100 text-purple-700 dark:bg-purple-900/30 dark:text-purple-400 text-xs font-mono">{ty}</span>)}<span className="px-1.5 py-0.5 rounded bg-gray-100 dark:bg-gray-700 text-xs">{tmpl.signing_algorithm}</span></div>
                  <div className="mt-2 flex flex-wrap gap-1">{tmpl.claims?.map(c => <span key={c.name} className="px-1.5 py-0.5 rounded bg-blue-50 dark:bg-blue-950/30 text-xs font-mono text-blue-600 dark:text-blue-400">{c.name}{c.required ? "*" : ""}</span>)}</div>
                  <p className="mt-2 text-xs text-gray-400">{tmpl.issued_count} issued · {tmpl.status_list_type}</p>
                </div>
                <div className="flex gap-1"><button onClick={() => { setEditTemplate(tmpl); setClaims(tmpl.claims || []); setShowEditor(true); }} aria-label="Edit" className="rounded p-1 text-gray-400 hover:bg-gray-100 dark:hover:bg-gray-700"><Code className="h-4 w-4" /></button><button onClick={() => revokeVC(tmpl.id)} aria-label="Delete" className="rounded p-1 text-red-400 hover:bg-red-50 dark:hover:bg-red-900/20"><Trash2 className="h-4 w-4" /></button></div>
              </div>
            </div>
          ))}</div>
        )}
      </>)}

      {/* ISSUE */}
      {tab === "issue" && (
        <div className="grid grid-cols-1 gap-6 lg:grid-cols-2">
          <div className={cardCls}>
            <h2 className="mb-4 flex items-center gap-2 text-sm font-semibold uppercase text-gray-400"><ShieldCheck className="h-4 w-4" /> Issue Credential</h2>
            <div className="space-y-3">
              <div><label className="text-sm font-medium">Template</label><select aria-label="VC template" value={issueTemplateId} onChange={e => setIssueTemplateId(e.target.value)} className="mt-1 block w-full rounded-lg border dark:border-gray-700 dark:bg-gray-900 px-3 py-2 text-sm"><option value="">Select template...</option>{templates.map(t => <option key={t.id} value={t.id}>{t.name}</option>)}</select></div>
              <div><label className="text-sm font-medium">User ID *</label><input aria-label="User ID" type="text" value={issueUserId} onChange={e => setIssueUserId(e.target.value)} placeholder="user-uuid" className="mt-1 w-full rounded-lg border dark:border-gray-700 dark:bg-gray-900 px-3 py-2 text-sm font-mono" /></div>
              <div><label className="text-sm font-medium">Claims (JSON)</label><textarea aria-label="Claims JSON" value={issueClaims} onChange={e => setIssueClaims(e.target.value)} rows={5} className="mt-1 w-full rounded-lg border dark:border-gray-700 dark:bg-gray-900 px-3 py-2 font-mono text-xs" placeholder='{"name":"Alice","degree":"BSc"}' /></div>
              <button onClick={issueVC} disabled={!issueTemplateId || !issueUserId || issuing} className="flex items-center gap-2 rounded-lg bg-purple-600 px-4 py-2 text-sm font-medium text-white hover:bg-purple-700 disabled:opacity-50">{issuing ? <Loader2 className="h-4 w-4 animate-spin" /> : <ShieldCheck className="h-4 w-4" />} Issue Credential</button>
            </div>
          </div>
          <div className={cardCls}>
            <h2 className="mb-4 flex items-center gap-2 text-sm font-semibold uppercase text-gray-400"><FileJson className="h-4 w-4" /> JSON-LD Schema Preview</h2>
            {issueTemplateId ? <pre className="overflow-x-auto rounded-lg bg-gray-900 p-3 text-xs text-green-400 font-mono">{JSON.stringify({ "@context": ["https://www.w3.org/2018/credentials/v1"], type: ["VerifiableCredential", ...(templates.find(t => t.id === issueTemplateId)?.type || [])], credentialSubject: { id: `did:web:${typeof window !== "undefined" ? window.location.hostname : "ggid.dev"}:${issueUserId || "user"}`, ...(JSON.parse(issueClaims || "{}")) } }, null, 2)}</pre> : <p className="text-sm text-gray-400">Select a template to preview schema.</p>}
          </div>
        </div>
      )}

      {/* VERIFY */}
      {tab === "verify" && (
        <div className="grid grid-cols-1 gap-6 lg:grid-cols-2">
          <div className={cardCls}>
            <h2 className="mb-4 flex items-center gap-2 text-sm font-semibold uppercase text-gray-400"><Eye className="h-4 w-4" /> Verifiable Presentation</h2>
            <textarea aria-label="VP JSON input" value={verifyInput} onChange={e => setVerifyInput(e.target.value)} rows={10} className="w-full rounded-lg border dark:border-gray-700 dark:bg-gray-900 px-3 py-2 font-mono text-xs" placeholder='{"@context":["https://www.w3.org/2018/credentials/v1"],"type":"VerifiablePresentation","verifiableCredential":{...}}' />
            <button onClick={verifyVP} disabled={!verifyInput || verifying} className="mt-3 flex items-center gap-2 rounded-lg bg-purple-600 px-4 py-2 text-sm font-medium text-white hover:bg-purple-700 disabled:opacity-50">{verifying ? <Loader2 className="h-4 w-4 animate-spin" /> : <Shield className="h-4 w-4" />} Verify Presentation</button>
          </div>
          <div className={cardCls}>
            <h2 className="mb-4 flex items-center gap-2 text-sm font-semibold uppercase text-gray-400"><ShieldCheck className="h-4 w-4" /> Verification Result</h2>
            {verifyResult ? (
              <div>
                <div className={"flex items-center gap-3 rounded-xl border-2 p-4 " + (verifyResult.valid ? "border-green-300 bg-green-50 dark:border-green-700 dark:bg-green-950/30" : "border-red-300 bg-red-50 dark:border-red-700 dark:bg-red-950/30")}>
                  {verifyResult.valid ? <CheckCircle className="h-8 w-8 text-green-500" /> : <XCircle className="h-8 w-8 text-red-500" />}
                  <div><p className={"text-xl font-bold " + (verifyResult.valid ? "text-green-700 dark:text-green-400" : "text-red-700 dark:text-red-400")}>{verifyResult.valid ? "VALID" : "INVALID"}</p></div>
                </div>
                <div className="mt-3 space-y-1">{verifyResult.checks?.map((c, i) => (
                  <div key={i} className={"flex items-center gap-2 rounded-lg p-2 " + (c.passed ? "bg-green-50 dark:bg-green-950/20" : "bg-red-50 dark:bg-red-950/20")}>{c.passed ? <CheckCircle className="h-3.5 w-3.5 text-green-500 shrink-0" /> : <XCircle className="h-3.5 w-3.5 text-red-500 shrink-0" />}<span className="font-medium text-xs">{c.name}</span>{i < (verifyResult.checks?.length || 1) - 1 && <ArrowRight className="h-3 w-3 text-gray-300 ml-auto" />}<span className="text-gray-400 text-xs ml-auto">{c.detail}</span></div>
                ))}</div>
                {verifyResult.claims && Object.keys(verifyResult.claims).length > 0 && (
                  <div className="mt-4"><p className="text-xs font-semibold uppercase text-gray-400 mb-2">Extracted Claims</p><div className="space-y-1">{Object.entries(verifyResult.claims).map(([k, v]) => <div key={k} className="flex justify-between text-xs"><span className="text-gray-500">{k}</span><span className="font-mono">{v}</span></div>)}</div></div>
                )}
              </div>
            ) : <div className="py-12 text-center"><Eye className="mx-auto h-10 w-10 text-gray-300" /><p className="mt-3 text-sm text-gray-400">Paste a VP and click Verify.</p></div>}
          </div>
        </div>
      )}

      {/* ISSUED VCs */}
      {tab === "issued" && (
        <div className={cardCls}>
          <h2 className="mb-4 flex items-center gap-2 text-sm font-semibold uppercase text-gray-400"><KeyRound className="h-4 w-4" /> Issued Credentials</h2>
          {issuedVCs.length === 0 ? <div className="py-8 text-center"><KeyRound className="mx-auto h-10 w-10 text-gray-300" /><p className="mt-3 text-sm text-gray-400">No credentials issued yet.</p></div> : (
            <div className="overflow-x-auto"><table className="w-full text-sm">
              <thead className="bg-gray-50 dark:bg-gray-900/50"><tr><th scope="col" className="px-4 py-3 text-left font-medium">Template</th><th scope="col" className="px-4 py-3 text-left font-medium">User</th><th scope="col" className="px-4 py-3 text-left font-medium">Issued</th><th scope="col" className="px-4 py-3 text-left font-medium">Expires</th><th scope="col" className="px-4 py-3 text-center font-medium">Status</th><th scope="col" className="px-4 py-3 text-right font-medium">Action</th></tr></thead>
              <tbody className="divide-y dark:divide-gray-800">{issuedVCs.map(vc => (
                <tr key={vc.id} className="hover:bg-gray-50 dark:hover:bg-gray-900/30">
                  <td className="px-4 py-3 text-xs font-medium">{vc.template_name}</td>
                  <td className="px-4 py-3 text-xs">{vc.username || vc.user_id}</td>
                  <td className="px-4 py-3 text-xs text-gray-500">{new Date(vc.issued_at).toLocaleDateString()}</td>
                  <td className="px-4 py-3 text-xs text-gray-500">{vc.expires_at ? new Date(vc.expires_at).toLocaleDateString() : "—"}</td>
                  <td className="px-4 py-3 text-center"><span className={"px-2 py-0.5 rounded text-xs font-medium " + (statusColors[vc.status] || "")}>{vc.status}</span></td>
                  <td className="px-4 py-3 text-right">{vc.status === "valid" && <button onClick={() => revokeVC(vc.id)} disabled={revokingId === vc.id} aria-label="Revoke" className="rounded-lg bg-red-50 px-2 py-1 text-xs text-red-600 hover:bg-red-100 dark:bg-red-950/20 disabled:opacity-50">{revokingId === vc.id ? <Loader2 className="h-3.5 w-3.5 animate-spin" /> : "Revoke"}</button>}</td>
                </tr>
              ))}</tbody>
            </table></div>
          )}
        </div>
      )}

      {/* OID4VCI */}
      {tab === "oid4vci" && (
        <div className="grid grid-cols-1 gap-6 lg:grid-cols-2">
          <div className={cardCls}>
            <h2 className="mb-4 flex items-center gap-2 text-sm font-semibold uppercase text-gray-400"><QrCode className="h-4 w-4" /> OID4VCI Issuance</h2>
            <div className="space-y-3">
              <div><label className="text-sm font-medium">Credential Offer URL</label><div className="mt-1 rounded-lg bg-gray-900 p-3 font-mono text-xs text-green-400 break-all">openid-credential-offer://?credential_offer_uri=https://ggid.dev/api/v1/identity/vc/offer/{issueTemplateId || "TEMPLATE_ID"}</div></div>
              <div className="flex items-center justify-center py-6"><div className="h-48 w-48 rounded-xl border-2 border-dashed border-gray-300 dark:border-gray-700 flex items-center justify-center"><QrCode className="h-24 w-24 text-gray-300" /></div></div>
              <div className="space-y-2">
                {[
                  { step: "Wallet scans QR", status: "pending" },
                  { step: "User authorizes issuance", status: "pending" },
                  { step: "Server signs credential", status: "pending" },
                  { step: "Wallet stores credential", status: "pending" },
                ].map((s, i) => (
                  <div key={i} className="flex items-center gap-3 rounded-lg border p-2 dark:border-gray-700">
                    <div className="flex h-6 w-6 items-center justify-center rounded-full bg-gray-200 dark:bg-gray-700 text-xs font-bold">{i + 1}</div>
                    <span className="flex-1 text-sm">{s.step}</span>
                    <span className="text-xs text-gray-400">{s.status}</span>
                  </div>
                ))}
              </div>
              <div>
                <label className="text-sm font-medium">Signing Algorithm</label>
                <div className="mt-1 grid grid-cols-2 gap-2">
                  {[{ id: "EdDSA", name: "Ed25519", desc: "Standard" }, { id: "SM2Signature2024", name: "SM2 (国密)", desc: "China compliant" }].map(a => (
                    <button key={a.id} className="rounded-lg border-2 border-gray-200 dark:border-gray-700 p-2 text-left hover:border-purple-400"><p className="text-sm font-medium">{a.name}</p><p className="text-xs text-gray-400">{a.desc}</p></button>
                  ))}
                </div>
              </div>
            </div>
          </div>
          <div className={cardCls}>
            <h2 className="mb-4 flex items-center gap-2 text-sm font-semibold uppercase text-gray-400"><Smartphone className="h-4 w-4" /> Wallet Connection Status</h2>
            <div className="rounded-xl bg-gray-50 p-6 text-center dark:bg-gray-900/50">
              <Smartphone className="mx-auto h-12 w-12 text-gray-300" />
              <p className="mt-3 text-sm text-gray-400">Waiting for wallet to scan QR code...</p>
              <div className="mt-4 flex items-center justify-center gap-1"><RefreshCw className="h-4 w-4 animate-spin text-purple-500" /><span className="text-xs text-gray-400">Listening for wallet connection</span></div>
            </div>
          </div>
        </div>
      )}

      {/* STATUS LIST */}
      {tab === "statuslist" && (
        <div className={cardCls}>
          <h2 className="mb-4 flex items-center gap-2 text-sm font-semibold uppercase text-gray-400"><List className="h-4 w-4" /> StatusList2021 Management</h2>
          <div className="space-y-4">
            <div className="grid grid-cols-3 gap-3">
              <div className="rounded-lg border p-3 dark:border-gray-700"><span className="text-xs text-gray-400">List Index Size</span><p className="text-xl font-bold mt-1">131,072</p></div>
              <div className="rounded-lg border p-3 dark:border-gray-700"><span className="text-xs text-gray-400">Revoked</span><p className="text-xl font-bold text-red-600 mt-1">0</p></div>
              <div className="rounded-lg border p-3 dark:border-gray-700"><span className="text-xs text-gray-400">Active</span><p className="text-xl font-bold text-green-600 mt-1">0</p></div>
            </div>
            <div>
              <p className="text-xs font-semibold uppercase text-gray-400 mb-2">Status Bitmap (first 256 positions)</p>
              <div className="grid grid-cols-16 gap-0.5" style={{ gridTemplateColumns: "repeat(32, 1fr)" }}>
                {Array.from({ length: 256 }).map((_, i) => (
                  <div key={i} className="aspect-square rounded-sm bg-green-200 dark:bg-green-900/30 hover:bg-red-300 cursor-pointer transition" title={`Index ${i}: Active`} />
                ))}
              </div>
              <div className="mt-2 flex gap-4 text-xs"><span className="flex items-center gap-1"><div className="h-3 w-3 rounded bg-green-200 dark:bg-green-900/30" /> Active</span><span className="flex items-center gap-1"><div className="h-3 w-3 rounded bg-red-300" /> Revoked</span></div>
            </div>
            <div className="flex items-center gap-2">
              <input aria-label="Revoke index" type="number" placeholder="Index to revoke..." className="w-40 rounded-lg border dark:border-gray-700 dark:bg-gray-900 px-3 py-2 text-sm" />
              <button className="flex items-center gap-1 rounded-lg bg-red-50 px-3 py-2 text-sm font-medium text-red-600 hover:bg-red-100 dark:bg-red-950/20">Revoke</button>
              <button className="flex items-center gap-1 rounded-lg border border-gray-300 px-3 py-2 text-sm dark:border-gray-700">Batch Revoke CSV</button>
            </div>
          </div>
        </div>
      )}

      {/* TRUST REGISTRY */}
      {tab === "trust" && (
        <div className={cardCls}>
          <h2 className="mb-4 flex items-center gap-2 text-sm font-semibold uppercase text-gray-400"><Award className="h-4 w-4" /> Trusted Issuer Registry</h2>
          <div className="space-y-3">
            <div className="rounded-lg border p-3 dark:border-gray-700"><div className="flex items-center gap-2"><ShieldCheck className="h-5 w-5 text-green-500" /><div className="flex-1"><p className="font-medium text-sm">did:web:ggid.dev</p><p className="text-xs text-gray-400">Self · Active</p></div><span className="px-2 py-0.5 rounded bg-green-100 text-green-700 dark:bg-green-900/30 dark:text-green-400 text-xs">Verified</span></div></div>
            <div className="rounded-lg border p-3 dark:border-gray-700"><div className="flex items-center gap-2"><KeyRound className="h-5 w-5 text-gray-400" /><div className="flex-1"><p className="font-mono text-sm">did:web:partner.example.com</p><p className="text-xs text-gray-400">Partner · Verified 2024-01-15</p></div><span className="px-2 py-0.5 rounded bg-green-100 text-green-700 dark:bg-green-900/30 dark:text-green-400 text-xs">Trusted</span></div></div>
            <div className="flex items-center gap-2"><input aria-label="Add trusted issuer" type="text" placeholder="did:web:new-issuer.com" className="flex-1 rounded-lg border dark:border-gray-700 dark:bg-gray-900 px-3 py-2 text-sm font-mono" /><button className="flex items-center gap-1 rounded-lg bg-purple-600 px-3 py-2 text-sm font-medium text-white hover:bg-purple-700"><Plus className="h-4 w-4" /> Add Trusted Issuer</button></div>
          </div>
        </div>
      )}

      {/* DID MANAGEMENT */}
      {tab === "did" && (
        <div className="grid grid-cols-1 gap-6 lg:grid-cols-2">
          <div className={cardCls}>
            <h2 className="mb-4 flex items-center gap-2 text-sm font-semibold uppercase text-gray-400"><Shield className="h-4 w-4" /> DID:web Document</h2>
            {didDoc ? (
              <div className="space-y-3">
                <div><span className="text-xs text-gray-400">DID</span><p className="font-mono text-sm break-all">{didDoc.did}</p></div>
                <div><span className="text-xs text-gray-400">Verification Method</span><p className="font-mono text-xs break-all">{didDoc.verification_method}</p></div>
                <div><span className="text-xs text-gray-400">Key Type</span><p className="text-sm">{didDoc.key_type}</p></div>
                <div><span className="text-xs text-gray-400">Created</span><p className="text-xs text-gray-500">{new Date(didDoc.created).toLocaleString()}</p></div>
                {didDoc.rotated_at && <div><span className="text-xs text-gray-400">Last Rotated</span><p className="text-xs text-gray-500">{new Date(didDoc.rotated_at).toLocaleString()}</p></div>}
                <div className="rounded-lg bg-blue-50 p-3 dark:bg-blue-950/30"><p className="text-xs text-blue-700 dark:text-blue-400">Well-known endpoint:<br /><span className="font-mono break-all">{didDoc.well_known_url}</span></p></div>
                <button onClick={rotateKey} disabled={rotatingKey} className="flex items-center gap-2 rounded-lg bg-amber-600 px-4 py-2 text-sm font-medium text-white hover:bg-amber-700 disabled:opacity-50">{rotatingKey ? <Loader2 className="h-4 w-4 animate-spin" /> : <KeyRound className="h-4 w-4" />} Rotate Signing Key</button>
              </div>
            ) : <div className="py-8 text-center"><Shield className="mx-auto h-10 w-10 text-gray-300" /><p className="mt-3 text-sm text-gray-400">No DID document configured.</p></div>}
          </div>
          <div className={cardCls}>
            <h2 className="mb-4 flex items-center gap-2 text-sm font-semibold uppercase text-gray-400"><FileJson className="h-4 w-4" /> DID Document Preview</h2>
            <pre className="overflow-x-auto rounded-lg bg-gray-900 p-3 text-xs text-green-400 font-mono max-h-96 overflow-y-auto">{JSON.stringify({ "@context": ["https://w3id.org/did-resolution/v1"], id: didDoc?.did || "did:web:ggid.dev", verificationMethod: [{ id: didDoc?.verification_method || "", type: didDoc?.key_type || "JsonWebKey2020", controller: didDoc?.did || "" }] }, null, 2)}</pre>
          </div>
        </div>
      )}

      </>)}

      {/* Template editor dialog */}
      {showEditor && editTemplate && (
        <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/50" onClick={() => setShowEditor(false)}>
          <div role="dialog" aria-modal="true" className="mx-4 max-h-[90vh] w-full max-w-lg overflow-y-auto rounded-xl bg-white p-6 shadow-xl dark:bg-gray-800" onClick={e => e.stopPropagation()}>
            <h3 className="flex items-center gap-2 text-lg font-semibold text-gray-900 dark:text-white"><FileJson className="h-5 w-5 text-purple-500" /> {editTemplate.id ? "Edit Template" : "New VC Template"}</h3>
            <div className="mt-4 space-y-3">
              <div><label className="text-sm font-medium">Template Name *</label><input aria-label="Template name" type="text" value={editTemplate.name || ""} onChange={e => setEditTemplate({ ...editTemplate, name: e.target.value })} placeholder="University Degree" className="mt-1 w-full rounded-lg border dark:border-gray-700 dark:bg-gray-900 px-3 py-2 text-sm" autoFocus /></div>
              <div><label className="text-sm font-medium">VC Types (comma-separated)</label><input aria-label="VC types" type="text" value={(editTemplate.type || []).join(", ")} onChange={e => setEditTemplate({ ...editTemplate, type: e.target.value.split(",").map(s => s.trim()).filter(Boolean) })} placeholder="DegreeCredential, AlumniCredential" className="mt-1 w-full rounded-lg border dark:border-gray-700 dark:bg-gray-900 px-3 py-2 text-sm font-mono" /></div>
              <div><label className="text-sm font-medium">Signing Algorithm</label><div className="mt-1 space-y-1">{signAlgos.map(a => <button key={a.id} onClick={() => setEditTemplate({ ...editTemplate, signing_algorithm: a.id as VCTemplate["signing_algorithm"] })} aria-pressed={editTemplate.signing_algorithm === a.id} className={"flex w-full items-center gap-2 rounded-lg border p-2 text-left " + (editTemplate.signing_algorithm === a.id ? "border-purple-500 bg-purple-50 dark:bg-purple-950/30" : "border-gray-200 dark:border-gray-700")}><div className="flex-1"><span className="text-sm font-medium">{a.name}</span><p className="text-xs text-gray-400">{a.desc}</p></div>{editTemplate.signing_algorithm === a.id && <Check className="h-4 w-4 text-purple-500" />}</button>)}</div></div>
              <div><label className="text-sm font-medium">Status List Type</label><select aria-label="Status list" value={editTemplate.status_list_type || "StatusList2021"} onChange={e => setEditTemplate({ ...editTemplate, status_list_type: e.target.value as VCTemplate["status_list_type"] })} className="mt-1 block rounded-lg border dark:border-gray-700 dark:bg-gray-900 px-3 py-2 text-sm"><option value="StatusList2021">StatusList2021</option><option value="RevocationList2020">RevocationList2020</option><option value="none">None</option></select></div>
              <div><label className="text-sm font-medium">Claims</label><div className="mt-1 space-y-1">{claims.map((c, i) => <div key={i} className="flex items-center gap-2"><input aria-label={`Claim ${i+1} name`} type="text" value={c.name} onChange={e => { const n = [...claims]; n[i] = { ...c, name: e.target.value }; setClaims(n); }} placeholder="field_name" className="flex-1 rounded border dark:border-gray-700 dark:bg-gray-900 px-2 py-1 text-xs font-mono" /><select aria-label={`Claim ${i+1} type`} value={c.type} onChange={e => { const n = [...claims]; n[i] = { ...c, type: e.target.value }; setClaims(n); }} className="rounded border dark:border-gray-700 dark:bg-gray-900 px-2 py-1 text-xs">{claimTypes.map(ct => <option key={ct}>{ct}</option>)}</select><button onClick={() => { setClaims(claims.filter((_, j) => j !== i)); }} className="text-red-400"><X className="h-3.5 w-3.5" /></button></div>)}</div><button onClick={() => setClaims([...claims, { name: "", type: "string", required: false }])} className="mt-2 flex items-center gap-1 text-xs text-purple-600 hover:underline"><Plus className="h-3 w-3" /> Add Claim</button></div>
            </div>
            <div className="mt-4 flex justify-end gap-2"><button onClick={() => setShowEditor(false)} className="rounded-lg border border-gray-300 px-4 py-2 text-sm dark:border-gray-700">Cancel</button><button onClick={saveTemplate} disabled={!editTemplate.name || saving} className="rounded-lg bg-purple-600 px-4 py-2 text-sm font-medium text-white hover:bg-purple-700 disabled:opacity-50">{saving ? <Loader2 className="h-4 w-4 animate-spin" /> : "Save"}</button></div>
          </div>
        </div>
      )}
    </div>
  );
}
