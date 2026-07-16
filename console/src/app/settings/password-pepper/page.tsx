"use client";

import { useState, useCallback } from "react";
import { useApi } from "@/lib/api";
import { useTranslations } from "@/lib/i18n";
import {
  Key, RotateCcw, AlertTriangle, Loader2, X, Check, Shield,
  Clock,
} from "lucide-react";

interface PepperStatus {
  active: boolean;
  active_since: string;
  pending_removal: boolean;
  rehash_in_progress: boolean;
  rehash_progress: number;
  total_users: number;
  rehashed_users: number;
}

export default function PasswordPepperPage() {
  const t = useTranslations();
  const { apiFetch } = useApi();
  const [status, setStatus] = useState<PepperStatus | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [confirmRotate, setConfirmRotate] = useState(false);
  const [rotating, setRotating] = useState(false);

  const load = useCallback(async () => {
    setLoading(true);
    setError(null);
    try {
      const data = await apiFetch<PepperStatus>("/api/v1/auth/password-pepper/status").catch(() => null);
      if (data) setStatus(data);
    } catch {
      setError("Failed to load pepper status");
    } finally {
      setLoading(false);
    }
  }, [apiFetch]);

  const handleRotate = async () => {
    setRotating(true);
    try {
      await apiFetch("/api/v1/auth/password-pepper/rotate", { method: "POST" });
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
            <Key className="h-6 w-6 text-indigo-600" /> Password Pepper
          </h1>
          <p className="mt-1 text-sm text-gray-500 dark:text-gray-400">Server-side pepper for additional password hashing security.</p>
        </div>
        {status?.active && (
          <button onClick={() => setConfirmRotate(true)} className="flex items-center gap-2 rounded-lg bg-indigo-600 px-4 py-2 text-sm font-medium text-white hover:bg-indigo-700"><RotateCcw className="h-4 w-4" /> Rotate Pepper</button>
        )}
      </div>

      {/* Warning banner */}
      <div className="flex items-start gap-3 rounded-lg bg-amber-50 px-4 py-3 dark:bg-amber-900/20">
        <AlertTriangle className="mt-0.5 h-5 w-5 shrink-0 text-amber-600" />
        <div>
          <p className="text-sm font-medium text-amber-800 dark:text-amber-400">Rotation triggers background re-hashing of all user passwords.</p>
          <p className="mt-1 text-xs text-amber-600 dark:text-amber-500">During re-hashing, both old and new pepper are valid. Users won't experience login disruption.</p>
        </div>
      </div>

      {error && (
        <div role="alert" className="flex items-center gap-2 rounded-lg bg-red-50 px-4 py-3 text-sm text-red-700 dark:bg-red-900/20 dark:text-red-400">
          <AlertTriangle className="h-4 w-4 shrink-0" />{error}<button onClick={() => setError(null)} aria-label="Dismiss error" className="ml-auto"><X className="h-4 w-4" /></button>
        </div>
      )}

      {loading ? (
        <div className="flex justify-center py-12"><Loader2 className="h-8 w-8 animate-spin text-indigo-600" /></div>
      ) : status ? (
        <>
          {/* Status cards */}
          <div className="grid grid-cols-2 gap-4">
            <div className={`${cardCls} ${status.active ? "border-green-300 dark:border-green-700" : "border-red-300 dark:border-red-700"}`}>
              <div className="flex items-center gap-2">
                {status.active ? <Shield className="h-5 w-5 text-green-600" /> : <Shield className="h-5 w-5 text-red-600" />}
                <span className="text-xs font-semibold uppercase text-gray-400">Status</span>
              </div>
              <p className={`mt-2 text-lg font-bold ${status.active ? "text-green-600" : "text-red-600"}`}>{status.active ? "Active" : "Inactive"}</p>
              {status.active_since && <p className="mt-1 flex items-center gap-1 text-xs text-gray-400"><Clock className="h-3 w-3" />Since {new Date(status.active_since).toLocaleDateString()}</p>}
            </div>
            <div className={cardCls}>
              <div className="flex items-center gap-2"><Key className="h-5 w-5 text-indigo-500" /><span className="text-xs font-semibold uppercase text-gray-400">Pending Removal</span></div>
              <p className="mt-2 text-lg font-bold text-gray-600 dark:text-gray-300">{status.pending_removal ? "Yes" : "No"}</p>
              <p className="mt-1 text-xs text-gray-400">Old pepper still validating remaining hashes</p>
            </div>
          </div>

          {/* Re-hash progress */}
          {status.rehash_in_progress && (
            <div className={cardCls}>
              <h3 className="mb-3 flex items-center gap-2 text-sm font-semibold text-gray-700 dark:text-gray-300"><Loader2 className="h-4 w-4 animate-spin text-indigo-600" /> Re-Hashing In Progress</h3>
              <div className="flex items-center justify-between text-sm">
                <span className="text-gray-500">{status.rehashed_users} / {status.total_users} users re-hashed</span>
                <span className="font-bold text-indigo-600">{status.rehash_progress}%</span>
              </div>
              <div className="mt-2 h-3 overflow-hidden rounded-full bg-gray-200 dark:bg-gray-700">
                <div className="h-full rounded-full bg-indigo-500 transition-all" style={{ width: `${status.rehash_progress}%` }} />
              </div>
            </div>
          )}

          {/* Summary */}
          <div className={cardCls}>
            <div className="grid grid-cols-3 gap-4 text-center">
              <div><p className="text-xs font-semibold uppercase text-gray-400">Total Users</p><p className="mt-1 text-xl font-bold text-indigo-600">{status.total_users}</p></div>
              <div><p className="text-xs font-semibold uppercase text-gray-400">Re-Hashed</p><p className="mt-1 text-xl font-bold text-green-600">{status.rehashed_users}</p></div>
              <div><p className="text-xs font-semibold uppercase text-gray-400">Remaining</p><p className="mt-1 text-xl font-bold text-orange-600">{status.total_users - status.rehashed_users}</p></div>
            </div>
          </div>
        </>
      ) : (
        <div className={cardCls}><div className="py-12 text-center"><Key className="mx-auto h-12 w-12 text-gray-300" /><p className="mt-4 text-sm text-gray-400">No pepper status available.</p></div></div>
      )}

      {/* Rotate confirmation */}
      {confirmRotate && (
        <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/50" onClick={() => !rotating && setConfirmRotate(false)}>
          <div role="dialog" aria-modal="true" className="w-full max-w-md rounded-xl bg-white p-6 shadow-xl dark:bg-gray-800" onClick={(e) => e.stopPropagation()}>
            <div className="flex items-center gap-3">
              <div className="rounded-full bg-amber-100 p-2 dark:bg-amber-900/30"><AlertTriangle className="h-5 w-5 text-amber-600" /></div>
              <div>
                <h2 className="font-semibold text-gray-900 dark:text-white">Rotate Pepper?</h2>
                <p className="text-sm text-gray-500">This generates a new secret and begins background re-hashing of <strong>{status?.total_users ?? "all"}</strong> user passwords. The old pepper remains valid until re-hashing completes.</p>
              </div>
            </div>
            <div className="mt-4 rounded-lg bg-gray-50 p-3 text-xs text-gray-400 dark:bg-gray-900/30">
              <p className="flex items-center gap-1"><Check className="h-3 w-3 text-green-500" />No login disruption for users</p>
              <p className="mt-1 flex items-center gap-1"><Check className="h-3 w-3 text-green-500" />Old hashes validated with old pepper during transition</p>
              <p className="mt-1 flex items-center gap-1"><Check className="h-3 w-3 text-green-500" />Automatic cleanup after re-hash completes</p>
            </div>
            <div className="mt-5 flex justify-end gap-2">
              <button onClick={() => setConfirmRotate(false)} disabled={rotating} className="rounded-lg px-4 py-2 text-sm text-gray-500 hover:bg-gray-100 dark:hover:bg-gray-700">Cancel</button>
              <button onClick={handleRotate} disabled={rotating} className="flex items-center gap-2 rounded-lg bg-indigo-600 px-4 py-2 text-sm font-medium text-white hover:bg-indigo-700 disabled:opacity-50">{rotating ? <Loader2 className="h-4 w-4 animate-spin" /> : <RotateCcw className="h-4 w-4" />}Rotate</button>
            </div>
          </div>
        </div>
      )}
    </div>
  );
}
