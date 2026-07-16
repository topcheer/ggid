"use client";

import { useMitreAttackMapping } from "@ggid/sdk-react";
import { Crosshair, Download, Activity } from "lucide-react";
import { useTranslations } from "@/lib/i18n";

export default function MitreAttackMappingPage() {
  const t = useTranslations();

  const { data, loading, error, refresh } = useMitreAttackMapping();

  if (loading) return <div className="p-8 text-gray-400">Loading MITRE ATT&CK mapping...</div>;
  if (error) return <div className="p-8 text-red-400">Error: {error}</div>;

  const tactics = ["reconnaissance", "credential_access", "lateral_movement", "exfiltration"];
  const tacticColors: Record<string, string> = {
    detected: "bg-green-900 text-green-300",
    mitigated: "bg-blue-900 text-blue-300",
    unknown: "bg-red-900 text-red-300",
  };

  return (
    <div className="min-h-screen bg-gray-950 text-white p-8">
      <div className="flex items-center justify-between mb-8">
        <div>
          <h1 className="text-2xl font-bold">MITRE ATT&CK Mapping</h1>
          <p className="text-sm text-gray-400 mt-1">Identity-focused threat technique mapping</p>
        </div>
        <div className="flex items-center gap-2">
          <button className="flex items-center gap-2 px-4 py-2 bg-gray-700 hover:bg-gray-600 rounded-lg text-sm font-medium transition">
            <Download className="w-4 h-4" /> Export STIX
          </button>
          <button onClick={refresh} className="px-4 py-2 bg-blue-600 hover:bg-blue-700 rounded-lg text-sm font-medium transition">Refresh</button>
        </div>
      </div>

      {/* Coverage per Tactic */}
      <div className="grid grid-cols-1 md:grid-cols-4 gap-4 mb-6">
        {tactics.map((tactic) => {
          const techniques = data?.techniques?.filter((t) => t.tactic === tactic) ?? [];
          const detected = techniques.filter((t) => t.detection_status === "detected").length;
          const pct = techniques.length > 0 ? Math.round((detected / techniques.length) * 100) : 0;
          return (
            <div key={tactic} className="bg-gray-900 rounded-xl p-4">
              <p className="text-xs text-gray-400 capitalize mb-1">{tactic.replace("_", " ")}</p>
              <div className="flex items-center justify-between">
                <p className="text-xl font-bold">{pct}%</p>
                <p className="text-xs text-gray-500">{detected}/{techniques.length}</p>
              </div>
              <div className="h-1.5 bg-gray-700 rounded-full mt-2">
                <div className={"h-full rounded-full " + (pct > 70 ? "bg-green-500" : pct > 40 ? "bg-yellow-500" : "bg-red-500")} style={{ width: pct + "%" }} />
              </div>
            </div>
          );
        })}
      </div>

      {/* Kill Chain Visual */}
      <div className="bg-gray-900 rounded-xl p-6 mb-6">
        <h2 className="text-sm font-semibold flex items-center gap-2 mb-4">
          <Crosshair className="w-4 h-4 text-blue-400" />
          Kill Chain Flow
        </h2>
        <div className="flex items-center gap-1 overflow-x-auto pb-2">
          {tactics.map((t, i) => (
            <div key={t} className="flex items-center gap-1 flex-shrink-0">
              <span className="text-xs px-3 py-2 bg-gray-800 rounded-lg border border-gray-700 capitalize">{t.replace("_", " ")}</span>
              {i < tactics.length - 1 && <span className="text-gray-600">{" -> "}</span>}
            </div>
          ))}
        </div>
      </div>

      {/* Techniques Table */}
      <div className="bg-gray-900 rounded-xl p-6">
        <h2 className="text-sm font-semibold flex items-center gap-2 mb-4">
          <Activity className="w-4 h-4 text-purple-400" />
          Identity Techniques
        </h2>
        <div className="overflow-x-auto">
          <table className="w-full text-sm">
            <thead>
              <tr className="border-b border-gray-800 text-gray-400">
                <th scope="col" className="text-left py-2 pr-3">T-ID</th>
                <th scope="col" className="text-left py-2 pr-3">Name</th>
                <th scope="col" className="text-left py-2 pr-3">Tactic</th>
                <th scope="col" className="text-left py-2 pr-3">Status</th>
              </tr>
            </thead>
            <tbody>
              {(data?.techniques ?? []).map((t) => (
                <tr key={t.t_id} className="border-b border-gray-800">
                  <td className="py-3 pr-3 font-mono text-xs text-purple-400">{t.t_id}</td>
                  <td className="py-3 pr-3 text-xs">{t.name}</td>
                  <td className="py-3 pr-3 text-xs text-gray-400 capitalize">{t.tactic.replace("_", " ")}</td>
                  <td className="py-3 pr-3">
                    <span className={"text-xs px-2 py-0.5 rounded " + (tacticColors[t.detection_status] ?? "bg-gray-700")}>
                      {t.detection_status}
                    </span>
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      </div>
    </div>
  );
}
