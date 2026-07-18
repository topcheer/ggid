"use client";

import { useState, useEffect, useCallback } from "react";
import { useTranslations } from "@/lib/i18n";
import { authHeader } from "@/lib/auth-helpers";
import {
  Grid3x3, List, Plus, Trash2, Loader2, Check, X,
  AlertTriangle, Shield, Save,
} from "lucide-react";

const API_BASE = process.env.NEXT_PUBLIC_API_URL || "http://localhost:8080";
type TabId = "matrix" | "rules";

interface SoDRule {
  id: string; role_a: string; role_b: string; severity: "high" | "medium" | "low"; description: string;
}

const ROLES = ["superadmin", "admin", "auditor", "developer", "analyst", "operator", "viewer", "billing_admin", "security_officer", "compliance_officer"];

const severityColors: Record<string, string> = {
  high: "bg-red-100 text-red-700 dark:bg-red-950 dark:text-red-300",
  medium: "bg-orange-100 text-orange-700 dark:bg-orange-950 dark:text-orange-300",
  low: "bg-yellow-100 text-yellow-700 dark:bg-yellow-950 dark:text-yellow-300",
};

export default function SoDMatrixPage() {
  const t = useTranslations();
  const [tab, setTab] = useState<TabId>("matrix");
  const [rules, setRules] = useState<SoDRule[]>([]);
  const [loading, setLoading] = useState(true);

  const load = useCallback(async () => {
    setLoading(true);
    try {
      const res = await fetch(`${API_BASE}/api/v1/policies/sod/rules`, { headers: { ...authHeader() } });
      if (res.ok) { const d = await res.json(); setRules(d.rules || d || []); return; }
    } catch { /* mock */ }
    setRules([
      { id: "r1", role_a: "developer", role_b: "auditor", severity: "high", description: "Cannot develop and audit the same system" },
      { id: "r2", role_a: "billing_admin", role_b: "admin", severity: "medium", description: "Cannot manage billing and user accounts" },
      { id: "r3", role_a: "developer", role_b: "billing_admin", severity: "high", description: "Cannot access source code and billing" },
      { id: "r4", role_a: "compliance_officer", role_b: "superadmin", severity: "medium", description: "Compliance oversight requires independence" },
      { id: "r5", role_a: "security_officer", role_b: "operator", severity: "low", description: "Security policy and operations should be separated" },
    ]);
  }, []);

  useEffect(() => { load(); }, [load]);

  // Build conflict set for matrix
  const conflictSet = new Set<string>();
  rules.forEach((r) => {
    conflictSet.add(`${r.role_a}::${r.role_b}`);
    conflictSet.add(`${r.role_b}::${r.role_a}`);
  });

  const tabs: { id: TabId; label: string; icon: typeof Grid3x3 }[] = [
    { id: "matrix", label: t("sodMatrix.tabs.matrix"), icon: Grid3x3 },
    { id: "rules", label: t("sodMatrix.tabs.rules"), icon: List },
  ];

  return (
    <div className="min-h-screen bg-gray-50 dark:bg-gray-950 p-4 md:p-8">
      <div className="max-w-5xl mx-auto">
        <div className="mb-6">
          <div className="flex items-center gap-3 mb-1">
            <Grid3x3 className="w-7 h-7 text-blue-600" />
            <h1 className="text-2xl font-bold text-gray-900 dark:text-white">{t("sodMatrix.title")}</h1>
          </div>
          <p className="text-gray-600 dark:text-gray-400 text-sm">{t("sodMatrix.description")}</p>
        </div>

        <div className="flex gap-1 mb-6 bg-gray-200 dark:bg-gray-800 rounded-lg p-1">
          {tabs.map(({ id, label, icon: Icon }) => (
            <button key={id} onClick={() => setTab(id)}
              className={`flex items-center gap-2 px-4 py-2 rounded-md text-sm font-medium transition-colors ${
                tab === id ? "bg-white dark:bg-gray-700 text-blue-600 dark:text-blue-400 shadow-sm" : "text-gray-600 dark:text-gray-400 hover:text-gray-900 dark:hover:text-white"
              }`}>
              <Icon className="w-4 h-4" />{label}
            </button>
          ))}
        </div>

        {loading ? (
          <div className="flex justify-center py-20"><Loader2 className="w-8 h-8 animate-spin text-blue-600" /></div>
        ) : (
          <>
            {tab === "matrix" && <MatrixTab roles={ROLES} conflictSet={conflictSet} />}
            {tab === "rules" && <RulesTab rules={rules} setRules={setRules} />}
          </>
        )}
      </div>
    </div>
  );
}

