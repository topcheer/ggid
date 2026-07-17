"use client";
import { useState, useCallback, useEffect } from "react";
import {
  Crosshair, Loader2, AlertCircle, X, RefreshCw, Shield, Zap,
  Activity, ChevronRight, CheckCircle2, XCircle, AlertTriangle,
  Clock, Ban, Lock, Radar, FlaskConical, Play,
} from "lucide-react";
import { authHeader } from "@/lib/auth-helpers";
import { useTranslations } from "@/lib/i18n";

const TENANT_ID = "00000000-0000-0000-0000-000000000001";

/* ─── MITRE ATT&CK Data (from KB-192: 8 new + 7 original = 15 rules) ─── */
interface DetectionRule { id: string; name: string; mitre_technique: string; mitre_tactic: string; severity: "critical" | "high" | "medium" | "low"; action: string; enabled: boolean; }
interface Detection { id: string; rule_id: string; rule_name: string; user_id: string; severity: string; mitre_technique: string; action: string; timestamp: string; status: string; }

const MITRE_TACTICS = ["Initial Access", "Credential Access", "Persistence", "Privilege Escalation", "Defense Evasion", "Discovery"];

const RULES: DetectionRule[] = [
  { id: "r-001", name: "Brute Force Login Burst", mitre_technique: "T1110", mitre_tactic: "Credential Access", severity: "high", action: "block_ip", enabled: true },
  { id: "r-002", name: "Impossible Travel", mitre_technique: "T1078", mitre_tactic: "Initial Access", severity: "high", action: "step_up_mfa", enabled: true },
  { id: "r-003", name: "MFA Fatigue Attack", mitre_technique: "T1621", mitre_tactic: "Credential Access", severity: "critical", action: "block", enabled: true },
  { id: "r-004", name: "Consent Phishing", mitre_technique: "T1562", mitre_tactic: "Defense Evasion", severity: "high", action: "revoke_consent", enabled: true },
  { id: "r-005", name: "Token Theft (Refresh Replay)", mitre_technique: "T1528", mitre_tactic: "Credential Access", severity: "critical", action: "revoke_tokens", enabled: true },
  { id: "r-006", name: "Privilege Escalation via Role Abuse", mitre_technique: "T1078.004", mitre_tactic: "Privilege Escalation", severity: "high", action: "alert", enabled: true },
  { id: "r-007", name: "Anomalous API Usage", mitre_technique: "T1190", mitre_tactic: "Initial Access", severity: "medium", action: "rate_limit", enabled: true },
  { id: "r-008", name: "Credential Stuffing", mitre_technique: "T1110.004", mitre_tactic: "Credential Access", severity: "high", action: "block_ip", enabled: true },
  { id: "r-009", name: "Session Hijacking", mitre_technique: "T1185", mitre_tactic: "Credential Access", severity: "critical", action: "terminate_session", enabled: true },
  { id: "r-010", name: "New Device Persistence", mitre_technique: "T1136", mitre_tactic: "Persistence", severity: "medium", action: "step_up_mfa", enabled: true },
  { id: "r-011", name: "Lateral Movement Detection", mitre_technique: "T1021", mitre_tactic: "Discovery", severity: "high", action: "alert", enabled: true },
  { id: "r-012", name: "Synthetic Identity Fraud", mitre_technique: "T1136.001", mitre_tactic: "Persistence", severity: "medium", action: "block", enabled: false },
  { id: "r-013", name: "Tor/VPN Egress Detection", mitre_technique: "T1090", mitre_tactic: "Defense Evasion", severity: "low", action: "flag", enabled: true },
  { id: "r-014", name: "Pass-the-Token", mitre_technique: "T1528", mitre_tactic: "Credential Access", severity: "critical", action: "revoke_tokens", enabled: true },
  { id: "r-015", name: "OAuth App Abuse", mitre_technique: "T1562.001", mitre_tactic: "Defense Evasion", severity: "high", action: "revoke_app", enabled: true },
];

