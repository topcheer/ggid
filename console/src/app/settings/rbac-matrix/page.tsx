'use client';
import { useState, useEffect } from 'react';
import { useTranslations } from "@/lib/i18n";

interface Role { id: string; name: string; description: string; }

export default function RbacMatrixPage() {
  const t = useTranslations();
  const [roles, setRoles] = useState<Role[]>([]);
  const [permissions, setPermissions] = useState<string[]>([]);
  const [matrix, setMatrix] = useState<Record<string, boolean[]>>({});
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [showForm, setShowForm] = useState(false);
  const [newRole, setNewRole] = useState({ name: '', description: '' });
  const [selectedRole, setSelectedRole] = useState('');

  useEffect(() => {
    fetch("/api/v1/policy/rbac-matrix", {
      headers: { "Content-Type": "application/json", "X-Tenant-ID": "00000000-0000-0000-0000-000000000001" },
    })
      .then(res => { if (!res.ok) return null; return res.json(); })
      .then(data => {
        setRoles(data.roles || []);
        setPermissions(data.permissions || []);
        setMatrix(data.matrix || {});
        if (data.roles && data.roles.length > 0) setSelectedRole(data.roles[0].id);
        setLoading(false);
      })
      .catch(err => { setError(err.message); setLoading(false); });
  }, []);

  const toggleCell = (roleId: string, permIdx: number) => {
    setMatrix(prev => ({ ...prev, [roleId]: prev[roleId]?.map((v, i) => i === permIdx ? !v : v) || [] }));
  };

  const addRole = () => {
    fetch("/api/v1/policy/roles", {
      method: "POST",
      headers: { "Content-Type": "application/json", "X-Tenant-ID": "00000000-0000-0000-0000-000000000001" },
      body: JSON.stringify(newRole),
    })
      .then(res => { if (!res.ok) return null; return res.json(); })
      .then(data => {
        setRoles(prev => [...prev, data]);
        setMatrix(prev => ({ ...prev, [data.id]: new Array(permissions.length).fill(false) }));
        setShowForm(false); setNewRole({ name: '', description: '' });
      })
      .catch(err => setError(err.message));
  };

  const effectivePerms = matrix[selectedRole]?.map((v, i) => v ? permissions[i] : null).filter(Boolean) as string[] || [];

  if (loading) return <div className="p-6"><h1 className="text-2xl font-bold">RBAC Matrix</h1><p className="text-gray-600 mt-2">Loading...</p></div>;
  if (error) return <div className="p-6"><h1 className="text-2xl font-bold">RBAC Matrix</h1><p className="text-red-600 mt-2">Error: {error}</p></div>;

  return (
    <div className="p-6 max-w-5xl mx-auto space-y-6">
      <div className="flex items-center justify-between">
        <div><h1 className="text-2xl font-bold">RBAC Matrix</h1><p className="text-gray-600">Role-permission matrix with inheritance and bulk assignment.</p></div>
        <button onClick={() => setShowForm(!showForm)} className="px-4 py-2 bg-blue-600 text-white rounded text-sm">{showForm ? 'Cancel' : 'Create Role'}</button>
      </div>

      {showForm && (
        <section className="bg-white rounded-lg shadow p-6 space-y-4">
          <h2 className="text-lg font-semibold">Create Role</h2>
          <div className="grid grid-cols-2 gap-4">
            <div><label className="text-sm font-medium">Role Name</label><input type="text" value={newRole.name} onChange={e => setNewRole(prev => ({ ...prev, name: e.target.value }))} className="w-full border rounded px-3 py-2 text-sm mt-1" /></div>
            <div><label className="text-sm font-medium">Description</label><input type="text" value={newRole.description} onChange={e => setNewRole(prev => ({ ...prev, description: e.target.value }))} className="w-full border rounded px-3 py-2 text-sm mt-1" /></div>
          </div>
          <button onClick={addRole} disabled={!newRole.name} className="px-4 py-2 bg-blue-600 text-white rounded text-sm disabled:opacity-50">Create</button>
        </section>
      )}

      <section className="bg-white rounded-lg shadow overflow-hidden">
        {roles.length === 0 ? <p className="p-6 text-center text-gray-500">No roles configured.</p> :
        <table className="w-full text-sm">
          <thead className="bg-gray-50"><tr className="text-left"><th className="p-3">Role</th>{permissions.map(p => <th key={p} className="p-3 font-mono text-xs">{p}</th>)}</tr></thead>
          <tbody>
            {roles.map(r => (
              <tr key={r.id} className="border-b">
                <td className="p-3"><div className="font-medium">{r.name}</div><div className="text-xs text-gray-500">{r.description}</div></td>
                {matrix[r.id]?.map((v, i) => <td key={i} className="p-3 text-center"><input type="checkbox" checked={v} onChange={() => toggleCell(r.id, i)} className="rounded" /></td>)}
              </tr>
            ))}
          </tbody>
        </table>}
      </section>

      <section className="bg-white rounded-lg shadow p-6 space-y-4">
        <h2 className="text-lg font-semibold">Effective Permissions</h2>
        <select value={selectedRole} onChange={e => setSelectedRole(e.target.value)} className="border rounded px-3 py-2 text-sm">
          {roles.map(r => <option key={r.id} value={r.id}>{r.name}</option>)}
        </select>
        <div className="flex flex-wrap gap-2">{effectivePerms.map(p => <span key={p} className="px-2 py-1 bg-green-100 text-green-700 rounded text-xs font-mono">{p}</span>)}</div>
      </section>

      <section className="bg-white rounded-lg shadow p-6 space-y-4">
        <h2 className="text-lg font-semibold">Permission Catalog</h2>
        <div className="flex flex-wrap gap-2">{permissions.map(p => <span key={p} className="px-3 py-1 bg-blue-100 text-blue-700 rounded text-xs font-mono">{p}</span>)}</div>
      </section>
    </div>
  );
}
