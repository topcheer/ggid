"use client";

import { useSamlAttributeMappingConfig } from "@ggid/sdk-react";
import { ArrowRight, Plus } from "lucide-react";
import { useTranslations } from "@/lib/i18n";

export default function SamlAttributeMappingConfigPage() {
  const t = useTranslations();
  const { data, loading, error, refresh, testMapping } = useSamlAttributeMappingConfig();
  if (loading) return <div className="p-8 text-gray-400">Loading SAML attribute mapping...</div>;
  if (error) return <div className="p-8 text-red-400">Error: {error}</div>;

  return (
    <div className="min-h-screen bg-gray-950 text-white p-8">
      <div className="flex items-center justify-between mb-8">
        <div><h1 className="text-2xl font-bold">SAML Attribute Mapping</h1><p className="text-sm text-gray-400 mt-1">Map IdP attributes to local user fields</p></div>
        <div className="flex gap-2"><button className="flex items-center gap-1 px-3 py-2 bg-gray-700 hover:bg-gray-600 rounded-lg text-sm font-medium transition"><Plus className="w-4 h-4" /> Add Mapping</button><button onClick={refresh} className="px-4 py-2 bg-blue-600 hover:bg-blue-700 rounded-lg text-sm font-medium transition">Save</button></div>
      </div>

      <div className="bg-gray-900 rounded-xl p-6 mb-6">
        <h2 className="text-sm font-semibold mb-4">Mappings</h2>
        <table className="w-full text-sm"><thead><tr className="border-b border-gray-800 text-gray-400"><th className="text-left py-2">Source Attribute</th><th className="text-left py-2"></th><th className="text-left py-2">Target Field</th><th className="text-left py-2">Transform</th><th className="text-left py-2">Actions</th></tr></thead>
          <tbody>{(data?.mappings ?? []).map((m: any) => (
            <tr key={m.id} className="border-b border-gray-800">
              <td className="py-2 font-mono text-xs text-blue-400">{m.source_attribute}</td>
              <td className="py-2"><ArrowRight className="w-3 h-3 text-gray-600" /></td>
              <td className="py-2 font-mono text-xs text-green-400">{m.target_field}</td>
              <td className="py-2"><span className="text-xs px-2 py-0.5 rounded bg-gray-700">{m.transform_rule}</span></td>
              <td className="py-2"><button onClick={() => testMapping(m.id)} className="text-xs px-2 py-1 bg-blue-900 text-blue-300 rounded hover:bg-blue-800">Test</button></td>
            </tr>
          ))}</tbody>
        </table>
      </div>

      <div className="bg-gray-900 rounded-xl p-6">
        <h2 className="text-sm font-semibold mb-3">Per-IdP Override</h2>
        <table className="w-full text-sm"><thead><tr className="border-b border-gray-800 text-gray-400"><th className="text-left py-2">IdP</th><th className="text-left py-2">Overrides</th><th className="text-left py-2">Status</th></tr></thead>
          <tbody>{(data?.per_idp ?? []).map((o: any) => (
            <tr key={o.idp} className="border-b border-gray-800"><td className="py-2">{o.idp}</td><td className="py-2 text-xs text-gray-400">{o.override_count} mappings</td><td className="py-2"><span className="text-xs px-2 py-0.5 rounded bg-green-900 text-green-300">{o.status}</span></td></tr>
          ))}</tbody>
        </table>
      </div>
    </div>
  );
}
