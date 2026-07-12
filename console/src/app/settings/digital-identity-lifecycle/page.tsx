'use client';
import { useState, useEffect } from 'react';

interface LifecycleStage {
  key: string;
  label: string;
  color: string;
}

interface UserJourney {
  id: string;
  user: string;
  stage: string;
  timestamp: string;
  event: string;
}

interface DeprovisionChecklist {
  item: string;
  done: boolean;
}

interface ProvisioningRule {
  id: string;
  trigger: string;
  action: string;
  enabled: boolean;
}

export default function DigitalIdentityLifecyclePage() {
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const stages: LifecycleStage[] = [
    { key: 'provision', label: 'Provision', color: 'bg-blue-600' },
    { key: 'activate', label: 'Activate', color: 'bg-green-600' },
    { key: 'modify', label: 'Modify', color: 'bg-amber-500' },
    { key: 'suspend', label: 'Suspend', color: 'bg-orange-500' },
    { key: 'deprovision', label: 'Deprovision', color: 'bg-red-600' },
  ];

  const [journeys, setJourneys] = useState<UserJourney[]>([]);

  const [checklist, setChecklist] = useState<DeprovisionChecklist[]>([
    { item: 'Revoke all OAuth tokens', done: true },
    { item: 'Disable SSO access', done: true },
    { item: 'Revoke VPN credentials', done: true },
    { item: 'Archive mailbox', done: false },
    { item: 'Revoke device certificates', done: false },
    { item: 'Remove from groups', done: false },
    { item: 'Transfer owned resources', done: false },
    { item: 'Notify manager', done: false },
  ]);

  const [rules, setRules] = useState<ProvisioningRule[]>([]);

  useEffect(() => {
    fetch("/api/v1/identity/user-lifecycle/stages", {
      headers: { "Content-Type": "application/json", "X-Tenant-ID": "00000000-0000-0000-0000-000000000001" },
    })
      .then(res => { if (!res.ok) throw new Error(`HTTP ${res.status}`); return res.json(); })
      .then(data => {
        setJourneys(data.journeys || data.items || []);
        setRules(data.rules || []);
        setLoading(false);
      })
      .catch(err => { setError(err.message); setLoading(false); });
  }, []);

  const [selectedStage, setSelectedStage] = useState('all');
  const [bulkTarget, setBulkTarget] = useState('suspend');

  const toggleChecklist = (idx: number) => {
    setChecklist(prev => prev.map((c, i) => i === idx ? { ...c, done: !c.done } : c));
  };

  const toggleRule = (id: string) => {
    setRules(prev => prev.map(r => r.id === id ? { ...r, enabled: !r.enabled } : r));
  };

  const filteredJourneys = selectedStage === 'all' ? journeys : journeys.filter(j => j.stage === selectedStage);
  const checklistProgress = Math.round((checklist.filter(c => c.done).length / checklist.length) * 100);

  if (loading) return (
    <div className="p-6"><h1 className="text-2xl font-bold mb-4">Digital Identity Lifecycle</h1><p>Loading...</p></div>
  );
  if (error) return (
    <div className="p-6"><h1 className="text-2xl font-bold mb-4">Digital Identity Lifecycle</h1><p className="text-red-600">Error: {error}</p></div>
  );
  return (
    <div className="p-6 max-w-5xl mx-auto space-y-6">
      <div>
        <h1 className="text-2xl font-bold">Digital Identity Lifecycle</h1>
        <p className="text-gray-600">Manage the full identity lifecycle from provisioning to deprovisioning.</p>
      </div>

      <section className="bg-white rounded-lg shadow p-6 space-y-4">
        <h2 className="text-lg font-semibold">Lifecycle Stages</h2>
        <div className="flex items-center gap-2">
          {stages.map((s, idx) => (
            <div key={s.key} className="flex items-center gap-2">
              <div className={`px-4 py-2 rounded-lg text-white text-sm font-medium ${s.color}`}>{s.label}</div>
              {idx < stages.length - 1 && <span className="text-gray-300 text-xl">{'->'}</span>}
            </div>
          ))}
        </div>
      </section>

      <section className="bg-white rounded-lg shadow p-6 space-y-4">
        <h2 className="text-lg font-semibold">Account State Machine</h2>
        <div className="grid grid-cols-5 gap-3 text-center text-xs">
          {stages.map(s => (
            <div key={s.key} className={`p-3 rounded border-2 ${selectedStage === s.key ? 'border-blue-500' : 'border-gray-200'}`}>
              <div className={`w-3 h-3 rounded-full mx-auto mb-1 ${s.color}`} />
              {s.label}
            </div>
          ))}
        </div>
        <p className="text-xs text-gray-400">States: provision {'->'} activate {'->'} modify (loop) {'->'} suspend {'->'} deprovision. Accounts can go from suspend back to activate.</p>
      </section>

      <section className="bg-white rounded-lg shadow p-6 space-y-4">
        <div className="flex items-center gap-3">
          <h2 className="text-lg font-semibold">User Journey Timeline</h2>
          <select value={selectedStage} onChange={e => setSelectedStage(e.target.value)} className="border rounded px-2 py-1 text-sm">
            <option value="all">All Stages</option>
            {stages.map(s => <option key={s.key} value={s.key}>{s.label}</option>)}
          </select>
        </div>
        <div className="space-y-3">
          {filteredJourneys.map(j => (
            <div key={j.id} className="flex gap-4 items-start">
              <div className={`w-3 h-3 rounded-full mt-1.5 ${stages.find(s => s.key === j.stage)?.color || 'bg-gray-400'}`} />
              <div className="flex-1">
                <div className="text-sm font-medium">{j.user} <span className="px-2 py-0.5 bg-gray-100 rounded text-xs capitalize">{j.stage}</span></div>
                <div className="text-xs text-gray-500">{j.timestamp} - {j.event}</div>
              </div>
            </div>
          ))}
        </div>
      </section>

      <section className="bg-white rounded-lg shadow p-6 space-y-4">
        <h2 className="text-lg font-semibold">Automated Provisioning Rules (HR-Triggered)</h2>
        <div className="space-y-2">
          {rules.map(r => (
            <div key={r.id} className="flex items-center gap-3 border-b pb-2">
              <label className="flex items-center gap-2">
                <input type="checkbox" checked={r.enabled} onChange={() => toggleRule(r.id)} className="rounded" />
                <span className={`text-sm ${r.enabled ? '' : 'text-gray-400'}`}>{r.trigger}</span>
              </label>
              <span className="text-gray-300">{'->'}</span>
              <span className="text-sm text-blue-600">{r.action}</span>
            </div>
          ))}
        </div>
      </section>

      <div className="grid grid-cols-2 gap-6">
        <section className="bg-white rounded-lg shadow p-6 space-y-4">
          <div className="flex items-center justify-between">
            <h2 className="text-lg font-semibold">Deprovisioning Checklist</h2>
            <span className="text-sm font-bold">{checklistProgress}%</span>
          </div>
          <div className="w-full bg-gray-200 rounded-full h-2">
            <div className="bg-blue-600 h-2 rounded-full" style={{ width: `${checklistProgress}%` }} />
          </div>
          <div className="space-y-2">
            {checklist.map((c, idx) => (
              <label key={idx} className="flex items-center gap-2 text-sm">
                <input type="checkbox" checked={c.done} onChange={() => toggleChecklist(idx)} className="rounded" />
                <span className={c.done ? 'line-through text-gray-400' : ''}>{c.item}</span>
              </label>
            ))}
          </div>
        </section>

        <section className="bg-white rounded-lg shadow p-6 space-y-4">
          <h2 className="text-lg font-semibold">Bulk Lifecycle Actions</h2>
          <p className="text-sm text-gray-500">Apply lifecycle changes to multiple users at once.</p>
          <div className="space-y-3">
            <select value={bulkTarget} onChange={e => setBulkTarget(e.target.value)} className="w-full border rounded px-3 py-2 text-sm">
              <option value="suspend">Suspend</option>
              <option value="activate">Activate</option>
              <option value="deprovision">Deprovision</option>
            </select>
            <textarea placeholder="Enter user emails (one per line)" rows={4} className="w-full border rounded px-3 py-2 text-sm" />
            <button className="px-4 py-2 bg-blue-600 text-white rounded text-sm">Apply to All</button>
          </div>
        </section>
      </div>
    </div>
  );
}