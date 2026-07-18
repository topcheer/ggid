"use client";
import { useTranslations } from "@/lib/i18n";

import { usePasswordlessConfig } from "@ggid/sdk-react";
import { KeyRound, Mail, Smartphone, Fingerprint } from "lucide-react";

export default function PasswordlessConfigPage() {
  const t = useTranslations();
  const { data, loading, error, refresh } = usePasswordlessConfig();
  if (loading) return <div className="p-8 text-gray-400">Loading passwordless config...</div>;
  if (error) return <div className="p-8 text-red-400">Error: {error}</div>;

  const icons: Record<string, React.ReactNode> = { magic_link: <Mail className="w-5 h-5" />, passkey: <KeyRound className="w-5 h-5" />, webauthn: <Fingerprint className="w-5 h-5" />, biometric: <Smartphone className="w-5 h-5" /> };

  return (
    <div className="min-h-screen bg-gray-950 text-white p-8">
      <div className="flex items-center justify-between mb-8">
        <div><h1 className="text-2xl font-bold">Passwordless Authentication</h1><p className="text-sm text-gray-400 mt-1">Configure passwordless login methods</p></div>
        <button onClick={refresh} className="px-4 py-2 bg-blue-600 hover:bg-blue-700 rounded-lg text-sm font-medium transition">Save</button>
      </div>

      <div className="bg-gray-900 rounded-xl p-6 mb-6">
        <h2 className="text-sm font-semibold mb-4">Enabled Methods</h2>
        <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
          {(data?.methods ?? []).map((m: any) => (
            <div key={m.id} className="flex items-center gap-3 bg-gray-800 rounded-lg p-4">
              <div className={m.enabled ? "text-green-400" : "text-gray-600"}>{icons[m.id] ?? <KeyRound className="w-5 h-5" />}</div>
              <div className="flex-1"><p className="text-sm font-medium">{m.label}</p><p className="text-xs text-gray-400">{m.description}</p></div>
              <label className="relative inline-flex items-center cursor-pointer">
                <input aria-label="Toggle option" type="checkbox" defaultChecked={m.enabled} className="sr-only peer" />
                <div className="w-9 h-5 bg-gray-700 rounded-full peer-checked:bg-green-600 after:content-[''] after:absolute after:top-0.5 after:left-0.5 after:bg-white after:rounded-full after:h-4 after:w-4 after:transition-all peer-checked:after:translate-x-4" />
              </label>
            </div>
          ))}
        </div>
      </div>

      <div className="grid grid-cols-1 md:grid-cols-2 gap-6 mb-6">
        <div className="bg-gray-900 rounded-xl p-6 space-y-3">
          <h2 className="text-sm font-semibold">Settings</h2>
          <div><label className="text-xs text-gray-400">Magic Link Expiry (minutes)</label><input aria-label="Input field" type="number" defaultValue={data?.magic_link_expiry_minutes} className="w-full mt-1 px-3 py-2 bg-gray-800 rounded-lg text-sm" /></div>
          <div><label className="text-xs text-gray-400">Passkey RP ID</label><input aria-label="Input field" type="text" defaultValue={data?.passkey_rp_id} className="w-full mt-1 px-3 py-2 bg-gray-800 rounded-lg text-sm" /></div>
          <div className="flex items-center gap-2"><input aria-label="Toggle option" type="checkbox" defaultChecked={data?.fallback_to_password} id="fb" /><label htmlFor="fb" className="text-sm">Fallback to password if passwordless fails</label></div>
        </div>
        <div className="bg-gray-900 rounded-xl p-6">
          <h2 className="text-sm font-semibold mb-3">Per-Role Requirement</h2>
          <table className="w-full text-sm"><thead><tr className="border-b border-gray-800 text-gray-400"><th className="text-left py-2">Role</th><th className="text-left py-2">Required Method</th></tr></thead>
            <tbody>{(data?.per_role ?? []).map((r: any) => (
              <tr key={r.role} className="border-b border-gray-800"><td className="py-2 text-sm">{r.role}</td><td className="py-2 text-xs"><span className={"px-2 py-0.5 rounded " + (r.method === "none" ? "bg-gray-700" : "bg-blue-900 text-blue-300")}>{r.method}</span></td></tr>
            ))}</tbody>
          </table>
        </div>
      </div>
    </div>
  );
}
