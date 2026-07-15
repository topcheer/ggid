"use client";
import { useState } from "react";
import { Download, Upload, FileText, CheckCircle, XCircle, Link2 } from "lucide-react";
import { useTranslations } from "@/lib/i18n";
interface MetadataInfo { entity_id: string; sso_url: string; slo_url: string; name_id_format: string; certificates: string[]; }
interface IdpEntry { id: string; entity_id: string; name: string; imported_at: string; status: "active" | "disabled"; }
export default function IdpMetadataImportPage() {
  const t = useTranslations();

  const [tab, setTab] = useState("url");
  const [url, setUrl] = useState("");
  const [xml, setXml] = useState("");
  const [metadata, setMetadata] = useState<MetadataInfo | null>(null);
  const [imported, setImported] = useState<IdpEntry[]>([]);
  const [loading, setLoading] = useState(false);
  const fetchPreview = async () => { setLoading(true); try { const body = tab === "url" ? JSON.stringify({ url }) : JSON.stringify({ xml }); const res = await fetch("/api/v1/auth/idp-metadata/preview", { method: "POST", headers: { "Content-Type": "application/json", "X-Tenant-ID": "00000000-0000-0000-0000-000000000001" }, body }); if (res.ok) setMetadata(await res.json()); } catch { /* noop */ } finally { setLoading(false); } };
  const doImport = async () => { setLoading(true); try { const body = tab === "url" ? JSON.stringify({ url }) : JSON.stringify({ xml }); await fetch("/api/v1/auth/idp-metadata/import", { method: "POST", headers: { "Content-Type": "application/json", "X-Tenant-ID": "00000000-0000-0000-0000-000000000001" }, body }); fetchList(); } catch { /* noop */ } finally { setLoading(false); } };
  const fetchList = async () => { try { const res = await fetch("/api/v1/auth/idp-metadata", { headers: { "X-Tenant-ID": "00000000-0000-0000-0000-000000000001" } }); if (res.ok) { const d = await res.json(); setImported(d.idps || d || []); } } catch { /* noop */ } };
  useState(() => { fetchList(); });
  return (
    <div className="space-y-6">
      <div><h1 className="text-2xl font-bold flex items-center gap-2"><Download className="w-6 h-6 text-blue-500" /> {t("idpMetadataImport.title")}</h1><p className="text-sm text-gray-500 mt-1">Import SAML IdP metadata from URL, XML paste, or file upload.</p></div>
      <div className="flex gap-2"><button onClick={() => setTab("url")} className={"px-3 py-1.5 rounded-lg text-xs font-medium " + (tab === "url" ? "bg-blue-600 text-white" : "border dark:border-gray-700")}>URL Import</button><button onClick={() => setTab("xml")} className={"px-3 py-1.5 rounded-lg text-xs font-medium " + (tab === "xml" ? "bg-blue-600 text-white" : "border dark:border-gray-700")}>XML Paste</button></div>
      {tab === "url" ? <div className="rounded-lg border dark:border-gray-800 p-4"><label className="text-sm font-medium">Metadata URL</label><div className="flex gap-2 mt-1"><input type="text" value={url} onChange={(e) => setUrl(e.target.value)} placeholder="https://idp.example.com/metadata" className="flex-1 px-3 py-2 rounded-lg border dark:border-gray-700 dark:bg-gray-900 text-sm font-mono" /><button onClick={fetchPreview} disabled={loading || !url} className="px-4 py-2 rounded-lg bg-blue-600 text-white text-sm font-medium flex items-center gap-1"><Link2 className="w-4 h-4" /> Fetch</button></div></div> : <div className="rounded-lg border dark:border-gray-800 p-4"><label className="text-sm font-medium">Paste XML Metadata</label><textarea value={xml} onChange={(e) => setXml(e.target.value)} rows={8} placeholder="<EntityDescriptor xmlns=...>" className="w-full mt-1 px-3 py-2 rounded-lg border dark:border-gray-700 dark:bg-gray-900 text-xs font-mono" /><button onClick={fetchPreview} disabled={loading || !xml} className="mt-2 px-4 py-2 rounded-lg bg-blue-600 text-white text-sm font-medium">Parse</button></div>}
      {metadata && (<div className="rounded-lg border dark:border-gray-800 p-4"><div className="flex items-center gap-2 mb-3"><CheckCircle className="w-5 h-5 text-green-500" /><h3 className="text-sm font-semibold">Metadata Valid</h3></div><div className="grid grid-cols-2 gap-3 text-sm"><div><span className="text-xs text-gray-500">Entity ID</span><p className="font-mono text-xs">{metadata.entity_id}</p></div><div><span className="text-xs text-gray-500">SSO URL</span><p className="font-mono text-xs">{metadata.sso_url}</p></div><div><span className="text-xs text-gray-500">SLO URL</span><p className="font-mono text-xs">{metadata.slo_url}</p></div><div><span className="text-xs text-gray-500">NameID Format</span><p className="font-mono text-xs">{metadata.name_id_format}</p></div></div><div className="mt-2"><span className="text-xs text-gray-500">Certificates ({metadata.certificates.length})</span>{metadata.certificates.map((c, i) => <p key={i} className="font-mono text-xs text-gray-400 truncate">{c.substring(0, 40)}...</p>)}</div><button onClick={doImport} disabled={loading} className="mt-3 px-4 py-2 rounded-lg bg-green-600 text-white text-sm font-medium flex items-center gap-2"><Upload className="w-4 h-4" /> Import IdP</button></div>)}
      {imported.length > 0 && <div className="rounded-lg border dark:border-gray-800 p-4"><h3 className="text-sm font-semibold mb-2">Imported IdPs</h3><div className="space-y-1">{imported.map((idp) => (<div key={idp.id} className="flex items-center gap-2 text-sm"><FileText className="w-3.5 h-3.5 text-gray-400" /><span className="font-medium">{idp.name}</span><span className="text-xs text-gray-400 font-mono">{idp.entity_id}</span><span className={"px-1.5 py-0.5 rounded text-xs " + (idp.status === "active" ? "bg-green-100 dark:bg-green-900/30 dark:text-green-400" : "bg-gray-100 dark:bg-gray-800")}>{idp.status}</span></div>))}</div></div>}
    </div>
  );
}
