'use client';
import { useState, useEffect } from 'react';
import { useTranslations } from "@/lib/i18n";

interface WorkflowStep {
  id: string;
  name: string;
  description: string;
  enabled: boolean;
}

export default function DeprovisioningWorkflowConfigPage() {
  const t = useTranslations();

  const [steps, setSteps] = useState<WorkflowStep[]>([]);
  const [rollback, setRollback] = useState(true);
  const [dryRun, setDryRun] = useState(false);
  const [notifyUser, setNotifyUser] = useState(false);
  const [notifyManager, setNotifyManager] = useState(true);
  const [notifyAdmin, setNotifyAdmin] = useState(true);
  const [cleanupDays, setCleanupDays] = useState(90);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  const toggleStep = (id: string) => {
    setSteps(prev => prev.map(s => s.id === id ? { ...s, enabled: !s.enabled } : s));
  };

  const moveStep = (idx: number, dir: 'up' | 'down') => {
    setSteps(prev => {
      const next = [...prev];
      const target = dir === 'up' ? idx - 1 : idx + 1;
      if (target < 0 || target >= next.length) return prev;
      [next[idx], next[target]] = [next[target], next[idx]];
      return next;
    });
  };

  useEffect(() => {
    fetch('/api/v1/identity/deprovisioning/config', {
      headers: { 'Content-Type': 'application/json', 'X-Tenant-ID': '00000000-0000-0000-0000-000000000001' },
    })
      .then(r => { if (!r.ok) throw new Error(`HTTP ${r.status}`); return r.json(); })
      .then(data => {
        if (data) {
          if (data.steps) setSteps(data.steps);
          if (data.rollback !== undefined) setRollback(data.rollback);
          if (data.dry_run !== undefined) setDryRun(data.dry_run);
          if (data.notify_user !== undefined) setNotifyUser(data.notify_user);
          if (data.notify_manager !== undefined) setNotifyManager(data.notify_manager);
          if (data.notify_admin !== undefined) setNotifyAdmin(data.notify_admin);
          if (data.cleanup_days) setCleanupDays(data.cleanup_days);
        }
        setLoading(false);
      })
      .catch(e => { setError(e.message); setLoading(false); });
  }, []);

  if (loading) return <div className="p-6"><p>Loading...</p></div>;
  if (error) return <div className="p-6 text-red-600">Error: {error}</div>;

  return (
    <div className="p-6 max-w-3xl mx-auto space-y-6">
      <div>
        <h1 className="text-2xl font-bold">Deprovisioning Workflow Configuration</h1>
        <p className="text-gray-600">Configure automated deprovisioning steps, notifications, and cleanup.</p>
      </div>

      <section className="bg-white rounded-lg shadow p-6 space-y-4">
        <h2 className="text-lg font-semibold">Workflow Steps (drag to reorder)</h2>
        <div className="space-y-2">
          {steps.map((s, idx) => (
            <div key={s.id} className="flex items-center gap-3 border rounded p-3">
              <div className="flex flex-col">
                <button onClick={() => moveStep(idx, 'up')} disabled={idx === 0} className="text-xs text-gray-400 disabled:opacity-30">â</button>
                <button onClick={() => moveStep(idx, 'down')} disabled={idx === steps.length - 1} className="text-xs text-gray-400 disabled:opacity-30">â</button>
              </div>
              <span className="w-6 h-6 bg-blue-100 text-blue-700 rounded-full flex items-center justify-center text-xs font-bold">{idx + 1}</span>
              <div className="flex-1">
                <div className="text-sm font-medium">{s.name}</div>
                <div className="text-xs text-gray-500">{s.description}</div>
              </div>
              <label className="flex items-center gap-1 text-sm">
                <input type="checkbox" checked={s.enabled} onChange={() => toggleStep(s.id)} className="rounded" />
                <span className="text-xs">{s.enabled ? 'enabled' : 'disabled'}</span>
              </label>
            </div>
          ))}
        </div>
      </section>

      <div className="grid grid-cols-2 gap-4">
        <label className="flex items-center justify-between bg-white rounded-lg shadow p-4">
          <span className="text-sm font-medium">Rollback on Failure</span>
          <input type="checkbox" checked={rollback} onChange={e => setRollback(e.target.checked)} className="rounded" />
        </label>
        <label className="flex items-center justify-between bg-white rounded-lg shadow p-4">
          <span className="text-sm font-medium">Dry-Run Mode</span>
          <input type="checkbox" checked={dryRun} onChange={e => setDryRun(e.target.checked)} className="rounded" />
        </label>
      </div>

      <section className="bg-white rounded-lg shadow p-6 space-y-4">
        <h2 className="text-lg font-semibold">Notification Configuration</h2>
        <div className="space-y-2">
          <label className="flex items-center justify-between">
            <span className="text-sm">Notify User</span>
            <input type="checkbox" checked={notifyUser} onChange={e => setNotifyUser(e.target.checked)} className="rounded" />
          </label>
          <label className="flex items-center justify-between">
            <span className="text-sm">Notify Manager</span>
            <input type="checkbox" checked={notifyManager} onChange={e => setNotifyManager(e.target.checked)} className="rounded" />
          </label>
          <label className="flex items-center justify-between">
            <span className="text-sm">Notify Admin</span>
            <input type="checkbox" checked={notifyAdmin} onChange={e => setNotifyAdmin(e.target.checked)} className="rounded" />
          </label>
        </div>
      </section>

      <section className="bg-white rounded-lg shadow p-6 space-y-4">
        <h2 className="text-lg font-semibold">Scheduled Cleanup</h2>
        <div className="flex items-center gap-3">
          <label className="text-sm font-medium">Clean up archived data after:</label>
          <input type="number" min={30} max={365} value={cleanupDays} onChange={e => setCleanupDays(parseInt(e.target.value) || 90)} className="w-24 border rounded px-2 py-1 text-sm" />
          <span className="text-sm text-gray-500">days</span>
        </div>
        <p className="text-xs text-gray-400">Archived user data is permanently deleted after {cleanupDays} days.</p>
      </section>
    </div>
  );
}
