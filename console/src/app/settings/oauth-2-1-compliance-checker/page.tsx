"use client";
import { useEffect, useState } from "react";
import { Loader2 } from "lucide-react";
import { useOAuth21ComplianceChecker, ComplianceCheckItem } from "@ggid/sdk-react";
import { useTranslations } from "@/lib/i18n";

interface LocalNonCompliantClient {
  client_id: string;
  client_name: string;
  issues: string[];
}

interface LocalRemediationAction {
  action: string;
  priority: "high" | "medium" | "low";
  description: string;
}

interface LocalComplianceChecker {
  overall_pct: number;
  checklist: ComplianceCheckItem[];
  non_compliant_clients: LocalNonCompliantClient[];
  remediation_actions: LocalRemediationAction[];
}

export default function OAuth21ComplianceCheckerPage() {
  const t = useTranslations();
  const { config, loading, error, fetchConfig } = useOAuth21ComplianceChecker();
  const [form, setForm] = useState<LocalComplianceChecker | null>(null);
  useEffect(() => { fetchConfig(); }, [fetchConfig]);
  useEffect(() => { if (config) setForm(config as unknown as LocalComplianceChecker); }, [config]);
  if (loading && !form) return <div className="flex justify-center py-12"><Loader2 className="h-8 w-8 animate-spin text-indigo-500" /></div>;
  if (error) return <div className="p-8 text-red-600">Error: {error}</div>;
  if (!form) return <div className="p-8 text-gray-400">No data</div>;
  const statusColors: Record<string, string> = { pass: "bg-green-100 text-green-700", fail: "bg-red-100 text-red-700", warn: "bg-yellow-100 text-yellow-700" };

  return (
    <div className="p-8 space-y-6 max-w-4xl">
      <h1 className="text-2xl font-bold">OAuth 2.1 Compliance Checker</h1>
      <p className="text-gray-600">Check your OAuth configuration against the OAuth 2.1 draft.</p>
      <div className="bg-white rounded-lg p-6 shadow">
        <div className="flex items-center gap-4 mb-4">
          <div className="text-3xl font-bold text-blue-600">{form.overall_pct}%</div>
          <div className="flex-1 bg-gray-200 rounded-full h-4"><div className="bg-blue-600 h-4 rounded-full" style={{ width: `${form.overall_pct}%` }} /></div>
        </div>
      </div>
      <div className="bg-white rounded-lg p-6 shadow"><h2 className="text-lg font-semibold mb-4">Checklist</h2><table className="w-full text-sm"><thead><tr className="border-b text-left"><th className="py-2">Item</th><th scope="col">Status</th><th>Detail</th></tr></thead><tbody>{form.checklist.map((c: any, i: number) => (<tr key={i} className="border-b"><td className="py-2 font-medium">{c.item}</td><td><span className={`px-2 py-1 rounded text-xs ${statusColors[c.status] || ""}`}>{c.status}</span></td><td className="text-xs">{c.detail}</td></tr>))}</tbody></table></div>
      <div className="bg-white rounded-lg p-6 shadow"><h2 className="text-lg font-semibold mb-4">Non-Compliant Clients</h2><div className="space-y-2">{form.non_compliant_clients.map((c: any, i: number) => (<div key={i} className="border-b py-2"><span className="font-medium">{c.client_name}</span><div className="text-xs text-gray-500">{c.client_id}: {c.issues.join(", ")}</div></div>))}</div></div>
      <div className="bg-white rounded-lg p-6 shadow"><h2 className="text-lg font-semibold mb-4">Remediation Actions</h2><table className="w-full text-sm"><thead><tr className="border-b text-left"><th className="py-2">Action</th><th scope="col">Priority</th></tr></thead><tbody>{form.remediation_actions.map((r: any, i: number) => (<tr key={i} className="border-b"><td className="py-2">{r.action}</td><td><span className={`px-2 py-1 rounded text-xs ${r.priority === "high" ? "bg-red-100 text-red-700" : r.priority === "medium" ? "bg-yellow-100 text-yellow-700" : "bg-gray-100 text-gray-500"}`}>{r.priority}</span></td></tr>))}</tbody></table></div>
    </div>
  );
}
