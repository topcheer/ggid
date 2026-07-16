"use client";

import { useState, useEffect, useCallback } from "react";
import { GitBranch, History, RotateCcw, X, AlertTriangle, MonitorSmartphone } from "lucide-react";
import { useTranslations } from "@/lib/i18n";

interface ClientVersion {
  version: number;
  config_hash: string;
  redirect_uris: string[];
  scopes: string[];
  grant_types: string[];
  created_at: string;
  created_by: string;
  change_description: string;
}

interface ClientSummary {
  client_id: string;
  client_name: string;
  current_version: number;
}

export default function ClientVersioningPage() {
  const t = useTranslations();

  const [clients, setClients] = useState<ClientSummary[]>([]);
  const [selectedId, setSelectedId] = useState("");
  const [versions, setVersions] = useState<ClientVersion[]>([]);
  const [loading, setLoading] = useState(false);
  const [diffPair, setDiffPair] = useState<{ a: ClientVersion; b: ClientVersion } | null>(null);
  const [rollbackTarget, setRollbackTarget] = useState<ClientVersion | null>(null);

  const fetchClients = useCallback(async () => {
    try {
      const res = await fetch("/api/v1/oauth/clients", { headers: { "Authorization": `Bearer ${localStorage.getItem("ggid_access_token") || ""}`, "X-Tenant-ID": "00000000-0000-0000-0000-000000000001" } });
      if (res.ok) {
        const data = await res.json();
        setClients(data.clients || data || []);
      }
    } catch { /* noop */ }
  }, []);

  useEffect(() => { fetchClients(); }, [fetchClients]);

  const fetchVersions = useCallback(async () => {
    if (!selectedId) return;
    setLoading(true);
    try {
      const res = await fetch(`/api/v1/oauth/clients/${selectedId}/versions`, { headers: { "Authorization": `Bearer ${localStorage.getItem("ggid_access_token") || ""}`, "X-Tenant-ID": "00000000-0000-0000-0000-000000000001" } });
      if (res.ok) {
        const data = await res.json();
        setVersions(data.versions || data || []);
      }
    } catch { /* noop */ }
    finally { setLoading(false); }
  }, [selectedId]);

  useEffect(() => {
    if (selectedId) fetchVersions();
  }, [selectedId, fetchVersions]);

  const doRollback = async () => {
    if (!rollbackTarget) return;
    try {
      await fetch(`/api/v1/oauth/clients/${selectedId}/rollback`, {
        method: "POST",
        headers: { "Authorization": `Bearer ${localStorage.getItem("ggid_access_token") || ""}`, "Content-Type": "application/json", "X-Tenant-ID": "00000000-0000-0000-0000-000000000001" },
        body: JSON.stringify({ version: rollbackTarget.version }),
      });
      setRollbackTarget(null);
      fetchVersions();
    } catch { /* noop */ }
  };

  const computeDiff = (a: ClientVersion, b: ClientVersion) => {
    const urisAdded = b.redirect_uris.filter((x) => !a.redirect_uris.includes(x));
    const urisRemoved = a.redirect_uris.filter((x) => !b.redirect_uris.includes(x));
    const scopesAdded = b.scopes.filter((x) => !a.scopes.includes(x));
    const scopesRemoved = a.scopes.filter((x) => !b.scopes.includes(x));
    const grantsAdded = b.grant_types.filter((x) => !a.grant_types.includes(x));
    const grantsRemoved = a.grant_types.filter((x) => !b.grant_types.includes(x));
    return { urisAdded, urisRemoved, scopesAdded, scopesRemoved, grantsAdded, grantsRemoved };
  };

  const diff = diffPair ? computeDiff(diffPair.a, diffPair.b) : null;
  const hasChanges = diff ? (diff.urisAdded.length + diff.urisRemoved.length + diff.scopesAdded.length + diff.scopesRemoved.length + diff.grantsAdded.length + diff.grantsRemoved.length) > 0 : false;

  return (
    <div className="space-y-6">
      <div>
        <h1 className="text-2xl font-bold flex items-center gap-2"><GitBranch className="w-6 h-6 text-blue-500" /> {t("clientVersioning.title")}</h1>
        <p className="text-sm text-gray-500 mt-1">View client config version history, diff versions, and rollback.</p>
      </div>

      <select aria-label="Selected id" value={selectedId} onChange={(e) => setSelectedId(e.target.value)} className="px-3 py-2 rounded-lg border dark:border-gray-700 dark:bg-gray-900 text-sm">
        <option value="">Select a client...</option>
        {clients.map((c) => <option key={c.client_id} value={c.client_id}>{c.client_name} (v{c.current_version})</option>)}
      </select>

      {versions.length > 0 && !loading && (
        <div className="grid grid-cols-1 lg:grid-cols-2 gap-4">
          {/* Version timeline */}
          <div className="rounded-lg border dark:border-gray-800">
            <div className="px-4 py-3 border-b dark:border-gray-800"><h3 className="font-semibold flex items-center gap-2"><History className="w-4 h-4" /> Version History ({versions.length})</h3></div>
            <div className="relative max-h-96 overflow-y-auto">
              {versions.map((v, i) => (
                <div key={v.version} className="relative flex gap-3 px-4 py-3">
                  {i < versions.length - 1 && <div className="absolute left-[27px] top-14 bottom-0 w-0.5 bg-gray-200 dark:bg-gray-800" />}
                  <div className="relative z-10 w-10 h-10 rounded-full flex items-center justify-center flex-shrink-0 bg-blue-50 dark:bg-blue-900/20 text-blue-600 font-bold text-xs">v{v.version}</div>
                  <div className="flex-1 min-w-0">
                    <div className="flex items-center gap-2">
                      <span className="text-sm font-medium">{v.change_description || `Version ${v.version}`}</span>
                      {v.version === versions[0]?.version && <span className="px-2 py-0.5 rounded text-xs bg-green-100 text-green-800 dark:bg-green-900/30 dark:text-green-400">Current</span>}
                    </div>
                    <p className="text-xs text-gray-400 mt-0.5">{v.created_by} · {v.created_at}</p>
                    <div className="flex items-center gap-2 mt-1">
                      <button onClick={() => setDiffPair({ a: v, b: versions[0] })} className="text-xs text-blue-600 hover:underline">Diff to current</button>
                      {v.version !== versions[0]?.version && (
                        <button onClick={() => setRollbackTarget(v)} className="text-xs text-orange-600 hover:underline flex items-center gap-1"><RotateCcw className="w-3 h-3" /> Rollback</button>
                      )}
                    </div>
                  </div>
                </div>
              ))}
            </div>
          </div>

          {/* Diff panel */}
          <div className="rounded-lg border dark:border-gray-800 p-4">
            <h3 className="font-semibold mb-3">Diff: v{diffPair?.a.version} → v{diffPair?.b.version}</h3>
            {diff && hasChanges ? (
              <div className="space-y-3">
                {(diff.urisAdded.length > 0 || diff.urisRemoved.length > 0) && (
                  <div><h4 className="text-sm font-medium mb-1">Redirect URIs</h4>{diff.urisAdded.map((v, i) => <div key={`ua${i}`} className="text-xs text-green-600 font-mono">+ {v}</div>)}{diff.urisRemoved.map((v, i) => <div key={`ur${i}`} className="text-xs text-red-600 font-mono line-through">- {v}</div>)}</div>
                )}
                {(diff.scopesAdded.length > 0 || diff.scopesRemoved.length > 0) && (
                  <div><h4 className="text-sm font-medium mb-1">Scopes</h4>{diff.scopesAdded.map((v, i) => <div key={`sa${i}`} className="text-xs text-green-600 font-mono">+ {v}</div>)}{diff.scopesRemoved.map((v, i) => <div key={`sr${i}`} className="text-xs text-red-600 font-mono line-through">- {v}</div>)}</div>
                )}
                {(diff.grantsAdded.length > 0 || diff.grantsRemoved.length > 0) && (
                  <div><h4 className="text-sm font-medium mb-1">Grant Types</h4>{diff.grantsAdded.map((v, i) => <div key={`ga${i}`} className="text-xs text-green-600 font-mono">+ {v}</div>)}{diff.grantsRemoved.map((v, i) => <div key={`gr${i}`} className="text-xs text-red-600 font-mono line-through">- {v}</div>)}</div>
                )}
              </div>
            ) : diff ? (
              <p className="text-sm text-gray-500">No changes between these versions.</p>
            ) : (
              <p className="text-sm text-gray-500">Click "Diff to current" on a version to see changes.</p>
            )}
          </div>
        </div>
      )}

      {!selectedId && <p className="text-sm text-gray-500 text-center py-8">Select a client to view version history.</p>}
      {loading && <p className="text-sm text-gray-500">Loading...</p>}

      {/* Rollback confirmation */}
      {rollbackTarget && (
        <div className="fixed inset-0 bg-black/50 flex items-center justify-center z-50" onClick={() => setRollbackTarget(null)}>
          <div role="dialog" aria-modal="true" className="bg-white dark:bg-gray-900 rounded-lg shadow-xl max-w-md w-full mx-4" onClick={(e) => e.stopPropagation()}>
            <div className="flex items-center justify-between px-6 py-4 border-b dark:border-gray-800">
              <h3 className="font-semibold flex items-center gap-2"><AlertTriangle className="w-5 h-5 text-orange-500" /> Confirm Rollback</h3>
              <button onClick={() => setRollbackTarget(null)} aria-label="Close"><X className="w-5 h-5 text-gray-400" /></button>
            </div>
            <div className="px-6 py-4 text-sm space-y-2">
              <p>Rolling back to <span className="font-bold">v{rollbackTarget.version}</span>.</p>
              <p className="text-gray-500">{rollbackTarget.change_description}</p>
              <p className="text-orange-600">This will replace the current client configuration.</p>
            </div>
            <div className="flex justify-end gap-2 px-6 py-4 border-t dark:border-gray-800">
              <button onClick={() => setRollbackTarget(null)} className="px-4 py-2 rounded-lg border dark:border-gray-700 text-sm">Cancel</button>
              <button onClick={doRollback} className="px-4 py-2 rounded-lg bg-orange-600 text-white text-sm font-medium hover:bg-orange-700 flex items-center gap-1"><RotateCcw className="w-4 h-4" /> Rollback</button>
            </div>
          </div>
        </div>
      )}
    </div>
  );
}
