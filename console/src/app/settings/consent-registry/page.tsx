"use client";

import { useState, useCallback } from "react";
import { Search, ShieldCheck, History, Save } from "lucide-react";
import { useTranslations } from "@/lib/i18n";

interface ConsentEntry {
  key: string;
  label: string;
  granted: boolean;
  granted_at: string | null;
  version: number;
}

interface ConsentData {
  user_id: string;
  username: string;
  consents: ConsentEntry[];
  history: { version: number; changed_at: string; changed_by: string; changes: string }[];
}

export default function ConsentRegistryPage() {
  const t = useTranslations();

  const [search, setSearch] = useState("");
  const [data, setData] = useState<ConsentData | null>(null);
  const [loading, setLoading] = useState(false);
  const [consents, setConsents] = useState<ConsentEntry[]>([]);
  const [saving, setSaving] = useState(false);

  const fetchData = useCallback(async (user: string) => {
    if (!user) return;
    setLoading(true);
    try {
      const res = await fetch(`/api/v1/identity/consent?user=${encodeURIComponent(user)}`, { headers: { "X-Tenant-ID": "00000000-0000-0000-0000-000000000001" } });
      if (res.ok) { const json = await res.json(); setData(json); setConsents(json.consents || []); }
    } catch { /* noop */ }
    finally { setLoading(false); }
  }, []);

  const toggle = (key: string) => {
    setConsents((prev) => prev.map((c) => c.key === key ? { ...c, granted: !c.granted, granted_at: !c.granted ? new Date().toISOString() : null } : c));
  };

  const save = async () => {
    if (!data) return;
    setSaving(true);
    try { await fetch(`/api/v1/identity/consent/${data.user_id}`, { method: "PUT", headers: { "Content-Type": "application/json", "X-Tenant-ID": "00000000-0000-0000-0000-000000000001" }, body: JSON.stringify({ consents }) }); } catch { /* noop */ } finally { setSaving(false); }
  };

  return (
    <div className="space-y-6">
      <div className="flex items-center justify-between">
        <div><h1 className="text-2xl font-bold flex items-center gap-2"><ShieldCheck className="w-6 h-6 text-blue-500" /> {t("consentRegistry.title")}</h1><p className="text-sm text-gray-500 mt-1">Manage user consent preferences with version history.</p></div>
        {data && <button aria-label="Save" onClick={save} disabled={saving} className="px-4 py-2 rounded-lg bg-blue-600 text-white text-sm font-medium hover:bg-blue-700 disabled:opacity-50 flex items-center gap-2"><Save className="w-4 h-4" /> {saving ? "Saving..." : "Save"}</button>}
      </div>

      <div className="relative max-w-md">
        <Search className="absolute left-3 top-1/2 -translate-y-1/2 w-4 h-4 text-gray-400" />
        <input aria-label="Search by username..." type="text" placeholder="Search by username..." value={search} onChange={(e) => setSearch(e.target.value)} className="w-full pl-9 pr-3 py-2 rounded-lg border dark:border-gray-700 dark:bg-gray-900 text-sm" />
      </div>

      {data && (
        <div className="grid grid-cols-1 lg:grid-cols-2 gap-4">
          {/* Consent toggles */}
          <div className="rounded-lg border dark:border-gray-800 p-4">
            <h3 className="font-semibold mb-3">{data.username} - Consent Preferences</h3>
            <div className="space-y-3">
              {consents.map((c) => (
                <div key={c.key} className="flex items-center justify-between p-3 rounded-lg border dark:border-gray-700">
                  <div><span className="font-medium text-sm">{c.label}</span><p className="text-xs text-gray-400 mt-0.5">{c.granted ? `Granted: ${c.granted_at}` : "Not granted"} · v{c.version}</p></div>
                  <button onClick={() => toggle(c.key)} className={`relative w-12 h-6 rounded-full transition-colors ${c.granted ? "bg-green-500" : "bg-gray-300 dark:bg-gray-700"}`}><span className={`absolute top-0.5 w-5 h-5 rounded-full bg-white transition-transform ${c.granted ? "translate-x-6" : "translate-x-0.5"}`} /></button>
                </div>
              ))}
            </div>
          </div>

          {/* Version history */}
          <div className="rounded-lg border dark:border-gray-800">
            <div className="px-4 py-3 border-b dark:border-gray-800"><h3 className="font-semibold flex items-center gap-2"><History className="w-4 h-4" /> Version History</h3></div>
            <div className="divide-y dark:divide-gray-800 max-h-64 overflow-y-auto">
              {data.history.map((h, i) => (
                <div key={i} className="px-4 py-2 text-sm"><div className="flex items-center justify-between"><span className="font-medium">v{h.version}</span><span className="text-xs text-gray-400">{h.changed_at}</span></div><p className="text-xs text-gray-500 mt-0.5">{h.changes}</p><p className="text-xs text-gray-400">By: {h.changed_by}</p></div>
              ))}
              {data.history.length === 0 && <p className="px-4 py-4 text-sm text-gray-500">No changes recorded.</p>}
            </div>
          </div>
        </div>
      )}

      {!data && !loading && search && <p className="text-sm text-gray-500">No consent data found.</p>}
      {!data && !search && <p className="text-sm text-gray-500 text-center py-8">Search for a user to manage consent.</p>}
    </div>
  );
}
