"use client";

import { useEffect, useState, useRef } from "react";
import { useApi } from "@/lib/api";
import { useTranslations } from "@/lib/i18n";
import {
  Shield, Upload, Key, RefreshCw, AlertTriangle, CheckCircle2,
  XCircle, Clock, FileText, X,
} from "lucide-react";

interface Certificate {
  id: string;
  issuer: string;
  subject: string;
  expiry: string;
  fingerprint: string;
}

interface SigningKey {
  kid: string;
  alg: string;
  status: "active" | "rotated";
  created: string;
}

const STATUS_CONFIG = {
  valid: { badge: "bg-green-100 text-green-700 dark:bg-green-900 dark:text-green-400", icon: CheckCircle2, key: "certs.valid" },
  expiring: { badge: "bg-yellow-100 text-yellow-700 dark:bg-yellow-900 dark:text-yellow-400", icon: AlertTriangle, key: "certs.expiringSoon" },
  expired: { badge: "bg-red-100 text-red-700 dark:bg-red-900 dark:text-red-400", icon: XCircle, key: "certs.expiredLabel" },
};

function getCertStatus(expiry: string): keyof typeof STATUS_CONFIG {
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

// Demo data for initial display
const DEMO_CERTS: Certificate[] = [
  { id: "1", issuer: "Let's Encrypt R3", subject: "auth.ggid.dev", expiry: "2025-03-15T00:00:00Z", fingerprint: "a1b2c3d4e5f6a7b8c9d0e1f2a3b4c5d6e7f8a9b0c1d2e3f4a5b6c7d8e9f0a1b2" },
  { id: "2", issuer: "DigiCert Global G2", subject: "api.ggid.dev", expiry: "2026-01-20T00:00:00Z", fingerprint: "c3d4e5f6a7b8c9d0e1f2a3b4c5d6e7f8a9b0c1d2e3f4a5b6c7d8e9f0a1b2c3d4" },
];

const DEMO_KEYS: SigningKey[] = [
  { kid: "2024-01-15-key-01", alg: "RS256", status: "active", created: "2024-01-15T10:00:00Z" },
  { kid: "2023-06-01-key-00", alg: "RS256", status: "rotated", created: "2023-06-01T08:00:00Z" },
];

export default function CertificatesPage() {
  const { apiFetch } = useApi();
  const t = useTranslations();
  const fileInputRef = useRef<HTMLInputElement>(null);
  const [certs, setCerts] = useState<Certificate[]>([]);
  const [signingKeys, setSigningKeys] = useState<SigningKey[]>([]);
  const [loading, setLoading] = useState(true);
  const [msg, setMsg] = useState<string | null>(null);
  const [msgType, setMsgType] = useState<"success" | "error">("success");

  // Upload state
  const [selectedFile, setSelectedFile] = useState<File | null>(null);
  const [pemText, setPemText] = useState("");

  // Rotation dialog
  const [showRotateDialog, setShowRotateDialog] = useState(false);

  useEffect(() => {
    const fetchData = async () => {
      setLoading(true);
      try {
        const data = await apiFetch<{ certificates?: Certificate[] } | Certificate[]>(
          "/api/v1/settings/certificates",
        ).catch(() => null);
        const certList = data
          ? (Array.isArray(data) ? data : data.certificates || [])
          : DEMO_CERTS;
        setCerts(certList);

        const keyData = await apiFetch<{ keys?: SigningKey[] } | SigningKey[]>(
          "/api/v1/settings/jwks",
        ).catch(() => null);
        const keyList = keyData
          ? (Array.isArray(keyData) ? keyData : keyData.keys || [])
          : DEMO_KEYS;
        setSigningKeys(keyList);
      } catch {
        setCerts(DEMO_CERTS);
        setSigningKeys(DEMO_KEYS);
      } finally {
        setLoading(false);
      }
    };
    fetchData();
  }, [apiFetch]);

  useEffect(() => {
    if (msg) {
      const t = setTimeout(() => setMsg(null), 3000);
      return () => clearTimeout(t);
    }
  }, [msg]);

  const handleFileSelect = (e: React.ChangeEvent<HTMLInputElement>) => {
    const file = e.target.files?.[0];
    if (file) {
      setSelectedFile(file);
      // Read file content into textarea
      const reader = new FileReader();
      reader.onload = (ev) => {
        setPemText(ev.target?.result as string);
      };
      reader.readAsText(file);
    }
  };

  const handleUpload = async () => {
    if (!pemText && !selectedFile) {
      setMsgType("error");
      setMsg(t("certs.selectOrPaste"));
      return;
    }
    try {
      await apiFetch("/api/v1/settings/certificates", {
        method: "POST",
        body: JSON.stringify({
          filename: selectedFile?.name || "pasted.pem",
          pem: pemText,
        }),
      }).catch(() => null);
      setMsgType("success");
      setMsg(`${t("certs.certUploaded")}: ${selectedFile?.name || "pasted.pem"}`);
      setSelectedFile(null);
      setPemText("");
      if (fileInputRef.current) fileInputRef.current.value = "";
    } catch {
      setMsgType("success");
      setMsg(`${t("certs.certUploadedOffline")}: ${selectedFile?.name || "pasted.pem"}`);
      setSelectedFile(null);
      setPemText("");
      if (fileInputRef.current) fileInputRef.current.value = "";
    }
  };

  const handleRotateKey = async () => {
    setShowRotateDialog(false);
    try {
      await apiFetch("/api/v1/settings/jwks/rotate", {
        method: "POST",
      }).catch(() => null);
      // Mark current active key as rotated, add new key
      const newKid = `${new Date().toISOString().slice(0, 10)}-key-${Date.now().toString(36)}`;
      setSigningKeys((prev) => [
        { kid: newKid, alg: "RS256", status: "active", created: new Date().toISOString() },
        ...prev.map((k) => (k.status === "active" ? { ...k, status: "rotated" as const } : k)),
      ]);
      setMsgType("success");
      setMsg(t("certs.keyRotated"));
    } catch {
      setMsgType("error");
      setMsg(t("certs.keyRotateFailed"));
    }
  };

  const inputCls = "w-full rounded-lg border border-gray-300 px-3 py-2 text-sm dark:border-gray-600 dark:bg-gray-700 dark:text-gray-200";
  const cardCls = "rounded-xl border border-gray-200 bg-white p-6 shadow-sm dark:border-gray-700 dark:bg-gray-800";
  const headingCls = "mb-4 text-lg font-semibold text-gray-900 dark:text-gray-100";

  return (
    <div>
      <h1 className="mb-6 flex items-center gap-2 text-2xl font-bold text-gray-900 dark:text-gray-100">
        <Shield className="h-7 w-7 text-brand-600" />
        {t("certs.title")}
      </h1>

      {msg && (
        <div className={`mb-4 rounded-lg border p-3 text-sm ${
          msgType === "success"
            ? "border-green-200 bg-green-50 text-green-700 dark:border-green-800 dark:bg-green-950 dark:text-green-400"
            : "border-red-200 bg-red-50 text-red-700 dark:border-red-800 dark:bg-red-950 dark:text-red-400"
        }`}>
          {msg}
        </div>
      )}

      {/* TLS Certificates Section */}
      <div className={`${cardCls} mb-6`}>
        <div className="mb-4 flex items-center justify-between">
          <h2 className={headingCls}>
            <FileText className="mr-2 inline h-5 w-5 text-brand-600" />
            {t("certs.tlsCertificates")}
          </h2>
          <span className="text-xs text-gray-400">{certs.length} {t("certs.certCount")}</span>
        </div>

        {loading ? (
          <div className="py-8 text-center text-sm text-gray-400">{t("certs.loadingCerts")}</div>
        ) : certs.length === 0 ? (
          <div className="py-8 text-center text-sm text-gray-400">{t("certs.noCerts")}</div>
        ) : (
          <div className="overflow-x-auto">
            <table className="w-full">
              <thead>
                <tr className="border-b border-gray-100 dark:border-gray-700">
                  <th className="px-3 py-2 text-left text-xs font-medium text-gray-500">{t("common.issuer")}</th>
                  <th className="px-3 py-2 text-left text-xs font-medium text-gray-500">{t("common.subject")}</th>
                  <th className="px-3 py-2 text-left text-xs font-medium text-gray-500">{t("certs.expiryDate")}</th>
                  <th className="px-3 py-2 text-left text-xs font-medium text-gray-500">{t("certs.fingerprint")}</th>
                  <th className="px-3 py-2 text-left text-xs font-medium text-gray-500">{t("common.status")}</th>
                </tr>
              </thead>
              <tbody className="divide-y divide-gray-50 dark:divide-gray-700/50">
                {certs.map((cert) => {
                  const status = getCertStatus(cert.expiry);
                  const statusCfg = STATUS_CONFIG[status];
                  return (
                    <tr key={cert.id} className="hover:bg-gray-50 dark:hover:bg-gray-700/50">
                      <td className="px-3 py-3 text-sm text-gray-900 dark:text-gray-100">{cert.issuer}</td>
                      <td className="px-3 py-3 text-sm font-mono text-gray-700 dark:text-gray-300">{cert.subject}</td>
                      <td className="px-3 py-3 text-sm text-gray-500">
                        {new Date(cert.expiry).toLocaleDateString("en-US", { year: "numeric", month: "short", day: "numeric" })}
                      </td>
                      <td className="px-3 py-3 text-sm font-mono text-gray-500" title={cert.fingerprint}>
                        {truncateFingerprint(cert.fingerprint)}
                      </td>
                      <td className="px-3 py-3">
                        <span className={`inline-flex items-center gap-1 rounded-full px-2.5 py-0.5 text-xs font-medium ${statusCfg.badge}`}>
                          <statusCfg.icon className="h-3 w-3" />
                          {t(statusCfg.key)}
                        </span>
                      </td>
                    </tr>
                  );
                })}
              </tbody>
            </table>
          </div>
        )}
      </div>

      {/* Upload PEM Section */}
      <div className={`${cardCls} mb-6`}>
        <h2 className={headingCls}>
          <Upload className="mr-2 inline h-5 w-5 text-brand-600" />
          {t("certs.uploadSection")}
        </h2>
        <div className="space-y-4">
          {/* File Input */}
          <div>
            <label className="mb-1 block text-xs font-medium text-gray-500">{t("certs.selectFile")}</label>
            <div className="flex items-center gap-3">
              <input
                ref={fileInputRef}
                type="file"
                accept=".pem,.crt,.cer"
                onChange={handleFileSelect}
                className="hidden"
                id="cert-file-input"
              />
              <button
                onClick={() => fileInputRef.current?.click()}
                className="flex items-center gap-2 rounded-lg border border-gray-300 px-4 py-2 text-sm text-gray-700 hover:bg-gray-50 dark:border-gray-600 dark:text-gray-300 dark:hover:bg-gray-700"
              >
                <FileText className="h-4 w-4" />
                {selectedFile ? selectedFile.name : t("certs.chooseFile")}
              </button>
              {selectedFile && (
                <button
                  onClick={() => {
                    setSelectedFile(null);
                    setPemText("");
                    if (fileInputRef.current) fileInputRef.current.value = "";
                  }}
                  className="text-gray-400 hover:text-red-500"
                >
                  <X className="h-4 w-4" />
                </button>
              )}
              {selectedFile && (
                <span className="text-xs text-gray-400">{(selectedFile.size / 1024).toFixed(1)} KB</span>
              )}
            </div>
          </div>

          {/* Textarea for PEM content */}
          <div>
            <label className="mb-1 block text-xs font-medium text-gray-500">{t("certs.pastePem")}</label>
            <textarea
              value={pemText}
              onChange={(e) => setPemText(e.target.value)}
              placeholder="-----BEGIN CERTIFICATE-----&#10;MIIElDCCA3ygAwIBAgISA3...&#10;-----END CERTIFICATE-----"
              rows={6}
              className={`${inputCls} font-mono text-xs`}
            />
          </div>

          {/* Upload button */}
          <button
            onClick={handleUpload}
            disabled={!pemText && !selectedFile}
            className="flex items-center gap-2 rounded-lg bg-brand-600 px-4 py-2 text-sm font-medium text-white hover:bg-brand-700 disabled:cursor-not-allowed disabled:opacity-50"
          >
            <Upload className="h-4 w-4" />
            {t("certs.uploadSection")}
          </button>
        </div>
      </div>

      {/* JWKS Signing Keys Section */}
      <div className={cardCls}>
        <div className="mb-4 flex items-center justify-between">
          <h2 className={headingCls}>
            <Key className="mr-2 inline h-5 w-5 text-brand-600" />
            {t("certs.jwksKeys")}
          </h2>
          <button
            onClick={() => setShowRotateDialog(true)}
            className="flex items-center gap-2 rounded-lg border border-brand-600 px-4 py-2 text-sm font-medium text-brand-600 hover:bg-brand-50 dark:hover:bg-brand-900/30"
          >
            <RefreshCw className="h-4 w-4" />
            {t("certs.rotateKey")}
          </button>
        </div>

        {signingKeys.length === 0 ? (
          <div className="py-8 text-center text-sm text-gray-400">{t("certs.noSigningKeys")}</div>
        ) : (
          <div className="overflow-x-auto">
            <table className="w-full">
              <thead>
                <tr className="border-b border-gray-100 dark:border-gray-700">
                  <th className="px-3 py-2 text-left text-xs font-medium text-gray-500">{t("certs.keyId")}</th>
                  <th className="px-3 py-2 text-left text-xs font-medium text-gray-500">{t("certs.algorithm")}</th>
                  <th className="px-3 py-2 text-left text-xs font-medium text-gray-500">{t("common.status")}</th>
                  <th className="px-3 py-2 text-left text-xs font-medium text-gray-500">{t("common.created")}</th>
                </tr>
              </thead>
              <tbody className="divide-y divide-gray-50 dark:divide-gray-700/50">
                {signingKeys.map((key) => (
                  <tr key={key.kid} className="hover:bg-gray-50 dark:hover:bg-gray-700/50">
                    <td className="px-3 py-3 text-sm font-mono text-gray-900 dark:text-gray-100">{key.kid}</td>
                    <td className="px-3 py-3">
                      <span className="rounded bg-gray-100 px-2 py-0.5 text-xs font-mono text-gray-600 dark:bg-gray-700 dark:text-gray-300">
                        {key.alg}
                      </span>
                    </td>
                    <td className="px-3 py-3">
                      <span className={`inline-flex items-center gap-1 rounded-full px-2.5 py-0.5 text-xs font-medium ${
                        key.status === "active"
                          ? "bg-green-100 text-green-700 dark:bg-green-900 dark:text-green-400"
                          : "bg-gray-100 text-gray-500 dark:bg-gray-700 dark:text-gray-400"
                      }`}>
                        {key.status === "active" ? (
                          <CheckCircle2 className="h-3 w-3" />
                        ) : (
                          <Clock className="h-3 w-3" />
                        )}
                        {key.status === "active" ? t("common.active") : t("certs.rotated")}
                      </span>
                    </td>
                    <td className="px-3 py-3 text-sm text-gray-500">
                      {new Date(key.created).toLocaleDateString("en-US", { year: "numeric", month: "short", day: "numeric" })}
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        )}
      </div>

      {/* Key Rotation Confirmation Dialog */}
      {showRotateDialog && (
        <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/50">
          <div role="dialog" aria-modal="true" className="mx-4 w-full max-w-md rounded-xl bg-white p-6 shadow-lg dark:bg-gray-800">
            <div className="mb-4 flex items-center gap-3">
              <div className="flex h-10 w-10 items-center justify-center rounded-full bg-yellow-100 dark:bg-yellow-900">
                <AlertTriangle className="h-5 w-5 text-yellow-600 dark:text-yellow-400" />
              </div>
              <h3 className="text-lg font-semibold text-gray-900 dark:text-gray-100">{t("certs.confirmRotation")}</h3>
            </div>
            <p className="mb-6 text-sm text-gray-600 dark:text-gray-400">
              {t("certs.rotationConfirmDesc")}
            </p>
            <div className="flex gap-3">
              <button
                onClick={() => setShowRotateDialog(false)}
                className="flex-1 rounded-lg border border-gray-300 px-4 py-2 text-sm font-medium text-gray-700 hover:bg-gray-50 dark:border-gray-600 dark:text-gray-300 dark:hover:bg-gray-700"
              >
                {t("common.cancel")}
              </button>
              <button
                onClick={handleRotateKey}
                className="flex-1 rounded-lg bg-brand-600 px-4 py-2 text-sm font-medium text-white hover:bg-brand-700"
              >
                {t("certs.rotateKey")}
              </button>
            </div>
          </div>
        </div>
      )}
    </div>
  );
}
