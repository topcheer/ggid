"use client";

import { useState, useEffect, useCallback } from "react";
import { Webhook, Plus, Trash2, X, Save, Send, Settings, Zap } from "lucide-react";
import { useTranslations } from "@/lib/i18n";
import { authHeader, isAuthenticated } from "@/lib/auth-helpers";

interface ProvisioningWebhook {
  id: string;
  url: string;
  events: string[];
  secret_masked: string;
  enabled: boolean;
  last_triggered: string | null;
  last_status: string;
  delivery_count: number;
  failure_count: number;
}

const eventTypes = ["user.created", "user.updated", "user.deleted", "role.granted", "role.revoked", "org.member_added", "org.member_removed"];

export default function ProvisioningWebhooksPage() {
  const t = useTranslations();

  const [webhooks, setWebhooks] = useState<ProvisioningWebhook[]>([]);
  const [loading, setLoading] = useState(false);
  const [showCreate, setShowCreate] = useState(false);
  const [editId, setEditId] = useState<string | null>(null);
  const [form, setForm] = useState({ url: "", events: [] as string[], secret: "" });
  const [saving, setSaving] = useState(false);
  const [testingId, setTestingId] = useState<string | null>(null);
  const [deleteId, setDeleteId] = useState<string | null>(null);

  const fetchWebhooks = useCallback(async () => {
    setLoading(true);
    try {
      const res = await fetch("/api/v1/identity/provisioning-webhooks", { headers: { ...authHeader(), "X-Tenant-ID": "00000000-0000-0000-0000-000000000001" } });
      if (res.ok) {
        const data = await res.json();
        setWebhooks(data.webhooks || data || []);
      }
    } catch { /* noop */ }
    finally { setLoading(false); }
  }, []);

  useEffect(() => { fetchWebhooks(); }, [fetchWebhooks]);

  const save = async () => {
    if (!form.url || form.events.length === 0) return;
    setSaving(true);
    try {
      const url = editId ? `/api/v1/identity/provisioning-webhooks/${editId}` : "/api/v1/identity/provisioning-webhooks";
      const method = editId ? "PUT" : "POST";
      await fetch(url, {
        method,
        headers: { "Content-Type": "application/json", "X-Tenant-ID": "00000000-0000-0000-0000-000000000001" },
        body: JSON.stringify(form),
      });
      setShowCreate(false);
      setEditId(null);
      setForm({ url: "", events: [], secret: "" });
      fetchWebhooks();
    } catch { /* noop */ }
    finally { setSaving(false); }
  };

  const startEdit = (w: ProvisioningWebhook) => {
    setEditId(w.id);
    setForm({ url: w.url, events: [...w.events], secret: "" });
    setShowCreate(true);
  };

  const toggleEvent = (event: string) => {
    setForm((prev) => ({
      ...prev,
      events: prev.events.includes(event) ? prev.events.filter((e) => e !== event) : [...prev.events, event],
    }));
  };

  const testWebhook = async (id: string) => {
    setTestingId(id);
    try {
      await fetch(`/api/v1/identity/provisioning-webhooks/${id}/test`, { method: "POST", headers: { ...authHeader(), "X-Tenant-ID": "00000000-0000-0000-0000-000000000001" } });
    } catch { /* noop */ }
    finally { setTestingId(null); }
  };

  const doDelete = async () => {
    if (!deleteId) return;
    try {
      await fetch(`/api/v1/identity/provisioning-webhooks/${deleteId}`, { method: "DELETE", headers: { ...authHeader(), "X-Tenant-ID": "00000000-0000-0000-0000-000000000001" } });
      setWebhooks((prev) => prev.filter((w) => w.id !== deleteId));
      setDeleteId(null);
    } catch { /* noop */ }
  };

  return (
    <div className="space-y-6">
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-2xl font-bold flex items-center gap-2"><Webhook className="w-6 h-6 text-blue-500" /> {t("provisioningWebhooks.title")}</h1>
          <p className="text-sm text-gray-500 mt-1">Configure SCIM provisioning webhooks for lifecycle events.</p>
        </div>
        <button onClick={() => { setEditId(null); setForm({ url: "", events: [], secret: "" }); setShowCreate(true); }} className="px-4 py-2 rounded-lg bg-blue-600 text-white text-sm font-medium hover:bg-blue-700 flex items-center gap-2"><Plus className="w-4 h-4" /> Add Webhook</button>
      </div>

      {/* Webhook list */}
      <div className="space-y-3">
        {webhooks.map((w) => (
          <div key={w.id} className="rounded-lg border dark:border-gray-800 p-4">
            <div className="flex items-center justify-between">
              <div className="flex-1">
                <div className="flex items-center gap-2">
                  <span className="font-medium text-sm font-mono truncate">{w.url}</span>
                  <span className={`px-2 py-0.5 rounded text-xs ${w.enabled ? "bg-green-100 text-green-800 dark:bg-green-900/30 dark:text-green-400" : "bg-gray-100 text-gray-500 dark:bg-gray-800"}`}>{w.enabled ? "Active" : "Disabled"}</span>
                </div>
                <div className="flex flex-wrap gap-1 mt-2">
                  {w.events.map((e: any, i: number) => <span key={i} className="px-1.5 py-0.5 rounded text-xs bg-purple-100 dark:bg-purple-900/30 dark:text-purple-400 font-mono">{e}</span>)}
                </div>
                <div className="flex items-center gap-3 mt-2 text-xs text-gray-400">
                  <span>Last triggered: {w.last_triggered || "Never"}</span>
                  <span className={`font-medium ${w.last_status === "ok" ? "text-green-600" : w.last_status === "failed" ? "text-red-600" : "text-gray-400"}`}>{w.last_status || "-"}</span>
                  <span>{w.delivery_count} deliveries, {w.failure_count} failures</span>
                </div>
              </div>
              <div className="flex items-center gap-2 ml-4">
                <button onClick={() => testWebhook(w.id)} disabled={testingId === w.id} className="px-3 py-1.5 rounded-lg text-xs font-medium text-blue-600 border border-blue-200 dark:border-blue-900 hover:bg-blue-50 dark:hover:bg-blue-900/20 flex items-center gap-1"><Send className="w-3 h-3" /> {testingId === w.id ? "Testing..." : "Test"}</button>
                <button onClick={() => startEdit(w)} className="p-1.5 rounded-lg text-gray-400 hover:bg-gray-100 dark:hover:bg-gray-800"><Settings className="w-4 h-4" /></button>
                <button onClick={() => setDeleteId(w.id)} className="p-1.5 rounded-lg text-red-400 hover:bg-red-50 dark:hover:bg-red-900/20"><Trash2 className="w-4 h-4" /></button>
              </div>
            </div>
          </div>
        ))}
        {webhooks.length === 0 && !loading && <p className="text-sm text-gray-500 text-center py-8">No webhooks configured.</p>}
      </div>

      {/* Create/Edit modal */}
      {showCreate && (
        <div className="fixed inset-0 bg-black/50 flex items-center justify-center z-50" onClick={() => setShowCreate(false)}>
          <div role="dialog" aria-modal="true" className="bg-white dark:bg-gray-900 rounded-lg shadow-xl max-w-md w-full mx-4" onClick={(e) => e.stopPropagation()}>
            <div className="flex items-center justify-between px-6 py-4 border-b dark:border-gray-800">
              <h3 className="font-semibold flex items-center gap-2"><Zap className="w-5 h-5 text-blue-500" /> {editId ? "Edit" : "Add"} Webhook</h3>
              <button onClick={() => setShowCreate(false)} aria-label="Close"><X className="w-5 h-5 text-gray-400" /></button>
            </div>
            <div className="px-6 py-4 space-y-3">
              <div><label className="text-sm font-medium">URL</label><input aria-label="https://api.example.com/webhooks/scim" type="text" value={form.url} onChange={(e) => setForm({ ...form, url: e.target.value })} placeholder="https://api.example.com/webhooks/scim" className="w-full mt-1 px-3 py-2 rounded-lg border dark:border-gray-700 dark:bg-gray-800 text-sm font-mono" /></div>
              <div>
                <label className="text-sm font-medium">Events</label>
                <div className="grid grid-cols-2 gap-1 mt-1">
                  {eventTypes.map((e) => (
                    <label key={e} className="flex items-center gap-2 text-xs cursor-pointer">
                      <input aria-label="Form" type="checkbox" checked={form.events.includes(e)} onChange={() => toggleEvent(e)} className="rounded" />
                      <span className="font-mono">{e}</span>
                    </label>
                  ))}
                </div>
              </div>
              <div><label className="text-sm font-medium">Secret {editId && "(leave empty to keep current)"}</label><input autoComplete="current-password" type="password" value={form.secret} onChange={(e) => setForm({ ...form, secret: e.target.value })} placeholder="whsec_..." className="w-full mt-1 px-3 py-2 rounded-lg border dark:border-gray-700 dark:bg-gray-800 text-sm font-mono" /></div>
            </div>
            <div className="flex justify-end gap-2 px-6 py-4 border-t dark:border-gray-800">
              <button onClick={() => setShowCreate(false)} className="px-4 py-2 rounded-lg border dark:border-gray-700 text-sm">Cancel</button>
              <button aria-label="Save" onClick={save} disabled={saving || !form.url || form.events.length === 0} className="px-4 py-2 rounded-lg bg-blue-600 text-white text-sm font-medium hover:bg-blue-700 disabled:opacity-50 flex items-center gap-1"><Save className="w-4 h-4" /> {saving ? "Saving..." : "Save"}</button>
            </div>
          </div>
        </div>
      )}

      {/* Delete confirmation */}
      {deleteId && (
        <div className="fixed inset-0 bg-black/50 flex items-center justify-center z-50" onClick={() => setDeleteId(null)}>
          <div role="dialog" aria-modal="true" className="bg-white dark:bg-gray-900 rounded-lg shadow-xl max-w-sm w-full mx-4" onClick={(e) => e.stopPropagation()}>
            <div className="px-6 py-4"><p className="text-sm">Delete this webhook? All future events will stop being delivered to this URL.</p></div>
            <div className="flex justify-end gap-2 px-6 py-4 border-t dark:border-gray-800">
              <button onClick={() => setDeleteId(null)} className="px-4 py-2 rounded-lg border dark:border-gray-700 text-sm">Cancel</button>
              <button onClick={doDelete} className="px-4 py-2 rounded-lg bg-red-600 text-white text-sm font-medium hover:bg-red-700">Delete</button>
            </div>
          </div>
        </div>
      )}
    </div>
  );
}
