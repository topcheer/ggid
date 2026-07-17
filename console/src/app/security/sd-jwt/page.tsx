"use client";
import { useState } from "react";
import { Shield, Loader2, AlertCircle, X, Check, CheckCircle, XCircle, Eye, EyeOff, KeyRound, FileJson, Download, Copy, TestTube, ArrowRight } from "lucide-react";
import { authHeader } from "@/lib/auth-helpers";

const TENANT_ID = "00000000-0000-0000-0000-000000000001";

interface Claim { name: string; value: string; disclosed: boolean; }
interface VerifyResult { valid: boolean; disclosed_claims: Record<string, string>; hidden_claims_count: number; issuer: string; checks: { name: string; passed: boolean }[]; }

const SAMPLE: Claim[] = [
  { name: "sub", value: "user:alice", disclosed: true },
  { name: "name", value: "Alice", disclosed: true },
  { name: "email", value: "alice@corp.com", disclosed: false },
  { name: "age_over_18", value: "true", disclosed: true },
  { name: "degree", value: "MSc CS", disclosed: false },
  { name: "salary", value: "180000", disclosed: false },
  { name: "department", value: "Eng", disclosed: true },
  { name: "nationality", value: "CN", disclosed: false },
];

type Tab = "issue" | "verify" | "simulate" | "exchange";

