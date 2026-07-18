"use client";

import { useState, useEffect, useCallback, useRef } from "react";
import { useApi } from "@/lib/api";
import { useTranslations } from "@/lib/i18n";
import {
  Shield,
  Plus,
  Trash2,
  Play,
  Save,
  FileJson,
  CheckCircle,
  XCircle,
  Loader2,
  Download,
  Upload,
  Grid3x3,
  Layers,
  ChevronDown,
  ChevronRight,
} from "lucide-react";

interface PolicyRule {
  subject: string;
  resource: string;
  action: string;
  effect: "allow" | "deny";
}

interface Policy {
  id?: string;
  name: string;
  description?: string;
  rules: PolicyRule[];
  priority?: number;
  effect?: "allow" | "deny";
}

// --- ABAC visual rule builder types ---
interface AbacCondition {
  id: string;
  target: "subject" | "resource";
  attribute: string;
  operator: "eq" | "ne" | "in" | "contains";
  value: string;
}

interface AbacRule {
  id: string;
  conditions: AbacCondition[];
  effect: "allow" | "deny";
}

// --- RBAC matrix types ---
type PermissionKey = "read" | "write" | "delete" | "admin";

interface RbacEntry {
  role: string;
  permissions: Record<PermissionKey, boolean>;
}

const PERMISSION_COLUMNS: PermissionKey[] = ["read", "write", "delete", "admin"];

const OPERATORS: { value: AbacCondition["operator"]; label: string }[] = [
  { value: "eq", label: "equals (=)" },
  { value: "ne", label: "not equals (≠)" },
  { value: "in", label: "in list" },
  { value: "contains", label: "contains" },
];

let condSeq = 0;
function makeCondition(): AbacCondition {
  return {
    id: `cond-${Date.now()}-${condSeq++}`,
    target: "subject",
    attribute: "",
    operator: "eq",
    value: "",
  };
}

let abacSeq = 0;
function makeAbacRule(): AbacRule {
  return {
    id: `abac-${Date.now()}-${abacSeq++}`,
    conditions: [makeCondition()],
    effect: "allow",
  };
}

