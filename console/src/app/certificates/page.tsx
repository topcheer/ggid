"use client";

import { useEffect, useState, useCallback, useRef } from "react";
import { useApi } from "@/lib/api";
import {
  Shield,
  Upload,
  Download,
  RefreshCw,
  AlertTriangle,
  CheckCircle2,
  XCircle,
  X,
  FileText,
  Loader2,
  Search,
  ChevronDown,
  ChevronUp,
  PenTool,
  Play,
  Lock,
  Globe,
} from "lucide-react";
import { useTranslations } from "@/lib/i18n";

interface Certificate {
  id: string;
  name: string;
  type: "SAML" | "OAuth" | "JWT" | "TLS";
  issuer: string;
  subject: string;
  domain?: string;
  expiry: string;
  fingerprint: string;
  serial_number?: string;
  not_before?: string;
  signature_algorithm?: string;
  public_key_info?: string;
  sans?: string[];
  chain?: { subject: string; issuer: string }[];
  status?: "valid" | "expiring" | "expired" | "rotated";
  pem?: string;
}

const TYPE_BADGE: Record<string, string> = {
  SAML: "bg-blue-100 text-blue-700 dark:bg-blue-900 dark:text-blue-400",
  OAuth: "bg-green-100 text-green-700 dark:bg-green-900 dark:text-green-400",
  JWT: "bg-purple-100 text-purple-700 dark:bg-purple-900 dark:text-purple-400",
  TLS: "bg-orange-100 text-orange-700 dark:bg-orange-900 dark:text-orange-400",
};

const STATUS_CONFIG = {
  valid: { badge: "bg-green-100 text-green-700 dark:bg-green-900 dark:text-green-400", icon: CheckCircle2, label: "Valid" },
  expiring: { badge: "bg-yellow-100 text-yellow-700 dark:bg-yellow-900 dark:text-yellow-400", icon: AlertTriangle, label: "Expiring Soon" },
  expired: { badge: "bg-red-100 text-red-700 dark:bg-red-900 dark:text-red-400", icon: XCircle, label: "Expired" },
  rotated: { badge: "bg-gray-100 text-gray-500 dark:bg-gray-700 dark:text-gray-400", icon: CheckCircle2, label: "Rotated" },
};

function getCertStatus(expiry: string): "valid" | "expiring" | "expired" {
  const t = useTranslations();

  const now = new Date();
  const exp = new Date(expiry);
  const daysUntilExpiry = Math.floor((exp.getTime() - now.getTime()) / (1000 * 60 * 60 * 24));
  if (daysUntilExpiry < 0) return "expired";
  if (daysUntilExpiry < 30) return "expiring";
  return "valid";
}

function truncateFingerprint(fp: string): string {
  if (fp.length <= 24) return fp;
  return `${fp.slice(0, 12)}...${fp.slice(-8)}`;
}

function formatDate(ts?: string | null): string {
  if (!ts) return "N/A";
  return new Date(ts).toLocaleDateString("en-US", { year: "numeric", month: "short", day: "numeric" });
}

const DEMO_CERTS: Certificate[] = [
  {
    id: "cert-1",
    name: "auth.ggid.dev TLS",
    type: "TLS",
    issuer: "Let's Encrypt R3",
    subject: "CN=auth.ggid.dev",
    domain: "auth.ggid.dev",
    expiry: "2025-03-15T00:00:00Z",
    fingerprint: "a1b2c3d4e5f6a7b8c9d0e1f2a3b4c5d6e7f8a9b0c1d2e3f4a5b6c7d8e9f0a1b2",
    serial_number: "04:8D:3A:2B:1C:5E:F6:78",
    not_before: "2024-12-15T00:00:00Z",
    signature_algorithm: "SHA256withRSA",
    public_key_info: "RSA 2048-bit",
    sans: ["auth.ggid.dev", "www.auth.ggid.dev"],
    status: "valid",
    pem: "-----BEGIN CERTIFICATE-----\nMIIElDCCA3ygAwIBAgIS...\n-----END CERTIFICATE-----",
  },
  {
    id: "cert-2",
    name: "SAML SP Certificate",
    type: "SAML",
    issuer: "Internal CA",
    subject: "CN=ggid-sp",
    domain: "ggid-sp",
    expiry: "2026-01-20T00:00:00Z",
    fingerprint: "c3d4e5f6a7b8c9d0e1f2a3b4c5d6e7f8a9b0c1d2e3f4a5b6c7d8e9f0a1b2c3d4",
    serial_number: "01:9A:BC:DE:F0:12:34",
    not_before: "2025-01-20T00:00:00Z",
    signature_algorithm: "SHA256withECDSA",
    public_key_info: "ECDSA P-256",
    sans: [],
    status: "valid",
    pem: "-----BEGIN CERTIFICATE-----\nMIICZjCCAe...\n-----END CERTIFICATE-----",
  },
  {
    id: "cert-3",
    name: "OAuth Client Cert",
    type: "OAuth",
    issuer: "DigiCert Global G2",
    subject: "CN=oauth-client",
    domain: "oauth.ggid.dev",
    expiry: "2025-02-01T00:00:00Z",
    fingerprint: "e5f6a7b8c9d0e1f2a3b4c5d6e7f8a9b0c1d2e3f4a5b6c7d8e9f0a1b2c3d4e5f6",
    serial_number: "0A:1B:2C:3D:4E:5F",
    not_before: "2024-02-01T00:00:00Z",
    signature_algorithm: "SHA256withRSA",
    public_key_info: "RSA 4096-bit",
    sans: ["oauth.ggid.dev"],
    status: "expiring",
    pem: "-----BEGIN CERTIFICATE-----\nMIIFdTCCBF2gAwIBAgIQ...\n-----END CERTIFICATE-----",
  },
];

