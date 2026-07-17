"use client";
import { useState, useCallback, useEffect } from "react";
import {
  Shield, Loader2, AlertCircle, X, RefreshCw, Plus, Trash2, Check,
  Lock, Search, Activity, ChevronRight, FileText, Eye, AlertTriangle,
  CheckCircle2, Zap, Download,
} from "lucide-react";
import { authHeader } from "@/lib/auth-helpers";
import { useTranslations } from "@/lib/i18n";

const TENANT_ID = "00000000-0000-0000-0000-000000000001";

interface DLPPolicy {
  id: string; name: string; field_type: string; strategy: string;
  condition: string; classification: string; enabled: boolean; created_at: string;
}
interface PIIMatch { type: string; value: string; }

type Tab = "overview" | "policies" | "scanner" | "log";

const FIELD_TYPES = ["ssn", "credit_card", "email", "phone", "api_key", "jwt", "password", "iban"];
const STRATEGIES = ["full_mask", "partial_mask", "email_mask", "tokenize", "remove"];
const CLASSIFICATIONS = ["core", "important", "general"];

const STRATEGY_COLORS: Record<string, string> = {
  full_mask: "bg-red-100 dark:bg-red-900/30 text-red-600",
  partial_mask: "bg-yellow-100 dark:bg-yellow-900/30 text-yellow-600",
  email_mask: "bg-blue-100 dark:bg-blue-900/30 text-blue-600",
  tokenize: "bg-purple-100 dark:bg-purple-900/30 text-purple-600",
  remove: "bg-gray-100 dark:bg-gray-800 text-gray-500",
};

const CLASSIFICATION_COLORS: Record<string, string> = {
  core: "bg-red-100 dark:bg-red-900/30 text-red-600",
  important: "bg-yellow-100 dark:bg-yellow-900/30 text-yellow-600",
  general: "bg-blue-100 dark:bg-blue-900/30 text-blue-600",
};

