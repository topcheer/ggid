"use client";

import { useState, useEffect, useCallback } from "react";
import { KeyRound, Search, Gauge, Lightbulb, CheckCircle2, XCircle } from "lucide-react";
import { useTranslations } from "@/lib/i18n";

interface SecretReport {
  client_id: string;
  client_name: string;
  secret_id: string;
  entropy_bits: number;
  crack_time: string;
  character_diversity: {
    lowercase: boolean;
    uppercase: boolean;
    numbers: boolean;
    symbols: boolean;
    length: number;
  };
  suggestions: string[];
  score: number;
}

interface Client {
  client_id: string;
  client_name: string;
}

export default function SecretStrengthPage() {
  const t = useTranslations();

  const [clients, setClients] = useState<Client[]>([]);
  const [selectedId, setSelectedId] = useState("");
  const [reports, setReports] = useState<SecretReport[]>([]);
  const [loading, setLoading] = useState(false);

  const fetchClients = useCallback(async () => {
    try {
      const res = await fetch("/api/v1/oauth/clients", { headers: { "Authorization": `Bearer ${localStorage.getItem("ggid_access_token") || ""}`, "X-Tenant-ID": "00000000-0000-0000-0000-000000000001" } });
      if (res.ok) {
        const data = await res.json();
        setClients(data.clients || data || []);
      }
    } catch { /* noop */ }
  }, []);

  useEffect(() => { fetchClients(); }, [fetchClients]);

  const fetchReports = useCallback(async () => {
    if (!selectedId) return;
    setLoading(true);
    try {
      const res = await fetch(`/api/v1/oauth/clients/${selectedId}/secret-strength`, { headers: { "Authorization": `Bearer ${localStorage.getItem("ggid_access_token") || ""}`, "X-Tenant-ID": "00000000-0000-0000-0000-000000000001" } });
      if (res.ok) {
        const data = await res.json();
        setReports(Array.isArray(data) ? data : [data]);
      }
    } catch { /* noop */ }
    finally { setLoading(false); }
  }, [selectedId]);

  useEffect(() => {
    if (selectedId) fetchReports();
  }, [selectedId, fetchReports]);

  const scoreColor = (score: number) => score >= 80 ? "text-green-600" : score >= 50 ? "text-yellow-600" : "text-red-600";
  const entropyColor = (bits: number) => bits >= 80 ? "#10b981" : bits >= 50 ? "#f59e0b" : "#ef4444";

  return (
    <div className="space-y-6">
      <div>
        <h1 className="text-2xl font-bold flex items-center gap-2"><KeyRound className="w-6 h-6 text-blue-500" /> {t("secretStrength.title")}</h1>
        <p className="text-sm text-gray-500 mt-1">Audit OAuth client secret strength and get improvement suggestions.</p>
      </div>

      {/* Client selector */}
      <select aria-label="Selected id" value={selectedId} onChange={(e) => setSelectedId(e.target.value)} className="px-3 py-2 rounded-lg border dark:border-gray-700 dark:bg-gray-900 text-sm">
        <option value="">Select a client...</option>
        {clients.map((c) => <option key={c.client_id} value={c.client_id}>{c.client_name} ({c.client_id})</option>)}
      </select>

      {loading && <p className="text-sm text-gray-500">Loading...</p>}

      {reports.length > 0 && !loading && (
        <div className="space-y-4">
          {reports.map((r) => (
            <div key={r.secret_id} className="rounded-lg border dark:border-gray-800 p-4 space-y-4">
              <div className="flex items-center justify-between">
                <div>
                  <h3 className="font-semibold">{r.client_name}</h3>
                  <p className="text-xs text-gray-400 font-mono">{r.secret_id}</p>
                </div>
                <div className="flex items-center gap-2">
                  <span className={`text-2xl font-bold ${scoreColor(r.score)}`}>{r.score}</span>
                  <span className="text-xs text-gray-400">/100</span>
                </div>
              </div>

              <div className="grid grid-cols-1 md:grid-cols-3 gap-4">
                {/* Entropy gauge */}
                <div className="flex flex-col items-center">
                  <span className="text-xs text-gray-500 mb-2">Entropy</span>
                  <div className="relative w-20 h-20">
                    <svg viewBox="0 0 64 64" className="w-full h-full">
                      <circle cx={32} cy={32} r={28} fill="none" stroke="currentColor" strokeWidth={6} className="text-gray-200 dark:text-gray-800" />
                      <circle cx={32} cy={32} r={28} fill="none" stroke={entropyColor(r.entropy_bits)} strokeWidth={6} strokeDasharray={`${Math.min(100, (r.entropy_bits / 128) * 100) * 1.76} 176`} strokeLinecap="round" transform="rotate(-90 32 32)" />
                    </svg>
                    <div className="absolute inset-0 flex flex-col items-center justify-center">
                      <span className="text-lg font-bold" style={{ color: entropyColor(r.entropy_bits) }}>{r.entropy_bits}</span>
                      <span className="text-[10px] text-gray-400">bits</span>
                    </div>
                  </div>
                </div>

                {/* Crack time */}
                <div className="flex flex-col items-center justify-center">
                  <span className="text-xs text-gray-500 mb-2 flex items-center gap-1"><Gauge className="w-3 h-3" /> Crack Time</span>
                  <p className="text-lg font-bold text-center">{r.crack_time}</p>
                </div>

                {/* Character diversity */}
                <div>
                  <span className="text-xs text-gray-500 mb-2 block">Character Diversity ({r.character_diversity.length} chars)</span>
                  <div className="grid grid-cols-2 gap-1">
                    {(["lowercase", "uppercase", "numbers", "symbols"] as const).map((type) => (
                      <div key={type} className="flex items-center gap-1 text-xs">
                        {r.character_diversity[type] ? <CheckCircle2 className="w-3 h-3 text-green-500" /> : <XCircle className="w-3 h-3 text-red-400" />}
                        <span className="capitalize">{type}</span>
                      </div>
                    ))}
                  </div>
                </div>
              </div>

              {/* Suggestions */}
              {r.suggestions.length > 0 && (
                <div className="rounded-lg bg-blue-50 dark:bg-blue-900/20 p-3">
                  <h4 className="text-sm font-medium flex items-center gap-2 mb-2"><Lightbulb className="w-4 h-4 text-blue-500" /> Suggestions</h4>
                  <ul className="space-y-1">
                    {r.suggestions.map((s, i) => <li key={i} className="text-xs text-gray-600 dark:text-gray-400 flex items-start gap-2"><span className="text-blue-400 mt-0.5">•</span> {s}</li>)}
                  </ul>
                </div>
              )}
            </div>
          ))}
        </div>
      )}

      {reports.length === 0 && !loading && selectedId && <p className="text-sm text-gray-500">No secrets found for this client.</p>}
      {!selectedId && <p className="text-sm text-gray-500 text-center py-8">Select a client to view secret strength report.</p>}
    </div>
  );
}
