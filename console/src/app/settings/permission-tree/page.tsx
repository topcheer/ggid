'use client';
import { useState, useEffect } from 'react';
import { useTranslations } from "@/lib/i18n";

interface TreeNode {
  id: string;
  name: string;
  type: 'role' | 'permission' | 'resource';
  inherited: boolean;
  children: TreeNode[];
  collapsed?: boolean;
}

interface UserPerm {
  user: string;
  permissions: string[];
}

export default function PermissionTreePage() {
  const t = useTranslations();
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [tree, setTree] = useState<TreeNode | null>(null);

  const [selectedUser, setSelectedUser] = useState('');
  const [gapAnalysis, setGapAnalysis] = useState<string[]>([]);

  useEffect(() => {
    fetch("/api/v1/policies/permissions/tree", {
      headers: { "Content-Type": "application/json", "X-Tenant-ID": "00000000-0000-0000-0000-000000000001" },
    })
      .then(res => { if (!res.ok) return null; return res.json(); })
      .then(data => { setTree(data.tree || data); setLoading(false); })
      .catch(err => { setError(err.message); setLoading(false); });
  }, []);

  const users: UserPerm[] = [
    { user: 'alice@ggid.io', permissions: ['admin:all', 'read:audit', 'write:users'] },
    { user: 'bob@ggid.io', permissions: ['write:users', 'read:policy'] },
    { user: 'carol@ggid.io', permissions: ['read:audit', 'read:users'] },
  ];

  const allPermissions = ['admin:all', 'read:audit', 'write:users', 'read:policy', 'read:users', 'write:policy', 'write:orgs'];

  const toggleCollapse = (node: TreeNode) => {
    if (!tree) return;
    const toggle = (n: TreeNode): TreeNode => ({
      ...n,
      collapsed: n.id === node.id ? !n.collapsed : n.collapsed,
      children: n.children.map(toggle),
    });
    setTree(toggle(tree));
  };

  const typeColor = (t: string): string =>
    t === 'role' ? 'bg-purple-100 text-purple-700' :
    t === 'permission' ? 'bg-blue-100 text-blue-700' :
    'bg-green-100 text-green-700';

  const renderNode = (node: TreeNode, depth: number): React.ReactNode => (
    <li key={node.id} className="ml-4">
      <div className="flex items-center gap-2">
        {node.children.length > 0 && (
          <button onClick={() => toggleCollapse(node)} aria-label="Toggle collapse" className="text-xs text-gray-400 hover:text-gray-600">
            {node.collapsed ? '+' : '-'}
          </button>
        )}
        <span className={`px-2 py-0.5 rounded text-xs ${typeColor(node.type)}`}>{node.type}</span>
        <span className="text-sm font-medium">{node.name}</span>
        {node.inherited && <span className="text-xs text-amber-600">(inherited)</span>}
      </div>
      {!node.collapsed && node.children.length > 0 && (
        <ul className="border-l border-gray-200 ml-3 mt-1 space-y-1">
          {node.children.map(c => renderNode(c, depth + 1))}
        </ul>
      )}
    </li>
  );

  const analyzeGaps = () => {
    const user = users.find(u => u.user === selectedUser);
    if (!user) { setGapAnalysis([]); return; }
    const gaps = allPermissions.filter(p => !user.permissions.includes(p));
    setGapAnalysis(gaps);
  };

  const exportTree = () => {
    const blob = new Blob([JSON.stringify(tree, null, 2)], { type: 'application/json' });
    const url = URL.createObjectURL(blob);
    const a = document.createElement('a');
    a.href = url; a.download = 'permission-tree.json'; a.click();
  };

  if (loading) return (<div className="p-6"><h1 className="text-2xl font-bold mb-4">Permission Tree</h1><p>Loading...</p></div>);
  if (error) return (<div className="p-6"><h1 className="text-2xl font-bold mb-4">Permission Tree</h1><p className="text-red-600">Error: {error}</p></div>);
  if (!tree) return (<div className="p-6"><h1 className="text-2xl font-bold mb-4">Permission Tree</h1><p>No data available</p></div>);
  return (
    <div className="p-6 max-w-5xl mx-auto space-y-6">
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-2xl font-bold">Permission Tree</h1>
          <p className="text-gray-600">Interactive role-permission-resource hierarchy with inheritance and gap analysis.</p>
        </div>
        <button onClick={exportTree} className="px-4 py-2 border rounded text-sm">Export Tree</button>
      </div>

      <div className="grid grid-cols-3 gap-6">
        <section className="col-span-2 bg-white rounded-lg shadow p-6 space-y-4">
          <h2 className="text-lg font-semibold">Permission Hierarchy</h2>
          <ul className="space-y-1">{renderNode(tree, 0)}</ul>
        </section>

        <section className="bg-white rounded-lg shadow p-6 space-y-4">
          <h2 className="text-lg font-semibold">Effective Permissions</h2>
          <select aria-label="Selected user" value={selectedUser} onChange={e => setSelectedUser(e.target.value)} className="w-full border rounded px-3 py-2 text-sm">
            <option value="">Select user...</option>
            {users.map(u => <option key={u.user} value={u.user}>{u.user}</option>)}
          </select>
          {selectedUser && (
            <div className="space-y-2">
              <div className="text-xs font-medium text-gray-500">Effective Permissions:</div>
              {users.find(u => u.user === selectedUser)?.permissions.map(p => (
                <div key={p} className="flex items-center gap-2">
                  <span className="w-2 h-2 bg-green-500 rounded-full" />
                  <span className="text-sm font-mono">{p}</span>
                </div>
              ))}
            </div>
          )}
          <button onClick={analyzeGaps} disabled={!selectedUser} className="px-3 py-1.5 bg-blue-600 text-white rounded text-sm disabled:opacity-50">Analyze Gaps</button>
        </section>
      </div>

      {gapAnalysis.length > 0 && (
        <section className="bg-white rounded-lg shadow p-6 space-y-4">
          <h2 className="text-lg font-semibold">Permission Gap Analysis</h2>
          <p className="text-sm text-gray-500">Permissions not assigned to <strong>{selectedUser}</strong>:</p>
          <div className="flex flex-wrap gap-2">
            {gapAnalysis.map(p => (
              <span key={p} className="px-2 py-1 bg-red-50 text-red-700 rounded text-xs font-mono">{p}</span>
            ))}
          </div>
        </section>
      )}

      <section className="bg-white rounded-lg shadow p-6 space-y-3">
        <h2 className="text-lg font-semibold">Legend</h2>
        <div className="flex items-center gap-4 text-sm">
          <span className="flex items-center gap-1"><span className="px-2 py-0.5 bg-purple-100 text-purple-700 rounded text-xs">role</span> Role node</span>
          <span className="flex items-center gap-1"><span className="px-2 py-0.5 bg-blue-100 text-blue-700 rounded text-xs">permission</span> Permission node</span>
          <span className="flex items-center gap-1"><span className="px-2 py-0.5 bg-green-100 text-green-700 rounded text-xs">resource</span> Resource node</span>
          <span className="flex items-center gap-1"><span className="text-amber-600 text-xs">(inherited)</span> Inherited from parent</span>
        </div>
      </section>
    </div>
  );
}