"use client";
import { useState } from "react";
import {
  Flag, Loader2, AlertCircle, X, Plus, Check, Save, ChevronRight,
  ToggleLeft, Users, Percent, Edit, Trash2, Globe,
} from "lucide-react";
import { useTranslations } from "@/lib/i18n";

const TENANT_ID = "00000000-0000-0000-0000-000000000001";

type Tab = "list" | "create";

interface FlagDef { key: string; name: string; desc: string; enabled: boolean; rollout: number; tenants: Record<string, boolean>; }

const FLAGS: FlagDef[] = [
  { key: "graphql_api", name: "GraphQL API", desc: "Enable /graphql endpoint", enabled: false, rollout: 0, tenants: { default: false } },
  { key: "dlp_egress", name: "DLP Egress Control", desc: "Gateway PII detection middleware", enabled: true, rollout: 100, tenants: { default: true } },
  { key: "ueba_scoring", name: "UEBA Scoring", desc: "Isolation forest anomaly detection", enabled: true, rollout: 100, tenants: { default: true } },
  { key: "soar_engine", name: "SOAR Engine", desc: "Automated threat response playbooks", enabled: true, rollout: 75, tenants: { default: true, enterprise: false } },
  { key: "pqc_signing", name: "PQC Signing", desc: "Post-quantum audit chain signatures", enabled: false, rollout: 0, tenants: { default: false } },
  { key: "consent_cascade", name: "Consent Cascade", desc: "GDPR Art.17 token revocation on consent withdrawal", enabled: true, rollout: 100, tenants: { default: true } },
  { key: "adaptive_mfa", name: "Adaptive MFA", desc: "Risk-based step-up authentication", enabled: true, rollout: 90, tenants: { default: true } },
];

