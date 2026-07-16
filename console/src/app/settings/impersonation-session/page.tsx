'use client';
import { useState, useEffect } from 'react';
import { useTranslations } from "@/lib/i18n";

interface ImpSession { id: string; impersonator: string; target: string; startedAt: string; expires: string; reason: string; }

export default function ImpersonationSessionPage() {
  const t = useTranslations();

  const [sessions, setSessions] = useState<ImpSession[]>([]);
  const [showForm, setShowForm] = useState(false);
  const [newSession, setNewSession] = useState({ target: '', reason: '', duration: 30 });
  const [auditLog, setAuditLog] = useState([] as { time: string; action: string; target: string }[]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    fetch('/api/v1/auth/impersonate', {
      method: 'POST',
      headers: { "Authorization": `Bearer ${localStorage.getItem("ggid_access_token") || ""}`, 'Content-Type': 'application/json', 'X-Tenant-ID': '00000000-0000-0000-0000-000000000001' },
      body: JSON.stringify({ action: 'list' }),
    })
      .then(r => { if (!r.ok) throw new Error(`HTTP ${r.status}`); return r.json(); })
      .then(data => {
        if (data) {
          if (data.sessions) setSessions(data.sessions);
          if (data.audit_log) setAuditLog(data.audit_log);
        }
        setLoading(false);
      })
      .catch(e => { setError(e.message); setLoading(false); });
  }, []);

  const start = () => {
    setSessions(prev => [...prev, { id: `s${prev.length + 1}`, impersonator: 'admin@ggid.io', target: newSession.target, startedAt: new Date().toISOString().slice(11, 16), expires: `${new Date(Date.now() + newSession.duration * 60000).toISOString().slice(11, 16)}`, reason: newSession.reason }]);
    setShowForm(false); setNewSession({ target: '', reason: '', duration: 30 });
  };
  const endSession = (id: string) => setSessions(prev => prev.filter(s => s.id !== id));

  if (loading) return <div className="p-6"><p>{t("big1.impersonationSession.loading")}</p></div>;
  if (error) return <div className="p-6 text-red-600">{t("big1.impersonationSession.error")}{error}</div>;

  return (
    <div className="p-6 max-w-4xl mx-auto space-y-6">
      <div className="flex items-center justify-between">
        <div><h1 className="text-2xl font-bold">{t("big1.impersonationSession.title")}</h1><p className="text-gray-600">{t("big1.impersonationSession.monitorAndManageActiveAdminImpersonationSessions")}</p></div>
        <button onClick={() => setShowForm(!showForm)} className="px-4 py-2 bg-blue-600 text-white rounded text-sm">{showForm ? t("big1.impersonationSession.cancel") : t("big1.impersonationSession.startSession")}</button>
      </div>

      <section className="bg-amber-50 border border-amber-200 rounded p-4 space-y-2">
        <h2 className="text-sm font-semibold text-amber-800">{t("big1.impersonationSession.safetyGuardrails")}</h2>
        <div className="grid grid-cols-2 gap-2 text-xs text-amber-700">
          <div>{t("big1.impersonationSession.maxDuration60Minutes")}</div><div>{t("big1.impersonationSession.reasonRequiredYes")}</div>
          <div>{t("big1.impersonationSession.auditLevelFull")}</div><div>{t("big1.impersonationSession.autoRevokeOnIdle15min")}</div>
          <div>{t("big1.impersonationSession.restrictedToRolesAdminSecurityAdmin")}</div><div>{t("big1.impersonationSession.targetConsentOptional")}</div>
        </div>
      </section>

      {showForm && (
        <section className="bg-white rounded-lg shadow p-6 space-y-4">
          <h2 className="text-lg font-semibold">{t("big1.impersonationSession.startImpersonation")}</h2>
          <div><label className="text-sm font-medium">{t("big1.impersonationSession.targetUser")}</label><input aria-label="user@ggid.io" type="text" placeholder="user@ggid.io" value={newSession.target} onChange={e => setNewSession(prev => ({ ...prev, target: e.target.value }))} className="w-full border rounded px-3 py-2 text-sm mt-1" /></div>
          <div><label className="text-sm font-medium">{t("big1.impersonationSession.reason")}</label><input aria-label="Why are you impersonating?" type="text" placeholder="Why are you impersonating?" value={newSession.reason} onChange={e => setNewSession(prev => ({ ...prev, reason: e.target.value }))} className="w-full border rounded px-3 py-2 text-sm mt-1" /></div>
          <div><label className="text-sm font-medium">{t("big1.impersonationSession.durationMinutes")}</label><input aria-label="new Session" type="number" min={5} max={60} value={newSession.duration} onChange={e => setNewSession(prev => ({ ...prev, duration: parseInt(e.target.value) || 30 }))} className="w-24 border rounded px-2 py-1 text-sm mt-1" /></div>
          <button aria-label="action" onClick={start} disabled={!newSession.target || !newSession.reason} className="px-4 py-2 bg-blue-600 text-white rounded text-sm disabled:opacity-50">{t("big1.impersonationSession.start")}</button>
        </section>
      )}

      <section className="bg-white rounded-lg shadow overflow-hidden">
        <h2 className="text-lg font-semibold p-6 pb-4">{t("big1.impersonationSession.activeSessions")}</h2>
        {sessions.length === 0 ? <p className="text-sm text-gray-400 px-6 pb-6">{t("big1.impersonationSession.noActiveImpersonationSessions")}</p> : (
          <table className="w-full text-sm"><thead className="bg-gray-50"><tr className="text-left"><th className="p-3">{t("big1.impersonationSession.impersonator")}</th><th className="p-3">{t("big1.impersonationSession.target")}</th><th className="p-3">{t("big1.impersonationSession.started")}</th><th className="p-3">{t("big1.impersonationSession.expires")}</th><th className="p-3">{t("big1.impersonationSession.reason")}</th><th className="p-3">{t("big1.impersonationSession.action")}</th></tr></thead>
            <tbody>{sessions.map(s => (
              <tr key={s.id} className="border-b"><td className="p-3 font-medium">{s.impersonator}</td><td className="p-3">{s.target}</td><td className="p-3 text-gray-500">{s.startedAt}</td><td className="p-3 text-gray-500">{s.expires}</td><td className="p-3 text-gray-600">{s.reason}</td><td className="p-3"><button onClick={() => endSession(s.id)} className="text-red-600 text-xs hover:underline">{t("big1.impersonationSession.end")}</button></td></tr>
            ))}</tbody></table>
        )}
      </section>

      <section className="bg-white rounded-lg shadow p-6 space-y-4">
        <h2 className="text-lg font-semibold">{t("big1.impersonationSession.sessionAuditLog")}</h2>
        <div className="space-y-2">{auditLog.map((l, i) => <div key={i} className="flex items-center gap-3 text-sm border-b pb-2"><span className="text-xs text-gray-500">{l.time}</span><span className="font-mono text-xs">{l.action}</span><span className="text-xs text-gray-500">{l.target}</span></div>)}</div>
      </section>
    </div>
  );
}