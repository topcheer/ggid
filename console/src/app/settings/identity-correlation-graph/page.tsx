'use client';

import { useState, useMemo, useCallback, useEffect } from 'react';

interface GraphNode {
  id: string;
  label: string;
  type: 'user' | 'email' | 'phone' | 'device' | 'ip';
  email?: string;
  phone?: string;
  device?: string;
  ip?: string;
  isSynthetic: boolean;
  confidence: number;
}

interface GraphEdge {
  source: string;
  target: string;
  correlationType: string;
  confidence: number;
}

const SAMPLE_NODES: GraphNode[] = [
  { id: 'u1', label: 'alice@corp.com', type: 'user', email: 'alice@corp.com', isSynthetic: false, confidence: 1.0 },
  { id: 'e1', label: 'alice@corp.com', type: 'email', email: 'alice@corp.com', isSynthetic: false, confidence: 0.95 },
  { id: 'p1', label: '+1-555-0100', type: 'phone', phone: '+1-555-0100', isSynthetic: false, confidence: 0.88 },
  { id: 'd1', label: 'MacBook-Pro-A1', type: 'device', device: 'MacBook-Pro-A1', isSynthetic: false, confidence: 0.92 },
  { id: 'ip1', label: '192.168.1.50', type: 'ip', ip: '192.168.1.50', isSynthetic: false, confidence: 0.85 },
  { id: 'u2', label: 'bob@corp.com', type: 'user', email: 'bob@corp.com', isSynthetic: false, confidence: 1.0 },
  { id: 'e2', label: 'bob@corp.com', type: 'email', email: 'bob@corp.com', isSynthetic: false, confidence: 0.95 },
  { id: 'ip2', label: '192.168.1.51', type: 'ip', ip: '192.168.1.51', isSynthetic: false, confidence: 0.78 },
  { id: 'u3', label: 'unknown@temp.com', type: 'user', email: 'unknown@temp.com', isSynthetic: true, confidence: 0.42 },
  { id: 'e3', label: 'unknown@temp.com', type: 'email', email: 'unknown@temp.com', isSynthetic: true, confidence: 0.38 },
  { id: 'ip3', label: '10.0.0.99', type: 'ip', ip: '10.0.0.99', isSynthetic: true, confidence: 0.35 },
];

const SAMPLE_EDGES: GraphEdge[] = [
  { source: 'u1', target: 'e1', correlationType: 'email_match', confidence: 0.95 },
  { source: 'u1', target: 'p1', correlationType: 'phone_match', confidence: 0.88 },
  { source: 'u1', target: 'd1', correlationType: 'device_match', confidence: 0.92 },
  { source: 'u1', target: 'ip1', correlationType: 'ip_match', confidence: 0.85 },
  { source: 'u2', target: 'e2', correlationType: 'email_match', confidence: 0.95 },
  { source: 'u2', target: 'ip2', correlationType: 'ip_match', confidence: 0.78 },
  { source: 'u1', target: 'ip2', correlationType: 'shared_ip', confidence: 0.65 },
  { source: 'u3', target: 'e3', correlationType: 'email_match', confidence: 0.38 },
  { source: 'u3', target: 'ip3', correlationType: 'ip_match', confidence: 0.35 },
  { source: 'u1', target: 'u3', correlationType: 'device_overlap', confidence: 0.42 },
];

const NODE_COLORS: Record<string, string> = {
  user: 'bg-blue-500',
  email: 'bg-green-500',
  phone: 'bg-purple-500',
  device: 'bg-orange-500',
  ip: 'bg-gray-500',
};

const NODE_POSITIONS: Record<string, { x: number; y: number }> = {
  u1: { x: 300, y: 150 },
  e1: { x: 150, y: 50 },
  p1: { x: 450, y: 50 },
  d1: { x: 150, y: 250 },
  ip1: { x: 450, y: 250 },
  u2: { x: 600, y: 350 },
  e2: { x: 550, y: 450 },
  ip2: { x: 700, y: 450 },
  u3: { x: 200, y: 400 },
  e3: { x: 100, y: 450 },
  ip3: { x: 300, y: 450 },
};