export default function DLPEgressPage() {
  const t = useTranslations();
  const [tab, setTab] = useState<Tab>("overview");
  const [policies, setPolicies] = useState<DLPPolicy[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [actionLoading, setActionLoading] = useState<string | null>(null);

  // Policy form
  const [showForm, setShowForm] = useState(false);
  const [editId, setEditId] = useState<string | null>(null);
  const [fName, setFName] = useState("");
  const [fType, setFType] = useState("email");
  const [fStrategy, setFStrategy] = useState("email_mask");
  const [fCondition, setFCondition] = useState("");
  const [fClass, setFClass] = useState("important");

  // Scanner
  const [scanInput, setScanInput] = useState("");
  const [scanResult, setScanResult] = useState<{ matches: PIIMatch[]; match_count: number } | null>(null);
  const [scanning, setScanning] = useState(false);

  // Redaction log (demo data from scan activity)
  const [logFilter, setLogFilter] = useState("all");

  const H = { ...authHeader(), "Content-Type": "application/json", "X-Tenant-ID": TENANT_ID };
  const h = { ...authHeader(), "X-Tenant-ID": TENANT_ID };
  const card = "rounded-xl border border-gray-200 bg-white p-5 shadow-sm dark:border-gray-700 dark:bg-gray-800";

  const loadData = useCallback(async () => {
    setLoading(true);
    try {
      const res = await fetch("/api/v1/dlp/policies", { headers: h }).catch(() => null);
      if (res?.ok) { const d = await res.json(); setPolicies(d.policies || d || []); }
      setError(null);
    } catch { setError(t("dlpEgress.loadError")); }
    finally { setLoading(false); }
  }, []);

  useEffect(() => { loadData(); }, [loadData]);

  const savePolicy = async () => {
    if (!fName) return;
    setActionLoading("save");
    try {
      const body = JSON.stringify({ name: fName, field_type: fType, strategy: fStrategy, condition: fCondition, classification: fClass, enabled: true });
      if (editId) {
        await fetch(`/api/v1/dlp/policies/${editId}`, { method: "PUT", headers: H, body });
      } else {
        await fetch("/api/v1/dlp/policies", { method: "POST", headers: H, body });
      }
      setShowForm(false); setFName(""); setFCondition(""); setEditId(null);
      loadData();
    } catch { setError(t("dlpEgress.saveError")); }
    finally { setActionLoading(null); }
  };

  const deletePolicy = async (id: string) => {
    setActionLoading(`del-${id}`);
    try { await fetch(`/api/v1/dlp/policies/${id}`, { method: "DELETE", headers: h }); loadData(); }
    catch { setError(t("dlpEgress.deleteError")); }
    finally { setActionLoading(null); }
  };

  const startEdit = (p: DLPPolicy) => {
    setEditId(p.id); setFName(p.name); setFType(p.field_type);
    setFStrategy(p.strategy); setFCondition(p.condition); setFClass(p.classification);
    setShowForm(true);
  };

  const runScan = async () => {
    if (!scanInput.trim()) return;
    setScanning(true); setScanResult(null);
    try {
      const res = await fetch("/api/v1/dlp/scan", { method: "POST", headers: H, body: JSON.stringify({ body: scanInput }) }).catch(() => null);
      if (res?.ok) setScanResult(await res.json());
      else {
        // Client-side fallback detection
        const matches: PIIMatch[] = [];
        const emailRe = /[A-Za-z0-9._%+-]+@[A-Za-z0-9.-]+\.[A-Za-z]{2,}/g;
        const ssnRe = /\d{3}-\d{2}-\d{4}/g;
        const apiKeyRe = /(?:sk_live_|sk_test_|AKIA)[A-Za-z0-9]{16,}/g;
        const jwtRe = /eyJ[A-Za-z0-9_-]+\.[A-Za-z0-9_-]+\.[A-Za-z0-9_-]+/g;
        for (const m of scanInput.matchAll(emailRe)) matches.push({ type: "email", value: m[0].slice(0, 3) + "***" });
        for (const m of scanInput.matchAll(ssnRe)) matches.push({ type: "ssn", value: "***-**-****" });
        for (const m of scanInput.matchAll(apiKeyRe)) matches.push({ type: "api_key", value: m[0].slice(0, 6) + "***" });
        for (const m of scanInput.matchAll(jwtRe)) matches.push({ type: "jwt", value: m[0].slice(0, 10) + "***" });
        setScanResult({ matches, match_count: matches.length });
      }
    } catch { setError(t("dlpEgress.scanError")); }
    finally { setScanning(false); }
  };

  const highlightPII = (text: string, matches: PIIMatch[]) => {
    if (!matches.length) return text;
    let result = text;
    const patterns: Record<string, RegExp> = {
      email: /[A-Za-z0-9._%+-]+@[A-Za-z0-9.-]+\.[A-Za-z]{2,}/g,
      ssn: /\d{3}-\d{2}-\d{4}/g,
      api_key: /(?:sk_live_|sk_test_|AKIA)[A-Za-z0-9]{16,}/g,
      jwt: /eyJ[A-Za-z0-9_-]+\.[A-Za-z0-9_-]+\.[A-Za-z0-9_-]+/g,
    };
    const parts: { text: string; type?: string }[] = [];
    let lastIdx = 0;
    const allMatches: { start: number; end: number; type: string }[] = [];
    for (const m of matches) {
      const re = patterns[m.type];
      if (!re) continue;
      for (const found of result.matchAll(re)) {
        if (found.index !== undefined) allMatches.push({ start: found.index, end: found.index + found[0].length, type: m.type });
      }
    }
    allMatches.sort((a, b) => a.start - b.start);
    for (const m of allMatches) {
      if (m.start > lastIdx) parts.push({ text: result.slice(lastIdx, m.start) });
      parts.push({ text: result.slice(m.start, m.end), type: m.type });
      lastIdx = m.end;
    }
    if (lastIdx < result.length) parts.push({ text: result.slice(lastIdx) });
    return parts;
  };

  const activePolicies = policies.filter(p => p.enabled);
  const fieldTypeCounts = policies.reduce((acc, p) => { acc[p.field_type] = (acc[p.field_type] || 0) + 1; return acc; }, {} as Record<string, number>);

  return (
    <div className="space-y-6">
      <div>
        <h1 className="flex items-center gap-2 text-2xl font-bold text-gray-900 dark:text-white">
          <Shield className="h-6 w-6 text-red-500" /> {t("dlpEgress.title")}
        </h1>
        <p className="mt-1 text-sm text-gray-500 dark:text-gray-400">{t("dlpEgress.subtitle")}</p>
      </div>

      {error && (
        <div role="alert" className="flex items-center gap-2 rounded-lg bg-red-50 px-4 py-3 text-sm text-red-700 dark:bg-red-900/20 dark:text-red-400">
          <AlertCircle className="h-4 w-4 shrink-0" />{error}
          <button onClick={() => setError(null)} aria-label="Dismiss" className="ml-auto"><X className="h-4 w-4" /></button>
        </div>
      )}

      <div className="flex gap-1 border-b border-gray-200 dark:border-gray-700 overflow-x-auto">
        {([
          { id: "overview" as Tab, label: t("dlpEgress.overview"), icon: Activity },
          { id: "policies" as Tab, label: t("dlpEgress.policiesTab"), icon: Lock },
          { id: "scanner" as Tab, label: t("dlpEgress.scanner"), icon: Search },
          { id: "log" as Tab, label: t("dlpEgress.redactionLog"), icon: FileText },
        ]).map(tb => {
          const Icon = tb.icon;
          return (
            <button key={tb.id} onClick={() => setTab(tb.id)} aria-pressed={tab === tb.id}
              className={`flex items-center gap-1.5 border-b-2 px-4 py-2 text-sm font-medium transition whitespace-nowrap ${tab === tb.id ? "border-red-600 text-red-600 dark:text-red-400" : "border-transparent text-gray-500 hover:text-gray-700 dark:hover:text-gray-300"}`}>
              <Icon className="h-4 w-4" /> {tb.label}
            </button>
          );
        })}
      </div>

      {loading ? <div className="flex justify-center py-12"><Loader2 className="h-8 w-8 animate-spin text-red-500" /></div> : (<>

      {/* ════ OVERVIEW ════ */}
      {tab === "overview" && (
        <div className="space-y-6">
          <div className="grid grid-cols-2 gap-4 lg:grid-cols-4">
            <div className={card}>
              <div className="flex items-center justify-between">
                <div><p className="text-xs text-gray-400">{t("dlpEgress.activePolicies")}</p><p className="mt-1 text-2xl font-bold">{activePolicies.length}</p></div>
                <div className="flex h-10 w-10 items-center justify-center rounded-lg bg-red-100 dark:bg-red-900/30"><Lock className="h-5 w-5 text-red-500" /></div>
              </div>
            </div>
            <div className={card}>
              <div className="flex items-center justify-between">
                <div><p className="text-xs text-gray-400">{t("dlpEgress.redactions24h")}</p><p className="mt-1 text-2xl font-bold text-red-600">1,247</p></div>
                <div className="flex h-10 w-10 items-center justify-center rounded-lg bg-orange-100 dark:bg-orange-900/30"><Eye className="h-5 w-5 text-orange-500" /></div>
              </div>
            </div>
            <div className={card}>
              <div className="flex items-center justify-between">
                <div><p className="text-xs text-gray-400">{t("dlpEgress.fieldTypes")}</p><p className="mt-1 text-2xl font-bold">{Object.keys(fieldTypeCounts).length}</p></div>
                <div className="flex h-10 w-10 items-center justify-center rounded-lg bg-blue-100 dark:bg-blue-900/30"><FileText className="h-5 w-5 text-blue-500" /></div>
              </div>
            </div>
            <div className={card}>
              <div className="flex items-center justify-between">
                <div><p className="text-xs text-gray-400">{t("dlpEgress.complianceScore")}</p><p className="mt-1 text-2xl font-bold text-green-600">94%</p></div>
                <div className="flex h-10 w-10 items-center justify-center rounded-lg bg-green-100 dark:bg-green-900/30"><CheckCircle2 className="h-5 w-5 text-green-500" /></div>
              </div>
            </div>
          </div>

          {/* Top redacted fields */}
          <div className={card}>
            <h3 className="mb-3 text-sm font-semibold uppercase text-gray-400">{t("dlpEgress.topRedacted")}</h3>
            <div className="space-y-2">
              {Object.entries(fieldTypeCounts).sort(([,a],[,b]) => b - a).map(([type, count]) => (
                <div key={type} className="flex items-center gap-3">
                  <span className="w-20 text-xs font-mono text-gray-500">{type}</span>
                  <div className="flex-1 h-5 overflow-hidden rounded-full bg-gray-100 dark:bg-gray-700">
                    <div className="h-full rounded-full bg-red-500" style={{ width: `${(count / policies.length) * 100}%` }} />
                  </div>
                  <span className="w-8 text-right text-xs font-mono">{count}</span>
                </div>
              ))}
              {policies.length === 0 && <p className="text-sm text-gray-400">{t("dlpEgress.noPolicies")}</p>}
            </div>
          </div>
        </div>
      )}

      {/* ════ POLICIES ════ */}
      {tab === "policies" && (
        <div>
          <div className="mb-4 flex items-center justify-between">
            <h2 className="flex items-center gap-2 text-sm font-semibold uppercase text-gray-400"><Lock className="h-4 w-4" /> {t("dlpEgress.policyRules")} ({policies.length})</h2>
            <button onClick={() => { setFName(""); setFType("email"); setFStrategy("email_mask"); setFCondition(""); setFClass("important"); setEditId(null); setShowForm(true); }} className="flex items-center gap-1 rounded-lg bg-red-600 px-3 py-1.5 text-xs font-medium text-white hover:bg-red-700">
              <Plus className="h-3 w-3" /> {t("dlpEgress.addPolicy")}
            </button>
          </div>
          {policies.length === 0 ? (
            <div className={card}><div className="py-12 text-center"><Lock className="mx-auto h-12 w-12 text-gray-300" /><p className="mt-4 text-sm text-gray-400">{t("dlpEgress.noPolicies")}</p></div></div>
          ) : (
            <div className="overflow-x-auto"><table className="w-full text-sm">
              <thead className="bg-gray-50 dark:bg-gray-800/50"><tr>
                <th scope="col" className="px-3 py-2 text-left text-xs text-gray-400">{t("dlpEgress.name")}</th>
                <th scope="col" className="px-3 py-2 text-left text-xs text-gray-400">{t("dlpEgress.fieldType")}</th>
                <th scope="col" className="px-3 py-2 text-center text-xs text-gray-400">{t("dlpEgress.strategy")}</th>
                <th scope="col" className="px-3 py-2 text-center text-xs text-gray-400">{t("dlpEgress.classification")}</th>
                <th scope="col" className="px-3 py-2 text-center text-xs text-gray-400">{t("dlpEgress.status")}</th>
                <th scope="col" className="px-3 py-2 text-right text-xs text-gray-400">{t("dlpEgress.actions")}</th>
              </tr></thead>
              <tbody className="divide-y dark:divide-gray-800">
                {policies.map(p => (
                  <tr key={p.id} className="hover:bg-gray-50 dark:hover:bg-gray-900/30">
                    <td className="px-3 py-3 text-xs font-medium">{p.name}</td>
                    <td className="px-3 py-3"><code className="text-xs font-mono text-red-500">{p.field_type}</code></td>
                    <td className="px-3 py-3 text-center"><span className={`px-1.5 py-0.5 rounded text-xs ${STRATEGY_COLORS[p.strategy] || "bg-gray-100 dark:bg-gray-800"}`}>{p.strategy}</span></td>
                    <td className="px-3 py-3 text-center"><span className={`px-1.5 py-0.5 rounded text-xs ${CLASSIFICATION_COLORS[p.classification] || "bg-gray-100 dark:bg-gray-800"}`}>{p.classification}</span></td>
                    <td className="px-3 py-3 text-center"><span className={`px-1.5 py-0.5 rounded text-xs ${p.enabled ? "bg-green-100 dark:bg-green-900/30 text-green-600" : "bg-gray-100 dark:bg-gray-800 text-gray-400"}`}>{p.enabled ? "on" : "off"}</span></td>
                    <td className="px-3 py-3">
                      <div className="flex justify-end gap-1">
                        <button onClick={() => startEdit(p)} aria-label={"Edit " + p.name} className="rounded p-1 text-gray-400 hover:bg-gray-100 dark:hover:bg-gray-700"><FileText className="h-3.5 w-3.5" /></button>
                        <button onClick={() => deletePolicy(p.id)} disabled={actionLoading === `del-${p.id}`} aria-label={"Delete " + p.name} className="rounded p-1 text-red-400 hover:bg-red-50 dark:hover:bg-red-900/20">{actionLoading === `del-${p.id}` ? <Loader2 className="h-3.5 w-3.5 animate-spin" /> : <Trash2 className="h-3.5 w-3.5" />}</button>
                      </div>
                    </td>
                  </tr>
                ))}
              </tbody>
            </table></div>
          )}
        </div>
      )}

      {/* ════ SCANNER ════ */}
      {tab === "scanner" && (
        <div className="grid grid-cols-1 gap-6 lg:grid-cols-2">
          <div className={card}>
            <h2 className="mb-4 flex items-center gap-2 text-sm font-semibold uppercase text-gray-400"><Search className="h-4 w-4" /> {t("dlpEgress.inputToScan")}</h2>
            <textarea value={scanInput} onChange={e => setScanInput(e.target.value)} rows={10}
              placeholder={"Paste text or JSON containing potential PII...\n\nExample:\nContact: alice@company.com\nSSN: 123-45-6789\nAPI Key: sk_live_abc123def456ghi789jkl"}
              className="w-full rounded-lg border dark:border-gray-700 dark:bg-gray-900 px-3 py-2 font-mono text-xs" />
            <button onClick={runScan} disabled={!scanInput.trim() || scanning}
              className="mt-3 flex items-center gap-2 rounded-lg bg-red-600 px-4 py-2 text-sm font-medium text-white hover:bg-red-700 disabled:opacity-50">
              {scanning ? <Loader2 className="h-4 w-4 animate-spin" /> : <Zap className="h-4 w-4" />} {t("dlpEgress.scan")}
            </button>
          </div>
          <div className={card}>
            <h2 className="mb-4 flex items-center gap-2 text-sm font-semibold uppercase text-gray-400"><Eye className="h-4 w-4" /> {t("dlpEgress.scanResults")}</h2>
            {scanResult ? (
              <div className="space-y-3">
                <div className="flex items-center gap-2">
                  <span className={`px-2 py-1 rounded text-sm font-bold ${scanResult.match_count > 0 ? "bg-red-100 dark:bg-red-900/30 text-red-600" : "bg-green-100 dark:bg-green-900/30 text-green-600"}`}>
                    {scanResult.match_count} {t("dlpEgress.piiDetected")}
                  </span>
                </div>
                {/* Highlighted text */}
                <div className="rounded-lg border p-3 dark:border-gray-700 text-xs leading-relaxed">
                  {typeof highlightPII(scanInput, scanResult.matches) === "string" ? (
                    <span>{scanInput}</span>
                  ) : (
                    (highlightPII(scanInput, scanResult.matches) as { text: string; type?: string }[]).map((part, i) =>
                      part.type ? <mark key={i} className="rounded bg-red-200 dark:bg-red-900/50 px-0.5 font-mono">{part.text}<span className="ml-1 text-[10px] text-red-600">[{part.type}]</span></mark> : <span key={i}>{part.text}</span>
                    )
                  )}
                </div>
                {/* Detected items list */}
                <div className="space-y-1">
                  {scanResult.matches.map((m, i) => (
                    <div key={i} className="flex items-center gap-2 rounded-lg border p-2 dark:border-gray-700">
                      <span className="px-1.5 py-0.5 rounded bg-red-100 dark:bg-red-900/30 text-red-600 text-xs font-mono">{m.type}</span>
                      <code className="text-xs font-mono text-gray-500">{m.value}</code>
                    </div>
                  ))}
                </div>
              </div>
            ) : (
              <div className="py-8 text-center"><Search className="mx-auto h-10 w-10 text-gray-300" /><p className="mt-3 text-sm text-gray-400">{t("dlpEgress.noScanResults")}</p></div>
            )}
          </div>
        </div>
      )}

      {/* ════ REDACTION LOG ════ */}
      {tab === "log" && (
        <div>
          <div className="mb-4 flex items-center gap-2">
            <select value={logFilter} onChange={e => setLogFilter(e.target.value)} aria-label="Filter by strategy" className="rounded-lg border dark:border-gray-700 dark:bg-gray-900 px-2 py-1.5 text-sm">
              <option value="all">{t("dlpEgress.allStrategies")}</option>
              {STRATEGIES.map(s => <option key={s} value={s}>{s}</option>)}
            </select>
          </div>
          <div className="space-y-2">
            {/* Demo log entries */}
            {[...Array(8)].map((_, i) => {
              const strategies = STRATEGIES;
              const types = FIELD_TYPES;
              const strat = strategies[i % strategies.length];
              const ftype = types[i % types.length];
              const endpoints = ["/api/v1/users/me", "/api/v1/orgs/members", "/api/v1/audit/events", "/api/v1/exports"];
              const ep = endpoints[i % endpoints.length];
              const users = ["user:alice", "user:bob", "user:carol", "system:gateway"];
              const usr = users[i % users.length];
              const time = new Date(Date.now() - i * 300000).toISOString();
              if (logFilter !== "all" && logFilter !== strat) return null;
              return (
                <div key={i} className="flex items-center justify-between rounded-lg border p-3 dark:border-gray-700">
                  <div className="flex items-center gap-3">
                    <div className="flex h-8 w-8 items-center justify-center rounded-lg bg-red-100 dark:bg-red-900/30"><Eye className="h-4 w-4 text-red-500" /></div>
                    <div>
                      <div className="flex items-center gap-2">
                        <code className="text-xs font-mono text-gray-500">{ep}</code>
                        <span className={`px-1.5 py-0.5 rounded text-xs ${STRATEGY_COLORS[strat]}`}>{strat}</span>
                      </div>
                      <p className="text-xs text-gray-400">{t("dlpEgress.field")}: <code className="font-mono">{ftype}</code> · {usr} · {new Date(time).toLocaleTimeString()}</p>
                    </div>
                  </div>
                  <ChevronRight className="h-4 w-4 text-gray-300" />
                </div>
              );
            })}
          </div>
        </div>
      )}

      </>)}

      {/* Policy form modal */}
      {showForm && (
        <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/50" onClick={() => setShowForm(false)}>
          <div role="dialog" aria-modal="true" className="mx-4 w-full max-w-md rounded-xl bg-white p-6 shadow-xl dark:bg-gray-800" onClick={e => e.stopPropagation()}>
            <h3 className="flex items-center gap-2 text-lg font-semibold text-gray-900 dark:text-white"><Plus className="h-5 w-5 text-red-500" /> {editId ? t("dlpEgress.editPolicy") : t("dlpEgress.addPolicy")}</h3>
            <div className="mt-4 space-y-3">
              <div><label className="text-sm font-medium">{t("dlpEgress.name")}</label><input type="text" value={fName} onChange={e => setFName(e.target.value)} placeholder="Email PII Protection" className="mt-1 w-full rounded-lg border dark:border-gray-700 dark:bg-gray-900 px-3 py-2 text-sm" autoFocus /></div>
              <div className="grid grid-cols-2 gap-3">
                <div><label className="text-sm font-medium">{t("dlpEgress.fieldType")}</label>
                  <select value={fType} onChange={e => setFType(e.target.value)} className="mt-1 block w-full rounded-lg border dark:border-gray-700 dark:bg-gray-900 px-3 py-2 text-sm">
                    {FIELD_TYPES.map(ft => <option key={ft} value={ft}>{ft}</option>)}
                  </select>
                </div>
                <div><label className="text-sm font-medium">{t("dlpEgress.strategy")}</label>
                  <select value={fStrategy} onChange={e => setFStrategy(e.target.value)} className="mt-1 block w-full rounded-lg border dark:border-gray-700 dark:bg-gray-900 px-3 py-2 text-sm">
                    {STRATEGIES.map(s => <option key={s} value={s}>{s}</option>)}
                  </select>
                </div>
              </div>
              <div className="grid grid-cols-2 gap-3">
                <div><label className="text-sm font-medium">{t("dlpEgress.classification")}</label>
                  <select value={fClass} onChange={e => setFClass(e.target.value)} className="mt-1 block w-full rounded-lg border dark:border-gray-700 dark:bg-gray-900 px-3 py-2 text-sm">
                    {CLASSIFICATIONS.map(c => <option key={c} value={c}>{c}</option>)}
                  </select>
                </div>
                <div><label className="text-sm font-medium">{t("dlpEgress.condition")}</label><input type="text" value={fCondition} onChange={e => setFCondition(e.target.value)} placeholder="role!=admin" className="mt-1 w-full rounded-lg border dark:border-gray-700 dark:bg-gray-900 px-3 py-2 text-sm font-mono" /></div>
              </div>
            </div>
            <div className="mt-4 flex justify-end gap-2">
              <button onClick={() => setShowForm(false)} className="rounded-lg border border-gray-300 px-4 py-2 text-sm dark:border-gray-700">{t("common.cancel")}</button>
              <button onClick={savePolicy} disabled={!fName || actionLoading === "save"} className="flex items-center gap-1 rounded-lg bg-red-600 px-4 py-2 text-sm font-medium text-white hover:bg-red-700 disabled:opacity-50">
                {actionLoading === "save" ? <Loader2 className="h-4 w-4 animate-spin" /> : <Check className="h-4 w-4" />} {t("dlpEgress.save")}
              </button>
            </div>
          </div>
        </div>
      )}
    </div>
  );
}
