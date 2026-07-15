"use client";

import { useAuditSiemForwarding } from "@ggid/sdk-react";
import { useTranslations } from "@/lib/i18n";
import { Send, AlertTriangle, CheckCircle, Activity } from "lucide-react";

export default function AuditSiemForwardingPage() {
  const t = useTranslations();
  const { data, loading, error, refresh, testForward } = useAuditSiemForwarding();

  if (loading) return <div className="p-8 text-gray-400">Loading SIEM forwarding...</div>;
  if (error) return <div className="p-8 text-red-400">Error: {error}</div>;

  return (
    <div className="min-h-screen bg-gray-950 text-white p-8">
      <div className="flex items-center justify-between mb-8">
        <div>
          <h1 className="text-2xl font-bold">SIEM Forwarding</h1>
          <p className="text-sm text-gray-400 mt-1">Forward audit events to SIEM destinations</p>
        </div>
        <button onClick={refresh} className="px-4 py-2 bg-blue-600 hover:bg-blue-700 rounded-lg text-sm font-medium transition">Refresh</button>
      </div>

      {/* Destinations */}
      <div className="space-y-4">
        {(data?.destinations ?? []).map((dest) => (
          <div key={dest.id} className="bg-gray-900 rounded-xl p-6">
            <div className="flex items-start justify-between mb-4">
              <div className="flex items-center gap-3">
                {dest.status === "connected" ? <CheckCircle className="w-5 h-5 text-green-400" /> : <AlertTriangle className="w-5 h-5 text-red-400" />}
                <div>
                  <h3 className="font-semibold">{dest.name}</h3>
                  <p className="text-xs text-gray-400">{dest.type} - {dest.format}</p>
                </div>
              </div>
              <div className="flex items-center gap-2">
                <span className={"text-xs px-2 py-0.5 rounded " + (dest.status === "connected" ? "bg-green-900 text-green-300" : "bg-red-900 text-red-300")}>{dest.status}</span>
                <button onClick={() => testForward(dest.id)} className="flex items-center gap-1 px-3 py-1 bg-gray-700 hover:bg-gray-600 rounded text-xs font-medium transition">
                  <Send className="w-3 h-3" /> Test
                </button>
              </div>
            </div>

            <div className="grid grid-cols-2 md:grid-cols-4 gap-4">
              <div>
                <p className="text-xs text-gray-500">Throughput</p>
                <p className="text-sm font-medium">{dest.throughput_events_per_min.toLocaleString()} ev/min</p>
              </div>
              <div>
                <p className="text-xs text-gray-500">Queue Depth</p>
                <p className="text-sm font-medium">{dest.queue_depth.toLocaleString()}</p>
              </div>
              <div>
                <p className="text-xs text-gray-500">Events Forwarded (24h)</p>
                <p className="text-sm font-medium">{dest.events_forwarded_24h.toLocaleString()}</p>
              </div>
              <div>
                <p className="text-xs text-gray-500">Last Error</p>
                <p className="text-xs text-gray-400">{dest.last_error || "None"}</p>
              </div>
            </div>

            <div className="mt-3">
              <p className="text-xs text-gray-500 mb-1">Event Filter ({dest.event_filter.length} types)</p>
              <div className="flex flex-wrap gap-1">
                {dest.event_filter.map((f) => (
                  <span key={f} className="text-xs px-1.5 py-0.5 bg-gray-800 rounded text-gray-400">{f}</span>
                ))}
              </div>
            </div>
          </div>
        ))}
      </div>
    </div>
  );
}
