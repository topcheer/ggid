'use client';
import { useState, useEffect } from 'react';
import { useTranslations } from "@/lib/i18n";
import { authHeader, isAuthenticated } from "@/lib/auth-helpers";

interface BreachRecord {
  id: string;
  user: string;
  breachName: string;
  date: string;
  dataClasses: string[];
}

export default function HibpBreachCheckPage() {
  const t = useTranslations();

  const [enabled, setEnabled] = useState(true);
  const [apiKey, setApiKey] = useState('');
  const [checkOnLogin, setCheckOnLogin] = useState(true);
  const [checkOnPasswordChange, setCheckOnPasswordChange] = useState(true);
  const [checkOnRegister, setCheckOnRegister] = useState(true);
  const [autoForceReset, setAutoForceReset] = useState(false);
  const [notifyUser, setNotifyUser] = useState(true);
  const [notifyAdmin, setNotifyAdmin] = useState(true);
  const [lastCheck, setLastCheck] = useState('');
  const [breaches, setBreaches] = useState<BreachRecord[]>([]);
  const [compromisedPasswords, setCompromisedPasswords] = useState<string[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    fetch('/api/v1/auth/password-breach-check', {
      method: 'POST',
      headers: { ...authHeader(), 'Content-Type': 'application/json', 'X-Tenant-ID': '00000000-0000-0000-0000-000000000001' },
    })
      .then(r => { if (!r.ok) throw new Error(`HTTP ${r.status}`); return r.json(); })
      .then(data => {
        if (data) {
          if (data.enabled !== undefined) setEnabled(data.enabled);
          if (data.api_key) setApiKey(data.api_key);
          if (data.check_on_login !== undefined) setCheckOnLogin(data.check_on_login);
          if (data.check_on_password_change !== undefined) setCheckOnPasswordChange(data.check_on_password_change);
          if (data.check_on_register !== undefined) setCheckOnRegister(data.check_on_register);
          if (data.auto_force_reset !== undefined) setAutoForceReset(data.auto_force_reset);
          if (data.notify_user !== undefined) setNotifyUser(data.notify_user);
          if (data.notify_admin !== undefined) setNotifyAdmin(data.notify_admin);
          if (data.last_check) setLastCheck(data.last_check);
          if (data.breaches) setBreaches(data.breaches);
          if (data.compromised_passwords) setCompromisedPasswords(data.compromised_passwords);
        }
        setLoading(false);
      })
      .catch(e => { setError(e.message); setLoading(false); });
  }, []);

  if (loading) return <div className="p-6"><p>{t("big1.hibpBreachCheck.loading")}</p></div>;
  if (error) return <div className="p-6 text-red-600">{t("big1.hibpBreachCheck.error")}{error}</div>;

  return (
    <div className="p-6 max-w-4xl mx-auto space-y-6">
      <div>
        <h1 className="text-2xl font-bold">{t("big1.hibpBreachCheck.title")}</h1>
        <p className="text-gray-600">{t("big1.hibpBreachCheck.haveIBeenPwnedIntegrationForCredentialBreachMonitoring")}</p>
      </div>

      <section className="bg-white rounded-lg shadow p-6 space-y-4">
        <h2 className="text-lg font-semibold">{t("big1.hibpBreachCheck.apiConfiguration")}</h2>
        <label className="flex items-center justify-between">
          <span className="text-sm font-medium">{t("big1.hibpBreachCheck.enableHibpBreachCheck")}</span>
          <input aria-label="Enabled" type="checkbox" checked={enabled} onChange={e => setEnabled(e.target.checked)} className="rounded" />
        </label>
        <div>
          <label className="text-sm font-medium">{t("big1.hibpBreachCheck.hibpApiKey")}</label>
          <input autoComplete="current-password" type="password" placeholder="Enter HIBP API key" value={apiKey} onChange={e => setApiKey(e.target.value)} className="w-full border rounded px-3 py-2 text-sm mt-1" />
          <p className="text-xs text-gray-400 mt-1">{t("big1.hibpBreachCheck.getYourAPIKeyAtHttpsHaveibeenpwnedComAPIKey")}</p>
        </div>
        <div className="text-sm text-gray-500">{t("big1.hibpBreachCheck.lastCheck")}{lastCheck}</div>
        <button onClick={() => setLastCheck(new Date().toISOString().slice(0, 16).replace('T', ' '))} className="px-4 py-2 bg-blue-600 text-white rounded text-sm">{t("big1.hibpBreachCheck.runCheckNow")}</button>
      </section>

      <section className="bg-white rounded-lg shadow p-6 space-y-4">
        <h2 className="text-lg font-semibold">{t("big1.hibpBreachCheck.checkTriggers")}</h2>
        <div className="space-y-2">
          <label className="flex items-center justify-between"><span className="text-sm">{t("big1.hibpBreachCheck.checkOnLogin")}</span><input aria-label="Check on login" type="checkbox" checked={checkOnLogin} onChange={e => setCheckOnLogin(e.target.checked)} className="rounded" /></label>
          <label className="flex items-center justify-between"><span className="text-sm">{t("big1.hibpBreachCheck.checkOnPasswordChange")}</span><input aria-label="Check on password change" type="checkbox" checked={checkOnPasswordChange} onChange={e => setCheckOnPasswordChange(e.target.checked)} className="rounded" /></label>
          <label className="flex items-center justify-between"><span className="text-sm">{t("big1.hibpBreachCheck.checkOnRegister")}</span><input aria-label="Check on register" type="checkbox" checked={checkOnRegister} onChange={e => setCheckOnRegister(e.target.checked)} className="rounded" /></label>
        </div>
      </section>

      <section className="bg-white rounded-lg shadow p-6 space-y-4">
        <h2 className="text-lg font-semibold">{t("big1.hibpBreachCheck.responseActions")}</h2>
        <div className="space-y-2">
          <label className="flex items-center justify-between"><span className="text-sm">{t("big1.hibpBreachCheck.autoForcePasswordResetOnBreach")}</span><input aria-label="Auto force reset" type="checkbox" checked={autoForceReset} onChange={e => setAutoForceReset(e.target.checked)} className="rounded" /></label>
          <label className="flex items-center justify-between"><span className="text-sm">{t("big1.hibpBreachCheck.notifyUser")}</span><input aria-label="Notify user" type="checkbox" checked={notifyUser} onChange={e => setNotifyUser(e.target.checked)} className="rounded" /></label>
          <label className="flex items-center justify-between"><span className="text-sm">{t("big1.hibpBreachCheck.notifyAdmin")}</span><input aria-label="Notify admin" type="checkbox" checked={notifyAdmin} onChange={e => setNotifyAdmin(e.target.checked)} className="rounded" /></label>
        </div>
      </section>

      <section className="bg-white rounded-lg shadow p-6 space-y-4">
        <h2 className="text-lg font-semibold">{t("big1.hibpBreachCheck.breachHistory")}</h2>
        <table className="w-full text-sm">
          <thead className="bg-gray-50">
            <tr className="text-left">
              <th scope="col" className="p-3">{t("big1.hibpBreachCheck.user")}</th>
              <th scope="col" className="p-3">{t("big1.hibpBreachCheck.breach")}</th>
              <th scope="col" className="p-3">{t("big1.hibpBreachCheck.date")}</th>
              <th scope="col" className="p-3">{t("big1.hibpBreachCheck.dataClasses")}</th>
            </tr>
          </thead>
          <tbody>
            {breaches.map(b => (
              <tr key={b.id} className="border-b">
                <td className="p-3 font-medium">{b.user}</td>
                <td className="p-3"><span className="px-2 py-0.5 bg-red-100 text-red-700 rounded text-xs">{b.breachName}</span></td>
                <td className="p-3 text-gray-500">{b.date}</td>
                <td className="p-3"><div className="flex flex-wrap gap-1">{b.dataClasses.map(d => <span key={d} className="px-1.5 py-0.5 bg-amber-100 text-amber-700 rounded text-xs">{d}</span>)}</div></td>
              </tr>
            ))}
          </tbody>
        </table>
      </section>

      <section className="bg-white rounded-lg shadow p-6 space-y-4">
        <h2 className="text-lg font-semibold">{t("big1.hibpBreachCheck.compromisedPasswordBlocklist")}{compromisedPasswords.length})</h2>
        <div className="flex flex-wrap gap-2">
          {compromisedPasswords.map(p => (
            <span key={p} className="px-2 py-1 bg-red-50 text-red-700 rounded text-xs font-mono">{p}</span>
          ))}
        </div>
        <p className="text-xs text-gray-400">{t("big1.hibpBreachCheck.passwordsMatchingTheseEntriesAreRejectedDuringRegistrationAndPasswordChange")}</p>
      </section>
    </div>
  );
}