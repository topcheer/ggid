"use client";

import { useState, useEffect, useCallback } from "react";
import { useApi } from "@/lib/api";
import { useTranslations } from "@/lib/i18n";
import {
  GitBranch, RotateCcw, Plus, Trash2, X, AlertCircle, Loader2,
  Check, Clock, ArrowRight, Shield,
} from "lucide-react";

interface PolicyVersion {
  id: string;
  version: number;
  description: string;
  author: string;
  created_at: string;
  is_active: boolean;
  change_count: number;
  hash: string;
}

export default function PolicyVersionsPage() {
  const t = useTranslations();
  const { apiFetch } = useApi();
  const [versions, setVersions] = useState<PolicyVersion[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [confirmRollback, setConfirmRollback] = useState<PolicyVersion | null>(null);
  const [rolling, setRolling] = useState(false);
  const [selectedCompare, setSelectedCompare] = useState<{ a: string; b: string }>({ a: "", b: "" });

  const load = useCallback(async () => {
    setLoading(true);
    setError(null);
    try {
      const data = await apiFetch<{ versions?: PolicyVersion[]; items?: PolicyVersion[] }>("/api/v1/policy/versions").catch(() => null);
      setVersions(data?.versions ?? data?.items ?? []);
    } catch {
      setError("Failed to load policy versions");
    } finally {
      setLoading(false);
    }
  }, [apiFetch]);

  useEffect(() => { load(); }, [load]);

  const handleRollback = async (v: PolicyVersion) => {
    setRolling(true);
    try {
      await apiFetch(`/api/v1/policy/versions/${v.id}/rollback`, { method: "POST" });
      setConfirmRollback(null);
      await load();
    } catch {
      setError("Failed to rollback");
    } finally {
      setRolling(false);
    }
  };

  const activeVersion = versions.find((v) => v.is_active);
  const cardCls = "rounded-xl border border-gray-200 bg-white p-5 shadow-sm dark:border-gray-700 dark:bg-gray-800";

  return (
    <div className="space-y-6">
      <div>
        <h1 className="flex items-center gap-2 text-2xl font-bold text-gray-900 dark:text-white">
          <GitBranch className="h-6 w-6 text-indigo-600" /> Policy Versions
        </h1>
        <p className="mt-1 text-sm text-gray-500 dark:text-gray-400">Version history with rollback capability for policy configurations.</p>
      </div>

      {error && (
        <div className="flex items-center gap-2 rounded-lg bg-red-50 px-4 py-3 text-sm text-red-700 dark:bg-red-900/20 dark:text-red-400">
          <AlertCircle className="h-4 w-4 shrink-0" />{error}<button onClick={() => setError(null)} aria-label="Dismiss error" className="ml-auto"><X className="h-4 w-4" /></button>
        </div>
      )}

      {/* Active version summary */}
      {activeVersion && (
        <div className={`${cardCls} border-green-300 dark:border-green-700`}>
          <div className="flex items-center gap-3">
            <div className="rounded-lg bg-green-100 p-2 dark:bg-green-900/30"><Shield className="h-5 w-5 text-green-600" /></div>
            <div>
              <h3 className="font-semibold text-gray-800 dark:text-gray-200">Active: Version {activeVersion.version}</h3>
              <p className="text-sm text-gray-400">{activeVersion.description}</p>
            </div>
            <div className="ml-auto text-right">
              <span className="font-mono text-xs text-gray-400">{activeVersion.hash.substring(0, 16)}</span>
              <p className="text-xs text-gray-400">{activeVersion.change_count} changes</p>
            </div>
          </div>
        </div>
      )}

      {/* Timeline */}
      {loading ? (
        <div className="flex justify-center py-12"><Loader2 className="h-8 w-8 animate-spin text-indigo-600" /></div>
      ) : versions.length === 0 ? (
        <div className={cardCls}><div className="py-12 text-center"><GitBranch className="mx-auto h-12 w-12 text-gray-300" /><p className="mt-4 text-sm text-gray-400">No policy versions recorded.</p></div></div>
      ) : (
        <div className="relative space-y-3">
          {/* Vertical line */}
          <div className="absolute bottom-0 left-4 top-2 w-0.5 bg-gray-200 dark:bg-gray-700" />
          {versions.map((v) => (
            <div key={v.id} className="relative flex items-start gap-4 pl-0">
              {/* Dot */}
              <div className={`relative z-10 mt-3 h-8 w-8 shrink-0 rounded-full border-2 ${v.is_active ? "border-green-500 bg-green-100 dark:bg-green-900/30" : "border-gray-300 bg-white dark:border-gray-600 dark:bg-gray-800"}`}>
                {v.is_active && <Check className="m-auto mt-1 h-4 w-4 text-green-600" />}
              </div>
              {/* Card */}
              <div className={`${cardCls} flex-1`}>
                <div className="flex items-start justify-between">
                  <div>
                    <div className="flex items-center gap-2">
                      <span className="font-semibold text-gray-800 dark:text-gray-200">Version {v.version}</span>
                      {v.is_active && <span className="rounded-full bg-green-100 px-2 py-0.5 text-xs font-medium text-green-700 dark:bg-green-900/30 dark:text-green-400">Active</span>}
                    </div>
                    <p className="mt-1 text-sm text-gray-500 dark:text-gray-400">{v.description}</p>
                    <div className="mt-2 flex items-center gap-3 text-xs text-gray-400">
                      <span className="flex items-center gap-1"><Clock className="h-3 w-3" />{new Date(v.created_at).toLocaleString()}</span>
                      <span>By {v.author}</span>
                      <span>{v.change_count} changes</span>
                    </div>
                  </div>
                  {!v.is_active && (
                    <button onClick={() => setConfirmRollback(v)} className="flex items-center gap-1.5 rounded-lg border border-orange-200 px-3 py-1.5 text-xs font-medium text-orange-600 hover:bg-orange-50 dark:border-orange-800 dark:hover:bg-orange-900/20">
                      <RotateCcw className="h-3.5 w-3.5" />Rollback
                    </button>
                  )}
                </div>
              </div>
            </div>
          ))}
        </div>
      )}

      {/* Rollback confirmation */}
      {confirmRollback && (
        <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/50" onClick={() => !rolling && setConfirmRollback(null)}>
          <div className="w-full max-w-sm rounded-xl bg-white p-6 shadow-xl dark:bg-gray-800" onClick={(e) => e.stopPropagation()}>
            <div className="flex items-center gap-3">
              <div className="rounded-full bg-orange-100 p-2 dark:bg-orange-900/30"><RotateCcw className="h-5 w-5 text-orange-600" /></div>
              <div>
                <h2 className="font-semibold text-gray-900 dark:text-white">Rollback to Version {confirmRollback.version}?</h2>
                <p className="text-sm text-gray-500">Current active policy will be archived. All policy rules revert to this snapshot.</p>
              </div>
            </div>
            <div className="mt-5 flex justify-end gap-2">
              <button onClick={() => setConfirmRollback(null)} disabled={rolling} className="rounded-lg px-4 py-2 text-sm text-gray-500 hover:bg-gray-100 dark:hover:bg-gray-700">Cancel</button>
              <button onClick={() => handleRollback(confirmRollback)} disabled={rolling} className="flex items-center gap-2 rounded-lg bg-orange-600 px-4 py-2 text-sm font-medium text-white hover:bg-orange-700 disabled:opacity-50">
                {rolling ? <Loader2 className="h-4 w-4 animate-spin" /> : <RotateCcw className="h-4 w-4" />}Rollback
              </button>
            </div>
          </div>
        </div>
      )}
    </div>
  );
}
