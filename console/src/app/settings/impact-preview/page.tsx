"use client";

import { useState } from "react";
import { Eye, AlertTriangle, CheckCircle, Users, FileWarning } from "lucide-react";
import { useTranslations } from "@/lib/i18n";

interface AffectedItem {
  id: string;
  type: "user" | "resource" | "role";
  name: string;
  current_access: string;
  projected_access: string;
  risk_level: "low" | "medium" | "high";
}

interface PreviewResult {
  total_affected: number;
  gain_access: number;
  lose_access: number;
  no_change: boolean;
  items: AffectedItem[];
}

interface Policy {
  id: string;
  name: string;
}

const riskColors: Record<string, string> = {
  low: "bg-gray-100 text-gray-800 dark:bg-gray-800 dark:text-gray-400",
  medium: "bg-yellow-100 text-yellow-800 dark:bg-yellow-900/30 dark:text-yellow-400",
  high: "bg-red-100 text-red-800 dark:bg-red-900/30 dark:text-red-400",
};

export default function ImpactPreviewPage() {
  const t = useTranslations();

  const [policies, setPolicies] = useState<Policy[]>([{ id: "p1", name: "Data Access Policy" }, { id: "p2", name: "Admin Access Policy" }, { id: "p3", name: "External Partner Policy" }]);
  const [policyId, setPolicyId] = useState("");
  const [changeDesc, setChangeDesc] = useState("");
  const [result, setResult] = useState<PreviewResult | null>(null);
  const [loading, setLoading] = useState(false);

  const preview = async () => {
    if (!policyId) return;
    setLoading(true);
    try {
      const res = await fetch("/api/v1/policy/impact-preview", { method: "POST", headers: { "Authorization": `Bearer ${localStorage.getItem("ggid_access_token") || ""}`, "Content-Type": "application/json", "X-Tenant-ID": "00000000-0000-0000-0000-000000000001" }, body: JSON.stringify({ policy_id: policyId, changes: changeDesc }) });
      if (res.ok) setResult(await res.json());
    } catch { /* noop */ }
    finally { setLoading(false); }
  };

  return (
    <div className="space-y-6">
      <div>
        <h1 className="text-2xl font-bold flex items-center gap-2"><Eye className="w-6 h-6 text-blue-500" /> {t("impactPreview.title")}</h1>
        <p className="text-sm text-gray-500 mt-1">Preview the impact of proposed policy changes before applying them.</p>
      </div>

      <div className="rounded-lg border dark:border-gray-800 p-4 space-y-3">
        <div className="grid grid-cols-1 md:grid-cols-2 gap-3">
          <div><label className="text-sm font-medium">Policy</label><select aria-label="Policy id" value={policyId} onChange={(e) => setPolicyId(e.target.value)} className="w-full mt-1 px-3 py-2 rounded-lg border dark:border-gray-700 dark:bg-gray-900 text-sm"><option value="">Select Policy</option>{policies.map((p) => <option key={p.id} value={p.id}>{p.name}</option>)}</select></div>
          <div><label className="text-sm font-medium">Proposed Change</label><input aria-label="e.g. add resource:finance:* to role:analyst" type="text" value={changeDesc} onChange={(e) => setChangeDesc(e.target.value)} placeholder="e.g. add resource:finance:* to role:analyst" className="w-full mt-1 px-3 py-2 rounded-lg border dark:border-gray-700 dark:bg-gray-900 text-sm font-mono" /></div>
        </div>
        <button aria-label="Eye" onClick={preview} disabled={loading || !policyId} className="px-4 py-2 rounded-lg bg-blue-600 text-white text-sm font-medium hover:bg-blue-700 disabled:opacity-50 flex items-center gap-2"><Eye className="w-4 h-4" /> {loading ? "Analyzing..." : "Preview Impact"}</button>
      </div>

      {result && (
        <>
          {result.no_change ? (
            <div className="rounded-lg border border-green-300 dark:border-green-800 bg-green-50 dark:bg-green-900/20 p-4 flex items-center gap-3"><CheckCircle className="w-8 h-8 text-green-500" /><div><span className="font-semibold text-green-700 dark:text-green-400">No Impact Detected</span><p className="text-sm text-gray-500">The proposed change will not affect any users or resources.</p></div></div>
          ) : (
            <>
              <div className="grid grid-cols-3 gap-4">
                <div className="rounded-lg border p-4 dark:border-gray-800 flex items-center gap-3"><Users className="w-8 h-8 text-blue-500" /><div><span className="text-sm text-gray-500">Total Affected</span><p className="text-xl font-bold">{result.total_affected}</p></div></div>
                <div className="rounded-lg border p-4 dark:border-gray-800 flex items-center gap-3"><AlertTriangle className="w-8 h-8 text-green-500" /><div><span className="text-sm text-gray-500">Gain Access</span><p className="text-xl font-bold text-green-600">+{result.gain_access}</p></div></div>
                <div className="rounded-lg border p-4 dark:border-gray-800 flex items-center gap-3"><FileWarning className="w-8 h-8 text-red-500" /><div><span className="text-sm text-gray-500">Lose Access</span><p className="text-xl font-bold text-red-600">-{result.lose_access}</p></div></div>
              </div>

              <div className="overflow-x-auto rounded-lg border dark:border-gray-800">
                <table className="w-full text-sm"><thead className="bg-gray-50 dark:bg-gray-900/50"><tr><th className="px-4 py-3 text-left font-medium">Type</th><th className="px-4 py-3 text-left font-medium">Name</th><th className="px-4 py-3 text-left font-medium">Current</th><th className="px-4 py-3 text-left font-medium">Projected</th><th className="px-4 py-3 text-left font-medium">Risk</th></tr></thead>
                  <tbody className="divide-y dark:divide-gray-800">{result.items.map((item) => (<tr key={item.id} className="hover:bg-gray-50 dark:hover:bg-gray-900/30"><td className="px-4 py-3"><span className="text-xs font-medium uppercase text-gray-500">{item.type}</span></td><td className="px-4 py-3 font-medium">{item.name}</td><td className="px-4 py-3 text-xs text-gray-500">{item.current_access}</td><td className="px-4 py-3 text-xs font-medium">{item.projected_access}</td><td className="px-4 py-3"><span className={`px-2 py-0.5 rounded text-xs ${riskColors[item.risk_level]}`}>{item.risk_level}</span></td></tr>))}</tbody>
                </table>
              </div>
            </>
          )}
        </>
      )}
    </div>
  );
}
