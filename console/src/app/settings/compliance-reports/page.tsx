'use client';
import { useState, useEffect } from 'react';
import { Loader2 } from 'lucide-react';
import { useTranslations } from "@/lib/i18n";
import { authHeader, isAuthenticated } from "@/lib/auth-helpers";

interface Section { name: string; score: number; status: string; gaps: number; }
interface Remediation { id: string; gap: string; priority: string; status: string; }

export default function ComplianceReportsPage() {
  const t = useTranslations();

  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [framework, setFramework] = useState('SOC2');
  const [dateRange, setDateRange] = useState({ start: '2026-06-01', end: '2026-07-12' });
  const [generating, setGenerating] = useState(false);
  const [generated, setGenerated] = useState(false);

  const sections: Section[] = [
    { name: 'Access Control', score: 92, status: 'compliant', gaps: 1 },
    { name: 'Audit Logging', score: 98, status: 'compliant', gaps: 0 },
    { name: 'Data Protection', score: 85, status: 'partial', gaps: 3 },
    { name: 'Incident Response', score: 78, status: 'partial', gaps: 5 },
    { name: 'Risk Assessment', score: 88, status: 'compliant', gaps: 2 },
  ];

  const remediations: Remediation[] = [
    { id: 'rm1', gap: 'Data-at-rest encryption for backup storage', priority: 'high', status: 'in_progress' },
    { id: 'rm2', gap: 'Documented incident response runbook', priority: 'medium', status: 'open' },
    { id: 'rm3', gap: 'Quarterly access review records', priority: 'medium', status: 'resolved' },
  ];

  useEffect(() => {
    const fetchData = async () => {
      try {
        const res = await fetch("/api/v1/audit/compliance-report", {
          method: "GET",
          headers: { ...authHeader(),
            "Content-Type": "application/json",
            "X-Tenant-ID": "00000000-0000-0000-0000-000000000001",
          },
        });
        if (!res.ok) return null;
        // Compliance data will be wired when API returns structured data
      } catch (e) {
        setError(e instanceof Error ? e.message : "Failed to load");
      } finally {
        setLoading(false);
      }
    };
    fetchData();
  }, []);

  const frameworks = ['SOC2', 'ISO27001', 'HIPAA', 'GDPR', 'PCI-DSS'];
  const overallScore = Math.round(sections.reduce((s, x) => s + x.score, 0) / sections.length);
  const statusColor = (s: string) => s === 'compliant' ? 'bg-green-100 text-green-700' : s === 'partial' ? 'bg-amber-100 text-amber-700' : 'bg-red-100 text-red-700';
  const priorityColor = (p: string) => p === 'high' ? 'bg-red-100 text-red-700' : 'bg-amber-100 text-amber-700';
  const remediationColor = (s: string) => s === 'resolved' ? 'bg-green-100 text-green-700' : s === 'in_progress' ? 'bg-blue-100 text-blue-700' : 'bg-gray-100 text-gray-600';

  const generate = () => { setGenerating(true); setTimeout(() => { setGenerating(false); setGenerated(true); }, 1500); };
  const exportReport = (format: 'pdf' | 'csv') => { const a = document.createElement('a'); a.href = '#'; a.download = `compliance-${framework}.${format}`; a.click(); };

  if (loading) return <div className="flex items-center justify-center py-12"><Loader2 className="h-8 w-8 animate-spin text-blue-500" /></div>;
  if (error) return <div className="p-8 text-red-500">Error: {error}</div>;

  return (
    <div className="p-6 max-w-5xl mx-auto space-y-6">
      <div><h1 className="text-2xl font-bold">{t("complianceReports.title")}</h1><p className="text-gray-600">Generate compliance reports, track gaps, and manage remediation.</p></div>

      <section className="bg-white rounded-lg shadow p-6 space-y-4">
        <h2 className="text-lg font-semibold">Report Configuration</h2>
        <div className="grid grid-cols-3 gap-4">
          <div><label className="text-sm font-medium">Framework</label><select aria-label="Framework" value={framework} onChange={e => setFramework(e.target.value)} className="w-full border rounded px-3 py-2 text-sm mt-1">{frameworks.map(f => <option key={f} value={f}>{f}</option>)}</select></div>
          <div><label className="text-sm font-medium">Start Date</label><input aria-label="date Range" type="date" value={dateRange.start} onChange={e => setDateRange(prev => ({ ...prev, start: e.target.value }))} className="w-full border rounded px-3 py-2 text-sm mt-1" /></div>
          <div><label className="text-sm font-medium">End Date</label><input aria-label="date Range" type="date" value={dateRange.end} onChange={e => setDateRange(prev => ({ ...prev, end: e.target.value }))} className="w-full border rounded px-3 py-2 text-sm mt-1" /></div>
        </div>
        <div className="flex gap-3">
          <button aria-label="action" onClick={generate} disabled={generating} className="px-4 py-2 bg-blue-600 text-white rounded text-sm disabled:opacity-50">{generating ? 'Generating...' : 'Generate Report'}</button>
          <label className="flex items-center gap-2 text-sm"><input aria-label="Toggle option" type="checkbox" className="rounded" />Schedule monthly</label>
        </div>
      </section>

      {generated && (
        <>
          <div className="grid grid-cols-2 gap-4">
            <section className="bg-white rounded-lg shadow p-6 text-center"><div className="text-5xl font-bold text-green-600">{overallScore}</div><div className="text-sm text-gray-500">Compliance Score ({framework})</div></section>
            <section className="bg-white rounded-lg shadow p-6 flex items-center justify-around">
              <button onClick={() => exportReport('pdf')} className="px-4 py-2 border rounded text-sm">Export PDF</button>
              <button onClick={() => exportReport('csv')} className="px-4 py-2 border rounded text-sm">Export CSV</button>
            </section>
          </div>

          <section className="bg-white rounded-lg shadow p-6 space-y-4">
            <h2 className="text-lg font-semibold">Sections</h2>
            <div className="space-y-3">{sections.map(s => (
              <div key={s.name} className="flex items-center gap-4">
                <span className="text-sm w-40">{s.name}</span>
                <div className="flex-1 bg-gray-200 rounded-full h-3 overflow-hidden"><div className={`h-3 rounded-full ${s.score >= 90 ? 'bg-green-500' : s.score >= 75 ? 'bg-amber-500' : 'bg-red-500'}`} style={{ width: `${s.score}%` }} /></div>
                <span className="text-sm font-bold w-12">{s.score}%</span>
                <span className={`px-2 py-0.5 rounded text-xs ${statusColor(s.status)}`}>{s.status}</span>
                <span className="text-xs text-gray-500 w-16">{s.gaps} gap(s)</span>
              </div>
            ))}</div>
          </section>

          <section className="bg-white rounded-lg shadow p-6 space-y-4">
            <h2 className="text-lg font-semibold">Remediation Tracking</h2>
            <table className="w-full text-sm"><thead className="bg-gray-50"><tr className="text-left"><th className="p-3">Gap</th><th className="p-3">Priority</th><th className="p-3">Status</th></tr></thead>
              <tbody>{remediations.map(r => (
                <tr key={r.id} className="border-b"><td className="p-3">{r.gap}</td><td className="p-3"><span className={`px-2 py-0.5 rounded text-xs ${priorityColor(r.priority)}`}>{r.priority}</span></td><td className="p-3"><span className={`px-2 py-0.5 rounded text-xs ${remediationColor(r.status)}`}>{r.status}</span></td></tr>
              ))}</tbody></table>
          </section>
        </>
      )}
    </div>
  );
}
