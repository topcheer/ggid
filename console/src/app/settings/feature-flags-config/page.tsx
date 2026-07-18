'use client';

import { useState, useCallback, useEffect } from 'react';
import { useTranslations } from '@/lib/i18n';
import { authHeader, isAuthenticated } from "@/lib/auth-helpers";

interface FeatureFlag {
  id: string;
  name: string;
  description: string;
  enabled: boolean;
  environment: 'dev' | 'staging' | 'prod' | 'all';
  percentage: number;
  created: string;
  targetedUsers: string[];
  targetedRoles: string[];
  targetedTenants: string[];
  killSwitch: boolean;
}

interface FlagAuditEntry {
  timestamp: string;
  actor: string;
  flagName: string;
  action: string;
  oldValue: string;
  newValue: string;
}

const INITIAL_FLAGS: FeatureFlag[] = [];

const AUDIT_LOG: FlagAuditEntry[] = [];

const ENV_COLORS: Record<string, string> = {
  dev: 'bg-blue-100 text-blue-700',
  staging: 'bg-yellow-100 text-yellow-700',
  prod: 'bg-red-100 text-red-700',
  all: 'bg-gray-100 text-gray-700',
};

export default function FeatureFlagsConfigPage() {
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [flags, setFlags] = useState<FeatureFlag[]>(INITIAL_FLAGS);
  const [auditLog, setAuditLog] = useState<FlagAuditEntry[]>(AUDIT_LOG);
  const [activeTab, setActiveTab] = useState<'flags' | 'create' | 'audit'>('flags');
  const [newName, setNewName] = useState('');
  const [newDescription, setNewDescription] = useState('');
  const [newDefaultState, setNewDefaultState] = useState(false);
  const [selectedFlag, setSelectedFlag] = useState<FeatureFlag | null>(null);
  const t = useTranslations();

  const toggleFlag = useCallback((id: string) => {
    setFlags(flags.map(f => f.id === id ? { ...f, enabled: !f.enabled, killSwitch: false } : f));
  }, [flags]);

  const killSwitchFlag = useCallback((id: string) => {
    setFlags(flags.map(f => f.id === id ? { ...f, enabled: false, killSwitch: true, percentage: 0 } : f));
  }, [flags]);

  const updatePercentage = useCallback((id: string, percentage: number) => {
    setFlags(flags.map(f => f.id === id ? { ...f, percentage } : f));
  }, [flags]);

  const updateEnvironment = useCallback((id: string, env: FeatureFlag['environment']) => {
    setFlags(flags.map(f => f.id === id ? { ...f, environment: env } : f));
  }, [flags]);

  const createFlag = useCallback(() => {
    if (!newName.trim()) return;
    const newFlag: FeatureFlag = {
      id: `ff-${String(flags.length + 1).padStart(3, '0')}`,
      name: newName.trim(),
      description: newDescription.trim(),
      enabled: newDefaultState,
      environment: 'dev',
      percentage: newDefaultState ? 100 : 0,
      created: new Date().toISOString(),
      targetedUsers: [],
      targetedRoles: [],
      targetedTenants: [],
      killSwitch: false,
    };
    setFlags([newFlag, ...flags]);
    setNewName('');
    setNewDescription('');
    setNewDefaultState(false);
    setActiveTab('flags');
  }, [flags, newName, newDescription, newDefaultState]);

  useEffect(() => {
    fetch("/api/v1/policy/feature-flags", {
      headers: { ...authHeader(), "Content-Type": "application/json", "X-Tenant-ID": "00000000-0000-0000-0000-000000000001" },
    })
      .then(res => { if (!res.ok) return null; return res.json(); })
      .then(data => {
        setFlags(data.flags || data.items || []);
        setAuditLog(data.auditLog || data.audit_log || []);
        setLoading(false);
      })
      .catch(err => { setError(err.message); setLoading(false); });
  }, []);

  if (loading) return (<div className="p-6"><h1 className="text-2xl font-bold mb-4">{t("featureFlagsConfig.title")}</h1><p>{t("featureFlagsConfig.loading")}</p></div>);
  if (error) return (<div className="p-6"><h1 className="text-2xl font-bold mb-4">{t("featureFlagsConfig.title")}</h1><p className="text-red-600">Error: {error}</p></div>);
  return (
    <div className="space-y-6">
      <div>
        <h1 className="text-2xl font-bold text-gray-900">Feature Flags Configuration</h1>
        <p className="mt-1 text-sm text-gray-500">{t("featureFlagsConfig.subtitle")}</p>
      </div>

      <div className="flex gap-2 border-b border-gray-200">
        {(['flags', 'create', 'audit'] as const).map(tab => (
          <button
            key={tab}
            onClick={() => setActiveTab(tab)}
            className={`px-4 py-2 text-sm font-medium border-b-2 ${
              activeTab === tab ? 'border-blue-500 text-blue-600' : 'border-transparent text-gray-500 hover:text-gray-700'
            }`}
          >
            {tab === 'flags' ? `Flags (${flags.length})` : tab === 'create' ? 'Create Flag' : 'Audit Log'}
          </button>
        ))}
      </div>

      {activeTab === 'flags' && (
        <div className="space-y-4">
          <div className="rounded-lg border border-gray-200 bg-white p-4 shadow-sm">
            <table className="w-full text-sm">
              <thead>
                <tr className="border-b border-gray-200 text-left text-xs text-gray-500">
                  <th scope="col" className="pb-2">{t("featureFlagsConfig.name")}</th>
                  <th scope="col" className="pb-2">{t("featureFlagsConfig.description")}</th>
                  <th scope="col" className="pb-2">{t("featureFlagsConfig.enabled")}</th>
                  <th scope="col" className="pb-2">{t("featureFlagsConfig.environment")}</th>
                  <th scope="col" className="pb-2">{t("featureFlagsConfig.rollout")}</th>
                  <th scope="col" className="pb-2">{t("featureFlagsConfig.created")}</th>
                  <th scope="col" className="pb-2">{t("featureFlagsConfig.actions")}</th>
                </tr>
              </thead>
              <tbody>
                {flags.map(f => (
                  <tr
                    key={f.id}
                    className={`border-b border-gray-100 cursor-pointer hover:bg-gray-50 ${selectedFlag?.id === f.id ? 'bg-blue-50' : ''}`}
                    onClick={() => setSelectedFlag(f)}
                  >
                    <td className="py-2">
                      <div className="font-mono text-xs font-medium">{f.name}</div>
                      {f.killSwitch && <span className="inline-flex rounded bg-red-100 px-1.5 py-0.5 text-[10px] text-red-700 mt-0.5">{t("featureFlagsConfig.kill")}</span>}
                    </td>
                    <td className="py-2 text-xs text-gray-600 max-w-[200px] truncate">{f.description}</td>
                    <td className="py-2">
                      <button
                        onClick={(e) => { e.stopPropagation(); toggleFlag(f.id); }}
                        className={`relative inline-flex h-5 w-9 items-center rounded-full transition ${f.enabled ? 'bg-green-500' : 'bg-gray-200'}`}
                      >
                        <span className={`inline-block h-3 w-3 transform rounded-full bg-white transition ${f.enabled ? 'translate-x-5' : 'translate-x-1'}`} />
                      </button>
                    </td>
                    <td className="py-2">
                      <span className={`inline-flex rounded px-2 py-0.5 text-xs ${ENV_COLORS[f.environment]}`}>{f.environment}</span>
                    </td>
                    <td className="py-2">
                      <div className="flex items-center gap-1">
                        <div className="h-1.5 w-12 rounded-full bg-gray-200">
                          <div className={`h-full rounded-full ${f.percentage === 100 ? 'bg-green-500' : f.percentage > 0 ? 'bg-blue-500' : 'bg-gray-300'}`} style={{ width: `${f.percentage}%` }} />
                        </div>
                        <span className="text-xs text-gray-500">{f.percentage}%</span>
                      </div>
                    </td>
                    <td className="py-2 text-xs text-gray-500">{f.created.slice(0, 10)}</td>
                    <td className="py-2">
                      <button
                        onClick={(e) => { e.stopPropagation(); killSwitchFlag(f.id); }}
                        className="rounded bg-red-100 px-2 py-0.5 text-xs font-medium text-red-700 hover:bg-red-200"
                      >{t("featureFlagsConfig.killBtn")}</button>
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>

          {selectedFlag && (
            <div className="rounded-lg border border-gray-200 bg-white p-4 shadow-sm">
              <div className="flex items-center justify-between">
                <h3 className="text-sm font-medium text-gray-700">{selectedFlag.name} — Detailed Config</h3>
                <span className={`inline-flex rounded px-2 py-0.5 text-xs ${ENV_COLORS[selectedFlag.environment]}`}>{selectedFlag.environment}</span>
              </div>

              <div className="mt-4 grid grid-cols-1 gap-4 md:grid-cols-2">
                <div>
                  <label className="block text-xs font-medium text-gray-600">Environment</label>
                  <select
                    value={selectedFlag.environment}
                    onChange={e => { updateEnvironment(selectedFlag.id, e.target.value as FeatureFlag['environment']); setSelectedFlag({ ...selectedFlag, environment: e.target.value as FeatureFlag['environment'] }); }}
                    className="mt-1 block w-full rounded-md border border-gray-300 px-3 py-2 text-sm"
                  >
                    <option value="dev">dev</option>
                    <option value="staging">staging</option>
                    <option value="prod">prod</option>
                    <option value="all">all</option>
                  </select>
                </div>
                <div>
                  <label className="block text-xs font-medium text-gray-600">{t("featureFlagsConfig.percentageRollout")} {selectedFlag.percentage}%</label>
                  <input
                    type="range"
                    min={0}
                    max={100}
                    step={5}
                    value={selectedFlag.percentage}
                    onChange={e => { const v = Number(e.target.value); updatePercentage(selectedFlag.id, v); setSelectedFlag({ ...selectedFlag, percentage: v }); }}
                    className="mt-2 w-full"
                  />
                </div>
              </div>

              <div className="mt-4 grid grid-cols-1 gap-4 md:grid-cols-3">
                <div>
                  <label className="block text-xs font-medium text-gray-600">{t("featureFlagsConfig.targetedUsers")} ({selectedFlag.targetedUsers.length})</label>
                  <div className="mt-1 space-y-1">
                    {selectedFlag.targetedUsers.length === 0 ? (
                      <span className="text-xs text-gray-400">{t("featureFlagsConfig.none")}</span>
                    ) : (
                      selectedFlag.targetedUsers.map(u => <div key={u} className="font-mono text-xs bg-gray-100 rounded px-2 py-0.5">{u}</div>)
                    )}
                  </div>
                </div>
                <div>
                  <label className="block text-xs font-medium text-gray-600">Targeted Roles ({selectedFlag.targetedRoles.length})</label>
                  <div className="mt-1 space-y-1">
                    {selectedFlag.targetedRoles.length === 0 ? (
                      <span className="text-xs text-gray-400">{t("featureFlagsConfig.none")}</span>
                    ) : (
                      selectedFlag.targetedRoles.map(r => <div key={r} className="font-mono text-xs bg-gray-100 rounded px-2 py-0.5">{r}</div>)
                    )}
                  </div>
                </div>
                <div>
                  <label className="block text-xs font-medium text-gray-600">Targeted Tenants ({selectedFlag.targetedTenants.length})</label>
                  <div className="mt-1 space-y-1">
                    {selectedFlag.targetedTenants.length === 0 ? (
                      <span className="text-xs text-gray-400">{t("featureFlagsConfig.none")}</span>
                    ) : (
                      selectedFlag.targetedTenants.map(t => <div key={t} className="font-mono text-xs bg-gray-100 rounded px-2 py-0.5">{t}</div>)
                    )}
                  </div>
                </div>
              </div>

              <div className="mt-4 flex gap-2">
                <button
                  onClick={() => toggleFlag(selectedFlag.id)}
                  className={`rounded-md px-4 py-2 text-sm font-medium text-white ${selectedFlag.enabled ? 'bg-gray-500 hover:bg-gray-600' : 'bg-green-600 hover:bg-green-700'}`}
                >
                  {selectedFlag.enabled ? 'Disable Flag' : 'Enable Flag'}
                </button>
                <button
                  onClick={() => killSwitchFlag(selectedFlag.id)}
                  className="rounded-md bg-red-600 px-4 py-2 text-sm font-medium text-white hover:bg-red-700"
                >
                  Instant Kill Switch
                </button>
              </div>
            </div>
          )}
        </div>
      )}

      {activeTab === 'create' && (
        <div className="rounded-lg border border-gray-200 bg-white p-4 shadow-sm">
          <h3 className="text-sm font-medium text-gray-700">Create New Feature Flag</h3>
          <div className="mt-4 space-y-4">
            <div>
              <label className="block text-xs font-medium text-gray-600">Flag Name</label>
              <input
                type="text"
                value={newName}
                onChange={e => setNewName(e.target.value)}
                placeholder="e.g. new_login_flow"
                className="mt-1 block w-full rounded-md border border-gray-300 px-3 py-2 text-sm font-mono"
              />
            </div>
            <div>
              <label className="block text-xs font-medium text-gray-600">Description</label>
              <textarea
                value={newDescription}
                onChange={e => setNewDescription(e.target.value)}
                rows={3}
                placeholder="Describe what this flag controls..."
                className="mt-1 block w-full rounded-md border border-gray-300 px-3 py-2 text-sm"
              />
            </div>
            <div>
              <label className="block text-xs font-medium text-gray-600">Default State</label>
              <div className="mt-1 flex items-center gap-3">
                <button
                  onClick={() => setNewDefaultState(true)}
                  className={`rounded-md px-4 py-2 text-sm font-medium ${newDefaultState ? 'bg-green-600 text-white' : 'bg-gray-200 text-gray-700'}`}
                >Enabled</button>
                <button
                  onClick={() => setNewDefaultState(false)}
                  className={`rounded-md px-4 py-2 text-sm font-medium ${!newDefaultState ? 'bg-gray-600 text-white' : 'bg-gray-200 text-gray-700'}`}
                >Disabled</button>
              </div>
            </div>
            <button
              onClick={createFlag}
              disabled={!newName.trim()}
              className="rounded-md bg-blue-600 px-4 py-2 text-sm font-medium text-white hover:bg-blue-700 disabled:opacity-50"
            >Create Flag</button>
          </div>
        </div>
      )}

      {activeTab === 'audit' && (
        <div className="rounded-lg border border-gray-200 bg-white p-4 shadow-sm">
          <h3 className="text-sm font-medium text-gray-700">Flag Audit Log ({auditLog.length})</h3>
          <div className="mt-2 space-y-2">
            {auditLog.map((entry: any, i: number) => (
              <div key={i} className="flex items-center gap-3 border-b border-gray-100 pb-2 text-sm">
                <span className="text-xs text-gray-400 font-mono w-40">{entry.timestamp.slice(0, 19).replace('T', ' ')}</span>
                <span className="font-medium text-gray-700 w-32">{entry.actor}</span>
                <span className="font-mono text-xs text-blue-600 w-40">{entry.flagName}</span>
                <span className="text-gray-600">{entry.action}</span>
                <span className="text-xs text-gray-400">{entry.oldValue} {'->'} {entry.newValue}</span>
              </div>
            ))}
          </div>
        </div>
      )}
    </div>
  );
}
