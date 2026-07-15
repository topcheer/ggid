"use client";
import { useEffect, useState } from "react";
import { useSecretSprawlPreventionConfig, SecretSprawlPreventionConfig, SecretInventoryEntry } from "@ggid/sdk-react";
import { useTranslations } from "@/lib/i18n";

export default function SecretSprawlPreventionConfigPage() {
  const t = useTranslations();

  const { config, loading, error, fetchConfig, updateConfig } = useSecretSprawlPreventionConfig();
  const [form, setForm] = useState<SecretSprawlPreventionConfig | null>(null);
  const [saving, setSaving] = useState(false);
  useEffect(() => { fetchConfig(); }, [fetchConfig]); useEffect(() => { if (config) setForm(config); }, [config]);
  const handleSave = async () => { if (!form) return; setSaving(true); await updateConfig(form); setSaving(false); };
  if (loading && !form) return <div className="p-8">Loading...</div>;
  if (error) return <div className="p-8 text-red-600">Error: {error}</div>;
  if (!form) return <div className="p-8">No data</div>;
  const statusColors: Record<string, string> = { compliant: "bg-green-100 text-green-700", expiring: "bg-yellow-100 text-yellow-700", overdue: "bg-red-100 text-red-700" };

  return (
    <div className="p-8 space-y-6 max-w-4xl">
      <h1 className="text-2xl font-bold">Secret Sprawl Prevention</h1>
      <p className="text-gray-600">Detect and prevent secrets across code, configs, CI/CD, and runtime.</p>
      <div className="bg-white rounded-lg p-6 shadow space-y-4"><h2 className="text-lg font-semibold">Detection Settings</h2><div><label className="block text-sm font-medium mb-1">Scan Sources</label><div className="text-sm text-gray-600">{form.scan_sources.join(", ")}</div></div><div className="grid grid-cols-2 gap-4"><div><label className="block text-sm font-medium mb-1">Rotation Enforcement (days)</label><input type="number" value={form.rotation_enforcement_days} onChange={(e) => setForm({ ...form, rotation_enforcement_days: parseInt(e.target.value) || 0 })} className="border rounded px-3 py-2 w-full" /></div><div><label className="block text-sm font-medium mb-1">Vault Migration Status</label><div className="text-sm font-medium pt-2">{form.vault_migration_status}</div></div></div><div className="flex items-center gap-3"><input type="checkbox" checked={form.ci_detection} onChange={(e) => setForm({ ...form, ci_detection: e.target.checked })} className="w-4 h-4" /><label>CI Detection</label></div><div className="flex items-center gap-3"><input type="checkbox" checked={form.runtime_validation} onChange={(e) => setForm({ ...form, runtime_validation: e.target.checked })} className="w-4 h-4" /><label>Runtime Validation</label></div></div>
      <div className="bg-white rounded-lg p-6 shadow"><div className="flex items-center justify-between mb-4"><h2 className="text-lg font-semibold">Secret Inventory</h2><div className="text-sm">Violations (24h): <span className={`font-bold ${form.violations_24h > 0 ? "text-red-600" : "text-green-600"}`}>{form.violations_24h}</span></div></div><table className="w-full text-sm"><thead><tr className="border-b text-left"><th className="py-2">Name</th><th>Source</th><th>Last Rotated</th><th>Age (days)</th><th>Status</th></tr></thead><tbody>{form.secret_inventory.map((s: SecretInventoryEntry, i: number) => (<tr key={i} className="border-b"><td className="py-2 font-mono text-xs">{s.name}</td><td>{s.source}</td><td className="text-xs text-gray-500">{s.last_rotated}</td><td>{s.age_days}</td><td><span className={`px-2 py-1 rounded text-xs ${statusColors[s.status] || ""}`}>{s.status}</span></td></tr>))}</tbody></table></div>
      <button onClick={handleSave} disabled={saving} aria-label="Save secret sprawl prevention configuration" className="px-6 py-2 bg-blue-600 text-white rounded-lg hover:bg-blue-700 disabled:opacity-50">{saving ? "Saving..." : "Save Changes"}</button>
    </div>
  );
}
