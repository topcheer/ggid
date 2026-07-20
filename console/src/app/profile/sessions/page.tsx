"use client";
import { useState, useEffect, useCallback } from "react";
import { Clock, Globe, Smartphone, Trash2, Loader2, AlertCircle, Shield, RefreshCw, MapPin } from "lucide-react";
import { usePageTitle } from "@/lib/usePageTitle";
import { authHeader } from "@/lib/auth-helpers";
import { API_BASE_URL } from "@/lib/api-config";

const API_BASE = API_BASE_URL;

interface Session { session_id: string; ip_address: string; user_agent: string; device_type: string; created_at: string; last_active: string; location: string; trusted: boolean; }

export default function ProfileSessionsPage() {
  usePageTitle("My Sessions");
  const [sessions, setSessions] = useState<Session[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState("");
  const [revokeTarget, setRevokeTarget] = useState<Session | null>(null);

  const load = useCallback(async () => {
    setLoading(true); setError("");
    try { const res = await fetch(`${API_BASE}/api/v1/auth/sessions`, { headers: { ...authHeader() } }); if (res.ok) { const d = await res.json(); setSessions(d.sessions || d.items || (Array.isArray(d) ? d : [])); } } catch { setError("Failed to load sessions"); }
    setLoading(false);
  }, []);

  useEffect(() => { load(); }, [load]);

  const handleRevoke = async (session: Session) => {
    try { await fetch(`${API_BASE}/api/v1/auth/sessions/${session.session_id}`, { method: "DELETE", headers: { ...authHeader() } }); setSessions(prev => prev.filter(s => s.session_id !== session.session_id)); setRevokeTarget(null); } catch { setError("Failed to revoke session"); }
  };

  const handleRevokeAll = async () => { try { await fetch(`${API_BASE}/api/v1/auth/sessions`, { method: "DELETE", headers: { ...authHeader() } }); setSessions([]); } catch { setError("Failed"); } };

  if (loading) return <div className="flex justify-center py-20"><Loader2 className="w-8 h-8 animate-spin text-blue-600" /></div>;

  return (
    <div className="mx-auto max-w-3xl p-6">
      <div className="flex items-center justify-between mb-6"><div><h1 className="text-2xl font-bold text-gray-900 dark:text-white">My Sessions</h1><p className="text-sm text-gray-500">Active login sessions across your devices.</p></div><div className="flex gap-2"><button onClick={load} className="flex items-center gap-1.5 rounded-lg border border-gray-300 px-3 py-2 text-sm dark:border-gray-700"><RefreshCw className="h-4 w-4" /> Refresh</button>{sessions.length > 0 && <button onClick={handleRevokeAll} className="flex items-center gap-1.5 rounded-lg border border-red-300 px-3 py-2 text-sm text-red-600 dark:border-red-800"><Trash2 className="h-4 w-4" /> Revoke All</button>}</div></div>
      {error && <div className="mb-4 flex items-center gap-2 rounded-lg border border-red-200 bg-red-50 px-4 py-3 text-sm text-red-700 dark:border-red-800 dark:bg-red-950"><AlertCircle className="h-4 w-4 shrink-0" /> {error}</div>}
      {sessions.length === 0 ? (<div className="rounded-xl border border-gray-200 bg-white p-8 text-center dark:border-gray-800 dark:bg-gray-900"><Shield className="mx-auto mb-3 h-12 w-12 text-gray-300" /><p className="text-sm text-gray-500">No active sessions.</p></div>) : (<div className="space-y-3">{sessions.map(s => (<div key={s.session_id} className="flex items-center justify-between rounded-xl border border-gray-200 bg-white p-4 dark:border-gray-800 dark:bg-gray-900"><div className="flex items-center gap-3"><div className="flex h-10 w-10 items-center justify-center rounded-lg bg-gray-100 dark:bg-gray-800"><Smartphone className="h-5 w-5 text-gray-500" /></div><div><p className="text-sm font-medium text-gray-900 dark:text-white">{s.device_type || s.user_agent?.substring(0, 40) || "Unknown Device"}</p><div className="flex items-center gap-3 text-xs text-gray-400">{s.ip_address && <span className="flex items-center gap-1"><Globe className="h-3 w-3" /> {s.ip_address.replace(/\/\d+$/, "")}</span>}{s.location && <span className="flex items-center gap-1"><MapPin className="h-3 w-3" /> {s.location}</span>}</div>{s.last_active && <p className="flex items-center gap-1 text-xs text-gray-400"><Clock className="h-3 w-3" /> Last active: {new Date(s.last_active).toLocaleString()}</p>}</div></div><button onClick={() => setRevokeTarget(s)} className="rounded p-1.5 text-gray-400 hover:bg-red-50 hover:text-red-600" title="Revoke"><Trash2 className="h-4 w-4" /></button></div>))}</div>)}
      {revokeTarget && (<div className="fixed inset-0 z-50 flex items-center justify-center bg-black/50"><div className="max-w-md rounded-xl bg-white p-6 dark:bg-gray-900 mx-4"><h3 className="mb-2 text-lg font-semibold">Revoke Session</h3><p className="mb-4 text-sm text-gray-500">This will sign out the device. Are you sure?</p><div className="flex justify-end gap-2"><button onClick={() => setRevokeTarget(null)} className="rounded-lg border border-gray-300 px-4 py-2 text-sm dark:border-gray-700">Cancel</button><button onClick={() => handleRevoke(revokeTarget)} className="rounded-lg bg-red-600 px-4 py-2 text-sm font-medium text-white hover:bg-red-700">Revoke</button></div></div></div>)}
    </div>
  );
}
