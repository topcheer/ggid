"use client";
import { useEffect, useState } from "react";
import { ShieldAlert, X } from "lucide-react";
import { authHeader } from "@/lib/auth-helpers";
import { API_BASE_URL } from "@/lib/api-config";

const API_BASE = API_BASE_URL;

/**
 * Impersonation Banner — shows a red banner at the top when the current
 * session is an impersonation session. Provides a quick "End Session" button.
 */
export function ImpersonationBanner() {
  const [active, setActive] = useState(false);
  const [tenantName, setTenantName] = useState("");
  const [sessionId, setSessionId] = useState("");
  const [ending, setEnding] = useState(false);

  useEffect(() => {
    const raw = localStorage.getItem("ggid_impersonation");
    if (raw) {
      try {
        const imp = JSON.parse(raw);
        setActive(true);
        setTenantName(imp.tenant_name || "unknown tenant");
        setSessionId(imp.session_id || "");
      } catch { /* ignore */ }
    }
  }, []);

  const handleEnd = async () => {
    setEnding(true);
    try {
      await fetch(`${API_BASE}/api/v1/impersonate/end`, {
        method: "POST",
        headers: { "Content-Type": "application/json", ...authHeader() },
        body: JSON.stringify({ session_id: sessionId }),
      });
    } catch { /* ignore */ }
    localStorage.removeItem("ggid_impersonation");
    setActive(false);
    window.location.reload();
  };

  if (!active) return null;

  return (
    <div className="sticky top-0 z-50 flex items-center justify-center gap-3 bg-red-600 px-4 py-2 text-white shadow-md">
      <ShieldAlert className="h-5 w-5 shrink-0" />
      <span className="text-sm font-medium">
        You are operating as <strong>{tenantName}</strong> administrator
      </span>
      <button
        onClick={handleEnd}
        disabled={ending}
        className="ml-4 flex items-center gap-1 rounded-lg bg-white/20 px-3 py-1 text-xs font-medium hover:bg-white/30 disabled:opacity-50"
      >
        {ending ? "Ending..." : "End Session"}
        <X className="h-3 w-3" />
      </button>
    </div>
  );
}
