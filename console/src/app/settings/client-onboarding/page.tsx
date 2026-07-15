"use client";
import { useTranslations } from "@/lib/i18n";
import { useState, useEffect } from "react";
import { Rocket, Check, ChevronRight, Copy } from "lucide-react";
const grantTypes = ["authorization_code", "client_credentials", "refresh_token", "password", "device_code", "implicit"];
const allScopes = ["openid", "profile", "email", "read:users", "write:users", "read:roles", "write:roles", "audit:read"];
const steps = ["App Info", "Grant Types", "Redirect URIs", "Scopes", "Review"];
export default function ClientOnboardingPage() {

  const t = useTranslations();
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [data, setData] = useState<any[]>([]);

  useEffect(() => {
    const fetchData = async () => {
      try {
        const res = await fetch("/api/v1/oauth/clients", {
          method: "GET",
          headers: {
            "Content-Type": "application/json",
            "X-Tenant-ID": "00000000-0000-0000-0000-000000000001",
          },
        });
        if (!res.ok) throw new Error(`HTTP ${res.status}`);
        const json = await res.json();
        setData(Array.isArray(json) ? json : [json]);
      } catch (e) {
        setError(e instanceof Error ? e.message : "Failed to load");
      } finally {
        setLoading(false);
      }
    };
    fetchData();
  }, []);

  if (loading) return <div className="p-8">Loading...</div>;
  if (error) return <div className="p-8 text-red-500">Error: {error}</div>;
  if (!data || data.length === 0) return <div className="p-8 text-gray-500">{t("backend.clientOnboarding.noData")}</div>;
  const [step, setStep] = useState(0);
  const [form, setForm] = useState({ name: "", description: "", grants: ["authorization_code"], redirects: [""], scopes: ["openid", "profile"] });
  const [creds, setCreds] = useState<{ client_id: string; client_secret: string } | null>(null);
  const [copied, setCopied] = useState("");
  const toggle = (arr: string[], val: string, key: "grants" | "scopes") => setForm({ ...form, [key]: arr.includes(val) ? arr.filter((x) => x !== val) : [...arr, val] });
  const submit = () => { setCreds({ client_id: "cli_" + Math.random().toString(36).substring(2, 12), client_secret: "sec_" + Math.random().toString(36).substring(2, 20) }); };
  const copy = (val: string, label: string) => { navigator.clipboard.writeText(val); setCopied(label); setTimeout(() => setCopied(""), 2000); };
  return (
    <div className="space-y-6">
      <div><h1 className="text-2xl font-bold flex items-center gap-2"><Rocket className="w-6 h-6 text-blue-500" /> {t("backend.clientOnboarding.title")}</h1><p className="text-sm text-gray-500 mt-1">Register a new OAuth/OIDC client with a step-by-step wizard.</p></div>
      <div className="flex items-center gap-2">{steps.map((s, i) => (<div key={s} className="flex items-center gap-2"><div className={"w-7 h-7 rounded-full flex items-center justify-center text-xs font-bold " + (i < step ? "bg-green-500 text-white" : i === step ? "bg-blue-600 text-white" : "bg-gray-200 dark:bg-gray-800")}>{i < step ? <Check className="w-4 h-4" /> : i + 1}</div><span className={"text-xs " + (i === step ? "font-bold" : "text-gray-400")}>{s}</span>{i < steps.length - 1 && <ChevronRight className="w-3 h-3 text-gray-300" />}</div>))}</div>
      {step === 0 && <div className="rounded-lg border dark:border-gray-800 p-4 space-y-3 max-w-md"><div><label className="text-sm font-medium">{t("backend.clientOnboarding.appName")}</label><input type="text" value={form.name} onChange={(e) => setForm({ ...form, name: e.target.value })} className="w-full mt-1 px-3 py-2 rounded-lg border dark:border-gray-700 dark:bg-gray-900 text-sm" /></div><div><label className="text-sm font-medium">{t("backend.clientOnboarding.description")}</label><input type="text" value={form.description} onChange={(e) => setForm({ ...form, description: e.target.value })} className="w-full mt-1 px-3 py-2 rounded-lg border dark:border-gray-700 dark:bg-gray-900 text-sm" /></div></div>}
      {step === 1 && <div className="rounded-lg border dark:border-gray-800 p-4"><div className="grid grid-cols-2 gap-2">{grantTypes.map((g) => <label key={g} className="flex items-center gap-2 cursor-pointer"><input type="checkbox" checked={form.grants.includes(g)} onChange={() => toggle(form.grants, g, "grants")} className="rounded" /><span className="text-sm font-mono">{g}</span></label>)}</div></div>}
      {step === 2 && <div className="rounded-lg border dark:border-gray-800 p-4 space-y-2 max-w-md"><div className="text-sm text-gray-500">{t("backend.clientOnboarding.redirectUris")}</div>{form.redirects.map((r, i) => (<div key={i} className="flex gap-2"><input type="text" value={r} onChange={(e) => { const rd = [...form.redirects]; rd[i] = e.target.value; setForm({ ...form, redirects: rd }); }} placeholder="https://app.example.com/callback" className="flex-1 px-3 py-2 rounded-lg border dark:border-gray-700 dark:bg-gray-900 text-sm font-mono" /><button onClick={() => setForm({ ...form, redirects: form.redirects.filter((_, x) => x !== i) })} className="text-red-500">\u2715</button></div>))}<button onClick={() => setForm({ ...form, redirects: [...form.redirects, ""] })} className="text-sm text-blue-600">+ Add URI</button></div>}
      {step === 3 && <div className="rounded-lg border dark:border-gray-800 p-4"><div className="grid grid-cols-2 gap-2">{allScopes.map((s) => <label key={s} className="flex items-center gap-2 cursor-pointer"><input type="checkbox" checked={form.scopes.includes(s)} onChange={() => toggle(form.scopes, s, "scopes")} className="rounded" /><span className="text-sm font-mono">{s}</span></label>)}</div></div>}
      {step === 4 && <div className="rounded-lg border dark:border-gray-800 p-4 space-y-3"><div className="text-sm"><span className="text-gray-500">Name:</span> {form.name}</div><div className="text-sm"><span className="text-gray-500">Grants:</span> {form.grants.join(", ")}</div><div className="text-sm"><span className="text-gray-500">Scopes:</span> {form.scopes.join(", ")}</div><div className="text-sm"><span className="text-gray-500">Redirects:</span> {form.redirects.filter(Boolean).join(", ")}</div>{!creds ? <button onClick={submit} className="px-4 py-2 rounded-lg bg-green-600 text-white text-sm font-medium">{t("backend.clientOnboarding.generateCredentials")}</button> : <div className="rounded-lg border dark:border-gray-800 p-4 space-y-2"><div className="flex items-center gap-2"><span className="text-xs text-gray-500 w-24">Client ID:</span><span className="font-mono text-xs flex-1">{creds.client_id}</span><button onClick={() => copy(creds.client_id, "id")}>{copied === "id" ? <Check className="w-3 h-3 text-green-500" /> : <Copy className="w-3 h-3" />}</button></div><div className="flex items-center gap-2"><span className="text-xs text-gray-500 w-24">Client Secret:</span><span className="font-mono text-xs flex-1">{creds.client_secret}</span><button onClick={() => copy(creds.client_secret, "secret")}>{copied === "secret" ? <Check className="w-3 h-3 text-green-500" /> : <Copy className="w-3 h-3" />}</button></div></div>}</div>}
      <div className="flex justify-between"><button onClick={() => setStep(Math.max(0, step - 1))} disabled={step === 0} className="px-4 py-2 rounded-lg border dark:border-gray-700 text-sm disabled:opacity-30">{t("backend.clientOnboarding.back")}</button>{step < 4 ? <button onClick={() => setStep(step + 1)} disabled={(step === 0 && !form.name)} className="px-4 py-2 rounded-lg bg-blue-600 text-white text-sm font-medium disabled:opacity-30">{t("backend.clientOnboarding.next")}</button> : <span />}</div>
    </div>
  );
}