export default function IdentityCorrelationGraphPage() {
  const [searchQuery, setSearchQuery] = useState('alice@corp.com');
  const [depth, setDepth] = useState(3);
  const [selectedNode, setSelectedNode] = useState<GraphNode | null>(null);
  const [nodes, setNodes] = useState<GraphNode[]>([]);
  const [edges, setEdges] = useState<GraphEdge[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    fetch('/api/v1/identity/groups/', {
      headers: { 'Content-Type': 'application/json', 'X-Tenant-ID': '00000000-0000-0000-0000-000000000001' },
    })
      .then(r => { if (!r.ok) throw new Error(`HTTP ${r.status}`); return r.json(); })
      .then(data => {
        if (data) {
          if (data.nodes) setNodes(data.nodes);
          if (data.edges) setEdges(data.edges);
        }
        setLoading(false);
      })
      .catch(e => { setError(e.message); setLoading(false); });
  }, []);

  const filteredNodes = useMemo(() => {
    if (!searchQuery) return nodes;
    const match = nodes.find(n => n.label.toLowerCase().includes(searchQuery.toLowerCase()));
    if (!match) return nodes;
    const visited = new Set<string>([match.id]);
    const queue = [match.id];
    for (let d = 0; d < depth && queue.length > 0; d++) {
      const levelSize = queue.length;
      for (let i = 0; i < levelSize; i++) {
        const curr = queue.shift()!;
        edges.forEach(e => {
          const neighbor = e.source === curr ? e.target : e.target === curr ? e.source : null;
          if (neighbor && !visited.has(neighbor)) {
            visited.add(neighbor);
            queue.push(neighbor);
          }
        });
      }
    }
    return nodes.filter(n => visited.has(n.id));
  }, [searchQuery, nodes, edges, depth]);

  const filteredNodeIds = useMemo(() => new Set(filteredNodes.map(n => n.id)), [filteredNodes]);
  const filteredEdges = useMemo(() => edges.filter(e => filteredNodeIds.has(e.source) && filteredNodeIds.has(e.target)), [edges, filteredNodeIds]);

  const nodeCorrelations = useCallback((nodeId: string) => {
    return edges.filter(e => e.source === nodeId || e.target === nodeId).map(e => {
      const otherId = e.source === nodeId ? e.target : e.source;
      const otherNode = nodes.find(n => n.id === otherId);
      return { ...e, otherNode };
    });
  }, [edges, nodes]);

  const exportGraph = useCallback(() => {
    const data = JSON.stringify({ nodes: filteredNodes, edges: filteredEdges }, null, 2);
    const blob = new Blob([data], { type: 'application/json' });
    const url = URL.createObjectURL(blob);
    const a = document.createElement('a');
    a.href = url;
    a.download = 'identity-correlation-graph.json';
    a.click();
    URL.revokeObjectURL(url);
  }, [filteredNodes, filteredEdges]);

  if (loading) return <div className="space-y-6"><p>Loading...</p></div>;
  if (error) return <div className="space-y-6 text-red-600">Error: {error}</div>;

  return (
    <div className="space-y-6">
      <div>
        <h1 className="text-2xl font-bold text-gray-900">Identity Correlation Graph</h1>
        <p className="mt-1 text-sm text-gray-500">Visualize identity correlations, detect synthetic identities, and explore relationship depth.</p>
      </div>

      <div className="flex flex-wrap items-center gap-4">
        <div className="flex-1 min-w-[200px]">
          <input
            type="text"
            placeholder="Search by email, phone, device, or IP..."
            value={searchQuery}
            onChange={e => setSearchQuery(e.target.value)}
            className="block w-full rounded-md border border-gray-300 px-3 py-2 text-sm"
          />
        </div>
        <div className="flex items-center gap-2">
          <label className="text-sm text-gray-600">Depth:</label>
          <input
            type="range"
            min={1}
            max={5}
            value={depth}
            onChange={e => setDepth(Number(e.target.value))}
            className="w-24"
          />
          <span className="text-sm font-medium text-gray-700">{depth}</span>
        </div>
        <button
          onClick={exportGraph}
          className="rounded-md bg-gray-600 px-4 py-2 text-sm font-medium text-white hover:bg-gray-700"
        >
          Export Graph
        </button>
      </div>

      <div className="grid grid-cols-1 gap-4 lg:grid-cols-3">
        <div className="lg:col-span-2 rounded-lg border border-gray-200 bg-white p-4 shadow-sm">
          <div className="flex items-center justify-between mb-2">
            <h3 className="text-sm font-medium text-gray-700">Correlation Graph</h3>
            <span className="text-xs text-gray-400">{filteredNodes.length} nodes, {filteredEdges.length} edges</span>
          </div>
          <svg viewBox="0 0 800 500" className="w-full h-[500px] border border-gray-100 rounded bg-gray-50">
            {filteredEdges.map((edge, i) => {
              const src = NODE_POSITIONS[edge.source];
              const tgt = NODE_POSITIONS[edge.target];
              if (!src || !tgt) return null;
              const midX = (src.x + tgt.x) / 2;
              const midY = (src.y + tgt.y) / 2;
              return (
                <g key={i}>
                  <line
                    x1={src.x} y1={src.y} x2={tgt.x} y2={tgt.y}
                    stroke={edge.confidence > 0.7 ? '#3b82f6' : edge.confidence > 0.5 ? '#f59e0b' : '#ef4444'}
                    strokeWidth={edge.confidence > 0.7 ? 2 : 1}
                    strokeOpacity={0.5}
                  />
                  <text x={midX} y={midY - 4} textAnchor="middle" className="fill-gray-400 text-[8px]">
                    {edge.correlationType}
                  </text>
                </g>
              );
            })}
            {filteredNodes.map(node => {
              const pos = NODE_POSITIONS[node.id];
              if (!pos) return null;
              return (
                <g key={node.id} onClick={() => setSelectedNode(node)} className="cursor-pointer">
                  <circle
                    cx={pos.x} cy={pos.y} r={18}
                    fill={node.isSynthetic ? '#ef4444' : NODE_COLORS[node.type] || '#6b7280'}
                    fillOpacity={0.8}
                    stroke={selectedNode?.id === node.id ? '#1e40af' : 'white'}
                    strokeWidth={selectedNode?.id === node.id ? 3 : 2}
                  />
                  {node.isSynthetic && (
                    <circle cx={pos.x + 14} cy={pos.y - 14} r={6} fill="#fbbf24" stroke="white" strokeWidth={1} />
                  )}
                  <text x={pos.x} y={pos.y + 35} textAnchor="middle" className="fill-gray-700 text-[9px] font-medium">
                    {node.label.length > 20 ? node.label.slice(0, 18) + '...' : node.label}
                  </text>
                </g>
              );
            })}
          </svg>

          <div className="mt-2 flex flex-wrap gap-3 text-xs">
            {Object.entries(NODE_COLORS).map(([type, color]) => (
              <div key={type} className="flex items-center gap-1">
                <div className={`h-3 w-3 rounded-full ${color}`} />
                <span className="text-gray-600 capitalize">{type}</span>
              </div>
            ))}
            <div className="flex items-center gap-1">
              <div className="h-3 w-3 rounded-full bg-red-500" />
              <span className="text-gray-600">Synthetic Identity</span>
            </div>
            <div className="flex items-center gap-1">
              <div className="h-3 w-3 rounded-full bg-yellow-400 border border-gray-300" />
              <span className="text-gray-600">Synthetic Alert Badge</span>
            </div>
          </div>
        </div>

        <div className="space-y-4">
          {selectedNode ? (
            <>
              <div className="rounded-lg border border-gray-200 bg-white p-4 shadow-sm">
                <h3 className="text-sm font-medium text-gray-700">Node Details</h3>
                <div className="mt-2 space-y-1 text-sm">
                  <div className="flex justify-between"><span className="text-gray-500">ID:</span><span className="font-mono text-xs">{selectedNode.id}</span></div>
                  <div className="flex justify-between"><span className="text-gray-500">Label:</span><span>{selectedNode.label}</span></div>
                  <div className="flex justify-between"><span className="text-gray-500">Type:</span><span className="capitalize">{selectedNode.type}</span></div>
                  {selectedNode.email && <div className="flex justify-between"><span className="text-gray-500">Email:</span><span className="text-xs">{selectedNode.email}</span></div>}
                  {selectedNode.phone && <div className="flex justify-between"><span className="text-gray-500">Phone:</span><span className="text-xs">{selectedNode.phone}</span></div>}
                  {selectedNode.device && <div className="flex justify-between"><span className="text-gray-500">Device:</span><span className="text-xs">{selectedNode.device}</span></div>}
                  {selectedNode.ip && <div className="flex justify-between"><span className="text-gray-500">IP:</span><span className="font-mono text-xs">{selectedNode.ip}</span></div>}
                  <div className="flex justify-between"><span className="text-gray-500">Confidence:</span><span className="font-medium">{(selectedNode.confidence * 100).toFixed(0)}%</span></div>
                  <div className="flex justify-between">
                    <span className="text-gray-500">Synthetic:</span>
                    {selectedNode.isSynthetic ? (
                      <span className="inline-flex rounded bg-red-100 px-2 py-0.5 text-xs text-red-700">ALERT</span>
                    ) : (
                      <span className="text-xs text-green-600">No</span>
                    )}
                  </div>
                </div>
              </div>

              <div className="rounded-lg border border-gray-200 bg-white p-4 shadow-sm">
                <h3 className="text-sm font-medium text-gray-700">Correlations ({nodeCorrelations(selectedNode.id).length})</h3>
                <div className="mt-2 space-y-2">
                  {nodeCorrelations(selectedNode.id).map((corr, i) => (
                    <div key={i} className="flex items-center justify-between border-b border-gray-100 pb-1 text-xs">
                      <div>
                        <span className="font-medium text-gray-700">{corr.otherNode?.label}</span>
                        <span className="ml-2 text-gray-400">{corr.correlationType}</span>
                      </div>
                      <div className="flex items-center gap-1">
                        <div className="h-1.5 w-12 rounded-full bg-gray-200">
                          <div
                            className={`h-full rounded-full ${corr.confidence > 0.7 ? 'bg-green-500' : corr.confidence > 0.5 ? 'bg-yellow-500' : 'bg-red-500'}`}
                            style={{ width: `${corr.confidence * 100}%` }}
                          />
                        </div>
                        <span className="text-gray-500">{(corr.confidence * 100).toFixed(0)}%</span>
                      </div>
                    </div>
                  ))}
                </div>
              </div>
            </>
          ) : (
            <div className="rounded-lg border border-gray-200 bg-white p-4 shadow-sm">
              <p className="text-sm text-gray-400">Click a node to view correlation details.</p>
            </div>
          )}

          <div className="rounded-lg border border-gray-200 bg-white p-4 shadow-sm">
            <h3 className="text-sm font-medium text-gray-700">Confidence Legend</h3>
            <div className="mt-2 space-y-1 text-xs">
              <div className="flex items-center gap-2"><div className="h-2 w-8 bg-green-500 rounded" /><span>High ({'>'}70%)</span></div>
              <div className="flex items-center gap-2"><div className="h-2 w-8 bg-yellow-500 rounded" /><span>Medium (50-70%)</span></div>
              <div className="flex items-center gap-2"><div className="h-2 w-8 bg-red-500 rounded" /><span>Low ({'<'}50%)</span></div>
            </div>
          </div>
        </div>
      </div>
    </div>
  );
}