const ATTACK_SCENARIOS = [
  { id: "mfa_fatigue", name: "MFA Fatigue Attack", technique: "T1621", desc: "Send repeated MFA push notifications to overwhelm user into approving" },
  { id: "consent_phish", name: "Consent Phishing", technique: "T1562", desc: "Trick user into granting OAuth consent to malicious app" },
  { id: "token_theft", name: "Token Theft & Replay", technique: "T1528", desc: "Steal refresh token and replay from different location" },
  { id: "credential_stuff", name: "Credential Stuffing", technique: "T1110.004", desc: "Automated login attempts using breached credentials" },
  { id: "session_hijack", name: "Session Hijacking", technique: "T1185", desc: "Hijack active session cookie/token from victim" },
];

type Tab = "matrix" | "rules" | "detections" | "simulation";

const SEV_CFG: Record<string, { color: string; bg: string }> = {
  critical: { color: "text-red-600", bg: "bg-red-100 dark:bg-red-900/30" },
  high: { color: "text-orange-600", bg: "bg-orange-100 dark:bg-orange-900/30" },
  medium: { color: "text-yellow-600", bg: "bg-yellow-100 dark:bg-yellow-900/30" },
  low: { color: "text-blue-600", bg: "bg-blue-100 dark:bg-blue-900/30" },
};

