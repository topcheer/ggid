"use client";
import { useTranslations } from "@/lib/i18n";
import { useState, useEffect, useCallback } from "react";
import { Shield, Upload, FileKey, RefreshCw, AlertTriangle, Ban } from "lucide-react";

interface Cert { id: string; name: string; issuer: string; type: "TLS" | "signing" | "JWT"; expiry_date: string; fingerprint: string; auto_renew: boolean; days_to_expiry: number; }

const typeColors: Record<string, string> = { TLS: "bg-blue-100 dark:bg-blue-900/30 dark:text-blue-400", signing: "bg-purple-100 dark:bg-purple-900/30 dark:text-purple-400", JWT: "bg-teal-100 dark:bg-teal-900/30 dark:text-teal-400" };

export default function CertificateManagementPage() {
  const t = useTranslations();
  const [certs, setCerts] = useState<Cert[]>([]);
  const [loading, setLoading] = useState(false);
  const [showUpload, setShowUpload] = useState(false);
  const [showCSR, setShowCSR] = useState(false);

  const fetchData = useCallback(async () => {
    setLoading(true);
    try {
      const res = await fetch("/api/v1/auth/certificates", { headers: { "X-Tenant-ID": "00000000-0000-0000-0000-000000000001" } });
      if (res.ok) { const d = await res.json(); setCerts(d.certificates || d || []); }
    } catch { /* noop */ }
    finally { setLoading(false); }
  }, []);

  useEffect(() => { fetchData(); }, [fetchData]);

  const renewCert = async (id: string) => {
    try { await fetch("/api/v1/auth/certificates/" + id + "/renew", { method: "POST", headers: { "X-Tenant-ID": "00000000-0000-0000-0000-000000000001" } }); fetchData(); }
    catch { /* noop */ }
  };

  const revokeCert = async (id: string) => {
    try { await fetch("/api/v1/auth/certificates/" + id, { method: "DELETE", headers: { "X-Tenant-ID": "00000000-0000-0000-0000-000000000001" } }); fetchData(); }
    catch { /* noop */ }
  };

  const expiringSoon = certs.filter((c) => c.days_to_expiry <= 30);

  return (
    <div className="space-y-6">
      <div className="flex items-center justify-between">
        <div><h1 className="text-2xl font-bold flex items-center gap-2"><Shield className="w-6 h-6 text-blue-500" /> {t("backend.certManagement.title")}</h1><p className="text-sm text-gray-500 mt-1">Manage TLS, signing, and JWT certificates with renewal alerts.</p></div>
        <div className="flex gap-2"><button onClick={() => setShowUpload(true)} className="px-4 py-2 rounded-lg bg-blue-600 text-white text-sm font-medium flex items-center gap-2"><Upload className="w-4 h-4" /> Upload</button><button onClick={() => setShowCSR(true)} className="px-4 py-2 rounded-lg border dark:border-gray-700 text-sm flex items-center gap-2"><FileKey className="w-4 h-4" /> {t("backend.certManagement.generateCsr")}</button></div>
      </div>

      {expiringSoon.length > 0 && (<div className="rounded-lg border border-yellow-300 dark:border-yellow-800 bg-yellow-50 dark:bg-yellow-900/20 p-4"><div className="flex items-center gap-2"><AlertTriangle className="w-5 h-5 text-yellow-500" /><span className="font-semibold text-yellow-700 dark:text-yellow-400">{expiringSoon.length} certificate(s) expiring within 30 days</span></div><div className="mt-2 space-y-1">{expiringSoon.map((c) => (<div key={c.id} className="text-sm flex items-center gap-2"><span className="font-medium">{c.name}</span><span className="text-xs text-gray-500">({c.days_to_expiry} days)</span><button onClick={() => renewCert(c.id)} className="text-xs text-blue-600 hover:underline flex items-center gap-1"><RefreshCw className="w-3 h-3" /> Renew</button></div>))}</div></div>)}

      <div className="overflow-x-auto rounded-lg border dark:border-gray-800">
        <table className="w-full text-sm"><thead className="bg-gray-50 dark:bg-gray-900/50"><tr><th className="px-4 py-3 text-left font-medium">{t("backend.certManagement.name")}</th><th className="px-4 py-3 text-left font-medium">{t("backend.certManagement.issuer")}</th><th className="px-4 py-3 text-left font-medium">Type</th><th className="px-4 py-3 text-left font-medium">{t("backend.certManagement.expiry")}</th><th className="px-4 py-3 text-left font-medium">Auto-Renew</th><th className="px-4 py-3 text-left font-medium">{t("backend.certManagement.actions")}</th></tr></thead>
          <tbody className="divide-y dark:divide-gray-800">{certs.map((c) => (<tr key={c.id} className="hover:bg-gray-50 dark:hover:bg-gray-900/30"><td className="px-4 py-3"><span className="font-medium">{c.name}</span><p className="text-xs text-gray-400 font-mono">{c.fingerprint.substring(0, 24)}...</p></td><td className="px-4 py-3 text-xs text-gray-500">{c.issuer}</td><td className="px-4 py-3"><span className={"px-2 py-0.5 rounded text-xs " + typeColors[c.type]}>{c.type}</span></td><td className="px-4 py-3"><span className={"text-xs " + (c.days_to_expiry <= 7 ? "text-red-600 font-medium" : c.days_to_expiry <= 30 ? "text-yellow-600" : "text-gray-500")}>{c.expiry_date} ({c.days_to_expiry}d)</span></td><td className="px-4 py-3">{c.auto_renew ? <span className="text-xs text-green-600">Yes</span> : <span className="text-xs text-gray-400">{t("backend.certManagement.no")}</span>}</td><td className="px-4 py-3"><div className="flex gap-2"><button onClick={() => renewCert(c.id)} className="text-xs text-blue-600 hover:underline">Renew</button><button onClick={() => revokeCert(c.id)} className="text-xs text-red-600 hover:underline flex items-center gap-1"><Ban className="w-3 h-3" /> Revoke</button></div></td></tr>))}{certs.length === 0 && !loading && <tr><td colSpan={6} className="px-4 py-8 text-center text-gray-500">No certificates.</td></tr>}</tbody>
        </table>
      </div>

      {showUpload && (<div className="fixed inset-0 bg-black/50 flex items-center justify-center z-50" onClick={() => setShowUpload(false)}><div className="bg-white dark:bg-gray-900 rounded-lg shadow-xl max-w-md w-full mx-4" onClick={(e) => e.stopPropagation()}><div className="flex items-center justify-between px-6 py-4 border-b dark:border-gray-800"><h3 className="font-semibold">Upload Certificate</h3><button onClick={() => setShowUpload(false)} aria-label="Close dialog" className="text-gray-400">X</button></div><div className="px-6 py-4 space-y-3"><div><label className="text-sm font-medium">Certificate (PEM)</label><textarea placeholder="-----BEGIN CERTIFICATE-----" className="w-full mt-1 px-3 py-2 rounded-lg border dark:border-gray-700 dark:bg-gray-800 text-sm font-mono h-24" /></div><div><label className="text-sm font-medium">Private Key (PEM)</label><textarea placeholder="-----BEGIN PRIVATE KEY-----" className="w-full mt-1 px-3 py-2 rounded-lg border dark:border-gray-700 dark:bg-gray-800 text-sm font-mono h-24" /></div><div><label className="text-sm font-medium">{t("backend.certManagement.name")}</label><input type="text" placeholder="prod-tls-cert" className="w-full mt-1 px-3 py-2 rounded-lg border dark:border-gray-700 dark:bg-gray-800 text-sm" /></div></div><div className="flex justify-end gap-2 px-6 py-4 border-t dark:border-gray-800"><button onClick={() => setShowUpload(false)} className="px-4 py-2 rounded-lg border dark:border-gray-700 text-sm">{t("backend.certManagement.cancel")}</button><button onClick={() => setShowUpload(false)} className="px-4 py-2 rounded-lg bg-blue-600 text-white text-sm font-medium">Upload</button></div></div></div>)}

      {showCSR && (<div className="fixed inset-0 bg-black/50 flex items-center justify-center z-50" onClick={() => setShowCSR(false)}><div className="bg-white dark:bg-gray-900 rounded-lg shadow-xl max-w-md w-full mx-4" onClick={(e) => e.stopPropagation()}><div className="flex items-center justify-between px-6 py-4 border-b dark:border-gray-800"><h3 className="font-semibold">{t("backend.certManagement.generateCsr")}</h3><button onClick={() => setShowCSR(false)} aria-label="Close dialog" className="text-gray-400">X</button></div><div className="px-6 py-4 space-y-3"><div className="grid grid-cols-2 gap-2"><div><label className="text-sm font-medium">{t("backend.certManagement.cn")}</label><input type="text" placeholder="auth.example.com" className="w-full mt-1 px-3 py-2 rounded-lg border dark:border-gray-700 dark:bg-gray-800 text-sm" /></div><div><label className="text-sm font-medium">Type</label><select className="w-full mt-1 px-3 py-2 rounded-lg border dark:border-gray-700 dark:bg-gray-800 text-sm"><option>TLS</option><option>signing</option><option>{t("backend.certManagement.jwt")}</option></select></div></div><div><label className="text-sm font-medium">Organization</label><input type="text" placeholder="GGID Inc" className="w-full mt-1 px-3 py-2 rounded-lg border dark:border-gray-700 dark:bg-gray-800 text-sm" /></div></div><div className="flex justify-end gap-2 px-6 py-4 border-t dark:border-gray-800"><button onClick={() => setShowCSR(false)} className="px-4 py-2 rounded-lg border dark:border-gray-700 text-sm">{t("backend.certManagement.cancel")}</button><button onClick={() => setShowCSR(false)} className="px-4 py-2 rounded-lg bg-blue-600 text-white text-sm font-medium">{t("backend.certManagement.generate")}</button></div></div></div>)}
    </div>
  );
}
