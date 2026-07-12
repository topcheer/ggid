"use client";

import { useState, useCallback } from "react";
import { UserPlus, CheckSquare, Square, Rocket, X } from "lucide-react";

interface PreboardingTask {
  id: string;
  label: string;
  done: boolean;
}

interface ProvisionApp {
  id: string;
  name: string;
  auto: boolean;
}

interface JoinerData {
  employee_id: string;
  start_date: string;
  department: string;
  role_templates: string[];
  provision_apps: ProvisionApp[];
  preboarding: PreboardingTask[];
  status: "draft" | "pending" | "in_progress" | "completed";
}

const statusColors: Record<string, string> = {
  draft: "bg-gray-100 text-gray-800 dark:bg-gray-800 dark:text-gray-400",
  pending: "bg-blue-100 text-blue-800 dark:bg-blue-900/30 dark:text-blue-400",
  in_progress: "bg-yellow-100 text-yellow-800 dark:bg-yellow-900/30 dark:text-yellow-400",
  completed: "bg-green-100 text-green-800 dark:bg-green-900/30 dark:text-green-400",
};

const availableTemplates = ["engineer_standard", "admin_standard", "contractor_limited", "external_partner"];
const availableApps = ["slack", "github", "jira", "gcp", "vault"];

export default function JoinerFlowPage() {
  const [form, setForm] = useState<JoinerData>({ employee_id: "", start_date: "", department: "", role_templates: [], provision_apps: availableApps.map((a) => ({ id: a, name: a, auto: true })), preboarding: [{ id: "t1", label: "Create AD account", done: false }, { id: "t2", label: "Assign laptop", done: false }, { id: "t3", label: "Provision email", done: false }, { id: "t4", label: "Schedule orientation", done: false }], status: "draft" });
  const [submitted, setSubmitted] = useState(false);

  const toggleTemplate = (t: string) => {
    setForm((prev) => ({ ...prev, role_templates: prev.role_templates.includes(t) ? prev.role_templates.filter((x) => x !== t) : [...prev.role_templates, t] }));
  };

  const toggleApp = (id: string) => setForm((prev) => ({ ...prev, provision_apps: prev.provision_apps.map((a) => a.id === id ? { ...a, auto: !a.auto } : a) }));

  const toggleTask = (id: string) => setForm((prev) => ({ ...prev, preboarding: prev.preboarding.map((t) => t.id === id ? { ...t, done: !t.done } : t) }));

  const submit = useCallback(async () => {
    if (!form.employee_id || !form.start_date) return;
    try { await fetch("/api/v1/identity/joiner-flow", { method: "POST", headers: { "Content-Type": "application/json", "X-Tenant-ID": "00000000-0000-0000-0000-000000000001" }, body: JSON.stringify(form) }); setSubmitted(true); }
    catch { /* noop */ }
  }, [form]);

  const tasksDone = form.preboarding.filter((t) => t.done).length;

  return (
    <div className="space-y-6">
      <div>
        <h1 className="text-2xl font-bold flex items-center gap-2"><UserPlus className="w-6 h-6 text-green-500" /> Joiner Flow</h1>
        <p className="text-sm text-gray-500 mt-1">Automate employee onboarding with role templates and app provisioning.</p>
      </div>

      <div className="rounded-lg border dark:border-gray-800 p-4 space-y-4">
        <div className="grid grid-cols-1 md:grid-cols-3 gap-3">
          <div><label className="text-sm font-medium">Employee ID</label><input type="text" value={form.employee_id} onChange={(e) => setForm({ ...form, employee_id: e.target.value })} placeholder="emp-xxxx" className="w-full mt-1 px-3 py-2 rounded-lg border dark:border-gray-700 dark:bg-gray-900 text-sm font-mono" /></div>
          <div><label className="text-sm font-medium">Start Date</label><input type="date" value={form.start_date} onChange={(e) => setForm({ ...form, start_date: e.target.value })} className="w-full mt-1 px-3 py-2 rounded-lg border dark:border-gray-700 dark:bg-gray-900 text-sm" /></div>
          <div><label className="text-sm font-medium">Department</label><input type="text" value={form.department} onChange={(e) => setForm({ ...form, department: e.target.value })} placeholder="Engineering" className="w-full mt-1 px-3 py-2 rounded-lg border dark:border-gray-700 dark:bg-gray-900 text-sm" /></div>
        </div>
      </div>

      <div className="rounded-lg border dark:border-gray-800 p-4">
        <h3 className="text-sm font-semibold mb-3">Role Templates</h3>
        <div className="flex flex-wrap gap-2">{availableTemplates.map((t) => (
          <button key={t} onClick={() => toggleTemplate(t)} className={`px-3 py-1.5 rounded-lg text-xs font-mono ${form.role_templates.includes(t) ? "bg-blue-600 text-white" : "border dark:border-gray-700"}`}>{t}</button>
        ))}</div>
      </div>

      <div className="rounded-lg border dark:border-gray-800 p-4">
        <h3 className="text-sm font-semibold mb-3">Auto-Provisioned Apps</h3>
        <div className="space-y-1">{form.provision_apps.map((a) => (
          <button key={a.id} onClick={() => toggleApp(a.id)} className="flex items-center gap-2 text-sm w-full hover:bg-gray-50 dark:hover:bg-gray-900/30 px-2 py-1 rounded">
            {a.auto ? <CheckSquare className="w-4 h-4 text-green-500" /> : <Square className="w-4 h-4 text-gray-400" />}
            <span className={a.auto ? "" : "text-gray-400"}>{a.name}</span>
          </button>
        ))}</div>
      </div>

      <div className="rounded-lg border dark:border-gray-800 p-4">
        <div className="flex items-center justify-between mb-3"><h3 className="text-sm font-semibold">Preboarding Tasks</h3><span className="text-xs text-gray-400">{tasksDone}/{form.preboarding.length} done</span></div>
        <div className="space-y-1">{form.preboarding.map((t) => (
          <button key={t.id} onClick={() => toggleTask(t.id)} className="flex items-center gap-2 text-sm w-full hover:bg-gray-50 dark:hover:bg-gray-900/30 px-2 py-1 rounded">
            {t.done ? <CheckSquare className="w-4 h-4 text-green-500" /> : <Square className="w-4 h-4 text-gray-400" />}
            <span className={t.done ? "line-through text-gray-400" : ""}>{t.label}</span>
          </button>
        ))}</div>
      </div>

      <div className="flex items-center gap-3">
        <span className={`px-2 py-0.5 rounded text-xs ${statusColors[submitted ? "in_progress" : form.status]}`}>{submitted ? "in_progress" : form.status}</span>
        <button onClick={submit} disabled={!form.employee_id || !form.start_date || submitted} className="px-4 py-2 rounded-lg bg-green-600 text-white text-sm font-medium hover:bg-green-700 disabled:opacity-50 flex items-center gap-2"><Rocket className="w-4 h-4" /> {submitted ? "Started" : "Start Onboarding"}</button>
      </div>
    </div>
  );
}
