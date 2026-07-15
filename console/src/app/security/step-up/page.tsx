"use client";

import { useState } from "react";
import { useApi } from "@/lib/api";
import {
  ShieldCheck, Send, AlertCircle, Loader2, X, Check, Clock, KeyRound,
} from "lucide-react";
import { useTranslations } from "@/lib/i18n";

interface Challenge {
  id: string;
  user_id: string;
  user_name: string;
  reason: string;
  status: "pending" | "completed" | "expired";
  created_at: string;
  expires_at: string;
  method: string;
}

export default function StepUpAuthPage() {
  const t = useTranslations();

  const { apiFetch } = useApi();
  const [challenges, setChallenges] = useState<Challenge[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [showTrigger, setShowTrigger] = useState(false);
  const [form, setForm] = useState({ user_id: "", reason: "", method: "totp" });
  const [triggering, setTriggering] = useState(false);

  useState(() => {
    (async () => {
      try {
        const data = await apiFetch<{ challenges?: Challenge[]; items?: Challenge[] }>("/api/v1/auth/step-up/challenges").catch(() => null);
        setChallenges(data?.challenges ?? data?.items ?? []);
      } catch { setError("Failed to load challenges"); }
      finally { setLoading(false); }
    })();
  });

  const handleTrigger = async () => {
    if (!form.user_id.trim()) return;
    setTriggering(true);
    try {
      await apiFetch("/api/v1/auth/step-up/trigger", { method: "POST", body: JSON.stringify(form) });
      setForm({ user_id: "", reason: "", method: "totp" }); setShowTrigger(false);
      const data = await apiFetch<{ challenges?: Challenge[]; items?: Challenge[] }>("/api/v1/auth/step-up/challenges").catch(() => null);
      setChallenges(data?.challenges ?? data?.items ?? []);
    } catch { setError("Failed to trigger challenge"); }
    finally { setTriggering(false); }
  };

  const cardCls = "rounded-xl border border-gray-200 bg-white p-5 shadow-sm dark:border-gray-700 dark:bg-gray-800";
  const pending = challenges.filter((c) => c.status === "pending");

  return (
    <div className="space-y-6">
      <div className="flex items-center justify-between">
        <div>
          <h1 className="flex items-center gap-2 text-2xl font-bold text-gray-900 dark:text-white"><ShieldCheck className="h-6 w-6 text-indigo-600" /> {t("securityStepUp.title")}</h1>
          <p className="mt-1 text-sm text-gray-500 dark:text-gray-400">Additional verification challenges for sensitive operations.</p>
        </div>
        <button onClick={() => setShowTrigger(true)} className="flex items-center gap-2 rounded-lg bg-indigo-600 px-4 py-2 text-sm font-medium text-white hover:bg-indigo-700"><Send className="h-4 w-4" /> Trigger Challenge</button>
      </div>

      {error && <div className="flex items-center gap-2 rounded-lg bg-red-50 px-4 py-3 text-sm text-red-700 dark:bg-red-900/20 dark:text-red-400"><AlertCircle className="h-4 w-4 shrink-0" />{error}<button onClick={() => setError(null)} className="ml-auto"><X className="h-4 w-4" /></button></div>}

      {/* Summary */}
      <div className="grid grid-cols-3 gap-4">
        <div className={cardCls}><p className="text-xs font-semibold uppercase text-gray-400">Pending</p><p className="mt-1 text-2xl font-bold text-orange-600">{pending.length}</p></div>
        <div className={cardCls}><p className="text-xs font-semibold uppercase text-gray-400">Completed</p><p className="mt-1 text-2xl font-bold text-green-600">{challenges.filter((c) => c.status === "completed").length}</p></div>
        <div className={cardCls}><p className="text-xs font-semibold uppercase text-gray-400">Expired</p><p className="mt-1 text-2xl font-bold text-gray-500">{challenges.filter((c) => c.status === "expired").length}</p></div>
      </div>

      {loading ? <div className="flex justify-center py-12"><Loader2 className="h-8 w-8 animate-spin text-indigo-600" /></div>
      : challenges.length === 0 ? <div className={cardCls}><div className="py-12 text-center"><ShieldCheck className="mx-auto h-12 w-12 text-gray-300" /><p className="mt-4 text-sm text-gray-400">No step-up challenges.</p></div></div>
      : (
        <div className="space-y-3">
          {challenges.map((c) => (
            <div key={c.id} className={cardCls}>
              <div className="flex items-center justify-between">
                <div>
                  <div className="flex items-center gap-2">
                    <span className="font-medium text-gray-800 dark:text-gray-200">{c.user_name}</span>
                    <span className={`rounded-full px-2 py-0.5 text-xs font-medium ${c.status === "pending" ? "bg-orange-100 text-orange-700 dark:bg-orange-900/30 dark:text-orange-400" : c.status === "completed" ? "bg-green-100 text-green-700 dark:bg-green-900/30 dark:text-green-400" : "bg-gray-100 text-gray-500 dark:bg-gray-700"}`}>{c.status}</span>
                  </div>
                  <p className="mt-1 text-sm text-gray-400">{c.reason}</p>
                  <div className="mt-1 flex items-center gap-3 text-xs text-gray-400">
                    <span className="flex items-center gap-1"><KeyRound className="h-3 w-3" />{c.method}</span>
                    <span className="flex items-center gap-1"><Clock className="h-3 w-3" />Created {new Date(c.created_at).toLocaleString()}</span>
                    {c.status === "pending" && <span>Expires: {new Date(c.expires_at).toLocaleTimeString()}</span>}
                  </div>
                </div>
              </div>
            </div>
          ))}
        </div>
      )}

      {/* Trigger modal */}
      {showTrigger && (
        <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/50" onClick={() => !triggering && setShowTrigger(false)}>
          <div className="w-full max-w-md rounded-xl bg-white p-6 shadow-xl dark:bg-gray-800" onClick={(e) => e.stopPropagation()}>
            <div className="flex items-center justify-between"><h2 className="text-lg font-semibold text-gray-900 dark:text-white">Trigger Step-Up Challenge</h2><button onClick={() => setShowTrigger(false)}><X className="h-5 w-5 text-gray-400" /></button></div>
            <div className="mt-4 space-y-3">
              <div><label className="text-sm font-medium text-gray-700 dark:text-gray-300">User ID</label><input value={form.user_id} onChange={(e) => setForm((p) => ({ ...p, user_id: e.target.value }))} className="mt-1 w-full rounded-lg border border-gray-300 px-3 py-2 text-sm dark:border-gray-600 dark:bg-gray-700 dark:text-white" /></div>
              <div><label className="text-sm font-medium text-gray-700 dark:text-gray-300">Reason</label><input value={form.reason} onChange={(e) => setForm((p) => ({ ...p, reason: e.target.value }))} placeholder="Sensitive operation" className="mt-1 w-full rounded-lg border border-gray-300 px-3 py-2 text-sm dark:border-gray-600 dark:bg-gray-700 dark:text-white" /></div>
              <div><label className="text-sm font-medium text-gray-700 dark:text-gray-300">Method</label><select value={form.method} onChange={(e) => setForm((p) => ({ ...p, method: e.target.value }))} className="mt-1 w-full rounded-lg border border-gray-300 px-3 py-2 text-sm dark:border-gray-600 dark:bg-gray-700 dark:text-white"><option value="totp">TOTP</option><option value="webauthn">WebAuthn</option><option value="email">Email OTP</option></select></div>
            </div>
            <div className="mt-5 flex justify-end gap-2"><button onClick={() => setShowTrigger(false)} className="rounded-lg px-4 py-2 text-sm text-gray-500 hover:bg-gray-100 dark:hover:bg-gray-700">Cancel</button><button onClick={handleTrigger} disabled={!form.user_id.trim() || triggering} className="flex items-center gap-2 rounded-lg bg-indigo-600 px-4 py-2 text-sm font-medium text-white hover:bg-indigo-700 disabled:opacity-50">{triggering ? <Loader2 className="h-4 w-4 animate-spin" /> : <Send className="h-4 w-4" />}Trigger</button></div>
          </div>
        </div>
      )}
    </div>
  );
}