export default function SDJWTPage() {
  const [tab, setTab] = useState<Tab>("issue");
  const [claims, setClaims] = useState<Claim[]>(SAMPLE);
  const [error, setError] = useState<string | null>(null);
  const [issued, setIssued] = useState("");
  const [issuing, setIssuing] = useState(false);
  const [verifyInput, setVerifyInput] = useState("");
  const [verifyResult, setVerifyResult] = useState<VerifyResult | null>(null);
  const [verifying, setVerifying] = useState(false);
  const [simSet, setSimSet] = useState<Set<string>>(new Set(["sub", "name", "age_over_18", "department"]));

  const issue = async () => {
    setIssuing(true);
    try {
      const res = await fetch("/api/v1/identity/vc/sd-jwt/issue", {
        method: "POST", headers: { ...authHeader(), "Content-Type": "application/json", "X-Tenant-ID": TENANT_ID },
        body: JSON.stringify({ claims }),
      });
      if (res.ok) { const d = await res.json(); setIssued(d.sd_jwt || d.token || "eyJ...preview..."); }
    } catch { setError("Network error"); }
    finally { setIssuing(false); }
  };

  const verify = async () => {
    if (!verifyInput) return;
    setVerifying(true); setVerifyResult(null);
    try {
      const res = await fetch("/api/v1/identity/vc/sd-jwt/verify", {
        method: "POST", headers: { ...authHeader(), "Content-Type": "application/json", "X-Tenant-ID": TENANT_ID },
        body: JSON.stringify({ presentation: verifyInput }),
      });
      if (res.ok) setVerifyResult(await res.json());
      else setError("Verification failed");
    } catch { setError("Network error"); }
    finally { setVerifying(false); }
  };

  const card = "rounded-xl border border-gray-200 bg-white p-5 shadow-sm dark:border-gray-700 dark:bg-gray-800";

  return (
    <div className="space-y-6">
      <div>
        <h1 className="flex items-center gap-2 text-2xl font-bold text-gray-900 dark:text-white">
          <Shield className="h-6 w-6 text-purple-500" />
          {"SD-JWT & OpenID4VP"}
        </h1>
        <p className="mt-1 text-sm text-gray-500 dark:text-gray-400">
          Selective Disclosure JWT issuance, verification, disclosure simulation, and credential exchange.
        </p>
      </div>

      {error && (
        <div role="alert" className="flex items-center gap-2 rounded-lg bg-red-50 px-4 py-3 text-sm text-red-700 dark:bg-red-900/20 dark:text-red-400">
          <AlertCircle className="h-4 w-4 shrink-0" />
          {error}
          <button onClick={() => setError(null)} aria-label="Dismiss" className="ml-auto">
            <X className="h-4 w-4" />
          </button>
        </div>
      )}

      {/* Tabs */}
      <div className="flex gap-1 border-b border-gray-200 dark:border-gray-700 overflow-x-auto">
        {([
          { id: "issue" as Tab, label: "Issuance", icon: KeyRound },
          { id: "verify" as Tab, label: "Verify", icon: Eye },
          { id: "simulate" as Tab, label: "Simulator", icon: TestTube },
          { id: "exchange" as Tab, label: "Exchange", icon: ArrowRight },
        ]).map(tb => {
          const Icon = tb.icon;
          return (
            <button
              key={tb.id}
              onClick={() => setTab(tb.id)}
              aria-pressed={tab === tb.id}
              className={`flex items-center gap-1.5 border-b-2 px-4 py-2 text-sm font-medium transition whitespace-nowrap ${tab === tb.id ? "border-purple-600 text-purple-600 dark:text-purple-400" : "border-transparent text-gray-500 hover:text-gray-700 dark:hover:text-gray-300"}`}
            >
              <Icon className="h-4 w-4" /> {tb.label}
            </button>
          );
        })}
      </div>

      {/* ISSUE TAB */}
      {tab === "issue" && (
        <div className="grid grid-cols-1 gap-6 lg:grid-cols-2">
          <div className={card}>
            <h2 className="mb-4 flex items-center gap-2 text-sm font-semibold uppercase text-gray-400">
              <KeyRound className="h-4 w-4" /> Claim Disclosure
            </h2>
            <div className="space-y-2">
              {claims.map((c, i) => (
                <div key={c.name} className="flex items-center gap-3 rounded-lg border p-2 dark:border-gray-700">
                  <button
                    onClick={() => setClaims(prev => prev.map((cl, j) => j === i ? { ...cl, disclosed: !cl.disclosed } : cl))}
                    aria-pressed={c.disclosed}
                    className={`flex h-7 w-7 items-center justify-center rounded-lg ${c.disclosed ? "bg-green-100 dark:bg-green-900/30" : "bg-gray-100 dark:bg-gray-800"}`}
                  >
                    {c.disclosed ? <Eye className="h-3.5 w-3.5 text-green-500" /> : <EyeOff className="h-3.5 w-3.5 text-gray-400" />}
                  </button>
                  <div className="flex-1">
                    <span className="font-mono text-xs text-purple-600 dark:text-purple-400">{c.name}</span>
                    <p className="text-xs text-gray-500">{c.value}</p>
                  </div>
                  <span className={`px-1.5 py-0.5 rounded text-xs ${c.disclosed ? "bg-green-100 dark:bg-green-900/30 text-green-600" : "bg-gray-100 dark:bg-gray-800 text-gray-400"}`}>
                    {c.disclosed ? "Disclosed" : "Hidden"}
                  </span>
                </div>
              ))}
            </div>
            <p className="mt-2 text-xs text-gray-400">Hidden claims are SHA-256 hashed. Verifier confirms existence without value.</p>
            <button onClick={issue} disabled={issuing} className="mt-4 flex items-center gap-2 rounded-lg bg-purple-600 px-4 py-2 text-sm font-medium text-white hover:bg-purple-700 disabled:opacity-50">
              {issuing ? <Loader2 className="h-4 w-4 animate-spin" /> : <KeyRound className="h-4 w-4" />} Issue SD-JWT
            </button>
          </div>

          <div className={card}>
            <h2 className="mb-4 flex items-center gap-2 text-sm font-semibold uppercase text-gray-400">
              <FileJson className="h-4 w-4" /> SD-JWT Preview
            </h2>
            {issued ? (
              <div>
                <pre className="overflow-x-auto rounded-lg bg-gray-900 p-3 text-xs text-green-400 font-mono max-h-48 overflow-y-auto break-all">{issued}</pre>
                <div className="mt-3 flex gap-2">
                  <button className="flex items-center gap-1 rounded-lg border border-gray-300 px-3 py-1.5 text-xs dark:border-gray-700"><Copy className="h-3 w-3" /> Copy</button>
                  <button className="flex items-center gap-1 rounded-lg border border-gray-300 px-3 py-1.5 text-xs dark:border-gray-700"><Download className="h-3 w-3" /> Download</button>
                </div>
                <div className="mt-3 grid grid-cols-2 gap-2">
                  <div className="rounded-lg border p-2 dark:border-gray-700"><p className="text-xs text-gray-400">Disclosed</p><p className="text-lg font-bold text-green-600">{claims.filter(c => c.disclosed).length}</p></div>
                  <div className="rounded-lg border p-2 dark:border-gray-700"><p className="text-xs text-gray-400">Hidden</p><p className="text-lg font-bold text-gray-400">{claims.filter(c => !c.disclosed).length}</p></div>
                </div>
              </div>
            ) : (
              <div className="py-8 text-center">
                <KeyRound className="mx-auto h-10 w-10 text-gray-300" />
                <p className="mt-3 text-sm text-gray-400">Configure claims and issue to preview.</p>
              </div>
            )}
          </div>
        </div>
      )}

      {/* VERIFY TAB */}
      {tab === "verify" && (
        <div className="grid grid-cols-1 gap-6 lg:grid-cols-2">
          <div className={card}>
            <h2 className="mb-4 flex items-center gap-2 text-sm font-semibold uppercase text-gray-400">
              <Eye className="h-4 w-4" /> OpenID4VP Presentation
            </h2>
            <textarea aria-label="VP input" value={verifyInput} onChange={e => setVerifyInput(e.target.value)} rows={10} className="w-full rounded-lg border dark:border-gray-700 dark:bg-gray-900 px-3 py-2 font-mono text-xs" placeholder="Paste SD-JWT + disclosures..." />
            <button onClick={verify} disabled={!verifyInput || verifying} className="mt-3 flex items-center gap-2 rounded-lg bg-purple-600 px-4 py-2 text-sm font-medium text-white hover:bg-purple-700 disabled:opacity-50">
              {verifying ? <Loader2 className="h-4 w-4 animate-spin" /> : <Shield className="h-4 w-4" />} Verify
            </button>
          </div>

          <div className={card}>
            <h2 className="mb-4 flex items-center gap-2 text-sm font-semibold uppercase text-gray-400">
              <CheckCircle className="h-4 w-4" /> Result
            </h2>
            {verifyResult ? (
              <div>
                <div className={`flex items-center gap-3 rounded-xl border-2 p-4 ${verifyResult.valid ? "border-green-300 bg-green-50 dark:border-green-700 dark:bg-green-950/30" : "border-red-300 bg-red-50 dark:border-red-700 dark:bg-red-950/30"}`}>
                  {verifyResult.valid ? <CheckCircle className="h-8 w-8 text-green-500" /> : <XCircle className="h-8 w-8 text-red-500" />}
                  <div>
                    <p className={`text-lg font-bold ${verifyResult.valid ? "text-green-700 dark:text-green-400" : "text-red-700 dark:text-red-400"}`}>
                      {verifyResult.valid ? "VALID" : "INVALID"}
                    </p>
                    <p className="text-xs text-gray-500">Issuer: {verifyResult.issuer || "—"}</p>
                  </div>
                </div>
                {verifyResult.checks?.map((c, i) => (
                  <div key={i} className="mt-2 flex items-center gap-2 text-xs">
                    {c.passed ? <CheckCircle className="h-3.5 w-3.5 text-green-500" /> : <XCircle className="h-3.5 w-3.5 text-red-500" />}
                    <span>{c.name}</span>
                  </div>
                ))}
                {verifyResult.disclosed_claims && Object.keys(verifyResult.disclosed_claims).length > 0 && (
                  <div className="mt-3">
                    <p className="text-xs font-semibold text-gray-400 mb-1">Disclosed ({verifyResult.hidden_claims_count || 0} hidden)</p>
                    <div className="flex flex-wrap gap-1">
                      {Object.entries(verifyResult.disclosed_claims).map(([k, v]) => (
                        <span key={k} className="px-1.5 py-0.5 rounded bg-green-50 dark:bg-green-950/30 text-xs font-mono">{k}={v}</span>
                      ))}
                    </div>
                  </div>
                )}
              </div>
            ) : (
              <div className="py-8 text-center">
                <Eye className="mx-auto h-10 w-10 text-gray-300" />
                <p className="mt-3 text-sm text-gray-400">Paste a presentation and verify.</p>
              </div>
            )}
          </div>
        </div>
      )}

      {/* SIMULATE TAB */}
      {tab === "simulate" && (
        <div className="grid grid-cols-1 gap-6 lg:grid-cols-2">
          <div className={card}>
            <h2 className="mb-4 flex items-center gap-2 text-sm font-semibold uppercase text-gray-400">
              <TestTube className="h-4 w-4" /> Select Claims to Disclose
            </h2>
            <p className="text-xs text-gray-500 mb-3">Holder selects which claims the verifier can see.</p>
            <div className="space-y-2">
              {SAMPLE.map(c => {
                const isD = simSet.has(c.name);
                return (
                  <label key={c.name} className="flex items-center gap-3 rounded-lg border p-2 dark:border-gray-700 cursor-pointer hover:bg-gray-50 dark:hover:bg-gray-900/30">
                    <input type="checkbox" checked={isD} onChange={() => {
                      setSimSet(prev => { const n = new Set(prev); if (n.has(c.name)) n.delete(c.name); else n.add(c.name); return n; });
                    }} className="rounded" />
                    <div className="flex-1">
                      <span className="font-mono text-xs text-purple-600 dark:text-purple-400">{c.name}</span>
                      <p className={`text-xs ${isD ? "text-gray-600 dark:text-gray-300" : "text-gray-300 line-through"}`}>{c.value}</p>
                    </div>
                    {isD ? <Eye className="h-3.5 w-3.5 text-green-500" /> : <EyeOff className="h-3.5 w-3.5 text-gray-300" />}
                  </label>
                );
              })}
            </div>
          </div>

          <div className={card}>
            <h2 className="mb-4 flex items-center gap-2 text-sm font-semibold uppercase text-gray-400">
              <Eye className="h-4 w-4" /> What Verifier Sees
            </h2>
            <div className="space-y-2">
              {SAMPLE.map(c => {
                const isD = simSet.has(c.name);
                return (
                  <div key={c.name} className={`flex items-center gap-2 rounded-lg border p-2 ${isD ? "border-green-200 dark:border-green-800" : "border-gray-200 dark:border-gray-700 opacity-50"}`}>
                    <span className="font-mono text-xs flex-1">{c.name}</span>
                    <span className="text-xs flex-1">{isD ? c.value : <span className="text-gray-400 font-mono">sha256(****)</span>}</span>
                    <span className={`px-1.5 py-0.5 rounded text-xs ${isD ? "bg-green-100 dark:bg-green-900/30 text-green-600" : "bg-gray-100 dark:bg-gray-800 text-gray-400"}`}>{isD ? "visible" : "hidden"}</span>
                  </div>
                );
              })}
            </div>
            <div className="mt-3 rounded-lg bg-blue-50 p-3 dark:bg-blue-950/30">
              <p className="text-xs text-blue-700 dark:text-blue-400">
                Verifier sees {simSet.size} disclosed claims + {SAMPLE.length - simSet.size} hash proofs.
              </p>
            </div>
          </div>
        </div>
      )}

      {/* EXCHANGE TAB */}
      {tab === "exchange" && (
        <div className={card}>
          <h2 className="mb-6 flex items-center gap-2 text-sm font-semibold uppercase text-gray-400">
            <ArrowRight className="h-4 w-4" /> Holder ↔ Verifier Exchange Flow
          </h2>
          <div className="space-y-3">
            {[
              { step: 1, actor: "Verifier", action: "Sends authorization request with presentation_definition" },
              { step: 2, actor: "Holder", action: "Wallet receives request, shows consent screen" },
              { step: 3, actor: "Holder", action: "Generates VP token with selected disclosures" },
              { step: 4, actor: "Verifier", action: "Verifies signature, checks disclosures, validates status" },
            ].map(s => (
              <div key={s.step} className="flex items-start gap-4">
                <div className="flex h-8 w-8 items-center justify-center rounded-full bg-purple-600 text-white text-xs font-bold shrink-0">{s.step}</div>
                <div className="flex-1 rounded-lg border p-3 dark:border-gray-700">
                  <div className="flex items-center gap-2">
                    <span className={`px-1.5 py-0.5 rounded text-xs font-bold ${s.actor === "Holder" ? "bg-blue-100 dark:bg-blue-900/30 text-blue-600" : "bg-purple-100 dark:bg-purple-900/30 text-purple-600"}`}>{s.actor}</span>
                    <span className="text-sm font-medium">{s.action}</span>
                  </div>
                </div>
              </div>
            ))}
            <div className="rounded-lg border-2 border-green-300 dark:border-green-700 bg-green-50 dark:bg-green-950/20 p-3">
              <p className="text-xs text-green-700 dark:text-green-400">
                <CheckCircle className="inline h-4 w-4 mr-1" /> Exchange complete. Verifier confirmed claims without seeing hidden data.
              </p>
            </div>
          </div>
        </div>
      )}
    </div>
  );
}
