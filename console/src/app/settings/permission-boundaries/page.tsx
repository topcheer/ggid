"use client";

import { useState, useEffect, useCallback } from "react";
import { Shield, Edit3, Save, X, Plus, Trash2, AlertTriangle, CheckCircle2 } from "lucide-react";
import { useTranslations } from "@/lib/i18n";

interface PermissionBoundary {
  id: string;
  role: string;
  max_scopes: string[];
  denied_actions: string[];
  violation_count: number;
  last_updated: string;
}

interface Violation {
  user: string;
  scope: string;
  action: string;
  timestamp: string;
}

export default function PermissionBoundariesPage() {
  const t = useTranslations();

  const [boundaries, setBoundaries] = useState<PermissionBoundary[]>([]);
  const [loading, setLoading] = useState(false);
  const [editingId, setEditingId] = useState<string | null>(null);
  const [editForm, setEditForm] = useState<PermissionBoundary | null>(null);
  const [newScope, setNewScope] = useState("");
  const [newDenied, setNewDenied] = useState("");
  const [saving, setSaving] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [showViolations, setShowViolations] = useState<string | null>(null);
  const [violations, setViolations] = useState<Violation[]>([]);

  const fetchData = useCallback(async () => {
    setLoading(true);
    setError(null);
    try {
      const res = await fetch("/api/v1/policy/permission-boundaries", { headers: { "X-Tenant-ID": "00000000-0000-0000-0000-000000000001" } });
      if (res.ok) {
        const data = await res.json();
        setBoundaries(data.boundaries || data || []);
      } else {
        setError(`Failed to load boundaries: ${res.status} ${res.statusText}`);
      }
    } catch (err) {
      setError(err instanceof Error ? err.message : "Failed to load permission boundaries");
    } finally { setLoading(false); }
  }, []);

  useEffect(() => { fetchData(); }, [fetchData]);

  const startEdit = (b: PermissionBoundary) => {
    setEditingId(b.id);
    setEditForm({ ...b, max_scopes: [...b.max_scopes], denied_actions: [...b.denied_actions] });
    setNewScope("");
    setNewDenied("");
  };

  const saveEdit = async () => {
    if (!editForm) return;
    setSaving(true);
    setError(null);
    try {
      const res = await fetch(`/api/v1/policy/permission-boundaries/${editForm.id}`, {
        method: "PUT",
        headers: { "Content-Type": "application/json", "X-Tenant-ID": "00000000-0000-0000-0000-000000000001" },
        body: JSON.stringify(editForm),
      });
      if (!res.ok) {
        setError(`Save failed: ${res.status} ${res.statusText}`);
      } else {
        setBoundaries((prev) => prev.map((b) => b.id === editForm.id ? editForm : b));
        setEditingId(null);
        setEditForm(null);
      }
    } catch (err) {
      setError(err instanceof Error ? err.message : "Save failed");
    } finally { setSaving(false); }
  };

  const fetchViolations = async (roleId: string) => {
    try {
      const res = await fetch(`/api/v1/policy/permission-boundaries/${roleId}/violations`, { headers: { "X-Tenant-ID": "00000000-0000-0000-0000-000000000001" } });
      if (res.ok) {
        const data = await res.json();
        setViolations(data.violations || data || []);
      } else {
        setError(`Failed to load violations: ${res.status} ${res.statusText}`);
      }
    } catch (err) {
      setError(err instanceof Error ? err.message : "Failed to load violations");
    }
  };

  const totalViolations = boundaries.reduce((s, b) => s + b.violation_count, 0);

  return (
    <div className="space-y-6">
      {error && <div className="flex items-center gap-2 rounded-lg border border-red-200 bg-red-50 px-4 py-3 text-sm text-red-700 dark:border-red-800 dark:bg-red-900/20 dark:text-red-400"><AlertTriangle className="h-4 w-4 shrink-0" />{error}<button onClick={() => setError(null)} aria-label="Dismiss error" className="ml-auto text-red-700 hover:text-red-900"><X className="h-4 w-4" /></button></div>}
      <div>
        <h1 className="text-2xl font-bold flex items-center gap-2"><Shield className="w-6 h-6 text-blue-500" /> {t("permissionBoundaries.title")}</h1>
        <p className="text-sm text-gray-500 mt-1">Configure per-role scope limits and denied actions to enforce least privilege.</p>
      </div>

      <div className="grid grid-cols-2 md:grid-cols-3 gap-4">
        <div className="rounded-lg border p-4 dark:border-gray-800"><span className="text-sm text-gray-500">Roles with Boundaries</span><p className="text-2xl font-bold mt-1">{boundaries.length}</p></div>
        <div className="rounded-lg border p-4 dark:border-gray-800"><span className="text-sm text-gray-500">Total Violations</span><p className="text-2xl font-bold mt-1 text-red-600">{totalViolations}</p></div>
        <div className="rounded-lg border p-4 dark:border-gray-800"><span className="text-sm text-gray-500">Clean Roles</span><p className="text-2xl font-bold mt-1 text-green-600">{boundaries.filter((b) => b.violation_count === 0).length}</p></div>
      </div>

      <div className="space-y-4">
        {boundaries.map((b) => (
          <div key={b.id} className="rounded-lg border dark:border-gray-800 overflow-hidden">
            {editingId === b.id && editForm ? (
              <div className="p-4 space-y-4">
                <div className="flex items-center justify-between">
                  <h3 className="font-semibold flex items-center gap-2"><Shield className="w-4 h-4" /> {editForm.role}</h3>
                  <button onClick={() => { setEditingId(null); setEditForm(null); }} aria-label="Cancel editing"><X className="w-5 h-5 text-gray-400" /></button>
                </div>
                <div>
                  <label className="text-sm font-medium">Max Scopes ({editForm.max_scopes.length})</label>
                  <div className="mt-1 space-y-1">
                    {editForm.max_scopes.map((s, i) => (
                      <div key={i} className="flex items-center gap-2"><span className="flex-1 px-3 py-1.5 rounded border dark:border-gray-700 dark:bg-gray-800 text-sm font-mono">{s}</span><button onClick={() => setEditForm({ ...editForm, max_scopes: editForm.max_scopes.filter((_, idx) => idx !== i) })} aria-label="Remove scope" className="p-1 text-red-400"><Trash2 className="w-4 h-4" /></button></div>
                    ))}
                    <div className="flex items-center gap-2"><input type="text" value={newScope} onChange={(e) => setNewScope(e.target.value)} placeholder="read:users" className="flex-1 px-3 py-1.5 rounded border dark:border-gray-700 dark:bg-gray-800 text-sm font-mono" /><button onClick={() => { if (newScope.trim()) { setEditForm({ ...editForm, max_scopes: [...editForm.max_scopes, newScope.trim()] }); setNewScope(""); } }} aria-label="Add scope" className="p-1.5 rounded bg-blue-600 text-white"><Plus className="w-4 h-4" /></button></div>
                  </div>
                </div>
                <div>
                  <label className="text-sm font-medium">Denied Actions ({editForm.denied_actions.length})</label>
                  <div className="mt-1 space-y-1">
                    {editForm.denied_actions.map((a, i) => (
                      <div key={i} className="flex items-center gap-2"><span className="flex-1 px-3 py-1.5 rounded border dark:border-gray-700 dark:bg-gray-800 text-sm font-mono">{a}</span><button onClick={() => setEditForm({ ...editForm, denied_actions: editForm.denied_actions.filter((_, idx) => idx !== i) })} aria-label="Remove denied action" className="p-1 text-red-400"><Trash2 className="w-4 h-4" /></button></div>
                    ))}
                    <div className="flex items-center gap-2"><input type="text" value={newDenied} onChange={(e) => setNewDenied(e.target.value)} placeholder="delete:organizations" className="flex-1 px-3 py-1.5 rounded border dark:border-gray-700 dark:bg-gray-800 text-sm font-mono" /><button onClick={() => { if (newDenied.trim()) { setEditForm({ ...editForm, denied_actions: [...editForm.denied_actions, newDenied.trim()] }); setNewDenied(""); } }} aria-label="Add denied action" className="p-1.5 rounded bg-blue-600 text-white"><Plus className="w-4 h-4" /></button></div>
                  </div>
                </div>
                <div className="flex justify-end gap-2">
                  <button onClick={() => { setEditingId(null); setEditForm(null); }} aria-label="Cancel changes" className="px-4 py-2 rounded-lg border dark:border-gray-700 text-sm">Cancel</button>
                  <button onClick={saveEdit} disabled={saving} aria-label="Save permission boundary" className="px-4 py-2 rounded-lg bg-blue-600 text-white text-sm font-medium hover:bg-blue-700 disabled:opacity-50 flex items-center gap-1"><Save className="w-4 h-4" /> {saving ? "Saving..." : "Save"}</button>
                </div>
              </div>
            ) : (
              <div className="p-4">
                <div className="flex items-center justify-between">
                  <div className="flex items-center gap-2">
                    <Shield className="w-4 h-4 text-gray-400" />
                    <span className="font-semibold font-mono">{b.role}</span>
                    {b.violation_count > 0 ? (
                      <button onClick={() => { setShowViolations(showViolations === b.id ? null : b.id); if (showViolations !== b.id) fetchViolations(b.id); }} aria-label={`View ${b.violation_count} violations for ${b.role}`} className="px-2 py-0.5 rounded text-xs bg-red-100 text-red-800 dark:bg-red-900/30 dark:text-red-400 hover:underline">{b.violation_count} violations</button>
                    ) : (
                      <span className="flex items-center gap-1 text-xs text-green-600"><CheckCircle2 className="w-3 h-3" /> No violations</span>
                    )}
                  </div>
                  <button onClick={() => startEdit(b)} aria-label={`Edit boundary for ${b.role}`} className="text-blue-600 hover:underline text-sm font-medium flex items-center gap-1"><Edit3 className="w-3 h-3" /> Edit</button>
                </div>
                <div className="grid grid-cols-2 gap-4 mt-4">
                  <div>
                    <span className="text-xs text-gray-400">Max Scopes ({b.max_scopes.length})</span>
                    <div className="flex flex-wrap gap-1 mt-1">{b.max_scopes.map((s, i) => <span key={i} className="px-2 py-0.5 rounded text-xs bg-blue-100 dark:bg-blue-900/30 dark:text-blue-400 font-mono">{s}</span>)}{b.max_scopes.length === 0 && <span className="text-xs text-gray-400">Any</span>}</div>
                  </div>
                  <div>
                    <span className="text-xs text-gray-400">Denied Actions ({b.denied_actions.length})</span>
                    <div className="flex flex-wrap gap-1 mt-1">{b.denied_actions.map((a, i) => <span key={i} className="px-2 py-0.5 rounded text-xs bg-red-100 dark:bg-red-900/30 dark:text-red-400 font-mono">{a}</span>)}{b.denied_actions.length === 0 && <span className="text-xs text-gray-400">None</span>}</div>
                  </div>
                </div>
                {/* Violation preview */}
                {showViolations === b.id && violations.length > 0 && (
                  <div className="mt-3 rounded-lg bg-red-50 dark:bg-red-900/20 p-3">
                    <h4 className="text-sm font-medium flex items-center gap-2 mb-2"><AlertTriangle className="w-4 h-4 text-red-500" /> Recent Violations</h4>
                    <div className="space-y-1 max-h-32 overflow-y-auto">
                      {violations.map((v, i) => (
                        <div key={i} className="flex items-center justify-between text-xs">
                          <span><span className="font-medium">{v.user}</span> attempted <span className="font-mono">{v.action}</span> on <span className="font-mono">{v.scope}</span></span>
                          <span className="text-gray-400">{v.timestamp}</span>
                        </div>
                      ))}
                    </div>
                  </div>
                )}
                <p className="text-xs text-gray-400 mt-3">Last updated: {b.last_updated}</p>
              </div>
            )}
          </div>
        ))}
        {boundaries.length === 0 && !loading && <p className="text-sm text-gray-500 text-center py-8">No permission boundaries configured.</p>}
      </div>
    </div>
  );
}
