"use client";

import { useState, useEffect, useCallback } from "react";
import { Shield, Eye, Clock } from "lucide-react";

interface Attribute {
  name: string;
  pii_classification: "public" | "internal" | "confidential" | "restricted";
  mask_rule: string;
  access_frequency: number;
  last_accessed_by: string;
  last_accessed_at: string;
  retention_days: number;
}

const piiColors: Record<string, string> = {
  public: "bg-gray-100 text-gray-800 dark:bg-gray-800 dark:text-gray-400",
  internal: "bg-blue-100 text-blue-800 dark:bg-blue-900/30 dark:text-blue-400",
  confidential: "bg-yellow-100 text-yellow-800 dark:bg-yellow-900/30 dark:text-yellow-400",
  restricted: "bg-red-100 text-red-800 dark:bg-red-900/30 dark:text-red-400",
};

export default function AttributeGovernancePage() {
  const [attrs, setAttrs] = useState<Attribute[]>([]);
  const [loading, setLoading] = useState(false);

  const fetchData = useCallback(async () => {
    setLoading(true);
    try {
      const res = await fetch("/api/v1/identity/attribute-governance", { headers: { "X-Tenant-ID": "00000000-0000-0000-0000-000000000001" } });
      if (res.ok) { const d = await res.json(); setAttrs(d.attributes || d || []); }
    } catch { /* noop */ }
    finally { setLoading(false); }
  }, []);

  useEffect(() => { fetchData(); }, [fetchData]);

  const restricted = attrs.filter((a) => a.pii_classification === "restricted" || a.pii_classification === "confidential").length;

  return (
    <div className="space-y-6">
      <div>
        <h1 className="text-2xl font-bold flex items-center gap-2"><Shield className="w-6 h-6 text-red-500" /> Attribute Governance</h1>
        <p className="text-sm text-gray-500 mt-1">Track sensitive user attributes with PII classification and access controls.</p>
      </div>

      <div className="grid grid-cols-3 gap-4">
        <div className="rounded-lg border p-4 dark:border-gray-800"><span className="text-sm text-gray-500">Total Attributes</span><p className="text-xl font-bold mt-1">{attrs.length}</p></div>
        <div className="rounded-lg border p-4 dark:border-gray-800"><span className="text-sm text-gray-500">Confidential+</span><p className="text-xl font-bold text-red-600 mt-1">{restricted}</p></div>
        <div className="rounded-lg border p-4 dark:border-gray-800"><span className="text-sm text-gray-500">Masked</span><p className="text-xl font-bold text-green-600 mt-1">{attrs.filter((a) => a.mask_rule !== "none").length}</p></div>
      </div>

      <div className="overflow-x-auto rounded-lg border dark:border-gray-800">
        <table className="w-full text-sm"><thead className="bg-gray-50 dark:bg-gray-900/50"><tr><th className="px-4 py-3 text-left font-medium">Attribute</th><th className="px-4 py-3 text-left font-medium">PII Class</th><th className="px-4 py-3 text-left font-medium">Mask Rule</th><th className="px-4 py-3 text-left font-medium">Access Freq</th><th className="px-4 py-3 text-left font-medium">Last Accessed By</th><th className="px-4 py-3 text-left font-medium">Retention</th></tr></thead>
          <tbody className="divide-y dark:divide-gray-800">{attrs.map((a) => (
            <tr key={a.name} className="hover:bg-gray-50 dark:hover:bg-gray-900/30"><td className="px-4 py-3 font-mono text-xs font-medium">{a.name}</td><td className="px-4 py-3"><span className={`px-2 py-0.5 rounded text-xs ${piiColors[a.pii_classification]}`}>{a.pii_classification}</span></td><td className="px-4 py-3"><span className="flex items-center gap-1 text-xs"><Eye className="w-3 h-3 text-gray-400" /><span className="font-mono">{a.mask_rule}</span></span></td><td className="px-4 py-3"><span className="font-bold">{a.access_frequency}</span><span className="text-xs text-gray-400 ml-1">/mo</span></td><td className="px-4 py-3"><div><span className="text-xs font-mono">{a.last_accessed_by}</span><p className="text-xs text-gray-400 flex items-center gap-1"><Clock className="w-3 h-3" />{a.last_accessed_at}</p></div></td><td className="px-4 py-3 text-xs text-gray-500">{a.retention_days}d</td></tr>
          ))}{attrs.length === 0 && !loading && <tr><td colSpan={6} className="px-4 py-8 text-center text-gray-500">No attributes found.</td></tr>}</tbody>
        </table>
      </div>
    </div>
  );
}
