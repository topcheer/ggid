"use client";

import { useIdpMetadataImportConfig } from "@ggid/sdk-react";
import { Upload, FileText, CheckCircle, AlertCircle } from "lucide-react";
import { useState } from "react";

export default function IdpMetadataImportConfigPage() {
  const { data, loading, error, refresh, importMetadata } = useIdpMetadataImportConfig();
  const [tab, setTab] = useState("url");
  const [urlInput, setUrlInput] = useState("");
  const [xmlInput, setXmlInput] = useState("");
  if (loading) return <div className="p-8 text-gray-400">Loading...</div>;
  if (error) return <div className="p-8 text-red-400">Error: {error}</div>;

  return (
    <div className="min-h-screen bg-gray-950 text-white p-8">
      <div className="mb-8"><h1 className="text-2xl font-bold">IdP Metadata Import</h1><p className="text-sm text-gray-400 mt-1">Import SAML/OIDC provider metadata</p></div>

      <div className="bg-gray-900 rounded-xl p-6 mb-6">
        <div className="flex gap-2 mb-4"><button onClick={() => setTab("url")} className={"px-3 py-1.5 rounded-lg text-sm font-medium transition " + (tab === "url" ? "bg-blue-600" : "bg-gray-800")}>URL</button><button onClick={() => setTab("xml")} className={"px-3 py-1.5 rounded-lg text-sm font-medium transition " + (tab === "xml" ? "bg-blue-600" : "bg-gray-800")}>XML Paste</button><button onClick={() => setTab("upload")} className={"px-3 py-1.5 rounded-lg text-sm font-medium transition " + (tab === "upload" ? "bg-blue-600" : "bg-gray-800")}>Upload</button></div>

        {tab === "url" && <input type="text" value={urlInput} onChange={(e) => setUrlInput(e.target.value)} placeholder="https://idp.example.com/metadata" className="w-full px-3 py-2 bg-gray-800 rounded-lg text-sm" />}
        {tab === "xml" && <textarea value={xmlInput} onChange={(e) => setXmlInput(e.target.value)} placeholder="<EntityDescriptor..." rows={6} className="w-full px-3 py-2 bg-gray-800 rounded-lg text-sm font-mono" />}
        {tab === "upload" && <div className="border-2 border-dashed border-gray-700 rounded-lg p-8 text-center"><Upload className="w-8 h-8 text-gray-600 mx-auto mb-2" /><p className="text-sm text-gray-400">Drop metadata XML file here</p></div>}

        <button onClick={() => importMetadata(tab === "url" ? urlInput : xmlInput)} className="mt-3 px-4 py-2 bg-blue-600 hover:bg-blue-700 rounded-lg text-sm font-medium transition">Fetch & Preview</button>
      </div>

      {data?.preview && (
        <div className="bg-gray-900 rounded-xl p-6 mb-6">
          <h2 className="text-sm font-semibold mb-3 flex items-center gap-2"><FileText className="w-4 h-4 text-blue-400" /> Metadata Preview</h2>
          <div className="grid grid-cols-2 gap-3 text-sm">
            <div><span className="text-xs text-gray-400">Entity ID</span><p className="font-mono">{data.preview.entity_id}</p></div>
            <div><span className="text-xs text-gray-400">SSO URL</span><p className="font-mono text-xs">{data.preview.sso_url}</p></div>
            <div><span className="text-xs text-gray-400">NameID Format</span><p>{data.preview.name_id_format}</p></div>
            <div><span className="text-xs text-gray-400">Certificates</span><p>{data.preview.cert_count} cert(s)</p></div>
          </div>
          <div className="mt-3 flex items-center gap-2"><span className={"text-xs px-2 py-0.5 rounded flex items-center gap-1 " + (data.preview.valid ? "bg-green-900 text-green-300" : "bg-red-900 text-red-300")}>{data.preview.valid ? <CheckCircle className="w-3 h-3" /> : <AlertCircle className="w-3 h-3" />}{data.preview.valid ? "Valid" : "Invalid"}</span><button onClick={refresh} className="ml-auto px-3 py-1 bg-green-700 hover:bg-green-600 rounded text-xs">Import</button></div>
        </div>
      )}

      <div className="bg-gray-900 rounded-xl p-6">
        <h2 className="text-sm font-semibold mb-3">Saved IdPs</h2>
        <div className="space-y-2">{(data?.saved_idps ?? []).map((idp) => (
          <div key={idp.entity_id} className="flex items-center gap-3 bg-gray-800 rounded p-3"><CheckCircle className="w-4 h-4 text-green-400" /><div className="flex-1"><p className="text-sm font-medium">{idp.name}</p><p className="text-xs text-gray-400 font-mono">{idp.entity_id}</p></div><span className="text-xs text-gray-500">imported {idp.imported_at}</span></div>
        ))}</div>
      </div>
    </div>
  );
}
