"use client";

import { useState, useEffect, useCallback } from "react";
import { ArrowRightLeft, Save, Play, Plus, Trash2, GitCompare, AlertTriangle, X } from "lucide-react";
import { useTranslations } from "@/lib/i18n";
import { authHeader, isAuthenticated } from "@/lib/auth-helpers";

interface ClientConfig {
  client_id: string;
  client_name: string;
  redirect_uris: string[];
  scopes: string[];
  grant_types: string[];
}

interface DiffResult {
  redirect_uris: { added: string[]; removed: string[] };
  scopes: { added: string[]; removed: string[] };
  grant_types: { added: string[]; removed: string[] };
}

export default function ClientMigrationPage() {
  const t = useTranslations();

  const [clients, setClients] = useState<ClientConfig[]>([]);
  const [selectedId, setSelectedId] = useState("");
  const [original, setOriginal] = useState<ClientConfig | null>(null);
  const [draft, setDraft] = useState<ClientConfig | null>(null);
  const [diff, setDiff] = useState<DiffResult | null>(null);
  const [gracePeriod, setGracePeriod] = useState(7);
  const [loading, setLoading] = useState(false);
  const [executing, setExecuting] = useState(false);
  const [showConfirm, setShowConfirm] = useState(false);
  const [newItem, setNewItem] = useState({ redirect_uris: "", scopes: "", grant_types: "" });

  const fetchClients = useCallback(async () => {
    try {
      const res = await fetch("/api/v1/oauth/clients", { headers: { ...authHeader(), "X-Tenant-ID": "00000000-0000-0000-0000-000000000001" } });
      if (res.ok) {
        const data = await res.json();
        setClients(data.clients || data || []);
      }
    } catch { /* noop */ }
  }, []);

  useEffect(() => { fetchClients(); }, [fetchClients]);

  const selectClient = (id: string) => {
    setSelectedId(id);
    const c = clients.find((cl: any) => cl.client_id === id);
    if (c) {
      setOriginal({ ...c, redirect_uris: [...c.redirect_uris], scopes: [...c.scopes], grant_types: [...c.grant_types] });
      setDraft({ ...c, redirect_uris: [...c.redirect_uris], scopes: [...c.scopes], grant_types: [...c.grant_types] });
      setDiff(null);
    }
  };

  const computeDiff = async () => {
    if (!original || !draft) return;
    try {
      const res = await fetch("/api/v1/oauth/client-migration/preview", {
        method: "POST",
        headers: { ...authHeader(), "Content-Type": "application/json", "X-Tenant-ID": "00000000-0000-0000-0000-000000000001" },
        body: JSON.stringify({ client_id: original.client_id, original, proposed: draft }),
      });
      if (res.ok) {
        setDiff(await res.json());
      } else {
        // local diff fallback
        setDiff({
          redirect_uris: {
            added: draft.redirect_uris.filter((x: any) => !original.redirect_uris.includes(x)),
            removed: original.redirect_uris.filter((x: any) => !draft.redirect_uris.includes(x)),
          },
          scopes: {
            added: draft.scopes.filter((x: any) => !original.scopes.includes(x)),
            removed: original.scopes.filter((x: any) => !draft.scopes.includes(x)),
          },
          grant_types: {
            added: draft.grant_types.filter((x: any) => !original.grant_types.includes(x)),
            removed: original.grant_types.filter((x: any) => !draft.grant_types.includes(x)),
          },
        });
      }
    } catch {
      setDiff({
        redirect_uris: {
          added: draft.redirect_uris.filter((x: any) => !original.redirect_uris.includes(x)),
          removed: original.redirect_uris.filter((x: any) => !draft.redirect_uris.includes(x)),
        },
        scopes: {
          added: draft.scopes.filter((x: any) => !original.scopes.includes(x)),
          removed: original.scopes.filter((x: any) => !draft.scopes.includes(x)),
        },
        grant_types: {
          added: draft.grant_types.filter((x: any) => !original.grant_types.includes(x)),
          removed: original.grant_types.filter((x: any) => !draft.grant_types.includes(x)),
        },
      });
    }
  };

  const execute = async () => {
    if (!draft) return;
    setExecuting(true);
    try {
      await fetch("/api/v1/oauth/client-migration/execute", {
        method: "POST",
        headers: { ...authHeader(), "Content-Type": "application/json", "X-Tenant-ID": "00000000-0000-0000-0000-000000000001" },
        body: JSON.stringify({ client_id: draft.client_id, config: draft, grace_period_days: gracePeriod }),
      });
      setShowConfirm(false);
      setOriginal({ ...draft, redirect_uris: [...draft.redirect_uris], scopes: [...draft.scopes], grant_types: [...draft.grant_types] });
      setDiff(null);
    } catch { /* noop */ }
    finally { setExecuting(false); }
  };

  const addField = (field: "redirect_uris" | "scopes" | "grant_types", value: string) => {
    if (!draft || !value.trim()) return;
    setDraft({ ...draft, [field]: [...draft[field], value.trim()] });
  };
  const removeField = (field: "redirect_uris" | "scopes" | "grant_types", idx: number) => {
    if (!draft) return;
    setDraft({ ...draft, [field]: draft[field].filter((_, i) => i !== idx) });
  };

  const hasChanges = original && draft ? JSON.stringify(original) !== JSON.stringify(draft) : false;

  const DiffSection = ({ title, field }: { title: string; field: keyof DiffResult }) => {
    if (!diff) return null;
    const d = diff[field];
    if (d.added.length === 0 && d.removed.length === 0) return null;
    return (
      <div>
        <h4 className="text-sm font-medium mb-1">{title}</h4>
        <div className="space-y-1">
          {d.added.map((v: any, i: number) => (<div key={`a${i}`} className="flex items-center gap-2 text-xs"><span className="text-green-600 font-mono">+ {v}</span></div>))}
          {d.removed.map((v: any, i: number) => (<div key={`r${i}`} className="flex items-center gap-2 text-xs"><span className="text-red-600 line-through font-mono">- {v}</span></div>))}
        </div>
      </div>
    );
  };

  return (
    <div className="space-y-6">
      <div>
        <h1 className="text-2xl font-bold flex items-center gap-2"><ArrowRightLeft className="w-6 h-6 text-blue-500" /> {t("clientMigration.title")}</h1>
        <p className="text-sm text-gray-500 mt-1">Migrate OAuth client config with diff preview and grace period.</p>
      </div>

      {/* Client selector */}
      <select aria-label="Selected id" value={selectedId} onChange={(e) => selectClient(e.target.value)} className="px-3 py-2 rounded-lg border dark:border-gray-700 dark:bg-gray-900 text-sm">
        <option value="">Select a client...</option>
        {clients.map((c: any) => (<option key={c.client_id} value={c.client_id}>{c.client_name} ({c.client_id})</option>))}
      </select>

      {draft && (
        <div className="grid grid-cols-1 lg:grid-cols-2 gap-4">
          {/* Edit form */}
          <div className="space-y-4">
            {(["redirect_uris", "scopes", "grant_types"] as const).map((field: any) => (
              <div key={field} className="rounded-lg border dark:border-gray-800 p-4">
                <h3 className="font-semibold text-sm mb-2 capitalize">{field.replace("_", " ")}</h3>
                <div className="space-y-1">
                  {draft[field].map((val: any, i: number) => (
                    <div key={i} className="flex items-center gap-2">
                      <span className="flex-1 px-2 py-1 rounded text-xs font-mono bg-gray-100 dark:bg-gray-800 truncate">{val}</span>
                      <button onClick={() => removeField(field, i)} className="p-1 text-red-400 hover:text-red-600"><Trash2 className="w-3 h-3" /></button>
                    </div>
                  ))}
                  <div className="flex items-center gap-2">
                    <input aria-label="Input field" type="text" value={newItem[field]} onChange={(e) => setNewItem({ ...newItem, [field]: e.target.value })} placeholder={`Add ${field.replace("_", " ")}...`} className="flex-1 px-2 py-1 rounded border dark:border-gray-700 dark:bg-gray-800 text-xs font-mono"
                      onKeyDown={(e) => { if (e.key === "Enter") { addField(field, newItem[field]); setNewItem({ ...newItem, [field]: "" }); } }}
                    />
                    <button onClick={() => { addField(field, newItem[field]); setNewItem({ ...newItem, [field]: "" }); }} className="p-1 rounded bg-blue-600 text-white"><Plus className="w-3 h-3" /></button>
                  </div>
                </div>
              </div>
            ))}
            <div className="flex items-center gap-2">
              <label className="text-sm font-medium">Grace Period (days)</label>
              <input aria-label="grace Period" type="number" value={gracePeriod} onChange={(e) => setGracePeriod(parseInt(e.target.value) || 0)} min={0} className="w-20 px-2 py-1 rounded border dark:border-gray-700 dark:bg-gray-800 text-sm" />
            </div>
          </div>

          {/* Diff preview */}
          <div className="space-y-4">
            <div className="flex items-center gap-2">
              <button onClick={computeDiff} disabled={!hasChanges} className="px-3 py-2 rounded-lg bg-blue-600 text-white text-sm font-medium hover:bg-blue-700 disabled:opacity-50 flex items-center gap-2"><GitCompare className="w-4 h-4" /> Preview Diff</button>
              <button onClick={() => setShowConfirm(true)} disabled={!diff} className="px-3 py-2 rounded-lg bg-green-600 text-white text-sm font-medium hover:bg-green-700 disabled:opacity-50 flex items-center gap-2"><Play className="w-4 h-4" /> Execute Migration</button>
            </div>
            {diff ? (
              <div className="rounded-lg border dark:border-gray-800 p-4 space-y-3">
                <DiffSection title="Redirect URIs" field="redirect_uris" />
                <DiffSection title="Scopes" field="scopes" />
                <DiffSection title="Grant Types" field="grant_types" />
                {diff.redirect_uris.added.length === 0 && diff.redirect_uris.removed.length === 0 && diff.scopes.added.length === 0 && diff.scopes.removed.length === 0 && diff.grant_types.added.length === 0 && diff.grant_types.removed.length === 0 && (
                  <p className="text-sm text-gray-500">No changes detected.</p>
                )}
              </div>
            ) : (
              <p className="text-sm text-gray-500 text-center py-8">Click "Preview Diff" to see changes.</p>
            )}
          </div>
        </div>
      )}

      {!draft && <p className="text-sm text-gray-500 text-center py-8">Select a client to begin migration.</p>}

      {/* Execute confirmation */}
      {showConfirm && draft && (
        <div className="fixed inset-0 bg-black/50 flex items-center justify-center z-50" onClick={() => setShowConfirm(false)}>
          <div role="dialog" aria-modal="true" className="bg-white dark:bg-gray-900 rounded-lg shadow-xl max-w-md w-full mx-4" onClick={(e) => e.stopPropagation()}>
            <div className="flex items-center justify-between px-6 py-4 border-b dark:border-gray-800">
              <h3 className="font-semibold flex items-center gap-2"><AlertTriangle className="w-5 h-5 text-orange-500" /> Confirm Migration</h3>
              <button onClick={() => setShowConfirm(false)} aria-label="Close"><X className="w-5 h-5 text-gray-400" /></button>
            </div>
            <div className="px-6 py-4 space-y-2 text-sm">
              <p>Migrating <span className="font-medium">{draft.client_name}</span> with a <span className="font-medium">{gracePeriod}-day</span> grace period.</p>
              <p className="text-gray-500">Old tokens remain valid during grace period. New configuration takes effect immediately for new token requests.</p>
            </div>
            <div className="flex justify-end gap-2 px-6 py-4 border-t dark:border-gray-800">
              <button onClick={() => setShowConfirm(false)} className="px-4 py-2 rounded-lg border dark:border-gray-700 text-sm">Cancel</button>
              <button aria-label="Play" onClick={execute} disabled={executing} className="px-4 py-2 rounded-lg bg-green-600 text-white text-sm font-medium hover:bg-green-700 disabled:opacity-50 flex items-center gap-1"><Play className="w-4 h-4" /> {executing ? "Executing..." : "Execute"}</button>
            </div>
          </div>
        </div>
      )}
    </div>
  );
}
