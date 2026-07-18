"use client";
import { useState, useEffect, useCallback } from "react";
import {
  KeyRound, Shield, Copy, Check, RefreshCw, Loader2, AlertTriangle,
  Download, Eye, EyeOff, X,
} from "lucide-react";
import { useTranslations } from "@/lib/i18n";
import { authHeader } from "@/lib/auth-helpers";

const TENANT_ID = "00000000-0000-0000-0000-000000000001";

interface BackupCodesResponse {
  codes: string[];
  generated_at: string;
}

interface RemainingResponse {
  remaining: number;
  total: number;
}

export default function BackupCodesPage() {
  const t = useTranslations();
  const [codes, setCodes] = useState<string[]>([]);
  const [generatedAt, setGeneratedAt] = useState<string | null>(null);
  const [remaining, setRemaining] = useState<number | null>(null);
  const [total, setTotal] = useState<number>(10);
  const [loading, setLoading] = useState(true);
  const [generating, setGenerating] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [saved, setSaved] = useState(false);
  const [showCodes, setShowCodes] = useState(false);
  const [copiedAll, setCopiedAll] = useState(false);
  const [copiedIdx, setCopiedIdx] = useState<number | null>(null);
  const [confirmRegenerate, setConfirmRegenerate] = useState(false);

  const loadRemaining = useCallback(async () => {
    try {
      const res = await fetch("/api/v1/auth/mfa/backup-codes/remaining", {
        headers: { ...authHeader(), "X-Tenant-ID": TENANT_ID },
      });
      if (res.ok) {
        const data: RemainingResponse = await res.json();
        setRemaining(data.remaining);
        setTotal(data.total || 10);
      }
    } catch {
      // Endpoint may not be available yet
    } finally {
      setLoading(false);
    }
  }, []);

  useEffect(() => { loadRemaining(); }, [loadRemaining]);

  const generate = async () => {
    setGenerating(true);
    setError(null);
    setSaved(false);
    try {
      const res = await fetch("/api/v1/auth/mfa/backup-codes/generate", {
        method: "POST",
        headers: { ...authHeader(), "Content-Type": "application/json", "X-Tenant-ID": TENANT_ID },
      });
      if (!res.ok) {
        const body = await res.json().catch(() => ({}));
        throw new Error(body.error || `HTTP ${res.status}`);
      }
      const data: BackupCodesResponse = await res.json();
      setCodes(data.codes || []);
      setGeneratedAt(data.generated_at || new Date().toISOString());
      setShowCodes(true);
      setRemaining(data.codes?.length || 0);
      setConfirmRegenerate(false);
    } catch (e) {
      setError(e instanceof Error ? e.message : "Failed to generate backup codes");
    } finally {
      setGenerating(false);
    }
  };

  const copyCode = async (code: string, idx: number) => {
    try {
      await navigator.clipboard.writeText(code);
      setCopiedIdx(idx);
      setTimeout(() => setCopiedIdx(null), 2000);
    } catch { /* clipboard may not be available */ }
  };

  const copyAll = async () => {
    try {
      const text = codes.map((c: any, i: number) => `${String(i + 1).padStart(2, "0")}. ${c}`).join("\n");
      await navigator.clipboard.writeText(text);
      setCopiedAll(true);
      setTimeout(() => setCopiedAll(false), 3000);
    } catch { /* noop */ }
  };

  const downloadCodes = () => {
    const text = `GGID Backup Codes\nGenerated: ${new Date(generatedAt || Date.now()).toLocaleString()}\n\n${codes.map((c: any, i: number) => `${String(i + 1).padStart(2, "0")}. ${c}`).join("\n")}\n\nStore these codes in a safe place. Each code can only be used once.\n`;
    const blob = new Blob([text], { type: "text/plain" });
    const url = URL.createObjectURL(blob);
    const a = document.createElement("a");
    a.href = url;
    a.download = "ggid-backup-codes.txt";
    a.click();
    URL.revokeObjectURL(url);
    setSaved(true);
    setTimeout(() => setSaved(false), 3000);
  };

  const cardCls = "rounded-xl border border-gray-200 bg-white p-6 shadow-sm dark:border-gray-800 dark:bg-gray-900";

  return (
    <div className="min-h-screen bg-gray-50 p-6 dark:bg-gray-950">
      <div className="mx-auto max-w-3xl space-y-6">
        {/* Header */}
        <div>
          <h1 className="flex items-center gap-2 text-2xl font-bold text-gray-900 dark:text-white">
            <KeyRound className="h-6 w-6 text-indigo-600" />
            MFA Backup Codes
          </h1>
          <p className="mt-1 text-sm text-gray-500 dark:text-gray-400">
            Generate one-time recovery codes for account access when your authenticator device is unavailable.
          </p>
        </div>

        {/* Error */}
        {error && (
          <div role="alert" className="flex items-center gap-2 rounded-lg bg-red-50 px-4 py-3 text-sm text-red-700 dark:bg-red-900/20 dark:text-red-400">
            <AlertTriangle className="h-4 w-4 shrink-0" />{error}
            <button onClick={() => setError(null)} aria-label="Dismiss error" className="ml-auto"><X className="h-4 w-4" /></button>
          </div>
        )}

        {/* Status card */}
        <div className={cardCls}>
          <div className="flex items-center justify-between">
            <div>
              <h2 className="flex items-center gap-2 text-lg font-semibold text-gray-900 dark:text-white">
                <Shield className="h-5 w-5 text-gray-400" /> Code Status
              </h2>
              {loading ? (
                <p className="mt-2 text-sm text-gray-400">Loading...</p>
              ) : remaining !== null ? (
                <div className="mt-2">
                  <p className="text-sm text-gray-600 dark:text-gray-400">
                    <span className="font-bold text-gray-900 dark:text-white">{remaining}</span> of {total} codes remaining
                  </p>
                  <div className="mt-2 h-2 w-full overflow-hidden rounded-full bg-gray-200 dark:bg-gray-700">
                    <div
                      className={"h-full rounded-full transition-all " + (remaining === 0 ? "bg-red-500" : remaining <= 3 ? "bg-amber-500" : "bg-green-500")}
                      style={{ width: `${total > 0 ? (remaining / total) * 100 : 0}%` }}
                    />
                  </div>
                </div>
              ) : (
                <p className="mt-2 text-sm text-gray-400">No backup codes generated.</p>
              )}
            </div>
            <button
              onClick={() => setConfirmRegenerate(true)}
              disabled={generating}
              className="flex items-center gap-2 rounded-lg bg-indigo-600 px-4 py-2 text-sm font-medium text-white hover:bg-indigo-700 disabled:opacity-50"
            >
              {generating ? <Loader2 className="h-4 w-4 animate-spin" /> : <RefreshCw className="h-4 w-4" />}
              {remaining !== null ? "Regenerate" : "Generate Codes"}
            </button>
          </div>
        </div>

        {/* Codes display */}
        {codes.length > 0 && (
          <div className={cardCls}>
            <div className="mb-4 flex items-center justify-between">
              <h2 className="text-lg font-semibold text-gray-900 dark:text-white">Your Backup Codes</h2>
              <div className="flex items-center gap-2">
                <button
                  onClick={() => setShowCodes(!showCodes)}
                  className="flex items-center gap-1 text-xs text-gray-500 hover:text-gray-700 dark:hover:text-gray-300"
                  aria-label={showCodes ? "Hide codes" : "Show codes"}
                >
                  {showCodes ? <EyeOff className="h-3.5 w-3.5" /> : <Eye className="h-3.5 w-3.5" />}
                  {showCodes ? "Hide" : "Reveal"}
                </button>
                <button onClick={copyAll} className="flex items-center gap-1 text-xs text-gray-500 hover:text-gray-700 dark:hover:text-gray-300" aria-label="Copy all codes">
                  {copiedAll ? <Check className="h-3.5 w-3.5 text-green-500" /> : <Copy className="h-3.5 w-3.5" />}
                  Copy All
                </button>
                <button onClick={downloadCodes} className="flex items-center gap-1 text-xs text-gray-500 hover:text-gray-700 dark:hover:text-gray-300" aria-label="Download codes">
                  <Download className="h-3.5 w-3.5" /> Download
                </button>
              </div>
            </div>

            {generatedAt && (
              <p className="mb-3 text-xs text-gray-400">
                Generated: {new Date(generatedAt).toLocaleString()}
              </p>
            )}

            <div className="grid gap-2 sm:grid-cols-2">
              {codes.map((code: any, idx: number) => (
                <div
                  key={idx}
                  className="flex items-center justify-between rounded-lg border border-gray-200 px-3 py-2 dark:border-gray-700"
                >
                  <span className="font-mono text-sm text-gray-700 dark:text-gray-300">
                    {showCodes ? (
                      <span className="flex items-center gap-2">
                        <span className="text-xs text-gray-400">{String(idx + 1).padStart(2, "0")}</span>
                        {code}
                      </span>
                    ) : (
                      <span className="tracking-widest">•••• ••••</span>
                    )}
                  </span>
                  <button
                    onClick={() => copyCode(code, idx)}
                    className="text-gray-400 hover:text-gray-600 dark:hover:text-gray-300"
                    aria-label={`Copy code ${idx + 1}`}
                  >
                    {copiedIdx === idx ? <Check className="h-3.5 w-3.5 text-green-500" /> : <Copy className="h-3.5 w-3.5" />}
                  </button>
                </div>
              ))}
            </div>

            {saved && (
              <p className="mt-3 flex items-center gap-1 text-sm text-green-600 dark:text-green-400">
                <Check className="h-4 w-4" /> Downloaded. Store in a safe location.
              </p>
            )}

            {/* Warning */}
            <div className="mt-4 flex items-start gap-3 rounded-lg bg-amber-50 p-3 dark:bg-amber-950/30">
              <AlertTriangle className="h-5 w-5 shrink-0 text-amber-600" />
              <div>
                <p className="text-sm font-medium text-amber-800 dark:text-amber-400">Important</p>
                <ul className="mt-1 space-y-0.5 text-xs text-amber-700 dark:text-amber-500">
                  <li>Each code can only be used once.</li>
                  <li>Generating new codes invalidates all previous codes.</li>
                  <li>Store these in a secure password manager or print them.</li>
                </ul>
              </div>
            </div>
          </div>
        )}

        {/* Regenerate confirmation */}
        {confirmRegenerate && (
          <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/50" onClick={() => setConfirmRegenerate(false)}>
            <div role="dialog" aria-modal="true" className="mx-4 w-full max-w-md rounded-xl bg-white p-6 shadow-xl dark:bg-gray-800" onClick={e => e.stopPropagation()}>
              <h3 className="flex items-center gap-2 text-lg font-semibold text-gray-900 dark:text-white">
                <AlertTriangle className="h-5 w-5 text-amber-500" /> Regenerate Backup Codes?
              </h3>
              <p className="mt-2 text-sm text-gray-600 dark:text-gray-400">
                This will invalidate all {remaining} remaining codes and generate a new set. This action cannot be undone.
              </p>
              <div className="mt-4 flex justify-end gap-2">
                <button onClick={() => setConfirmRegenerate(false)} className="rounded-lg border border-gray-300 px-4 py-2 text-sm dark:border-gray-700">Cancel</button>
                <button onClick={generate} className="rounded-lg bg-indigo-600 px-4 py-2 text-sm font-medium text-white hover:bg-indigo-700">Regenerate</button>
              </div>
            </div>
          </div>
        )}
      </div>
    </div>
  );
}
