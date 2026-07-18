"use client";

import { useState, useEffect, useCallback } from "react";
import { useApi } from "@/lib/api";
import {
  Building2, Plus, Trash2, X, AlertCircle, Loader2, Check, Pencil,
  ChevronRight, ChevronDown, Users, DollarSign, Hash,
} from "lucide-react";
import { useTranslations } from "@/lib/i18n";

interface Department {
  id: string;
  name: string;
  parent_id: string | null;
  budget: number;
  headcount: number;
  cost_center: string;
  children: Department[];
}

export default function DepartmentsPage() {
  const t = useTranslations();

  const { apiFetch } = useApi();
  const [departments, setDepartments] = useState<Department[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [expanded, setExpanded] = useState<Set<string>>(new Set());
  const [showAdd, setShowAdd] = useState(false);
  const [editDept, setEditDept] = useState<Department | null>(null);
  const [confirmDelete, setConfirmDelete] = useState<Department | null>(null);
  const [form, setForm] = useState({ name: "", parent_id: "", budget: 0, headcount: 0, cost_center: "" });

  const load = useCallback(async () => {
    setLoading(true);
    setError(null);
    try {
      const data = await apiFetch<{ departments?: Department[]; items?: Department[] }>("/api/v1/orgs/departments").catch(() => null);
      const depts = data?.departments ?? data?.items ?? [];
      setDepartments(depts);
      setExpanded(new Set(depts.filter((d: any) => d.children?.length > 0).map((d: any) => d.id)));
    } catch {
      setError("Failed to load departments");
    } finally {
      setLoading(false);
    }
  }, [apiFetch]);

  useEffect(() => { load(); }, [load]);

  const toggle = (id: string) => setExpanded((prev) => { const n = new Set(prev); n.has(id) ? n.delete(id) : n.add(id); return n; });

  const flatten = (depts: Department[]): Department[] => {
    const result: Department[] = [];
    for (const d of depts) { result.push(d); if (d.children?.length) result.push(...flatten(d.children)); }
    return result;
  };

  const handleSave = async () => {
    try {
      if (editDept) {
        await apiFetch(`/api/v1/orgs/departments/${editDept.id}`, { method: "PATCH", body: JSON.stringify(form) });
      } else {
        await apiFetch("/api/v1/orgs/departments", { method: "POST", body: JSON.stringify({ ...form, parent_id: form.parent_id || null }) });
      }
      setShowAdd(false);
      setEditDept(null);
      setForm({ name: "", parent_id: "", budget: 0, headcount: 0, cost_center: "" });
      await load();
    } catch {
      setError("Failed to save department");
    }
  };

  const handleDelete = async (id: string) => {
    try {
      await apiFetch(`/api/v1/orgs/departments/${id}`, { method: "DELETE" });
      setConfirmDelete(null);
      await load();
    } catch {
      setError("Failed to delete department");
    }
  };

  const startEdit = (d: Department) => {
    setEditDept(d);
    setForm({ name: d.name, parent_id: d.parent_id ?? "", budget: d.budget, headcount: d.headcount, cost_center: d.cost_center });
    setShowAdd(true);
  };

  const startAdd = (parentId?: string) => {
    setEditDept(null);
    setForm({ name: "", parent_id: parentId ?? "", budget: 0, headcount: 0, cost_center: "" });
    setShowAdd(true);
  };

  const cardCls = "rounded-xl border border-gray-200 bg-white p-5 shadow-sm dark:border-gray-700 dark:bg-gray-800";
  const allDepts = flatten(departments);

  const renderDept = (dept: Department, depth: number): React.ReactNode => {
    const isOpen = expanded.has(dept.id);
    const hasChildren = dept.children?.length > 0;
    return (
      <div key={dept.id}>
        <div className={`flex items-center gap-3 rounded-lg py-2.5 hover:bg-gray-50 dark:hover:bg-gray-800/50`} style={{ paddingLeft: depth * 24 + 12 }}>
          <button onClick={() => hasChildren && toggle(dept.id)} aria-label="Toggle department children" className="shrink-0">
            {hasChildren ? (isOpen ? <ChevronDown className="h-4 w-4 text-gray-400" /> : <ChevronRight className="h-4 w-4 text-gray-400" />) : <span className="inline-block w-4" />}
          </button>
          <Building2 className="h-4 w-4 shrink-0 text-indigo-500" />
          <span className="flex-1 font-medium text-gray-800 dark:text-gray-200">{dept.name}</span>
          <div className="hidden items-center gap-4 text-xs text-gray-400 sm:flex">
            <span className="flex items-center gap-1"><Users className="h-3 w-3" />{dept.headcount}</span>
            <span className="flex items-center gap-1"><DollarSign className="h-3 w-3" />{(dept.budget / 1000).toFixed(0)}K</span>
            {dept.cost_center && <span className="flex items-center gap-1"><Hash className="h-3 w-3" />{dept.cost_center}</span>}
          </div>
          <div className="flex items-center gap-1">
            <button onClick={() => startAdd(dept.id)} aria-label={"Add sub-department to " + dept.name} className="rounded p-1 text-gray-400 hover:bg-gray-100 dark:hover:bg-gray-700" title="Add sub-department"><Plus className="h-3.5 w-3.5" /></button>
            <button onClick={() => startEdit(dept)} aria-label={"Edit " + dept.name} className="rounded p-1 text-gray-400 hover:bg-gray-100 dark:hover:bg-gray-700"><Pencil className="h-3.5 w-3.5" /></button>
            <button onClick={() => setConfirmDelete(dept)} aria-label={"Delete " + dept.name} className="rounded p-1 text-red-400 hover:bg-red-50 dark:hover:bg-red-900/20"><Trash2 className="h-3.5 w-3.5" /></button>
          </div>
        </div>
        {/* Mobile details */}
        <div className="flex items-center gap-4 pb-1 text-xs text-gray-400 sm:hidden" style={{ paddingLeft: depth * 24 + 48 }}>
          <span className="flex items-center gap-1"><Users className="h-3 w-3" />{dept.headcount}</span>
          <span className="flex items-center gap-1"><DollarSign className="h-3 w-3" />{(dept.budget / 1000).toFixed(0)}K</span>
          {dept.cost_center && <span className="flex items-center gap-1"><Hash className="h-3 w-3" />{dept.cost_center}</span>}
        </div>
        {isOpen && hasChildren && dept.children.map((child: any) => renderDept(child, depth + 1))}
      </div>
    );
  };

  return (
    <div className="space-y-6">
      <div className="flex items-center justify-between">
        <div>
          <h1 className="flex items-center gap-2 text-2xl font-bold text-gray-900 dark:text-white">
            <Building2 className="h-6 w-6 text-indigo-600" /> {t("departments.title")}
          </h1>
          <p className="mt-1 text-sm text-gray-500 dark:text-gray-400">{t("departments.subtitle")}</p>
        </div>
        <button onClick={() => startAdd()} className="flex items-center gap-2 rounded-lg bg-indigo-600 px-4 py-2 text-sm font-medium text-white hover:bg-indigo-700"><Plus className="h-4 w-4" /> {t("departments.addDepartment")}</button>
      </div>

      {error && (
        <div role="alert" className="flex items-center gap-2 rounded-lg bg-red-50 px-4 py-3 text-sm text-red-700 dark:bg-red-900/20 dark:text-red-400">
          <AlertCircle className="h-4 w-4 shrink-0" />{error}<button onClick={() => setError(null)} aria-label="Dismiss error" className="ml-auto"><X className="h-4 w-4" /></button>
        </div>
      )}

      {loading ? (
        <div className="flex justify-center py-12"><Loader2 className="h-8 w-8 animate-spin text-indigo-600" /></div>
      ) : departments.length === 0 ? (
        <div className={cardCls}><div className="py-12 text-center"><Building2 className="mx-auto h-12 w-12 text-gray-300" /><p className="mt-4 text-sm text-gray-400">{t("departments.noDepartments")}</p></div></div>
      ) : (
        <div className={cardCls}>
          {/* Summary row */}
          <div className="mb-3 grid grid-cols-3 gap-4 border-b border-gray-100 pb-3 dark:border-gray-700">
            <div><p className="text-xs font-semibold uppercase text-gray-400">Total Departments</p><p className="mt-1 text-xl font-bold text-indigo-600">{allDepts.length}</p></div>
            <div><p className="text-xs font-semibold uppercase text-gray-400">Total Headcount</p><p className="mt-1 text-xl font-bold text-indigo-600">{allDepts.reduce((sum: any, d: any) => sum + d.headcount, 0)}</p></div>
            <div><p className="text-xs font-semibold uppercase text-gray-400">Total Budget</p><p className="mt-1 text-xl font-bold text-indigo-600">${(allDepts.reduce((sum: any, d: any) => sum + d.budget, 0) / 1000000).toFixed(1)}M</p></div>
          </div>
          {departments.map((d: any) => renderDept(d, 0))}
        </div>
      )}

      {/* Add/Edit modal */}
      {showAdd && (
        <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/50" onClick={() => setShowAdd(false)}>
          <div role="dialog" aria-modal="true" className="w-full max-w-md rounded-xl bg-white p-6 shadow-xl dark:bg-gray-800" onClick={(e) => e.stopPropagation()}>
            <div className="flex items-center justify-between">
              <h2 className="text-lg font-semibold text-gray-900 dark:text-white">{editDept ? t("departments.editDepartment") : t("departments.addDepartment")}</h2>
              <button onClick={() => setShowAdd(false)} aria-label="Close"><X className="h-5 w-5 text-gray-400" /></button>
            </div>
            <div className="mt-4 space-y-3">
              <div><label className="text-sm font-medium text-gray-700 dark:text-gray-300">{t("departments.departmentName")}</label><input aria-label="Engineering" value={form.name} onChange={(e) => setForm((p) => ({ ...p, name: e.target.value }))} placeholder="Engineering" className="mt-1 w-full rounded-lg border border-gray-300 px-3 py-2 text-sm dark:border-gray-600 dark:bg-gray-700 dark:text-white" /></div>
              <div>
                <label className="text-sm font-medium text-gray-700 dark:text-gray-300">{t("departments.parentDepartment")}</label>
                <select aria-label="form" value={form.parent_id} onChange={(e) => setForm((p) => ({ ...p, parent_id: e.target.value }))} className="mt-1 w-full rounded-lg border border-gray-300 px-3 py-2 text-sm dark:border-gray-600 dark:bg-gray-700 dark:text-white">
                  <option value="">{t("departments.rootLevel")}</option>
                  {allDepts.filter((d: any) => d.id !== editDept?.id).map((d: any) => <option key={d.id} value={d.id}>{d.name}</option>)}
                </select>
              </div>
              <div className="grid grid-cols-2 gap-3">
                <div><label className="text-sm font-medium text-gray-700 dark:text-gray-300">{t("departments.headcount")}</label><input aria-label="form" type="number" value={form.headcount} onChange={(e) => setForm((p) => ({ ...p, headcount: Number(e.target.value) }))} className="mt-1 w-full rounded-lg border border-gray-300 px-3 py-2 text-sm dark:border-gray-600 dark:bg-gray-700 dark:text-white" /></div>
                <div><label className="text-sm font-medium text-gray-700 dark:text-gray-300">{t("departments.budget")}</label><input aria-label="form" type="number" value={form.budget} onChange={(e) => setForm((p) => ({ ...p, budget: Number(e.target.value) }))} className="mt-1 w-full rounded-lg border border-gray-300 px-3 py-2 text-sm dark:border-gray-600 dark:bg-gray-700 dark:text-white" /></div>
              </div>
              <div><label className="text-sm font-medium text-gray-700 dark:text-gray-300">{t("departments.costCenter")}</label><input aria-label="CC-1000" value={form.cost_center} onChange={(e) => setForm((p) => ({ ...p, cost_center: e.target.value }))} placeholder="CC-1000" className="mt-1 w-full rounded-lg border border-gray-300 px-3 py-2 text-sm dark:border-gray-600 dark:bg-gray-700 dark:text-white" /></div>
            </div>
            <div className="mt-5 flex justify-end gap-2">
              <button onClick={() => setShowAdd(false)} className="rounded-lg px-4 py-2 text-sm text-gray-500 hover:bg-gray-100 dark:hover:bg-gray-700">Cancel</button>
              <button onClick={handleSave} disabled={!form.name.trim()} className="flex items-center gap-2 rounded-lg bg-indigo-600 px-4 py-2 text-sm font-medium text-white hover:bg-indigo-700 disabled:opacity-50"><Check className="h-4 w-4" />Save</button>
            </div>
          </div>
        </div>
      )}

      {/* Delete confirm */}
      {confirmDelete && (
        <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/50" onClick={() => setConfirmDelete(null)}>
          <div role="dialog" aria-modal="true" className="w-full max-w-sm rounded-xl bg-white p-6 shadow-xl dark:bg-gray-800" onClick={(e) => e.stopPropagation()}>
            <div className="flex items-center gap-3"><div className="rounded-full bg-red-100 p-2 dark:bg-red-900/30"><Trash2 className="h-5 w-5 text-red-600" /></div><div><h2 className="font-semibold text-gray-900 dark:text-white">Delete {confirmDelete.name}?</h2><p className="text-sm text-gray-500">{t("departments.deleteConfirm")}</p></div></div>
            <div className="mt-5 flex justify-end gap-2"><button onClick={() => setConfirmDelete(null)} className="rounded-lg px-4 py-2 text-sm text-gray-500 hover:bg-gray-100 dark:hover:bg-gray-700">Cancel</button><button onClick={() => handleDelete(confirmDelete.id)} className="rounded-lg bg-red-600 px-4 py-2 text-sm font-medium text-white hover:bg-red-700">Delete</button></div>
          </div>
        </div>
      )}
    </div>
  );
}
