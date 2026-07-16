"use client";
import { useEffect, useState } from "react";
import { useAuditTamperDetectionConfig, AuditTamperDetectionConfig } from "@ggid/sdk-react";
import { useTranslations } from "@/lib/i18n";

export default function AuditTamperDetectionConfigPage() {
  const t = useTranslations();
  const { config, loading, error, fetchConfig, updateConfig } = useAuditTamperDetectionConfig();
  const [form, setForm] = useState<AuditTamperDetectionConfig | null>(null);
  const [saving, setSaving] = useState(false);

  useEffect(() => { fetchConfig(); }, [fetchConfig]);
  useEffect(() => { if (config) setForm(config); }, [config]);

  const handleSave = async () => { if (!form) return; setSaving(true); await updateConfig(form); setSaving(false); };

  if (loading && !form) return <div className="p-8">Loading...</div>;
  if (error) return <div className="p-8 text-red-600">Error: {error}</div>;
  if (!form) return <div className="p-8">No data</div>;

  return (
    <div className="p-8 space-y-6 max-w-4xl">
      <h1 className="text-2xl font-bold">Audit Tamper Detection Configuration</h1>
      <p className="text-gray-600">Configure audit log integrity verification and tamper detection.</p>

      <div className="bg-white rounded-lg p-6 shadow space-y-4">
        <h2 className="text-lg font-semibold">Verification Settings</h2>
        <div><label className="block text-sm font-medium mb-1">Verify Interval (minutes)</label><input aria-label="form" type="number" value={form.verify_interval_minutes} onChange={(e) => setForm({ ...form, verify_interval_minutes: parseInt(e.target.value) || 0 })} className="border rounded px-3 py-2 w-32" /></div>
        <div><label className="block text-sm font-medium mb-1">Insertion Gap Threshold</label><input aria-label="form" type="number" value={form.insertion_gap_threshold} onChange={(e) => setForm({ ...form, insertion_gap_threshold: parseInt(e.target.value) || 0 })} className="border rounded px-3 py-2 w-32" /></div>
        <div><label className="block text-sm font-medium mb-1">Reorder Detection Sensitivity</label><select aria-label="form" value={form.reorder_detection_sensitivity} onChange={(e) => setForm({ ...form, reorder_detection_sensitivity: e.target.value as AuditTamperDetectionConfig["reorder_detection_sensitivity"] })} className="border rounded px-3 py-2"><option value="low">Low</option><option value="medium">Medium</option><option value="high">High</option></select></div>
      </div>

      <div className="bg-white rounded-lg p-6 shadow space-y-3">
        <h2 className="text-lg font-semibold">Alerting & Forensics</h2>
        <div className="flex items-center gap-3"><input aria-label="Form" type="checkbox" checked={form.alert_on_tamper} onChange={(e) => setForm({ ...form, alert_on_tamper: e.target.checked })} className="w-4 h-4" /><label>Alert on Tamper Detection</label></div>
        <div className="flex items-center gap-3"><input aria-label="Form" type="checkbox" checked={form.forensics_auto_collection} onChange={(e) => setForm({ ...form, forensics_auto_collection: e.target.checked })} className="w-4 h-4" /><label>Forensics Auto-Collection</label></div>
      </div>

      <div className="bg-white rounded-lg p-6 shadow">
        <h2 className="text-lg font-semibold mb-4">Recovery Procedure Template</h2>
        <pre className="bg-gray-50 rounded p-4 text-xs overflow-x-auto whitespace-pre-wrap">{form.recovery_procedure_template}</pre>
      </div>

      <button aria-label="action" onClick={handleSave} disabled={saving} className="px-6 py-2 bg-blue-600 text-white rounded-lg hover:bg-blue-700 disabled:opacity-50">{saving ? "Saving..." : "Save Changes"}</button>
    </div>
  );
}
