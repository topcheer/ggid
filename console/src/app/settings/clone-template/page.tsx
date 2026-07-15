"use client";

import { useState, useEffect, useCallback } from "react";
import { Copy, Save, Play, Search, Shield, Users as UsersIcon, Key, X, Check } from "lucide-react";
import { useTranslations } from "@/lib/i18n";

interface UserSummary {
  user_id: string;
  username: string;
  email: string;
}

interface TemplateData {
  roles: string[];
  groups: string[];
  permissions: string[];
  org_id: string;
  attributes: Record<string, string>;
}

interface SavedTemplate {
  id: string;
  name: string;
  description: string;
  source_user: string;
  data: TemplateData;
  created_at: string;
}

export default function CloneTemplatePage() {
  const t = useTranslations();

  const [users, setUsers] = useState<UserSummary[]>([]);
  const [search, setSearch] = useState("");
  const [selectedUserId, setSelectedUserId] = useState("");
  const [template, setTemplate] = useState<TemplateData | null>(null);
  const [loading, setLoading] = useState(false);
  const [showSave, setShowSave] = useState(false);
  const [showCreate, setShowCreate] = useState(false);
  const [templates, setTemplates] = useState<SavedTemplate[]>([]);
  const [templateName, setTemplateName] = useState("");
  const [templateDesc, setTemplateDesc] = useState("");
  const [targetUsername, setTargetUsername] = useState("");
  const [targetEmail, setTargetEmail] = useState("");
  const [selectedTemplateId, setSelectedTemplateId] = useState("");
  const [creating, setCreating] = useState(false);

  const fetchUsers = useCallback(async () => {
    try {
      const res = await fetch("/api/v1/identity/users", { headers: { "X-Tenant-ID": "00000000-0000-0000-0000-000000000001" } });
      if (res.ok) {
        const data = await res.json();
        setUsers(data.users || data || []);
      }
    } catch { /* noop */ }
  }, []);

  const fetchTemplates = useCallback(async () => {
    try {
      const res = await fetch("/api/v1/identity/clone-templates", { headers: { "X-Tenant-ID": "00000000-0000-0000-0000-000000000001" } });
      if (res.ok) {
        const data = await res.json();
        setTemplates(data.templates || data || []);
      }
    } catch { /* noop */ }
  }, []);

  useEffect(() => { fetchUsers(); fetchTemplates(); }, [fetchUsers, fetchTemplates]);

  const previewTemplate = useCallback(async () => {
    if (!selectedUserId) return;
    setLoading(true);
    try {
      const res = await fetch(`/api/v1/identity/users/${selectedUserId}/template`, { headers: { "X-Tenant-ID": "00000000-0000-0000-0000-000000000001" } });
      if (res.ok) {
        setTemplate(await res.json());
      }
    } catch { /* noop */ }
    finally { setLoading(false); }
  }, [selectedUserId]);

  const saveTemplate = async () => {
    if (!template || !templateName) return;
    try {
      await fetch("/api/v1/identity/clone-templates", {
        method: "POST",
        headers: { "Content-Type": "application/json", "X-Tenant-ID": "00000000-0000-0000-0000-000000000001" },
        body: JSON.stringify({ name: templateName, description: templateDesc, source_user: selectedUserId, data: template }),
      });
      setShowSave(false);
      setTemplateName("");
      setTemplateDesc("");
      fetchTemplates();
    } catch { /* noop */ }
  };

  const createFromTemplate = async () => {
    if (!selectedTemplateId || !targetUsername) return;
    setCreating(true);
    try {
      await fetch("/api/v1/identity/clone-templates/apply", {
        method: "POST",
        headers: { "Content-Type": "application/json", "X-Tenant-ID": "00000000-0000-0000-0000-000000000001" },
        body: JSON.stringify({ template_id: selectedTemplateId, username: targetUsername, email: targetEmail }),
      });
      setShowCreate(false);
      setSelectedTemplateId("");
      setTargetUsername("");
      setTargetEmail("");
    } catch { /* noop */ }
    finally { setCreating(false); }
  };

  const filteredUsers = users.filter((u) => !search || u.username.toLowerCase().includes(search.toLowerCase()));

  return (
    <div className="space-y-6">
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-2xl font-bold flex items-center gap-2"><Copy className="w-6 h-6 text-blue-500" /> {t("cloneTemplate.title")}</h1>
          <p className="text-sm text-gray-500 mt-1">Clone user roles, groups, and permissions as reusable templates.</p>
        </div>
        <div className="flex gap-2">
          <button onClick={() => setShowSave(true)} disabled={!template} className="px-4 py-2 rounded-lg bg-blue-600 text-white text-sm font-medium hover:bg-blue-700 disabled:opacity-50 flex items-center gap-2"><Save className="w-4 h-4" /> Save Template</button>
          <button onClick={() => setShowCreate(true)} disabled={templates.length === 0} className="px-4 py-2 rounded-lg bg-green-600 text-white text-sm font-medium hover:bg-green-700 disabled:opacity-50 flex items-center gap-2"><Play className="w-4 h-4" /> Create from Template</button>
        </div>
      </div>

      <div className="grid grid-cols-1 lg:grid-cols-3 gap-4">
        {/* User selector */}
        <div className="space-y-3">
          <div className="relative">
            <Search className="absolute left-3 top-1/2 -translate-y-1/2 w-4 h-4 text-gray-400" />
            <input type="text" placeholder="Search source user..." value={search} onChange={(e) => setSearch(e.target.value)} className="w-full pl-9 pr-3 py-2 rounded-lg border dark:border-gray-700 dark:bg-gray-900 text-sm" />
          </div>
          <div className="rounded-lg border dark:border-gray-800 max-h-80 overflow-y-auto">
            <div className="divide-y dark:divide-gray-800">
              {filteredUsers.slice(0, 30).map((u) => (
                <button key={u.user_id} onClick={() => { setSelectedUserId(u.user_id); }} className={`w-full text-left px-3 py-2 hover:bg-gray-50 dark:hover:bg-gray-900/30 ${selectedUserId === u.user_id ? "bg-blue-50 dark:bg-blue-900/20" : ""}`}>
                  <div className="text-sm font-medium">{u.username}</div>
                  <div className="text-xs text-gray-400">{u.email}</div>
                </button>
              ))}
            </div>
          </div>
          <button onClick={previewTemplate} disabled={!selectedUserId || loading} className="w-full px-4 py-2 rounded-lg bg-blue-600 text-white text-sm font-medium hover:bg-blue-700 disabled:opacity-50">{loading ? "Loading..." : "Preview Template"}</button>
        </div>

        {/* Template preview */}
        <div className="lg:col-span-2">
          {template ? (
            <div className="space-y-4">
              <div className="rounded-lg border dark:border-gray-800 p-4">
                <h3 className="font-semibold mb-3 flex items-center gap-2"><Shield className="w-4 h-4" /> Roles ({template.roles.length})</h3>
                <div className="flex flex-wrap gap-1">
                  {template.roles.map((r, i) => <span key={i} className="px-2 py-0.5 rounded text-xs bg-blue-100 text-blue-800 dark:bg-blue-900/30 dark:text-blue-400 font-mono">{r}</span>)}
                </div>
              </div>
              <div className="rounded-lg border dark:border-gray-800 p-4">
                <h3 className="font-semibold mb-3 flex items-center gap-2"><UsersIcon className="w-4 h-4" /> Groups ({template.groups.length})</h3>
                <div className="flex flex-wrap gap-1">
                  {template.groups.map((g, i) => <span key={i} className="px-2 py-0.5 rounded text-xs bg-purple-100 text-purple-800 dark:bg-purple-900/30 dark:text-purple-400 font-mono">{g}</span>)}
                </div>
              </div>
              <div className="rounded-lg border dark:border-gray-800 p-4">
                <h3 className="font-semibold mb-3 flex items-center gap-2"><Key className="w-4 h-4" /> Permissions ({template.permissions.length})</h3>
                <div className="space-y-1 max-h-40 overflow-y-auto">
                  {template.permissions.map((p, i) => <div key={i} className="text-xs font-mono text-gray-500">{p}</div>)}
                </div>
              </div>
              {Object.keys(template.attributes).length > 0 && (
                <div className="rounded-lg border dark:border-gray-800 p-4">
                  <h3 className="font-semibold mb-3">Attributes</h3>
                  <div className="grid grid-cols-2 gap-2">
                    {Object.entries(template.attributes).map(([k, v]) => (
                      <div key={k} className="text-xs"><span className="text-gray-400">{k}:</span> <span className="font-mono">{v}</span></div>
                    ))}
                  </div>
                </div>
              )}
            </div>
          ) : (
            <p className="text-sm text-gray-500 text-center py-8">Select a source user and click Preview Template.</p>
          )}
        </div>
      </div>

      {/* Saved templates list */}
      <div className="rounded-lg border dark:border-gray-800">
        <div className="px-4 py-3 border-b dark:border-gray-800"><h3 className="font-semibold">Saved Templates ({templates.length})</h3></div>
        <div className="divide-y dark:divide-gray-800">
          {templates.map((t) => (
            <div key={t.id} className="px-4 py-3 flex items-center justify-between">
              <div>
                <span className="font-medium text-sm">{t.name}</span>
                <span className="text-xs text-gray-400 ml-2">from {t.source_user} · {t.created_at}</span>
              </div>
              <div className="flex items-center gap-2 text-xs text-gray-400">
                <span>{t.data.roles.length} roles</span>
                <span>{t.data.permissions.length} perms</span>
              </div>
            </div>
          ))}
          {templates.length === 0 && <p className="px-4 py-3 text-sm text-gray-500">No saved templates.</p>}
        </div>
      </div>

      {/* Save template modal */}
      {showSave && (
        <div className="fixed inset-0 bg-black/50 flex items-center justify-center z-50" onClick={() => setShowSave(false)}>
          <div className="bg-white dark:bg-gray-900 rounded-lg shadow-xl max-w-md w-full mx-4" onClick={(e) => e.stopPropagation()}>
            <div className="flex items-center justify-between px-6 py-4 border-b dark:border-gray-800">
              <h3 className="font-semibold">Save Template</h3>
              <button onClick={() => setShowSave(false)}><X className="w-5 h-5 text-gray-400" /></button>
            </div>
            <div className="px-6 py-4 space-y-3">
              <div><label className="text-sm font-medium">Template Name</label><input type="text" value={templateName} onChange={(e) => setTemplateName(e.target.value)} placeholder="Standard Developer" className="w-full mt-1 px-3 py-2 rounded-lg border dark:border-gray-700 dark:bg-gray-800 text-sm" /></div>
              <div><label className="text-sm font-medium">Description</label><input type="text" value={templateDesc} onChange={(e) => setTemplateDesc(e.target.value)} placeholder="Base developer access" className="w-full mt-1 px-3 py-2 rounded-lg border dark:border-gray-700 dark:bg-gray-800 text-sm" /></div>
            </div>
            <div className="flex justify-end gap-2 px-6 py-4 border-t dark:border-gray-800">
              <button onClick={() => setShowSave(false)} className="px-4 py-2 rounded-lg border dark:border-gray-700 text-sm">Cancel</button>
              <button onClick={saveTemplate} disabled={!templateName} className="px-4 py-2 rounded-lg bg-blue-600 text-white text-sm font-medium hover:bg-blue-700 disabled:opacity-50">Save</button>
            </div>
          </div>
        </div>
      )}

      {/* Create from template modal */}
      {showCreate && (
        <div className="fixed inset-0 bg-black/50 flex items-center justify-center z-50" onClick={() => setShowCreate(false)}>
          <div className="bg-white dark:bg-gray-900 rounded-lg shadow-xl max-w-md w-full mx-4" onClick={(e) => e.stopPropagation()}>
            <div className="flex items-center justify-between px-6 py-4 border-b dark:border-gray-800">
              <h3 className="font-semibold">Create User from Template</h3>
              <button onClick={() => setShowCreate(false)}><X className="w-5 h-5 text-gray-400" /></button>
            </div>
            <div className="px-6 py-4 space-y-3">
              <div><label className="text-sm font-medium">Template</label>
                <select value={selectedTemplateId} onChange={(e) => setSelectedTemplateId(e.target.value)} className="w-full mt-1 px-3 py-2 rounded-lg border dark:border-gray-700 dark:bg-gray-800 text-sm">
                  <option value="">Select...</option>
                  {templates.map((t) => <option key={t.id} value={t.id}>{t.name}</option>)}
                </select>
              </div>
              <div><label className="text-sm font-medium">New Username</label><input type="text" value={targetUsername} onChange={(e) => setTargetUsername(e.target.value)} className="w-full mt-1 px-3 py-2 rounded-lg border dark:border-gray-700 dark:bg-gray-800 text-sm" /></div>
              <div><label className="text-sm font-medium">Email</label><input type="text" value={targetEmail} onChange={(e) => setTargetEmail(e.target.value)} className="w-full mt-1 px-3 py-2 rounded-lg border dark:border-gray-700 dark:bg-gray-800 text-sm" /></div>
            </div>
            <div className="flex justify-end gap-2 px-6 py-4 border-t dark:border-gray-800">
              <button onClick={() => setShowCreate(false)} className="px-4 py-2 rounded-lg border dark:border-gray-700 text-sm">Cancel</button>
              <button onClick={createFromTemplate} disabled={!selectedTemplateId || !targetUsername || creating} className="px-4 py-2 rounded-lg bg-green-600 text-white text-sm font-medium hover:bg-green-700 disabled:opacity-50">{creating ? "Creating..." : "Create User"}</button>
            </div>
          </div>
        </div>
      )}
    </div>
  );
}
