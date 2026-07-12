"use client";

import { useState, useEffect, useCallback } from "react";
import { CheckSquare, Square, ArrowRight, MonitorSmartphone } from "lucide-react";

interface ChecklistStep {
  key: string;
  label: string;
  description: string;
  completed: boolean;
  completed_at: string | null;
}

interface Checklist {
  client_id: string;
  client_name: string;
  steps: ChecklistStep[];
  completion_pct: number;
}

interface Client {
  client_id: string;
  client_name: string;
}

const stepIcons: Record<string, string> = {
  redirect_uris: "🔗",
  scopes: "🛡",
  branding: "🎨",
  consent_tested: "✓",
  secret_stored: "🔑",
  admin_approved: "✅",
};

export default function OnboardingChecklistPage() {
  const [clients, setClients] = useState<Client[]>([]);
  const [selectedId, setSelectedId] = useState("");
  const [checklist, setChecklist] = useState<Checklist | null>(null);
  const [loading, setLoading] = useState(false);
  const [togglingStep, setTogglingStep] = useState<string | null>(null);

  const fetchClients = useCallback(async () => {
    try {
      const res = await fetch("/api/v1/oauth/clients", { headers: { "X-Tenant-ID": "00000000-0000-0000-0000-000000000001" } });
      if (res.ok) {
        const data = await res.json();
        setClients(data.clients || data || []);
      }
    } catch { /* noop */ }
  }, []);

  const fetchChecklist = useCallback(async () => {
    if (!selectedId) return;
    setLoading(true);
    try {
      const res = await fetch(`/api/v1/oauth/clients/${selectedId}/onboarding`, { headers: { "X-Tenant-ID": "00000000-0000-0000-0000-000000000001" } });
      if (res.ok) setChecklist(await res.json());
    } catch { /* noop */ }
    finally { setLoading(false); }
  }, [selectedId]);

  useEffect(() => { fetchClients(); }, [fetchClients]);
  useEffect(() => { if (selectedId) fetchChecklist(); }, [selectedId, fetchChecklist]);

  const toggleStep = async (stepKey: string) => {
    if (!checklist) return;
    setTogglingStep(stepKey);
    try {
      const step = checklist.steps.find((s) => s.key === stepKey);
      await fetch(`/api/v1/oauth/clients/${selectedId}/onboarding/${stepKey}`, {
        method: "POST",
        headers: { "Content-Type": "application/json", "X-Tenant-ID": "00000000-0000-0000-0000-000000000001" },
        body: JSON.stringify({ completed: !step?.completed }),
      });
      setChecklist((prev) => prev ? {
        ...prev,
        steps: prev.steps.map((s) => s.key === stepKey ? { ...s, completed: !s.completed, completed_at: !s.completed ? new Date().toISOString() : null } : s),
        completion_pct: Math.round((prev.steps.filter((s) => s.key === stepKey ? !s.completed : s.completed).length / prev.steps.length) * 100),
      } : null);
    } catch { /* noop */ }
    finally { setTogglingStep(null); }
  };

  return (
    <div className="space-y-6">
      <div>
        <h1 className="text-2xl font-bold flex items-center gap-2"><CheckSquare className="w-6 h-6 text-blue-500" /> Onboarding Checklist</h1>
        <p className="text-sm text-gray-500 mt-1">Track OAuth client onboarding progress with a 6-step checklist.</p>
      </div>

      <select value={selectedId} onChange={(e) => setSelectedId(e.target.value)} className="px-3 py-2 rounded-lg border dark:border-gray-700 dark:bg-gray-900 text-sm">
        <option value="">Select a client...</option>
        {clients.map((c) => <option key={c.client_id} value={c.client_id}>{c.client_name}</option>)}
      </select>

      {checklist && (
        <>
          {/* Progress bar */}
          <div className="rounded-lg border dark:border-gray-800 p-4">
            <div className="flex items-center justify-between mb-2">
              <div className="flex items-center gap-2">
                <MonitorSmartphone className="w-5 h-5 text-blue-500" />
                <span className="font-semibold">{checklist.client_name}</span>
              </div>
              <span className={`text-2xl font-bold ${checklist.completion_pct === 100 ? "text-green-600" : checklist.completion_pct >= 50 ? "text-yellow-600" : "text-gray-500"}`}>{checklist.completion_pct}%</span>
            </div>
            <div className="w-full h-3 rounded-full bg-gray-200 dark:bg-gray-800 overflow-hidden">
              <div className={`h-full rounded-full transition-all ${checklist.completion_pct === 100 ? "bg-green-500" : "bg-blue-500"}`} style={{ width: `${checklist.completion_pct}%` }} />
            </div>
            <p className="text-xs text-gray-400 mt-2">{checklist.steps.filter((s) => s.completed).length} of {checklist.steps.length} steps completed</p>
          </div>

          {/* Steps */}
          <div className="space-y-2">
            {checklist.steps.map((step, i) => (
              <div key={step.key} className="rounded-lg border dark:border-gray-800 p-4">
                <div className="flex items-center justify-between">
                    <div className="flex items-center gap-3">
                      <button onClick={() => toggleStep(step.key)} disabled={togglingStep === step.key} className="flex-shrink-0">
                        {step.completed ? <CheckSquare className="w-5 h-5 text-green-500" /> : <Square className="w-5 h-5 text-gray-300" />}
                      </button>
                      <div className="text-lg">{stepIcons[step.key] || "📋"}</div>
                      <div>
                        <div className="flex items-center gap-2">
                          <span className={`font-medium text-sm ${step.completed ? "text-gray-400 line-through" : ""}`}>Step {i + 1}: {step.label}</span>
                          {step.completed && <span className="px-2 py-0.5 rounded text-xs bg-green-100 dark:bg-green-900/30 dark:text-green-400">Done</span>}
                        </div>
                        <p className="text-xs text-gray-500 mt-0.5">{step.description}</p>
                        {step.completed_at && <p className="text-xs text-gray-400 mt-0.5">Completed: {step.completed_at}</p>}
                      </div>
                    </div>
                    {!step.completed && i > 0 && checklist.steps[i - 1].completed && (
                      <span className="text-xs text-blue-600 flex items-center gap-1"><ArrowRight className="w-3 h-3" /> Next</span>
                    )}
                  </div>
                </div>
              ))}
          </div>

          {checklist.completion_pct === 100 && (
            <div className="rounded-lg border border-green-200 dark:border-green-900 bg-green-50 dark:bg-green-900/20 p-4 flex items-center gap-2">
              <CheckSquare className="w-5 h-5 text-green-500" />
              <span className="font-semibold text-green-700 dark:text-green-400">Onboarding complete! Client is ready for production use.</span>
            </div>
          )}
        </>
      )}

      {!checklist && !loading && selectedId && <p className="text-sm text-gray-500">No checklist found.</p>}
      {!selectedId && <p className="text-sm text-gray-500 text-center py-8">Select a client to view onboarding progress.</p>}
    </div>
  );
}