export default function MITREPage() {
  const t = useTranslations();
  const [tab, setTab] = useState<Tab>("matrix");
  const [detections, setDetections] = useState<Detection[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [rules, setRules] = useState(RULES);

  // Filters
  const [detSev, setDetSev] = useState("all");
  const [simScenario, setSimScenario] = useState("");
  const [simResult, setSimResult] = useState<{ fired: boolean; rule: string; action: string } | null>(null);
  const [simulating, setSimulating] = useState(false);

  const h = { ...authHeader(), "X-Tenant-ID": TENANT_ID };
  const card = "rounded-xl border border-gray-200 bg-white p-5 shadow-sm dark:border-gray-700 dark:bg-gray-800";

  const loadDetections = useCallback(async () => {
    setLoading(true);
    try {
      const res = await fetch("/api/v1/audit/itdr/detections?page_size=50", { headers: h }).catch(() => null);
      if (res?.ok) { const d = await res.json(); setDetections(d.detections || []); }
    } catch { setError(t("mitre.loadError")); }
    finally { setLoading(false); }
  }, []);

  useEffect(() => { loadDetections(); }, [loadDetections]);

  const toggleRule = (id: string) => {
    setRules(prev => prev.map(r => r.id === id ? { ...r, enabled: !r.enabled } : r));
  };

  const runSimulation = () => {
    if (!simScenario) return;
    setSimulating(true); setSimResult(null);
    setTimeout(() => {
      const scenario = ATTACK_SCENARIOS.find(s => s.id === simScenario);
      const matchingRules = rules.filter(r => r.enabled && scenario && r.mitre_technique.includes(scenario.technique.slice(0, 4)));
      const fired = matchingRules.length > 0;
      setSimResult({ fired, rule: fired ? matchingRules[0].name : "No matching rule", action: fired ? matchingRules[0].action : "none" });
      setSimulating(false);
    }, 1500);
  };

  // Build MITRE coverage matrix
  const allTechniques = [...new Set(RULES.map(r => `${r.mitre_technique}|${r.mitre_tactic}`))];
  const coverageByTactic = MITRE_TACTICS.map(tactic => {
    const techs = allTechniques.filter(t => t.split("|")[1] === tactic);
    return { tactic, techniques: techs.map(t => {
      const [tech, _] = t.split("|");
      const rule = RULES.find(r => r.mitre_technique === tech && r.mitre_tactic === tactic);
      return { technique: tech, covered: !!rule?.enabled, ruleName: rule?.name || "No detection", tactic };
    })};
  });

  const enabledRules = rules.filter(r => r.enabled);
  const coveragePct = Math.round((enabledRules.length / rules.length) * 100);

  const filteredDetections = detSev === "all" ? detections : detections.filter(d => d.severity === detSev);

  return (
    <div className="space-y-6">
      <div className="flex items-center gap-2 rounded-lg bg-amber-50 dark:bg-amber-900/20 px-4 py-2 text-xs text-amber-600 dark:text-amber-400">
        <AlertTriangle className="h-3.5 w-3.5 shrink-0" /> DEMO DATA — MITRE technique mappings from KB-192. Live detection data from /api/v1/audit/itdr/detections.
      </div>

      <div>
        <h1 className="flex items-center gap-2 text-2xl font-bold text-gray-900 dark:text-white">
          <Crosshair className="h-6 w-6 text-orange-500" /> {t("mitre.title")}
        </h1>
        <p className="mt-1 text-sm text-gray-500 dark:text-gray-400">{t("mitre.subtitle")}</p>
      </div>

      {error && (
        <div role="alert" className="flex items-center gap-2 rounded-lg bg-red-50 px-4 py-3 text-sm text-red-700 dark:bg-red-900/20 dark:text-red-400">
          <AlertCircle className="h-4 w-4 shrink-0" />{error}
          <button onClick={() => setError(null)} aria-label="Dismiss" className="ml-auto"><X className="h-4 w-4" /></button>
        </div>
      )}

      <div className="flex gap-1 border-b border-gray-200 dark:border-gray-700 overflow-x-auto">
        {([
          { id: "matrix" as Tab, label: t("mitre.coverage"), icon: Crosshair },
          { id: "rules" as Tab, label: `${t("mitre.rules")} (${rules.length})`, icon: Shield },
          { id: "detections" as Tab, label: t("mitre.recentDetections"), icon: Activity },
          { id: "simulation" as Tab, label: t("mitre.simulation"), icon: FlaskConical },
        ]).map(tb => {
          const Icon = tb.icon;
          return (
            <button key={tb.id} onClick={() => setTab(tb.id)} aria-pressed={tab === tb.id}
              className={`flex items-center gap-1.5 border-b-2 px-4 py-2 text-sm font-medium transition whitespace-nowrap ${tab === tb.id ? "border-orange-600 text-orange-600 dark:text-orange-400" : "border-transparent text-gray-500 hover:text-gray-700 dark:hover:text-gray-300"}`}>
              <Icon className="h-4 w-4" /> {tb.label}
            </button>
          );
        })}
      </div>

      {/* ════ COVERAGE MATRIX ════ */}
      {tab === "matrix" && (
        <div className="space-y-6">
          <div className={card}>
            <div className="flex items-center justify-between mb-4">
              <h3 className="text-sm font-semibold uppercase text-gray-400">{t("mitre.coverageScore")}</h3>
              <span className={`text-2xl font-bold ${coveragePct >= 70 ? "text-green-600" : coveragePct >= 50 ? "text-yellow-600" : "text-red-600"}`}>{coveragePct}%</span>
            </div>
            <div className="h-3 overflow-hidden rounded-full bg-gray-200 dark:bg-gray-700">
              <div className={`h-full rounded-full ${coveragePct >= 70 ? "bg-green-500" : "bg-yellow-500"}`} style={{ width: `${coveragePct}%` }} />
            </div>
            <p className="mt-2 text-xs text-gray-400">{enabledRules.length}/{rules.length} {t("mitre.rulesCovering")}</p>
          </div>

          <div className="space-y-4">
            {coverageByTactic.map(col => (
              <div key={col.tactic} className={card}>
                <h3 className="mb-3 text-sm font-semibold flex items-center gap-2"><Radar className="h-4 w-4 text-orange-400" /> {col.tactic}</h3>
                <div className="grid grid-cols-1 gap-2 sm:grid-cols-2 lg:grid-cols-3">
                  {col.techniques.map(tech => (
                    <div key={tech.technique} className={`flex items-center gap-2 rounded-lg border p-2 ${tech.covered ? "border-green-200 dark:border-green-800 bg-green-50 dark:bg-green-950/20" : "border-red-200 dark:border-red-800 bg-red-50 dark:bg-red-950/20"}`}>
                      {tech.covered ? <CheckCircle2 className="h-4 w-4 text-green-500 shrink-0" /> : <XCircle className="h-4 w-4 text-red-500 shrink-0" />}
                      <div className="min-w-0">
                        <code className="text-xs font-mono text-gray-600 dark:text-gray-300">{tech.technique}</code>
                        <p className="text-xs text-gray-400 truncate">{tech.ruleName}</p>
                      </div>
                    </div>
                  ))}
                  {col.techniques.length === 0 && <p className="text-xs text-gray-300">No techniques mapped</p>}
                </div>
              </div>
            ))}
          </div>
        </div>
      )}

      {/* ════ RULES ════ */}
      {tab === "rules" && (
        <div className="overflow-x-auto"><table className="w-full text-sm">
          <thead className="bg-gray-50 dark:bg-gray-800/50"><tr>
            <th scope="col" className="px-3 py-2 text-left text-xs text-gray-400">{t("mitre.ruleName")}</th>
            <th scope="col" className="px-3 py-2 text-center text-xs text-gray-400">MITRE</th>
            <th scope="col" className="px-3 py-2 text-left text-xs text-gray-400">{t("mitre.tactic")}</th>
            <th scope="col" className="px-3 py-2 text-center text-xs text-gray-400">{t("mitre.severity")}</th>
            <th scope="col" className="px-3 py-2 text-center text-xs text-gray-400">{t("mitre.action")}</th>
            <th scope="col" className="px-3 py-2 text-center text-xs text-gray-400">{t("mitre.enabled")}</th>
          </tr></thead>
          <tbody className="divide-y dark:divide-gray-800">
            {rules.map(r => {
              const cfg = SEV_CFG[r.severity] || SEV_CFG.medium;
              return (
                <tr key={r.id} className="hover:bg-gray-50 dark:hover:bg-gray-900/30">
                  <td className="px-3 py-3 text-xs font-medium">{r.name}</td>
                  <td className="px-3 py-3 text-center"><code className="text-xs font-mono text-orange-500">{r.mitre_technique}</code></td>
                  <td className="px-3 py-3 text-xs text-gray-500">{r.mitre_tactic}</td>
                  <td className="px-3 py-3 text-center"><span className={`px-1.5 py-0.5 rounded text-xs font-medium ${cfg.bg} ${cfg.color}`}>{r.severity}</span></td>
                  <td className="px-3 py-3 text-center"><code className="text-xs font-mono">{r.action}</code></td>
                  <td className="px-3 py-3 text-center">
                    <button onClick={() => toggleRule(r.id)} aria-label={"Toggle " + r.name} aria-pressed={r.enabled}
                      className={`relative h-5 w-9 rounded-full transition ${r.enabled ? "bg-green-500" : "bg-gray-300 dark:bg-gray-700"}`}>
                      <span className={`absolute top-0.5 h-4 w-4 rounded-full bg-white transition ${r.enabled ? "left-4" : "left-0.5"}`} />
                    </button>
                  </td>
                </tr>
              );
            })}
          </tbody>
        </table></div>
      )}

      {/* ════ DETECTIONS ════ */}
      {tab === "detections" && (
        <div>
          <div className="mb-4">
            <select value={detSev} onChange={e => setDetSev(e.target.value)} aria-label="Filter severity" className="rounded-lg border dark:border-gray-700 dark:bg-gray-900 px-2 py-1.5 text-sm">
              <option value="all">{t("mitre.allSeverities")}</option>
              <option value="critical">Critical</option><option value="high">High</option><option value="medium">Medium</option><option value="low">Low</option>
            </select>
          </div>
          {loading ? <div className="flex justify-center py-8"><Loader2 className="h-8 w-8 animate-spin text-orange-500" /></div> :
          filteredDetections.length === 0 ? (
            <div className={card}><div className="py-12 text-center"><CheckCircle2 className="mx-auto h-12 w-12 text-green-300" /><p className="mt-4 text-sm text-gray-400">{t("mitre.noDetections")}</p></div></div>
          ) : (
            <div className="space-y-2">
              {filteredDetections.map(d => {
                const cfg = SEV_CFG[d.severity] || SEV_CFG.medium;
                return (
                  <div key={d.id} className="flex items-center justify-between rounded-lg border p-3 dark:border-gray-700">
                    <div className="flex items-center gap-3">
                      <div className={`flex h-8 w-8 items-center justify-center rounded-lg ${cfg.bg}`}><AlertTriangle className={`h-4 w-4 ${cfg.color}`} /></div>
                      <div>
                        <div className="flex items-center gap-2">
                          <span className="text-xs font-medium">{d.rule_name}</span>
                          <code className="text-xs font-mono text-orange-500">{d.mitre_technique || "—"}</code>
                          <span className={`px-1.5 py-0.5 rounded text-xs ${cfg.bg} ${cfg.color}`}>{d.severity}</span>
                        </div>
                        <p className="text-xs text-gray-400">{d.user_id} · {new Date(d.timestamp).toLocaleString()}</p>
                      </div>
                    </div>
                    <span className="px-1.5 py-0.5 rounded text-xs bg-gray-100 dark:bg-gray-700 font-mono">{d.action}</span>
                  </div>
                );
              })}
            </div>
          )}
        </div>
      )}

      {/* ════ SIMULATION ════ */}
      {tab === "simulation" && (
        <div className="grid grid-cols-1 gap-6 lg:grid-cols-2">
          <div className={card}>
            <h2 className="mb-4 flex items-center gap-2 text-sm font-semibold uppercase text-gray-400"><FlaskConical className="h-4 w-4" /> {t("mitre.attackSimulation")}</h2>
            <div className="space-y-2">
              {ATTACK_SCENARIOS.map(s => (
                <button key={s.id} onClick={() => { setSimScenario(s.id); setSimResult(null); }} aria-pressed={simScenario === s.id}
                  className={`w-full text-left rounded-lg border p-3 transition ${simScenario === s.id ? "border-orange-500 bg-orange-50 dark:bg-orange-950/30" : "border-gray-200 dark:border-gray-700"}`}>
                  <div className="flex items-center justify-between">
                    <span className="text-sm font-medium">{s.name}</span>
                    <code className="text-xs font-mono text-orange-500">{s.technique}</code>
                  </div>
                  <p className="text-xs text-gray-400 mt-1">{s.desc}</p>
                </button>
              ))}
            </div>
            <button onClick={runSimulation} disabled={!simScenario || simulating}
              className="mt-4 flex items-center gap-2 rounded-lg bg-orange-600 px-4 py-2 text-sm font-medium text-white hover:bg-orange-700 disabled:opacity-50">
              {simulating ? <Loader2 className="h-4 w-4 animate-spin" /> : <Play className="h-4 w-4" />} {t("mitre.runSimulation")}
            </button>
          </div>
          <div className={card}>
            <h2 className="mb-4 flex items-center gap-2 text-sm font-semibold uppercase text-gray-400"><Radar className="h-4 w-4" /> {t("mitre.simulationResult")}</h2>
            {simResult ? (
              <div className={`flex items-center gap-3 rounded-xl border-2 p-4 ${simResult.fired ? "border-green-300 bg-green-50 dark:border-green-700 dark:bg-green-950/30" : "border-red-300 bg-red-50 dark:border-red-700 dark:bg-red-950/30"}`}>
                {simResult.fired ? <CheckCircle2 className="h-8 w-8 text-green-500" /> : <XCircle className="h-8 w-8 text-red-500" />}
                <div>
                  <p className={`text-lg font-bold ${simResult.fired ? "text-green-700 dark:text-green-400" : "text-red-700 dark:text-red-400"}`}>
                    {simResult.fired ? t("mitre.detected") : t("mitre.notDetected")}
                  </p>
                  {simResult.fired && <p className="text-xs text-gray-500 dark:text-gray-400">{t("mitre.triggeredBy")}: {simResult.rule} → {simResult.action}</p>}
                </div>
              </div>
            ) : (
              <div className="py-8 text-center"><FlaskConical className="mx-auto h-10 w-10 text-gray-300" /><p className="mt-3 text-sm text-gray-400">{t("mitre.selectScenario")}</p></div>
            )}
          </div>
        </div>
      )}
    </div>
  );
}
