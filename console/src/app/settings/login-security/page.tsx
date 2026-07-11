"use client";

import { useState, useEffect, useCallback } from "react";
import { useApi } from "@/lib/api";
import {
  Lock, Shield, Unlock, AlertCircle, Loader2, X, Plus,
  Trash2, Save, Clock, Ban, Globe,
} from "lucide-react";

interface Lockout {
  id: string;
  user_id: string;
  user_name: string;
  failed_count: number;
  locked_until: string;
  locked_at: string;
  ip_address: string;
}

interface LoginPolicy {
  max_attempts: number;
  lockout_duration_minutes: number;
}

interface IpAllowEntry {
  id: string;
  cidr: string;
  description: string;
  created_at: string;
}

export default function LoginSecurityPage() {
  const { apiFetch } = useApi();
  const [lockouts, setLockouts] = useState<Lockout[]>([]);
  const [policy, setPolicy] = useState<LoginPolicy>({ max_attempts: 5, lockout_duration_minutes: 30 });
  const [ipAllow, setIpAllow] = useState<IpAllowEntry[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [editPolicy, setEditPolicy] = useState(false);
  const [draftPolicy, setDraftPolicy] = useState<LoginPolicy>({ max_attempts: 5, lockout_duration_minutes: 30 });
  const [showIpAdd, setShowIpAdd] = useState(false);
  const [ipForm, setIpForm] = useState({ cidr: "", description: "" });
  const [confirmIpDelete, setConfirmIpDelete] = useState<string | null>(null);
  const [unlocking, setUnlocking] = useState<string | null>(null);

  const load = useCallback(async () => {
    setLoading(true);
    setError(null);
    try {
      const [lockRes, polRes, ipRes] = await Promise.all([
        apiFetch<{ lockouts?: Lockout[]; items?: Lockout[] }>("/api/v1/auth/lockouts").catch(() => ({ lockouts: [], items: [] as Lockout[] })),
        apiFetch<LoginPolicy>("/api/v1/auth/login-policy").catch(() => null),
        apiFetch<{ entries?: IpAllowEntry[]; items?: IpAllowEntry[] }>("/api/v1/auth/ip-allowlist").catch(() => ({ entries: [], items: [] as IpAllowEntry[] })),
      ]);
      setLockouts(lockRes.lockouts ?? lockRes.items ?? []);
      if (polRes) { setPolicy(polRes); setDraftPolicy(polRes); }
      setIpAllow(ipRes.entries ?? ipRes.items ?? []);
    } catch {
      setError("Failed to load login security data");
    } finally {
      setLoading(false);
    }
  }, [apiFetch]);

  useEffect(() => { load(); }, [load]);

  const handleUnlock = async (userId: string) => {
    setUnlocking(userId);
    try {
      await apiFetch(`/api/v1/auth/lockouts/${userId}/unlock`, { method: "POST" });
      setLockouts((prev) => prev.filter((l) => l.user_id !== userId));
    } catch {
      setError("Failed to unlock user");
    } finally {
      setUnlocking(null);
    }
  };

  const handleSavePolicy = async () => {
    try {
      await apiFetch("/api/v1/auth/login-policy", { method: "PUT", body: JSON.stringify(draftPolicy) });
      setPolicy(draftPolicy);
      setEditPolicy(false);
    } catch {
      setError("Failed to save policy");
    }
  };

  const handleAddIp = async () => {
    if (!ipForm.cidr.trim()) return;
    try {
      await apiFetch("/api/v1/auth/ip-allowlist", { method: "POST", body: JSON.stringify(ipForm) });
      setIpForm({ cidr: "", description: "" });
      setShowIpAdd(false);
      await load();
    } catch {
      setError("Failed to add IP entry");
    }
  };

  const handleDeleteIp = async (id: string) => {
    try {
      await apiFetch(`/api/v1/auth/ip-allowlist/${id}`, { method: "DELETE" });
      setConfirmIpDelete(null);
      await load();
    } catch {
      setError("Failed to delete IP entry");
    }
  };

  const cardCls = "rounded-xl border border-gray-200 bg-white p-5 shadow-sm dark:border-gray-700 dark:bg-gray-800";

  return (
    <div className="space-y-6">
      <div>
        <h1 className="flex items-center gap-2 text-2xl font-bold text-gray-900 dark:text-white">
          <Shield className="h-6 w-6 text-indigo-600" /> Login Security
        </h1>
        <p className="mt-1 text-sm text-gray-500 dark:text-gray-400">Brute-force protection, lockouts, and IP allowlisting.</p>
      </div>

      {error && (
        <div className="flex items-center gap-2 rounded-lg bg-red-50 px-4 py-3 text-sm text-red-700 dark:bg-red-900/20 dark:text-red-400">
          <AlertCircle className="h-4 w-4 shrink-0" />{error}
          <button onClick={() => setError(null)} className="ml-auto"><X className="h-4 w-4" /></button>
        </div>
      )}

      {/* Policy card */}
      <div className="grid gap-4 md:grid-cols-3">
        <div className={cardCls}>
          <h3 className="text-xs font-semibold uppercase text-gray-500">Max Attempts</h3>
          {editPolicy ? (
            <input type="number" value={draftPolicy.max_attempts} onChange={(e) => setDraftPolicy((p) => ({ ...p, max_attempts: Number(e.target.value) }))} className="mt-2 w-full rounded-lg border border-gray-300 px-3 py-1.5 text-lg font-bold text-gray-800 dark:border-gray-600 dark:bg-gray-700 dark:text-white" />
          ) : (
            <p className="mt-2 text-2xl font-bold text-indigo-600">{policy.max_attempts}</p>
          )}
        </div>
        <div className={cardCls}>
          <h3 className="text-xs font-semibold uppercase text-gray-500">Lockout Duration</h3>
          {editPolicy ? (
            <input type="number" value={draftPolicy.lockout_duration_minutes} onChange={(e) => setDraftPolicy((p) => ({ ...p, lockout_duration_minutes: Number(e.target.value) }))} className="mt-2 w-full rounded-lg border border-gray-300 px-3 py-1.5 text-lg font-bold text-gray-800 dark:border-gray-600 dark:bg-gray-700 dark:text-white" />
          ) : (
            <p className="mt-2 text-2xl font-bold text-indigo-600">{policy.lockout_duration_minutes} <span className="text-sm text-gray-400">min</span></p>
          )}
        </div>
        <div className={`${cardCls} flex items-center justify-center`}>
          {editPolicy ? (
            <div className="flex gap-2">
              <button onClick={handleSavePolicy} className="flex items-center gap-1.5 rounded-lg bg-green-600 px-4 py-2 text-sm font-medium text-white hover:bg-green-700"><Save className="h-4 w-4" /> Save</button>
              <button onClick={() => { setEditPolicy(false); setDraftPolicy(policy); }} className="rounded-lg px-4 py-2 text-sm text-gray-500 hover:bg-gray-100 dark:hover:bg-gray-700">Cancel</button>
            </div>
          ) : (
            <button onClick={() => setEditPolicy(true)} className="rounded-lg border border-gray-300 px-4 py-2 text-sm font-medium text-gray-600 hover:bg-gray-50 dark:border-gray-600 dark:text-gray-300 dark:hover:bg-gray-700">Edit Policy</button>
          )}
        </div>
      </div>

      <div className="grid gap-6 lg:grid-cols-2">
        {/* Active lockouts */}
        <div>
          <h2 className="mb-3 flex items-center gap-2 text-sm font-semibold uppercase text-gray-500"><Lock className="h-4 w-4" /> Active Lockouts ({lockouts.length})</h2>
          {loading ? (
            <div className="flex justify-center py-8"><Loader2 className="h-6 w-6 animate-spin text-indigo-600" /></div>
          ) : lockouts.length === 0 ? (
            <div className={cardCls}><div className="py-8 text-center"><Shield className="mx-auto h-10 w-10 text-green-300" /><p className="mt-3 text-sm text-gray-400">No active lockouts.</p></div></div>
          ) : (
            <div className="space-y-3">
              {lockouts.map((l) => (
                <div key={l.id} className={`${cardCls} border-red-200 dark:border-red-800`}>
                  <div className="flex items-center justify-between">
                    <div>
                      <span className="font-medium text-gray-800 dark:text-gray-200">{l.user_name}</span>
                      <div className="mt-1 flex items-center gap-3 text-xs text-gray-400">
                        <span className="flex items-center gap-1"><Ban className="h-3 w-3" /> {l.failed_count} failed</span>
                        <span className="flex items-center gap-1"><Clock className="h-3 w-3" /> Until {new Date(l.locked_until).toLocaleTimeString()}</span>
                      </div>
                      <p className="mt-0.5 font-mono text-xs text-gray-400">IP: {l.ip_address}</p>
                    </div>
                    <button onClick={() => handleUnlock(l.user_id)} disabled={unlocking === l.user_id} className="flex items-center gap-1.5 rounded-lg bg-indigo-600 px-3 py-1.5 text-xs font-medium text-white hover:bg-indigo-700 disabled:opacity-50">
                      {unlocking === l.user_id ? <Loader2 className="h-3.5 w-3.5 animate-spin" /> : <Unlock className="h-3.5 w-3.5" />} Unlock
                    </button>
                  </div>
                </div>
              ))}
            </div>
          )}
        </div>

        {/* IP Allowlist */}
        <div>
          <div className="mb-3 flex items-center justify-between">
            <h2 className="flex items-center gap-2 text-sm font-semibold uppercase text-gray-500"><Globe className="h-4 w-4" /> IP Allowlist ({ipAllow.length})</h2>
            <button onClick={() => setShowIpAdd(true)} className="flex items-center gap-1 rounded-lg border border-gray-300 px-2 py-1 text-xs font-medium text-gray-500 hover:bg-gray-50 dark:border-gray-600 dark:hover:bg-gray-700"><Plus className="h-3 w-3" /> Add</button>
          </div>
          {ipAllow.length === 0 ? (
            <div className={cardCls}><div className="py-8 text-center"><Globe className="mx-auto h-10 w-10 text-gray-300" /><p className="mt-3 text-sm text-gray-400">No IP restrictions. All IPs allowed.</p></div></div>
          ) : (
            <div className="space-y-2">
              {ipAllow.map((ip) => (
                <div key={ip.id} className={`${cardCls} flex items-center justify-between py-3`}>
                  <div>
                    <code className="font-mono text-sm text-gray-700 dark:text-gray-300">{ip.cidr}</code>
                    {ip.description && <p className="text-xs text-gray-400">{ip.description}</p>}
                  </div>
                  <button onClick={() => setConfirmIpDelete(ip.id)} className="rounded-lg p-1.5 text-red-500 hover:bg-red-50 dark:hover:bg-red-900/20"><Trash2 className="h-4 w-4" /></button>
                </div>
              ))}
            </div>
          )}
        </div>
      </div>

      {/* Add IP modal */}
      {showIpAdd && (
        <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/50" onClick={() => setShowIpAdd(false)}>
          <div className="w-full max-w-sm rounded-xl bg-white p-6 shadow-xl dark:bg-gray-800" onClick={(e) => e.stopPropagation()}>
            <div className="flex items-center justify-between">
              <h2 className="text-lg font-semibold text-gray-900 dark:text-white">Add IP/CIDR</h2>
              <button onClick={() => setShowIpAdd(false)}><X className="h-5 w-5 text-gray-400" /></button>
            </div>
            <div className="mt-4 space-y-3">
              <input value={ipForm.cidr} onChange={(e) => setIpForm((p) => ({ ...p, cidr: e.target.value }))} placeholder="10.0.0.0/8 or 192.168.1.100" className="w-full rounded-lg border border-gray-300 px-3 py-2 font-mono text-sm dark:border-gray-600 dark:bg-gray-700 dark:text-white" />
              <input value={ipForm.description} onChange={(e) => setIpForm((p) => ({ ...p, description: e.target.value }))} placeholder="Office network (optional)" className="w-full rounded-lg border border-gray-300 px-3 py-2 text-sm dark:border-gray-600 dark:bg-gray-700 dark:text-white" />
            </div>
            <div className="mt-5 flex justify-end gap-2">
              <button onClick={() => setShowIpAdd(false)} className="rounded-lg px-4 py-2 text-sm text-gray-500 hover:bg-gray-100 dark:hover:bg-gray-700">Cancel</button>
              <button onClick={handleAddIp} disabled={!ipForm.cidr.trim()} className="rounded-lg bg-indigo-600 px-4 py-2 text-sm font-medium text-white hover:bg-indigo-700 disabled:opacity-50">Add</button>
            </div>
          </div>
        </div>
      )}

      {/* IP delete confirm */}
      {confirmIpDelete && (
        <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/50" onClick={() => setConfirmIpDelete(null)}>
          <div className="w-full max-w-sm rounded-xl bg-white p-6 shadow-xl dark:bg-gray-800" onClick={(e) => e.stopPropagation()}>
            <div className="flex items-center gap-3">
              <div className="rounded-full bg-red-100 p-2 dark:bg-red-900/30"><Trash2 className="h-5 w-5 text-red-600" /></div>
              <p className="text-sm text-gray-500">Remove this IP from the allowlist?</p>
            </div>
            <div className="mt-5 flex justify-end gap-2">
              <button onClick={() => setConfirmIpDelete(null)} className="rounded-lg px-4 py-2 text-sm text-gray-500 hover:bg-gray-100 dark:hover:bg-gray-700">Cancel</button>
              <button onClick={() => handleDeleteIp(confirmIpDelete)} className="rounded-lg bg-red-600 px-4 py-2 text-sm font-medium text-white hover:bg-red-700">Remove</button>
            </div>
          </div>
        </div>
      )}
    </div>
  );
}
