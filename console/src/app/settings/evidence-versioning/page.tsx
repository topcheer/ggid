"use client";

import { useState, useEffect, useCallback } from "react";
import { History, Plus, RotateCcw, X, FileText, GitCompare } from "lucide-react";
import { useTranslations } from "@/lib/i18n";

interface EvidenceVersion {
  version: number;
  content_hash: string;
  collected_by: string;
  collected_at: string;
  change_description: string;
  content_preview: string;
  size_bytes: number;
}

interface EvidenceItem {
  id: string;
  control_id: string;
  framework: string;
  current_version: number;
  versions: EvidenceVersion[];
}

export default function EvidenceVersioningPage() {
  const t = useTranslations();

  const [items, setItems] = useState<EvidenceItem[]>([]);
  const [selectedId, setSelectedId] = useState("");
  const [loading, setLoading] = useState(false);
  const [showCreate, setShowCreate] = useState(false);
  const [rollbackVersion, setRollbackVersion] = useState<{ item: EvidenceItem; version: EvidenceVersion } | null>(null);
  const [diffPair, setDiffPair] = useState<{ a: EvidenceVersion; b: EvidenceVersion } | null>(null);
  const [newDescription, setNewDescription] = useState("");
  const [creating, setCreating] = useState(false);

  const fetchItems = useCallback(async () => {
    try {
      const res = await fetch("/api/v1/audit/evidence-versioning", { headers: { "Authorization": `Bearer ${localStorage.getItem("ggid_access_token") || ""}`, "X-Tenant-ID": "00000000-0000-0000-0000-000000000001" } });
      if (res.ok) {
        const data = await res.json();
        setItems(data.items || data || []);
      }
    } catch { /* noop */ }
  }, []);

  useEffect(() => { fetchItems(); }, [fetchItems]);

  const selectedItem = items.find((i) => i.id === selectedId);

  const createVersion = async () => {
    if (!selectedId || !newDescription) return;
    setCreating(true);
    try {
      await fetch(`/api/v1/audit/evidence-versioning/${selectedId}/versions`, {
        method: "POST",
        headers: { "Authorization": `Bearer ${localStorage.getItem("ggid_access_token") || ""}`, "Content-Type": "application/json", "X-Tenant-ID": "00000000-0000-0000-0000-000000000001" },
        body: JSON.stringify({ change_description: newDescription }),
      });
      setShowCreate(false);
      setNewDescription("");
      fetchItems();
    } catch { /* noop */ }
    finally { setCreating(false); }
  };

  const doRollback = async () => {
    if (!rollbackVersion) return;
    try {
      await fetch(`/api/v1/audit/evidence-versioning/${rollbackVersion.item.id}/rollback`, {
        method: "POST",
        headers: { "Authorization": `Bearer ${localStorage.getItem("ggid_access_token") || ""}`, "Content-Type": "application/json", "X-Tenant-ID": "00000000-0000-0000-0000-000000000001" },
        body: JSON.stringify({ version: rollbackVersion.version.version }),
      });
      setRollbackVersion(null);
      fetchItems();
    } catch { /* noop */ }
  };

  return (
    <div className="space-y-6">
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-2xl font-bold flex items-center gap-2"><History className="w-6 h-6 text-blue-500" /> {t("evidenceVersioning.title")}</h1>
          <p className="text-sm text-gray-500 mt-1">Track evidence content changes with version history and rollback.</p>
        </div>
        {selectedId && <button onClick={() => setShowCreate(true)} className="px-4 py-2 rounded-lg bg-blue-600 text-white text-sm font-medium hover:bg-blue-700 flex items-center gap-2"><Plus className="w-4 h-4" /> New Version</button>}
      </div>

      {/* Evidence selector */}
      <select aria-label="Selected id" value={selectedId} onChange={(e) => setSelectedId(e.target.value)} className="px-3 py-2 rounded-lg border dark:border-gray-700 dark:bg-gray-900 text-sm">
        <option value="">Select evidence...</option>
        {items.map((i) => <option key={i.id} value={i.id}>{i.control_id} ({i.framework}) - v{i.current_version}</option>)}
      </select>

      {selectedItem && (
        <div className="grid grid-cols-1 lg:grid-cols-2 gap-4">
          {/* Version timeline */}
          <div className="rounded-lg border dark:border-gray-800">
            <div className="px-4 py-3 border-b dark:border-gray-800"><h3 className="font-semibold flex items-center gap-2"><History className="w-4 h-4" /> Version History ({selectedItem.versions.length})</h3></div>
            <div className="relative max-h-96 overflow-y-auto">
              {selectedItem.versions.map((v, i) => (
                <div key={v.version} className="relative flex gap-3 px-4 py-3">
                  {i < selectedItem.versions.length - 1 && <div className="absolute left-[27px] top-14 bottom-0 w-0.5 bg-gray-200 dark:bg-gray-800" />}
                  <div className="relative z-10 w-10 h-10 rounded-full flex items-center justify-center flex-shrink-0 bg-blue-50 dark:bg-blue-900/20 text-blue-600 font-bold text-xs">v{v.version}</div>
                  <div className="flex-1 min-w-0">
                    <div className="flex items-center gap-2">
                      <span className="text-sm font-medium">{v.change_description || `Version ${v.version}`}</span>
                      {v.version === selectedItem.current_version && <span className="px-2 py-0.5 rounded text-xs bg-green-100 text-green-800 dark:bg-green-900/30 dark:text-green-400">Current</span>}
                    </div>
                    <p className="text-xs text-gray-400 mt-0.5">{v.collected_by} · {v.collected_at} · {(v.size_bytes / 1024).toFixed(1)}KB</p>
                    <p className="text-xs text-gray-400 font-mono mt-0.5 truncate">{v.content_hash.substring(0, 24)}...</p>
                    <div className="flex items-center gap-2 mt-1">
                      {i > 0 && <button onClick={() => setDiffPair({ a: selectedItem.versions[i], b: selectedItem.versions[0] })} className="text-xs text-blue-600 hover:underline">Diff to current</button>}
                      {v.version !== selectedItem.current_version && <button onClick={() => setRollbackVersion({ item: selectedItem, version: v })} className="text-xs text-orange-600 hover:underline flex items-center gap-1"><RotateCcw className="w-3 h-3" /> Rollback</button>}
                    </div>
                  </div>
                </div>
              ))}
            </div>
          </div>

          {/* Diff panel */}
          <div className="rounded-lg border dark:border-gray-800 p-4">
            <h3 className="font-semibold mb-3 flex items-center gap-2"><GitCompare className="w-4 h-4" /> Diff: v{diffPair?.a.version} → v{diffPair?.b.version}</h3>
            {diffPair ? (
              <div className="space-y-3 text-sm">
                <div className="rounded-lg bg-gray-50 dark:bg-gray-900/50 p-3">
                  <h4 className="text-xs font-medium mb-1">v{diffPair.a.version}</h4>
                  <p className="text-xs font-mono text-gray-500">{diffPair.a.content_hash.substring(0, 32)}...</p>
                  <p className="text-xs mt-1">{diffPair.a.change_description}</p>
                  <pre className="text-xs mt-2 max-h-32 overflow-y-auto whitespace-pre-wrap">{diffPair.a.content_preview}</pre>
                </div>
                <div className="text-center"><RotateCcw className="w-4 h-4 text-gray-400 mx-auto rotate-90" /></div>
                <div className="rounded-lg bg-blue-50 dark:bg-blue-900/20 p-3">
                  <h4 className="text-xs font-medium mb-1">v{diffPair.b.version} (current)</h4>
                  <p className="text-xs font-mono text-blue-500">{diffPair.b.content_hash.substring(0, 32)}...</p>
                  <p className="text-xs mt-1">{diffPair.b.change_description}</p>
                  <pre className="text-xs mt-2 max-h-32 overflow-y-auto whitespace-pre-wrap">{diffPair.b.content_preview}</pre>
                </div>
              </div>
            ) : (
              <p className="text-sm text-gray-500">Click "Diff to current" on a version to compare.</p>
            )}
          </div>
        </div>
      )}

      {!selectedId && <p className="text-sm text-gray-500 text-center py-8">Select an evidence item to view versions.</p>}

      {/* Create version modal */}
      {showCreate && (
        <div className="fixed inset-0 bg-black/50 flex items-center justify-center z-50" onClick={() => setShowCreate(false)}>
          <div role="dialog" aria-modal="true" className="bg-white dark:bg-gray-900 rounded-lg shadow-xl max-w-md w-full mx-4" onClick={(e) => e.stopPropagation()}>
            <div className="flex items-center justify-between px-6 py-4 border-b dark:border-gray-800">
              <h3 className="font-semibold flex items-center gap-2"><FileText className="w-5 h-5 text-blue-500" /> New Version</h3>
              <button onClick={() => setShowCreate(false)} aria-label="Close"><X className="w-5 h-5 text-gray-400" /></button>
            </div>
            <div className="px-6 py-4 space-y-3">
              <div><label className="text-sm font-medium">Change Description</label><textarea aria-label="Updated evidence collection scope..." value={newDescription} onChange={(e) => setNewDescription(e.target.value)} rows={3} placeholder="Updated evidence collection scope..." className="w-full mt-1 px-3 py-2 rounded-lg border dark:border-gray-700 dark:bg-gray-800 text-sm" /></div>
            </div>
            <div className="flex justify-end gap-2 px-6 py-4 border-t dark:border-gray-800">
              <button onClick={() => setShowCreate(false)} className="px-4 py-2 rounded-lg border dark:border-gray-700 text-sm">Cancel</button>
              <button aria-label="action" onClick={createVersion} disabled={!newDescription || creating} className="px-4 py-2 rounded-lg bg-blue-600 text-white text-sm font-medium hover:bg-blue-700 disabled:opacity-50">{creating ? "Creating..." : "Create Version"}</button>
            </div>
          </div>
        </div>
      )}

      {/* Rollback confirmation */}
      {rollbackVersion && (
        <div className="fixed inset-0 bg-black/50 flex items-center justify-center z-50" onClick={() => setRollbackVersion(null)}>
          <div role="dialog" aria-modal="true" className="bg-white dark:bg-gray-900 rounded-lg shadow-xl max-w-md w-full mx-4" onClick={(e) => e.stopPropagation()}>
            <div className="flex items-center justify-between px-6 py-4 border-b dark:border-gray-800">
              <h3 className="font-semibold flex items-center gap-2"><RotateCcw className="w-5 h-5 text-orange-500" /> Confirm Rollback</h3>
              <button onClick={() => setRollbackVersion(null)} aria-label="Close"><X className="w-5 h-5 text-gray-400" /></button>
            </div>
            <div className="px-6 py-4 text-sm space-y-2">
              <p>Rolling back <span className="font-mono font-medium">{rollbackVersion.item.control_id}</span> to v{rollbackVersion.version.version}.</p>
              <p className="text-orange-600">{rollbackVersion.version.change_description}</p>
            </div>
            <div className="flex justify-end gap-2 px-6 py-4 border-t dark:border-gray-800">
              <button onClick={() => setRollbackVersion(null)} className="px-4 py-2 rounded-lg border dark:border-gray-700 text-sm">Cancel</button>
              <button onClick={doRollback} className="px-4 py-2 rounded-lg bg-orange-600 text-white text-sm font-medium hover:bg-orange-700">Rollback</button>
            </div>
          </div>
        </div>
      )}
    </div>
  );
}
