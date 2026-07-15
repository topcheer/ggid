"use client";

import { useState } from "react";
import { useApi } from "@/lib/api";
import {
  Clock, Loader2, AlertCircle, X, Plus, Trash2, Save, Calendar,
} from "lucide-react";
import { useTranslations } from "@/lib/i18n";

interface ScheduleRule {
  id: string;
  name: string;
  cron: string;
  start_time: string;
  end_time: string;
  timezone: string;
  allowed_roles: string[];
  effect: "allow" | "deny";
  enabled: boolean;
  description: string;
}

const weekdays = ["SUN", "MON", "TUE", "WED", "THU", "FRI", "SAT"];
const hours = Array.from({ length: 24 }, (_, i) => `${String(i).padStart(2, "0")}:00`);

export default function TimeBasedAccessPage() {
  const t = useTranslations();

  const { apiFetch } = useApi();
  const [rules, setRules] = useState<ScheduleRule[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [editing, setEditing] = useState<ScheduleRule | null>(null);
  const [roleInput, setRoleInput] = useState("");

  useState(() => {
    (async () => {
      try { setRules(await apiFetch<ScheduleRule[]>("/api/v1/policy/time-based/rules").catch(() => [])); }
      catch { setError("Failed to load time-based access rules"); }
      finally { setLoading(false); }
    })();
  });

  const handleSave = async () => {
    if (!editing) return;
    try {
      if (editing.id) {
        await apiFetch(`/api/v1/policy/time-based/rules/${editing.id}`, { method: "PUT", body: JSON.stringify(editing) });
      } else {
        const created = await apiFetch<ScheduleRule>("/api/v1/policy/time-based/rules", { method: "POST", body: JSON.stringify(editing) });
        setRules((p) => [...p, created]);
      }
      setEditing(null);
      setRules(await apiFetch<ScheduleRule[]>("/api/v1/policy/time-based/rules").catch(() => rules));
    } catch { setError("Save failed"); }
  };

  const handleDelete = async (id: string) => {
    try { await apiFetch(`/api/v1/policy/time-based/rules/${id}`, { method: "DELETE" }); setRules((p) => p.filter((r) => r.id !== id)); }
    catch { setError("Delete failed"); }
  };

  const addRole = () => {
    if (!editing || !roleInput.trim()) return;
    setEditing({ ...editing, allowed_roles: [...editing.allowed_roles, roleInput.trim()] });
    setRoleInput("");
  };

  const removeRole = (idx: number) => {
    if (!editing) return;
    setEditing({ ...editing, allowed_roles: editing.allowed_roles.filter((_, i) => i !== idx) });
  };

  const cardCls = "rounded-xl border border-gray-200 bg-white p-5 shadow-sm dark:border-gray-700 dark:bg-gray-800";

  return (
    <div className="space-y-6">
      <div className="flex items-center justify-between">
        <div>
          <h1 className="flex items-center gap-2 text-2xl font-bold text-gray-900 dark:text-white"><Clock className="h-6 w-6 text-cyan-600" /> {t("timeBasedAccess.title")}</h1>
          <p className="mt-1 text-sm text-gray-500 dark:text-gray-400">Schedule-based access rules with cron expressions and role restrictions.</p>
        </div>
        <button onClick={() => setEditing({ id: "", name: "", cron: "", start_time: "09:00", end_time: "17:00", timezone: "UTC", allowed_roles: [], effect: "allow", enabled: true, description: "" })} className="flex items-center gap-2 rounded-lg bg-cyan-600 px-4 py-2 text-sm font-medium text-white hover:bg-cyan-700"><Plus className="h-4 w-4" /> New Rule</button>
      </div>

      {error && <div className="flex items-center gap-2 rounded-lg bg-red-50 px-4 py-3 text-sm text-red-700 dark:bg-red-900/20 dark:text-red-400"><AlertCircle className="h-4 w-4 shrink-0" />{error}<button onClick={() => setError(null)} className="ml-auto"><X className="h-4 w-4" /></button></div>}

      {loading ? <div className="flex justify-center py-12"><Loader2 className="h-8 w-8 animate-spin text-cyan-600" /></div>
      : rules.length === 0 ? (
        <div className={cardCls}><div className="py-12 text-center"><Clock className="mx-auto h-12 w-12 text-gray-300" /><p className="mt-4 text-sm text-gray-400">No time-based access rules yet.</p></div></div>
      ) : (
        <div className="space-y-3">
          {rules.map((r) => (
            <div key={r.id} className={cardCls}>
              <div className="flex items-start justify-between">
                <div className="flex-1">
                  <div className="flex items-center gap-2">
                    <h3 className="font-semibold text-gray-900 dark:text-white">{r.name}</h3>
                    <span className={`rounded-full px-2 py-0.5 text-xs font-medium ${r.effect === "allow" ? "bg-green-100 text-green-700 dark:bg-green-900/30" : "bg-red-100 text-red-700 dark:bg-red-900/30"}`}>{r.effect}</span>
                    {!r.enabled && <span className="rounded-full bg-gray-200 px-2 py-0.5 text-xs text-gray-500 dark:bg-gray-700">disabled</span>}
                  </div>
                  {r.description && <p className="mt-1 text-sm text-gray-500">{r.description}</p>}
                  <div className="mt-2 flex flex-wrap items-center gap-3 text-xs text-gray-400">
                    <span className="flex items-center gap-1"><Calendar className="h-3 w-3" /> {r.cron || `${r.start_time}–${r.end_time}`}</span>
                    <span>TZ: {r.timezone}</span>
                    {r.allowed_roles.length > 0 && <span>Roles: {r.allowed_roles.join(", ")}</span>}
                  </div>
                </div>
                <div className="flex gap-1">
                  <button onClick={() => setEditing({ ...r })} className="rounded p-1.5 text-gray-400 hover:text-cyan-600"><Clock className="h-4 w-4" /></button>
                  <button onClick={() => handleDelete(r.id)} className="rounded p-1.5 text-gray-400 hover:bg-red-50 hover:text-red-600"><Trash2 className="h-4 w-4" /></button>
                </div>
              </div>
            </div>
          ))}
        </div>
      )}

      {/* Edit modal */}
      {editing && (
        <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/50" onClick={() => setEditing(null)}>
          <div className="max-h-[90vh] w-full max-w-lg overflow-y-auto rounded-xl bg-white p-6 shadow-xl dark:bg-gray-800" onClick={(e) => e.stopPropagation()}>
            <div className="mb-4 flex items-center justify-between"><h3 className="text-lg font-bold text-gray-900 dark:text-white">{editing.id ? "Edit Rule" : "New Rule"}</h3><button onClick={() => setEditing(null)}><X className="h-5 w-5 text-gray-400" /></button></div>
            <div className="space-y-4">
              <div><label className="mb-1 block text-xs font-semibold uppercase text-gray-400">Name</label><input value={editing.name} onChange={(e) => setEditing({ ...editing, name: e.target.value })} className="w-full rounded-lg border border-gray-300 px-3 py-2 text-sm dark:border-gray-600 dark:bg-gray-900 dark:text-gray-200" /></div>
              <div><label className="mb-1 block text-xs font-semibold uppercase text-gray-400">Description</label><input value={editing.description} onChange={(e) => setEditing({ ...editing, description: e.target.value })} className="w-full rounded-lg border border-gray-300 px-3 py-2 text-sm dark:border-gray-600 dark:bg-gray-900 dark:text-gray-200" /></div>
              <div className="flex gap-4">
                <div className="flex-1"><label className="mb-1 block text-xs font-semibold uppercase text-gray-400">Effect</label><select value={editing.effect} onChange={(e) => setEditing({ ...editing, effect: e.target.value as "allow" | "deny" })} className="w-full rounded-lg border border-gray-300 px-3 py-2 text-sm dark:border-gray-600 dark:bg-gray-900 dark:text-gray-200"><option value="allow">Allow</option><option value="deny">Deny</option></select></div>
                <div><label className="mb-1 block text-xs font-semibold uppercase text-gray-400">Enabled</label><button onClick={() => setEditing({ ...editing, enabled: !editing.enabled })} className={`flex h-[38px] items-center rounded-lg px-4 text-sm font-medium ${editing.enabled ? "bg-green-100 text-green-700 dark:bg-green-900/30" : "bg-gray-100 text-gray-500 dark:bg-gray-700"}`}>{editing.enabled ? "Enabled" : "Disabled"}</button></div>
              </div>
              {/* Cron picker */}
              <div><label className="mb-1 block text-xs font-semibold uppercase text-gray-400">Cron Expression</label><input value={editing.cron} onChange={(e) => setEditing({ ...editing, cron: e.target.value })} placeholder="0 9 * * 1-5" className="w-full rounded-lg border border-gray-300 px-3 py-2 font-mono text-sm dark:border-gray-600 dark:bg-gray-900 dark:text-gray-200" />
                <div className="mt-2 flex gap-1">
                  {weekdays.map((d, i) => (
                    <button key={d} onClick={() => setEditing({ ...editing, cron: `0 ${editing.start_time.split(":")[0] || "9"} * * ${i === 0 ? 0 : i}` })} className="rounded bg-gray-100 px-2 py-1 text-xs text-gray-500 hover:bg-cyan-100 dark:bg-gray-700">{d}</button>
                  ))}
                </div>
                <p className="mt-1 text-xs text-gray-400">Quick pick: click a weekday for a default 9am rule.</p>
              </div>
              <div className="grid grid-cols-3 gap-3">
                <div><label className="mb-1 block text-xs font-semibold uppercase text-gray-400">Start</label><select value={editing.start_time} onChange={(e) => setEditing({ ...editing, start_time: e.target.value })} className="w-full rounded-lg border border-gray-300 px-3 py-2 text-sm dark:border-gray-600 dark:bg-gray-900 dark:text-gray-200">{hours.map((h) => <option key={h} value={h}>{h}</option>)}</select></div>
                <div><label className="mb-1 block text-xs font-semibold uppercase text-gray-400">End</label><select value={editing.end_time} onChange={(e) => setEditing({ ...editing, end_time: e.target.value })} className="w-full rounded-lg border border-gray-300 px-3 py-2 text-sm dark:border-gray-600 dark:bg-gray-900 dark:text-gray-200">{hours.map((h) => <option key={h} value={h}>{h}</option>)}</select></div>
                <div><label className="mb-1 block text-xs font-semibold uppercase text-gray-400">Timezone</label><select value={editing.timezone} onChange={(e) => setEditing({ ...editing, timezone: e.target.value })} className="w-full rounded-lg border border-gray-300 px-3 py-2 text-sm dark:border-gray-600 dark:bg-gray-900 dark:text-gray-200"><option>UTC</option><option>America/New_York</option><option>Europe/London</option><option>Asia/Tokyo</option><option>America/Los_Angeles</option></select></div>
              </div>
              {/* Allowed roles */}
              <div>
                <label className="mb-1 block text-xs font-semibold uppercase text-gray-400">Allowed Roles</label>
                <div className="flex gap-2">
                  <input value={roleInput} onChange={(e) => setRoleInput(e.target.value)} onKeyDown={(e) => e.key === "Enter" && addRole()} placeholder="role name" className="flex-1 rounded-lg border border-gray-300 px-3 py-2 text-sm dark:border-gray-600 dark:bg-gray-900 dark:text-gray-200" />
                  <button onClick={addRole} className="rounded-lg bg-gray-100 px-3 text-sm text-gray-600 dark:bg-gray-700 dark:text-gray-300"><Plus className="h-4 w-4" /></button>
                </div>
                {editing.allowed_roles.length > 0 && (
                  <div className="mt-2 flex flex-wrap gap-2">
                    {editing.allowed_roles.map((role, i) => (
                      <span key={i} className="flex items-center gap-1 rounded-full bg-indigo-100 px-2 py-1 text-xs text-indigo-700 dark:bg-indigo-900/30 dark:text-indigo-400">{role}<button onClick={() => removeRole(i)}><X className="h-3 w-3" /></button></span>
                    ))}
                  </div>
                )}
              </div>
              <button onClick={handleSave} className="flex w-full items-center justify-center gap-2 rounded-lg bg-cyan-600 py-2 text-sm font-medium text-white hover:bg-cyan-700"><Save className="h-4 w-4" /> Save Rule</button>
            </div>
          </div>
        </div>
      )}
    </div>
  );
}
