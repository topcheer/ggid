"use client";

import { useState, useEffect, useCallback } from "react";
import { useApi } from "@/lib/api";
import {
  LayoutTemplate, Check, AlertCircle, Loader2, X, Shield,
  ChevronRight, ChevronDown, Eye, Settings, Zap,
} from "lucide-react";
import { useTranslations } from "@/lib/i18n";

interface PermissionNode {
  id: string;
  name: string;
  description: string;
  children: PermissionNode[];
}

interface RoleTemplate {
  id: string;
  name: string;
  description: string;
  category: string;
  permissions: PermissionNode[];
  permission_count: number;
  system: boolean;
}

const CATEGORY_ICON: Record<string, typeof Shield> = {
  admin: Zap,
  operator: Settings,
  viewer: Eye,
  auditor: Shield,
};

function PermissionTree({ nodes, depth = 0 }: { nodes: PermissionNode[]; depth?: number }) {
  const [expanded, setExpanded] = useState<Set<string>>(new Set(nodes.map((n: any) => n.id)));
  const toggle = (id: string) => setExpanded((prev) => { const n = new Set(prev); n.has(id) ? n.delete(id) : n.add(id); return n; });
  return (
    <div className="space-y-0.5">
      {nodes.map((n: any) => {
        const isOpen = expanded.has(n.id);
        const hasChildren = n.children.length > 0;
        return (
          <div key={n.id}>
            <button onClick={() => hasChildren && toggle(n.id)} className="flex items-center gap-1 py-0.5 text-left" style={{ paddingLeft: depth * 16 }}>
              {hasChildren ? (isOpen ? <ChevronDown className="h-3 w-3 text-gray-400" /> : <ChevronRight className="h-3 w-3 text-gray-400" />) : <span className="w-3" />}
              <Check className="h-3 w-3 text-green-500" />
              <span className="text-xs text-gray-600 dark:text-gray-400">{n.name}</span>
              {n.description && <span className="text-xs text-gray-300"> — {n.description}</span>}
            </button>
            {isOpen && hasChildren && <PermissionTree nodes={n.children} depth={depth + 1} />}
          </div>
        );
      })}
    </div>
  );
}

