"use client";

import { useState, useCallback } from "react";
import { useApi } from "@/lib/api";
import {
  UserX, Search, Check, X, AlertCircle, Loader2, ChevronRight,
  History, ArrowRight, ClipboardList, Shield,
} from "lucide-react";

interface DeprovisionAction {
  name: string;
  status: "success" | "failed";
  message: string;
}

interface DeprovisionResult {
  user_id: string;
  status: "completed" | "partial" | "failed";
  actions: DeprovisionAction[];
  completed_at: string;
}

interface HistoryEntry {
  id: string;
  user_id: string;
  user_name: string;
  initiated_by: string;
  status: string;
  actions_summary: { total: number; success: number; failed: number };
  completed_at: string;
}

const CHECKLIST = [
  { key: "revoke_tokens", label: "Revoke all tokens", default: true },
  { key: "disable_account", label: "Disable account", default: true },
  { key: "remove_sessions", label: "Terminate active sessions", default: true },
  { key: "transfer_data", label: "Transfer data ownership", default: false },
];

export default function DeprovisioningPage() {
  const { apiFetch } = useApi();
  const [step, setStep] = useState<"search" | "confirm" | "result">("search");
  const [searchQuery, setSearchQuery] = useState("");
  const [searchResults, setSearchResults] = useState<{ id: string; name: string; email: string }[]>([]);
  const [selectedUser, setSelectedUser] = useState<{ id: string; name: string; email: string } | null>(null);
  const [checklist, setChecklist] = useState<Record<string, boolean>>(
    Object.fromEntries(CHECKLIST.map((c) => [c.key, c.default]))
  );
  const [reason, setReason] = useState("");
  const [transferTarget, setTransferTarget] = useState("");
  const [executing, setExecuting] = useState(false);
  const [result, setResult] = useState<DeprovisionResult | null>(null);
  const [history, setHistory] = useState<HistoryEntry[]>([]);
  const [error, setError] = useState<string | null>(null);

  const handleSearch = useCallback(async () => {
    if (!searchQuery.trim()) return;
    try {
      const data = await apiFetch<{ users?: { id: string; name: string; email: string }[]; items?: typeof searchResults }>(
        `/api/v1/users?search=${encodeURIComponent(searchQuery)}&limit=10`
      ).catch(() => null);
      setSearchResults(data?.users ?? data?.items ?? []);
    } catch {
      setError("Search failed");
    }
  }, [apiFetch, searchQuery]);

  const loadHistory = useCallback(async () => {
    try {
      const data = await apiFetch<{ entries?: HistoryEntry[]; items?: HistoryEntry[] }>(
        "/api/v1/users/deprovision/history?limit=10"
      ).catch(() => null);
      setHistory(data?.entries ?? data?.items ?? []);
    } catch {
      /* noop */
    }
  }, [apiFetch]);

  const handleExecute = async () => {
    if (!selectedUser) return;
    setExecuting(true);
    setError(null);
    try {
      const data = await apiFetch<DeprovisionResult>("/api/v1/users/deprovision", {
        method: "POST",
        body: JSON.stringify({
          user_id: selectedUser.id,
          ...checklist,
          transfer_data: checklist.transfer_data ? transferTarget : undefined,
          reason,
        }),
      });
      setResult(data);
      setStep("result");
      await loadHistory();
    } catch {
      setError("Deprovisioning failed");
    } finally {
      setExecuting(false);
    }
  };

  const reset = () => {
    setStep("search");
    setSelectedUser(null);
    setResult(null);
    setReason("");
    setTransferTarget("");
    setSearchQuery("");
    setSearchResults([]);
  };

  const cardCls = "rounded-xl border border-gray-200 bg-white p-5 shadow-sm dark:border-gray-700 dark:bg-gray-800";

  return (
    <div className="space-y-6">
      <div>
        <h1 className="flex items-center gap-2 text-2xl font-bold text-gray-900 dark:text-white">
          <UserX className="h-6 w-6 text-indigo-600" /> User Deprovisioning
        </h1>
        <p className="mt-1 text-sm text-gray-500 dark:text-gray-400">Securely offboard users with a guided checklist workflow.</p>
      </div>

      {error && (
        <div className="flex items-center gap-2 rounded-lg bg-red-50 px-4 py-3 text-sm text-red-700 dark:bg-red-900/20 dark:text-red-400">
          <AlertCircle className="h-4 w-4 shrink-0" />{error}<button onClick={() => setError(null)} className="ml-auto"><X className="h-4 w-4" /></button>
        </div>
      )}

      {/* Stepper */}
      <div className="flex items-center gap-2 text-sm">
        {[
          { key: "search", label: "Find User", icon: Search },
          { key: "confirm", label: "Confirm Actions", icon: ClipboardList },
          { key: "result", label: "Result", icon: Check },
        ].map((s, i) => {
          const Icon = s.icon;
          const active = step === s.key;
          const done = ["search", "confirm", "result"].indexOf(step) > i;
          return (
            <div key={s.key} className="flex items-center gap-2">
              <div className={`flex items-center gap-1.5 rounded-lg px-3 py-1.5 ${active ? "bg-indigo-600 text-white" : done ? "bg-green-100 text-green-700 dark:bg-green-900/30 dark:text-green-400" : "bg-gray-100 text-gray-400 dark:bg-gray-700"}`}>
                <Icon className="h-3.5 w-3.5" /><span className="font-medium">{s.label}</span>
              </div>
              {i < 2 && <ChevronRight className="h-4 w-4 text-gray-300" />}
            </div>
          );
        })}
      </div>

      {/* Step 1: Search */}
      {step === "search" && (
        <div className={cardCls}>
          <div className="flex gap-2">
            <input value={searchQuery} onChange={(e) => setSearchQuery(e.target.value)} onKeyDown={(e) => e.key === "Enter" && handleSearch()} placeholder="Search by name or email..." className="flex-1 rounded-lg border border-gray-300 px-4 py-2 text-sm dark:border-gray-600 dark:bg-gray-700 dark:text-white" />
            <button onClick={handleSearch} className="flex items-center gap-2 rounded-lg bg-indigo-600 px-4 py-2 text-sm font-medium text-white hover:bg-indigo-700"><Search className="h-4 w-4" />Search</button>
          </div>
          {searchResults.length > 0 && (
            <div className="mt-4 space-y-2">
              {searchResults.map((u) => (
                <button key={u.id} onClick={() => { setSelectedUser(u); setStep("confirm"); }} className="flex w-full items-center justify-between rounded-lg border border-gray-200 p-3 text-left hover:border-indigo-300 hover:bg-indigo-50/50 dark:border-gray-700 dark:hover:bg-indigo-900/10">
                  <div>
                    <p className="font-medium text-gray-800 dark:text-gray-200">{u.name}</p>
                    <p className="text-xs text-gray-400">{u.email}</p>
                  </div>
                  <ArrowRight className="h-4 w-4 text-gray-400" />
                </button>
              ))}
            </div>
          )}
        </div>
      )}

      {/* Step 2: Confirm */}
      {step === "confirm" && selectedUser && (
        <div className={cardCls}>
          <div className="mb-4 flex items-center gap-3 rounded-lg bg-gray-50 p-3 dark:bg-gray-900/30">
            <div className="rounded-lg bg-red-100 p-2 dark:bg-red-900/30"><UserX className="h-5 w-5 text-red-600" /></div>
            <div>
              <p className="font-medium text-gray-800 dark:text-gray-200">Deprovisioning: {selectedUser.name}</p>
              <p className="text-xs text-gray-400">{selectedUser.email}</p>
            </div>
          </div>
          <h3 className="mb-3 text-sm font-semibold uppercase text-gray-500">Actions Checklist</h3>
          <div className="space-y-2">
            {CHECKLIST.map((c) => (
              <label key={c.key} className="flex items-center gap-3 rounded-lg border border-gray-200 p-3 dark:border-gray-700">
                <input type="checkbox" checked={checklist[c.key]} onChange={(e) => setChecklist((p) => ({ ...p, [c.key]: e.target.checked }))} className="h-4 w-4 rounded border-gray-300 text-indigo-600" />
                <span className="text-sm text-gray-700 dark:text-gray-300">{c.label}</span>
                {c.key === "transfer_data" && checklist.transfer_data && (
                  <input value={transferTarget} onChange={(e) => setTransferTarget(e.target.value)} placeholder="Transfer to user ID" className="ml-auto w-48 rounded border border-gray-300 px-2 py-1 text-xs dark:border-gray-600 dark:bg-gray-700 dark:text-white" />
                )}
              </label>
            ))}
          </div>
          <div className="mt-4">
            <label className="text-sm font-medium text-gray-700 dark:text-gray-300">Reason (optional)</label>
            <input value={reason} onChange={(e) => setReason(e.target.value)} placeholder="e.g. Employee departure" className="mt-1 w-full rounded-lg border border-gray-300 px-3 py-2 text-sm dark:border-gray-600 dark:bg-gray-700 dark:text-white" />
          </div>
          <div className="mt-5 flex justify-end gap-2">
            <button onClick={() => setStep("search")} className="rounded-lg px-4 py-2 text-sm text-gray-500 hover:bg-gray-100 dark:hover:bg-gray-700">Back</button>
            <button onClick={handleExecute} disabled={executing} className="flex items-center gap-2 rounded-lg bg-red-600 px-4 py-2 text-sm font-medium text-white hover:bg-red-700 disabled:opacity-50">
              {executing ? <Loader2 className="h-4 w-4 animate-spin" /> : <Shield className="h-4 w-4" />}Execute Deprovisioning
            </button>
          </div>
        </div>
      )}

      {/* Step 3: Result */}
      {step === "result" && result && (
        <div className="space-y-4">
          <div className={`${cardCls} ${result.status === "completed" ? "border-green-300 dark:border-green-700" : "border-orange-300 dark:border-orange-700"}`}>
            <div className="flex items-center gap-3">
              <div className={`rounded-full p-2 ${result.status === "completed" ? "bg-green-100 dark:bg-green-900/30" : "bg-orange-100 dark:bg-orange-900/30"}`}>
                {result.status === "completed" ? <Check className="h-6 w-6 text-green-600" /> : <AlertCircle className="h-6 w-6 text-orange-600" />}
              </div>
              <div>
                <h2 className="text-lg font-bold text-gray-900 dark:text-white">Deprovisioning {result.status}</h2>
                <p className="text-sm text-gray-400">Completed at {new Date(result.completed_at).toLocaleString()}</p>
              </div>
            </div>
          </div>
          <div className={cardCls}>
            <h3 className="mb-3 text-sm font-semibold uppercase text-gray-500">Action Results</h3>
            <div className="space-y-2">
              {result.actions.map((a, i) => (
                <div key={i} className="flex items-center justify-between rounded-lg border border-gray-200 p-3 dark:border-gray-700">
                  <span className="text-sm text-gray-700 dark:text-gray-300">{a.name}</span>
                  <div className="flex items-center gap-2">
                    {a.message && <span className="text-xs text-gray-400">{a.message}</span>}
                    {a.status === "success" ? <Check className="h-4 w-4 text-green-500" /> : <X className="h-4 w-4 text-red-500" />}
                  </div>
                </div>
              ))}
            </div>
          </div>
          <button onClick={reset} className="flex items-center gap-2 rounded-lg bg-indigo-600 px-4 py-2 text-sm font-medium text-white hover:bg-indigo-700">New Deprovisioning</button>
        </div>
      )}

      {/* History */}
      <div>
        <h2 className="mb-3 flex items-center gap-2 text-sm font-semibold uppercase text-gray-500"><History className="h-4 w-4" /> Recent Deprovisionings</h2>
        <button onClick={loadHistory} className="mb-3 text-xs text-indigo-600 hover:underline">Load history</button>
        {history.length > 0 && (
          <div className="space-y-2">
            {history.map((h) => (
              <div key={h.id} className={`${cardCls} flex items-center justify-between py-3`}>
                <div>
                  <span className="font-medium text-gray-800 dark:text-gray-200">{h.user_name}</span>
                  <p className="text-xs text-gray-400">By {h.initiated_by} · {new Date(h.completed_at).toLocaleString()}</p>
                </div>
                <div className="flex items-center gap-3 text-xs">
                  <span className="text-gray-400">{h.actions_summary.success}/{h.actions_summary.total} actions</span>
                  <span className={`rounded-full px-2 py-0.5 font-medium ${h.status === "completed" ? "bg-green-100 text-green-700 dark:bg-green-900/30 dark:text-green-400" : "bg-orange-100 text-orange-700 dark:bg-orange-900/30 dark:text-orange-400"}`}>{h.status}</span>
                </div>
              </div>
            ))}
          </div>
        )}
      </div>
    </div>
  );
}
