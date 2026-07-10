"use client";

import { useState, useEffect, useCallback, useRef } from "react";
import { useApi } from "@/lib/api";
import {
  Save,
  Loader2,
  Shield,
  Plus,
  Trash2,
  Download,
  Upload,
  GripVertical,
  Search,
  Check,
  X,
} from "lucide-react";

interface IPRule {
  id: string;
  cidr: string;
  enabled: boolean;
  priority: number;
}

const STORAGE_KEY = "ggid_ip_allowlist";

const defaultRules: IPRule[] = [
  { id: "rule-1", cidr: "0.0.0.0/0", enabled: true, priority: 1 },
];

// Validate CIDR notation (IPv4)
function isValidCIDR(input: string): boolean {
  const m = input.match(/^(\d{1,3})\.(\d{1,3})\.(\d{1,3})\.(\d{1,3})\/(\d{1,2})$/);
  if (!m) return false;
  const octets = [m[1], m[2], m[3], m[4]].map(Number);
  if (octets.some((o) => o > 255)) return false;
  const prefix = Number(m[5]);
  return prefix >= 0 && prefix <= 32;
}

// Validate IPv4 address
function isValidIP(ip: string): boolean {
  const parts = ip.split(".");
  if (parts.length !== 4) return false;
  return parts.every((p) => {
    const n = Number(p);
    return n >= 0 && n <= 255;
  });
}

// Check if IP is within CIDR
function isIPInCIDR(ip: string, cidr: string): boolean {
  const [cidrIP, prefixStr] = cidr.split("/");
  const prefix = parseInt(prefixStr, 10);
  if (isNaN(prefix) || prefix < 0 || prefix > 32) return false;

  const ipParts = ip.split(".").map(Number);
  const cidrParts = cidrIP.split(".").map(Number);
  if (ipParts.length !== 4 || cidrParts.length !== 4) return false;

  const ipNum = (ipParts[0] << 24) | (ipParts[1] << 16) | (ipParts[2] << 8) | ipParts[3];
  const cidrNum = (cidrParts[0] << 24) | (cidrParts[1] << 16) | (cidrParts[2] << 8) | cidrParts[3];
  const mask = prefix === 0 ? 0 : (0xffffffff << (32 - prefix)) >>> 0;

  return (ipNum & mask) === (cidrNum & mask);
}

// Check IP against sorted rules — first match wins
function checkIP(ip: string, rules: IPRule[]): boolean {
  const sorted = [...rules]
    .filter((r) => r.enabled)
    .sort((a, b) => a.priority - b.priority);
  return sorted.some((r) => isIPInCIDR(ip, r.cidr));
}

