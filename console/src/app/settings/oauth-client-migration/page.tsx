"use client";
import { useState } from "react";
import { ArrowRight, Play, RotateCcw } from "lucide-react";
import { useTranslations } from "@/lib/i18n";

interface MigrationPreview { scopes_to_migrate: number; grants_to_migrate: number; tokens_to_migrate: number; conflicts: string[]; }

export default function OAuthClientMigrationPage() {
  const t = useTranslations();

  const [source, setSource] = useState("");
  const [target, setTarget] = useState("");
  const [migrateScopes, setMigrateScopes] = useState(true);
  const [migrateGrants, setMigrateGrants] = useState(true);
  const [migrateTokens, setMigrateTokens] = useState(false);
  const [notifyUsers, setNotifyUsers] = useState(true);
  const [preview, setPreview] = useState<MigrationPreview | null>(null);
  const [executing, setExecuting] = useState(false);
  const [previewing, setPreviewing] = useState(false);
  const [error, setError] = useState("");
  const [success, setSuccess] = useState("");

  const retryPreview = () => { setError(""); doPreview(); };

  const doPreview = async () => {
    setPreviewing(true); setError(""); setPreview(null);
    try {
      const res = await fetch("/api/v1/oauth/client-migration/preview", { method: "POST", headers: { "Authorization": `Bearer ${localStorage.getItem("ggid_access_token") || ""}`, "Content-Type": "application/json", "X-Tenant-ID": "00000000-0000-0000-0000-000000000001" }, body: JSON.stringify({ source, target }) });
      if (!res.ok) return null;
      setPreview(await res.json());
    } catch (e) {
      setError(e instanceof Error ? e.message : "Failed to preview migration");
    } finally { setPreviewing(false); }
  };

  const execute = async () => {
    setExecuting(true); setError(""); setSuccess("");
    try {
      const res = await fetch("/api/v1/oauth/client-migration/execute", { method: "POST", headers: { "Authorization": `Bearer ${localStorage.getItem("ggid_access_token") || ""}`, "Content-Type": "application/json", "X-Tenant-ID": "00000000-0000-0000-0000-000000000001" }, body: JSON.stringify({ source, target, migrate_scopes: migrateScopes, migrate_grants: migrateGrants, migrate_tokens: migrateTokens, notify_users: notifyUsers }) });
      if (!res.ok) return null;
      setSuccess("Migration completed successfully."); setPreview(null);
    } catch (e) {
      setError(e instanceof Error ? e.message : "Failed to execute migration");
    } finally { setExecuting(false); }
  };

  return (
    <div className="space-y-6">
      <div><h1 className="text-2xl font-bold flex items-center gap-2"><ArrowRight className="w-6 h-6 text-blue-500" /> {t("oauthClientMigration.title")}</h1><p className="text-sm text-gray-500 mt-1">Migrate OAuth clients with scope, grant, and token transfer.</p></div>

      <div className="rounded-lg border dark:border-gray-800 p-4 space-y-3">
        <div className="grid grid-cols-2 gap-3">
          <div><label className="text-sm font-medium">Source Client</label><input type="text" value={source} onChange={(e) => setSource(e.target.value)} placeholder="client-old-123" aria-label="Source client ID" className="w-full mt-1 px-3 py-2 rounded-lg border dark:border-gray-700 dark:bg-gray-900 text-sm font-mono" /></div>
          <div><label className="text-sm font-medium">Target Client</label><input type="text" value={target} onChange={(e) => setTarget(e.target.value)} placeholder="client-new-456" aria-label="Target client ID" className="w-full mt-1 px-3 py-2 rounded-lg border dark:border-gray-700 dark:bg-gray-900 text-sm font-mono" /></div>
        </div>
        <div className="flex flex-wrap gap-4">
          <label className="flex items-center gap-2 text-sm"><input aria-label="Migrate scopes" type="checkbox" checked={migrateScopes} onChange={(e) => setMigrateScopes(e.target.checked)} className="rounded" /> Migrate Scopes</label>
          <label className="flex items-center gap-2 text-sm"><input aria-label="Migrate grants" type="checkbox" checked={migrateGrants} onChange={(e) => setMigrateGrants(e.target.checked)} className="rounded" /> Migrate Grants</label>
          <label className="flex items-center gap-2 text-sm"><input aria-label="Migrate tokens" type="checkbox" checked={migrateTokens} onChange={(e) => setMigrateTokens(e.target.checked)} className="rounded" /> Migrate Tokens</label>
          <label className="flex items-center gap-2 text-sm"><input aria-label="Notify users" type="checkbox" checked={notifyUsers} onChange={(e) => setNotifyUsers(e.target.checked)} className="rounded" /> Notify Users</label>
        </div>
        <div className="flex gap-2"><button onClick={doPreview} disabled={!source || !target || previewing} className="px-4 py-2 rounded-lg border dark:border-gray-700 text-sm disabled:opacity-50" aria-label="Preview migration">{previewing ? "Previewing..." : "Preview"}</button><button onClick={execute} disabled={executing || !source || !target} className="px-4 py-2 rounded-lg bg-blue-600 text-white text-sm font-medium disabled:opacity-50 flex items-center gap-2" aria-label="Execute migration"><Play className="w-4 h-4" /> {executing ? "Migrating..." : "Execute"}</button></div>
        {error && <div className="rounded-lg border border-red-200 dark:border-red-900 bg-red-50 dark:bg-red-900/20 p-3 text-sm text-red-600 flex items-center justify-between"><span>{error}</span><button onClick={retryPreview} className="text-xs underline hover:text-red-700">Retry</button></div>}
        {success && <div className="rounded-lg border border-green-200 dark:border-green-900 bg-green-50 dark:bg-green-900/20 p-3 text-sm text-green-700 dark:text-green-400">{success}</div>}
      </div>

      {previewing && <div className="rounded-lg border dark:border-gray-800 p-8 text-center"><div className="inline-block w-5 h-5 border-2 border-current border-t-transparent rounded-full animate-spin text-blue-600 mb-2" /><div className="text-sm text-gray-500">Loading preview...</div></div>}
      {preview && (<div className="rounded-lg border dark:border-gray-800 p-4 space-y-2"><h3 className="text-sm font-semibold">Migration Preview</h3><div className="grid grid-cols-3 gap-4 text-sm"><div><span className="text-gray-500">Scopes:</span> <span className="font-bold">{preview.scopes_to_migrate}</span></div><div><span className="text-gray-500">Grants:</span> <span className="font-bold">{preview.grants_to_migrate}</span></div><div><span className="text-gray-500">Tokens:</span> <span className="font-bold">{preview.tokens_to_migrate}</span></div></div>{preview.conflicts.length > 0 && (<div className="mt-2"><span className="text-xs font-semibold text-red-600">Conflicts:</span><div className="mt-1 space-y-1">{preview.conflicts.map((c, i) => (<div key={i} className="text-xs text-red-500">{c}</div>))}</div></div>)}</div>)}
    </div>
  );
}
