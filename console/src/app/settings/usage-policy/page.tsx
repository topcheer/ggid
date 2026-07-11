"use client";

import { useState, useEffect, useCallback } from "react";
import { Sliders, Save, Plus, Trash2, MonitorSmartphone } from "lucide-react";

interface ClientPolicy {
  client_id: string;
  client_name: string;
  max_tokens_per_day: number;
  max_sessions: number;
  allowed_ip_ranges: string[];
  enabled: boolean;
}

export default function UsagePolicyPage() {
  const [policies, setPolicies] = useState<ClientPolicy[]>([]);
  const [loading, setLoading] = useState(false);
  const [editingId, setEditingId] = useState<string | null>(null);
  const [editForm, setEditForm] = useState<ClientPolicy | null>(null);
  const [newIpRange, setNewIpRange] = useState("");
  const [saving, setSaving] = useState(false);

  const fetchPolicies = useCallback(async () => {
    setLoading(true);
    try {
      const res = await fetch("/api/v1/oauth/usage-policy", { headers: { "X-Tenant-ID": "00000000-0000-0000-0000-000000000001" } });
      if (res.ok) {
        const data = await res.json();
        setPolicies(data.policies || data || []);
      }
    } catch {
      /* noop */
    } finally {
      setLoading(false);
    }
  }, []);

  useEffect(() => {
    fetchPolicies();
  }, [fetchPolicies]);

  const startEdit = (p: ClientPolicy) => {
    setEditingId(p.client_id);
    setEditForm({ ...p, allowed_ip_ranges: [...p.allowed_ip_ranges] });
    setNewIpRange("");
  };

  const saveEdit = async () => {
    if (!editForm) return;
    setSaving(true);
    try {
      await fetch(`/api/v1/oauth/usage-policy/${editForm.client_id}`, {
        method: "PUT",
        headers: { "Content-Type": "application/json", "X-Tenant-ID": "00000000-0000-0000-0000-000000000001" },
        body: JSON.stringify(editForm),
      });
      setPolicies((prev) => prev.map((p) => p.client_id === editForm.client_id ? editForm : p));
      setEditingId(null);
      setEditForm(null);
    } catch {
      /* noop */
    } finally {
      setSaving(false);
    }
  };

  const addIpRange = () => {
    if (!editForm || !newIpRange) return;
    setEditForm({ ...editForm, allowed_ip_ranges: [...editForm.allowed_ip_ranges, newIpRange] });
    setNewIpRange("");
  };

  const removeIpRange = (idx: number) => {
    if (!editForm) return;
    setEditForm({ ...editForm, allowed_ip_ranges: editForm.allowed_ip_ranges.filter((_, i) => i !== idx) });
  };

  return (
    <div className="space-y-6">
      <div>
        <h1 className="text-2xl font-bold flex items-center gap-2"><Sliders className="w-6 h-6 text-blue-500" /> Usage Policy</h1>
        <p className="text-sm text-gray-500 mt-1">Configure per-client token limits, session caps, and IP restrictions.</p>
      </div>

      {/* Policy cards */}
      <div className="space-y-4">
        {policies.map((p) => (
          <div key={p.client_id} className="rounded-lg border dark:border-gray-800 overflow-hidden">
            {editingId === p.client_id && editForm ? (
              /* Edit mode */
              <div className="p-4 space-y-4">
                <div className="flex items-center justify-between">
                  <h3 className="font-semibold flex items-center gap-2"><MonitorSmartphone className="w-4 h-4" /> {editForm.client_name}</h3>
                  <span className="text-xs text-gray-400 font-mono">{editForm.client_id}</span>
                </div>

                <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
                  <div>
                    <label className="text-sm font-medium">Max Tokens / Day</label>
                    <input type="number" value={editForm.max_tokens_per_day} onChange={(e) => setEditForm({ ...editForm, max_tokens_per_day: parseInt(e.target.value) || 0 })} className="w-full mt-1 px-3 py-2 rounded-lg border dark:border-gray-700 dark:bg-gray-800 text-sm" />
                  </div>
                  <div>
                    <label className="text-sm font-medium">Max Concurrent Sessions</label>
                    <input type="number" value={editForm.max_sessions} onChange={(e) => setEditForm({ ...editForm, max_sessions: parseInt(e.target.value) || 0 })} className="w-full mt-1 px-3 py-2 rounded-lg border dark:border-gray-700 dark:bg-gray-800 text-sm" />
                  </div>
                </div>

                <div>
                  <label className="text-sm font-medium">Allowed IP Ranges (CIDR)</label>
                  <div className="mt-1 space-y-1">
                    {editForm.allowed_ip_ranges.map((range, i) => (
                      <div key={i} className="flex items-center gap-2">
                        <span className="flex-1 px-3 py-1.5 rounded-lg border dark:border-gray-700 dark:bg-gray-800 text-sm font-mono">{range}</span>
                        <button onClick={() => removeIpRange(i)} className="p-1.5 rounded-lg text-red-500 hover:bg-red-50 dark:hover:bg-red-900/20"><Trash2 className="w-4 h-4" /></button>
                      </div>
                    ))}
                    <div className="flex items-center gap-2">
                      <input type="text" value={newIpRange} onChange={(e) => setNewIpRange(e.target.value)} placeholder="10.0.0.0/8" className="flex-1 px-3 py-1.5 rounded-lg border dark:border-gray-700 dark:bg-gray-800 text-sm font-mono" />
                      <button onClick={addIpRange} className="p-1.5 rounded-lg bg-blue-600 text-white hover:bg-blue-700"><Plus className="w-4 h-4" /></button>
                    </div>
                  </div>
                </div>

                <div className="flex items-center gap-4">
                  <label className="flex items-center gap-2 text-sm cursor-pointer">
                    <input type="checkbox" checked={editForm.enabled} onChange={(e) => setEditForm({ ...editForm, enabled: e.target.checked })} className="rounded" />
                    Enabled
                  </label>
                </div>

                <div className="flex justify-end gap-2">
                  <button onClick={() => { setEditingId(null); setEditForm(null); }} className="px-4 py-2 rounded-lg border dark:border-gray-700 text-sm">Cancel</button>
                  <button onClick={saveEdit} disabled={saving} className="px-4 py-2 rounded-lg bg-blue-600 text-white text-sm font-medium hover:bg-blue-700 disabled:opacity-50 flex items-center gap-1"><Save className="w-4 h-4" /> {saving ? "Saving..." : "Save"}</button>
                </div>
              </div>
            ) : (
              /* View mode */
              <div className="p-4">
                <div className="flex items-center justify-between">
                  <div className="flex items-center gap-2">
                    <MonitorSmartphone className="w-4 h-4 text-gray-400" />
                    <span className="font-semibold">{p.client_name}</span>
                    <span className={`px-2 py-0.5 rounded text-xs ${p.enabled ? "bg-green-100 text-green-800 dark:bg-green-900/30 dark:text-green-400" : "bg-gray-100 text-gray-500 dark:bg-gray-800"}`}>{p.enabled ? "Active" : "Disabled"}</span>
                  </div>
                  <button onClick={() => startEdit(p)} className="text-blue-600 hover:underline text-sm font-medium">Edit</button>
                </div>
                <div className="grid grid-cols-2 md:grid-cols-3 gap-4 mt-4">
                  <div>
                    <span className="text-xs text-gray-400">Max Tokens/Day</span>
                    <p className="text-lg font-bold mt-0.5">{p.max_tokens_per_day.toLocaleString()}</p>
                  </div>
                  <div>
                    <span className="text-xs text-gray-400">Max Sessions</span>
                    <p className="text-lg font-bold mt-0.5">{p.max_sessions}</p>
                  </div>
                  <div>
                    <span className="text-xs text-gray-400">IP Ranges</span>
                    <p className="text-sm font-mono mt-0.5">{p.allowed_ip_ranges.length || "Any"}</p>
                  </div>
                </div>
                {p.allowed_ip_ranges.length > 0 && (
                  <div className="flex flex-wrap gap-1 mt-3">
                    {p.allowed_ip_ranges.map((r, i) => (
                      <span key={i} className="px-2 py-0.5 rounded text-xs font-mono bg-gray-100 dark:bg-gray-800 text-gray-600">{r}</span>
                    ))}
                  </div>
                )}
              </div>
            )}
          </div>
        ))}
        {policies.length === 0 && !loading && (
          <p className="text-sm text-gray-500 text-center py-8">No usage policies configured.</p>
        )}
        {loading && <p className="text-sm text-gray-500 text-center py-8">Loading...</p>}
      </div>
    </div>
  );
}
