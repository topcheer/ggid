"use client";
import { useState } from "react";
import { ArrowRight, Play, RotateCcw } from "lucide-react";

interface MigrationPreview { scopes_to_migrate: number; grants_to_migrate: number; tokens_to_migrate: number; conflicts: string[]; }

export default function OAuthClientMigrationPage() {
  const [source, setSource] = useState("");
  const [target, setTarget] = useState("");
  const [migrateScopes, setMigrateScopes] = useState(true);
  const [migrateGrants, setMigrateGrants] = useState(true);
  const [migrateTokens, setMigrateTokens] = useState(false);
  const [notifyUsers, setNotifyUsers] = useState(true);
  const [preview, setPreview] = useState<MigrationPreview | null>(null);
  const [executing, setExecuting] = useState(false);

  const doPreview = async () => {
    try { const res = await fetch("/api/v1/oauth/client-migration/preview", { method: "POST", headers: { "Content-Type": "application/json", "X-Tenant-ID": "00000000-0000-0000-0000-000000000001" }, body: JSON.stringify({ source, target }) }); if (res.ok) setPreview(await res.json()); }
    catch { /* noop */ }
  };

  const execute = async () => {
    setExecuting(true);
    try { await fetch("/api/v1/oauth/client-migration/execute", { method: "POST", headers: { "Content-Type": "application/json", "X-Tenant-ID": "00000000-0000-0000-0000-000000000001" }, body: JSON.stringify({ source, target, migrate_scopes: migrateScopes, migrate_grants: migrateGrants, migrate_tokens: migrateTokens, notify_users: notifyUsers }) }); }
    catch { /* noop */ }
    finally { setExecuting(false); }
  };

  return (
    <div className="space-y-6">
      <div><h1 className="text-2xl font-bold flex items-center gap-2"><ArrowRight className="w-6 h-6 text-blue-500" /> Client Migration</h1><p className="text-sm text-gray-500 mt-1">Migrate OAuth clients with scope, grant, and token transfer.</p></div>

      <div className="rounded-lg border dark:border-gray-800 p-4 space-y-3">
        <div className="grid grid-cols-2 gap-3">
          <div><label className="text-sm font-medium">Source Client</label><input type="text" value={source} onChange={(e) => setSource(e.target.value)} placeholder="client-old-123" className="w-full mt-1 px-3 py-2 rounded-lg border dark:border-gray-700 dark:bg-gray-900 text-sm font-mono" /></div>
          <div><label className="text-sm font-medium">Target Client</label><input type="text" value={target} onChange={(e) => setTarget(e.target.value)} placeholder="client-new-456" className="w-full mt-1 px-3 py-2 rounded-lg border dark:border-gray-700 dark:bg-gray-900 text-sm font-mono" /></div>
        </div>
        <div className="flex flex-wrap gap-4">
          <label className="flex items-center gap-2 text-sm"><input type="checkbox" checked={migrateScopes} onChange={(e) => setMigrateScopes(e.target.checked)} className="rounded" /> Migrate Scopes</label>
          <label className="flex items-center gap-2 text-sm"><input type="checkbox" checked={migrateGrants} onChange={(e) => setMigrateGrants(e.target.checked)} className="rounded" /> Migrate Grants</label>
          <label className="flex items-center gap-2 text-sm"><input type="checkbox" checked={migrateTokens} onChange={(e) => setMigrateTokens(e.target.checked)} className="rounded" /> Migrate Tokens</label>
          <label className="flex items-center gap-2 text-sm"><input type="checkbox" checked={notifyUsers} onChange={(e) => setNotifyUsers(e.target.checked)} className="rounded" /> Notify Users</label>
        </div>
        <div className="flex gap-2"><button onClick={doPreview} disabled={!source || !target} className="px-4 py-2 rounded-lg border dark:border-gray-700 text-sm">Preview</button><button onClick={execute} disabled={executing || !source || !target} className="px-4 py-2 rounded-lg bg-blue-600 text-white text-sm font-medium disabled:opacity-50 flex items-center gap-2"><Play className="w-4 h-4" /> {executing ? "Migrating..." : "Execute"}</button></div>
      </div>

      {preview && (<div className="rounded-lg border dark:border-gray-800 p-4 space-y-2"><h3 className="text-sm font-semibold">Migration Preview</h3><div className="grid grid-cols-3 gap-4 text-sm"><div><span className="text-gray-500">Scopes:</span> <span className="font-bold">{preview.scopes_to_migrate}</span></div><div><span className="text-gray-500">Grants:</span> <span className="font-bold">{preview.grants_to_migrate}</span></div><div><span className="text-gray-500">Tokens:</span> <span className="font-bold">{preview.tokens_to_migrate}</span></div></div>{preview.conflicts.length > 0 && (<div className="mt-2"><span className="text-xs font-semibold text-red-600">Conflicts:</span><div className="mt-1 space-y-1">{preview.conflicts.map((c, i) => (<div key={i} className="text-xs text-red-500">{c}</div>))}</div></div>)}</div>)}
    </div>
  );
}
