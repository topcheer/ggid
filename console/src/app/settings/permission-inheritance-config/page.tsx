'use client';
import { useState } from 'react';

interface RoleInheritance {
  role: string;
  parent: string;
  enabled: boolean;
  ownPermissions: string[];
}

export default function PermissionInheritanceConfigPage() {
  const [roles, setRoles] = useState<RoleInheritance[]>([
    { role: 'super-admin', parent: '-', enabled: true, ownPermissions: ['admin:all'] },
    { role: 'admin', parent: 'super-admin', enabled: true, ownPermissions: ['write:users', 'read:audit'] },
    { role: 'developer', parent: 'admin', enabled: true, ownPermissions: ['write:users'] },
    { role: 'auditor', parent: '-', enabled: true, ownPermissions: ['read:audit', 'read:users'] },
    { role: 'viewer', parent: 'auditor', enabled: false, ownPermissions: ['read:users'] },
  ]);

  const [selectedRole, setSelectedRole] = useState('admin');
  const [cycleWarning, setCycleWarning] = useState(false);
  const allRoles = ['super-admin', 'admin', 'developer', 'auditor', 'viewer', '-'];

  const setParent = (roleName: string, parent: string) => {
    setRoles(prev => prev.map(r => r.role === roleName ? { ...r, parent } : r));
    let current: string | undefined = parent;
    const visited = new Set<string>([roleName]);
    while (current && current !== '-') {
      if (visited.has(current)) { setCycleWarning(true); return; }
      visited.add(current);
      const next = roles.find(r => r.role === current);
      current = next?.parent;
    }
    setCycleWarning(false);
  };

  const toggleInheritance = (roleName: string) => {
    setRoles(prev => prev.map(r => r.role === roleName ? { ...r, enabled: !r.enabled } : r));
  };

  const getEffective = (roleName: string): string[] => {
    const role = roles.find(r => r.role === roleName);
    if (!role || !role.enabled) return role?.ownPermissions || [];
    const inherited = role.parent !== '-' ? getEffective(role.parent) : [];
    return Array.from(new Set([...role.ownPermissions, ...inherited]));
  };

  const effective = getEffective(selectedRole);
  const selected = roles.find(r => r.role === selectedRole);
  const inherited = selected && selected.parent !== '-' && selected.enabled ? getEffective(selected.parent) : [];

  return (
    <div className="p-6 max-w-5xl mx-auto space-y-6">
      <div>
        <h1 className="text-2xl font-bold">Permission Inheritance Configuration</h1>
        <p className="text-gray-600">Configure role hierarchy and permission inheritance with cycle detection.</p>
      </div>

      {cycleWarning && (
        <div className="bg-red-50 border border-red-200 rounded p-3 text-sm text-red-700">
          <strong>Cycle detected!</strong> The parent assignment creates a circular inheritance chain.
        </div>
      )}

      <section className="bg-white rounded-lg shadow p-6 space-y-4">
        <h2 className="text-lg font-semibold">Inheritance Tree</h2>
        <div className="space-y-2">
          {roles.map(r => (
            <div key={r.role} className="flex items-center gap-3 border-b pb-2">
              <span className="font-mono text-sm w-32">{r.role}</span>
              <span className="text-gray-300">{'->'}</span>
              <select value={r.parent} onChange={e => setParent(r.role, e.target.value)} className="border rounded px-2 py-1 text-sm">
                {allRoles.map(p => <option key={p} value={p}>{p}</option>)}
              </select>
              <label className="flex items-center gap-1 ml-4">
                <input type="checkbox" checked={r.enabled} onChange={() => toggleInheritance(r.role)} className="rounded" />
                <span className="text-xs">{r.enabled ? 'inheriting' : 'standalone'}</span>
              </label>
              <span className="text-xs text-gray-400">{r.ownPermissions.length} own perms</span>
            </div>
          ))}
        </div>
      </section>

      <div className="grid grid-cols-2 gap-6">
        <section className="bg-white rounded-lg shadow p-6 space-y-4">
          <h2 className="text-lg font-semibold">Effective Permissions Calculator</h2>
          <select value={selectedRole} onChange={e => setSelectedRole(e.target.value)} className="w-full border rounded px-3 py-2 text-sm">
            {roles.map(r => <option key={r.role} value={r.role}>{r.role}</option>)}
          </select>
          <div>
            <div className="text-xs font-medium text-gray-500 mb-2">Effective Permissions ({effective.length}):</div>
            <div className="flex flex-wrap gap-1">
              {effective.map(p => (
                <span key={p} className={`px-2 py-0.5 rounded text-xs ${inherited.includes(p) ? 'bg-blue-100 text-blue-700' : 'bg-green-100 text-green-700'}`}>
                  {p}{inherited.includes(p) ? ' (inh)' : ''}
                </span>
              ))}
            </div>
          </div>
        </section>

        <section className="bg-white rounded-lg shadow p-6 space-y-4">
          <h2 className="text-lg font-semibold">Permission Diff (Own vs Inherited)</h2>
          {selected && (
            <div className="space-y-3">
              <div>
                <div className="text-xs font-medium text-green-600 mb-1">Own Permissions:</div>
                <div className="flex flex-wrap gap-1">
                  {selected.ownPermissions.map(p => <span key={p} className="px-2 py-0.5 bg-green-100 text-green-700 rounded text-xs">{p}</span>)}
                </div>
              </div>
              <div>
                <div className="text-xs font-medium text-blue-600 mb-1">Inherited:</div>
                <div className="flex flex-wrap gap-1">
                  {inherited.map(p => <span key={p} className="px-2 py-0.5 bg-blue-100 text-blue-700 rounded text-xs">{p}</span>)}
                </div>
              </div>
            </div>
          )}
        </section>
      </div>
    </div>
  );
}