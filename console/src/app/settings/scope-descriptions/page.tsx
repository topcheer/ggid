"use client";
import { useTranslations } from "@/lib/i18n";

import { useState, useEffect, useCallback } from "react";
import { Languages, Save, Search, Globe } from "lucide-react";

interface ScopeDescription {
  scope: string;
  descriptions: Record<string, string>;
}

const languages = [
  { code: "en", label: "English", flag: "EN" },
  { code: "zh", label: "中文", flag: "ZH" },
  { code: "ja", label: "日本語", flag: "JA" },
  { code: "de", label: "Deutsch", flag: "DE" },
  { code: "fr", label: "Français", flag: "FR" },
];

export default function ScopeDescriptionsPage() {
  const t = useTranslations();
  const [scopes, setScopes] = useState<ScopeDescription[]>([]);
  const [loading, setLoading] = useState(false);
  const [search, setSearch] = useState("");
  const [editScope, setEditScope] = useState<string | null>(null);
  const [editValues, setEditValues] = useState<Record<string, string>>({});
  const [saving, setSaving] = useState(false);

  const fetchScopes = useCallback(async () => {
    setLoading(true);
    try {
      const res = await fetch("/api/v1/oauth/scope-descriptions", { headers: { "X-Tenant-ID": "00000000-0000-0000-0000-000000000001" } });
      if (res.ok) {
        const data = await res.json();
        setScopes(data.scopes || data || []);
      }
    } catch { /* noop */ }
    finally { setLoading(false); }
  }, []);

  useEffect(() => { fetchScopes(); }, [fetchScopes]);

  const startEdit = (scope: ScopeDescription) => {
    setEditScope(scope.scope);
    const vals: Record<string, string> = {};
    languages.forEach((l) => { vals[l.code] = scope.descriptions[l.code] || ""; });
    setEditValues(vals);
  };

  const saveEdit = async () => {
    if (!editScope) return;
    setSaving(true);
    try {
      await fetch(`/api/v1/oauth/scope-descriptions/${encodeURIComponent(editScope)}`, {
        method: "PUT",
        headers: { "Content-Type": "application/json", "X-Tenant-ID": "00000000-0000-0000-0000-000000000001" },
        body: JSON.stringify({ descriptions: editValues }),
      });
      setScopes((prev) => prev.map((s) => s.scope === editScope ? { ...s, descriptions: { ...editValues } } : s));
      setEditScope(null);
    } catch { /* noop */ }
    finally { setSaving(false); }
  };

  const filtered = scopes.filter((s) => !search || s.scope.toLowerCase().includes(search.toLowerCase()));
  const completionForLang = (code: string) => scopes.filter((s) => s.descriptions[code]).length;

  return (
    <div className="space-y-6">
      <div>
        <h1 className="text-2xl font-bold flex items-center gap-2"><Languages className="w-6 h-6 text-blue-500" />{t("scopeDescriptions.title")}</h1>
        <p className="text-sm text-gray-500 mt-1">Edit OAuth scope descriptions across multiple languages.</p>
      </div>

      {/* Language completion stats */}
      <div className="grid grid-cols-5 gap-3">
        {languages.map((l) => (
          <div key={l.code} className="rounded-lg border p-3 dark:border-gray-800 text-center">
            <span className="text-xs font-bold" style={{ color: completionForLang(l.code) === scopes.length && scopes.length > 0 ? "#10b981" : "#f59e0b" }}>{l.flag}</span>
            <p className="text-lg font-bold mt-1">{completionForLang(l.code)}/{scopes.length}</p>
            <span className="text-xs text-gray-400">{l.label}</span>
          </div>
        ))}
      </div>

      {/* Search */}
      <div className="relative max-w-md">
        <Search className="absolute left-3 top-1/2 -translate-y-1/2 w-4 h-4 text-gray-400" />
        <input aria-label="Search scopes..." type="text" placeholder="Search scopes..." value={search} onChange={(e) => setSearch(e.target.value)} className="w-full pl-9 pr-3 py-2 rounded-lg border dark:border-gray-700 dark:bg-gray-900 text-sm" />
      </div>

      {/* Scope list */}
      <div className="space-y-2">
        {filtered.map((s) => (
          <div key={s.scope} className="rounded-lg border dark:border-gray-800 overflow-hidden">
            {editScope === s.scope ? (
              <div className="p-4 space-y-3">
                <div className="flex items-center justify-between">
                  <h3 className="font-semibold font-mono text-sm">{s.scope}</h3>
                  <div className="flex items-center gap-2">
                    <button onClick={() => setEditScope(null)} className="px-3 py-1.5 rounded-lg border dark:border-gray-700 text-sm">Cancel</button>
                    <button onClick={saveEdit} disabled={saving} className="px-3 py-1.5 rounded-lg bg-blue-600 text-white text-sm font-medium hover:bg-blue-700 disabled:opacity-50 flex items-center gap-1"><Save className="w-4 h-4" /> {saving ? "Saving..." : "Save"}</button>
                  </div>
                </div>
                <div className="grid grid-cols-1 md:grid-cols-2 gap-3">
                  {languages.map((l) => (
                    <div key={l.code}>
                      <label className="text-xs font-medium flex items-center gap-1"><Globe className="w-3 h-3" /> {l.flag} - {l.label}</label>
                      <textarea value={editValues[l.code] || ""} onChange={(e) => setEditValues({ ...editValues, [l.code]: e.target.value })} rows={2} placeholder={`${s.scope} description in ${l.label}`} className="w-full mt-1 px-3 py-2 rounded-lg border dark:border-gray-700 dark:bg-gray-800 text-sm" />
                    </div>
                  ))}
                </div>
              </div>
            ) : (
              <div className="p-4">
                <div className="flex items-center justify-between">
                  <span className="font-mono text-sm font-medium">{s.scope}</span>
                  <button onClick={() => startEdit(s)} className="text-blue-600 hover:underline text-sm font-medium">Edit</button>
                </div>
                <div className="grid grid-cols-1 md:grid-cols-5 gap-2 mt-3">
                  {languages.map((l) => (
                    <div key={l.code} className="text-xs">
                      <span className="text-gray-400 font-bold">{l.flag}</span>
                      <p className={`mt-0.5 ${s.descriptions[l.code] ? "text-gray-600 dark:text-gray-400" : "text-gray-300 dark:text-gray-700 italic"}`}>{s.descriptions[l.code] || "Not set"}</p>
                    </div>
                  ))}
                </div>
              </div>
            )}
          </div>
        ))}
        {filtered.length === 0 && !loading && <p className="text-sm text-gray-500 text-center py-8">No scopes found.</p>}
      </div>
    </div>
  );
}
