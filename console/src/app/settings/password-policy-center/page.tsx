"use client";
import { useState, useEffect } from "react";
import { useTranslations } from "@/lib/i18n";

interface ComplexityRules {
  min_length: number;
  require_uppercase: boolean;
  require_lowercase: boolean;
  require_digit: boolean;
  require_special: boolean;
}

interface PerRoleOverride {
  role: string;
  min_length: number;
  history_count: number;
  expiry_days: number;
}

export default function PasswordPolicyCenterPage() {
  const t = useTranslations();

  const [complexity, setComplexity] = useState<ComplexityRules>({ min_length: 12, require_uppercase: true, require_lowercase: true, require_digit: true, require_special: true });
  const [breachDetection, setBreachDetection] = useState(true);
  const [hibpApiKey, setHibpApiKey] = useState("");
  const [historyCount, setHistoryCount] = useState(5);
  const [expiryDays, setExpiryDays] = useState(90);
  const [blocklist, setBlocklist] = useState<string[]>([]);
  const [newBlockEntry, setNewBlockEntry] = useState("");
  const [pepperStatus, setPepperStatus] = useState(true);
  const [overrides, setOverrides] = useState<PerRoleOverride[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    fetch("/api/v1/auth/password-policy", {
      headers: { "Content-Type": "application/json", "X-Tenant-ID": "00000000-0000-0000-0000-000000000001" },
    })
      .then(r => { if (!r.ok) throw new Error(`HTTP ${r.status}`); return r.json(); })
      .then(data => {
        if (data) {
          if (data.complexity) setComplexity(data.complexity);
          if (data.breach_detection !== undefined) setBreachDetection(data.breach_detection);
          if (data.hibp_api_key) setHibpApiKey(data.hibp_api_key);
          if (data.history_count) setHistoryCount(data.history_count);
          if (data.expiry_days) setExpiryDays(data.expiry_days);
          if (data.blocklist) setBlocklist(data.blocklist);
          if (data.pepper_status !== undefined) setPepperStatus(data.pepper_status);
          if (data.overrides) setOverrides(data.overrides);
        }
        setLoading(false);
      })
      .catch(e => { setError(e.message); setLoading(false); });
  }, []);

  if (loading) return <div className="p-8"><p>Loading...</p></div>;
  if (error) return <div className="p-8 text-red-600">Error: {error}</div>;

  return (
    <div className="p-8 space-y-6 max-w-5xl">
      <h1 className="text-2xl font-bold">Password Policy Center</h1>
      <p className="text-gray-600">Configure password complexity, breach detection, history, and per-role overrides.</p>

      <div className="bg-white rounded-lg p-6 shadow space-y-4">
        <h2 className="text-lg font-semibold">Complexity Rules</h2>
        <div><label className="block text-sm font-medium mb-1">Minimum Length</label><input type="number" value={complexity.min_length} onChange={(e) => setComplexity({ ...complexity, min_length: parseInt(e.target.value) || 0 })} className="border rounded px-3 py-2 w-32" /></div>
        <div className="grid grid-cols-2 gap-3">
          <div className="flex items-center gap-3"><input type="checkbox" checked={complexity.require_uppercase} onChange={(e) => setComplexity({ ...complexity, require_uppercase: e.target.checked })} className="w-4 h-4" /><label>Require Uppercase</label></div>
          <div className="flex items-center gap-3"><input type="checkbox" checked={complexity.require_lowercase} onChange={(e) => setComplexity({ ...complexity, require_lowercase: e.target.checked })} className="w-4 h-4" /><label>Require Lowercase</label></div>
          <div className="flex items-center gap-3"><input type="checkbox" checked={complexity.require_digit} onChange={(e) => setComplexity({ ...complexity, require_digit: e.target.checked })} className="w-4 h-4" /><label>Require Digit</label></div>
          <div className="flex items-center gap-3"><input type="checkbox" checked={complexity.require_special} onChange={(e) => setComplexity({ ...complexity, require_special: e.target.checked })} className="w-4 h-4" /><label>Require Special Character</label></div>
        </div>
      </div>

      <div className="bg-white rounded-lg p-6 shadow space-y-4">
        <h2 className="text-lg font-semibold">Breach Detection (HIBP)</h2>
        <div className="flex items-center gap-3"><input type="checkbox" checked={breachDetection} onChange={(e) => setBreachDetection(e.target.checked)} className="w-4 h-4" /><label>Check passwords against HaveIBeenPwned database</label></div>
        {breachDetection && (<div><label className="block text-sm font-medium mb-1">HIBP API Key</label><input autoComplete="current-password" type="password" value={hibpApiKey} onChange={(e) => setHibpApiKey(e.target.value)} placeholder="Enter HIBP API key" className="border rounded px-3 py-2 w-full" /></div>)}
      </div>

      <div className="bg-white rounded-lg p-6 shadow space-y-4">
        <h2 className="text-lg font-semibold">History &amp; Expiry</h2>
        <div className="grid grid-cols-2 gap-4">
          <div><label className="block text-sm font-medium mb-1">Password History (remember last N)</label><input type="number" value={historyCount} onChange={(e) => setHistoryCount(parseInt(e.target.value) || 0)} className="border rounded px-3 py-2 w-32" /></div>
          <div><label className="block text-sm font-medium mb-1">Expiry (days, 0 = never)</label><input type="number" value={expiryDays} onChange={(e) => setExpiryDays(parseInt(e.target.value) || 0)} className="border rounded px-3 py-2 w-32" /></div>
        </div>
      </div>

      <div className="bg-white rounded-lg p-6 shadow space-y-4">
        <h2 className="text-lg font-semibold">Common Password Blocklist</h2>
        <div className="flex flex-wrap gap-2">{blocklist.map((p: string, i: number) => (<span key={i} className="px-2 py-1 bg-gray-100 rounded text-sm flex items-center gap-2">{p}<button onClick={() => setBlocklist(blocklist.filter((_, j) => j !== i))} className="text-red-500 hover:text-red-700">x</button></span>))}</div>
        <div className="flex gap-2"><input aria-label="Add password to blocklist" type="text" value={newBlockEntry} onChange={(e) => setNewBlockEntry(e.target.value)} placeholder="Add password to blocklist" className="border rounded px-3 py-2 flex-1 text-sm" /><button onClick={() => { if (newBlockEntry) { setBlocklist([...blocklist, newBlockEntry]); setNewBlockEntry(""); } }} className="px-4 py-2 bg-blue-600 text-white rounded text-sm hover:bg-blue-700">Add</button></div>
      </div>

      <div className="bg-white rounded-lg p-6 shadow flex items-center gap-4">
        <h2 className="text-lg font-semibold">Pepper Status</h2>
        <span className={`px-3 py-1 rounded text-sm ${pepperStatus ? "bg-green-100 text-green-700" : "bg-red-100 text-red-700"}`}>{pepperStatus ? "Active" : "Inactive"}</span>
        <button onClick={() => setPepperStatus(!pepperStatus)} className="text-xs text-blue-600 hover:underline">Toggle</button>
      </div>

      <div className="bg-white rounded-lg p-6 shadow">
        <h2 className="text-lg font-semibold mb-4">Per-Role Overrides</h2>
        <table className="w-full text-sm"><thead><tr className="border-b text-left"><th className="py-2">Role</th><th scope="col">Min Length</th><th>History</th><th>Expiry (days)</th></tr></thead><tbody>{overrides.map((o: PerRoleOverride, i: number) => (<tr key={i} className="border-b"><td className="py-2 font-medium">{o.role}</td><td>{o.min_length}</td><td>{o.history_count > 0 ? o.history_count : "none"}</td><td>{o.expiry_days > 0 ? `${o.expiry_days}d` : "never"}</td></tr>))}</tbody></table>
      </div>
    </div>
  );
}
