"use client";
import { useTranslations } from "@/lib/i18n";

import { useState, useEffect, useCallback } from "react";
import { useApi } from "@/lib/api";
import {
  KeyRound, RotateCcw, AlertCircle, Loader2, X, Check, Clock,
  History,
} from "lucide-react";

interface JWKSStatus {
  active_kid: string;
  algorithm: string;
  created_at: string;
  previous_keys: { kid: string; retired_at: string }[];
  rotation_interval_hours: number;
  next_rotation: string;
  grace_period_hours: number;
}

export default function JwksPage() {
  const t = useTranslations();
  const { apiFetch } = useApi();
  const [status, setStatus] = useState<JWKSStatus | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [confirmRotate, setConfirmRotate] = useState(false);
  const [rotating, setRotating] = useState(false);

  const load = useCallback(async () => {
    setLoading(true);
    setError(null);
    try {
      const data = await apiFetch<JWKSStatus>("/api/v1/oauth/jwks/rotation-status").catch(() => null);
      if (data) setStatus(data);
    } catch {
      setError("Failed to load JWKS status");
    } finally {
      setLoading(false);
    }
  }, [apiFetch]);

  useEffect(() => { load(); }, [load]);

  const handleRotate = async () => {
    setRotating(true);
    try {
      await apiFetch("/api/v1/oauth/jwks/rotate", { method: "POST" });
      setConfirmRotate(false);
      await load();
    } catch {
      setError("Rotation failed");
    } finally {
      setRotating(false);
    }
  };

  const cardCls = "rounded-xl border border-gray-200 bg-white p-5 shadow-sm dark:border-gray-700 dark:bg-gray-800";

  return (
    <div className="space-y-6">
      <div className="flex items-center justify-between">
        <div>
          <h1 className="flex items-center gap-2 text-2xl font-bold text-gray-900 dark:text-white">
            <KeyRound className="h-6 w-6 text-indigo-600" /> {t("backend.jwks.title")}
          </h1>
          <p className="mt-1 text-sm text-gray-500 dark:text-gray-400">Signing key lifecycle with automatic rotation and grace periods.</p>
        </div>
        <button onClick={() => setConfirmRotate(true)} className="flex items-center gap-2 rounded-lg bg-indigo-600 px-4 py-2 text-sm font-medium text-white hover:bg-indigo-700"><RotateCcw className="h-4 w-4" /> Rotate Key</button>
      </div>

      {error && (
        <div className="flex items-center gap-2 rounded-lg bg-red-50 px-4 py-3 text-sm text-red-700 dark:bg-red-900/20 dark:text-red-400">
          <AlertCircle className="h-4 w-4 shrink-0" />{error}<button onClick={() => setError(null)} aria-label="Dismiss error" className="ml-auto"><X className="h-4 w-4" /></button>
        </div>
      )}

      {loading ? (
        <div className="flex justify-center py-12"><Loader2 className="h-8 w-8 animate-spin text-indigo-600" /></div>
      ) : status ? (
        <>
          {/* Active key */}
          <div className={`${cardCls} border-green-300 dark:border-green-700`}>
            <div className="flex items-center gap-3">
              <div className="rounded-lg bg-green-100 p-2 dark:bg-green-900/30"><Check className="h-5 w-5 text-green-600" /></div>
              <div className="flex-1">
                <h3 className="font-semibold text-gray-800 dark:text-gray-200">{t("backend.jwks.activeKey")}</h3>
                <div className="mt-1 flex flex-wrap items-center gap-4 text-sm">
                  <span className="font-mono text-gray-600 dark:text-gray-300">kid: {status.active_kid}</span>
                  <span className="rounded bg-indigo-100 px-2 py-0.5 text-xs text-indigo-700 dark:bg-indigo-900/30 dark:text-indigo-400">{status.algorithm}</span>
                  <span className="flex items-center gap-1 text-xs text-gray-400"><Clock className="h-3 w-3" />Created {new Date(status.created_at).toLocaleDateString()}</span>
                </div>
              </div>
            </div>
          </div>

          {/* Rotation config */}
          <div className="grid grid-cols-2 gap-4">
            <div className={cardCls}>
              <p className="text-xs font-semibold uppercase text-gray-400">{t("backend.jwks.rotationInterval")}</p>
              <p className="mt-1 text-2xl font-bold text-indigo-600">{status.rotation_interval_hours}h</p>
            </div>
            <div className={cardCls}>
              <p className="text-xs font-semibold uppercase text-gray-400">{t("backend.jwks.gracePeriod")}</p>
              <p className="mt-1 text-2xl font-bold text-indigo-600">{status.grace_period_hours}h</p>
            </div>
          </div>

          {/* Next rotation */}
          <div className={cardCls}>
            <p className="flex items-center gap-2 text-sm text-gray-500"><Clock className="h-4 w-4 text-indigo-500" />Next automatic rotation: <strong className="text-gray-700 dark:text-gray-300">{new Date(status.next_rotation).toLocaleString()}</strong></p>
          </div>

          {/* Previous keys history */}
          {status.previous_keys.length > 0 && (
            <div>
              <h2 className="mb-3 flex items-center gap-2 text-sm font-semibold uppercase text-gray-500"><History className="h-4 w-4" /> Rotation History</h2>
              <div className="hidden overflow-hidden rounded-xl border border-gray-200 shadow-sm md:block dark:border-gray-700">
                <table className="w-full text-sm">
                  <thead className="bg-gray-50 dark:bg-gray-800"><tr className="text-left text-xs font-semibold uppercase text-gray-500">
                    <th className="px-4 py-3">{t("backend.jwks.keyId")}</th><th className="px-4 py-3">{t("backend.jwks.retiredAt")}</th><th className="px-4 py-3">{t("backend.jwks.status")}</th>
                  </tr></thead>
                  <tbody className="divide-y divide-gray-100 dark:divide-gray-700">
                    {status.previous_keys.map((k) => (
                      <tr key={k.kid} className="hover:bg-gray-50 dark:hover:bg-gray-800/50">
                        <td className="px-4 py-3 font-mono text-xs text-gray-500">{k.kid}</td>
                        <td className="px-4 py-3 text-gray-400">{new Date(k.retired_at).toLocaleString()}</td>
                        <td className="px-4 py-3"><span className="rounded-full bg-gray-100 px-2 py-0.5 text-xs text-gray-500 dark:bg-gray-700">{t("backend.jwks.retired")}</span></td>
                      </tr>
                    ))}
                  </tbody>
                </table>
              </div>
              <div className="space-y-2 md:hidden">
                {status.previous_keys.map((k) => (
                  <div key={k.kid} className={`${cardCls} py-3`}>
                    <p className="font-mono text-xs text-gray-500">{k.kid}</p>
                    <p className="text-xs text-gray-400">Retired: {new Date(k.retired_at).toLocaleString()}</p>
                  </div>
                ))}
              </div>
            </div>
          )}
        </>
      ) : (
        <div className={cardCls}><div className="py-12 text-center"><KeyRound className="mx-auto h-12 w-12 text-gray-300" /><p className="mt-4 text-sm text-gray-400">No JWKS status available.</p></div></div>
      )}

      {/* Rotate confirmation */}
      {confirmRotate && (
        <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/50" onClick={() => !rotating && setConfirmRotate(false)}>
          <div role="dialog" aria-modal="true" className="w-full max-w-sm rounded-xl bg-white p-6 shadow-xl dark:bg-gray-800" onClick={(e) => e.stopPropagation()}>
            <div className="flex items-center gap-3">
              <div className="rounded-full bg-indigo-100 p-2 dark:bg-indigo-900/30"><RotateCcw className="h-5 w-5 text-indigo-600" /></div>
              <div>
                <h2 className="font-semibold text-gray-900 dark:text-white">Rotate Signing Key?</h2>
                <p className="text-sm text-gray-500">A new key pair will be generated. The old key remains valid during the grace period ({status?.grace_period_hours ?? 24}h).</p>
              </div>
            </div>
            <div className="mt-5 flex justify-end gap-2">
              <button onClick={() => setConfirmRotate(false)} disabled={rotating} className="rounded-lg px-4 py-2 text-sm text-gray-500 hover:bg-gray-100 dark:hover:bg-gray-700">{t("backend.jwks.cancel")}</button>
              <button onClick={handleRotate} disabled={rotating} className="flex items-center gap-2 rounded-lg bg-indigo-600 px-4 py-2 text-sm font-medium text-white hover:bg-indigo-700 disabled:opacity-50">{rotating ? <Loader2 className="h-4 w-4 animate-spin" /> : <RotateCcw className="h-4 w-4" />}Rotate</button>
            </div>
          </div>
        </div>
      )}
    </div>
  );
}