export default function IPAllowlistPage() {
  const { apiFetch, TENANT_ID } = useApi();
  const [rules, setRules] = useState<IPRule[]>(defaultRules);
  const [allowlistEnabled, setAllowlistEnabled] = useState(true);
  const [newCIDR, setNewCIDR] = useState("");
  const [cidrError, setCidrError] = useState<string | null>(null);
  const [msg, setMsg] = useState<string | null>(null);
  const [saving, setSaving] = useState(false);

  // Test IP
  const [testIP, setTestIP] = useState("");
  const [testResult, setTestResult] = useState<"allowed" | "denied" | null>(null);

  // Drag state
  const dragIndex = useRef<number | null>(null);
  const [dragOverIndex, setDragOverIndex] = useState<number | null>(null);

  // File input ref for import
  const fileInputRef = useRef<HTMLInputElement>(null);

  // Load from localStorage
  useEffect(() => {
    const stored = typeof window !== "undefined" ? localStorage.getItem(STORAGE_KEY) : null;
    if (stored) {
      try {
        const parsed = JSON.parse(stored);
        if (parsed.rules) setRules(parsed.rules);
        if (parsed.allowlist_enabled !== undefined) setAllowlistEnabled(parsed.allowlist_enabled);
      } catch {
        // ignore
      }
    }
  }, []);

  useEffect(() => {
    if (msg) {
      const t = setTimeout(() => setMsg(null), 3000);
      return () => clearTimeout(t);
    }
  }, [msg]);

  const handleAddRule = () => {
    setCidrError(null);
    const cidr = newCIDR.trim();
    if (!cidr) {
      setCidrError("Please enter a CIDR notation");
      return;
    }
    if (!isValidCIDR(cidr)) {
      setCidrError("Invalid CIDR format. Use x.x.x.x/y (e.g., 192.168.1.0/24)");
      return;
    }
    // Check duplicate
    if (rules.some((r) => r.cidr === cidr)) {
      setCidrError("This CIDR rule already exists");
      return;
    }
    setRules((prev) => [
      ...prev,
      { id: `rule-${Date.now()}`, cidr, enabled: true, priority: prev.length + 1 },
    ]);
    setNewCIDR("");
  };

  const handleDeleteRule = (id: string) => {
    setRules((prev) =>
      prev.filter((r) => r.id !== id).map((r, i) => ({ ...r, priority: i + 1 })),
    );
    setMsg("Rule deleted");
  };

  const handleToggleRule = (id: string) => {
    setRules((prev) => prev.map((r) => (r.id === id ? { ...r, enabled: !r.enabled } : r)));
  };

  // Drag handlers
  const handleDragStart = (index: number) => {
    dragIndex.current = index;
  };

  const handleDragOver = (e: React.DragEvent, index: number) => {
    e.preventDefault();
    setDragOverIndex(index);
  };

  const handleDrop = (index: number) => {
    const from = dragIndex.current;
    if (from === null || from === index) {
      dragIndex.current = null;
      setDragOverIndex(null);
      return;
    }
    const newRules = [...rules];
    const [moved] = newRules.splice(from, 1);
    newRules.splice(index, 0, moved);
    // Reassign priorities
    const reprioritized = newRules.map((r, i) => ({ ...r, priority: i + 1 }));
    setRules(reprioritized);
    dragIndex.current = null;
    setDragOverIndex(null);
  };

  const handleDragEnd = () => {
    dragIndex.current = null;
    setDragOverIndex(null);
  };

  // Test IP
  const handleTestIP = () => {
    setTestResult(null);
    if (!isValidIP(testIP.trim())) {
      setCidrError("Invalid IP address format");
      return;
    }
    setCidrError(null);
    if (!allowlistEnabled) {
      setTestResult("allowed");
      return;
    }
    setTestResult(checkIP(testIP.trim(), rules) ? "allowed" : "denied");
  };

  // Export
  const handleExport = () => {
    const data = JSON.stringify(
      { allowlist_enabled: allowlistEnabled, rules },
      null,
      2,
    );
    const blob = new Blob([data], { type: "application/json" });
    const url = URL.createObjectURL(blob);
    const a = document.createElement("a");
    a.href = url;
    a.download = "ip-allowlist.json";
    a.click();
    URL.revokeObjectURL(url);
  };

  // Import
  const handleImport = (e: React.ChangeEvent<HTMLInputElement>) => {
    const file = e.target.files?.[0];
    if (!file) return;
    const reader = new FileReader();
    reader.onload = () => {
      try {
        const parsed = JSON.parse(reader.result as string);
        if (Array.isArray(parsed.rules)) {
          const validRules = parsed.rules.filter(
            (r: { cidr?: string }) => r.cidr && isValidCIDR(r.cidr),
          );
          if (validRules.length > 0) {
            setRules(validRules.map((r: IPRule, i: number) => ({
              ...r,
              id: r.id || `rule-${Date.now()}-${i}`,
              priority: i + 1,
            })));
            if (parsed.allowlist_enabled !== undefined) {
              setAllowlistEnabled(parsed.allowlist_enabled);
            }
            setMsg(`Imported ${validRules.length} rules`);
          } else {
            setCidrError("No valid CIDR rules found in file");
          }
        } else {
          setCidrError("Invalid file format: missing rules array");
        }
      } catch {
        setCidrError("Failed to parse JSON file");
      }
    };
    reader.readAsText(file);
    if (fileInputRef.current) fileInputRef.current.value = "";
  };

  const handleSave = async () => {
    setSaving(true);
    try {
      await apiFetch(`/api/v1/tenants/${TENANT_ID}/ip-allowlist`, {
        method: "PUT",
        body: JSON.stringify({ allowlist_enabled: allowlistEnabled, rules }),
      });
      setMsg("IP allowlist saved to server");
    } catch {
      localStorage.setItem(
        STORAGE_KEY,
        JSON.stringify({ allowlist_enabled: allowlistEnabled, rules }),
      );
      setMsg("Endpoint unavailable — saved to localStorage");
    } finally {
      setSaving(false);
    }
  };

  return (
    <div className="max-w-3xl">
      <div className="mb-6 flex items-center justify-between">
        <div className="flex items-center gap-3">
          <Shield className="h-7 w-7 text-brand-600" />
          <div>
            <h1 className="text-2xl font-bold dark:text-gray-100">IP Allowlist</h1>
            <p className="text-sm text-gray-500">
              Restrict access to specific IP ranges using CIDR rules
            </p>
          </div>
        </div>
        {/* Master toggle */}
        <label className="flex cursor-pointer items-center gap-3">
          <span className="text-sm font-medium text-gray-600 dark:text-gray-400">
            {allowlistEnabled ? "Enabled" : "Disabled"}
          </span>
          <button
            type="button"
            onClick={() => setAllowlistEnabled(!allowlistEnabled)}
            className={`relative inline-flex h-6 w-11 shrink-0 items-center rounded-full transition-colors ${
              allowlistEnabled ? "bg-brand-600" : "bg-gray-300 dark:bg-gray-600"
            }`}
          >
            <span
              className={`inline-block h-4 w-4 transform rounded-full bg-white transition-transform ${
                allowlistEnabled ? "translate-x-6" : "translate-x-1"
              }`}
            />
          </button>
        </label>
      </div>

      {msg && (
        <div className="mb-4 flex items-center gap-2 rounded-lg border border-green-200 bg-green-50 p-3 text-sm text-green-700 dark:border-green-800 dark:bg-green-950 dark:text-green-400">
          <Check className="h-4 w-4" /> {msg}
        </div>
      )}
      {cidrError && (
        <div className="mb-4 flex items-center gap-2 rounded-lg border border-red-200 bg-red-50 p-3 text-sm text-red-700 dark:border-red-800 dark:bg-red-950 dark:text-red-400">
          <X className="h-4 w-4" /> {cidrError}
        </div>
      )}

      <div className="space-y-6">
        {/* Add new CIDR */}
        <div className="rounded-xl border border-gray-200 bg-white p-6 shadow-sm dark:border-gray-700 dark:bg-gray-800">
          <h2 className="mb-3 text-sm font-semibold text-gray-700 dark:text-gray-300">
            Add CIDR Rule
          </h2>
          <div className="flex gap-2">
            <input
              type="text"
              value={newCIDR}
              onChange={(e) => {
                setNewCIDR(e.target.value);
                setCidrError(null);
              }}
              onKeyDown={(e) => e.key === "Enter" && handleAddRule()}
              placeholder="e.g., 192.168.1.0/24"
              className="flex-1 rounded-lg border border-gray-300 px-3 py-2 text-sm focus:border-brand-500 focus:outline-none focus:ring-1 focus:ring-brand-500 dark:border-gray-600 dark:bg-gray-900 dark:text-gray-100"
            />
            <button
              onClick={handleAddRule}
              className="flex items-center gap-1.5 rounded-lg bg-brand-600 px-4 py-2 text-sm font-medium text-white hover:bg-brand-700"
            >
              <Plus className="h-4 w-4" /> Add
            </button>
          </div>
          <p className="mt-2 text-xs text-gray-400">
            Format: x.x.x.x/y — prefix length 0-32 (use 0.0.0.0/0 to allow all)
          </p>
        </div>

        {/* Rules list */}
        <div className="rounded-xl border border-gray-200 bg-white p-6 shadow-sm dark:border-gray-700 dark:bg-gray-800">
          <div className="mb-4 flex items-center justify-between">
            <h2 className="text-sm font-semibold text-gray-700 dark:text-gray-300">
              CIDR Rules ({rules.length})
            </h2>
            <p className="text-xs text-gray-400">Drag to reorder priority (first match wins)</p>
          </div>

          {rules.length === 0 ? (
            <div className="py-8 text-center text-sm text-gray-400">
              No rules yet. Add a CIDR rule above to get started.
            </div>
          ) : (
            <div className="space-y-2">
              {rules.map((rule, index) => (
                <div
                  key={rule.id}
                  draggable
                  onDragStart={() => handleDragStart(index)}
                  onDragOver={(e) => handleDragOver(e, index)}
                  onDrop={() => handleDrop(index)}
                  onDragEnd={handleDragEnd}
                  className={`flex items-center gap-3 rounded-lg border p-3 transition-all ${
                    dragOverIndex === index
                      ? "border-brand-400 bg-brand-50 dark:bg-brand-950"
                      : "border-gray-200 bg-gray-50 dark:border-gray-700 dark:bg-gray-900"
                  } ${!rule.enabled ? "opacity-50" : ""}`}
                >
                  <GripVertical className="h-4 w-4 cursor-grab text-gray-400" />

                  {/* Priority badge */}
                  <span className="flex h-6 w-6 items-center justify-center rounded-full bg-brand-100 text-xs font-semibold text-brand-700 dark:bg-brand-900 dark:text-brand-300">
                    {index + 1}
                  </span>

                  {/* CIDR */}
                  <code className="flex-1 text-sm font-mono text-gray-700 dark:text-gray-300">
                    {rule.cidr}
                  </code>

                  {/* Toggle */}
                  <button
                    type="button"
                    onClick={() => handleToggleRule(rule.id)}
                    className={`relative inline-flex h-5 w-9 shrink-0 items-center rounded-full transition-colors ${
                      rule.enabled ? "bg-green-500" : "bg-gray-300 dark:bg-gray-600"
                    }`}
                  >
                    <span
                      className={`inline-block h-3 w-3 transform rounded-full bg-white transition-transform ${
                        rule.enabled ? "translate-x-5" : "translate-x-1"
                      }`}
                    />
                  </button>

                  {/* Delete */}
                  <button
                    onClick={() => handleDeleteRule(rule.id)}
                    className="text-gray-400 hover:text-red-500"
                    title="Delete rule"
                  >
                    <Trash2 className="h-4 w-4" />
                  </button>
                </div>
              ))}
            </div>
          )}

          {/* Export / Import */}
          <div className="mt-4 flex gap-2 border-t border-gray-100 pt-4 dark:border-gray-700">
            <button
              onClick={handleExport}
              className="flex items-center gap-1.5 rounded-lg border border-gray-300 px-3 py-1.5 text-sm text-gray-600 hover:bg-gray-50 dark:border-gray-600 dark:text-gray-400 dark:hover:bg-gray-700"
            >
              <Download className="h-4 w-4" /> Export
            </button>
            <button
              onClick={() => fileInputRef.current?.click()}
              className="flex items-center gap-1.5 rounded-lg border border-gray-300 px-3 py-1.5 text-sm text-gray-600 hover:bg-gray-50 dark:border-gray-600 dark:text-gray-400 dark:hover:bg-gray-700"
            >
              <Upload className="h-4 w-4" /> Import
            </button>
            <input
              ref={fileInputRef}
              type="file"
              accept="application/json"
              onChange={handleImport}
              className="hidden"
            />
          </div>
        </div>

        {/* Test IP */}
        <div className="rounded-xl border border-gray-200 bg-white p-6 shadow-sm dark:border-gray-700 dark:bg-gray-800">
          <h2 className="mb-3 flex items-center gap-2 text-sm font-semibold text-gray-700 dark:text-gray-300">
            <Search className="h-4 w-4 text-brand-600" /> Test IP Address
          </h2>
          <p className="mb-3 text-xs text-gray-400">
            Check whether a specific IP address would be allowed or denied by the current rules.
          </p>
          <div className="flex items-center gap-2">
            <input
              type="text"
              value={testIP}
              onChange={(e) => {
                setTestIP(e.target.value);
                setTestResult(null);
              }}
              onKeyDown={(e) => e.key === "Enter" && handleTestIP()}
              placeholder="e.g., 192.168.1.100"
              className="flex-1 rounded-lg border border-gray-300 px-3 py-2 text-sm focus:border-brand-500 focus:outline-none focus:ring-1 focus:ring-brand-500 dark:border-gray-600 dark:bg-gray-900 dark:text-gray-100"
            />
            <button
              onClick={handleTestIP}
              className="rounded-lg border border-brand-600 px-4 py-2 text-sm font-medium text-brand-600 hover:bg-brand-50 dark:hover:bg-brand-950"
            >
              Test
            </button>
          </div>
          {testResult && (
            <div
              className={`mt-3 flex items-center gap-2 rounded-lg p-3 text-sm font-medium ${
                testResult === "allowed"
                  ? "border border-green-200 bg-green-50 text-green-700 dark:border-green-800 dark:bg-green-950 dark:text-green-400"
                  : "border border-red-200 bg-red-50 text-red-700 dark:border-red-800 dark:bg-red-950 dark:text-red-400"
              }`}
            >
              {testResult === "allowed" ? (
                <>
                  <Check className="h-4 w-4" /> Allowed — IP matches an allowlist rule
                </>
              ) : (
                <>
                  <X className="h-4 w-4" /> Denied — IP does not match any rule
                </>
              )}
            </div>
          )}
        </div>

        {/* Save */}
        <div className="flex justify-end">
          <button
            onClick={handleSave}
            disabled={saving}
            className="flex items-center gap-1.5 rounded-lg bg-brand-600 px-6 py-2.5 text-sm font-medium text-white hover:bg-brand-700 disabled:opacity-50"
          >
            {saving ? <Loader2 className="h-4 w-4 animate-spin" /> : <Save className="h-4 w-4" />}
            Save Allowlist
          </button>
        </div>
      </div>
    </div>
  );
}
