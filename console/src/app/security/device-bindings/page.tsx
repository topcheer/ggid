"use client";

import { useState, useEffect, useCallback } from "react";
import { useApi } from "@/lib/api";
import {
  Smartphone, Trash2, X, AlertCircle, Loader2, Shield,
  Fingerprint, Monitor, Tablet,
} from "lucide-react";
import { useTranslations } from "@/lib/i18n";

interface DeviceBinding {
  id: string;
  user_id: string;
  user_name: string;
  device_name: string;
  device_type: "mobile" | "desktop" | "tablet" | "other";
  fingerprint: string;
  bound_at: string;
  last_seen?: string;
  status: "active" | "revoked";
}

const TYPE_ICON = {
  mobile: Smartphone,
  desktop: Monitor,
  tablet: Tablet,
  other: Shield,
};

const maskFingerprint = (fp: string) => {
  const t = useTranslations();

  if (fp.length <= 8) return fp;
  return `${fp.substring(0, 6)}••••${fp.substring(fp.length - 4)}`;
};

export default function DeviceBindingsPage() {
  const { apiFetch } = useApi();
  const [bindings, setBindings] = useState<DeviceBinding[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [confirmUnbind, setConfirmUnbind] = useState<DeviceBinding | null>(null);
  const [unbinding, setUnbinding] = useState<string | null>(null);

  const load = useCallback(async () => {
    setLoading(true);
    setError(null);
    try {
      const data = await apiFetch<{ bindings?: DeviceBinding[]; items?: DeviceBinding[] }>("/api/v1/security/device-bindings").catch(() => null);
      setBindings(data?.bindings ?? data?.items ?? []);
    } catch {
      setError("Failed to load device bindings");
    } finally {
      setLoading(false);
    }
  }, [apiFetch]);

  useEffect(() => { load(); }, [load]);

  const handleUnbind = async (id: string) => {
    setUnbinding(id);
    try {
      await apiFetch(`/api/v1/security/device-bindings/${id}`, { method: "DELETE" });
      setConfirmUnbind(null);
      setBindings((prev) => prev.filter((b) => b.id !== id));
    } catch {
      setError("Failed to unbind device");
    } finally {
      setUnbinding(null);
    }
  };

  const active = bindings.filter((b) => b.status === "active");
  const cardCls = "rounded-xl border border-gray-200 bg-white p-5 shadow-sm dark:border-gray-700 dark:bg-gray-800";

  return (
    <div className="space-y-6">
      <div>
        <h1 className="flex items-center gap-2 text-2xl font-bold text-gray-900 dark:text-white">
          <Smartphone className="h-6 w-6 text-indigo-600" /> Device Bindings
        </h1>
        <p className="mt-1 text-sm text-gray-500 dark:text-gray-400">Manage trusted devices bound to user accounts for enhanced authentication.</p>
      </div>

      {/* Summary */}
      <div className="grid grid-cols-3 gap-4">
        <div className={cardCls}><p className="text-xs font-semibold uppercase text-gray-400">Total Devices</p><p className="mt-1 text-2xl font-bold text-indigo-600">{bindings.length}</p></div>
        <div className={cardCls}><p className="text-xs font-semibold uppercase text-gray-400">Active</p><p className="mt-1 text-2xl font-bold text-green-600">{active.length}</p></div>
        <div className={cardCls}><p className="text-xs font-semibold uppercase text-gray-400">Device Types</p><p className="mt-1 text-2xl font-bold text-indigo-600">{new Set(bindings.map((b) => b.device_type)).size}</p></div>
      </div>

      {error && (
        <div className="flex items-center gap-2 rounded-lg bg-red-50 px-4 py-3 text-sm text-red-700 dark:bg-red-900/20 dark:text-red-400">
          <AlertCircle className="h-4 w-4 shrink-0" />{error}<button onClick={() => setError(null)} className="ml-auto"><X className="h-4 w-4" /></button>
        </div>
      )}

      {loading ? (
        <div className="flex justify-center py-12"><Loader2 className="h-8 w-8 animate-spin text-indigo-600" /></div>
      ) : bindings.length === 0 ? (
        <div className={cardCls}><div className="py-12 text-center"><Smartphone className="mx-auto h-12 w-12 text-gray-300" /><p className="mt-4 text-sm text-gray-400">No device bindings.</p></div></div>
      ) : (
        <div className="space-y-3">
          {bindings.map((b) => {
            const Icon = TYPE_ICON[b.device_type] ?? Shield;
            return (
              <div key={b.id} className={cardCls}>
                <div className="flex items-center justify-between">
                  <div className="flex items-center gap-3">
                    <div className="rounded-lg bg-gray-100 p-2 dark:bg-gray-700"><Icon className="h-5 w-5 text-gray-500" /></div>
                    <div>
                      <div className="flex items-center gap-2">
                        <span className="font-medium text-gray-800 dark:text-gray-200">{b.device_name}</span>
                        <span className={`rounded-full px-2 py-0.5 text-xs font-medium ${b.status === "active" ? "bg-green-100 text-green-700 dark:bg-green-900/30 dark:text-green-400" : "bg-gray-100 text-gray-500"}`}>{b.status}</span>
                      </div>
                      <p className="text-sm text-gray-500 dark:text-gray-400">{b.user_name}</p>
                      <div className="mt-1 flex items-center gap-3 text-xs text-gray-400">
                        <span className="flex items-center gap-1 font-mono"><Fingerprint className="h-3 w-3" />{maskFingerprint(b.fingerprint)}</span>
                        <span>Bound: {new Date(b.bound_at).toLocaleDateString()}</span>
                        {b.last_seen && <span>Seen: {new Date(b.last_seen).toLocaleDateString()}</span>}
                      </div>
                    </div>
                  </div>
                  {b.status === "active" && (
                    <button onClick={() => setConfirmUnbind(b)} className="flex items-center gap-1.5 rounded-lg border border-red-200 px-3 py-1.5 text-xs font-medium text-red-500 hover:bg-red-50 dark:border-red-800 dark:hover:bg-red-900/20"><Trash2 className="h-3.5 w-3.5" /> Unbind</button>
                  )}
                </div>
              </div>
            );
          })}
        </div>
      )}

      {/* Unbind confirmation */}
      {confirmUnbind && (
        <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/50" onClick={() => !unbinding && setConfirmUnbind(null)}>
          <div className="w-full max-w-sm rounded-xl bg-white p-6 shadow-xl dark:bg-gray-800" onClick={(e) => e.stopPropagation()}>
            <div className="flex items-center gap-3">
              <div className="rounded-full bg-red-100 p-2 dark:bg-red-900/30"><Trash2 className="h-5 w-5 text-red-600" /></div>
              <div>
                <h2 className="font-semibold text-gray-900 dark:text-white">Unbind Device?</h2>
                <p className="text-sm text-gray-500">Device <strong>{confirmUnbind.device_name}</strong> bound to <strong>{confirmUnbind.user_name}</strong> will require re-authentication.</p>
              </div>
            </div>
            <div className="mt-5 flex justify-end gap-2">
              <button onClick={() => setConfirmUnbind(null)} disabled={!!unbinding} className="rounded-lg px-4 py-2 text-sm text-gray-500 hover:bg-gray-100 dark:hover:bg-gray-700">Cancel</button>
              <button onClick={() => handleUnbind(confirmUnbind.id)} disabled={!!unbinding} className="flex items-center gap-2 rounded-lg bg-red-600 px-4 py-2 text-sm font-medium text-white hover:bg-red-700 disabled:opacity-50">{unbinding ? <Loader2 className="h-4 w-4 animate-spin" /> : null}Unbind</button>
            </div>
          </div>
        </div>
      )}
    </div>
  );
}
