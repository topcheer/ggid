'use client';
import { useState, useEffect } from 'react';
import { useTranslations } from "@/lib/i18n";

export default function SessionManagementConfigPage() {
  const t = useTranslations();
  const [idleTimeout, setIdleTimeout] = useState(30);
  const [absoluteTimeout, setAbsoluteTimeout] = useState(480);
  const [maxConcurrent, setMaxConcurrent] = useState(3);
  const [fixationPrevention, setFixationPrevention] = useState(true);
  const [bindIp, setBindIp] = useState(true);
  const [bindDevice, setBindDevice] = useState(false);
  const [bindGeo, setBindGeo] = useState(false);
  const [stepUpTimeout, setStepUpTimeout] = useState(300);
  const [storage, setStorage] = useState('redis');
  const [logoutBehavior, setLogoutBehavior] = useState('all_sessions');
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    fetch('/api/v1/auth/session-timeout/config', {
      headers: { "Authorization": `Bearer ${localStorage.getItem("ggid_access_token") || ""}`, 'Content-Type': 'application/json', 'X-Tenant-ID': '00000000-0000-0000-0000-000000000001' },
    })
      .then(r => { if (!r.ok) throw new Error(`HTTP ${r.status}`); return r.json(); })
      .then(data => {
        if (data) {
          if (data.idle_timeout) setIdleTimeout(data.idle_timeout);
          if (data.absolute_timeout) setAbsoluteTimeout(data.absolute_timeout);
          if (data.max_concurrent) setMaxConcurrent(data.max_concurrent);
          if (data.fixation_prevention !== undefined) setFixationPrevention(data.fixation_prevention);
          if (data.bind_ip !== undefined) setBindIp(data.bind_ip);
          if (data.bind_device !== undefined) setBindDevice(data.bind_device);
          if (data.bind_geo !== undefined) setBindGeo(data.bind_geo);
          if (data.step_up_timeout) setStepUpTimeout(data.step_up_timeout);
          if (data.storage) setStorage(data.storage);
          if (data.logout_behavior) setLogoutBehavior(data.logout_behavior);
        }
        setLoading(false);
      })
      .catch(e => { setError(e.message); setLoading(false); });
  }, []);

  if (loading) return <div className="p-6"><p>{t("sessionMgmtConfig.loading")}</p></div>;
  if (error) return <div className="p-6 text-red-600">Error: {error}</div>;

  return (
    <div className="p-6 max-w-4xl mx-auto space-y-6">
      <div><h1 className="text-2xl font-bold">{t("sessionMgmtConfig.title")}</h1><p className="text-gray-600">{t("sessionMgmtConfig.subtitle")}</p></div>

      <section className="bg-white rounded-lg shadow p-6 space-y-4">
        <h2 className="text-lg font-semibold">{t("sessionMgmtConfig.sessionLifetime")}</h2>
        <div><label className="text-sm font-medium">{t("sessionMgmtConfig.idleTimeout")}: {idleTimeout}min</label><input aria-label="idle Timeout" type="range" min={5} max={120} value={idleTimeout} onChange={e => setIdleTimeout(parseInt(e.target.value))} className="w-full mt-2" /><div className="flex justify-between text-xs text-gray-400"><span>5min</span><span>2h</span></div></div>
        <div><label className="text-sm font-medium">{t("sessionMgmtConfig.absoluteTimeout")}: {absoluteTimeout}min ({Math.round(absoluteTimeout / 60)}h)</label><input aria-label="absolute Timeout" type="range" min={60} max={1440} step={30} value={absoluteTimeout} onChange={e => setAbsoluteTimeout(parseInt(e.target.value))} className="w-full mt-2" /><div className="flex justify-between text-xs text-gray-400"><span>1h</span><span>24h</span></div></div>
        <div><label className="text-sm font-medium">{t("sessionMgmtConfig.maxConcurrent")}</label><input aria-label="max Concurrent" type="number" min={1} max={20} value={maxConcurrent} onChange={e => setMaxConcurrent(parseInt(e.target.value) || 3)} className="w-24 border rounded px-2 py-1 text-sm mt-1" /></div>
      </section>

      <div className="grid grid-cols-2 gap-4">
        <label className="flex items-center justify-between bg-white rounded-lg shadow p-4"><span className="text-sm font-medium">{t("sessionMgmtConfig.sessionFixation")}</span><input aria-label="Fixation prevention" type="checkbox" checked={fixationPrevention} onChange={e => setFixationPrevention(e.target.checked)} className="rounded" /></label>
        <div className="bg-white rounded-lg shadow p-4"><label className="text-sm font-medium">{t("sessionMgmtConfig.stepUpAuth")}: {stepUpTimeout}s</label><input aria-label="step Up Timeout" type="range" min={60} max={1800} step={60} value={stepUpTimeout} onChange={e => setStepUpTimeout(parseInt(e.target.value))} className="w-full mt-2" /></div>
      </div>

      <section className="bg-white rounded-lg shadow p-6 space-y-4">
        <h2 className="text-lg font-semibold">{t("sessionMgmtConfig.sessionBinding")}</h2>
        <p className="text-sm text-gray-500">{t("sessionMgmtConfig.bindingDesc")}</p>
        <div className="space-y-2">
          <label className="flex items-center justify-between"><span className="text-sm">{t("sessionMgmtConfig.bindToIp")}</span><input aria-label="Bind ip" type="checkbox" checked={bindIp} onChange={e => setBindIp(e.target.checked)} className="rounded" /></label>
          <label className="flex items-center justify-between"><span className="text-sm">{t("sessionMgmtConfig.bindToDevice")}</span><input aria-label="Bind device" type="checkbox" checked={bindDevice} onChange={e => setBindDevice(e.target.checked)} className="rounded" /></label>
          <label className="flex items-center justify-between"><span className="text-sm">{t("sessionMgmtConfig.bindToGeo")}</span><input aria-label="Bind geo" type="checkbox" checked={bindGeo} onChange={e => setBindGeo(e.target.checked)} className="rounded" /></label>
        </div>
        {(bindIp || bindDevice || bindGeo) && <p className="text-xs text-amber-600">{t("sessionMgmtConfig.bindingWarning")}</p>}
      </section>

      <div className="grid grid-cols-2 gap-4">
        <div className="bg-white rounded-lg shadow p-6 space-y-4">
          <h2 className="text-lg font-semibold">{t("sessionMgmtConfig.sessionStorage")}</h2>
          <select aria-label="Storage" value={storage} onChange={e => setStorage(e.target.value)} className="w-full border rounded px-3 py-2 text-sm">
            <option value="redis">{t("sessionMgmtConfig.storageRedis")}</option>
            <option value="jwt">{t("sessionMgmtConfig.storageJwt")}</option>
            <option value="hybrid">{t("sessionMgmtConfig.storageHybrid")}</option>
          </select>
          {storage === 'jwt' && <p className="text-xs text-amber-600">JWT sessions cannot be revoked before expiry.</p>}
        </div>
        <div className="bg-white rounded-lg shadow p-6 space-y-4">
          <h2 className="text-lg font-semibold">Logout Behavior</h2>
          <select aria-label="Logout behavior" value={logoutBehavior} onChange={e => setLogoutBehavior(e.target.value)} className="w-full border rounded px-3 py-2 text-sm">
            <option value="current">Current session only</option>
            <option value="all_sessions">All sessions for user</option>
            <option value="all_devices">All sessions across all devices</option>
          </select>
        </div>
      </div>
    </div>
  );
}