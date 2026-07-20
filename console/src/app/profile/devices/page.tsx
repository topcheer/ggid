"use client";
import { useState, useEffect, useCallback } from "react";
import { Smartphone, Laptop, Tablet, Trash2, Loader2, AlertCircle, CheckCircle2, Fingerprint } from "lucide-react";
import { usePageTitle } from "@/lib/usePageTitle";
import { authHeader } from "@/lib/auth-helpers";
import { API_BASE_URL } from "@/lib/api-config";

const API_BASE = API_BASE_URL;

interface Device { id: string; name: string; platform: string; os: string; last_used: string; trusted: boolean; type: string; }

export default function ProfileDevicesPage() {
  usePageTitle("My Devices");
  const [devices, setDevices] = useState<Device[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState("");
  const [revokeTarget, setRevokeTarget] = useState<Device | null>(null);

  const load = useCallback(async () => {
    setLoading(true); setError("");
    try {
      const res = await fetch(`${API_BASE}/api/v1/auth/webauthn/credentials`, { headers: { ...authHeader() } });
      if (res.ok) {
        const d = await res.json();
        const creds = d.credentials || d.items || [];
        setDevices(creds.map((c: any) => ({ id: c.id || c.credential_id || "", name: c.name || c.device_name || "Unknown Device", platform: c.platform || "", os: c.os || c.transports?.join(", ") || "", last_used: c.last_used || c.created_at || "", trusted: c.trusted || false, type: c.type || "passkey" })));
      } else {
        const sRes = await fetch(`${API_BASE}/api/v1/auth/sessions`, { headers: { ...authHeader() } });
        if (sRes.ok) {
          const sd = await sRes.json();
          const sessions = sd.sessions || sd.items || [];
          setDevices(sessions.map((s: any) => ({ id: s.session_id || s.id || "", name: s.device_name || s.user_agent?.substring(0, 40) || "Unknown Device", platform: parsePlatform(s.user_agent || ""), os: s.user_agent || "", last_used: s.last_active || s.created_at || "", trusted: s.trusted === "true" || s.trusted === true, type: "session" })));
        }
      }
    } catch { setError("Failed to load devices"); }
    setLoading(false);
  }, []);

  useEffect(() => { load(); }, [load]);

  const handleRevoke = async (device: Device) => {
    try { await fetch(`${API_BASE}/api/v1/auth/webauthn/credentials/${device.id}`, { method: "DELETE", headers: { ...authHeader() } }); setDevices(prev => prev.filter(d => d.id !== device.id)); setRevokeTarget(null); } catch { setError("Failed to revoke device"); }
  };

  if (loading) return <div className="flex justify-center py-20"><Loader2 className="w-8 h-8 animate-spin text-blue-600" /></div>;

  return (
    <div className="mx-auto max-w-3xl p-6">
      <h1 className="mb-1 text-2xl font-bold text-gray-900 dark:text-white">My Devices</h1>
      <p className="mb-6 text-sm text-gray-500">Manage registered devices and passkeys associated with your account.</p>
      {error && <div className="mb-4 flex items-center gap-2 rounded-lg border border-red-200 bg-red-50 px-4 py-3 text-sm text-red-700 dark:border-red-800 dark:bg-red-950"><AlertCircle className="h-4 w-4 shrink-0" /> {error}</div>}
      {devices.length === 0 ? (
        <div className="rounded-xl border border-gray-200 bg-white p-8 text-center dark:border-gray-800 dark:bg-gray-900">
          <Fingerprint className="mx-auto mb-3 h-12 w-12 text-gray-300" />
          <p className="text-sm text-gray-500">No registered devices. Set up a passkey from Profile {"->"} Security tab.</p>
        </div>
      ) : (
        <div className="space-y-3">{devices.map(d => { const Icon = (d.platform.includes("iPhone") || d.platform.includes("Android")) ? Smartphone : d.platform.includes("iPad") ? Tablet : Laptop; return (
          <div key={d.id} className="flex items-center justify-between rounded-xl border border-gray-200 bg-white p-4 dark:border-gray-800 dark:bg-gray-900">
            <div className="flex items-center gap-3"><div className="flex h-10 w-10 items-center justify-center rounded-lg bg-gray-100 dark:bg-gray-800"><Icon className="h-5 w-5 text-gray-500" /></div><div><p className="text-sm font-medium text-gray-900 dark:text-white">{d.name}</p><p className="text-xs text-gray-500">{d.type === "passkey" ? "Passkey" : "Session"} - {d.platform || "Unknown platform"}</p>{d.last_used && <p className="text-xs text-gray-400">Last used: {new Date(d.last_used).toLocaleString()}</p>}</div></div>
            <div className="flex items-center gap-2">{d.trusted && <span className="flex items-center gap-1 rounded-full bg-green-50 px-2 py-0.5 text-xs font-medium text-green-600 dark:bg-green-950"><CheckCircle2 className="h-3 w-3" /> Trusted</span>}<button onClick={() => setRevokeTarget(d)} className="rounded p-1.5 text-gray-400 hover:bg-red-50 hover:text-red-600"><Trash2 className="h-4 w-4" /></button></div>
          </div>
        ); })}</div>
      )}
      {revokeTarget && (<div className="fixed inset-0 z-50 flex items-center justify-center bg-black/50"><div className="max-w-md rounded-xl bg-white p-6 dark:bg-gray-900 mx-4"><h3 className="mb-2 text-lg font-semibold">Remove Device</h3><p className="mb-4 text-sm text-gray-500">Remove <strong>{revokeTarget.name}</strong>? You won't be able to use it for authentication.</p><div className="flex justify-end gap-2"><button onClick={() => setRevokeTarget(null)} className="rounded-lg border border-gray-300 px-4 py-2 text-sm dark:border-gray-700">Cancel</button><button onClick={() => handleRevoke(revokeTarget)} className="rounded-lg bg-red-600 px-4 py-2 text-sm font-medium text-white hover:bg-red-700">Remove</button></div></div></div>)}
    </div>
  );
}

function parsePlatform(ua: string): string { if (ua.includes("iPhone") || ua.includes("iPad")) return "iOS"; if (ua.includes("Android")) return "Android"; if (ua.includes("Mac")) return "macOS"; if (ua.includes("Windows")) return "Windows"; if (ua.includes("Linux")) return "Linux"; return ""; }
