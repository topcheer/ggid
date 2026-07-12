'use client';
import { useState, useEffect } from 'react';

interface OrgNode {
  id: string;
  name: string;
  level: number;
  path: string;
  memberCount: number;
  created: string;
  children: OrgNode[];
  collapsed?: boolean;
}

export default function OrgTreeViewerPage() {
  const [tree, setTree] = useState<OrgNode | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    fetch('/api/v1/orgs/tree', {
      headers: { 'Content-Type': 'application/json', 'X-Tenant-ID': '00000000-0000-0000-0000-000000000001' },
    })
      .then(r => { if (!r.ok) throw new Error(`HTTP ${r.status}`); return r.json(); })
      .then(data => { if (data) setTree(data); setLoading(false); })
      .catch(e => { setError(e.message); setLoading(false); });
  }, []);

  const [selected, setSelected] = useState<OrgNode | null>(null);
  const [search, setSearch] = useState('');

  const countOrgs = (n: OrgNode): number => 1 + n.children.reduce((s, c) => s + countOrgs(c), 0);
  const maxDepth = (n: OrgNode): number => n.children.length === 0 ? 0 : 1 + Math.max(...n.children.map(maxDepth));
  const totalMembers = (n: OrgNode): number => n.memberCount + n.children.reduce((s, c) => s + totalMembers(c), 0);

  const toggleCollapse = (node: OrgNode) => {
    const toggle = (n: OrgNode): OrgNode => ({
      ...n, collapsed: n.id === node.id ? !n.collapsed : n.collapsed,
      children: n.children.map(toggle),
    });
    setTree(tree ? toggle(tree) : null);
  };

  const matches = (n: OrgNode, q: string): boolean => n.name.toLowerCase().includes(q.toLowerCase());

  const renderNode = (node: OrgNode): React.ReactNode => {
    const highlighted = search && matches(node, search);
    return (
      <li key={node.id} className="ml-4">
        <div className="flex items-center gap-2">
          {node.children.length > 0 && (
            <button onClick={() => toggleCollapse(node)} className="text-xs text-gray-400">{node.collapsed ? '+' : '-'}</button>
          )}
          <button
            onClick={() => setSelected(node)}
            className={`px-3 py-1.5 rounded border text-sm hover:bg-gray-50 ${selected?.id === node.id ? 'border-blue-500 bg-blue-50' : 'border-gray-200'} ${highlighted ? 'ring-2 ring-yellow-400' : ''}`}
          >
            <span className="font-medium">{node.name}</span>
            <span className="text-xs text-gray-400 ml-2">{node.memberCount} members</span>
          </button>
        </div>
        {!node.collapsed && node.children.length > 0 && (
          <ul className="border-l border-gray-200 ml-3 mt-1 space-y-1">
            {node.children.map(renderNode)}
          </ul>
        )}
      </li>
    );
  };

  const exportTree = (format: 'json' | 'csv') => {
    const data = format === 'json' ? JSON.stringify(tree, null, 2) :
      ['id,name,path,level,memberCount', ...(() => {
        const rows: string[] = [];
        const flatten = (n: OrgNode) => { rows.push(`${n.id},${n.name},${n.path},${n.level},${n.memberCount}`); n.children.forEach(flatten); };
        if (tree) flatten(tree);
        return rows;
      })()].join('\n');
    const blob = new Blob([data], { type: format === 'json' ? 'application/json' : 'text/csv' });
    const url = URL.createObjectURL(blob);
    const a = document.createElement('a');
    a.href = url; a.download = `org-tree.${format}`; a.click();
  };

  if (loading) return <div className="p-6"><p>Loading...</p></div>;
  if (error) return <div className="p-6 text-red-600">Error: {error}</div>;
  if (!tree) return <div className="p-6"><p>No data available</p></div>;

  return (
    <div className="p-6 max-w-6xl mx-auto space-y-6">
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-2xl font-bold">Organization Tree Viewer</h1>
          <p className="text-gray-600">Interactive org hierarchy with search, detail panel, and export.</p>
        </div>
        <div className="flex gap-2">
          <button onClick={() => exportTree('json')} className="px-3 py-1.5 border rounded text-sm">Export JSON</button>
          <button onClick={() => exportTree('csv')} className="px-3 py-1.5 border rounded text-sm">Export CSV</button>
        </div>
      </div>

      <div className="grid grid-cols-4 gap-4">
        <div className="bg-white rounded-lg shadow p-4 text-center"><div className="text-2xl font-bold">{countOrgs(tree)}</div><div className="text-sm text-gray-500">Total Orgs</div></div>
        <div className="bg-white rounded-lg shadow p-4 text-center"><div className="text-2xl font-bold">{maxDepth(tree)}</div><div className="text-sm text-gray-500">Max Depth</div></div>
        <div className="bg-white rounded-lg shadow p-4 text-center"><div className="text-2xl font-bold">{totalMembers(tree)}</div><div className="text-sm text-gray-500">Total Members</div></div>
        <div className="bg-white rounded-lg shadow p-4 text-center"><div className="text-2xl font-bold">{Math.round(totalMembers(tree) / countOrgs(tree))}</div><div className="text-sm text-gray-500">Avg Members</div></div>
      </div>

      <div className="grid grid-cols-3 gap-6">
        <section className="col-span-2 bg-white rounded-lg shadow p-6 space-y-4">
          <input type="text" placeholder="Search organizations..." value={search} onChange={e => setSearch(e.target.value)} className="w-full border rounded px-3 py-2 text-sm" />
          <ul className="space-y-1">{renderNode(tree)}</ul>
        </section>

        <section className="bg-white rounded-lg shadow p-6 space-y-4">
          <h2 className="text-lg font-semibold">Org Details</h2>
          {selected ? (
            <div className="space-y-3 text-sm">
              <div><div className="text-xs text-gray-500">Name</div><div className="font-medium">{selected.name}</div></div>
              <div><div className="text-xs text-gray-500">Path</div><div className="font-mono text-xs">{selected.path}</div></div>
              <div><div className="text-xs text-gray-500">Level</div><div>{selected.level}</div></div>
              <div><div className="text-xs text-gray-500">Members</div><div className="font-bold">{selected.memberCount}</div></div>
              <div><div className="text-xs text-gray-500">Created</div><div>{selected.created}</div></div>
              <div><div className="text-xs text-gray-500">Sub-organizations</div><div>{selected.children.length}</div></div>
              <div className="border-t pt-3">
                <div className="text-xs font-medium text-gray-500 mb-2">Role Assignments:</div>
                <div className="space-y-1">
                  <div className="flex items-center gap-2"><span className="px-2 py-0.5 bg-purple-100 text-purple-700 rounded text-xs">admin</span><span className="text-xs text-gray-500">2 assigned</span></div>
                  <div className="flex items-center gap-2"><span className="px-2 py-0.5 bg-blue-100 text-blue-700 rounded text-xs">developer</span><span className="text-xs text-gray-500">15 assigned</span></div>
                  <div className="flex items-center gap-2"><span className="px-2 py-0.5 bg-green-100 text-green-700 rounded text-xs">viewer</span><span className="text-xs text-gray-500">28 assigned</span></div>
                </div>
              </div>
            </div>
          ) : <p className="text-sm text-gray-400">Click an org node to view details.</p>}
        </section>
      </div>
    </div>
  );
}