"use client";
import { useState, useEffect } from "react";
import {
  Smartphone, Loader2, AlertCircle, X, Plus, Check, RefreshCw,
  ChevronRight, CheckCircle2, XCircle, AlertTriangle, Ban,
  Server, Shield, Activity, Clock,
} from "lucide-react";
import { useTranslations } from "@/lib/i18n";

type Tab = "connectors" | "fleet" | "alerts";

const CONNECTOR_TYPES = ["microsoft-intune", "jamf-pro", "google-android"];
const CONNECTORS = [
  { id: "c1", type: "microsoft-intune", name: "Microsoft Intune", status: "connected", last_sync: new Date(Date.now() - 1800000).toISOString(), device_count: 247 },
  { id: "c2", type: "jamf-pro", name: "Jamf Pro", status: "connected", last_sync: new Date(Date.now() - 3600000).toISOString(), device_count: 89 },
  { id: "c3", type: "google-android", name: "Android Enterprise", status: "error", last_sync: new Date(Date.now() - 86400000).toISOString(), device_count: 0 },
];

const FLEET = [
  { id: "d1", user: "alice@company.com", device: "MacBook Pro", os: "macOS 14.2", connector: "Jamf Pro", compliant: true, last_check: new Date(Date.now() - 600000).toISOString() },
  { id: "d2", user: "bob@company.com", device: "Surface Pro", os: "Windows 11", connector: "Intune", compliant: true, last_check: new Date(Date.now() - 1200000).toISOString() },
  { id: "d3", user: "carol@company.com", device: "iPhone 15", os: "iOS 17.3", connector: "Intune", compliant: false, last_check: new Date(Date.now() - 3600000).toISOString() },
  { id: "d4", user: "dave@company.com", device: "Pixel 8", os: "Android 14", connector: "Android Ent.", compliant: false, last_check: new Date(Date.now() - 7200000).toISOString() },
  { id: "d5", user: "eve@company.com", device: "iPad Pro", os: "iPadOS 17", connector: "Jamf Pro", compliant: true, last_check: new Date(Date.now() - 1800000).toISOString() },
];

const ALERTS = FLEET.filter(d => !d.compliant).map((d: any, i: number) => ({ ...d, reason: i === 0 ? "OS update required" : "MDM profile missing", severity: i === 0 ? "medium" : "high" }));

