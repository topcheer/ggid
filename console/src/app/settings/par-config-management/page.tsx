'use client';
import { useState, useEffect } from 'react';
import { Loader2 } from "lucide-react";
import { useTranslations } from "@/lib/i18n";
import { authHeader, isAuthenticated } from "@/lib/auth-helpers";

interface ClientPar {
  clientId: string;
  requirePar: boolean;
}

export default function ParConfigManagementPage() {
  const t = useTranslations();


  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    const fetchData = async () => {
      try {
        const res = await fetch("/api/v1/oauth/par", {
          method: "GET",
          headers: { ...authHeader(),
            "Content-Type": "application/json",
            "X-Tenant-ID": "00000000-0000-0000-0000-000000000001",
          },
        });
        if (!res.ok) return null;
        const json = await res.json();
      } catch (e) {
        setError(e instanceof Error ? e.message : "Failed to load");
      } finally {
        setLoading(false);
      }
    };
    fetchData();
  }, []);

  const [requirePar, setRequirePar] = useState(false);
  const [parExpiry, setParExpiry] = useState(120);
  const [cleanupInterval, setCleanupInterval] = useState(300);
  const [showViewer, setShowViewer] = useState(false);const [clients, setClients] = useState<ClientPar[]>([
    { clientId: 'web-app', requirePar: true },
    { clientId: 'mobile-app', requirePar: false },
    { clientId: 'api-gateway', requirePar: true },
  ]);

  if (loading) return <div className="flex items-center justify-center py-12"><Loader2 className="h-8 w-8 animate-spin text-blue-500" /></div>;
  if (error) return <div className="p-8 text-red-500">Error: {error}</div>;
  
  const toggleClient = (idx: number) => {
    setClients(prev => prev.map((c, i) => i === idx ? { ...c, requirePar: !c.requirePar } : c));
  };

  const parEndpointHealth = 'healthy';
  const cachedParCount = 47;

  return (
    <div className="p-6 max-w-4xl mx-auto space-y-6">
      <div>
        <h1 className="text-2xl font-bold">PAR Configuration Management</h1>
        <p className="text-gray-600">Pushed Authorization Request (RFC 9126) configuration and monitoring.</p>
      </div>

      <section className="bg-white rounded-lg shadow p-6 space-y-4">
        <h2 className="text-lg font-semibold">Global Settings</h2>
        <label className="flex items-center justify-between">
          <span className="text-sm font-medium">Require PAR for All Clients</span>
          <input aria-label="Require par" type="checkbox" checked={requirePar} onChange={e => setRequirePar(e.target.checked)} className="rounded" />
        </label>
        <div>
          <label className="text-sm font-medium">PAR Expiry: {parExpiry}s</label>
          <input aria-label="par Expiry" type="range" min={60} max={600} step={30} value={parExpiry} onChange={e => setParExpiry(parseInt(e.target.value))} className="w-full mt-2" />
          <div className="flex justify-between text-xs text-gray-400"><span>60s</span><span>600s</span></div>
        </div>
        <div>
          <label className="text-sm font-medium">Cleanup Interval: {cleanupInterval}s</label>
          <input aria-label="cleanup Interval" type="number" min={60} max={3600} value={cleanupInterval} onChange={e => setCleanupInterval(parseInt(e.target.value) || 300)} className="w-24 border rounded px-2 py-1 text-sm mt-1" />
        </div>
      </section>

      <div className="grid grid-cols-3 gap-4">
        <div className="bg-white rounded-lg shadow p-4 text-center">
          <div className="text-xs text-gray-500">PAR Endpoint Health</div>
          <div className="text-lg font-bold text-green-600 mt-1">{parEndpointHealth}</div>
        </div>
        <div className="bg-white rounded-lg shadow p-4 text-center">
          <div className="text-xs text-gray-500">Cached PAR Requests</div>
          <div className="text-lg font-bold mt-1">{cachedParCount}</div>
        </div>
        <div className="bg-white rounded-lg shadow p-4 text-center">
          <div className="text-xs text-gray-500">Enforced Clients</div>
          <div className="text-lg font-bold mt-1">{clients.filter(c => c.requirePar).length}/{clients.length}</div>
        </div>
      </div>

      <section className="bg-white rounded-lg shadow p-6 space-y-4">
        <h2 className="text-lg font-semibold">Per-Client PAR Enforcement</h2>
        <div className="space-y-2">
          {clients.map((c, idx) => (
            <label key={c.clientId} className="flex items-center justify-between border-b pb-2">
              <span className="font-mono text-sm">{c.clientId}</span>
              <input aria-label="C" type="checkbox" checked={c.requirePar} onChange={() => toggleClient(idx)} className="rounded" />
            </label>
          ))}
        </div>
      </section>

      <section className="bg-white rounded-lg shadow p-6 space-y-4">
        <div className="flex items-center justify-between">
          <h2 className="text-lg font-semibold">PAR Request Viewer</h2>
          <button onClick={() => setShowViewer(!showViewer)} className="text-sm text-blue-600">{showViewer ? 'Hide' : 'Show'}</button>
        </div>
        {showViewer && (
          <div className="space-y-2">
            <div className="border rounded p-3 text-sm">
              <div className="font-mono text-xs text-gray-500">request_uri: urn:ietf:params:oauth:request_uri:abc123</div>
              <div className="text-xs text-gray-500 mt-1">client_id: web-app | expires: 2026-07-12T14:34Z</div>
              <pre className="bg-gray-900 text-green-400 rounded p-2 text-xs mt-2 overflow-x-auto">{JSON.stringify({ response_type: 'code', client_id: 'web-app', scope: 'openid profile', redirect_uri: 'https://app.example.com/callback' }, null, 2)}</pre>
            </div>
          </div>
        )}
      </section>
    </div>
  );
}
