'use client';
import { useState, useEffect } from 'react';
import { useTranslations } from "@/lib/i18n";

export default function IntrospectionCacheConfigPage() {
  const t = useTranslations();

  const [enabled, setEnabled] = useState(true);
  const [activeTtl, setActiveTtl] = useState(120);
  const [inactiveTtl, setInactiveTtl] = useState(1800);
  const [maxSize, setMaxSize] = useState(10000);
  const [cacheWarming, setCacheWarming] = useState(false);
  const [invalidateToken, setInvalidateToken] = useState('');
  const [redisStatus, setRedisStatus] = useState('');
  const [stats, setStats] = useState({ hitRate: 0, missRate: 0, totalRequests: 0, cachedTokens: 0 });
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    fetch('/api/v1/auth/expiry-status', {
      headers: { 'Content-Type': 'application/json', 'X-Tenant-ID': '00000000-0000-0000-0000-000000000001' },
    })
      .then(r => { if (!r.ok) throw new Error(`HTTP ${r.status}`); return r.json(); })
      .then(data => {
        if (data) {
          if (data.enabled !== undefined) setEnabled(data.enabled);
          if (data.active_ttl) setActiveTtl(data.active_ttl);
          if (data.inactive_ttl) setInactiveTtl(data.inactive_ttl);
          if (data.max_size) setMaxSize(data.max_size);
          if (data.cache_warming !== undefined) setCacheWarming(data.cache_warming);
          if (data.redis_status) setRedisStatus(data.redis_status);
          if (data.stats) setStats(data.stats);
        }
        setLoading(false);
      })
      .catch(e => { setError(e.message); setLoading(false); });
  }, []);

  if (loading) return <div className="p-6"><p>{t("big1.introspectionCacheConfig.loading")}</p></div>;
  if (error) return <div className="p-6 text-red-600">{t("big1.introspectionCacheConfig.error")}{error}</div>;

  return (
    <div className="p-6 max-w-4xl mx-auto space-y-6">
      <div>
        <h1 className="text-2xl font-bold">{t("big1.introspectionCacheConfig.title")}</h1>
        <p className="text-gray-600">{t("big1.introspectionCacheConfig.configureTokenIntrospectionCachingForOAuthTokenValidation")}</p>
      </div>

      <div className="flex items-center gap-3">
        <span className={`px-3 py-1 rounded text-sm ${redisStatus === 'connected' ? 'bg-green-100 text-green-700' : 'bg-red-100 text-red-700'}`}>{t("big1.introspectionCacheConfig.redis")}{redisStatus}</span>
      </div>

      <section className="bg-white rounded-lg shadow p-6 space-y-4">
        <h2 className="text-lg font-semibold">{t("big1.introspectionCacheConfig.cacheSettings")}</h2>
        <label className="flex items-center justify-between">
          <span className="text-sm font-medium">{t("big1.introspectionCacheConfig.cacheEnabled")}</span>
          <input type="checkbox" checked={enabled} onChange={e => setEnabled(e.target.checked)} className="rounded" />
        </label>
        <label className="flex items-center justify-between">
          <span className="text-sm font-medium">{t("big1.introspectionCacheConfig.cacheWarmingPrePopulateOnStartup")}</span>
          <input type="checkbox" checked={cacheWarming} onChange={e => setCacheWarming(e.target.checked)} className="rounded" />
        </label>
        <div>
          <label className="text-sm font-medium">{t("big1.introspectionCacheConfig.activeTokenTTL")}{activeTtl}{t("big1.introspectionCacheConfig.s")}</label>
          <input type="range" min={30} max={300} value={activeTtl} onChange={e => setActiveTtl(parseInt(e.target.value))} className="w-full mt-2" />
          <div className="flex justify-between text-xs text-gray-400"><span>{t("big1.introspectionCacheConfig.30s")}</span><span>{t("big1.introspectionCacheConfig.300s")}</span></div>
        </div>
        <div>
          <label className="text-sm font-medium">{t("big1.introspectionCacheConfig.inactiveTokenTTL")}{inactiveTtl}{t("big1.introspectionCacheConfig.s")}</label>
          <input type="range" min={300} max={3600} step={60} value={inactiveTtl} onChange={e => setInactiveTtl(parseInt(e.target.value))} className="w-full mt-2" />
          <div className="flex justify-between text-xs text-gray-400"><span>{t("big1.introspectionCacheConfig.300s")}</span><span>{t("big1.introspectionCacheConfig.3600s")}</span></div>
        </div>
        <div>
          <label className="text-sm font-medium">{t("big1.introspectionCacheConfig.maxCacheSize")}</label>
          <input type="number" min={1000} max={100000} value={maxSize} onChange={e => setMaxSize(parseInt(e.target.value) || 10000)} className="w-32 border rounded px-2 py-1 text-sm mt-1" />
        </div>
      </section>

      <section className="bg-white rounded-lg shadow p-6 space-y-4">
        <h2 className="text-lg font-semibold">{t("big1.introspectionCacheConfig.cacheStatistics")}</h2>
        <div className="grid grid-cols-4 gap-4">
          <div className="border rounded p-4 text-center">
            <div className="text-2xl font-bold text-green-600">{stats.hitRate}%</div>
            <div className="text-sm text-gray-500">{t("big1.introspectionCacheConfig.hitRate")}</div>
          </div>
          <div className="border rounded p-4 text-center">
            <div className="text-2xl font-bold text-red-600">{stats.missRate}%</div>
            <div className="text-sm text-gray-500">{t("big1.introspectionCacheConfig.missRate")}</div>
          </div>
          <div className="border rounded p-4 text-center">
            <div className="text-2xl font-bold">{stats.totalRequests.toLocaleString()}</div>
            <div className="text-sm text-gray-500">{t("big1.introspectionCacheConfig.totalRequests")}</div>
          </div>
          <div className="border rounded p-4 text-center">
            <div className="text-2xl font-bold text-blue-600">{stats.cachedTokens.toLocaleString()}</div>
            <div className="text-sm text-gray-500">{t("big1.introspectionCacheConfig.cachedTokens")}</div>
          </div>
        </div>
      </section>

      <section className="bg-white rounded-lg shadow p-6 space-y-4">
        <h2 className="text-lg font-semibold">{t("big1.introspectionCacheConfig.invalidateByToken")}</h2>
        <div className="flex gap-3">
          <input aria-label="Paste token to invalidate..." type="text" placeholder="Paste token to invalidate..." value={invalidateToken} onChange={e => setInvalidateToken(e.target.value)} className="flex-1 border rounded px-3 py-2 text-sm font-mono" />
          <button disabled={!invalidateToken} className="px-4 py-2 bg-red-600 text-white rounded text-sm disabled:opacity-50">{t("big1.introspectionCacheConfig.invalidate")}</button>
        </div>
      </section>
    </div>
  );
}
