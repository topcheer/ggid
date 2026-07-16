"use client";
import { useState, useEffect } from "react";
import { useTranslations } from "@/lib/i18n";

interface VelocityRule {
  name: string;
  distance_km: number;
  time_hours: number;
  action: "challenge" | "block" | "log";
  enabled: boolean;
}

const samplePairs = [
  { from: "San Francisco, US", to: "Tokyo, JP", distance_km: 8277, time_hours: 1.5, triggered: true },
  { from: "New York, US", to: "London, UK", distance_km: 5570, time_hours: 2.0, triggered: true },
  { from: "Berlin, DE", to: "Paris, FR", distance_km: 1054, time_hours: 3.0, triggered: false },
];

export default function GeoVelocityRulesPage() {
  const t = useTranslations();

  const [rules, setRules] = useState<VelocityRule[]>([]);
  const [exemptions, setExemptions] = useState<string[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [simFrom, setSimFrom] = useState("");
  const [simTo, setSimTo] = useState("");
  const [simResult, setSimResult] = useState<string | null>(null);

  useEffect(() => {
    fetch("/api/v1/auth/velocity-rules", {
      headers: { "Content-Type": "application/json", "X-Tenant-ID": "00000000-0000-0000-0000-000000000001" },
    })
      .then(r => { if (!r.ok) throw new Error(`HTTP ${r.status}`); return r.json(); })
      .then(data => {
        if (data) {
          if (data.rules) setRules(data.rules);
          if (data.exemptions) setExemptions(data.exemptions);
        }
        setLoading(false);
      })
      .catch(e => { setError(e.message); setLoading(false); });
  }, []);

  const actionColors: Record<string, string> = { block: "bg-red-100 text-red-700", challenge: "bg-yellow-100 text-yellow-700", log: "bg-gray-100 text-gray-600" };

  if (loading) return <div className="p-8"><p>{t("big1.geoVelocityRules.loading")}</p></div>;
  if (error) return <div className="p-8 text-red-600">{t("big1.geoVelocityRules.error")}{error}</div>;

  return (
    <div className="p-8 space-y-6 max-w-5xl">
      <h1 className="text-2xl font-bold">{t("big1.geoVelocityRules.title")}</h1>
      <p className="text-gray-600">{t("big1.geoVelocityRules.detectImpossibleTravelAndAnomalousGeographicPatterns")}</p>

      <div className="bg-white rounded-lg p-6 shadow"><h2 className="text-lg font-semibold mb-4">{t("big1.geoVelocityRules.activeRules")}</h2><table className="w-full text-sm"><thead><tr className="border-b text-left"><th className="py-2">{t("big1.geoVelocityRules.name")}</th><th scope="col">{t("big1.geoVelocityRules.distanceKm")}</th><th>{t("big1.geoVelocityRules.timeH")}</th><th>{t("big1.geoVelocityRules.action")}</th><th>{t("big1.geoVelocityRules.enabled")}</th></tr></thead><tbody>{rules.map((r: VelocityRule, i: number) => (<tr key={i} className="border-b"><td className="py-2 font-medium">{r.name}</td><td>{r.distance_km}</td><td>{r.time_hours}</td><td><span className={`px-2 py-1 rounded text-xs ${actionColors[r.action] || ""}`}>{r.action}</span></td><td><input type="checkbox" checked={r.enabled} onChange={(e) => { const next = [...rules]; next[i] = { ...r, enabled: e.target.checked }; setRules(next); }} className="w-4 h-4" /></td></tr>))}</tbody></table></div>

      <div className="bg-white rounded-lg p-6 shadow"><h2 className="text-lg font-semibold mb-4">{t("big1.geoVelocityRules.velocityMapPreview")}</h2><table className="w-full text-sm"><thead><tr className="border-b text-left"><th className="py-2">{t("big1.geoVelocityRules.from")}</th><th scope="col">{t("big1.geoVelocityRules.to")}</th><th>{t("big1.geoVelocityRules.distance")}</th><th>{t("big1.geoVelocityRules.time")}</th><th>{t("big1.geoVelocityRules.triggered")}</th></tr></thead><tbody>{samplePairs.map((p, i) => (<tr key={i} className={`border-b ${p.triggered ? "bg-red-50" : ""}`}><td className="py-2">{p.from}</td><td>{p.to}</td><td>{p.distance_km}{t("big1.geoVelocityRules.km")}</td><td>{p.time_hours}{t("big1.geoVelocityRules.h")}</td><td>{p.triggered ? <span className="text-red-600 font-medium text-xs">{t("big1.geoVelocityRules.yes")}</span> : <span className="text-green-600 text-xs">{t("big1.geoVelocityRules.no")}</span>}</td></tr>))}</tbody></table></div>

      <div className="bg-white rounded-lg p-6 shadow space-y-4"><h2 className="text-lg font-semibold">{t("big1.geoVelocityRules.testSimulation")}</h2><div className="flex gap-3 items-end"><div className="flex-1"><label className="block text-sm font-medium mb-1">{t("big1.geoVelocityRules.locationA")}</label><input type="text" value={simFrom} onChange={(e) => setSimFrom(e.target.value)} placeholder="San Francisco" className="border rounded px-3 py-2 w-full text-sm" /></div><div className="flex-1"><label className="block text-sm font-medium mb-1">{t("big1.geoVelocityRules.locationB")}</label><input type="text" value={simTo} onChange={(e) => setSimTo(e.target.value)} placeholder="Tokyo" className="border rounded px-3 py-2 w-full text-sm" /></div><button onClick={() => setSimResult(`Rule "Impossible Travel" would TRIGGER: impossible travel from ${simFrom || "A"} to ${simTo || "B"}`)} className="px-4 py-2 bg-blue-600 text-white rounded hover:bg-blue-700">{t("big1.geoVelocityRules.simulate")}</button></div>{simResult && <div className="bg-yellow-50 border border-yellow-300 rounded p-3 text-sm text-yellow-800">{simResult}</div>}</div>

      <div className="bg-white rounded-lg p-6 shadow"><h2 className="text-lg font-semibold mb-2">{t("big1.geoVelocityRules.exemptions")}</h2><div className="flex flex-wrap gap-2">{exemptions.map((e: string, i: number) => <span key={i} className="px-2 py-1 bg-gray-100 rounded text-sm font-mono">{e}</span>)}</div></div>
    </div>
  );
}