// ============ Matrix Tab ============

function MatrixTab({ roles, conflictSet }: { roles: string[]; conflictSet: Set<string> }) {
  const t = useTranslations();

  return (
    <div className="bg-white dark:bg-gray-900 rounded-xl border border-gray-200 dark:border-gray-800 p-6">
      <h3 className="text-sm font-semibold text-gray-900 dark:text-white mb-1">{t("sodMatrix.matrix.title")}</h3>
      <p className="text-xs text-gray-500 dark:text-gray-400 mb-4">{t("sodMatrix.matrix.description")}</p>

      {roles.length === 0 ? (
        <div className="text-center py-12"><Grid3x3 className="w-12 h-12 mx-auto mb-3 text-gray-300" /><p className="text-sm text-gray-500">{t("sodMatrix.matrix.noRoles")}</p></div>
      ) : (
        <div className="overflow-x-auto">
          <table className="w-full text-xs border-collapse">
            <thead>
              <tr>
                <th className="sticky left-0 bg-gray-50 dark:bg-gray-800 p-2 border border-gray-200 dark:border-gray-700 text-gray-500 font-medium min-w-[120px]">{t("sodMatrix.rules.roleA")}</th>
                {roles.map((r) => (
                  <th key={r} className="p-2 border border-gray-200 dark:border-gray-700 text-gray-500 font-medium min-w-[60px]">
                    <div className="rotate-[-45deg] origin-bottom-left whitespace-nowrap text-xs h-16 flex items-end pb-1">{r}</div>
                  </th>
                ))}
              </tr>
            </thead>
            <tbody>
              {roles.map((roleA) => (
                <tr key={roleA}>
                  <td className="sticky left-0 bg-gray-50 dark:bg-gray-800 p-2 border border-gray-200 dark:border-gray-700 text-gray-700 dark:text-gray-300 font-medium text-xs whitespace-nowrap">{roleA}</td>
                  {roles.map((roleB) => {
                    const isSelf = roleA === roleB;
                    const isConflict = conflictSet.has(`${roleA}::${roleB}`);
                    return (
                      <td key={roleB} className={`p-1 border border-gray-200 dark:border-gray-700 text-center`}>
                        {isSelf ? (
                          <div className="w-6 h-6 mx-auto rounded bg-gray-200 dark:bg-gray-700 flex items-center justify-center">
                            <span className="text-gray-400 text-xs">—</span>
                          </div>
                        ) : isConflict ? (
                          <div className="w-6 h-6 mx-auto rounded bg-red-500 flex items-center justify-center" title={t("sodMatrix.matrix.conflict")}>
                            <X className="w-3.5 h-3.5 text-white" />
                          </div>
                        ) : (
                          <div className="w-6 h-6 mx-auto rounded bg-green-50 dark:bg-green-950/30 flex items-center justify-center" title={t("sodMatrix.matrix.noConflict")}>
                            <Check className="w-3.5 h-3.5 text-green-500" />
                          </div>
                        )}
                      </td>
                    );
                  })}
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      )}

      <div className="flex items-center gap-4 mt-4 pt-4 border-t border-gray-200 dark:border-gray-800">
        <div className="flex items-center gap-2"><div className="w-4 h-4 rounded bg-red-500" /><span className="text-xs text-gray-500">{t("sodMatrix.matrix.conflict")}</span></div>
        <div className="flex items-center gap-2"><div className="w-4 h-4 rounded bg-green-50 dark:bg-green-950/30 border border-green-300" /><span className="text-xs text-gray-500">{t("sodMatrix.matrix.noConflict")}</span></div>
      </div>
    </div>
  );
}

// ============ Rules Tab ============

function RulesTab({ rules, setRules }: { rules: SoDRule[]; setRules: (r: SoDRule[]) => void }) {
  const t = useTranslations();
  const [showForm, setShowForm] = useState(false);
  const [newRoleA, setNewRoleA] = useState("");
  const [newRoleB, setNewRoleB] = useState("");
  const [newSeverity, setNewSeverity] = useState<"high" | "medium" | "low">("high");
  const [newDesc, setNewDesc] = useState("");
  const [msg, setMsg] = useState<string | null>(null);

  const addRule = async () => {
    if (!newRoleA || !newRoleB || newRoleA === newRoleB) return;
    const rule: SoDRule = { id: `r${Date.now()}`, role_a: newRoleA, role_b: newRoleB, severity: newSeverity, description: newDesc };
    setRules([...rules, rule]);
    setShowForm(false); setNewRoleA(""); setNewRoleB(""); setNewSeverity("high"); setNewDesc("");
    setMsg(t("sodMatrix.rules.saved"));
    setTimeout(() => setMsg(null), 3000);
  };

  const deleteRule = (id: string) => {
    if (!confirm(t("sodMatrix.rules.confirmDelete"))) return;
    setRules(rules.filter((r) => r.id !== id));
    setMsg(t("sodMatrix.rules.deleted"));
    setTimeout(() => setMsg(null), 3000);
  };

  return (
    <div className="bg-white dark:bg-gray-900 rounded-xl border border-gray-200 dark:border-gray-800 p-6">
      <div className="flex items-center justify-between mb-4">
        <div>
          <h3 className="text-sm font-semibold text-gray-900 dark:text-white">{t("sodMatrix.rules.title")}</h3>
          <p className="text-xs text-gray-500 dark:text-gray-400 mt-1">{t("sodMatrix.rules.description")}</p>
        </div>
        <button onClick={() => setShowForm(!showForm)} className="flex items-center gap-1.5 px-3 py-1.5 bg-blue-600 hover:bg-blue-700 text-white rounded-lg text-sm font-medium">
          <Plus className="w-4 h-4" />{t("sodMatrix.rules.addRule")}
        </button>
      </div>

      {msg && <div className="flex items-center gap-2 px-4 py-2 mb-3 rounded-lg bg-green-50 text-green-700 dark:bg-green-950 dark:text-green-300 text-sm"><Check className="w-4 h-4" />{msg}</div>}

      {showForm && (
        <div className="border border-gray-200 dark:border-gray-700 rounded-lg p-4 space-y-3 bg-gray-50 dark:bg-gray-800/50 mb-4">
          <div className="grid grid-cols-2 gap-3">
            <div>
              <label className="block text-xs font-medium text-gray-600 dark:text-gray-400 mb-1">{t("sodMatrix.rules.roleA")}</label>
              <select value={newRoleA} onChange={(e) => setNewRoleA(e.target.value)} className="w-full px-2 py-1.5 rounded border border-gray-300 dark:border-gray-700 bg-white dark:bg-gray-800 text-xs text-gray-900 dark:text-white">
                <option value="">{t("sodMatrix.rules.selectRoleA")}</option>
                {ROLES.map((r) => <option key={r} value={r}>{r}</option>)}
              </select>
            </div>
            <div>
              <label className="block text-xs font-medium text-gray-600 dark:text-gray-400 mb-1">{t("sodMatrix.rules.roleB")}</label>
              <select value={newRoleB} onChange={(e) => setNewRoleB(e.target.value)} className="w-full px-2 py-1.5 rounded border border-gray-300 dark:border-gray-700 bg-white dark:bg-gray-800 text-xs text-gray-900 dark:text-white">
                <option value="">{t("sodMatrix.rules.selectRoleB")}</option>
                {ROLES.filter((r) => r !== newRoleA).map((r) => <option key={r} value={r}>{r}</option>)}
              </select>
            </div>
          </div>
          <div>
            <label className="block text-xs font-medium text-gray-600 dark:text-gray-400 mb-1">{t("sodMatrix.rules.severity")}</label>
            <div className="flex gap-2">
              {(["high", "medium", "low"] as const).map((s) => (
                <button key={s} onClick={() => setNewSeverity(s)} className={`px-3 py-1 rounded-lg text-xs font-medium border-2 ${newSeverity === s ? "border-blue-500 " + severityColors[s] : "border-gray-200 dark:border-gray-700 text-gray-500"}`}>
                  {t(`sodMatrix.rules.severity${s.replace(/^./, (m) => m.toUpperCase())}`)}
                </button>
              ))}
            </div>
          </div>
          <div>
            <label className="block text-xs font-medium text-gray-600 dark:text-gray-400 mb-1">{t("sodMatrix.rules.description")}</label>
            <input type="text" value={newDesc} onChange={(e) => setNewDesc(e.target.value)} placeholder={t("sodMatrix.rules.placeholderDesc")}
              className="w-full px-2 py-1.5 rounded border border-gray-300 dark:border-gray-700 bg-white dark:bg-gray-800 text-xs text-gray-900 dark:text-white" />
          </div>
          <div className="flex gap-2">
            <button onClick={addRule} disabled={!newRoleA || !newRoleB || newRoleA === newRoleB} className="flex items-center gap-1.5 px-4 py-1.5 bg-blue-600 hover:bg-blue-700 disabled:opacity-50 text-white rounded-lg text-xs font-medium">
              <Save className="w-3.5 h-3.5" />{t("sodMatrix.rules.save")}
            </button>
            <button onClick={() => setShowForm(false)} className="px-4 py-1.5 bg-gray-200 dark:bg-gray-700 text-gray-700 dark:text-gray-300 rounded-lg text-xs font-medium">Cancel</button>
          </div>
        </div>
      )}

      <div className="space-y-2">
        {rules.length === 0 ? (
          <div className="text-center py-8"><List className="w-10 h-10 mx-auto mb-2 text-gray-300" /><p className="text-sm text-gray-500">{t("sodMatrix.rules.noRules")}</p></div>
        ) : (
          rules.map((r) => (
            <div key={r.id} className="flex items-center gap-3 p-3 rounded-lg border border-gray-200 dark:border-gray-800">
              <Shield className="w-4 h-4 text-gray-400 flex-shrink-0" />
              <div className="flex-1 min-w-0">
                <div className="flex items-center gap-2">
                  <span className="text-sm font-medium text-gray-900 dark:text-white">{r.role_a}</span>
                  <X className="w-3 h-3 text-red-500" />
                  <span className="text-sm font-medium text-gray-900 dark:text-white">{r.role_b}</span>
                </div>
                {r.description && <p className="text-xs text-gray-500 mt-0.5">{r.description}</p>}
              </div>
              <span className={`px-2 py-0.5 text-xs rounded-full ${severityColors[r.severity]}`}>{t(`sodMatrix.rules.severity${r.severity.replace(/^./, (m) => m.toUpperCase())}`)}</span>
              <button onClick={() => deleteRule(r.id)} className="p-1.5 hover:bg-red-50 dark:hover:bg-red-950 rounded"><Trash2 className="w-4 h-4 text-red-500" /></button>
            </div>
          ))
        )}
      </div>
    </div>
  );
}
