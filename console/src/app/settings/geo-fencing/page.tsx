"use client";
import { useState, useEffect, useCallback } from "react";
import { Globe, Plus, X, Ban, Save, Loader2, AlertCircle } from "lucide-react";
import { useTranslations } from "@/lib/i18n";
interface GeoRule { id: string; countries: string[]; cidrs: string[]; action: "allow" | "deny" | "challenge"; }
interface GeoData { enabled: boolean; rules: GeoRule[]; whitelist_ips: string[]; }
const actionColors: Record<string, string> = { allow: "bg-green-100 text-green-800 dark:bg-green-900/30 dark:text-green-400", deny: "bg-red-100 text-red-800 dark:bg-red-900/30 dark:text-red-400", challenge: "bg-yellow-100 text-yellow-800 dark:bg-yellow-900/30 dark:text-yellow-400" };
export default function GeoFencingPage() {
  const t = useTranslations();

  const [data, setData] = useState<GeoData>({ enabled: true, rules: [{ id: "r1", countries: ["CN", "RU"], cidrs: [], action: "deny" }], whitelist_ips: ["10.0.0.0/8"] });
  const [showAdd, setShowAdd] = useState(false);
  const [form, setForm] = useState({ countries: "", cidrs: "", action: "deny" });
  const [saving, setSaving] = useState(false);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState("");
  const [saved, setSaved] = useState(false);
  const addRule = () => { const rule: GeoRule = { id: "r" + Date.now(), countries: form.countries.split(",").map((s) => s.trim()).filter(Boolean), cidrs: form.cidrs.split(",").map((s) => s.trim()).filter(Boolean), action: form.action as GeoRule["action"] }; setData({ ...data, rules: [...data.rules, rule] }); setShowAdd(false); setForm({ countries: "", cidrs: "", action: "deny" }); };
  const removeRule = (id: string) => setData({ ...data, rules: data.rules.filter((r) => r.id !== id) });

  useEffect(() => {
    setLoading(true); setError("");
    fetch("/api/v1/settings/geo-fencing", { headers: { "Authorization": `Bearer ${localStorage.getItem("ggid_access_token") || ""}`, "X-Tenant-ID": "00000000-0000-0000-0000-000000000001" } })
      .then(async (res) => { if (res.ok) { const d = await res.json(); if (d) setData(d); } })
      .catch(() => { /* use defaults */ })
      .finally(() => setLoading(false));
  }, []);

  const save = async () => {
    setSaving(true); setError("");
    try {
      const res = await fetch("/api/v1/settings/geo-fencing", { method: "PUT", headers: { "Authorization": `Bearer ${localStorage.getItem("ggid_access_token") || ""}`, "Content-Type": "application/json", "X-Tenant-ID": "00000000-0000-0000-0000-000000000001" }, body: JSON.stringify(data) });
      if (!res.ok) return null;
      setSaved(true); setTimeout(() => setSaved(false), 2000);
    } catch (e) {
      setError(e instanceof Error ? e.message : "Failed to save geo-fencing rules");
    } finally { setSaving(false); }
  };

  return (
    <div className="space-y-6">
      <div className="flex items-center justify-between"><div><h1 className="text-2xl font-bold flex items-center gap-2"><Globe className="w-6 h-6 text-blue-500" /> {t("big1.geoFencing.title")}</h1><p className="text-sm text-gray-500 mt-1">{t("big1.geoFencing.controlAccessByGeographicLocationAndIPRanges")}</p></div><div className="flex items-center gap-3"><label className="flex items-center gap-2 text-sm"><input type="checkbox" checked={data.enabled} onChange={(e) => setData({ ...data, enabled: e.target.checked })} className="rounded" aria-label="Toggle geo-fencing" />{t("big1.geoFencing.enabled")}</label><button onClick={() => setShowAdd(true)} className="px-4 py-2 rounded-lg bg-blue-600 text-white text-sm font-medium flex items-center gap-2" aria-label="Add geo rule"><Plus className="w-4 h-4" />{t("big1.geoFencing.addRule")}</button><button onClick={save} disabled={saving} className="px-4 py-2 rounded-lg bg-green-600 text-white text-sm font-medium flex items-center gap-2 disabled:opacity-50" aria-label="Save geo-fencing rules">{saving ? <Loader2 className="w-4 h-4 animate-spin" /> : <Save className="w-4 h-4" />}{t("big1.geoFencing.save")}</button></div></div>
      {loading && <div className="flex items-center gap-2 text-sm text-gray-500"><Loader2 className="w-4 h-4 animate-spin" />{t("big1.geoFencing.loadingGeoFencingRules")}</div>}
      {error && <div className="rounded-lg border border-red-200 dark:border-red-900 bg-red-50 dark:bg-red-900/20 p-3 text-sm text-red-600 flex items-center gap-2"><AlertCircle className="w-4 h-4" /> {error}</div>}
      {saved && <div className="text-sm text-green-600">{t("big1.geoFencing.rulesSavedSuccessfully")}</div>}
      <div className="overflow-x-auto rounded-lg border dark:border-gray-800"><table className="w-full text-sm"><thead className="bg-gray-50 dark:bg-gray-900/50"><tr><th className="px-4 py-3 text-left font-medium">{t("big1.geoFencing.countries")}</th><th className="px-4 py-3 text-left font-medium">{t("big1.geoFencing.cidrRanges")}</th><th className="px-4 py-3 text-left font-medium">{t("big1.geoFencing.action")}</th><th className="px-4 py-3 text-left font-medium"></th></tr></thead><tbody className="divide-y dark:divide-gray-800">{data.rules.map((r) => (<tr key={r.id} className="hover:bg-gray-50 dark:hover:bg-gray-900/30"><td className="px-4 py-3"><div className="flex flex-wrap gap-1">{r.countries.map((c) => <span key={c} className="px-1.5 py-0.5 rounded text-xs bg-gray-100 dark:bg-gray-800 font-mono">{c}</span>)}{r.countries.length === 0 && <span className="text-xs text-gray-400">{t("big1.geoFencing.all")}</span>}</div></td><td className="px-4 py-3"><div className="flex flex-wrap gap-1">{r.cidrs.map((c) => <span key={c} className="px-1.5 py-0.5 rounded text-xs font-mono text-gray-500">{c}</span>)}{r.cidrs.length === 0 && <span className="text-xs text-gray-400">-</span>}</div></td><td className="px-4 py-3"><span className={"px-2 py-0.5 rounded text-xs " + actionColors[r.action]}>{r.action}</span></td><td className="px-4 py-3"><button onClick={() => removeRule(r.id)} className="text-red-500"><Ban className="w-4 h-4" /></button></td></tr>))}</tbody></table></div>
      <div className="rounded-lg border dark:border-gray-800 p-4"><h3 className="text-sm font-semibold mb-2">{t("big1.geoFencing.whitelistIps")}</h3><div className="flex flex-wrap gap-1">{data.whitelist_ips.map((ip) => <span key={ip} className="px-2 py-0.5 rounded text-xs bg-green-100 dark:bg-green-900/30 dark:text-green-400 font-mono">{ip}</span>)}</div></div>
      {showAdd && (<div className="fixed inset-0 bg-black/50 flex items-center justify-center z-50" onClick={() => setShowAdd(false)}><div role="dialog" aria-modal="true" className="bg-white dark:bg-gray-900 rounded-lg shadow-xl max-w-md w-full mx-4" onClick={(e) => e.stopPropagation()}><div className="flex items-center justify-between px-6 py-4 border-b dark:border-gray-800"><h3 className="font-semibold">{t("big1.geoFencing.addGeoRule")}</h3><button onClick={() => setShowAdd(false)} aria-label="Close"><X className="w-5 h-5 text-gray-400" /></button></div><div className="px-6 py-4 space-y-3"><div><label className="text-sm font-medium">{t("big1.geoFencing.countriesCommaSeparated")}</label><input type="text" value={form.countries} onChange={(e) => setForm({ ...form, countries: e.target.value })} placeholder="US, CA, GB" className="w-full mt-1 px-3 py-2 rounded-lg border dark:border-gray-700 dark:bg-gray-800 text-sm" /></div><div><label className="text-sm font-medium">{t("big1.geoFencing.cidrRangesCommaSeparated")}</label><input type="text" value={form.cidrs} onChange={(e) => setForm({ ...form, cidrs: e.target.value })} placeholder="192.168.0.0/16" className="w-full mt-1 px-3 py-2 rounded-lg border dark:border-gray-700 dark:bg-gray-800 text-sm font-mono" /></div><div><label className="text-sm font-medium">{t("big1.geoFencing.action")}</label><select value={form.action} onChange={(e) => setForm({ ...form, action: e.target.value })} className="w-full mt-1 px-3 py-2 rounded-lg border dark:border-gray-700 dark:bg-gray-800 text-sm"><option value="allow">{t("big1.geoFencing.allow")}</option><option value="deny">{t("big1.geoFencing.deny")}</option><option value="challenge">{t("big1.geoFencing.challengeMFA")}</option></select></div></div><div className="flex justify-end gap-2 px-6 py-4 border-t dark:border-gray-800"><button onClick={() => setShowAdd(false)} className="px-4 py-2 rounded-lg border dark:border-gray-700 text-sm">{t("big1.geoFencing.cancel")}</button><button onClick={addRule} className="px-4 py-2 rounded-lg bg-blue-600 text-white text-sm font-medium" aria-label="Action">{t("big1.geoFencing.add")}</button></div></div></div>)}
    </div>
  );
}