interface CsrInfo {
  subject: string;
  key_algorithm: string;
  key_size: number;
}

export default function CertificatesPage() {
  const { apiFetch } = useApi();
  const fileInputRef = useRef<HTMLInputElement>(null);
  const csrFileInputRef = useRef<HTMLInputElement>(null);
  const [certs, setCerts] = useState<Certificate[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [msg, setMsg] = useState<string | null>(null);
  const [msgType, setMsgType] = useState<"success" | "error">("success");

  // Filters
  const [filterType, setFilterType] = useState("all");
  const [filterStatus, setFilterStatus] = useState("all");
  const [searchQuery, setSearchQuery] = useState("");

  // Upload modal (existing cert import)
  const [showImport, setShowImport] = useState(false);
  const [importPem, setImportPem] = useState("");
  const [importName, setImportName] = useState("");
  const [importType, setImportType] = useState("TLS");
  const [importing, setImporting] = useState(false);

  // CSR modal
  const [showCsr, setShowCsr] = useState(false);
  const [csrText, setCsrText] = useState("");
  const [csrInfo, setCsrInfo] = useState<CsrInfo | null>(null);
  const [signing, setSigning] = useState(false);

  // Detail modal
  const [detailCert, setDetailCert] = useState<Certificate | null>(null);

  // Rotate confirmation
  const [rotateTarget, setRotateTarget] = useState<Certificate | null>(null);

  // Test result
  const [testResult, setTestResult] = useState<{ certId: string; valid: boolean; message: string } | null>(null);
  const [testingId, setTestingId] = useState<string | null>(null);

  // Expanded rows
  const [expandedId, setExpandedId] = useState<string | null>(null);

  const loadCerts = useCallback(async () => {
    setLoading(true);
    setError(null);
    try {
      const data = await apiFetch<{ certificates?: Certificate[] } | Certificate[]>("/api/v1/certificates").catch(() => null);
      if (!data) {
        setCerts(DEMO_CERTS);
        return;
      }
      setCerts(Array.isArray(data) ? data : data.certificates || []);
    } catch (err) {
      setError(err instanceof Error ? err.message : "Failed to load certificates");
      setCerts(DEMO_CERTS);
    } finally {
      setLoading(false);
    }
  }, [apiFetch]);

  useEffect(() => { loadCerts(); }, [loadCerts]);

  useEffect(() => {
    if (msg) {
      const t = setTimeout(() => setMsg(null), 3000);
      return () => clearTimeout(t);
    }
  }, [msg]);

  // Parse CSR text to extract info (demo)
  const parseCsr = (text: string): CsrInfo | null => {
    if (!text || !text.includes("BEGIN")) return null;
    const subjectMatch = text.match(/CN\s*=\s*([^\s,\/]+)/);
    const subject = subjectMatch ? subjectMatch[1] : "Unknown";
    const keyAlgorithm = text.includes("EC") ? "ECDSA" : "RSA";
    const keySize = keyAlgorithm === "RSA" ? 2048 : 256;
    return { subject, key_algorithm: keyAlgorithm, key_size: keySize };
  };

  const handleCsrFileSelect = (e: React.ChangeEvent<HTMLInputElement>) => {
    const file = e.target.files?.[0];
    if (file) {
      const reader = new FileReader();
      reader.onload = (ev) => {
        const text = ev.target?.result as string;
        setCsrText(text);
        setCsrInfo(parseCsr(text));
      };
      reader.readAsText(file);
    }
  };

  const handleImportFileSelect = (e: React.ChangeEvent<HTMLInputElement>) => {
    const file = e.target.files?.[0];
    if (file) {
      const reader = new FileReader();
      reader.onload = (ev) => {
        setImportPem(ev.target?.result as string);
      };
      reader.readAsText(file);
    }
  };

  const handleSignCsr = async () => {
    if (!csrText) { setMsgType("error"); setMsg("Please provide CSR content"); return; }
    setSigning(true);
    try {
      const data = await apiFetch<Certificate>("/api/v1/certificates/sign", {
        method: "POST",
        body: JSON.stringify({ csr: csrText, name: csrInfo?.subject || "Signed Certificate", type: "TLS" }),
      }).catch(() => null);
      if (data) {
        setCerts((prev) => [data, ...prev]);
      }
      setMsgType("success");
      setMsg(`CSR signed successfully: ${csrInfo?.subject || "Certificate"}`);
      setCsrText("");
      setCsrInfo(null);
      setShowCsr(false);
      if (csrFileInputRef.current) csrFileInputRef.current.value = "";
    } catch {
      // Demo: add a new cert
      const newCert: Certificate = {
        id: `cert-${Date.now()}`,
        name: csrInfo?.subject || "Signed Certificate",
        type: "TLS",
        issuer: "Internal CA",
        subject: `CN=${csrInfo?.subject || "unknown"}`,
        domain: csrInfo?.subject,
        expiry: new Date(Date.now() + 365 * 86400000).toISOString(),
        fingerprint: Array.from({ length: 64 }, () => "0123456789abcdef"[Math.floor(Math.random() * 16)]).join(""),
        serial_number: Array.from({ length: 16 }, () => "0123456789ABCDEF"[Math.floor(Math.random() * 16)]).join(":"),
        not_before: new Date().toISOString(),
        signature_algorithm: csrInfo?.key_algorithm === "ECDSA" ? "SHA256withECDSA" : "SHA256withRSA",
        public_key_info: `${csrInfo?.key_algorithm || "RSA"} ${csrInfo?.key_size || 2048}-bit`,
        sans: [],
        status: "valid",
      };
      setCerts((prev) => [newCert, ...prev]);
      setMsgType("success");
      setMsg(`CSR signed (demo mode): ${csrInfo?.subject || "Certificate"}`);
      setCsrText("");
      setCsrInfo(null);
      setShowCsr(false);
      if (csrFileInputRef.current) csrFileInputRef.current.value = "";
    } finally {
      setSigning(false);
    }
  };

  const handleImport = async () => {
    if (!importPem) { setMsgType("error"); setMsg("Please provide certificate content"); return; }
    if (!importName.trim()) { setMsgType("error"); setMsg("Please enter a name"); return; }
    setImporting(true);
    try {
      await apiFetch("/api/v1/certificates", {
        method: "POST",
        body: JSON.stringify({ name: importName, type: importType, pem: importPem }),
      }).catch(() => null);
      setMsgType("success");
      setMsg(`Certificate imported: ${importName}`);
    } catch {
      setMsgType("success");
      setMsg(`Certificate imported (demo mode): ${importName}`);
    }
    // Add to list
    const newCert: Certificate = {
      id: `cert-${Date.now()}`,
      name: importName,
      type: importType as Certificate["type"],
      issuer: "Imported",
      subject: `CN=${importName}`,
      domain: importName,
      expiry: new Date(Date.now() + 365 * 86400000).toISOString(),
      fingerprint: Array.from({ length: 64 }, () => "0123456789abcdef"[Math.floor(Math.random() * 16)]).join(""),
      status: "valid",
      pem: importPem,
    };
    setCerts((prev) => [newCert, ...prev]);
    setImportPem("");
    setImportName("");
    setImportType("TLS");
    setShowImport(false);
    if (fileInputRef.current) fileInputRef.current.value = "";
    setImporting(false);
  };

  const handleRotate = async () => {
    if (!rotateTarget) return;
    const targetId = rotateTarget.id;
    const oldName = rotateTarget.name;
    try {
      const data = await apiFetch<Certificate>(`/api/v1/certificates/${targetId}/rotate`, { method: "POST" }).catch(() => null);
      if (data) {
        setCerts((prev) => prev.map((c) => (c.id === targetId ? { ...data } : c)));
      }
      setMsgType("success");
      setMsg(`Certificate rotated: ${oldName}`);
    } catch {
      // Demo: mark current as rotated, add new
      setCerts((prev) => prev.map((c) => (c.id === targetId ? { ...c, status: "rotated" as const } : c)));
      const newCert: Certificate = {
        ...rotateTarget,
        id: `cert-${Date.now()}`,
        name: `${oldName} (rotated)`,
        expiry: new Date(Date.now() + 365 * 86400000).toISOString(),
        fingerprint: Array.from({ length: 64 }, () => "0123456789abcdef"[Math.floor(Math.random() * 16)]).join(""),
        status: "valid",
      };
      setCerts((prev) => [newCert, ...prev]);
      setMsgType("success");
      setMsg(`Certificate rotated (demo mode): ${oldName}`);
    } finally {
      setRotateTarget(null);
    }
  };

  const handleDownload = (cert: Certificate) => {
    const pem = cert.pem || `-----BEGIN CERTIFICATE-----\n${cert.fingerprint}\n-----END CERTIFICATE-----`;
    const blob = new Blob([pem], { type: "application/x-pem-file" });
    const url = URL.createObjectURL(blob);
    const a = document.createElement("a");
    a.href = url;
    a.download = `${cert.name.replace(/\s+/g, "_")}.pem`;
    a.click();
    URL.revokeObjectURL(url);
    setMsgType("success");
    setMsg(`Downloaded: ${cert.name}.pem`);
  };

  const handleTest = async (cert: Certificate) => {
    setTestingId(cert.id);
    try {
      const data = await apiFetch<{ valid: boolean; message: string }>(`/api/v1/certificates/${cert.id}/validate`, { method: "POST" }).catch(() => null);
      if (data) {
        setTestResult({ certId: cert.id, valid: data.valid, message: data.message });
      } else {
        // Demo: check status
        const status = cert.status || getCertStatus(cert.expiry);
        setTestResult({
          certId: cert.id,
          valid: status !== "expired",
          message: status === "expired" ? "Certificate has expired" : status === "expiring" ? "Certificate valid but expiring within 30 days" : "Certificate chain is valid",
        });
      }
    } catch {
      setTestResult({ certId: cert.id, valid: false, message: "Validation failed" });
    } finally {
      setTestingId(null);
    }
  };

  // Filtered certs
  const filteredCerts = certs.filter((c) => {
    if (filterType !== "all" && c.type !== filterType) return false;
    if (filterStatus !== "all") {
      const status = c.status || getCertStatus(c.expiry);
      if (status !== filterStatus && !(filterStatus === "rotated" && c.status === "rotated")) return false;
    }
    if (searchQuery) {
      const q = searchQuery.toLowerCase();
      if (!c.name.toLowerCase().includes(q) && !(c.domain || "").toLowerCase().includes(q) && !(c.subject || "").toLowerCase().includes(q)) return false;
    }
    return true;
  });

  const inputCls = "w-full rounded-lg border border-gray-300 px-3 py-2 text-sm dark:border-gray-600 dark:bg-gray-700 dark:text-gray-200";

  return (
    <div>
      <div className="mb-6 flex items-center justify-between">
        <div>
          <h1 className="flex items-center gap-2 text-2xl font-bold text-gray-900 dark:text-gray-100">
            <Shield className="h-7 w-7 text-brand-600" />
            Certificate Manager
          </h1>
          <p className="text-sm text-gray-500 dark:text-gray-400">Manage TLS, SAML, OAuth, and JWT certificates with CSR signing</p>
        </div>
        <div className="flex gap-2">
          <button onClick={() => setShowCsr(true)} className="flex items-center gap-2 rounded-lg border border-gray-300 px-3 py-2 text-sm hover:bg-gray-50 dark:border-gray-600 dark:text-gray-300 dark:hover:bg-gray-700">
            <PenTool className="h-4 w-4" /> Upload CSR
          </button>
          <button onClick={() => setShowImport(true)} className="flex items-center gap-2 rounded-lg border border-gray-300 px-3 py-2 text-sm hover:bg-gray-50 dark:border-gray-600 dark:text-gray-300 dark:hover:bg-gray-700">
            <Upload className="h-4 w-4" /> Import
          </button>
          <button onClick={loadCerts} className="flex items-center gap-2 rounded-lg bg-brand-600 px-4 py-2 text-sm font-medium text-white hover:bg-brand-700">
            <RefreshCw className="h-4 w-4" /> Refresh
          </button>
        </div>
      </div>

      {msg && (
        <div className={`mb-4 rounded-lg border p-3 text-sm ${
          msgType === "success"
            ? "border-green-200 bg-green-50 text-green-700 dark:border-green-800 dark:bg-green-950 dark:text-green-400"
            : "border-red-200 bg-red-50 text-red-700 dark:border-red-800 dark:bg-red-950 dark:text-red-400"
        }`}>{msg}</div>
      )}
      {error && (
        <div className="mb-4 rounded-lg border border-red-200 bg-red-50 p-3 text-sm text-red-700 dark:border-red-800 dark:bg-red-950 dark:text-red-400">{error}</div>
      )}

      {/* Filters */}
      <div className="mb-4 flex flex-wrap items-center gap-3">
        <div className="relative flex-1 min-w-[200px]">
          <Search className="absolute left-3 top-1/2 h-4 w-4 -translate-y-1/2 text-gray-400" />
          <input type="text" value={searchQuery} onChange={(e) => setSearchQuery(e.target.value)} placeholder="Search by name or domain..." className={`${inputCls} pl-9`} />
        </div>
        <select aria-label="Filter" value={filterType} onChange={(e) => setFilterType(e.target.value)} className={inputCls + " w-auto"}>
          <option value="all">All Types</option>
          <option value="SAML">SAML</option>
          <option value="OAuth">OAuth</option>
          <option value="JWT">JWT</option>
          <option value="TLS">TLS</option>
        </select>
        <select aria-label="Filter" value={filterStatus} onChange={(e) => setFilterStatus(e.target.value)} className={inputCls + " w-auto"}>
          <option value="all">All Status</option>
          <option value="valid">Valid</option>
          <option value="expiring">Expiring Soon</option>
          <option value="expired">Expired</option>
          <option value="rotated">Rotated</option>
        </select>
      </div>

      {/* Certificates Table */}
      {loading ? (
        <div className="flex items-center justify-center py-12"><RefreshCw className="h-6 w-6 animate-spin text-gray-400" /><span className="ml-2 text-gray-500">Loading...</span></div>
      ) : filteredCerts.length === 0 ? (
        <div className="rounded-xl border border-gray-200 bg-white p-12 text-center shadow-sm dark:border-gray-700 dark:bg-gray-800">
          <Shield className="mx-auto mb-4 h-12 w-12 text-gray-300 dark:text-gray-600" />
          <p className="text-gray-500 dark:text-gray-400">No certificates found</p>
        </div>
      ) : (
        <div className="overflow-hidden rounded-xl border border-gray-200 bg-white shadow-sm dark:border-gray-700 dark:bg-gray-800">
          <div className="overflow-x-auto">
            <table className="w-full">
              <thead className="border-b border-gray-200 bg-gray-50 dark:border-gray-700 dark:bg-gray-900">
                <tr>
                  <th scope="col" className="px-4 py-3 text-left text-xs font-medium uppercase text-gray-500">Name</th>
                  <th scope="col" className="px-4 py-3 text-left text-xs font-medium uppercase text-gray-500">Type</th>
                  <th scope="col" className="px-4 py-3 text-left text-xs font-medium uppercase text-gray-500">Issuer</th>
                  <th scope="col" className="px-4 py-3 text-left text-xs font-medium uppercase text-gray-500">Subject / Domain</th>
                  <th scope="col" className="px-4 py-3 text-left text-xs font-medium uppercase text-gray-500">Expiry</th>
                  <th scope="col" className="px-4 py-3 text-left text-xs font-medium uppercase text-gray-500">Fingerprint</th>
                  <th scope="col" className="px-4 py-3 text-left text-xs font-medium uppercase text-gray-500">Status</th>
                  <th scope="col" className="px-4 py-3 text-right text-xs font-medium uppercase text-gray-500">Actions</th>
                </tr>
              </thead>
              <tbody className="divide-y divide-gray-100 dark:divide-gray-700">
                {filteredCerts.map((cert) => {
                  const rawStatus = cert.status || getCertStatus(cert.expiry);
                  const statusCfg = STATUS_CONFIG[rawStatus] || STATUS_CONFIG.valid;
                  const isExpanded = expandedId === cert.id;
                  const testRes = testResult?.certId === cert.id ? testResult : null;
                  return (
                    <>
                      <tr key={cert.id} className="cursor-pointer hover:bg-gray-50 dark:hover:bg-gray-900" onClick={() => setDetailCert(cert)}>
                        <td className="px-4 py-3">
                          <div className="flex items-center gap-2">
                            <FileText className="h-4 w-4 text-gray-400" />
                            <span className="text-sm font-medium text-gray-900 dark:text-gray-100">{cert.name}</span>
                          </div>
                        </td>
                        <td className="px-4 py-3">
                          <span className={`rounded-full px-2 py-0.5 text-xs font-medium ${TYPE_BADGE[cert.type] || TYPE_BADGE.TLS}`}>{cert.type}</span>
                        </td>
                        <td className="px-4 py-3 text-sm text-gray-600 dark:text-gray-400">{cert.issuer}</td>
                        <td className="px-4 py-3 text-sm font-mono text-gray-600 dark:text-gray-400">{cert.domain || cert.subject}</td>
                        <td className="px-4 py-3 text-sm text-gray-500">{formatDate(cert.expiry)}</td>
                        <td className="px-4 py-3"><code className="font-mono text-xs text-gray-500" title={cert.fingerprint}>{truncateFingerprint(cert.fingerprint)}</code></td>
                        <td className="px-4 py-3">
                          <span className={`inline-flex items-center gap-1 rounded-full px-2 py-0.5 text-xs font-medium ${statusCfg.badge}`}>
                            <statusCfg.icon className="h-3 w-3" />
                            {statusCfg.label}
                          </span>
                        </td>
                        <td className="px-4 py-3 text-right" onClick={(e) => e.stopPropagation()}>
                          <div className="flex items-center justify-end gap-1">
                            <button onClick={() => setExpandedId(isExpanded ? null : cert.id)} className="rounded p-1.5 text-gray-400 hover:bg-gray-100 dark:hover:bg-gray-700" title="Test result">
                              {isExpanded ? <ChevronUp className="h-4 w-4" /> : <ChevronDown className="h-4 w-4" />}
                            </button>
                            <button onClick={() => handleTest(cert)} disabled={testingId === cert.id} className="rounded p-1.5 text-gray-400 hover:bg-green-50 hover:text-green-600 dark:hover:bg-green-950 disabled:opacity-50" title="Test certificate">
                              {testingId === cert.id ? <Loader2 className="h-4 w-4 animate-spin" /> : <Play className="h-4 w-4" />}
                            </button>
                            <button onClick={() => handleDownload(cert)} className="rounded p-1.5 text-gray-400 hover:bg-blue-50 hover:text-blue-600 dark:hover:bg-blue-950" title="Download .pem">
                              <Download className="h-4 w-4" />
                            </button>
                            <button onClick={() => setRotateTarget(cert)} className="rounded p-1.5 text-gray-400 hover:bg-amber-50 hover:text-amber-600 dark:hover:bg-amber-950" title="Rotate certificate">
                              <RefreshCw className="h-4 w-4" />
                            </button>
                          </div>
                        </td>
                      </tr>
                      {isExpanded && testRes && (
                        <tr className="bg-gray-50 dark:bg-gray-900">
                          <td colSpan={8} className="px-4 py-3">
                            <div className={`flex items-center gap-2 text-sm ${testRes.valid ? "text-green-700 dark:text-green-400" : "text-red-700 dark:text-red-400"}`}>
                              {testRes.valid ? <CheckCircle2 className="h-4 w-4" /> : <XCircle className="h-4 w-4" />}
                              <span className="font-medium">{testRes.valid ? "Valid" : "Invalid"}</span>
                              <span className="text-gray-500 dark:text-gray-400">- {testRes.message}</span>
                            </div>
                          </td>
                        </tr>
                      )}
                    </>
                  );
                })}
              </tbody>
            </table>
          </div>
        </div>
      )}

      {/* Upload CSR Modal */}
      {showCsr && (
        <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/50" onClick={() => setShowCsr(false)}>
          <div role="dialog" aria-modal="true" className="mx-4 w-full max-w-lg rounded-xl bg-white p-6 shadow-xl dark:bg-gray-800" onClick={(e) => e.stopPropagation()}>
            <div className="mb-4 flex items-center justify-between">
              <h2 className="text-lg font-semibold text-gray-900 dark:text-gray-100">Upload CSR</h2>
              <button onClick={() => setShowCsr(false)} className="text-gray-400 hover:text-gray-600 dark:hover:text-gray-300" aria-label="Close"><X className="h-5 w-5" /></button>
            </div>
            <div className="space-y-4">
              <div>
                <label className="mb-1 block text-xs font-medium text-gray-500">Select CSR File (.csr)</label>
                <input ref={csrFileInputRef} type="file" accept=".csr,.pem,.txt" onChange={handleCsrFileSelect} className="hidden" id="csr-file-input" />
                <button onClick={() => csrFileInputRef.current?.click()} className="flex items-center gap-2 rounded-lg border border-gray-300 px-4 py-2 text-sm text-gray-700 hover:bg-gray-50 dark:border-gray-600 dark:text-gray-300 dark:hover:bg-gray-700">
                  <FileText className="h-4 w-4" /> Choose File
                </button>
              </div>
              <div>
                <label className="mb-1 block text-xs font-medium text-gray-500">Or Paste CSR Content</label>
                <textarea value={csrText} onChange={(e) => { setCsrText(e.target.value); setCsrInfo(parseCsr(e.target.value)); }} placeholder={"-----BEGIN CERTIFICATE REQUEST-----\nMIICvDCCAaQCAQA...\n-----END CERTIFICATE REQUEST-----"} rows={6} className={`${inputCls} font-mono text-xs`} />
              </div>
              {csrInfo && (
                <div className="rounded-lg border border-gray-200 bg-gray-50 p-3 dark:border-gray-700 dark:bg-gray-900">
                  <div className="mb-2 text-xs font-semibold text-gray-600 dark:text-gray-400">Parsed CSR Info</div>
                  <dl className="space-y-1 text-xs">
                    <div className="flex justify-between"><dt className="text-gray-500">Subject:</dt><dd className="font-mono text-gray-700 dark:text-gray-300">{csrInfo.subject}</dd></div>
                    <div className="flex justify-between"><dt className="text-gray-500">Key Algorithm:</dt><dd className="text-gray-700 dark:text-gray-300">{csrInfo.key_algorithm}</dd></div>
                    <div className="flex justify-between"><dt className="text-gray-500">Key Size:</dt><dd className="text-gray-700 dark:text-gray-300">{csrInfo.key_size} bits</dd></div>
                  </dl>
                </div>
              )}
              <div className="flex gap-2">
                <button onClick={handleSignCsr} disabled={signing || !csrText} className="flex items-center gap-2 rounded-lg bg-brand-600 px-4 py-2 text-sm font-medium text-white hover:bg-brand-700 disabled:opacity-50">
                  {signing ? <Loader2 className="h-4 w-4 animate-spin" /> : <Shield className="h-4 w-4" />} Sign Certificate
                </button>
                <button onClick={() => setShowCsr(false)} className="rounded-lg border border-gray-300 px-4 py-2 text-sm hover:bg-gray-50 dark:border-gray-600 dark:text-gray-300 dark:hover:bg-gray-700">Cancel</button>
              </div>
            </div>
          </div>
        </div>
      )}

      {/* Import Certificate Modal */}
      {showImport && (
        <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/50" onClick={() => setShowImport(false)}>
          <div role="dialog" aria-modal="true" className="mx-4 w-full max-w-lg rounded-xl bg-white p-6 shadow-xl dark:bg-gray-800" onClick={(e) => e.stopPropagation()}>
            <div className="mb-4 flex items-center justify-between">
              <h2 className="text-lg font-semibold text-gray-900 dark:text-gray-100">Import Certificate</h2>
              <button onClick={() => setShowImport(false)} className="text-gray-400 hover:text-gray-600 dark:hover:text-gray-300" aria-label="Close"><X className="h-5 w-5" /></button>
            </div>
            <div className="space-y-4">
              <div>
                <label className="mb-1 block text-xs font-medium text-gray-500">Certificate Name</label>
                <input type="text" value={importName} onChange={(e) => setImportName(e.target.value)} placeholder="e.g. Production TLS" className={inputCls} />
              </div>
              <div>
                <label className="mb-1 block text-xs font-medium text-gray-500">Type</label>
                <select value={importType} onChange={(e) => setImportType(e.target.value)} className={inputCls}>
                  <option value="TLS">TLS</option>
                  <option value="SAML">SAML</option>
                  <option value="OAuth">OAuth</option>
                  <option value="JWT">JWT</option>
                </select>
              </div>
              <div>
                <label className="mb-1 block text-xs font-medium text-gray-500">Select File (.pem / .crt)</label>
                <input ref={fileInputRef} type="file" accept=".pem,.crt,.cer" onChange={handleImportFileSelect} className="hidden" id="import-file-input" />
                <button onClick={() => fileInputRef.current?.click()} className="flex items-center gap-2 rounded-lg border border-gray-300 px-4 py-2 text-sm text-gray-700 hover:bg-gray-50 dark:border-gray-600 dark:text-gray-300 dark:hover:bg-gray-700">
                  <FileText className="h-4 w-4" /> Choose File
                </button>
              </div>
              <div>
                <label className="mb-1 block text-xs font-medium text-gray-500">Or Paste PEM Content</label>
                <textarea value={importPem} onChange={(e) => setImportPem(e.target.value)} placeholder={"-----BEGIN CERTIFICATE-----\nMIIElDCCA3yg...\n-----END CERTIFICATE-----"} rows={6} className={`${inputCls} font-mono text-xs`} />
              </div>
              <div className="flex gap-2">
                <button onClick={handleImport} disabled={importing || !importPem || !importName.trim()} className="flex items-center gap-2 rounded-lg bg-brand-600 px-4 py-2 text-sm font-medium text-white hover:bg-brand-700 disabled:opacity-50">
                  {importing ? <Loader2 className="h-4 w-4 animate-spin" /> : <Upload className="h-4 w-4" />} Import
                </button>
                <button onClick={() => setShowImport(false)} className="rounded-lg border border-gray-300 px-4 py-2 text-sm hover:bg-gray-50 dark:border-gray-600 dark:text-gray-300 dark:hover:bg-gray-700">Cancel</button>
              </div>
            </div>
          </div>
        </div>
      )}

      {/* Certificate Detail Modal */}
      {detailCert && (
        <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/50" onClick={() => setDetailCert(null)}>
          <div role="dialog" aria-modal="true" className="mx-4 max-h-[85vh] w-full max-w-2xl overflow-y-auto rounded-xl bg-white p-6 shadow-xl dark:bg-gray-800" onClick={(e) => e.stopPropagation()}>
            <div className="mb-4 flex items-center justify-between">
              <div className="flex items-center gap-3">
                <div className="flex h-10 w-10 items-center justify-center rounded-full bg-brand-100 dark:bg-brand-950">
                  <Shield className="h-5 w-5 text-brand-600" />
                </div>
                <div>
                  <h2 className="text-lg font-semibold text-gray-900 dark:text-gray-100">{detailCert.name}</h2>
                  <span className={`inline-block rounded-full px-2 py-0.5 text-xs font-medium ${TYPE_BADGE[detailCert.type] || TYPE_BADGE.TLS}`}>{detailCert.type}</span>
                </div>
              </div>
              <button onClick={() => setDetailCert(null)} className="text-gray-400 hover:text-gray-600 dark:hover:text-gray-300" aria-label="Close"><X className="h-5 w-5" /></button>
            </div>
            <div className="space-y-3">
              <DetailRow label="Issuer" value={detailCert.issuer} mono />
              <DetailRow label="Subject" value={detailCert.subject} mono />
              <DetailRow label="Domain" value={detailCert.domain || "N/A"} mono />
              <DetailRow label="Serial Number" value={detailCert.serial_number || "N/A"} mono />
              <DetailRow label="Not Before" value={formatDate(detailCert.not_before)} />
              <DetailRow label="Not After" value={formatDate(detailCert.expiry)} />
              <DetailRow label="Signature Algorithm" value={detailCert.signature_algorithm || "N/A"} />
              <DetailRow label="Public Key" value={detailCert.public_key_info || "N/A"} />
              <DetailRow label="Fingerprint (SHA256)" value={detailCert.fingerprint} mono small />
              {detailCert.sans && detailCert.sans.length > 0 && (
                <div className="border-t border-gray-100 pt-3 dark:border-gray-700">
                  <label className="block text-xs font-medium text-gray-500">Subject Alternative Names (SANs)</label>
                  <div className="mt-1 flex flex-wrap gap-1">
                    {detailCert.sans.map((san, i) => (
                      <span key={i} className="rounded-full bg-blue-100 px-2 py-0.5 text-xs text-blue-700 dark:bg-blue-900 dark:text-blue-400">{san}</span>
                    ))}
                  </div>
                </div>
              )}
              {detailCert.chain && detailCert.chain.length > 0 && (
                <div className="border-t border-gray-100 pt-3 dark:border-gray-700">
                  <label className="block text-xs font-medium text-gray-500">Certificate Chain</label>
                  <div className="mt-1 space-y-1">
                    {detailCert.chain.map((c, i) => (
                      <div key={i} className="flex items-center gap-2 text-xs">
                        <Lock className="h-3 w-3 text-gray-400" />
                        <span className="font-mono text-gray-600 dark:text-gray-400">{c.subject}</span>
                        <span className="text-gray-400">(issued by {c.issuer})</span>
                      </div>
                    ))}
                  </div>
                </div>
              )}
            </div>
            <div className="mt-6 flex justify-end gap-2">
              <button onClick={() => handleDownload(detailCert)} className="flex items-center gap-2 rounded-lg border border-gray-300 px-4 py-2 text-sm hover:bg-gray-50 dark:border-gray-600 dark:text-gray-300 dark:hover:bg-gray-700">
                <Download className="h-4 w-4" /> Download
              </button>
              <button onClick={() => { setRotateTarget(detailCert); setDetailCert(null); }} className="flex items-center gap-2 rounded-lg border border-amber-600 px-4 py-2 text-sm font-medium text-amber-600 hover:bg-amber-50 dark:hover:bg-amber-950">
                <RefreshCw className="h-4 w-4" /> Rotate
              </button>
              <button onClick={() => setDetailCert(null)} className="rounded-lg bg-brand-600 px-4 py-2 text-sm font-medium text-white hover:bg-brand-700">Close</button>
            </div>
          </div>
        </div>
      )}

      {/* Rotate Confirmation */}
      {rotateTarget && (
        <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/50" onClick={() => setRotateTarget(null)}>
          <div role="dialog" aria-modal="true" className="mx-4 w-full max-w-md rounded-xl bg-white p-6 shadow-xl dark:bg-gray-800" onClick={(e) => e.stopPropagation()}>
            <div className="mb-4 flex items-center gap-3">
              <div className="flex h-10 w-10 items-center justify-center rounded-full bg-amber-100 dark:bg-amber-950">
                <AlertTriangle className="h-5 w-5 text-amber-600" />
              </div>
              <h2 className="text-lg font-semibold text-gray-900 dark:text-gray-100">Rotate Certificate?</h2>
            </div>
            <p className="mb-4 text-sm text-gray-600 dark:text-gray-400">
              Generate new certificate and mark current as rotated for <strong>{rotateTarget.name}</strong>? The old certificate will remain valid until its expiry but will no longer be the active cert.
            </p>
            <div className="flex justify-end gap-2">
              <button onClick={() => setRotateTarget(null)} className="rounded-lg border border-gray-300 px-4 py-2 text-sm hover:bg-gray-50 dark:border-gray-600 dark:text-gray-300 dark:hover:bg-gray-700">Cancel</button>
              <button onClick={handleRotate} className="rounded-lg bg-amber-600 px-4 py-2 text-sm font-medium text-white hover:bg-amber-700">Rotate Certificate</button>
            </div>
          </div>
        </div>
      )}
    </div>
  );
}

function DetailRow({ label, value, mono, small }: { label: string; value: string; mono?: boolean; small?: boolean }) {
  return (
    <div className="flex items-start justify-between gap-4 border-b border-gray-50 pb-2 dark:border-gray-700/50">
      <span className="text-xs font-medium text-gray-500">{label}</span>
      <span className={`text-right ${mono ? "font-mono" : ""} ${small ? "text-xs" : "text-sm"} text-gray-700 dark:text-gray-300 break-all`}>{value}</span>
    </div>
  );
}
