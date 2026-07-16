'use client';
import { useState, useEffect } from 'react';
import { useTranslations } from "@/lib/i18n";

interface ImpersonationRecord {
  id: string;
  impersonator: string;
  target: string;
  reason: string;
  startedAt: string;
  duration: string;
  status: string;
}

export default function ImpersonationConfigPage() {
  const t = useTranslations();

  const [allowedImpersonators, setAllowedImpersonators] = useState<string[]>([]);
  const [requireReason, setRequireReason] = useState(true);
  const [maxDuration, setMaxDuration] = useState(60);
  const [auditLevel, setAuditLevel] = useState('full');
  const [requireTargetConsent, setRequireTargetConsent] = useState(false);
  const [autoRevokeOnIdle, setAutoRevokeOnIdle] = useState(true);
  const [restrictToRoles, setRestrictToRoles] = useState<string[]>([]);
  const [newImpersonator, setNewImpersonator] = useState('');
  const [newRole, setNewRole] = useState('');
  const [history, setHistory] = useState<ImpersonationRecord[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    fetch('/api/v1/auth/impersonation/config', {
      headers: { "Authorization": `Bearer ${localStorage.getItem("ggid_access_token") || ""}`, 'Content-Type': 'application/json', 'X-Tenant-ID': '00000000-0000-0000-0000-000000000001' },
    })
      .then(r => { if (!r.ok) throw new Error(`HTTP ${r.status}`); return r.json(); })
      .then(data => {
        if (data) {
          if (data.allowed_impersonators) setAllowedImpersonators(data.allowed_impersonators);
          if (data.require_reason !== undefined) setRequireReason(data.require_reason);
          if (data.max_duration) setMaxDuration(data.max_duration);
          if (data.audit_level) setAuditLevel(data.audit_level);
          if (data.require_target_consent !== undefined) setRequireTargetConsent(data.require_target_consent);
          if (data.auto_revoke_on_idle !== undefined) setAutoRevokeOnIdle(data.auto_revoke_on_idle);
          if (data.restrict_to_roles) setRestrictToRoles(data.restrict_to_roles);
          if (data.history) setHistory(data.history);
        }
        setLoading(false);
      })
      .catch(e => { setError(e.message); setLoading(false); });
  }, []);

  const addImpersonator = () => { if (newImpersonator) { setAllowedImpersonators(prev => [...prev, newImpersonator]); setNewImpersonator(''); } };
  const removeImpersonator = (u: string) => setAllowedImpersonators(prev => prev.filter(x => x !== u));
  const addRole = () => { if (newRole) { setRestrictToRoles(prev => [...prev, newRole]); setNewRole(''); } };
  const removeRole = (r: string) => setRestrictToRoles(prev => prev.filter(x => x !== r));

  const statusColor = (s: string): string =>
    s === 'active' ? 'bg-green-100 text-green-700' : 'bg-gray-200 text-gray-600';

  if (loading) return <div className="p-6"><p>{t("big1.impersonationConfig.loading")}</p></div>;
  if (error) return <div className="p-6 text-red-600">{t("big1.impersonationConfig.error")}{error}</div>;

  return (
    <div className="p-6 max-w-5xl mx-auto space-y-6">
      <div>
        <h1 className="text-2xl font-bold">{t("big1.impersonationConfig.title")}</h1>
        <p className="text-gray-600">{t("big1.impersonationConfig.configureAdminImpersonationPoliciesRestrictionsAndAuditTrail")}</p>
      </div>

      <div className="grid grid-cols-2 gap-4">
        <label className="flex items-center justify-between bg-white rounded-lg shadow p-4">
          <span className="text-sm font-medium">{t("big1.impersonationConfig.requireReason")}</span>
          <input aria-label="Require reason" type="checkbox" checked={requireReason} onChange={e => setRequireReason(e.target.checked)} className="rounded" />
        </label>
        <label className="flex items-center justify-between bg-white rounded-lg shadow p-4">
          <span className="text-sm font-medium">{t("big1.impersonationConfig.requireTargetConsent")}</span>
          <input aria-label="Require target consent" type="checkbox" checked={requireTargetConsent} onChange={e => setRequireTargetConsent(e.target.checked)} className="rounded" />
        </label>
        <label className="flex items-center justify-between bg-white rounded-lg shadow p-4">
          <span className="text-sm font-medium">{t("big1.impersonationConfig.autoRevokeOnIdle")}</span>
          <input aria-label="Auto revoke on idle" type="checkbox" checked={autoRevokeOnIdle} onChange={e => setAutoRevokeOnIdle(e.target.checked)} className="rounded" />
        </label>
        <div className="bg-white rounded-lg shadow p-4">
          <label className="text-sm font-medium">{t("big1.impersonationConfig.auditLevel")}</label>
          <select aria-label="audit Level" value={auditLevel} onChange={e => setAuditLevel(e.target.value)} className="w-full border rounded px-3 py-2 text-sm mt-1">
            <option value="full">{t("big1.impersonationConfig.fullAllActionsLogged")}</option>
            <option value="summary">{t("big1.impersonationConfig.summaryStartEndOnly")}</option>
            <option value="disabled">{t("big1.impersonationConfig.disabledNotRecommended")}</option>
          </select>
        </div>
      </div>

      <section className="bg-white rounded-lg shadow p-6 space-y-4">
        <h2 className="text-lg font-semibold">{t("big1.impersonationConfig.maxDuration")}{maxDuration}{t("big1.impersonationConfig.minutes")}</h2>
        <input aria-label="Max duration" type="range" min={5} max={480} step={5} value={maxDuration} onChange={e => setMaxDuration(parseInt(e.target.value))} className="w-full" />
        <div className="flex justify-between text-xs text-gray-400"><span>{t("big1.impersonationConfig.5min")}</span><span>{t("big1.impersonationConfig.8h")}</span></div>
      </section>

      <section className="bg-white rounded-lg shadow p-6 space-y-4">
        <h2 className="text-lg font-semibold">{t("big1.impersonationConfig.allowedImpersonators")}</h2>
        <div className="space-y-2">
          {allowedImpersonators.map(u => (
            <div key={u} className="flex items-center gap-2"><span className="font-mono text-sm flex-1">{u}</span><button onClick={() => removeImpersonator(u)}aria-label={"Remove impersonator " + u} className="text-red-600 text-xs">{t("big1.impersonationConfig.remove")}</button></div>
          ))}
        </div>
        <div className="flex gap-2">
          <input aria-label="user@ggid.io" type="text" placeholder="user@ggid.io" value={newImpersonator} onChange={e => setNewImpersonator(e.target.value)} className="flex-1 border rounded px-2 py-1 text-sm" />
          <button onClick={addImpersonator} aria-label="Add impersonator" className="px-3 py-1 bg-blue-600 text-white rounded text-sm">{t("big1.impersonationConfig.add")}</button>
        </div>
      </section>

      <section className="bg-white rounded-lg shadow p-6 space-y-4">
        <h2 className="text-lg font-semibold">{t("big1.impersonationConfig.restrictedToRoles")}</h2>
        <p className="text-sm text-gray-500">{t("big1.impersonationConfig.onlyUsersWithTheseRolesCanBeImpersonated")}</p>
        <div className="flex flex-wrap gap-2">
          {restrictToRoles.map(r => (
            <div key={r} className="flex items-center gap-1"><span className="px-2 py-1 bg-purple-100 text-purple-700 rounded text-xs">{r}</span><button onClick={() => removeRole(r)}aria-label={"Remove role " + r} className="text-red-600 text-xs">{t("big1.impersonationConfig.x")}</button></div>
          ))}
        </div>
        <div className="flex gap-2">
          <input aria-label="role-name" type="text" placeholder="role-name" value={newRole} onChange={e => setNewRole(e.target.value)} className="flex-1 border rounded px-2 py-1 text-sm" />
          <button onClick={addRole} aria-label="Add restricted role" className="px-3 py-1 bg-blue-600 text-white rounded text-sm">{t("big1.impersonationConfig.add")}</button>
        </div>
      </section>

      <section className="bg-white rounded-lg shadow p-6 space-y-4">
        <h2 className="text-lg font-semibold">{t("big1.impersonationConfig.impersonationHistory")}</h2>
        <table className="w-full text-sm">
          <thead className="bg-gray-50">
            <tr className="text-left">
              <th scope="col" className="p-3">{t("big1.impersonationConfig.impersonator")}</th><th className="p-3">{t("big1.impersonationConfig.target")}</th><th className="p-3">{t("big1.impersonationConfig.reason")}</th><th className="p-3">{t("big1.impersonationConfig.started")}</th><th className="p-3">{t("big1.impersonationConfig.duration")}</th><th className="p-3">{t("big1.impersonationConfig.status")}</th>
            </tr>
          </thead>
          <tbody>
            {history.map(r => (
              <tr key={r.id} className="border-b">
                <td className="p-3 font-medium">{r.impersonator}</td>
                <td className="p-3">{r.target}</td>
                <td className="p-3 text-gray-600">{r.reason}</td>
                <td className="p-3 text-gray-500">{r.startedAt}</td>
                <td className="p-3 text-gray-500">{r.duration}</td>
                <td className="p-3"><span className={`px-2 py-0.5 rounded text-xs ${statusColor(r.status)}`}>{r.status}</span></td>
              </tr>
            ))}
          </tbody>
        </table>
      </section>
    </div>
  );
}