export default function FeatureFlagsPage() {
  const t = useTranslations();
  const [tab, setTab] = useState<Tab>("list");
  const [flags, setFlags] = useState(FLAGS);
  const [editingKey, setEditingKey] = useState<string | null>(null);
  const [fName, setFName] = useState("");
  const [fKey, setFKey] = useState("");
  const [fDesc, setFDesc] = useState("");
  const [fDefault, setFDefault] = useState(false);
  const [fRollout, setFRollout] = useState(0);
  const [saving, setSaving] = useState(false);

  const card = "rounded-xl border border-gray-200 bg-white p-5 shadow-sm dark:border-gray-700 dark:bg-gray-800";

  const toggleFlag = (key: string) => setFlags(prev => prev.map(f => f.key === key ? { ...f, enabled: !f.enabled } : f));

  const startEdit = (f: FlagDef) => { setEditingKey(f.key); setFName(f.name); setFKey(f.key); setFDesc(f.desc); setFDefault(f.enabled); setFRollout(f.rollout); setTab("create"); };
  const startNew = () => { setEditingKey(null); setFName(""); setFKey(""); setFDesc(""); setFDefault(false); setFRollout(0); setTab("create"); };

  const saveFlag = () => {
    if (!fName || !fKey) return; setSaving(true);
    setTimeout(() => {
      if (editingKey) { setFlags(prev => prev.map(f => f.key === editingKey ? { ...f, name: fName, desc: fDesc, enabled: fDefault, rollout: fRollout } : f)); }
      else { setFlags(prev => [...prev, { key: fKey, name: fName, desc: fDesc, enabled: fDefault, rollout: fRollout, tenants: { default: fDefault } }]); }
      setSaving(false); setTab("list");
    }, 600);
  };

  return (
    <div className="space-y-6">
      <div className="flex items-center justify-between">
        <div><h1 className="flex items-center gap-2 text-2xl font-bold text-gray-900 dark:text-white"><Flag className="h-6 w-6 text-indigo-500" /> {t("flags.title")}</h1><p className="mt-1 text-sm text-gray-500 dark:text-gray-400">{t("flags.subtitle")}</p></div>
        <button onClick={startNew} className="flex items-center gap-1 rounded-lg bg-indigo-600 px-3 py-1.5 text-xs font-medium text-white hover:bg-indigo-700"><Plus className="h-3 w-3" /> {t("flags.create")}</button>
      </div>

      <div className="flex gap-1 border-b border-gray-200 dark:border-gray-700 overflow-x-auto">
        {([["list", `${t("flags.flagList")} (${flags.length})`, ToggleLeft], ["create", t("flags.createTab"), Edit]] as const).map(([id, label, Icon]) => (
          <button key={id} onClick={() => setTab(id as Tab)} aria-pressed={tab === id} className={`flex items-center gap-1.5 border-b-2 px-4 py-2 text-sm font-medium transition whitespace-nowrap ${tab === id ? "border-indigo-600 text-indigo-600 dark:text-indigo-400" : "border-transparent text-gray-500 hover:text-gray-700 dark:hover:text-gray-300"}`}><Icon className="h-4 w-4" /> {label}</button>
        ))}
      </div>

      {/* LIST */}
      {tab === "list" && (
        <div className="space-y-3">{flags.map(f => (
          <div key={f.key} className={`${card} flex items-center justify-between !p-3`}>
            <div className="flex items-center gap-3 flex-1 min-w-0"><div className="flex h-9 w-9 items-center justify-center rounded-lg bg-indigo-100 dark:bg-indigo-900/30"><Flag className="h-4 w-4 text-indigo-500" /></div><div className="min-w-0"><div className="flex items-center gap-2"><code className="text-sm font-mono text-indigo-500">{f.key}</code><span className={`px-1.5 py-0.5 rounded text-xs font-medium ${f.enabled ? "bg-green-100 dark:bg-green-900/30 text-green-600" : "bg-gray-100 dark:bg-gray-800 text-gray-400"}`}>{f.enabled ? "on" : "off"}</span></div><p className="text-xs text-gray-400 truncate">{f.desc}</p><div className="flex items-center gap-2 mt-1"><span className="flex items-center gap-1 text-xs text-gray-400"><Percent className="h-2.5 w-2.5" />{f.rollout}%</span><span className="flex items-center gap-1 text-xs text-gray-400"><Globe className="h-2.5 w-2.5" />{Object.keys(f.tenants).length} {t("flags.tenants")}</span></div></div></div>
            <div className="flex items-center gap-2"><button onClick={() => startEdit(f)} aria-label={"Edit " + f.key} className="rounded p-1.5 text-gray-400 hover:bg-gray-100 dark:hover:bg-gray-700"><Edit className="h-3.5 w-3.5" /></button><button onClick={() => toggleFlag(f.key)} aria-pressed={f.enabled} className={`relative h-6 w-11 rounded-full transition ${f.enabled ? "bg-green-500" : "bg-gray-300 dark:bg-gray-700"}`}><span className={`absolute top-0.5 h-5 w-5 rounded-full bg-white transition ${f.enabled ? "left-5" : "left-0.5"}`} /></button></div>
          </div>
        ))}</div>
      )}

      {/* CREATE/EDIT */}
      {tab === "create" && (
        <div className="grid grid-cols-1 gap-6 lg:grid-cols-2">
          <div className={card}>
            <h3 className="mb-4 flex items-center gap-2 text-sm font-semibold uppercase text-gray-400">{editingKey ? t("flags.editFlag") : t("flags.newFlag")}</h3>
            <div className="space-y-3">
              <div><label className="text-sm font-medium">{t("flags.displayName")}</label><input type="text" value={fName} onChange={e => setFName(e.target.value)} placeholder="GraphQL API" className="mt-1 w-full rounded-lg border dark:border-gray-700 dark:bg-gray-900 px-3 py-2 text-sm" autoFocus /></div>
              <div><label className="text-sm font-medium">{t("flags.flagKey")}</label><input type="text" value={fKey} onChange={e => setFKey(e.target.value)} placeholder="graphql_api" disabled={!!editingKey} className="mt-1 w-full rounded-lg border dark:border-gray-700 dark:bg-gray-900 px-3 py-2 text-sm font-mono disabled:opacity-50" /></div>
              <div><label className="text-sm font-medium">{t("flags.description")}</label><input type="text" value={fDesc} onChange={e => setFDesc(e.target.value)} placeholder="Enable /graphql endpoint" className="mt-1 w-full rounded-lg border dark:border-gray-700 dark:bg-gray-900 px-3 py-2 text-sm" /></div>
              <div><label className="flex items-center justify-between rounded-lg border p-3 dark:border-gray-700"><span className="text-sm font-medium">{t("flags.defaultState")}</span><button onClick={() => setFDefault(!fDefault)} aria-pressed={fDefault} className={`relative h-6 w-11 rounded-full transition ${fDefault ? "bg-green-500" : "bg-gray-300 dark:bg-gray-700"}`}><span className={`absolute top-0.5 h-5 w-5 rounded-full bg-white transition ${fDefault ? "left-5" : "left-0.5"}`} /></button></label></div>
              <div><label className="text-sm font-medium flex items-center gap-2"><Percent className="h-4 w-4" /> {t("flags.rolloutPercent")}</label><div className="mt-1 flex items-center gap-3"><input type="range" min={0} max={100} value={fRollout} onChange={e => setFRollout(parseInt(e.target.value))} className="flex-1 accent-indigo-500" /><span className="text-sm font-mono w-12">{fRollout}%</span></div></div>
              <button onClick={saveFlag} disabled={!fName || !fKey || saving} className="flex items-center gap-2 rounded-lg bg-indigo-600 px-4 py-2 text-sm font-medium text-white hover:bg-indigo-700 disabled:opacity-50">{saving ? <Loader2 className="h-4 w-4 animate-spin" /> : <Save className="h-4 w-4" />} {t("flags.save")}</button>
            </div>
          </div>
          <div className={card}>
            <h3 className="mb-3 text-sm font-semibold uppercase text-gray-400">{t("flags.tenantOverrides")}</h3>
            <div className="space-y-2">{["default", "enterprise", "trial"].map(tn => (
              <div key={tn} className="flex items-center justify-between rounded-lg border p-3 dark:border-gray-700"><div className="flex items-center gap-2"><Users className="h-4 w-4 text-gray-400" /><code className="text-xs font-mono">{tn}</code></div><select className="rounded-lg border dark:border-gray-700 dark:bg-gray-900 px-2 py-1 text-xs"><option value="inherit">{t("flags.inherit")}</option><option value="on">on</option><option value="off">off</option></select></div>
            ))}</div>
          </div>
        </div>
      )}
    </div>
  );
}
