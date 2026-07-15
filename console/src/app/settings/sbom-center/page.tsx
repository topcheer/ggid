'use client';
import { useState, useEffect } from 'react';
import { useTranslations } from "@/lib/i18n";

interface SbomComponent {
  name: string;
  version: string;
  license: string;
  severity: string;
  description: string;
  cpe: string;
  purl: string;
  vulnerabilities: number;
}

interface DependencyNode {
  name: string;
  version: string;
  children: DependencyNode[];
}

export default function SbomCenterPage() {
  const t = useTranslations();

  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [data, setData] = useState<any[]>([]);

  useEffect(() => {
    const fetchData = async () => {
      try {
        const res = await fetch("/api/v1/audit/sbom", {
          method: "GET",
          headers: {
            "Content-Type": "application/json",
            "X-Tenant-ID": "00000000-0000-0000-0000-000000000001",
          },
        });
        if (!res.ok) throw new Error(`HTTP ${res.status}`);
        const json = await res.json();
        setData(Array.isArray(json) ? json : [json]);
      } catch (e) {
        setError(e instanceof Error ? e.message : "Failed to load");
      } finally {
        setLoading(false);
      }
    };
    fetchData();
  }, []);

  if (loading) return <div className="p-8">Loading...</div>;
  if (error) return <div className="p-8 text-red-500">Error: {error}</div>;
  if (!data || data.length === 0) return <div className="p-8 text-gray-500">No data available</div>;
  const [components, setComponents] = useState<SbomComponent[]>([
    { name: 'gin-gonic/gin', version: 'v1.10.0', license: 'MIT', severity: 'low', description: 'HTTP web framework for Go', cpe: 'cpe:2.3:a:gin-gonic:gin:1.10.0:*:*:*:*:*:*:*', purl: 'pkg:golang/github.com/gin-gonic/gin@v1.10.0', vulnerabilities: 0 },
    { name: 'golang-jwt/jwt', version: 'v5.2.1', license: 'MIT', severity: 'medium', description: 'JWT implementation for Go', cpe: 'cpe:2.3:a:golang-jwt:jwt:5.2.1:*:*:*:*:*:*:*', purl: 'pkg:golang/github.com/golang-jwt/jwt/v5@v5.2.1', vulnerabilities: 2 },
    { name: 'lib/pq', version: 'v1.10.9', license: 'BSD-2-Clause', severity: 'low', description: 'Pure Go Postgres driver', cpe: 'cpe:2.3:a:lib:pq:1.10.9:*:*:*:*:*:*:*', purl: 'pkg:golang/github.com/lib/pq@v1.10.9', vulnerabilities: 0 },
    { name: 'redis/go-redis', version: 'v9.6.1', license: 'BSD-2-Clause', severity: 'high', description: 'Redis client for Go', cpe: 'cpe:2.3:a:redis:go-redis:9.6.1:*:*:*:*:*:*:*', purl: 'pkg:golang/github.com/redis/go-redis/v9@v9.6.1', vulnerabilities: 4 },
    { name: 'nats-io/nats.go', version: 'v1.36.0', license: 'Apache-2.0', severity: 'low', description: 'NATS client for Go', cpe: 'cpe:2.3:a:nats-io:nats.go:1.36.0:*:*:*:*:*:*:*', purl: 'pkg:golang/github.com/nats-io/nats.go@v1.36.0', vulnerabilities: 1 },
  ]);

  const [selectedComponent, setSelectedComponent] = useState<SbomComponent | null>(null);
  const [showJson, setShowJson] = useState(false);

  const severityColor = (s: string): string =>
    s === 'high' ? 'bg-red-100 text-red-700' :
    s === 'medium' ? 'bg-amber-100 text-amber-700' :
    'bg-green-100 text-green-700';

  const totalVulns = components.reduce((sum, c) => sum + c.vulnerabilities, 0);

  const dependencyTree: DependencyNode = {
    name: 'ggid', version: 'v1.0.0', children: [
      { name: 'gin-gonic/gin', version: 'v1.10.0', children: [
        { name: 'golang/protobuf', version: 'v1.5.4', children: [] },
      ]},
      { name: 'golang-jwt/jwt', version: 'v5.2.1', children: [] },
      { name: 'redis/go-redis', version: 'v9.6.1', children: [] },
    ]
  };

  const renderTree = (node: DependencyNode, depth: number): React.ReactNode => (
    <li key={node.name} className="ml-4">
      <div className="flex items-center gap-2 text-sm">
        <span className="font-medium">{node.name}</span>
        <span className="text-xs text-gray-400">{node.version}</span>
      </div>
      {node.children.length > 0 && (
        <ul className="border-l border-gray-200 ml-3 mt-1 space-y-1">
          {node.children.map(c => renderTree(c, depth + 1))}
        </ul>
      )}
    </li>
  );

  const cycloneDxJson = JSON.stringify({
    bomFormat: 'CycloneDX',
    specVersion: '1.5',
    version: 1,
    metadata: { component: { type: 'application', name: 'ggid', version: '1.0.0' } },
    components: components.map(c => ({
      type: 'library', name: c.name, version: c.version,
      licenses: [{ license: { id: c.license } }],
      cpe: c.cpe, purl: c.purl,
    }))
  }, null, 2);

  return (
    <div className="p-6 max-w-5xl mx-auto space-y-6">
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-2xl font-bold">{t("sbom.title")}</h1>
          <p className="text-gray-600">Software Bill of Materials - CycloneDX format with vulnerability tracking.</p>
        </div>
        <div className="flex gap-2">
          <button onClick={() => setShowJson(!showJson)} className="px-3 py-1.5 border rounded text-sm">
            {showJson ? 'Hide JSON' : 'View CycloneDX JSON'}
          </button>
          <button
            onClick={() => {
              const blob = new Blob([cycloneDxJson], { type: 'application/json' });
              const url = URL.createObjectURL(blob);
              const a = document.createElement('a');
              a.href = url; a.download = 'sbom.cyclonedx.json'; a.click();
            }}
            className="px-4 py-2 bg-blue-600 text-white rounded text-sm"
          >Export SBOM</button>
        </div>
      </div>

      <div className="grid grid-cols-3 gap-4">
        <div className="bg-white rounded-lg shadow p-4 text-center">
          <div className="text-2xl font-bold">{components.length}</div>
          <div className="text-sm text-gray-500">Components</div>
        </div>
        <div className="bg-white rounded-lg shadow p-4 text-center">
          <div className="text-2xl font-bold text-amber-600">{totalVulns}</div>
          <div className="text-sm text-gray-500">Vulnerabilities</div>
        </div>
        <div className="bg-white rounded-lg shadow p-4 text-center">
          <div className="text-2xl font-bold text-red-600">{components.filter(c => c.severity === 'high').length}</div>
          <div className="text-sm text-gray-500">High Severity</div>
        </div>
      </div>

      {showJson && (
        <section className="bg-white rounded-lg shadow p-6 space-y-4">
          <h2 className="text-lg font-semibold">CycloneDX JSON</h2>
          <pre className="bg-gray-900 text-green-400 rounded p-4 text-xs overflow-x-auto max-h-96">{cycloneDxJson}</pre>
        </section>
      )}

      <section className="bg-white rounded-lg shadow overflow-hidden">
        <table className="w-full text-sm">
          <thead className="bg-gray-50">
            <tr className="text-left">
              <th className="p-3">Name</th>
              <th className="p-3">Version</th>
              <th className="p-3">License</th>
              <th className="p-3">Severity</th>
              <th className="p-3">Vulns</th>
              <th className="p-3">Action</th>
            </tr>
          </thead>
          <tbody>
            {components.map(c => (
              <tr key={c.purl} className="border-b hover:bg-gray-50">
                <td className="p-3 font-medium">{c.name}</td>
                <td className="p-3 font-mono text-xs text-gray-500">{c.version}</td>
                <td className="p-3"><span className="px-2 py-0.5 bg-gray-100 rounded text-xs">{c.license}</span></td>
                <td className="p-3"><span className={`px-2 py-0.5 rounded text-xs capitalize ${severityColor(c.severity)}`}>{c.severity}</span></td>
                <td className="p-3">
                  {c.vulnerabilities > 0 ? (
                    <span className="px-2 py-0.5 bg-red-50 text-red-700 rounded text-xs font-bold">{c.vulnerabilities}</span>
                  ) : (
                    <span className="text-green-600 text-xs">0</span>
                  )}
                </td>
                <td className="p-3">
                  <button onClick={() => setSelectedComponent(c)} className="text-blue-600 text-xs hover:underline">Details</button>
                </td>
              </tr>
            ))}
          </tbody>
        </table>
      </section>

      <section className="bg-white rounded-lg shadow p-6 space-y-4">
        <h2 className="text-lg font-semibold">Dependency Tree</h2>
        <ul className="space-y-1">{renderTree(dependencyTree, 0)}</ul>
      </section>

      {selectedComponent && (
        <div className="fixed inset-0 bg-black/50 flex items-center justify-center z-50">
          <div className="bg-white rounded-lg p-6 max-w-lg w-full mx-4 space-y-4">
            <div className="flex items-center justify-between">
              <h2 className="text-lg font-semibold">{selectedComponent.name}</h2>
              <button onClick={() => setSelectedComponent(null)} className="text-gray-400 hover:text-gray-600">X</button>
            </div>
            <div className="space-y-2 text-sm">
              <div><span className="text-gray-500">Version:</span> {selectedComponent.version}</div>
              <div><span className="text-gray-500">License:</span> {selectedComponent.license}</div>
              <div><span className="text-gray-500">Description:</span> {selectedComponent.description}</div>
              <div><span className="text-gray-500">CPE:</span> <span className="font-mono text-xs">{selectedComponent.cpe}</span></div>
              <div><span className="text-gray-500">PURL:</span> <span className="font-mono text-xs">{selectedComponent.purl}</span></div>
              <div><span className="text-gray-500">Vulnerabilities:</span> <span className={`font-bold ${selectedComponent.vulnerabilities > 0 ? 'text-red-600' : 'text-green-600'}`}>{selectedComponent.vulnerabilities}</span></div>
              <div><span className="text-gray-500">Severity:</span> <span className={`px-2 py-0.5 rounded text-xs capitalize ${severityColor(selectedComponent.severity)}`}>{selectedComponent.severity}</span></div>
            </div>
          </div>
        </div>
      )}
    </div>
  );
}