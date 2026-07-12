"use client";
import { useEffect, useState } from "react";
import { useComplianceAutomationConfig, ComplianceAutomationConfig, ContinuousMonitoringRule, FrameworkMapping, RemediationTrigger } from "@ggid/sdk-react";

export default function ComplianceAutomationConfigPage() {
  const { config, loading, error, fetchConfig, updateConfig } = useComplianceAutomationConfig();
  const [form, setForm] = useState<ComplianceAutomationConfig | null>(null);
  const [saving, setSaving] = useState(false);
  useEffect(() => { fetchConfig(); }, [fetchConfig]); useEffect(() => { if (config) setForm(config); }, [config]);
  const handleSave = async () => { if (!form) return; setSaving(true); await updateConfig(form); setSaving(false); };
  if (loading && !form) return <div className="p-8">Loading...</div>;
  if (error) return <div className="p-8 text-red-600">Error: {error}</div>;
  if (!form) return <div className="p-8">No data</div>;

  return (
    <div className="p-8 space-y-6 max-w-4xl">
      <h1 className="text-2xl font-bold">Compliance Automation Configuration</h1>
      <p className="text-gray-600">Configure automated evidence collection, monitoring, and audit readiness.</p>
      <div className="bg-white rounded-lg p-6 shadow space-y-4"><h2 className="text-lg font-semibold">General</h2><div><label className="block text-sm font-medium mb-1">Evidence Collection Schedule (cron)</label><input type="text" value={form.evidence_collection_schedule} onChange={(e) => setForm({ ...form, evidence_collection_schedule: e.target.value })} className="border rounded px-3 py-2 w-full font-mono text-sm" /></div><div className="flex items-center gap-3"><input type="checkbox" checked={form.drift_detection} onChange={(e) => setForm({ ...form, drift_detection: e.target.checked })} className="w-4 h-4" /><label>Drift Detection</label></div><div><label className="block text-sm font-medium mb-2">Audit Readiness Score</label><div className="flex items-center gap-4"><div className="text-3xl font-bold text-green-600">{form.audit_readiness_score}%</div><div className="flex-1 bg-gray-200 rounded-full h-4"><div className="bg-green-600 h-4 rounded-full" style={{ width: `${form.audit_readiness_score}%` }} /></div></div></div></div>
      <div className="bg-white rounded-lg p-6 shadow"><h2 className="text-lg font-semibold mb-4">Continuous Monitoring Rules</h2><table className="w-full text-sm"><thead><tr className="border-b text-left"><th className="py-2">Control ID</th><th>Description</th><th>Frequency</th></tr></thead><tbody>{form.continuous_monitoring_rules.map((r: ContinuousMonitoringRule, i: number) => (<tr key={i} className="border-b"><td className="py-2 font-mono">{r.control_id}</td><td className="text-xs">{r.description}</td><td><span className="px-2 py-1 bg-blue-100 text-blue-700 rounded text-xs">{r.frequency}</span></td></tr>))}</tbody></table></div>
      <div className="bg-white rounded-lg p-6 shadow"><h2 className="text-lg font-semibold mb-4">Framework Mapping</h2><div className="grid grid-cols-2 gap-4">{form.framework_mapping.map((f: FrameworkMapping, i: number) => { const pct = f.controls_total > 0 ? Math.round((f.controls_met / f.controls_total) * 100) : 0; return (<div key={i} className="border rounded p-3"><div className="font-medium">{f.framework}</div><div className="text-sm text-gray-500">{f.controls_met}/{f.controls_total} controls ({pct}%)</div><div className="bg-gray-200 rounded-full h-2 mt-1"><div className="bg-blue-600 h-2 rounded-full" style={{ width: `${pct}%` }} /></div></div>); })}</div></div>
      <div className="bg-white rounded-lg p-6 shadow"><h2 className="text-lg font-semibold mb-4">Remediation Triggers</h2><table className="w-full text-sm"><thead><tr className="border-b text-left"><th className="py-2">Condition</th><th>Action</th></tr></thead><tbody>{form.remediation_triggers.map((r: RemediationTrigger, i: number) => (<tr key={i} className="border-b"><td className="py-2">{r.condition}</td><td className="text-xs">{r.action}</td></tr>))}</tbody></table></div>
      <button onClick={handleSave} disabled={saving} className="px-6 py-2 bg-blue-600 text-white rounded-lg hover:bg-blue-700 disabled:opacity-50">{saving ? "Saving..." : "Save Changes"}</button>
    </div>
  );
}
