"use client";
import { useState, useEffect, useCallback } from "react";
import { ArrowRight, Save, Play } from "lucide-react";
import { useTranslations } from "@/lib/i18n";

interface Mapping { id: string; source_attribute: string; target_field: string; transform: "direct" | "regex" | "constant"; transform_value: string; }
interface IdpOverride { idp_id: string; idp_name: string; overrides: number; }

export default function SamlAttributeMappingPage() {
  const t = useTranslations();
  const [mappings, setMappings] = useState<Mapping[]>([]);
  const [overrides, setOverrides] = useState<IdpOverride[]>([]);
  const [loading, setLoading] = useState(false);
  const [showAdd, setShowAdd] = useState(false);
  const [form, setForm] = useState<{ source_attribute: string; target_field: string; transform: "direct" | "regex" | "constant"; transform_value: string }>({ source_attribute: "", target_field: "", transform: "direct", transform_value: "" });
  const [preview, setPreview] = useState<{ input: string; output: string } | null>(null);

  const fetchData = useCallback(async () => {
    setLoading(true);
    try {
      const res = await fetch("/api/v1/auth/saml-attribute-mapping", { headers: { "X-Tenant-ID": "00000000-0000-0000-0000-000000000001" } });
      if (res.ok) { const d = await res.json(); setMappings(d.mappings || []); setOverrides(d.idp_overrides || []); }
    } catch { /* noop */ }
    finally { setLoading(false); }
  }, []);

  useEffect(() => { fetchData(); }, [fetchData]);

  const addMapping = async () => {
    if (!form.source_attribute || !form.target_field) return;
    try { await fetch("/api/v1/auth/saml-attribute-mapping", { method: "POST", headers: { "Content-Type": "application/json", "X-Tenant-ID": "00000000-0000-0000-0000-000000000001" }, body: JSON.stringify(form) }); setShowAdd(false); setForm({ source_attribute: "", target_field: "", transform: "direct", transform_value: "" }); fetchData(); }
    catch { /* noop */ }
  };

  const testMapping = async (id: string) => {
    try { const res = await fetch("/api/v1/auth/saml-attribute-mapping/" + id + "/test", { method: "POST", headers: { "X-Tenant-ID": "00000000-0000-0000-0000-000000000001" } }); if (res.ok) setPreview(await res.json()); }
    catch { /* noop */ }
  };

  return (
    <div className="space-y-6">
      <div className="flex items-center justify-between">
        <div><h1 className="text-2xl font-bold flex items-center gap-2"><ArrowRight className="w-6 h-6 text-blue-500" /> SAML Attribute Mapping</h1><p className="text-sm text-gray-500 mt-1">Map incoming SAML attributes to local user fields with optional transforms.</p></div>
        <button onClick={() => setShowAdd(true)} className="px-4 py-2 rounded-lg bg-blue-600 text-white text-sm font-medium">Add Mapping</button>
      </div>

      <div className="overflow-x-auto rounded-lg border dark:border-gray-800">
        <table className="w-full text-sm"><thead className="bg-gray-50 dark:bg-gray-900/50"><tr><th className="px-4 py-3 text-left font-medium">Source Attribute</th><th className="px-4 py-3 text-left font-medium">Target Field</th><th className="px-4 py-3 text-left font-medium">Transform</th><th className="px-4 py-3 text-left font-medium">Value</th><th className="px-4 py-3 text-left font-medium">Action</th></tr></thead>
          <tbody className="divide-y dark:divide-gray-800">{mappings.map((m) => (<tr key={m.id} className="hover:bg-gray-50 dark:hover:bg-gray-900/30"><td className="px-4 py-3 font-mono text-xs">{m.source_attribute}</td><td className="px-4 py-3 font-mono text-xs font-medium">{m.target_field}</td><td className="px-4 py-3"><span className="px-2 py-0.5 rounded text-xs bg-gray-100 dark:bg-gray-800">{m.transform}</span></td><td className="px-4 py-3 text-xs text-gray-500">{m.transform_value || "-"}</td><td className="px-4 py-3"><button onClick={() => testMapping(m.id)} className="text-xs text-blue-600 hover:underline flex items-center gap-1"><Play className="w-3 h-3" /> Test</button></td></tr>))}{mappings.length === 0 && !loading && <tr><td colSpan={5} className="px-4 py-8 text-center text-gray-500">No mappings.</td></tr>}</tbody>
        </table>
      </div>

      {overrides.length > 0 && (<div className="rounded-lg border dark:border-gray-800 p-4"><h3 className="text-sm font-semibold mb-2">Per-IdP Overrides</h3><div className="space-y-1">{overrides.map((o) => (<div key={o.idp_id} className="flex items-center justify-between text-sm"><span className="font-medium">{o.idp_name}</span><span className="text-xs text-gray-500">{o.overrides} override(s)</span></div>))}</div></div>)}

      {preview && (<div className="rounded-lg border dark:border-gray-800 p-4"><h3 className="text-sm font-semibold mb-2">Test Result</h3><div className="text-sm space-y-1"><div><span className="text-gray-500">Input:</span> <span className="font-mono text-xs">{preview.input}</span></div><div><span className="text-gray-500">Output:</span> <span className="font-mono text-xs text-green-600">{preview.output}</span></div></div></div>)}

      {showAdd && (<div className="fixed inset-0 bg-black/50 flex items-center justify-center z-50" onClick={() => setShowAdd(false)}><div className="bg-white dark:bg-gray-900 rounded-lg shadow-xl max-w-md w-full mx-4" onClick={(e) => e.stopPropagation()}><div className="px-6 py-4 border-b dark:border-gray-800"><h3 className="font-semibold">Add Mapping</h3></div><div className="px-6 py-4 space-y-3"><div><label className="text-sm font-medium">Source Attribute</label><input type="text" value={form.source_attribute} onChange={(e) => setForm({ ...form, source_attribute: e.target.value })} placeholder="http://schemas.../email" className="w-full mt-1 px-3 py-2 rounded-lg border dark:border-gray-700 dark:bg-gray-800 text-sm font-mono" /></div><div><label className="text-sm font-medium">Target Field</label><input type="text" value={form.target_field} onChange={(e) => setForm({ ...form, target_field: e.target.value })} placeholder="email" className="w-full mt-1 px-3 py-2 rounded-lg border dark:border-gray-700 dark:bg-gray-800 text-sm" /></div><div><label className="text-sm font-medium">Transform</label><select value={form.transform} onChange={(e) => setForm({ ...form, transform: e.target.value as "direct" | "regex" | "constant" })} className="w-full mt-1 px-3 py-2 rounded-lg border dark:border-gray-700 dark:bg-gray-800 text-sm"><option value="direct">Direct</option><option value="regex">Regex</option><option value="constant">Constant</option></select></div>{form.transform !== "direct" && <div><label className="text-sm font-medium">Transform Value</label><input type="text" value={form.transform_value} onChange={(e) => setForm({ ...form, transform_value: e.target.value })} placeholder={form.transform === "regex" ? "^([^.]+)@" : "static_value"} className="w-full mt-1 px-3 py-2 rounded-lg border dark:border-gray-700 dark:bg-gray-800 text-sm" /></div>}</div><div className="flex justify-end gap-2 px-6 py-4 border-t dark:border-gray-800"><button onClick={() => setShowAdd(false)} className="px-4 py-2 rounded-lg border dark:border-gray-700 text-sm">Cancel</button><button onClick={addMapping} disabled={!form.source_attribute || !form.target_field} className="px-4 py-2 rounded-lg bg-blue-600 text-white text-sm font-medium disabled:opacity-50 flex items-center gap-1"><Save className="w-4 h-4" /> Save</button></div></div></div>)}
    </div>
  );
}
