"use client";

import { useWebauthnConfig } from "@ggid/sdk-react";
import { Fingerprint, Shield } from "lucide-react";

export default function WebauthnConfigPage() {
  const { data, loading, error, refresh } = useWebauthnConfig();
  if (loading) return <div className="p-8 text-gray-400">Loading WebAuthn config...</div>;
  if (error) return <div className="p-8 text-red-400">Error: {error}</div>;

  return (
    <div className="min-h-screen bg-gray-950 text-white p-8">
      <div className="flex items-center justify-between mb-8">
        <div><h1 className="text-2xl font-bold">WebAuthn Configuration</h1><p className="text-sm text-gray-400 mt-1">Relying party settings and authenticator policies</p></div>
        <button onClick={refresh} className="px-4 py-2 bg-blue-600 hover:bg-blue-700 rounded-lg text-sm font-medium transition">Save</button>
      </div>

      <div className="grid grid-cols-1 md:grid-cols-2 gap-6">
        <div className="bg-gray-900 rounded-xl p-6 space-y-4">
          <h2 className="text-sm font-semibold flex items-center gap-2"><Fingerprint className="w-4 h-4 text-blue-400" /> Relying Party</h2>
          <div><label className="text-xs text-gray-400">RP ID</label><input type="text" defaultValue={data?.rp_id} className="w-full mt-1 px-3 py-2 bg-gray-800 rounded-lg text-sm" /></div>
          <div><label className="text-xs text-gray-400">RP Name</label><input type="text" defaultValue={data?.rp_name} className="w-full mt-1 px-3 py-2 bg-gray-800 rounded-lg text-sm" /></div>
          <div><label className="text-xs text-gray-400">Origin</label><input type="text" defaultValue={data?.origin} className="w-full mt-1 px-3 py-2 bg-gray-800 rounded-lg text-sm" /></div>
        </div>
        <div className="bg-gray-900 rounded-xl p-6 space-y-4">
          <h2 className="text-sm font-semibold flex items-center gap-2"><Shield className="w-4 h-4 text-green-400" /> Security Policy</h2>
          <div><label className="text-xs text-gray-400">Attestation Requirement</label><select defaultValue={data?.attestation_requirement} className="w-full mt-1 px-3 py-2 bg-gray-800 rounded-lg text-sm"><option>none</option><option>indirect</option><option>direct</option></select></div>
          <div><label className="text-xs text-gray-400">User Verification</label><select defaultValue={data?.user_verification} className="w-full mt-1 px-3 py-2 bg-gray-800 rounded-lg text-sm"><option>required</option><option>preferred</option><option>discouraged</option></select></div>
          <div><label className="text-xs text-gray-400">Timeout: {data?.timeout_seconds}s</label><input type="range" min="30" max="600" defaultValue={data?.timeout_seconds} className="w-full mt-1" /></div>
        </div>
      </div>

      <div className="bg-gray-900 rounded-xl p-6 mt-6">
        <h2 className="text-sm font-semibold mb-3">Supported Algorithms</h2>
        <div className="grid grid-cols-3 gap-3">
          {(data?.supported_algorithms ?? []).map((alg) => (
            <label key={alg.id} className="flex items-center gap-2 bg-gray-800 rounded-lg p-3 cursor-pointer">
              <input type="checkbox" defaultChecked={alg.enabled} />
              <div><p className="text-sm font-medium font-mono">{alg.id}</p><p className="text-xs text-gray-400">COSE: {alg.cose_id}</p></div>
            </label>
          ))}
        </div>
      </div>

      <div className="bg-gray-900 rounded-xl p-6 mt-6">
        <h2 className="text-sm font-semibold mb-3">Per-Platform Configuration</h2>
        <div className="grid grid-cols-1 md:grid-cols-3 gap-4">
          {(data?.per_platform ?? []).map((p) => (
            <div key={p.platform} className="bg-gray-800 rounded-lg p-4">
              <p className="text-sm font-semibold mb-2">{p.platform}</p>
              <div className="space-y-1 text-xs text-gray-400">
                <div className="flex justify-between"><span>Authenticator Type</span><span>{p.authenticator_type}</span></div>
                <div className="flex justify-between"><span>Attachment</span><span>{p.attachment}</span></div>
                <div className="flex justify-between"><span>Enabled</span><span className={p.enabled ? "text-green-400" : "text-red-400"}>{p.enabled ? "Yes" : "No"}</span></div>
              </div>
            </div>
          ))}
        </div>
      </div>
    </div>
  );
}
