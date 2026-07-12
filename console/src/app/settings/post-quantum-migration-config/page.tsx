"use client";
import { useEffect, useState } from "react";
import { usePostQuantumMigrationConfig, PostQuantumMigrationConfig, ImpactAssessment } from "@ggid/sdk-react";

export default function PostQuantumMigrationConfigPage() {
  const { config, loading, error, fetchConfig, updateConfig } = usePostQuantumMigrationConfig();
  const [form, setForm] = useState<PostQuantumMigrationConfig | null>(null);
  const [saving, setSaving] = useState(false);
  useEffect(() => { fetchConfig(); }, [fetchConfig]); useEffect(() => { if (config) setForm(config); }, [config]);
  const handleSave = async () => { if (!form) return; setSaving(true); await updateConfig(form); setSaving(false); };
  if (loading && !form) return <div className="p-8">Loading...</div>;
  if (error) return <div className="p-8 text-red-600">Error: {error}</div>;
  if (!form) return <div className="p-8">No data</div>;
  const diffColors: Record<string, string> = { easy: "bg-green-100 text-green-700", medium: "bg-yellow-100 text-yellow-700", hard: "bg-red-100 text-red-700" };

  return (
    <div className="p-8 space-y-6 max-w-4xl">
      <h1 className="text-2xl font-bold">Post-Quantum Cryptography Migration</h1>
      <p className="text-gray-600">Plan and execute migration to quantum-resistant algorithms.</p>
      <div className="bg-white rounded-lg p-6 shadow space-y-4"><h2 className="text-lg font-semibold">Algorithm Strategy</h2><div><label className="block text-sm font-medium mb-1">Current Algorithms</label><div className="text-sm text-gray-600">{form.current_algs.join(", ")}</div></div><div><label className="block text-sm font-medium mb-1">Target Algorithms</label><div className="text-sm text-gray-600">{form.target_algs.join(", ")}</div></div><div className="flex items-center gap-3"><input type="checkbox" checked={form.hybrid_mode} onChange={(e) => setForm({ ...form, hybrid_mode: e.target.checked })} className="w-4 h-4" /><label>Hybrid Mode (classical + PQC)</label></div><div><label className="block text-sm font-medium mb-1">Migration Timeline (weeks)</label><input type="number" value={form.migration_timeline_weeks} onChange={(e) => setForm({ ...form, migration_timeline_weeks: parseInt(e.target.value) || 0 })} className="border rounded px-3 py-2 w-32" /></div><div className="flex items-center gap-3"><input type="checkbox" checked={form.test_toggle} onChange={(e) => setForm({ ...form, test_toggle: e.target.checked })} className="w-4 h-4" /><label>Enable PQC Test Mode</label></div></div>
      <div className="bg-white rounded-lg p-6 shadow"><h2 className="text-lg font-semibold mb-4">Impact Assessment Per Component</h2><table className="w-full text-sm"><thead><tr className="border-b text-left"><th className="py-2">Component</th><th>Current</th><th>Target</th><th>Difficulty</th></tr></thead><tbody>{form.impact_assessment.map((a: ImpactAssessment, i: number) => (<tr key={i} className="border-b"><td className="py-2 font-medium">{a.component}</td><td className="font-mono text-xs">{a.current_alg}</td><td className="font-mono text-xs">{a.target_alg}</td><td><span className={`px-2 py-1 rounded text-xs ${diffColors[a.migration_difficulty] || ""}`}>{a.migration_difficulty}</span></td></tr>))}</tbody></table></div>
      <button onClick={handleSave} disabled={saving} className="px-6 py-2 bg-blue-600 text-white rounded-lg hover:bg-blue-700 disabled:opacity-50">{saving ? "Saving..." : "Save Changes"}</button>
    </div>
  );
}
