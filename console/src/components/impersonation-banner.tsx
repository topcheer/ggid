"use client";
import { useEffect, useState } from "react";
import { ShieldAlert, X } from "lucide-react";
import { useApi } from "@/lib/api";

/**
 * Impersonation Banner — shows a red banner at the top when the current
 * session is an impersonation session. Provides a quick "End Session" button.
 *
 * Reads from ggid_impersonation_state (written by /admin/impersonate page):
 * { userId, username, email, reason, startedAt, tokenId }
 */
export function ImpersonationBanner() {
  const { apiFetch } = useApi();
  const [active, setActive] = useState(false);
  const [username, setUsername] = useState("");
  const [email, setEmail] = useState("");
  const [reason, setReason] = useState("");
  const [tokenId, setTokenId] = useState("");
  const [ending, setEnding] = useState(false);

  useEffect(() => {
    const raw = localStorage.getItem("ggid_impersonation_state");
    if (raw) {
      try {
        const imp = JSON.parse(raw);
        if (imp && imp.userId) {
          setActive(true);
          setUsername(imp.username || "unknown user");
          setEmail(imp.email || "");
          setReason(imp.reason || "");
          setTokenId(imp.tokenId || "");
        }
      } catch { /* ignore */ }
    }
  }, []);

  const handleEnd = async () => {
    setEnding(true);
    try {
      await apiFetch("/api/v1/auth/impersonate/revoke", {
        method: "POST",
        body: JSON.stringify({ token_id: tokenId }),
      }).catch(() => null);
    } catch { /* ignore */ }
    localStorage.removeItem("ggid_impersonation_state");
    localStorage.removeItem("ggid_impersonation_token");
    setActive(false);
    window.location.reload();
  };

  if (!active) return null;

  return (
    <div className="sticky top-0 z-50 flex items-center justify-center gap-3 bg-red-600 px-4 py-2 text-white shadow-md">
      <ShieldAlert className="h-5 w-5 shrink-0" />
      <span className="text-sm font-medium">
        Impersonating <strong>{username}</strong>{email ? ` (${email})` : ""}
        {reason && <span className="text-white/70"> — {reason}</span>}
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