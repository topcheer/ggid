"use client";
import { useState } from "react";
import {
  Workflow, Loader2, AlertCircle, X, Plus, Check, Play,
  Lock, MessageSquare, Smartphone, Fingerprint, Mail, Globe,
  KeySquare, Shield, ChevronRight, ChevronDown, Eye, Zap,
  Layers, User, ArrowRight, Save,
} from "lucide-react";
import { useTranslations } from "@/lib/i18n";

type Tab = "flows" | "editor" | "preview";

const STEP_TYPES = [
  { type: "password", label: "Password", icon: Lock, color: "text-blue-600 bg-blue-50 dark:bg-blue-900/30" },
  { type: "sms_otp", label: "SMS OTP", icon: MessageSquare, color: "text-purple-600 bg-purple-50 dark:bg-purple-900/30" },
  { type: "totp", label: "TOTP", icon: Smartphone, color: "text-indigo-600 bg-indigo-50 dark:bg-indigo-900/30" },
  { type: "webauthn", label: "Passkey", icon: Fingerprint, color: "text-green-600 bg-green-50 dark:bg-green-900/30" },
  { type: "email_link", label: "Email Link", icon: Mail, color: "text-pink-600 bg-pink-50 dark:bg-pink-900/30" },
  { type: "social_google", label: "Google", icon: Globe, color: "text-red-600 bg-red-50 dark:bg-red-900/30" },
  { type: "saml", label: "SAML SSO", icon: KeySquare, color: "text-amber-600 bg-amber-50 dark:bg-amber-900/30" },
  { type: "risk_check", label: "Risk Check", icon: Shield, color: "text-orange-600 bg-orange-50 dark:bg-orange-900/30" },
];

interface FlowStep { id: string; type: string; label: string; required: boolean; }
interface Flow { id: string; name: string; desc: string; steps: FlowStep[]; enabled: boolean; default: boolean; }

const PRESET_FLOWS: Flow[] = [
  { id: "f1", name: "Standard Login", desc: "Password + optional MFA", enabled: true, default: true, steps: [
    { id: "s1", type: "password", label: "Password", required: true },
    { id: "s2", type: "totp", label: "TOTP (if enrolled)", required: false },
  ]},
  { id: "f2", name: "Passwordless", desc: "Passkey-first experience", enabled: true, default: false, steps: [
    { id: "s1", type: "webauthn", label: "Passkey", required: true },
    { id: "s2", type: "email_link", label: "Email fallback", required: false },
  ]},
  { id: "f3", name: "High Security", desc: "Password + MFA + risk check", enabled: false, default: false, steps: [
    { id: "s1", type: "password", label: "Password", required: true },
    { id: "s2", type: "risk_check", label: "Risk Evaluation", required: true },
    { id: "s3", type: "totp", label: "TOTP", required: true },
  ]},
];

