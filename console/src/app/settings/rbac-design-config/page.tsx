"use client";
import { useEffect, useState } from "react";
import { useRbacDesignConfig, SodPair } from "@ggid/sdk-react";
import { useTranslations } from "@/lib/i18n";

interface LocalRoleNode {
  level: number;
  name: string;
  parent?: string;
}

interface LocalRoleTemplate {
  name: string;
  description: string;
  permissions: string[];
}

interface LocalRbacDesignConfig {
  inheritance_enabled: boolean;
  auto_inherit_from_parent: boolean;
  max_depth: number;
  delegation_max_depth: number;
  role_hierarchy: LocalRoleNode[];
  role_templates: LocalRoleTemplate[];
  sod_pairs: SodPair[];
}

export default function RbacDesignConfigPage() {
  const t = useTranslations();
  const { config, loading, error, fetchConfig, updateConfig } = useRbacDesignConfig();
  const [form, setForm] = useState<LocalRbacDesignConfig | null>(null);
  const [saving, setSaving] = useState(false);

  useEffect(() => { fetchConfig(); }, [fetchConfig]);
  useEffect(() => { if (config) setForm(config as unknown as LocalRbacDesignConfig); }, [config]);

  const handleSave = async () => { if (!form) return; setSaving(true); await updateConfig(form as unknown as Parameters<typeof updateConfig>[0]); setSaving(false); };

  if (loading && !form) return <div className="p-8">Loading...</div>;
  if (error) return <div className="p-8 text-red-600">Error: {error}</div>;
  if (!form) return <div className="p-8">No data</div>;

  return (
    <div className="p-8 space-y-6 max-w-4xl">
      <h1 className="text-2xl font-bold">RBAC Design Configuration</h1>
      <p className="text-gray-600">Configure Role-Based Access Control hierarchy, inheritance, and Segregation of Duties.</p>

      <div className="bg-white rounded-lg p-6 shadow space-y-4">
        <h2 className="text-lg font-semibold">Hierarchy Settings</h2>
        <div className="flex items-center gap-3">
          <input type="checkbox" checked={form.inheritance_enabled}
            onChange={(e) => setForm({ ...form, inheritance_enabled: e.target.checked })}
            className="w-4 h-4" />
          <label>Inheritance Enabled</label>
        </div>
        <div className="flex items-center gap-3">
          <input type="checkbox" checked={form.auto_inherit_from_parent}
            onChange={(e) => setForm({ ...form, auto_inherit_from_parent: e.target.checked })}
            className="w-4 h-4" />
          <label>Auto Inherit from Parent</label>
        </div>
        <div>
          <label className="block text-sm font-medium mb-1">Max Depth</label>
          <input type="number" value={form.max_depth}
            onChange={(e) => setForm({ ...form, max_depth: parseInt(e.target.value) || 0 })}
            className="border rounded px-3 py-2 w-32" />
        </div>
        <div>
          <label className="block text-sm font-medium mb-1">Delegation Max Depth</label>
          <input type="number" value={form.delegation_max_depth}
            onChange={(e) => setForm({ ...form, delegation_max_depth: parseInt(e.target.value) || 0 })}
            className="border rounded px-3 py-2 w-32" />
        </div>
      </div>

      <div className="bg-white rounded-lg p-6 shadow">
        <h2 className="text-lg font-semibold mb-4">Role Hierarchy</h2>
        <div className="space-y-1">
          {form.role_hierarchy.map((r, i) => (
            <div key={i} className="flex items-center py-1" style={{ paddingLeft: `${r.level * 24}px` }}>
              <span className="text-gray-400 mr-2">{r.level > 0 ? "|-" : ""}</span>
              <span className="font-medium">{r.name}</span>
              {r.parent && <span className="ml-2 text-xs text-gray-500">(parent: {r.parent})</span>}
            </div>
          ))}
        </div>
      </div>

      <div className="bg-white rounded-lg p-6 shadow">
        <h2 className="text-lg font-semibold mb-4">Role Templates</h2>
        <div className="space-y-3">
          {form.role_templates.map((t, i) => (
            <div key={i} className="border-b pb-2">
              <div className="font-medium">{t.name}</div>
              <div className="text-sm text-gray-600">{t.description}</div>
              <div className="text-xs text-blue-600">Permissions: {t.permissions.join(", ")}</div>
            </div>
          ))}
        </div>
      </div>

      <div className="bg-white rounded-lg p-6 shadow">
        <h2 className="text-lg font-semibold mb-4">Segregation of Duties</h2>
        <table className="w-full text-sm">
          <thead>
            <tr className="border-b text-left">
              <th className="py-2">Role A</th>
              <th>Role B</th>
              <th>Description</th>
            </tr>
          </thead>
          <tbody>
            {form.sod_pairs.map((p, i) => (
              <tr key={i} className="border-b">
                <td className="py-2">{p.role_a}</td>
                <td>{p.role_b}</td>
                <td>{p.description}</td>
              </tr>
            ))}
          </tbody>
        </table>
      </div>

      <button onClick={handleSave} disabled={saving} aria-label="Save RBAC design changes" className="px-6 py-2 bg-blue-600 text-white rounded-lg hover:bg-blue-700 disabled:opacity-50">{saving ? "Saving..." : "Save Changes"}</button>
    </div>
  );
}