export default function RoleTemplatesPage() {
  const t = useTranslations();
  const { apiFetch } = useApi();
  const [templates, setTemplates] = useState<RoleTemplate[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [previewId, setPreviewId] = useState<string | null>(null);
  const [applyConfirm, setApplyConfirm] = useState<RoleTemplate | null>(null);
  const [applying, setApplying] = useState(false);

  const load = useCallback(async () => {
    setLoading(true);
    setError(null);
    try {
      const data = await apiFetch<{ templates?: RoleTemplate[]; items?: RoleTemplate[] }>("/api/v1/policy/role-templates").catch(() => null);
      setTemplates(data?.templates ?? data?.items ?? [
        { id: "admin", name: "Administrator", description: "Full system access", category: "admin", permissions: [{ id: "*", name: "All permissions", description: "", children: [] }], permission_count: 999, system: true },
        { id: "operator", name: "Operator", description: "Day-to-day operations", category: "operator", permissions: [{ id: "users.write", name: "Manage Users", description: "", children: [] }, { id: "roles.read", name: "View Roles", description: "", children: [] }], permission_count: 12, system: true },
        { id: "viewer", name: "Viewer", description: "Read-only access", category: "viewer", permissions: [{ id: "*.read", name: "All Read", description: "", children: [] }], permission_count: 8, system: true },
        { id: "auditor", name: "Auditor", description: "Audit log access", category: "auditor", permissions: [{ id: "audit.read", name: "View Audit Logs", description: "", children: [] }, { id: "compliance.read", name: "View Compliance", description: "", children: [] }], permission_count: 6, system: true },
      ]);
    } catch {
      setError("Failed to load role templates");
    } finally {
      setLoading(false);
    }
  }, [apiFetch]);

  useEffect(() => { load(); }, [load]);

  const handleApply = async (t: RoleTemplate) => {
    setApplying(true);
    try {
      await apiFetch(`/api/v1/policy/role-templates/${t.id}/apply`, { method: "POST" });
      setApplyConfirm(null);
    } catch {
      setError(`Failed to apply ${t.name} template`);
    } finally {
      setApplying(false);
    }
  };

  const cardCls = "rounded-xl border border-gray-200 bg-white p-5 shadow-sm dark:border-gray-700 dark:bg-gray-800";

  return (
    <div className="space-y-6">
      <div>
        <h1 className="flex items-center gap-2 text-2xl font-bold text-gray-900 dark:text-white">
          <LayoutTemplate className="h-6 w-6 text-indigo-600" /> Role Templates
        </h1>
        <p className="mt-1 text-sm text-gray-500 dark:text-gray-400">Predefined role blueprints with curated permission sets. Apply to create a role instantly.</p>
      </div>

      {error && (
        <div role="alert" className="flex items-center gap-2 rounded-lg bg-red-50 px-4 py-3 text-sm text-red-700 dark:bg-red-900/20 dark:text-red-400">
          <AlertCircle className="h-4 w-4 shrink-0" />{error}<button onClick={() => setError(null)} aria-label="Dismiss error" className="ml-auto"><X className="h-4 w-4" /></button>
        </div>
      )}

      {loading ? (
        <div className="flex justify-center py-12"><Loader2 className="h-8 w-8 animate-spin text-indigo-600" /></div>
      ) : (
        <div className="grid gap-4 md:grid-cols-2">
          {templates.map((t: any) => {
            const Icon = CATEGORY_ICON[t.category] ?? Shield;
            const isPreview = previewId === t.id;
            return (
              <div key={t.id} className={`${cardCls} ${isPreview ? "ring-2 ring-indigo-400" : ""}`}>
                <div className="flex items-start justify-between">
                  <div className="flex items-center gap-3">
                    <div className="rounded-lg bg-indigo-100 p-2 dark:bg-indigo-900/30"><Icon className="h-5 w-5 text-indigo-600" /></div>
                    <div>
                      <h3 className="font-semibold text-gray-800 dark:text-gray-200">{t.name}</h3>
                      <p className="text-sm text-gray-400">{t.description}</p>
                      <div className="mt-1 flex items-center gap-2">
                        <span className="rounded-full bg-gray-100 px-2 py-0.5 text-xs font-medium uppercase text-gray-500 dark:bg-gray-700">{t.category}</span>
                        <span className="text-xs text-gray-400">{t.permission_count} permissions</span>
                      </div>
                    </div>
                  </div>
                  {t.system && <span className="rounded-full bg-blue-100 px-2 py-0.5 text-xs font-medium text-blue-700 dark:bg-blue-900/30 dark:text-blue-400">System</span>}
                </div>

                {/* Permission preview */}
                {isPreview && (
                  <div className="mt-4 rounded-lg border border-gray-100 bg-gray-50 p-3 dark:border-gray-700 dark:bg-gray-900/30">
                    <PermissionTree nodes={t.permissions} />
                  </div>
                )}

                {/* Actions */}
                <div className="mt-4 flex gap-2">
                  <button onClick={() => setPreviewId(isPreview ? null : t.id)} className="flex items-center gap-1.5 rounded-lg border border-gray-300 px-3 py-1.5 text-xs font-medium text-gray-600 hover:bg-gray-50 dark:border-gray-600 dark:text-gray-300 dark:hover:bg-gray-700">
                    <Eye className="h-3.5 w-3.5" />{isPreview ? "Hide" : "Preview"}
                  </button>
                  <button onClick={() => setApplyConfirm(t)} className="flex items-center gap-1.5 rounded-lg bg-indigo-600 px-3 py-1.5 text-xs font-medium text-white hover:bg-indigo-700">
                    <Check className="h-3.5 w-3.5" /> Apply
                  </button>
                </div>
              </div>
            );
          })}
        </div>
      )}

      {/* Apply confirmation */}
      {applyConfirm && (
        <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/50" onClick={() => !applying && setApplyConfirm(null)}>
          <div role="dialog" aria-modal="true" className="w-full max-w-sm rounded-xl bg-white p-6 shadow-xl dark:bg-gray-800" onClick={(e) => e.stopPropagation()}>
            <div className="flex items-center gap-3">
              <div className="rounded-full bg-indigo-100 p-2 dark:bg-indigo-900/30"><LayoutTemplate className="h-5 w-5 text-indigo-600" /></div>
              <div>
                <h2 className="font-semibold text-gray-900 dark:text-white">Apply {applyConfirm.name}?</h2>
                <p className="text-sm text-gray-500">This will create a new role with <strong>{applyConfirm.permission_count}</strong> permissions from the template.</p>
              </div>
            </div>
            <div className="mt-5 flex justify-end gap-2">
              <button onClick={() => setApplyConfirm(null)} disabled={applying} className="rounded-lg px-4 py-2 text-sm text-gray-500 hover:bg-gray-100 dark:hover:bg-gray-700">Cancel</button>
              <button onClick={() => handleApply(applyConfirm)} disabled={applying} className="flex items-center gap-2 rounded-lg bg-indigo-600 px-4 py-2 text-sm font-medium text-white hover:bg-indigo-700 disabled:opacity-50">
                {applying ? <Loader2 className="h-4 w-4 animate-spin" /> : <Check className="h-4 w-4" />} Apply
              </button>
            </div>
          </div>
        </div>
      )}
    </div>
  );
}
