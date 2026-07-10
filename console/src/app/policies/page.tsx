"use client";

import { useState, useEffect, useCallback } from "react";
import { useApi } from "@/lib/api";
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
}

export default function PoliciesPage() {
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
    setPolicyJson(JSON.stringify(p, null, 2));
  };

  const handleCreatePolicy = async () => {
    const payload = {
      name: policyName || "Untitled Policy",
      rules: rules,
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
    setRules(rules.map((r, i) => (i === index ? { ...r, [field]: value } : r)));
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
  };

  return (
    <div>
      <div className="mb-6 flex items-center justify-between">
        <h1 className="flex items-center gap-2 text-2xl font-bold dark:text-gray-100">
          <Shield className="h-6 w-6 text-brand-600" /> Policy Editor
        </h1>
        <div className="flex gap-2">
          {selectedPolicy && (
            <button
              onClick={resetEditor}
              className="rounded-lg border border-gray-300 px-4 py-2 text-sm font-medium hover:bg-gray-50 dark:border-gray-600 dark:hover:bg-gray-700"
            >
              New Policy
            </button>
          )}
        </div>
      </div>

      {msg && (
        <div className="mb-4 rounded-lg border border-green-200 bg-green-50 p-3 text-sm text-green-700 dark:border-green-800 dark:bg-green-950 dark:text-green-400">
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
              <h2 className="text-sm font-semibold text-gray-700 dark:text-gray-300">Policies</h2>
            </div>
            <div className="max-h-[500px] overflow-y-auto">
              {loading ? (
                <div className="flex items-center justify-center gap-2 p-8 text-gray-500">
                  <Loader2 className="h-4 w-4 animate-spin" /> Loading...
                </div>
              ) : policies.length === 0 ? (
                <p className="p-8 text-center text-sm text-gray-500">No policies yet</p>
              ) : (
                <ul className="divide-y divide-gray-100 dark:divide-gray-700">
                  {policies.map((p) => (
                    <li key={p.id || p.name} className="group flex items-center justify-between p-3 hover:bg-gray-50 dark:hover:bg-gray-700">
                      <button onClick={() => selectPolicy(p)} className="flex-1 text-left">
                        <p className="text-sm font-medium text-gray-900 dark:text-gray-200">{p.name}</p>
                        <p className="text-xs text-gray-500">{p.rules?.length || 0} rules</p>
                      </button>
                      {p.id && (
                        <button
                          onClick={() => handleDeletePolicy(p.id!, p.name)}
                          className="rounded p-1 text-gray-400 opacity-0 hover:bg-red-50 hover:text-red-600 group-hover:opacity-100"
                          title="Delete"
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

        {/* Editor + Dry Run */}
        <div className="space-y-6 lg:col-span-2">
          {/* Rule Builder */}
          <div className="rounded-xl border border-gray-200 bg-white p-6 shadow-sm dark:border-gray-700 dark:bg-gray-800">
            <div className="mb-4 flex items-center justify-between">
              <h2 className="flex items-center gap-2 text-lg font-semibold dark:text-gray-100">
                <FileJson className="h-5 w-5 text-brand-600" />
                {selectedPolicy ? "Edit Policy" : "Create New Policy"}
              </h2>
              <div className="flex gap-2">
                <button
                  onClick={handleCreatePolicy}
                  className="flex items-center gap-1.5 rounded-lg bg-brand-600 px-4 py-2 text-sm font-medium text-white hover:bg-brand-700"
                >
                  <Save className="h-4 w-4" /> {selectedPolicy ? "Update" : "Create"}
                </button>
              </div>
            </div>

            <div className="mb-4">
              <label className="mb-1 block text-xs font-medium text-gray-500">Policy Name</label>
              <input
                value={policyName}
                onChange={(e) => setPolicyName(e.target.value)}
                placeholder="e.g. admin-full-access"
                className="w-full rounded-lg border border-gray-300 px-3 py-2 text-sm dark:border-gray-600 dark:bg-gray-700 dark:text-gray-200"
              />
            </div>

            {/* Rules */}
            <div className="mb-3 flex items-center justify-between">
              <span className="text-sm font-medium text-gray-700 dark:text-gray-300">Rules</span>
              <button
                onClick={addRule}
                className="flex items-center gap-1 rounded-lg border border-gray-300 px-3 py-1.5 text-xs font-medium hover:bg-gray-50 dark:border-gray-600 dark:hover:bg-gray-700"
              >
                <Plus className="h-3.5 w-3.5" /> Add Rule
              </button>
            </div>

            {rules.length === 0 ? (
              <p className="py-4 text-center text-sm text-gray-400">No rules. Click "Add Rule" to start.</p>
            ) : (
              <div className="space-y-2">
                {rules.map((rule, i) => (
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

          {/* JSON Editor */}
          <div className="rounded-xl border border-gray-200 bg-white p-6 shadow-sm dark:border-gray-700 dark:bg-gray-800">
            <div className="mb-3 flex items-center justify-between">
              <h2 className="text-lg font-semibold dark:text-gray-100">Raw JSON</h2>
              <button
                onClick={syncJsonToRules}
                className="rounded-lg border border-gray-300 px-3 py-1.5 text-xs font-medium hover:bg-gray-50 dark:border-gray-600 dark:hover:bg-gray-700"
              >
                Sync JSON → Rules
              </button>
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
              <Play className="h-5 w-5 text-brand-600" /> Dry-Run Test
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
            >
              {dryRunLoading ? <Loader2 className="h-4 w-4 animate-spin" /> : <Play className="h-4 w-4" />}
              Evaluate
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
