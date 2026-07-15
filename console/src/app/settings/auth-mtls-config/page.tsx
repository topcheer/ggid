"use client";
import { useEffect, useState } from "react";
import { useAuthMtlsConfig, AuthMtlsConfig, TrustedCaCert } from "@ggid/sdk-react";
import { useTranslations } from "@/lib/i18n";

export default function AuthMtlsConfigPage() {
  const t = useTranslations();

  const { config, loading, error, fetchConfig, updateConfig } = useAuthMtlsConfig();
  const [form, setForm] = useState<AuthMtlsConfig | null>(null);
  const [saving, setSaving] = useState(false);

  useEffect(() => { fetchConfig(); }, [fetchConfig]);
  useEffect(() => { if (config) setForm(config); }, [config]);

  const handleSave = async () => { if (!form) return; setSaving(true); await updateConfig(form); setSaving(false); };

  if (loading && !form) return <div className="p-8">Loading...</div>;
  if (error) return <div className="p-8 text-red-600">Error: {error}</div>;
  if (!form) return <div className="p-8">No data</div>;

  return (
    <div className="p-8 space-y-6 max-w-4xl">
      <h1 className="text-2xl font-bold">mTLS Authentication Configuration</h1>
      <p className="text-gray-600">Configure mutual TLS certificate-based authentication.</p>

      <div className="flex items-center gap-3 bg-white rounded-lg p-4 shadow">
        <input type="checkbox" checked={form.require_mtls} onChange={(e) => setForm({ ...form, require_mtls: e.target.checked })} className="w-5 h-5" />
        <label className="font-medium">Require mTLS</label>
      </div>

      <div className="bg-white rounded-lg p-6 shadow">
        <h2 className="text-lg font-semibold mb-4">Trusted CA Certificates</h2>
        <table className="w-full text-sm"><thead><tr className="border-b text-left"><th className="py-2">Name</th><th>Fingerprint</th><th>Expiry</th></tr></thead><tbody>
          {form.trusted_ca_certs.map((c: TrustedCaCert, i: number) => (
            <tr key={i} className="border-b"><td className="py-2 font-medium">{c.name}</td><td className="font-mono text-xs">{c.fingerprint}</td><td className="text-xs text-gray-500">{c.expiry}</td></tr>
          ))}
        </tbody></table>
      </div>

      <div className="bg-white rounded-lg p-6 shadow space-y-4">
        <h2 className="text-lg font-semibold">Binding & Revocation</h2>
        <div className="flex items-center gap-3"><input type="checkbox" checked={form.per_client_cert_binding} onChange={(e) => setForm({ ...form, per_client_cert_binding: e.target.checked })} className="w-4 h-4" /><label>Per-Client Certificate Binding</label></div>
        <div>
          <label className="block text-sm font-medium mb-1">Revocation Check</label>
          <select value={form.revocation_check} onChange={(e) => setForm({ ...form, revocation_check: e.target.value as AuthMtlsConfig["revocation_check"] })} className="border rounded px-3 py-2">
            <option value="none">None</option><option value="CRL">CRL</option><option value="OCSP">OCSP</option><option value="both">Both</option>
          </select>
        </div>
      </div>

      <div className="bg-white rounded-lg p-6 shadow space-y-3">
        <h2 className="text-lg font-semibold">Fallback Options</h2>
        <div className="flex items-center gap-3"><input type="checkbox" checked={form.allow_self_signed} onChange={(e) => setForm({ ...form, allow_self_signed: e.target.checked })} className="w-4 h-4" /><label>Allow Self-Signed Certificates</label></div>
        <div className="flex items-center gap-3"><input type="checkbox" checked={form.fallback_to_bearer} onChange={(e) => setForm({ ...form, fallback_to_bearer: e.target.checked })} className="w-4 h-4" /><label>Fallback to Bearer Token</label></div>
      </div>

      <button onClick={handleSave} disabled={saving} className="px-6 py-2 bg-blue-600 text-white rounded-lg hover:bg-blue-700 disabled:opacity-50">{saving ? "Saving..." : "Save Changes"}</button>
    </div>
  );
}
