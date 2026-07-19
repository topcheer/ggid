'use client';
import { useState, useEffect } from 'react';
import { useTranslations } from "@/lib/i18n";
import { authHeader, isAuthenticated } from "@/lib/auth-helpers";

export default function IpReputationConfigPage() {
  const t = useTranslations();

  const [enabled, setEnabled] = useState(true);
  const [provider, setProvider] = useState('internal');
  const [checkInterval, setCheckInterval] = useState(300);
  const [blockThreshold, setBlockThreshold] = useState(80);
  const [suspiciousThreshold, setSuspiciousThreshold] = useState(50);
  const [allowlist, setAllowlist] = useState<string[]>([]);
  const [blocklist, setBlocklist] = useState<string[]>([]);
  const [blockedCountries, setBlockedCountries] = useState<string[]>(['CN', 'RU']);
  const [asnBlocklist, setAsnBlocklist] = useState<string[]>(['AS12345', 'AS67890']);
  const [newIp, setNewIp] = useState('');
  const [newCountry, setNewCountry] = useState('');
  const [newAsn, setNewAsn] = useState('');
  const [listMode, setListMode] = useState<'allow' | 'block'>('allow');

  const countries = ['CN', 'RU', 'KP', 'IR', 'SY', 'CU', 'VN', 'TR', 'BR', 'IN'];
  const [stats, setStats] = useState({ checkedToday: 0, blockedToday: 0, suspiciousToday: 0, avgScore: 0 });
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    fetch('/api/v1/auth/tor-vpn/detect', {
      method: 'POST',
      headers: { ...authHeader(), 'Content-Type': 'application/json', 'X-Tenant-ID': '00000000-0000-0000-0000-000000000001' },
    })
      .then(r => { if (!r.ok) throw new Error(`HTTP ${r.status}`); return r.json(); })
      .then(data => {
        if (data) {
          if (data.enabled !== undefined) setEnabled(data.enabled);
          if (data.provider) setProvider(data.provider);
          if (data.check_interval) setCheckInterval(data.check_interval);
          if (data.block_threshold) setBlockThreshold(data.block_threshold);
          if (data.suspicious_threshold) setSuspiciousThreshold(data.suspicious_threshold);
          if (data.allowlist) setAllowlist(data.allowlist);
          if (data.blocklist) setBlocklist(data.blocklist);
          if (data.blocked_countries) setBlockedCountries(data.blocked_countries);
          if (data.asn_blocklist) setAsnBlocklist(data.asn_blocklist);
          if (data.stats) setStats(data.stats);
        }
        setLoading(false);
      })
      .catch(e => { setError(e.message); setLoading(false); });
  }, []);

  const addIp = () => {
    if (listMode === 'allow') setAllowlist(prev => [...prev, newIp]);
    else setBlocklist(prev => [...prev, newIp]);
    setNewIp('');
  };
  const removeIp = (ip: string, mode: 'allow' | 'block') => {
    if (mode === 'allow') setAllowlist(prev => prev.filter(i => i !== ip));
    else setBlocklist(prev => prev.filter(i => i !== ip));
  };
  const toggleCountry = (c: string) => setBlockedCountries(prev => prev.includes(c) ? prev.filter(x => x !== c) : [...prev, c]);
  const addAsn = () => { if (newAsn) { setAsnBlocklist(prev => [...prev, newAsn]); setNewAsn(''); } };
  const removeAsn = (a: string) => setAsnBlocklist(prev => prev.filter(x => x !== a));

  if (loading) return <div className="p-6"><p>{t("big1.ipReputationConfig.loading")}</p></div>;
  if (error) return <div className="p-6 text-red-600">{t("big1.ipReputationConfig.error")}{error}</div>;

  return (
    <div className="p-6 max-w-4xl mx-auto space-y-6">
      <div>
        <h1 className="text-2xl font-bold">{t("big1.ipReputationConfig.title")}</h1>
        <p className="text-gray-600">{t("big1.ipReputationConfig.configureIPReputationScoringGeoBlockingAndASNFiltering")}</p>
      </div>

      <div className="grid grid-cols-4 gap-4">
        <div className="bg-white rounded-lg shadow p-4 text-center"><div className="text-2xl font-bold">{stats.checkedToday.toLocaleString()}</div><div className="text-sm text-gray-500">{t("big1.ipReputationConfig.checked24h")}</div></div>
        <div className="bg-white rounded-lg shadow p-4 text-center"><div className="text-2xl font-bold text-red-600">{stats.blockedToday}</div><div className="text-sm text-gray-500">{t("big1.ipReputationConfig.blocked24h")}</div></div>
        <div className="bg-white rounded-lg shadow p-4 text-center"><div className="text-2xl font-bold text-amber-600">{stats.suspiciousToday}</div><div className="text-sm text-gray-500">{t("big1.ipReputationConfig.suspicious24h")}</div></div>
        <div className="bg-white rounded-lg shadow p-4 text-center"><div className="text-2xl font-bold">{stats.avgScore}</div><div className="text-sm text-gray-500">{t("big1.ipReputationConfig.avgScore")}</div></div>
      </div>

      <section className="bg-white rounded-lg shadow p-6 space-y-4">
        <h2 className="text-lg font-semibold">{t("big1.ipReputationConfig.generalSettings")}</h2>
        <label className="flex items-center justify-between"><span className="text-sm font-medium">{t("big1.ipReputationConfig.enableIpReputation")}</span><input aria-label="Enabled" type="checkbox" checked={enabled} onChange={e => setEnabled(e.target.checked)} className="rounded" /></label>
        <div className="grid grid-cols-2 gap-4">
          <div><label className="text-sm font-medium">{t("big1.ipReputationConfig.reputationProvider")}</label><select aria-label="provider" value={provider} onChange={e => setProvider(e.target.value)} className="w-full border rounded px-3 py-2 text-sm mt-1"><option value="internal">{t("big1.ipReputationConfig.internalBuiltInDatabase")}</option><option value="external">{t("big1.ipReputationConfig.externalAPIAbuseIPDB")}</option><option value="hybrid">{t("big1.ipReputationConfig.hybridInternalExternal")}</option></select></div>
          <div><label className="text-sm font-medium">{t("big1.ipReputationConfig.checkIntervalS")}</label><input aria-label="check Interval" type="number" min={60} value={checkInterval} onChange={e => setCheckInterval(parseInt(e.target.value) || 300)} className="w-full border rounded px-2 py-1 text-sm mt-1" /></div>
        </div>
        <div className="grid grid-cols-2 gap-4">
          <div><label className="text-sm font-medium">{t("big1.ipReputationConfig.blockThreshold")}{blockThreshold}</label><input aria-label="block Threshold" type="range" min={0} max={100} value={blockThreshold} onChange={e => setBlockThreshold(parseInt(e.target.value))} className="w-full mt-2" /></div>
          <div><label className="text-sm font-medium">{t("big1.ipReputationConfig.suspiciousThreshold")}{suspiciousThreshold}</label><input aria-label="suspicious Threshold" type="range" min={0} max={100} value={suspiciousThreshold} onChange={e => setSuspiciousThreshold(parseInt(e.target.value))} className="w-full mt-2" /></div>
        </div>
      </section>

      <div className="grid grid-cols-2 gap-6">
        <section className="bg-white rounded-lg shadow p-6 space-y-4">
          <h2 className="text-lg font-semibold">{t("big1.ipReputationConfig.allowlist")}</h2>
          <div className="space-y-2">{allowlist.map(ip => <div key={ip} className="flex items-center gap-2"><span className="font-mono text-xs flex-1">{ip}</span><button onClick={() => removeIp(ip, 'allow')} className="text-red-600 text-xs">{t("big1.ipReputationConfig.remove")}</button></div>)}</div>
          <div className="flex gap-2"><input aria-label="CIDR" type="text" placeholder="CIDR" value={listMode === 'allow' ? newIp : ''} onChange={e => { setNewIp(e.target.value); setListMode('allow'); }} className="flex-1 border rounded px-2 py-1 text-sm font-mono" /><button onClick={() => { setListMode('allow'); addIp(); }} className="px-3 py-1 bg-green-600 text-white rounded text-sm">{t("big1.ipReputationConfig.add")}</button></div>
        </section>
        <section className="bg-white rounded-lg shadow p-6 space-y-4">
          <h2 className="text-lg font-semibold">{t("big1.ipReputationConfig.blocklist")}</h2>
          <div className="space-y-2">{blocklist.map(ip => <div key={ip} className="flex items-center gap-2"><span className="font-mono text-xs flex-1">{ip}</span><button onClick={() => removeIp(ip, 'block')} className="text-red-600 text-xs">{t("big1.ipReputationConfig.remove")}</button></div>)}</div>
          <div className="flex gap-2"><input aria-label="CIDR" type="text" placeholder="CIDR" value={listMode === 'block' ? newIp : ''} onChange={e => { setNewIp(e.target.value); setListMode('block'); }} className="flex-1 border rounded px-2 py-1 text-sm font-mono" /><button onClick={() => { setListMode('block'); addIp(); }} className="px-3 py-1 bg-red-600 text-white rounded text-sm">{t("big1.ipReputationConfig.add")}</button></div>
        </section>
      </div>

      <section className="bg-white rounded-lg shadow p-6 space-y-4">
        <h2 className="text-lg font-semibold">{t("big1.ipReputationConfig.geoBlockCountryLevel")}</h2>
        <div className="flex flex-wrap gap-2">
          {countries.map(c => (
            <label key={c} className={`px-3 py-1 rounded text-sm cursor-pointer ${blockedCountries.includes(c) ? 'bg-red-100 text-red-700' : 'bg-gray-100 text-gray-600'}`}>
              <input aria-label="Blocked countries" type="checkbox" checked={blockedCountries.includes(c)} onChange={() => toggleCountry(c)} className="hidden" />{c}
            </label>
          ))}
        </div>
      </section>

      <section className="bg-white rounded-lg shadow p-6 space-y-4">
        <h2 className="text-lg font-semibold">{t("big1.ipReputationConfig.asnBlocklist")}</h2>
        <div className="flex flex-wrap gap-2">
          {asnBlocklist.map(a => <div key={a} className="flex items-center gap-1"><span className="px-2 py-1 bg-red-50 text-red-700 rounded text-xs font-mono">{a}</span><button onClick={() => removeAsn(a)} className="text-red-600 text-xs">{t("big1.ipReputationConfig.x")}</button></div>)}
        </div>
        <div className="flex gap-2"><input aria-label="AS12345" type="text" placeholder="AS12345" value={newAsn} onChange={e => setNewAsn(e.target.value)} className="flex-1 border rounded px-2 py-1 text-sm font-mono" /><button onClick={addAsn} className="px-3 py-1 bg-red-600 text-white rounded text-sm" aria-label="Action">{t("big1.ipReputationConfig.add")}</button></div>
      </section>
    </div>
  );
}