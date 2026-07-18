"use client";
import { useEffect, useState } from "react";
import { useAbacConditionConfig, AttributeSource } from "@ggid/sdk-react";
import { useTranslations } from "@/lib/i18n";

interface LocalConditionTemplate {
  name: string;
  description: string;
  expression: string;
}

interface LocalAbacConditionConfig {
  attribute_sources: AttributeSource[];
  operators_per_type: Record<string, string[]>;
  condition_templates: LocalConditionTemplate[];
  evaluation_cache_ttl: number;
  default_deny: boolean;
}

export default function AbacConditionConfigPage() {
  const t = useTranslations();

  const { config, loading, error, fetchConfig, updateConfig } = useAbacConditionConfig();
  const [form, setForm] = useState<LocalAbacConditionConfig | null>(null);
  const [saving, setSaving] = useState(false);

  useEffect(() => { fetchConfig(); }, [fetchConfig]);
  useEffect(() => { if (config) setForm(config as unknown as LocalAbacConditionConfig); }, [config]);

  const handleSave = async () => { if (!form) return; setSaving(true); await updateConfig(form as unknown as Parameters<typeof updateConfig>[0]); setSaving(false); };

  if (loading && !form) return <div className="p-8">Loading...</div>;
  if (error) return <div className="p-8 text-red-600">Error: {error}</div>;
  if (!form) return <div className="p-8">No data</div>;

  const operatorEntries: [string, string[]][] = Object.entries(form.operators_per_type);

  return (
    <div className="p-8 space-y-6 max-w-4xl">
      <h1 className="text-2xl font-bold">ABAC Condition Configuration</h1>
      <p className="text-gray-600">Configure Attribute-Based Access Control conditions, operators, and templates.</p>

      <div className="bg-white rounded-lg p-6 shadow">
        <h2 className="text-lg font-semibold mb-4">Attribute Sources</h2>
        <div className="grid grid-cols-2 gap-4">
          {form.attribute_sources.map((src: AttributeSource, i: number) => (
            <div key={i} className="border rounded p-3">
              <div className="font-medium capitalize mb-1">{src.category}</div>
              <div className="text-sm text-gray-500">{src.attributes.join(", ")}</div>
            </div>
          ))}
        </div>
      </div>

      <div className="bg-white rounded-lg p-6 shadow">
        <h2 className="text-lg font-semibold mb-4">Operators Per Type</h2>
        <table className="w-full text-sm">
          <thead><tr className="border-b text-left"><th className="py-2">Type</th><th scope="col">Operators</th></tr></thead>
          <tbody>
            {operatorEntries.map(([type, ops], i) => {
              const typedOps: string[] = Array.isArray(ops) ? ops : [];
              return (
                <tr key={i} className="border-b"><td className="py-2 font-medium">{type}</td><td>{typedOps.join(", ")}</td></tr>
              );
            })}
          </tbody>
        </table>
      </div>

      <div className="bg-white rounded-lg p-6 shadow">
        <h2 className="text-lg font-semibold mb-4">Condition Templates</h2>
        <div className="space-y-3">
          {form.condition_templates.map((t: any, i: number) => (
            <div key={i} className="border-b pb-2">
              <div className="font-medium">{t.name}</div>
              <div className="text-sm text-gray-600">{t.description}</div>
              <code className="text-xs text-blue-600">{t.expression}</code>
            </div>
          ))}
        </div>
      </div>

      <div className="bg-white rounded-lg p-6 shadow space-y-4">
        <h2 className="text-lg font-semibold">Evaluation Settings</h2>
        <div>
          <label className="block text-sm font-medium mb-1">Evaluation Cache TTL (seconds)</label>
          <input aria-label="form" type="number" value={form.evaluation_cache_ttl}
            onChange={(e) => setForm({ ...form, evaluation_cache_ttl: parseInt(e.target.value) || 0 })}
            className="border rounded px-3 py-2 w-48" />
        </div>
        <div className="flex items-center gap-3">
          <input aria-label="Toggle" type="checkbox" checked={form.default_deny}
            onChange={(e) => setForm({ ...form, default_deny: e.target.checked })}
            className="w-4 h-4" />
          <label>Default Deny (deny if no policy matches)</label>
        </div>
      </div>

      <button onClick={handleSave} disabled={saving} aria-label="Save ABAC condition config" className="px-6 py-2 bg-blue-600 text-white rounded-lg hover:bg-blue-700 disabled:opacity-50">{saving ? "Saving..." : "Save Changes"}</button>
    </div>
  );
}
