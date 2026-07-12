'use client';
import { useState } from 'react';

interface WorkflowStep {
  id: string;
  name: string;
  description: string;
  enabled: boolean;
}

export default function DeprovisioningWorkflowConfigPage() {
  const [steps, setSteps] = useState<WorkflowStep[]>([
    { id: 's1', name: 'Revoke Tokens', description: 'Revoke all active OAuth/JWT tokens', enabled: true },
    { id: 's2', name: 'Remove Groups', description: 'Remove user from all groups and roles', enabled: true },
    { id: 's3', name: 'Disable Account', description: 'Set account status to disabled', enabled: true },
    { id: 's4', name: 'Archive Data', description: 'Archive user data to cold storage', enabled: true },
    { id: 's5', name: 'Audit Trail', description: 'Generate final audit trail entry', enabled: true },
  ]);

  const [rollback, setRollback] = useState(true);
  const [dryRun, setDryRun] = useState(false);
  const [notifyUser, setNotifyUser] = useState(false);
  const [notifyManager, setNotifyManager] = useState(true);
  const [notifyAdmin, setNotifyAdmin] = useState(true);
  const [cleanupDays, setCleanupDays] = useState(90);

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
