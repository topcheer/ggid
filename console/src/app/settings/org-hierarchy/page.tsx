'use client';
import { useState, useEffect } from 'react';
import { useTranslations } from "@/lib/i18n";
import { authHeader, isAuthenticated } from "@/lib/auth-helpers";

interface OrgNode {
  id: string;
  name: string;
  parentId: string | null;
  children: OrgNode[];
}

export default function OrgHierarchyPage() {
  const t = useTranslations();

  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [orgs, setOrgs] = useState<OrgNode[]>([]);

  useEffect(() => {
    fetch("/api/v1/orgs/tree", {
      headers: { ...authHeader(), "Content-Type": "application/json", "X-Tenant-ID": "00000000-0000-0000-0000-000000000001" },
    })
      .then(res => { if (!res.ok) return null; return res.json(); })
      .then(data => { setOrgs(Array.isArray(data) ? data : (data.orgs || data.items || [])); setLoading(false); })
      .catch(err => { setError(err.message); setLoading(false); });
  }, []);

  const [search, setSearch] = useState('');
  const [showForm, setShowForm] = useState(false);
  const [newOrg, setNewOrg] = useState({ name: '', parentId: 'o1' });
  const [deleteTarget, setDeleteTarget] = useState<OrgNode | null>(null);

  // Build tree structure
  const buildTree = (parentId: string | null): OrgNode[] => {
    return orgs
      .filter(o => o.parentId === parentId)
      .map(o => ({ ...o, children: buildTree(o.id) }));
  };

  const tree = buildTree(null);

  const getDepth = (id: string): number => {
    let depth = 0;
    let current = orgs.find(o => o.id === id);
    while (current && current.parentId) {
      depth++;
      current = orgs.find(o => o.id === current!.parentId);
    }
    return depth;
  };

  const getPath = (id: string): string[] => {
    const path: string[] = [];
    let current = orgs.find(o => o.id === id);
    while (current) {
      path.unshift(current.name);
      current = current.parentId ? orgs.find(o => o.id === current!.parentId) : undefined;
    }
    return path;
  };

  const [selectedOrg, setSelectedOrg] = useState<string | null>(null);

  const addOrg = () => {
    setOrgs(prev => [...prev, { id: `o${prev.length + 1}`, name: newOrg.name || `Org ${prev.length + 1}`, parentId: newOrg.parentId, children: [] }]);
    setShowForm(false);
    setNewOrg({ name: '', parentId: 'o1' });
  };

  const deleteOrg = (id: string) => {
    const hasChildren = orgs.some(o => o.parentId === id);
    if (hasChildren) return;
    setOrgs(prev => prev.filter(o => o.id !== id));
    setDeleteTarget(null);
  };

  const filteredOrgs = search ? orgs.filter(o => o.name.toLowerCase().includes(search.toLowerCase())) : orgs;

  const renderTree = (nodes: OrgNode[], depth: number): React.ReactNode => (
    <ul className={depth === 0 ? 'space-y-1' : 'border-l border-gray-200 ml-3 mt-1 space-y-1'}>
      {nodes.map(node => {
        const hasChildren = orgs.some(o => o.parentId === node.id);
        const isMatch = search && node.name.toLowerCase().includes(search.toLowerCase());
        return (
          <li key={node.id} className="ml-2">
            <div className={`flex items-center gap-2 px-2 py-1 rounded text-sm ${selectedOrg === node.id ? 'bg-blue-50 border border-blue-300' : 'hover:bg-gray-50'} ${isMatch ? 'bg-yellow-50' : ''}`}>
              <span className="text-xs text-gray-400">L{depth}</span>
              <button onClick={() => setSelectedOrg(node.id)} className="font-medium">{node.name}</button>
              {hasChildren && <span className="text-xs text-gray-400">{orgs.filter(o => o.parentId === node.id).length} children</span>}
              <button onClick={() => setDeleteTarget(node)} className="text-red-600 text-xs ml-auto">Delete</button>
            </div>
            {hasChildren && renderTree(buildTree(node.id), depth + 1)}
          </li>
        );
      })}
    </ul>
  );

  if (loading) return (
    <div className="p-6"><h1 className="text-2xl font-bold mb-4">Organization Hierarchy</h1><p>Loading...</p></div>
  );
  if (error) return (
    <div className="p-6"><h1 className="text-2xl font-bold mb-4">Organization Hierarchy</h1><p className="text-red-600">Error: {error}</p></div>
  );
  return (
    <div className="p-6 max-w-5xl mx-auto space-y-6">
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-2xl font-bold">Organization Hierarchy</h1>
          <p className="text-gray-600">Manage organizational structure, parent-child relationships, and hierarchy levels.</p>
        </div>
        <button onClick={() => setShowForm(!showForm)} className="px-4 py-2 bg-blue-600 text-white rounded text-sm">
          {showForm ? 'Cancel' : 'Create Org'}
        </button>
      </div>

      {showForm && (
        <section className="bg-white rounded-lg shadow p-6 space-y-4">
          <h2 className="text-lg font-semibold">Create Organization</h2>
          <div className="grid grid-cols-2 gap-4">
            <div>
              <label className="text-sm font-medium">Name</label>
              <input aria-label="Organization name" type="text" placeholder="Organization name" value={newOrg.name} onChange={e => setNewOrg(prev => ({ ...prev, name: e.target.value }))} className="w-full border rounded px-3 py-2 text-sm mt-1" />
            </div>
            <div>
              <label className="text-sm font-medium">Parent Organization</label>
              <select aria-label="new Org" value={newOrg.parentId} onChange={e => setNewOrg(prev => ({ ...prev, parentId: e.target.value }))} className="w-full border rounded px-3 py-2 text-sm mt-1">
                {orgs.map(o => <option key={o.id} value={o.id}>{'  '.repeat(getDepth(o.id))}{o.name}</option>)}
              </select>
            </div>
          </div>
          <button onClick={addOrg} disabled={!newOrg.name} className="px-4 py-2 bg-blue-600 text-white rounded text-sm disabled:opacity-50">Create</button>
        </section>
      )}

      <div className="flex items-center gap-3">
        <input aria-label="Search organizations..." type="text" placeholder="Search organizations..." value={search} onChange={e => setSearch(e.target.value)} className="flex-1 border rounded px-3 py-2 text-sm" />
        <span className="text-sm text-gray-500">{filteredOrgs.length} of {orgs.length}</span>
      </div>

      <div className="grid grid-cols-3 gap-6">
        <section className="col-span-2 bg-white rounded-lg shadow p-6 space-y-4">
          <h2 className="text-lg font-semibold">Organization Tree</h2>
          {renderTree(tree, 0)}
        </section>

        <section className="bg-white rounded-lg shadow p-6 space-y-4">
          <h2 className="text-lg font-semibold">Details</h2>
          {selectedOrg ? (
            <div className="space-y-3">
              <div>
                <div className="text-xs text-gray-500">Breadcrumb</div>
                <div className="text-sm">{getPath(selectedOrg).join(' / ')}</div>
              </div>
              <div>
                <div className="text-xs text-gray-500">Level</div>
                <div className="text-sm font-bold">L{getDepth(selectedOrg)}</div>
              </div>
              <div>
                <div className="text-xs text-gray-500">Direct Children</div>
                <div className="text-sm">{orgs.filter(o => o.parentId === selectedOrg).map(o => o.name).join(', ') || 'None'}</div>
              </div>
            </div>
          ) : (
            <p className="text-sm text-gray-400">Select an org to view details.</p>
          )}
        </section>
      </div>

      {deleteTarget && (
        <div className="fixed inset-0 bg-black/50 flex items-center justify-center z-50">
          <div className="bg-white rounded-lg p-6 max-w-md w-full mx-4 space-y-4">
            <h2 className="text-lg font-semibold">Delete Organization</h2>
            {orgs.some(o => o.parentId === deleteTarget.id) ? (
              <>
                <p className="text-sm text-red-600">Cannot delete <strong>{deleteTarget.name}</strong> — it has child organizations. Please move or delete all children first.</p>
                <button onClick={() => setDeleteTarget(null)} className="px-4 py-2 border rounded text-sm">Close</button>
              </>
            ) : (
              <>
                <p className="text-sm text-gray-600">Delete <strong>{deleteTarget.name}</strong>? This cannot be undone.</p>
                <div className="flex justify-end gap-3">
                  <button onClick={() => setDeleteTarget(null)} className="px-4 py-2 border rounded text-sm">Cancel</button>
                  <button onClick={() => deleteOrg(deleteTarget.id)} className="px-4 py-2 bg-red-600 text-white rounded text-sm">Confirm Delete</button>
                </div>
              </>
            )}
          </div>
        </div>
      )}
    </div>
  );
}