export default function PoliciesPage() {
  const t = useTranslations();
  const { apiFetch } = useApi();
  const [policies, setPolicies] = useState<Policy[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [msg, setMsg] = useState<string | null>(null);

  const [selectedPolicy, setSelectedPolicy] = useState<Policy | null>(null);
  const [policyJson, setPolicyJson] = useState("");
  const [rules, setRules] = useState<PolicyRule[]>([]);
  const [policyName, setPolicyName] = useState("");

  // Dry-run state
  const [dryRunSubject, setDryRunSubject] = useState("");
  const [dryRunResource, setDryRunResource] = useState("");
  const [dryRunAction, setDryRunAction] = useState("");
  const [dryRunResult, setDryRunResult] = useState<{ allow: boolean; detail?: string } | null>(null);
  const [dryRunLoading, setDryRunLoading] = useState(false);

  // Policy-level priority and effect
  const [policyPriority, setPolicyPriority] = useState(50);
  const [policyEffect, setPolicyEffect] = useState<"allow" | "deny">("allow");

  // RBAC matrix state
  const [rbacEntries, setRbacEntries] = useState<RbacEntry[]>([
    { role: "admin", permissions: { read: true, write: true, delete: true, admin: true } },
    { role: "editor", permissions: { read: true, write: true, delete: false, admin: false } },
    { role: "viewer", permissions: { read: true, write: false, delete: false, admin: false } },
  ]);
  const [rbacCollapsed, setRbacCollapsed] = useState(false);

  // ABAC visual builder state
  const [abacRules, setAbacRules] = useState<AbacRule[]>([]);
  const [abacCollapsed, setAbacCollapsed] = useState(false);

  const importFileRef = useRef<HTMLInputElement>(null);

  const refresh = useCallback(async () => {
    setLoading(true);
    try {
      const data = await apiFetch<{ policies?: Policy[]; items?: Policy[] }>("/api/v1/policies");
      const list = data.policies || data.items || [];
      setPolicies(list);
      setError(null);
    } catch (err) {
      setError(err instanceof Error ? err.message : "Failed to load policies");
    } finally {
      setLoading(false);
    }
  }, [apiFetch]);

  useEffect(() => {
    refresh();
  }, [refresh]);

  useEffect(() => {
    if (msg) {
      const t = setTimeout(() => setMsg(null), 3000);
      return () => clearTimeout(t);
    }
  }, [msg]);

  const selectPolicy = (p: Policy) => {
    setSelectedPolicy(p);
    setPolicyName(p.name);
    setRules(p.rules || []);
    setPolicyPriority(p.priority ?? 50);
    setPolicyEffect(p.effect ?? "allow");
    setPolicyJson(JSON.stringify(p, null, 2));
  };

  const handleCreatePolicy = async () => {
    const payload = {
      name: policyName || "Untitled Policy",
      rules: rules,
      priority: policyPriority,
      effect: policyEffect,
    };
    try {
      await apiFetch("/api/v1/policies", {
        method: "POST",
        body: JSON.stringify(payload),
      });
      setMsg("Policy created successfully");
      refresh();
      resetEditor();
    } catch (err) {
      alert(err instanceof Error ? err.message : "Failed to create policy");
    }
  };

  const handleDeletePolicy = async (id: string, name: string) => {
    if (!confirm(`Delete policy "${name}"?`)) return;
    try {
      await apiFetch(`/api/v1/policies/${id}`, { method: "DELETE" });
      setMsg("Policy deleted");
      refresh();
      if (selectedPolicy?.id === id) resetEditor();
    } catch (err) {
      alert(err instanceof Error ? err.message : "Failed to delete policy");
    }
  };

  const handleDryRun = async () => {
    setDryRunLoading(true);
    setDryRunResult(null);
    try {
      const data = await apiFetch<{ allow?: boolean; decision?: string; detail?: string }>(
        "/api/v1/policies/dry-run",
        {
          method: "POST",
          body: JSON.stringify({
            subject: dryRunSubject,
            resource: dryRunResource,
            action: dryRunAction,
          }),
        },
      );
      const allow = data.allow ?? data.decision === "allow";
      setDryRunResult({ allow, detail: data.detail });
    } catch (err) {
      setDryRunResult({
        allow: false,
        detail: err instanceof Error ? err.message : "Dry-run failed",
      });
    } finally {
      setDryRunLoading(false);
    }
  };

  const addRule = () => {
    setRules([...rules, { subject: "", resource: "", action: "", effect: "allow" }]);
  };

  const removeRule = (index: number) => {
    setRules(rules.filter((_, i) => i !== index));
  };

  const updateRule = (index: number, field: keyof PolicyRule, value: string) => {
    setRules(rules.map((r: any, i: any) => (i === index ? { ...r, [field]: value } : r)));
  };

  const syncJsonToRules = () => {
    try {
      const parsed = JSON.parse(policyJson);
      setRules(parsed.rules || []);
      setPolicyName(parsed.name || policyName);
      setMsg("JSON parsed and synced to rules");
    } catch (err) {
      alert("Invalid JSON: " + (err instanceof Error ? err.message : "parse error"));
    }
  };

  const resetEditor = () => {
    setSelectedPolicy(null);
    setPolicyName("");
    setRules([]);
    setPolicyJson("");
    setPolicyPriority(50);
    setPolicyEffect("allow");
  };

  // --- Export ALL policies as JSON ---
  const handleExportAllJson = () => {
    const exportData = {
      exported_at: new Date().toISOString(),
      policies,
    };
    const blob = new Blob([JSON.stringify(exportData, null, 2)], { type: "application/json" });
    const url = URL.createObjectURL(blob);
    const a = document.createElement("a");
    a.href = url;
    a.download = `all_policies_${new Date().toISOString().slice(0, 10)}.json`;
    a.click();
    URL.revokeObjectURL(url);
    setMsg("All policies exported as JSON");
  };

  // --- JSON export (current policy) ---
  const handleExportJson = () => {
    const exportData = {
      name: policyName || "Untitled Policy",
      rules,
      priority: policyPriority,
      effect: policyEffect,
      rbac: rbacEntries,
      abac: abacRules,
    };
    const blob = new Blob([JSON.stringify(exportData, null, 2)], { type: "application/json" });
    const url = URL.createObjectURL(blob);
    const a = document.createElement("a");
    a.href = url;
    a.download = `${(policyName || "policy").replace(/[^a-z0-9]/gi, "_")}.json`;
    a.click();
    URL.revokeObjectURL(url);
    setMsg("Policy exported as JSON");
  };

  // --- JSON import ---
  const handleImportJson = (e: React.ChangeEvent<HTMLInputElement>) => {
    const file = e.target.files?.[0];
    if (!file) return;
    const reader = new FileReader();
    reader.onload = (ev) => {
      try {
        const text = ev.target?.result as string;
        const parsed = JSON.parse(text);
        setPolicyName(parsed.name || "Imported Policy");
        setRules(parsed.rules || []);
        setPolicyPriority(typeof parsed.priority === "number" ? parsed.priority : 50);
        setPolicyEffect(parsed.effect === "deny" ? "deny" : "allow");
        setPolicyJson(JSON.stringify(parsed, null, 2));
        if (parsed.rbac && Array.isArray(parsed.rbac)) {
          setRbacEntries(parsed.rbac);
        }
        if (parsed.abac && Array.isArray(parsed.abac)) {
          setAbacRules(parsed.abac);
        }
        setMsg("Policy imported successfully");
      } catch (err) {
        alert("Failed to parse JSON file: " + (err instanceof Error ? err.message : "unknown error"));
      }
    };
    reader.readAsText(file);
    // Reset input so the same file can be re-imported
    e.target.value = "";
  };

  // --- RBAC matrix handlers ---
  const addRbacRole = () => {
    const name = prompt("Enter role name:");
    if (!name) return;
    setRbacEntries([
      ...rbacEntries,
      { role: name, permissions: { read: false, write: false, delete: false, admin: false } },
    ]);
  };

  const removeRbacRole = (index: number) => {
    setRbacEntries(rbacEntries.filter((_, i) => i !== index));
  };

  const toggleRbacPermission = (roleIndex: number, perm: PermissionKey) => {
    setRbacEntries(
      rbacEntries.map((entry: any, i: any) =>
        i === roleIndex
          ? { ...entry, permissions: { ...entry.permissions, [perm]: !entry.permissions[perm] } }
          : entry,
      ),
    );
  };

  const renameRbacRole = (index: number, name: string) => {
    setRbacEntries(rbacEntries.map((entry: any, i: any) => (i === index ? { ...entry, role: name } : entry)));
  };

  // --- ABAC visual builder handlers ---
  const addAbacRule = () => {
    setAbacRules([...abacRules, makeAbacRule()]);
  };

  const removeAbacRule = (ruleId: string) => {
    setAbacRules(abacRules.filter((r: any) => r.id !== ruleId));
  };

  const addAbacCondition = (ruleId: string) => {
    setAbacRules(
      abacRules.map((r: any) =>
        r.id === ruleId ? { ...r, conditions: [...r.conditions, makeCondition()] } : r,
      ),
    );
  };

  const removeAbacCondition = (ruleId: string, condId: string) => {
    setAbacRules(
      abacRules.map((r: any) =>
        r.id === ruleId
          ? { ...r, conditions: r.conditions.filter((c: any) => c.id !== condId) }
          : r,
      ),
    );
  };

  const updateAbacCondition = (ruleId: string, condId: string, field: keyof AbacCondition, value: string) => {
    setAbacRules(
      abacRules.map((r: any) =>
        r.id === ruleId
          ? {
              ...r,
              conditions: r.conditions.map((c: any) =>
                c.id === condId ? { ...c, [field]: value } : c,
              ),
            }
          : r,
      ),
    );
  };

  const updateAbacEffect = (ruleId: string, effect: "allow" | "deny") => {
    setAbacRules(abacRules.map((r: any) => (r.id === ruleId ? { ...r, effect } : r)));
  };

  return (
    <div>
      <div className="mb-6 flex items-center justify-between">
        <h1 className="flex items-center gap-2 text-2xl font-bold dark:text-gray-100">
          <Shield className="h-6 w-6 text-brand-600" /> {t("policies.policyEditor")}
        </h1>
        <div className="flex gap-2">
          <input
            ref={importFileRef}
            type="file"
            accept=".json"
            onChange={handleImportJson}
            className="hidden"
          />
          <button
            onClick={() => importFileRef.current?.click()}
            className="flex items-center gap-1.5 rounded-lg border border-gray-300 px-3 py-2 text-sm font-medium hover:bg-gray-50 dark:border-gray-600 dark:hover:bg-gray-700 dark:text-gray-200"
            title={t("policies.importPolicy")}
          >
            <Upload className="h-4 w-4" /> {t("policies.import")}
          </button>
          <button
            onClick={handleExportJson}
            className="flex items-center gap-1.5 rounded-lg border border-gray-300 px-3 py-2 text-sm font-medium hover:bg-gray-50 dark:border-gray-600 dark:hover:bg-gray-700 dark:text-gray-200"
            title={t("policies.exportPolicy")}
           aria-label="Download">
            <Download className="h-4 w-4" /> {t("policies.export")}
          </button>
          <button
            onClick={handleExportAllJson}
            className="flex items-center gap-1.5 rounded-lg border border-gray-300 px-3 py-2 text-sm font-medium hover:bg-gray-50 dark:border-gray-600 dark:hover:bg-gray-700 dark:text-gray-200"
            title={t("policies.exportAll")}
           aria-label="FileJson">
            <FileJson className="h-4 w-4" /> {t("policies.exportAllBtn")}
          </button>
          {selectedPolicy && (
            <button
              onClick={resetEditor}
              className="rounded-lg border border-gray-300 px-4 py-2 text-sm font-medium hover:bg-gray-50 dark:border-gray-600 dark:hover:bg-gray-700"
             aria-label="Action">
              {t("policies.newPolicy")}
            </button>
          )}
        </div>
      </div>

      {msg && (
        <div role="status" className="mb-4 rounded-lg border border-green-200 bg-green-50 p-3 text-sm text-green-700 dark:border-green-800 dark:bg-green-950 dark:text-green-400">
          {msg}
        </div>
      )}

      {error && (
        <div className="mb-4 rounded-lg border border-red-200 bg-red-50 p-4 text-sm text-red-700">{error}</div>
      )}

      <div className="grid grid-cols-1 gap-6 lg:grid-cols-3">
        {/* Policy List */}
        <div className="lg:col-span-1">
          <div className="rounded-xl border border-gray-200 bg-white shadow-sm dark:border-gray-700 dark:bg-gray-800">
            <div className="border-b border-gray-100 p-4 dark:border-gray-700">
              <h2 className="text-sm font-semibold text-gray-700 dark:text-gray-300">{t("policies.policies")}</h2>
            </div>
            <div className="max-h-[500px] overflow-y-auto">
              {loading ? (
                <div className="flex items-center justify-center gap-2 p-8 text-gray-500">
                  <Loader2 className="h-4 w-4 animate-spin" /> {t("common.loading")}
                </div>
              ) : policies.length === 0 ? (
                <p className="p-8 text-center text-sm text-gray-500">{t("policies.nopoliciesyet")}</p>
              ) : (
                <ul className="divide-y divide-gray-100 dark:divide-gray-700">
                  {policies.map((p: any) => (
                    <li key={p.id || p.name} className="group flex items-center justify-between p-3 hover:bg-gray-50 dark:hover:bg-gray-700">
                      <button onClick={() => selectPolicy(p)} className="flex-1 text-left">
                        <div className="flex items-center gap-2">
                          <p className="text-sm font-medium text-gray-900 dark:text-gray-200">{p.name}</p>
                          {p.effect && (
                            <span className={`rounded px-1.5 py-0.5 text-[10px] font-bold uppercase ${
                              p.effect === "allow"
                                ? "bg-green-100 text-green-700 dark:bg-green-900 dark:text-green-400"
                                : "bg-red-100 text-red-700 dark:bg-red-900 dark:text-red-400"
                            }`}>
                              {p.effect}
                            </span>
                          )}
                        </div>
                        <div className="flex items-center gap-2">
                          <p className="text-xs text-gray-500">{p.rules?.length || 0} rules</p>
                          {typeof p.priority === "number" && (
                            <span className="text-xs text-gray-400">· P:{p.priority}</span>
                          )}
                        </div>
                      </button>
                      {p.id && (
                        <button
                          onClick={() => handleDeletePolicy(p.id!, p.name)}
                          className="rounded p-1 text-gray-400 opacity-0 hover:bg-red-50 hover:text-red-600 group-hover:opacity-100"
                          title={t("common.delete")}
                        >
                          <Trash2 className="h-4 w-4" />
                        </button>
                      )}
                    </li>
                  ))}
                </ul>
              )}
            </div>
          </div>
        </div>

        {/* Editor sections */}
        <div className="space-y-6 lg:col-span-2">
          {/* Rule Builder */}
          <div className="rounded-xl border border-gray-200 bg-white p-6 shadow-sm dark:border-gray-700 dark:bg-gray-800">
            <div className="mb-4 flex items-center justify-between">
              <h2 className="flex items-center gap-2 text-lg font-semibold dark:text-gray-100">
                <FileJson className="h-5 w-5 text-brand-600" />
                {selectedPolicy ? t("policies.editPolicy") : t("policies.createNew")}
              </h2>
              <div className="flex gap-2">
                <button
                  onClick={handleCreatePolicy}
                  className="flex items-center gap-1.5 rounded-lg bg-brand-600 px-4 py-2 text-sm font-medium text-white hover:bg-brand-700"
                 aria-label="Save">
                  <Save className="h-4 w-4" /> {selectedPolicy ? t("policies.update") : t("common.create")}
                </button>
              </div>
            </div>

            <div className="mb-4">
              <label className="mb-1 block text-xs font-medium text-gray-500">{t("policies.policyName")}</label>
              <input
                value={policyName}
                onChange={(e) => setPolicyName(e.target.value)}
                placeholder="e.g. admin-full-access"
                className="w-full rounded-lg border border-gray-300 px-3 py-2 text-sm dark:border-gray-600 dark:bg-gray-700 dark:text-gray-200"
              />
            </div>

            {/* Priority Slider & Effect Toggle */}
            <div className="mb-4 grid grid-cols-2 gap-4">
              <div>
                <label className="mb-1 block text-xs font-medium text-gray-500">
                  {t("policies.priority")}: <span className="font-bold text-brand-600">{policyPriority}</span>
                </label>
                <input
                  type="range"
                  min={0}
                  max={100}
                  value={policyPriority}
                  onChange={(e) => setPolicyPriority(Number(e.target.value))}
                  className="w-full accent-brand-600"
                />
                <div className="flex justify-between text-xs text-gray-400">
                  <span>{t("policies.lowPriority")}</span>
                  <span>{t("policies.highPriority")}</span>
                </div>
              </div>
              <div>
                <label className="mb-1 block text-xs font-medium text-gray-500">{t("policies.defaulteffect")}</label>
                <select
                  value={policyEffect}
                  onChange={(e) => setPolicyEffect(e.target.value as "allow" | "deny")}
                  className={`w-full rounded-lg border px-3 py-2 text-sm font-medium ${
                    policyEffect === "allow"
                      ? "border-green-300 bg-green-50 text-green-700 dark:border-green-800 dark:bg-green-950 dark:text-green-400"
                      : "border-red-300 bg-red-50 text-red-700 dark:border-red-800 dark:bg-red-950 dark:text-red-400"
                  }`}
                >
                  <option value="allow">Allow</option>
                  <option value="deny">Deny</option>
                </select>
              </div>
            </div>

            {/* Basic Rules */}
            <div className="mb-3 flex items-center justify-between">
              <span className="text-sm font-medium text-gray-700 dark:text-gray-300">{t("policies.basicrules")}</span>
              <button
                onClick={addRule}
                className="flex items-center gap-1 rounded-lg border border-gray-300 px-3 py-1.5 text-xs font-medium hover:bg-gray-50 dark:border-gray-600 dark:hover:bg-gray-700"
               aria-label="Plus">
                <Plus className="h-3.5 w-3.5" /> {t("policies.addRule")}
              </button>
            </div>

            {rules.length === 0 ? (
              <p className="py-4 text-center text-sm text-gray-400">{t("policies.noRules")}</p>
            ) : (
              <div className="space-y-2">
                {rules.map((rule: any, i: any) => (
                  <div key={i} className="flex items-center gap-2">
                    <input
                      value={rule.subject}
                      onChange={(e) => updateRule(i, "subject", e.target.value)}
                      placeholder="subject"
                      className="flex-1 rounded border border-gray-300 px-2 py-1.5 text-xs dark:border-gray-600 dark:bg-gray-700 dark:text-gray-200"
                    />
                    <input
                      value={rule.resource}
                      onChange={(e) => updateRule(i, "resource", e.target.value)}
                      placeholder="resource"
                      className="flex-1 rounded border border-gray-300 px-2 py-1.5 text-xs dark:border-gray-600 dark:bg-gray-700 dark:text-gray-200"
                    />
                    <input
                      value={rule.action}
                      onChange={(e) => updateRule(i, "action", e.target.value)}
                      placeholder="action"
                      className="flex-1 rounded border border-gray-300 px-2 py-1.5 text-xs dark:border-gray-600 dark:bg-gray-700 dark:text-gray-200"
                    />
                    <select
                      value={rule.effect}
                      onChange={(e) => updateRule(i, "effect", e.target.value)}
                      className={`rounded border px-2 py-1.5 text-xs font-medium ${
                        rule.effect === "allow"
                          ? "border-green-300 bg-green-50 text-green-700"
                          : "border-red-300 bg-red-50 text-red-700"
                      }`}
                    >
                      <option value="allow">allow</option>
                      <option value="deny">deny</option>
                    </select>
                    <button
                      onClick={() => removeRule(i)}
                      className="rounded p-1 text-gray-400 hover:bg-red-50 hover:text-red-600"
                    >
                      <Trash2 className="h-4 w-4" />
                    </button>
                  </div>
                ))}
              </div>
            )}
          </div>

          {/* RBAC Role-Permission Matrix */}
          <div className="rounded-xl border border-gray-200 bg-white p-6 shadow-sm dark:border-gray-700 dark:bg-gray-800">
            <div className="mb-4 flex items-center justify-between">
              <h2 className="flex items-center gap-2 text-lg font-semibold dark:text-gray-100">
                <Grid3x3 className="h-5 w-5 text-brand-600" /> {t("policies.rbacMatrix")}
              </h2>
              <div className="flex items-center gap-2">
                <button
                  onClick={addRbacRole}
                  className="flex items-center gap-1 rounded-lg border border-gray-300 px-3 py-1.5 text-xs font-medium hover:bg-gray-50 dark:border-gray-600 dark:hover:bg-gray-700"
                 aria-label="Plus">
                  <Plus className="h-3.5 w-3.5" /> {t("policies.addRole")}
                </button>
                <button
                  onClick={() => setRbacCollapsed(!rbacCollapsed)}
                  className="rounded-lg border border-gray-300 p-1.5 hover:bg-gray-50 dark:border-gray-600 dark:hover:bg-gray-700"
                >
                  {rbacCollapsed ? <ChevronRight className="h-4 w-4" /> : <ChevronDown className="h-4 w-4" />}
                </button>
              </div>
            </div>

            {!rbacCollapsed && (
              <>
                <p className="mb-3 text-xs text-gray-500">
                  Grant or revoke permissions per role. Check / uncheck cells to toggle access.
                </p>
                <div className="overflow-x-auto">
                  <table className="w-full">
                    <thead>
                      <tr className="border-b border-gray-200 dark:border-gray-700">
                        <th scope="col" className="px-3 py-2 text-left text-xs font-medium uppercase text-gray-500">Role</th>
                        {PERMISSION_COLUMNS.map((col: any) => (
                          <th scope="col" key={col} className="px-3 py-2 text-center text-xs font-medium uppercase text-gray-500">
                            {col}
                          </th>
                        ))}
                        <th scope="col" className="px-3 py-2"></th>
                      </tr>
                    </thead>
                    <tbody className="divide-y divide-gray-100 dark:divide-gray-700">
                      {rbacEntries.map((entry, roleIdx) => (
                        <tr key={roleIdx} className="hover:bg-gray-50 dark:hover:bg-gray-700/50">
                          <td className="px-3 py-2">
                            <input
                              value={entry.role}
                              onChange={(e) => renameRbacRole(roleIdx, e.target.value)}
                              className="w-full rounded border border-transparent px-2 py-1 text-sm font-medium text-gray-900 hover:border-gray-300 focus:border-brand-500 focus:outline-none dark:bg-gray-700 dark:text-gray-200"
                            />
                          </td>
                          {PERMISSION_COLUMNS.map((perm: any) => (
                            <td key={perm} className="px-3 py-2 text-center">
                              <input
                                type="checkbox"
                                checked={entry.permissions[perm]}
                                onChange={() => toggleRbacPermission(roleIdx, perm)}
                                className="h-4 w-4 rounded border-gray-300 text-brand-600 focus:ring-brand-500"
                              />
                            </td>
                          ))}
                          <td className="px-3 py-2 text-right">
                            <button
                              onClick={() => removeRbacRole(roleIdx)}
                              className="rounded p-1 text-gray-400 hover:bg-red-50 hover:text-red-600"
                            >
                              <Trash2 className="h-4 w-4" />
                            </button>
                          </td>
                        </tr>
                      ))}
                    </tbody>
                  </table>
                </div>
              </>
            )}
          </div>

          {/* ABAC Visual Rule Builder */}
          <div className="rounded-xl border border-gray-200 bg-white p-6 shadow-sm dark:border-gray-700 dark:bg-gray-800">
            <div className="mb-4 flex items-center justify-between">
              <h2 className="flex items-center gap-2 text-lg font-semibold dark:text-gray-100">
                <Layers className="h-5 w-5 text-brand-600" /> {t("policies.abacBuilder")}
              </h2>
              <div className="flex items-center gap-2">
                <button
                  onClick={addAbacRule}
                  className="flex items-center gap-1 rounded-lg border border-gray-300 px-3 py-1.5 text-xs font-medium hover:bg-gray-50 dark:border-gray-600 dark:hover:bg-gray-700"
                 aria-label="Plus">
                  <Plus className="h-3.5 w-3.5" /> {t("policies.addRule")}
                </button>
                <button
                  onClick={() => setAbacCollapsed(!abacCollapsed)}
                  className="rounded-lg border border-gray-300 p-1.5 hover:bg-gray-50 dark:border-gray-600 dark:hover:bg-gray-700"
                >
                  {abacCollapsed ? <ChevronRight className="h-4 w-4" /> : <ChevronDown className="h-4 w-4" />}
                </button>
              </div>
            </div>

            {!abacCollapsed && (
              <>
                <p className="mb-3 text-xs text-gray-500">
                  Build attribute-based rules: IF conditions are met, THEN allow or deny access.
                </p>
                {abacRules.length === 0 ? (
                  <p className="py-4 text-center text-sm text-gray-400">
                    {t("policies.noAbacRules")}
                  </p>
                ) : (
                  <div className="space-y-4">
                    {abacRules.map((rule: any) => (
                      <div
                        key={rule.id}
                        className="rounded-lg border border-gray-200 bg-gray-50 p-4 dark:border-gray-700 dark:bg-gray-900/50"
                      >
                        <div className="mb-3 flex items-center justify-between">
                          <div className="flex items-center gap-2">
                            <span className="text-xs font-bold uppercase text-gray-400">{t("policies.if")}</span>
                          </div>
                          <button
                            onClick={() => removeAbacRule(rule.id)}
                            className="rounded p-1 text-gray-400 hover:bg-red-50 hover:text-red-600"
                          >
                            <Trash2 className="h-4 w-4" />
                          </button>
                        </div>

                        {/* Conditions */}
                        <div className="space-y-2">
                          {rule.conditions.map((cond, condIdx) => (
                            <div key={cond.id} className="flex items-center gap-2">
                              {condIdx === 0 ? (
                                <span className="w-8 text-right text-xs text-gray-400"></span>
                              ) : (
                                <span className="w-8 text-right text-xs font-bold text-gray-500">AND</span>
                              )}
                              <select
                                value={cond.target}
                                onChange={(e) => updateAbacCondition(rule.id, cond.id, "target", e.target.value)}
                                className="rounded border border-gray-300 px-2 py-1.5 text-xs dark:border-gray-600 dark:bg-gray-700 dark:text-gray-200"
                              >
                                <option value="subject">subject</option>
                                <option value="resource">resource</option>
                              </select>
                              <span className="text-xs text-gray-400">has</span>
                              <input
                                value={cond.attribute}
                                onChange={(e) => updateAbacCondition(rule.id, cond.id, "attribute", e.target.value)}
                                placeholder="attribute"
                                className="w-28 rounded border border-gray-300 px-2 py-1.5 text-xs dark:border-gray-600 dark:bg-gray-700 dark:text-gray-200"
                              />
                              <select
                                value={cond.operator}
                                onChange={(e) => updateAbacCondition(rule.id, cond.id, "operator", e.target.value)}
                                className="rounded border border-gray-300 px-2 py-1.5 text-xs dark:border-gray-600 dark:bg-gray-700 dark:text-gray-200"
                              >
                                {OPERATORS.map((op: any) => (
                                  <option key={op.value} value={op.value}>{op.label}</option>
                                ))}
                              </select>
                              <input
                                value={cond.value}
                                onChange={(e) => updateAbacCondition(rule.id, cond.id, "value", e.target.value)}
                                placeholder="value"
                                className="flex-1 rounded border border-gray-300 px-2 py-1.5 text-xs dark:border-gray-600 dark:bg-gray-700 dark:text-gray-200"
                              />
                              <button
                                onClick={() => removeAbacCondition(rule.id, cond.id)}
                                disabled={rule.conditions.length === 1}
                                className="rounded p-1 text-gray-400 hover:bg-red-50 hover:text-red-600 disabled:opacity-30"
                              >
                                <Trash2 className="h-3.5 w-3.5" />
                              </button>
                            </div>
                          ))}
                        </div>

                        {/* Add condition + THEN */}
                        <div className="mt-2 flex items-center gap-2">
                          <button
                            onClick={() => addAbacCondition(rule.id)}
                            className="flex items-center gap-1 text-xs text-brand-600 hover:underline"
                          >
                            <Plus className="h-3 w-3" /> {t("policies.addCondition")}
                          </button>
                        </div>

                        <div className="mt-3 flex items-center gap-2 border-t border-gray-200 pt-3 dark:border-gray-700">
                          <span className="text-xs font-bold uppercase text-gray-400">THEN</span>
                          <select
                            value={rule.effect}
                            onChange={(e) => updateAbacEffect(rule.id, e.target.value as "allow" | "deny")}
                            className={`rounded border px-3 py-1.5 text-xs font-semibold ${
                              rule.effect === "allow"
                                ? "border-green-300 bg-green-50 text-green-700 dark:border-green-800 dark:bg-green-950 dark:text-green-400"
                                : "border-red-300 bg-red-50 text-red-700 dark:border-red-800 dark:bg-red-950 dark:text-red-400"
                            }`}
                          >
                            <option value="allow">ALLOW</option>
                            <option value="deny">DENY</option>
                          </select>
                        </div>
                      </div>
                    ))}
                  </div>
                )}
              </>
            )}
          </div>

          {/* JSON Editor */}
          <div className="rounded-xl border border-gray-200 bg-white p-6 shadow-sm dark:border-gray-700 dark:bg-gray-800">
            <div className="mb-3 flex items-center justify-between">
              <h2 className="text-lg font-semibold dark:text-gray-100">{t("policies.rawjson")}</h2>
              <div className="flex gap-2">
                <button
                  onClick={() => importFileRef.current?.click()}
                  className="flex items-center gap-1 rounded-lg border border-gray-300 px-3 py-1.5 text-xs font-medium hover:bg-gray-50 dark:border-gray-600 dark:hover:bg-gray-700"
                >
                  <Upload className="h-3.5 w-3.5" /> {t("policies.importFile")}
                </button>
                <button
                  onClick={handleExportJson}
                  className="flex items-center gap-1 rounded-lg border border-gray-300 px-3 py-1.5 text-xs font-medium hover:bg-gray-50 dark:border-gray-600 dark:hover:bg-gray-700"
                 aria-label="Download">
                  <Download className="h-3.5 w-3.5" /> {t("policies.export")}
                </button>
                <button
                  onClick={syncJsonToRules}
                  className="rounded-lg border border-gray-300 px-3 py-1.5 text-xs font-medium hover:bg-gray-50 dark:border-gray-600 dark:hover:bg-gray-700"
                 aria-label="Action">
                  {t("policies.syncJsonToRules")}
                </button>
              </div>
            </div>
            <textarea
              value={policyJson}
              onChange={(e) => setPolicyJson(e.target.value)}
              rows={10}
              placeholder='{\n  "name": "my-policy",\n  "rules": []\n}'
              className="w-full rounded-lg border border-gray-300 px-3 py-2 font-mono text-sm dark:border-gray-600 dark:bg-gray-700 dark:text-gray-200"
            />
          </div>

          {/* Dry-Run Test */}
          <div className="rounded-xl border border-gray-200 bg-white p-6 shadow-sm dark:border-gray-700 dark:bg-gray-800">
            <h2 className="mb-4 flex items-center gap-2 text-lg font-semibold dark:text-gray-100">
              <Play className="h-5 w-5 text-brand-600" /> {t("policies.testEvaluator")}
            </h2>
            <div className="grid grid-cols-3 gap-3">
              <div>
                <label className="mb-1 block text-xs font-medium text-gray-500">Subject</label>
                <input
                  value={dryRunSubject}
                  onChange={(e) => setDryRunSubject(e.target.value)}
                  placeholder="user:alice"
                  className="w-full rounded-lg border border-gray-300 px-3 py-2 text-sm dark:border-gray-600 dark:bg-gray-700 dark:text-gray-200"
                />
              </div>
              <div>
                <label className="mb-1 block text-xs font-medium text-gray-500">Resource</label>
                <input
                  value={dryRunResource}
                  onChange={(e) => setDryRunResource(e.target.value)}
                  placeholder="document:123"
                  className="w-full rounded-lg border border-gray-300 px-3 py-2 text-sm dark:border-gray-600 dark:bg-gray-700 dark:text-gray-200"
                />
              </div>
              <div>
                <label className="mb-1 block text-xs font-medium text-gray-500">Action</label>
                <input
                  value={dryRunAction}
                  onChange={(e) => setDryRunAction(e.target.value)}
                  placeholder="read"
                  className="w-full rounded-lg border border-gray-300 px-3 py-2 text-sm dark:border-gray-600 dark:bg-gray-700 dark:text-gray-200"
                />
              </div>
            </div>
            <button
              onClick={handleDryRun}
              disabled={!dryRunSubject || !dryRunResource || !dryRunAction || dryRunLoading}
              className="mt-3 flex items-center gap-1.5 rounded-lg bg-brand-600 px-4 py-2 text-sm font-medium text-white hover:bg-brand-700 disabled:opacity-50"
             aria-label="Loader2">
              {dryRunLoading ? <Loader2 className="h-4 w-4 animate-spin" /> : <Play className="h-4 w-4" />}
              {t("policies.evaluate")}
            </button>

            {dryRunResult && (
              <div
                className={`mt-4 flex items-center gap-3 rounded-lg border p-4 ${
                  dryRunResult.allow
                    ? "border-green-200 bg-green-50 dark:border-green-800 dark:bg-green-950"
                    : "border-red-200 bg-red-50 dark:border-red-800 dark:bg-red-950"
                }`}
              >
                {dryRunResult.allow ? (
                  <CheckCircle className="h-6 w-6 text-green-600" />
                ) : (
                  <XCircle className="h-6 w-6 text-red-600" />
                )}
                <div>
                  <p
                    className={`text-lg font-bold ${
                      dryRunResult.allow ? "text-green-700 dark:text-green-400" : "text-red-700 dark:text-red-400"
                    }`}
                  >
                    {dryRunResult.allow ? "ALLOW" : "DENY"}
                  </p>
                  {dryRunResult.detail && (
                    <p className="text-sm text-gray-500">{dryRunResult.detail}</p>
                  )}
                </div>
              </div>
            )}
          </div>
        </div>
      </div>
    </div>
  );
}