export default function MDMPage() {
  const t = useTranslations();
  const [tab, setTab] = useState<Tab>("connectors");
  const [loading, setLoading] = useState(true);
  const [showForm, setShowForm] = useState(false);
  const [fType, setFType] = useState(CONNECTOR_TYPES[0]);

  const card = "rounded-xl border border-gray-200 bg-white p-5 shadow-sm dark:border-gray-700 dark:bg-gray-800";
  useEffect(() => { setLoading(false); }, []);

  return (
    <div className="space-y-6">
      <div className="flex items-center justify-between">
        <div><h1 className="flex items-center gap-2 text-2xl font-bold text-gray-900 dark:text-white"><Smartphone className="h-6 w-6 text-teal-500" /> {t("mdm.title")}</h1><p className="mt-1 text-sm text-gray-500 dark:text-gray-400">{t("mdm.subtitle")}</p></div>
        <button onClick={() => setShowForm(true)} className="flex items-center gap-1 rounded-lg bg-teal-600 px-3 py-1.5 text-xs font-medium text-white hover:bg-teal-700"><Plus className="h-3 w-3" /> {t("mdm.addConnector")}</button>
      </div>

      <div className="flex gap-1 border-b border-gray-200 dark:border-gray-700 overflow-x-auto">
        {([["connectors", t("mdm.connectors"), Server], ["fleet", `${t("mdm.fleet")} (${FLEET.length})`, Smartphone], ["alerts", `${t("mdm.alerts")} (${ALERTS.length})`, AlertTriangle]] as const).map(([id, label, Icon]) => (
          <button key={id} onClick={() => setTab(id as Tab)} aria-pressed={tab === id} className={`flex items-center gap-1.5 border-b-2 px-4 py-2 text-sm font-medium transition whitespace-nowrap ${tab === id ? "border-teal-600 text-teal-600 dark:text-teal-400" : "border-transparent text-gray-500 hover:text-gray-700 dark:hover:text-gray-300"}`}><Icon className="h-4 w-4" /> {label}</button>
        ))}
      </div>

      {loading ? <div className="flex justify-center py-12"><Loader2 className="h-8 w-8 animate-spin text-teal-500" /></div> : (<>

      {/* CONNECTORS */}
      {tab === "connectors" && (
        <div className="grid grid-cols-1 gap-4 sm:grid-cols-2 lg:grid-cols-3">{CONNECTORS.map(c => (
          <div key={c.id} className={card + " hover:shadow-md transition"}>
            <div className="flex items-start justify-between mb-3"><div className="flex items-center gap-3"><div className="flex h-10 w-10 items-center justify-center rounded-lg bg-gray-100 dark:bg-gray-700"><Server className="h-5 w-5 text-teal-500" /></div><div><h3 className="font-semibold text-sm">{c.name}</h3><code className="text-xs text-gray-400">{c.type}</code></div></div><span className={`flex items-center gap-1 px-1.5 py-0.5 rounded text-xs font-medium ${c.status === "connected" ? "bg-green-100 dark:bg-green-900/30 text-green-600" : "bg-red-100 dark:bg-red-900/30 text-red-600"}`}><span className={`h-1.5 w-1.5 rounded-full ${c.status === "connected" ? "bg-green-500" : "bg-red-500"}`} /> {c.status}</span></div>
            <div className="grid grid-cols-2 gap-2 text-center"><div><p className="text-lg font-bold">{c.device_count}</p><p className="text-xs text-gray-400">{t("mdm.devices")}</p></div><div><p className="text-xs text-gray-400 mt-1">{t("mdm.lastSync")}</p><p className="text-xs font-mono">{new Date(c.last_sync).toLocaleTimeString()}</p></div></div>
          </div>
        ))}</div>
      )}

      {/* FLEET */}
      {tab === "fleet" && (
        <div className="overflow-x-auto"><table className="w-full text-sm">
          <thead className="bg-gray-50 dark:bg-gray-800/50"><tr><th className="px-3 py-2 text-left text-xs text-gray-400">{t("mdm.user")}</th><th className="px-3 py-2 text-left text-xs text-gray-400">{t("mdm.device")}</th><th className="px-3 py-2 text-left text-xs text-gray-400">{t("mdm.os")}</th><th className="px-3 py-2 text-center text-xs text-gray-400">{t("mdm.connector")}</th><th className="px-3 py-2 text-center text-xs text-gray-400">{t("mdm.compliant")}</th><th className="px-3 py-2 text-left text-xs text-gray-400">{t("mdm.lastCheck")}</th></tr></thead>
          <tbody className="divide-y dark:divide-gray-800">{FLEET.map(d => (
            <tr key={d.id} className={`hover:bg-gray-50 dark:hover:bg-gray-900/30 ${!d.compliant ? "bg-red-50 dark:bg-red-950/10" : ""}`}><td className="px-3 py-3 text-xs font-mono">{d.user}</td><td className="px-3 py-3 text-xs">{d.device}</td><td className="px-3 py-3 text-xs">{d.os}</td><td className="px-3 py-3 text-center text-xs">{d.connector}</td><td className="px-3 py-3 text-center">{d.compliant ? <CheckCircle2 className="mx-auto h-4 w-4 text-green-500" /> : <XCircle className="mx-auto h-4 w-4 text-red-500" />}</td><td className="px-3 py-3 text-xs text-gray-400">{new Date(d.last_check).toLocaleTimeString()}</td></tr>
          ))}</tbody>
        </table></div>
      )}

      {/* ALERTS */}
      {tab === "alerts" && (
        <div className="space-y-2">{ALERTS.length === 0 ? <div className={card}><div className="py-12 text-center"><CheckCircle2 className="mx-auto h-12 w-12 text-green-300" /><p className="mt-4 text-sm text-gray-400">{t("mdm.allCompliant")}</p></div></div> :
          ALERTS.map(a => (
            <div key={a.id} className={`${card} flex items-center justify-between !p-3 border-red-200 dark:border-red-800`}>
              <div className="flex items-center gap-3"><div className={`flex h-8 w-8 items-center justify-center rounded-lg ${a.severity === "high" ? "bg-red-100 dark:bg-red-900/30" : "bg-yellow-100 dark:bg-yellow-900/30"}`}><AlertTriangle className={`h-4 w-4 ${a.severity === "high" ? "text-red-500" : "text-yellow-500"}`} /></div><div><div className="flex items-center gap-2"><span className="text-xs font-mono">{a.user}</span><span className="text-xs text-gray-400">{a.device}</span></div><p className="text-xs text-red-500">{a.reason}</p></div></div>
              <div className="flex items-center gap-2"><span className={`px-1.5 py-0.5 rounded text-xs font-medium ${a.severity === "high" ? "bg-red-100 dark:bg-red-900/30 text-red-600" : "bg-yellow-100 dark:bg-yellow-900/30 text-yellow-600"}`}>{a.severity}</span><button className="rounded-lg border border-gray-300 px-2 py-1 text-xs dark:border-gray-700">{t("mdm.remediate")}</button></div>
            </div>
          ))}
        </div>
      )}

      </>)}

      {showForm && (
        <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/50" onClick={() => setShowForm(false)}>
          <div role="dialog" aria-modal="true" className="mx-4 w-full max-w-md rounded-xl bg-white p-6 shadow-xl dark:bg-gray-800" onClick={e => e.stopPropagation()}>
            <h3 className="flex items-center gap-2 text-lg font-semibold text-gray-900 dark:text-white"><Plus className="h-5 w-5 text-teal-500" /> {t("mdm.addConnector")}</h3>
            <div className="mt-4 space-y-3"><div><label className="text-sm font-medium">{t("mdm.connectorType")}</label><select value={fType} onChange={e => setFType(e.target.value)} className="mt-1 block w-full rounded-lg border dark:border-gray-700 dark:bg-gray-900 px-3 py-2 text-sm">{CONNECTOR_TYPES.map(ct => <option key={ct} value={ct}>{ct}</option>)}</select></div></div>
            <div className="mt-4 flex justify-end gap-2"><button onClick={() => setShowForm(false)} className="rounded-lg border border-gray-300 px-4 py-2 text-sm dark:border-gray-700">{t("common.cancel")}</button><button onClick={() => setShowForm(false)} className="rounded-lg bg-teal-600 px-4 py-2 text-sm font-medium text-white hover:bg-teal-700">{t("mdm.connect")}</button></div>
          </div>
        </div>
      )}
    </div>
  );
}
