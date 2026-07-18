'use client';
import { useState, useEffect } from 'react';
import { Loader2 } from 'lucide-react';
import { useTranslations } from "@/lib/i18n";
import { authHeader, isAuthenticated } from "@/lib/auth-helpers";

export default function SecurityDashboardPage() {
  const t = useTranslations();

  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    const fetchData = async () => {
      try {
        const res = await fetch("/api/v1/audit/security-posture", {
          method: "GET",
          headers: { ...authHeader(),
            "Content-Type": "application/json",
            "X-Tenant-ID": "00000000-0000-0000-0000-000000000001",
          },
        });
        if (!res.ok) return null;
        // API returns posture data; will be wired when backend is ready
      } catch (e) {
        setError(e instanceof Error ? e.message : t("complianceDashboard.failedLoad"));
      } finally {
        setLoading(false);
      }
    };
    fetchData();
  }, [t]);

  // Static demo data (API not yet returning structured data)
  const score = 7.8;
  const incidents = [
    { id: 'i1', severity: 'high', title: 'Brute force attempt blocked', time: '14:30' },
    { id: 'i2', severity: 'medium', title: 'Expired certificate detected', time: '13:15' },
  ];
  const owasp = [
    { id: 'a01', name: 'Access Control', status: 'pass' },
    { id: 'a02', name: 'Cryptographic Failures', status: 'pass' },
    { id: 'a03', name: 'Injection', status: 'pass' },
    { id: 'a04', name: 'Insecure Design', status: 'warn' },
    { id: 'a05', name: 'Security Misconfiguration', status: 'pass' },
    { id: 'a06', name: 'Vulnerable Components', status: 'warn' },
    { id: 'a07', name: 'Auth Failures', status: 'pass' },
    { id: 'a08', name: 'Software/Data Integrity', status: 'pass' },
    { id: 'a09', name: 'Logging/Monitoring', status: 'pass' },
    { id: 'a10', name: 'SSRF', status: 'pass' },
  ];
  const compliance = [
    { name: 'SOC 2', status: 'compliant' },
    { name: 'GDPR', status: 'compliant' },
    { name: 'HIPAA', status: 'pending' },
    { name: 'ISO 27001', status: 'compliant' },
    { name: 'PCI DSS', status: 'n/a' },
  ];
  const threats = [
    { name: 'Brute Force', level: 'low', count: 3 },
    { name: 'Credential Stuffing', level: 'low', count: 1 },
    { name: 'Suspicious Logins', level: 'medium', count: 5 },
    { name: 'Privilege Escalation', level: 'none', count: 0 },
  ];
  const recommendations = [
    'Update expired TLS certificate for oauth.ggid.io',
    'Enable DPoP for all OAuth clients',
    'Review SoD violations for admin role holders',
    'Enable audit hash chain verification',
  ];

  const sevColor = (s: string) => s === 'high' ? 'bg-red-100 text-red-700' : s === 'medium' ? 'bg-amber-100 text-amber-700' : 'bg-green-100 text-green-700';
  const statusColor = (s: string) => s === 'pass' ? 'text-green-600' : s === 'warn' ? 'text-amber-600' : 'text-red-600';
  const compColor = (s: string) => s === 'compliant' ? 'bg-green-100 text-green-700' : s === 'pending' ? 'bg-amber-100 text-amber-700' : 'bg-gray-100 text-gray-500';
  const threatColor = (l: string) => l === 'high' ? 'bg-red-100 text-red-700' : l === 'medium' ? 'bg-amber-100 text-amber-700' : l === 'low' ? 'bg-yellow-100 text-yellow-700' : 'bg-green-100 text-green-700';
  const scoreColor = score >= 8 ? 'text-green-600' : score >= 6 ? 'text-amber-600' : 'text-red-600';

  if (loading) return <div className="flex items-center justify-center py-12"><Loader2 className="h-8 w-8 animate-spin text-blue-500" /></div>;
  if (error) return <div className="p-8 text-red-500">Error: {error}</div>;

  return (
    <div className="p-6 max-w-5xl mx-auto space-y-6">
      <div><h1 className="text-2xl font-bold">{t("secDashboard.title")}</h1><p className="text-gray-600">{t("secDashboard.subtitle")}</p></div>

      <div className="grid grid-cols-3 gap-4">
        <section className="bg-white rounded-lg shadow p-6 text-center">
          <h2 className="text-sm font-medium text-gray-500">{t("secDashboard.securityScore")}</h2>
          <div className={`text-5xl font-bold ${scoreColor} mt-2`}>{score}<span className="text-2xl text-gray-400">/10</span></div>
          <div className="mt-3 h-3 bg-gray-200 rounded-full overflow-hidden"><div className={`h-3 rounded-full ${score >= 8 ? 'bg-green-500' : 'bg-amber-500'}`} style={{ width: `${score * 10}%` }} /></div>
        </section>
        <section className="bg-white rounded-lg shadow p-6 space-y-3">
          <h2 className="text-sm font-semibold">{t("secDashboard.activeIncidents")}</h2>
          {incidents.map(i => <div key={i.id} className="flex items-center gap-2 text-sm"><span className={`px-2 py-0.5 rounded text-xs ${sevColor(i.severity)}`}>{i.severity}</span><span className="flex-1">{i.title}</span><span className="text-xs text-gray-400">{i.time}</span></div>)}
        </section>
        <section className="bg-white rounded-lg shadow p-6 space-y-2">
          <h2 className="text-sm font-semibold">{t("secDashboard.recommendations")}</h2>
          {recommendations.map((r: any, i: number) => <div key={i} className="text-xs text-gray-600 flex items-start gap-2"><span className="text-blue-600">-</span>{r}</div>)}
        </section>
      </div>

      <section className="bg-white rounded-lg shadow p-6 space-y-4">
        <h2 className="text-lg font-semibold">{t("secDashboard.threatIndicators")}</h2>
        <div className="grid grid-cols-4 gap-4">{threats.map(th => (
          <div key={th.name} className="border rounded p-3 text-center"><div className={`text-xs ${threatColor(th.level)} px-2 py-0.5 rounded inline-block capitalize`}>{th.level}</div><div className="text-sm font-medium mt-2">{th.name}</div><div className="text-2xl font-bold mt-1">{th.count}</div></div>
        ))}</div>
      </section>

      <section className="bg-white rounded-lg shadow p-6 space-y-4">
        <h2 className="text-lg font-semibold">{t("secDashboard.owaspChecklist")}</h2>
        <div className="grid grid-cols-2 gap-2">{owasp.map(o => (
          <div key={o.id} className="flex items-center gap-2 text-sm"><span className="font-mono text-xs text-gray-400">{o.id}</span><span className="flex-1">{o.name}</span><span className={`text-xs font-bold ${statusColor(o.status)}`}>{o.status === 'pass' ? 'PASS' : o.status === 'warn' ? 'WARN' : 'FAIL'}</span></div>
        ))}</div>
      </section>

      <section className="bg-white rounded-lg shadow p-6 space-y-4">
        <h2 className="text-lg font-semibold">{t("secDashboard.complianceBadges")}</h2>
        <div className="flex flex-wrap gap-3">{compliance.map(c => (
          <div key={c.name} className={`px-4 py-2 rounded-lg text-sm ${compColor(c.status)}`}><span className="font-bold">{c.name}</span> - {c.status}</div>
        ))}</div>
      </section>
    </div>
  );
}
