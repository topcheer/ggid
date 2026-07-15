'use client';
import { useState, useEffect } from 'react';
import { useTranslations } from "@/lib/i18n";

interface Template { id: string; name: string; category: string; permissions: string[]; version: string; }

export default function RoleTemplatesConfigPage() {
  const t = useTranslations();

  const [templates, setTemplates] = useState<Template[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [categoryFilter, setCategoryFilter] = useState('all');
  const [showDiff, setShowDiff] = useState<Template | null>(null);
  const [newTemplate, setNewTemplate] = useState(false);

  useEffect(() => {
    fetch("/api/v1/policy/role-templates", {
      headers: { "Content-Type": "application/json", "X-Tenant-ID": "00000000-0000-0000-0000-000000000001" },
    })
      .then(res => { if (!res.ok) return null; return res.json(); })
      .then(data => { setTemplates(data.templates || data.items || []); setLoading(false); })
      .catch(err => { setError(err.message); setLoading(false); });
  }, []);

  const categories = [...new Set(templates.map(t => t.category))];
  const filtered = templates.filter(t => categoryFilter === 'all' || t.category === categoryFilter);

  if (loading) return <div className="p-6"><h1 className="text-2xl font-bold">Role Templates Configuration</h1><p className="text-gray-600 mt-2">Loading...</p></div>;
  if (error) return <div className="p-6"><h1 className="text-2xl font-bold">Role Templates Configuration</h1><p className="text-red-600 mt-2">Error: {error}</p></div>;

  return (
    <div className="p-6 max-w-5xl mx-auto space-y-6">
      <div className="flex items-center justify-between">
        <div><h1 className="text-2xl font-bold">Role Templates Configuration</h1><p className="text-gray-600">Create, apply, and version role templates with baseline permissions.</p></div>
        <button onClick={() => setNewTemplate(!newTemplate)} className="px-4 py-2 bg-blue-600 text-white rounded text-sm">{newTemplate ? 'Cancel' : 'Create Template'}</button>
      </div>

      {newTemplate && (
        <section className="bg-white rounded-lg shadow p-6 space-y-4">
          <h2 className="text-lg font-semibold">Create Template from Existing Role</h2>
          <div className="grid grid-cols-2 gap-4">
            <div><label className="text-sm font-medium">Template Name</label><input type="text" className="w-full border rounded px-3 py-2 text-sm mt-1" /></div>
            <div><label className="text-sm font-medium">Category</label><select className="w-full border rounded px-3 py-2 text-sm mt-1">{categories.map(c => <option key={c} value={c}>{c}</option>)}</select></div>
          </div>
          <div><label className="text-sm font-medium">Source Role</label><select className="w-full border rounded px-3 py-2 text-sm mt-1"><option>admin</option><option>developer</option><option>auditor</option></select></div>
          <button className="px-4 py-2 bg-blue-600 text-white rounded text-sm">Create</button>
        </section>
      )}

      <div className="flex gap-2"><button onClick={() => setCategoryFilter('all')} className={`px-3 py-1 rounded text-sm ${categoryFilter === 'all' ? 'bg-blue-600 text-white' : 'bg-gray-100'}`}>All</button>{categories.map(c => <button key={c} onClick={() => setCategoryFilter(c)} className={`px-3 py-1 rounded text-sm capitalize ${categoryFilter === c ? 'bg-blue-600 text-white' : 'bg-gray-100'}`}>{c}</button>)}</div>

      <section className="bg-white rounded-lg shadow overflow-hidden">
        {filtered.length === 0 ? <p className="p-6 text-center text-gray-500">No templates configured.</p> :
        <table className="w-full text-sm"><thead className="bg-gray-50"><tr className="text-left"><th className="p-3">Template</th><th className="p-3">Category</th><th className="p-3">Permissions</th><th className="p-3">Version</th><th className="p-3">Actions</th></tr></thead>
          <tbody>{filtered.map(t => (
            <tr key={t.id} className="border-b">
              <td className="p-3 font-medium">{t.name}</td><td className="p-3"><span className="px-2 py-0.5 bg-blue-100 text-blue-700 rounded text-xs capitalize">{t.category}</span></td>
              <td className="p-3"><div className="flex flex-wrap gap-1">{t.permissions.map(p => <span key={p} className="px-1.5 py-0.5 bg-gray-100 rounded text-xs font-mono">{p}</span>)}</div></td>
              <td className="p-3 text-gray-500">v{t.version}</td>
              <td className="p-3"><div className="flex gap-2"><button onClick={() => setShowDiff(t)} className="text-blue-600 text-xs hover:underline">Diff</button><button className="text-green-600 text-xs hover:underline">Apply</button></div></td>
            </tr>))}</tbody></table>}
      </section>

      {showDiff && (
        <div className="fixed inset-0 bg-black/50 flex items-center justify-center z-50">
          <div className="bg-white rounded-lg p-6 max-w-lg w-full mx-4 space-y-4">
            <div className="flex items-center justify-between"><h2 className="text-lg font-semibold">Template Diff: {showDiff.name}</h2><button onClick={() => setShowDiff(null)} className="text-gray-400">x</button></div>
            <div className="space-y-3"><div><div className="text-xs font-medium text-green-600">Baseline Permissions:</div><div className="flex flex-wrap gap-1 mt-1">{showDiff.permissions.map(p => <span key={p} className="px-2 py-0.5 bg-green-100 text-green-700 rounded text-xs font-mono">{p}</span>)}</div></div></div>
          </div>
        </div>
      )}
    </div>
  );
}
