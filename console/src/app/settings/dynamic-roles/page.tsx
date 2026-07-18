"use client";

import React, { useEffect, useState } from "react";
import { useApi } from "@/lib/api";
import { useTranslations } from "@/lib/i18n";
import {
  Users2, Loader2, AlertCircle, X, Plus, Trash2, Play, CheckCircle, XCircle,
} from "lucide-react";

interface DynamicRole {
  id: string;
  name: string;
  description: string;
  enabled: boolean;
  conditions: { attribute: string; operator: string; value: string }[];
  assigned_roles: string[];
  user_count: number;
  last_evaluated: string;
}

export default function DynamicRolesPage() {
  const t = useTranslations();
  const { apiFetch } = useApi();
  const [roles, setRoles] = useState<DynamicRole[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [editing, setEditing] = useState<DynamicRole | null>(null);
  const [testUser, setTestUser] = useState("");
  const [testRole, setTestRole] = useState<string | null>(null);
  const [testResult, setTestResult] = useState<{ matched: boolean; reason: string } | null>(null);

  useEffect(() => {
    (async () => {
      try { setRoles(await apiFetch<DynamicRole[]>("/api/v1/policy/dynamic-roles").catch(() => [])); }
      catch { setError("Failed to load dynamic roles"); }
      finally { setLoading(false); }
    })();
  }, []);

  const handleSave = async () => {
    if (!editing) return;
    try {
      if (editing.id) { await apiFetch(`/api/v1/policy/dynamic-roles/${editing.id}`, { method: "PUT", body: JSON.stringify(editing) }); }
      else { const created = await apiFetch<DynamicRole>("/api/v1/policy/dynamic-roles", { method: "POST", body: JSON.stringify(editing) }); setRoles((p) => [...p, created]); }
      setEditing(null); setRoles(await apiFetch<DynamicRole[]>("/api/v1/policy/dynamic-roles").catch(() => roles));
    } catch { setError("Save failed"); }
  };

  const handleDelete = async (id: string) => {
    try { await apiFetch(`/api/v1/policy/dynamic-roles/${id}`, { method: "DELETE" }); setRoles((p) => p.filter((r: any) => r.id !== id)); }
    catch { setError("Delete failed"); }
  };

  const handleTest = async (roleId: string) => {
    if (!testUser.trim()) return;
    setTestRole(roleId); setTestResult(null);
    try { const result = await apiFetch<{ matched: boolean; reason: string }>(`/api/v1/policy/dynamic-roles/${roleId}/test`, { method: "POST", body: JSON.stringify({ user_id: testUser }) }); setTestResult(result); }
    catch { setTestResult({ matched: false, reason: "Test failed" }); }
    finally { setTestRole(null); }
  };

  const addCondition = () => { if (!editing) return; setEditing({ ...editing, conditions: [...editing.conditions, { attribute: "department", operator: "eq", value: "" }] }); };
  const updateCondition = (idx: number, field: string, val: string) => { if (!editing) return; setEditing({ ...editing, conditions: editing.conditions.map((c: any, i: number) => i === idx ? { ...c, [field]: val } : c) }); };
  const removeCondition = (idx: number) => { if (!editing) return; setEditing({ ...editing, conditions: editing.conditions.filter((_, i) => i !== idx) }); };

  const cardCls = "rounded-xl border border-gray-200 bg-white p-5 shadow-sm dark:border-gray-700 dark:bg-gray-800";

  return (
    <div className="space-y-6">
      <div className="flex items-center justify-between">
        <div>
          <h1 className="flex items-center gap-2 text-2xl font-bold text-gray-900 dark:text-white"><Users2 className="h-6 w-6 text-purple-600" /> {t("big1.dynamicRoles.title")}</h1>
          <p className="mt-1 text-sm text-gray-500 dark:text-gray-400">{t("big1.dynamicRoles.attributeBasedDynamicRoleAssignmentWithConditionBuilderAndUserTesting")}</p>
        </div>
        <button onClick={() => setEditing({ id: "", name: "", description: "", enabled: true, conditions: [], assigned_roles: [], user_count: 0, last_evaluated: "" })} className="flex items-center gap-2 rounded-lg bg-purple-600 px-4 py-2 text-sm font-medium text-white hover:bg-purple-700"><Plus className="h-4 w-4" />{t("big1.dynamicRoles.newRole")}</button>
      </div>

      {error && <div role="alert" className="flex items-center gap-2 rounded-lg bg-red-50 px-4 py-3 text-sm text-red-700 dark:bg-red-900/20 dark:text-red-400"><AlertCircle className="h-4 w-4 shrink-0" />{error}<button onClick={() => setError(null)} aria-label="Dismiss error" className="ml-auto"><X className="h-4 w-4" /></button></div>}

      {loading ? <div className="flex justify-center py-12"><Loader2 className="h-8 w-8 animate-spin text-purple-600" /></div>
      : roles.length === 0 ? (
        <div className={cardCls}><div className="py-12 text-center"><Users2 className="mx-auto h-12 w-12 text-gray-300" /><p className="mt-4 text-sm text-gray-400">{t("big1.dynamicRoles.noDynamicRolesDefined")}</p></div></div>
      ) : (
        <div className="space-y-3">
          {roles.map((r: any) => (
            <div key={r.id} className={cardCls}>
              <div className="flex items-start justify-between">
                <div className="flex-1">
                  <div className="flex items-center gap-2"><span className="font-semibold text-gray-900 dark:text-white">{r.name}</span>{r.user_count > 0 && <span className="rounded bg-purple-100 px-1.5 py-0.5 text-xs text-purple-600 dark:bg-purple-900/30">{r.user_count}{t("big1.dynamicRoles.users")}</span>}{!r.enabled && <span className="rounded-full bg-gray-200 px-2 py-0.5 text-xs text-gray-500 dark:bg-gray-700">{t("big1.dynamicRoles.disabled")}</span>}</div>
                  {r.description && <p className="mt-1 text-sm text-gray-500">{r.description}</p>}
                  <div className="mt-2 flex flex-wrap gap-1">{r.conditions.map((c: any, i: number) => <span key={i} className="rounded bg-gray-100 px-1.5 py-0.5 font-mono text-xs text-gray-600 dark:bg-gray-700 dark:text-gray-300">{c.attribute} {c.operator} {c.value}</span>)}{r.conditions.length === 0 && <span className="text-xs text-gray-400">{t("big1.dynamicRoles.noConditions")}</span>}</div>
                </div>
                <div className="flex gap-1"><button onClick={() => setEditing({ ...r })} className="rounded p-1.5 text-gray-400 hover:text-purple-600"><Users2 className="h-4 w-4" /></button><button onClick={() => handleDelete(r.id)} className="rounded p-1.5 text-gray-400 hover:bg-red-50 hover:text-red-600"><Trash2 className="h-4 w-4" /></button></div>
              </div>
            </div>
          ))}
        </div>
      )}

      {/* Test against user */}
      <div className={cardCls}>
        <h3 className="mb-3 text-sm font-semibold text-gray-700 dark:text-gray-300">{t("big1.dynamicRoles.testAssignment")}</h3>
        <div className="flex items-center gap-2">
          <input aria-label="User ID or email" value={testUser} onChange={(e) => setTestUser(e.target.value)} placeholder="User ID or email" className="flex-1 rounded-lg border border-gray-300 px-3 py-2 text-sm dark:border-gray-600 dark:bg-gray-900 dark:text-gray-200" />
          {roles.length > 0 && <select aria-label="Select option" onChange={(e) => { if (e.target.value) handleTest(e.target.value); }} className="rounded-lg border border-gray-300 px-3 py-2 text-sm dark:border-gray-600 dark:bg-gray-900 dark:text-gray-200"><option value="">{t("big1.dynamicRoles.testRole")}</option>{roles.map((r: any) => <option key={r.id} value={r.id}>{r.name}</option>)}</select>}
        </div>
        {testResult && <div className={`mt-2 flex items-center gap-2 text-sm ${testResult.matched ? "text-green-600" : "text-red-600"}`}>{testResult.matched ? <CheckCircle className="h-4 w-4" /> : <XCircle className="h-4 w-4" />}{testResult.reason}</div>}
      </div>

      {/* Edit modal */}
      {editing && (
        <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/50" onClick={() => setEditing(null)}>
          <div role="dialog" aria-modal="true" className="max-h-[90vh] w-full max-w-lg overflow-y-auto rounded-xl bg-white p-6 shadow-xl dark:bg-gray-800" onClick={(e) => e.stopPropagation()}>
            <div className="mb-4 flex items-center justify-between"><h3 className="text-lg font-bold text-gray-900 dark:text-white">{editing.id ? t("big1.dynamicRoles.editDynamicRole") : t("big1.dynamicRoles.newDynamicRole")}</h3><button onClick={() => setEditing(null)} aria-label="Close"><X className="h-5 w-5 text-gray-400" /></button></div>
            <div className="space-y-4">
              <div><label className="mb-1 block text-xs font-semibold uppercase text-gray-400">{t("big1.dynamicRoles.name")}</label><input aria-label="editing" value={editing.name} onChange={(e) => setEditing({ ...editing, name: e.target.value })} className="w-full rounded-lg border border-gray-300 px-3 py-2 text-sm dark:border-gray-600 dark:bg-gray-900 dark:text-gray-200" /></div>
              <div><label className="mb-1 block text-xs font-semibold uppercase text-gray-400">{t("big1.dynamicRoles.description")}</label><input aria-label="editing" value={editing.description} onChange={(e) => setEditing({ ...editing, description: e.target.value })} className="w-full rounded-lg border border-gray-300 px-3 py-2 text-sm dark:border-gray-600 dark:bg-gray-900 dark:text-gray-200" /></div>
              <div><label className="mb-1 block text-xs font-semibold uppercase text-gray-400">{t("big1.dynamicRoles.conditions")}</label><div className="space-y-2">{editing.conditions.map((c: any, idx: number) => (<div key={idx} className="flex items-center gap-2"><input aria-label="attribute" value={c.attribute} onChange={(e) => updateCondition(idx, "attribute", e.target.value)} placeholder="attribute" className="flex-1 rounded border border-gray-300 px-2 py-1.5 text-xs dark:border-gray-600 dark:bg-gray-900 dark:text-gray-200" /><select value={c.operator} onChange={(e) => updateCondition(idx, "operator", e.target.value)} className="rounded border border-gray-300 px-2 py-1.5 text-xs dark:border-gray-600 dark:bg-gray-900 dark:text-gray-200"><option value="eq">==</option><option value="ne">!=</option><option value="in">{t("big1.dynamicRoles.in")}</option><option value="contains">{t("big1.dynamicRoles.contains")}</option></select><input value={c.value} onChange={(e) => updateCondition(idx, "value", e.target.value)} placeholder="value" className="flex-1 rounded border border-gray-300 px-2 py-1.5 text-xs dark:border-gray-600 dark:bg-gray-900 dark:text-gray-200" /><button onClick={() => removeCondition(idx)} className="text-red-400"><Trash2 className="h-3 w-3" /></button></div>))}</div><button onClick={addCondition} className="mt-2 flex items-center gap-1 text-xs text-purple-600 hover:underline" aria-label="Plus"><Plus className="h-3 w-3" />{t("big1.dynamicRoles.addCondition")}</button></div>
              <div><label className="mb-1 block text-xs font-semibold uppercase text-gray-400">{t("big1.dynamicRoles.assignedRolesCommaSeparated")}</label><input aria-label="admin, editor" value={editing.assigned_roles.join(", ")} onChange={(e) => setEditing({ ...editing, assigned_roles: e.target.value.split(",").map((s: any) => s.trim()).filter(Boolean) })} placeholder="admin, editor" className="w-full rounded-lg border border-gray-300 px-3 py-2 text-sm dark:border-gray-600 dark:bg-gray-900 dark:text-gray-200" /></div>
              <button aria-label="action" onClick={handleSave} className="flex w-full items-center justify-center gap-2 rounded-lg bg-purple-600 py-2 text-sm font-medium text-white hover:bg-purple-700">{editing.id ? t("big1.dynamicRoles.update") : t("big1.dynamicRoles.create")}{t("big1.dynamicRoles.dynamicRole")}</button>
            </div>
          </div>
        </div>
      )}
    </div>
  );
}