export default function LoginFlowsPage() {
  const t = useTranslations();
  const [tab, setTab] = useState<Tab>("flows");
  const [flows, setFlows] = useState<Flow[]>(PRESET_FLOWS);
  const [editingFlow, setEditingFlow] = useState<Flow | null>(null);
  const [previewStep, setPreviewStep] = useState(0);
  const [showPreview, setShowPreview] = useState(false);

  const card = "rounded-xl border border-gray-200 bg-white p-5 shadow-sm dark:border-gray-700 dark:bg-gray-800";

  const startNew = () => { setEditingFlow({ id: `f${Date.now()}`, name: "", desc: "", steps: [], enabled: true, default: false }); setTab("editor"); };
  const addStep = (stepType: typeof STEP_TYPES[0]) => { if (!editingFlow) return; setEditingFlow({ ...editingFlow, steps: [...editingFlow.steps, { id: `s${Date.now()}`, type: stepType.type, label: stepType.label, required: false }] }); };
  const removeStep = (id: string) => { if (!editingFlow) return; setEditingFlow({ ...editingFlow, steps: editingFlow.steps.filter(s => s.id !== id) }); };
  const moveStep = (idx: number, dir: "up" | "down") => { if (!editingFlow) return; const steps = [...editingFlow.steps]; const target = dir === "up" ? idx - 1 : idx + 1; if (target < 0 || target >= steps.length) return; [steps[idx], steps[target]] = [steps[target], steps[idx]]; setEditingFlow({ ...editingFlow, steps }); };
  const saveFlow = () => { if (!editingFlow?.name) return; setFlows(prev => { const exists = prev.find(f => f.id === editingFlow.id); if (exists) return prev.map(f => f.id === editingFlow.id ? editingFlow : f); return [...prev, editingFlow]; }); setEditingFlow(null); setTab("flows"); };

  return (
    <div className="space-y-6">
      <div className="flex items-center justify-between">
        <div><h1 className="flex items-center gap-2 text-2xl font-bold text-gray-900 dark:text-white"><Workflow className="h-6 w-6 text-blue-500" /> {t("loginFlows.title")}</h1><p className="mt-1 text-sm text-gray-500 dark:text-gray-400">{t("loginFlows.subtitle")}</p></div>
        <button onClick={startNew} className="flex items-center gap-1 rounded-lg bg-blue-600 px-3 py-1.5 text-xs font-medium text-white hover:bg-blue-700"><Plus className="h-3 w-3" /> {t("loginFlows.newFlow")}</button>
      </div>

      <div className="flex gap-1 border-b border-gray-200 dark:border-gray-700 overflow-x-auto">
        {([["flows", t("loginFlows.flowList"), Layers], ["editor", t("loginFlows.editor"), Workflow], ["preview", t("loginFlows.preview"), Eye]] as const).map(([id, label, Icon]) => (
          <button key={id} onClick={() => setTab(id as Tab)} aria-pressed={tab === id} className={`flex items-center gap-1.5 border-b-2 px-4 py-2 text-sm font-medium transition whitespace-nowrap ${tab === id ? "border-blue-600 text-blue-600 dark:text-blue-400" : "border-transparent text-gray-500 hover:text-gray-700 dark:hover:text-gray-300"}`}><Icon className="h-4 w-4" /> {label}</button>
        ))}
      </div>

      {/* FLOWS */}
      {tab === "flows" && (
        <div className="space-y-3">{flows.map(f => (
          <div key={f.id} className={card}>
            <div className="flex items-start justify-between mb-3">
              <div className="flex items-center gap-3"><div className="flex h-10 w-10 items-center justify-center rounded-lg bg-blue-100 dark:bg-blue-900/30"><Workflow className="h-5 w-5 text-blue-500" /></div><div><div className="flex items-center gap-2"><h3 className="font-semibold text-sm">{f.name}</h3>{f.default && <span className="px-1.5 py-0.5 rounded text-xs bg-green-100 dark:bg-green-900/30 text-green-600">{t("loginFlows.default")}</span>}{!f.enabled && <span className="px-1.5 py-0.5 rounded text-xs bg-gray-100 dark:bg-gray-800 text-gray-400">{t("loginFlows.disabled")}</span>}</div><p className="text-xs text-gray-400">{f.desc}</p></div></div>
              <div className="flex gap-1"><button onClick={() => { setEditingFlow(f); setTab("editor"); }} aria-label={"Edit " + f.name} className="rounded p-1.5 text-gray-400 hover:bg-gray-100 dark:hover:bg-gray-700"><Workflow className="h-3.5 w-3.5" /></button><button onClick={() => { setEditingFlow(f); setShowPreview(true); setPreviewStep(0); setTab("preview"); }} aria-label={"Preview " + f.name} className="rounded p-1.5 text-gray-400 hover:bg-gray-100 dark:hover:bg-gray-700"><Eye className="h-3.5 w-3.5" /></button></div>
            </div>
            <div className="flex items-center gap-1 flex-wrap">{f.steps.map((s, i) => { const st = STEP_TYPES.find(t => t.type === s.type); const SIcon = st?.icon || Lock; return (
              <div key={s.id} className="flex items-center gap-1">{i > 0 && <ArrowRight className="h-3 w-3 text-gray-300" />}<span className={`flex items-center gap-1 rounded-lg px-2 py-1 text-xs ${st?.color || "bg-gray-100 dark:bg-gray-700"}`}><SIcon className="h-3 w-3" /> {s.label}</span></div>
            );})}</div>
          </div>
        ))}</div>
      )}

      {/* EDITOR */}
      {tab === "editor" && editingFlow && (
        <div className="grid grid-cols-1 gap-6 lg:grid-cols-3">
          <div className="lg:col-span-1">
            <div className={card}>
              <h3 className="mb-3 text-sm font-semibold uppercase text-gray-400">{t("loginFlows.stepPalette")}</h3>
              <div className="space-y-2">{STEP_TYPES.map(st => { const SIcon = st.icon; return (
                <button key={st.type} onClick={() => addStep(st)} className="flex w-full items-center gap-2 rounded-lg border p-2 text-left transition hover:bg-gray-50 dark:border-gray-700 dark:hover:bg-gray-900/30">
                  <span className={`flex h-7 w-7 items-center justify-center rounded ${st.color}`}><SIcon className="h-3.5 w-3.5" /></span><span className="text-xs font-medium">{st.label}</span><Plus className="ml-auto h-3 w-3 text-gray-400" /></button>
              );})}</div>
            </div>
          </div>
          <div className="lg:col-span-2">
            <div className={card}>
              <div className="mb-4 space-y-3"><div><label className="text-sm font-medium">{t("loginFlows.flowName")}</label><input type="text" value={editingFlow.name} onChange={e => setEditingFlow({ ...editingFlow, name: e.target.value })} placeholder="Custom Login Flow" className="mt-1 w-full rounded-lg border dark:border-gray-700 dark:bg-gray-900 px-3 py-2 text-sm" /></div><div><label className="text-sm font-medium">{t("loginFlows.description")}</label><input type="text" value={editingFlow.desc} onChange={e => setEditingFlow({ ...editingFlow, desc: e.target.value })} placeholder="Password + risk-based MFA" className="mt-1 w-full rounded-lg border dark:border-gray-700 dark:bg-gray-900 px-3 py-2 text-sm" /></div></div>
              <div className="space-y-2">
                {editingFlow.steps.length === 0 && <div className="py-8 text-center text-sm text-gray-400">{t("loginFlows.noSteps")}</div>}
                {editingFlow.steps.map((s, i) => { const st = STEP_TYPES.find(t => t.type === s.type); const SIcon = st?.icon || Lock; return (
                  <div key={s.id} className="flex items-center gap-3 rounded-lg border p-3 dark:border-gray-700">
                    <span className="text-xs font-mono text-gray-400 w-6">{i + 1}</span>
                    <span className={`flex h-8 w-8 items-center justify-center rounded-lg ${st?.color || "bg-gray-100 dark:bg-gray-700"}`}><SIcon className="h-4 w-4" /></span>
                    <div className="flex-1"><span className="text-sm font-medium">{s.label}</span><p className="text-xs text-gray-400">{s.type}</p></div>
                    <label className="flex items-center gap-1 text-xs"><input type="checkbox" checked={s.required} onChange={() => setEditingFlow({ ...editingFlow, steps: editingFlow.steps.map(x => x.id === s.id ? { ...x, required: !x.required } : x) })} className="rounded" /> {t("loginFlows.required")}</label>
                    <div className="flex gap-1"><button onClick={() => moveStep(i, "up")} disabled={i === 0} className="rounded p-1 text-gray-400 hover:bg-gray-100 dark:hover:bg-gray-700 disabled:opacity-30"><ChevronDown className="h-3 w-3 rotate-180" /></button><button onClick={() => moveStep(i, "down")} disabled={i === editingFlow.steps.length - 1} className="rounded p-1 text-gray-400 hover:bg-gray-100 dark:hover:bg-gray-700 disabled:opacity-30"><ChevronDown className="h-3 w-3" /></button><button onClick={() => removeStep(s.id)} className="rounded p-1 text-red-400 hover:bg-red-50 dark:hover:bg-red-900/20"><X className="h-3 w-3" /></button></div>
                  </div>
                );})}
              </div>
              <div className="mt-4 flex justify-end gap-2"><button onClick={() => { setEditingFlow(null); setTab("flows"); }} className="rounded-lg border border-gray-300 px-4 py-2 text-sm dark:border-gray-700">{t("common.cancel")}</button><button onClick={saveFlow} disabled={!editingFlow.name} className="flex items-center gap-1 rounded-lg bg-blue-600 px-4 py-2 text-sm font-medium text-white hover:bg-blue-700 disabled:opacity-50"><Save className="h-4 w-4" /> {t("loginFlows.save")}</button></div>
            </div>
          </div>
        </div>
      )}

      {/* PREVIEW */}
      {tab === "preview" && (
        <div className={card}>
          <h3 className="mb-4 flex items-center gap-2 text-sm font-semibold uppercase text-gray-400"><Eye className="h-4 w-4" /> {t("loginFlows.livePreview")}</h3>
          {editingFlow ? (
            <div className="mx-auto max-w-sm">
              <div className="rounded-xl border-2 border-gray-200 dark:border-gray-700 p-6">
                <div className="text-center mb-6"><User className="mx-auto h-12 w-12 text-gray-300" /><h2 className="mt-2 text-lg font-bold">{t("loginFlows.signIn")}</h2><p className="text-xs text-gray-400">{editingFlow.name || t("loginFlows.untitled")}</p></div>
                {editingFlow.steps.slice(0, previewStep + 1).map((s, i) => { const st = STEP_TYPES.find(t => t.type === s.type); const SIcon = st?.icon || Lock; return (
                  <div key={s.id} className={`mb-3 ${i < previewStep ? "opacity-50" : ""}`}>
                    <label className="text-xs font-medium">{s.label}</label>
                    <div className="mt-1 flex items-center gap-2 rounded-lg border p-3 dark:border-gray-700"><SIcon className="h-4 w-4 text-gray-400" /><div className="flex-1 h-4 rounded bg-gray-100 dark:bg-gray-700" />{i < previewStep && <Check className="h-4 w-4 text-green-500" />}</div>
                  </div>
                );})}
                {previewStep < editingFlow.steps.length - 1 ? (
                  <button onClick={() => setPreviewStep(previewStep + 1)} className="mt-2 w-full rounded-lg bg-blue-600 py-2 text-sm font-medium text-white hover:bg-blue-700">{t("loginFlows.continue")}</button>
                ) : (
                  <div className="mt-2 flex items-center justify-center gap-2 rounded-lg bg-green-50 dark:bg-green-900/20 py-3 text-sm font-medium text-green-600"><Check className="h-4 w-4" /> {t("loginFlows.authSuccess")}</div>
                )}
                <button onClick={() => setPreviewStep(0)} className="mt-3 w-full text-center text-xs text-blue-600 hover:underline">{t("loginFlows.restart")}</button>
              </div>
            </div>
          ) : (
            <div className="py-8 text-center"><Eye className="mx-auto h-10 w-10 text-gray-300" /><p className="mt-3 text-sm text-gray-400">{t("loginFlows.selectFlow")}</p></div>
          )}
        </div>
      )}
    </div>
  );
}
