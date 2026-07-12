"use client";
import { useState } from "react";

interface LoginAttempt {
  user: string;
  ip: string;
  status: "success" | "failed" | "blocked";
  timestamp: string;
  device: string;
  location: string;
}

export default function LoginSecurityCenterPage() {
  const [attempts] = useState<LoginAttempt[]>([
    { user: "john.doe", ip: "192.168.1.50", status: "success", timestamp: "16:42:01", device: "Chrome / macOS", location: "San Francisco, US" },
    { user: "jane.smith", ip: "10.0.0.15", status: "failed", timestamp: "16:41:30", device: "Safari / iOS", location: "New York, US" },
    { user: "unknown", ip: "45.146.165.37", status: "blocked", timestamp: "16:40:15", device: "Unknown", location: "Unknown (TOR exit node)" },
    { user: "admin", ip: "172.16.0.1", status: "success", timestamp: "16:38:00", device: "Firefox / Linux", location: "Austin, US" },
    { user: "bob.wilson", ip: "45.146.165.37", status: "failed", timestamp: "16:35:22", device: "Unknown", location: "Unknown (TOR exit node)" },
  ]);
  const [blocklist, setBlocklist] = useState(["45.146.165.0/24", "193.27.228.0/22"]);
  const [newIp, setNewIp] = useState("");
  const botStats = { total_blocked: 1247, captcha_challenged: 892, rate_limited: 355, top_patterns: ["rapid retries", "credential stuffing", "TOR exit nodes"] };

  const statusColors: Record<string, string> = { success: "bg-green-100 text-green-700", failed: "bg-yellow-100 text-yellow-700", blocked: "bg-red-100 text-red-700" };

  return (
    <div className="p-8 space-y-6 max-w-5xl">
      <h1 className="text-2xl font-bold">Login Security Center</h1>
      <p className="text-gray-600">Monitor login attempts, suspicious activity, IP blocklist, and bot detection.</p>

      <div className="bg-white rounded-lg p-6 shadow">
        <h2 className="text-lg font-semibold mb-4">Recent Login Attempts</h2>
        <table className="w-full text-sm"><thead><tr className="border-b text-left"><th className="py-2">User</th><th>IP</th><th>Status</th><th>Time</th><th>Device</th><th>Location</th></tr></thead><tbody>
          {attempts.map((a: LoginAttempt, i: number) => (
            <tr key={i} className={`border-b ${a.status === "blocked" || (a.location.includes("TOR")) ? "bg-red-50" : ""}`}><td className="py-2 font-medium">{a.user}</td><td className="font-mono text-xs">{a.ip}</td><td><span className={`px-2 py-1 rounded text-xs ${statusColors[a.status] || ""}`}>{a.status}</span></td><td className="text-xs text-gray-500">{a.timestamp}</td><td className="text-xs">{a.device}</td><td className="text-xs">{a.location}</td></tr>
          ))}
        </tbody></table>
      </div>

      <div className="bg-white rounded-lg p-6 shadow space-y-4">
        <h2 className="text-lg font-semibold">IP Blocklist</h2>
        <div className="flex flex-wrap gap-2">{blocklist.map((ip: string, i: number) => (<span key={i} className="px-2 py-1 bg-red-50 text-red-700 rounded text-sm font-mono flex items-center gap-2">{ip}<button onClick={() => setBlocklist(blocklist.filter((_, j) => j !== i))} className="text-red-500 hover:text-red-700">x</button></span>))}</div>
        <div className="flex gap-2"><input type="text" value={newIp} onChange={(e) => setNewIp(e.target.value)} placeholder="Add IP or CIDR" className="border rounded px-3 py-2 flex-1 text-sm font-mono" /><button onClick={() => { if (newIp) { setBlocklist([...blocklist, newIp]); setNewIp(""); } }} className="px-4 py-2 bg-red-600 text-white rounded text-sm hover:bg-red-700">Block</button></div>
      </div>

      <div className="bg-white rounded-lg p-6 shadow">
        <h2 className="text-lg font-semibold mb-4">Bot Detection Stats (24h)</h2>
        <div className="grid grid-cols-3 gap-4"><div className="text-center"><div className="text-2xl font-bold text-red-600">{botStats.total_blocked}</div><div className="text-xs text-gray-500">Total Blocked</div></div><div className="text-center"><div className="text-2xl font-bold text-yellow-600">{botStats.captcha_challenged}</div><div className="text-xs text-gray-500">CAPTCHA Challenged</div></div><div className="text-center"><div className="text-2xl font-bold text-orange-600">{botStats.rate_limited}</div><div className="text-xs text-gray-500">Rate Limited</div></div></div>
        <div className="mt-4"><div className="text-sm font-medium mb-2">Top Attack Patterns</div><div className="space-y-1">{botStats.top_patterns.map((p, i) => (<div key={i} className="text-sm border-b py-1">{p}</div>))}</div></div>
      </div>
    </div>
  );
}
