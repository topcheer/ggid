'use client';
import { useState, useEffect } from 'react';

interface DelegationNode {
  id: string;
  name: string;
  type: string;
  scopes: string[];
  delegatedAt: string;
  expires: string;
  children: DelegationNode[];
  collapsed?: boolean;
}

export default function AgentDelegationGraphPage() {
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [selectedNode, setSelectedNode] = useState<DelegationNode | null>(null);
  const [cycleWarning, setCycleWarning] = useState(false);
  const [maxDepth, setMaxDepth] = useState(5);

  const [tree, setTree] = useState<DelegationNode | null>(null);

  useEffect(() => {
    fetch("/api/v1/identity/nhi/orphans", {
      headers: { "Content-Type": "application/json", "X-Tenant-ID": "00000000-0000-0000-0000-000000000001" },
    })
      .then(res => { if (!res.ok) throw new Error(`HTTP ${res.status}`); return res.json(); })
      .then(data => {
        const t = data.tree || data.data || data;
        setTree(t);
        if (t) setCycleWarning(detectCycle(t, new Set()));
        setLoading(false);
      })
      .catch(err => { setError(err.message); setLoading(false); });
  }, []);

  const computeDepth = (node: DelegationNode): number => {
    if (!node.children || node.children.length === 0) return 0;
    return 1 + Math.max(...node.children.map(computeDepth));
  };
  const currentDepth = tree ? computeDepth(tree) : 0;

  const toggleCollapse = (node: DelegationNode) => {
    const toggle = (n: DelegationNode): DelegationNode => ({
      ...n,
      collapsed: n.id === node.id ? !n.collapsed : n.collapsed,
      children: n.children.map(toggle),
    });
    if (tree) setTree(toggle(tree));
  };

  const detectCycle = (node: DelegationNode, visited: Set<string>): boolean => {
    if (visited.has(node.id)) return true;
    visited.add(node.id);
    return node.children.some(c => detectCycle(c, new Set(visited)));
  };
  // detectCycle moved to useEffect above

  const renderNode = (node: DelegationNode, depth: number): React.ReactNode => {
    const typeColor = node.type === 'user' ? 'bg-blue-100 text-blue-700' : node.type === 'agent' ? 'bg-purple-100 text-purple-700' : 'bg-teal-100 text-teal-700';
    const isMaxDepth = depth >= maxDepth;

    return (
      <li key={node.id} className="ml-4">
        <div className="flex items-center gap-2">
          {node.children.length > 0 && (
            <button onClick={() => toggleCollapse(node)} className="text-xs text-gray-400 hover:text-gray-600">
              {node.collapsed ? '+' : '-'}
            </button>
          )}
          <button
            onClick={() => setSelectedNode(node)}
            className={`flex items-center gap-2 px-3 py-1.5 rounded border text-sm hover:bg-gray-50 ${selectedNode?.id === node.id ? 'border-blue-500 bg-blue-50' : 'border-gray-200'}`}
          >
            <span className={`px-1.5 py-0.5 rounded text-xs ${typeColor}`}>{node.type}</span>
            <span className="font-medium">{node.name}</span>
            <span className="text-xs text-gray-400">{node.scopes.length} scopes</span>
          </button>
          {isMaxDepth && node.children.length > 0 && (
            <span className="text-xs text-amber-600">max depth reached</span>
          )}
        </div>
        {!node.collapsed && !isMaxDepth && node.children.length > 0 && (
          <ul className="border-l border-gray-200 ml-3 mt-1 space-y-1">
            {node.children.map(c => renderNode(c, depth + 1))}
          </ul>
        )}
      </li>
    );
  };

  if (loading) return (
    <div className="p-6"><h1 className="text-2xl font-bold mb-4">Agent Delegation Graph</h1><p>Loading...</p></div>
  );
  if (error) return (
    <div className="p-6"><h1 className="text-2xl font-bold mb-4">Agent Delegation Graph</h1><p className="text-red-600">Error: {error}</p></div>
  );
  if (!tree) return (
    <div className="p-6"><h1 className="text-2xl font-bold mb-4">Agent Delegation Graph</h1><p>No data available</p></div>
  );
  return (
    <div className="p-6 max-w-5xl mx-auto space-y-6">
      <div>
        <h1 className="text-2xl font-bold">Agent Delegation Graph</h1>
        <p className="text-gray-600">Visualize and inspect the delegation chain from users to agents and sub-agents.</p>
      </div>

      <div className="flex items-center gap-6 text-sm">
        <div className="flex items-center gap-2">
          <span className="text-gray-500">Max Depth:</span>
          <input type="number" min={1} max={10} value={maxDepth} onChange={e => setMaxDepth(parseInt(e.target.value) || 5)} className="w-16 border rounded px-2 py-1 text-sm" />
        </div>
        <div className="flex items-center gap-2">
          <span className="text-gray-500">Current Depth:</span>
          <span className="font-mono font-bold">{currentDepth}</span>
        </div>
        {cycleWarning && (
          <span className="px-3 py-1 bg-red-100 text-red-700 rounded text-sm">Cycle Detected!</span>
        )}
      </div>

      <div className="grid grid-cols-3 gap-6">
        <section className="col-span-2 bg-white rounded-lg shadow p-6">
          <h2 className="text-lg font-semibold mb-4">Delegation Tree</h2>
          <ul className="space-y-1">
            {renderNode(tree, 0)}
          </ul>
        </section>

        <section className="bg-white rounded-lg shadow p-6">
          <h2 className="text-lg font-semibold mb-4">Node Details</h2>
          {selectedNode ? (
            <div className="space-y-3">
              <div>
                <div className="text-xs text-gray-500">Name</div>
                <div className="font-medium">{selectedNode.name}</div>
              </div>
              <div>
                <div className="text-xs text-gray-500">Type</div>
                <div className="text-sm capitalize">{selectedNode.type}</div>
              </div>
              <div>
                <div className="text-xs text-gray-500">ID</div>
                <div className="font-mono text-xs">{selectedNode.id}</div>
              </div>
              <div>
                <div className="text-xs text-gray-500">Scopes</div>
                <div className="flex flex-wrap gap-1 mt-1">{selectedNode.scopes.map(s => <span key={s} className="px-1.5 py-0.5 bg-gray-100 rounded text-xs">{s}</span>)}</div>
              </div>
              <div>
                <div className="text-xs text-gray-500">Delegated At</div>
                <div className="text-sm">{selectedNode.delegatedAt}</div>
              </div>
              <div>
                <div className="text-xs text-gray-500">Expires</div>
                <div className="text-sm">{selectedNode.expires}</div>
              </div>
            </div>
          ) : (
            <p className="text-sm text-gray-400">Click a node to view delegation details.</p>
          )}
        </section>
      </div>
    </div>
  );
}
