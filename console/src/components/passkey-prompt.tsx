"use client";
import { useState, useEffect } from "react";
import { Fingerprint, X, Loader2, CheckCircle2, AlertCircle } from "lucide-react";
import { offerPasskeyUpgrade, userHasPasskey } from "@/lib/webauthn-conditional";

const DISMISS_KEY = "passkey_prompt_dismissed";
const DISMISS_DAYS = 7;

export function PasskeyPrompt() {
  const [show, setShow] = useState(false);
  const [status, setStatus] = useState<"idle" | "loading" | "success" | "error">("idle");
  const [errorMsg, setErrorMsg] = useState("");

  useEffect(() => {
    let cancelled = false;
    async function check() {
      // Check if dismissed recently
      const dismissed = localStorage.getItem(DISMISS_KEY);
      if (dismissed) {
        const days = (Date.now() - parseInt(dismissed)) / 86400000;
        if (days < DISMISS_DAYS) return;
      }

      // Check WebAuthn support
      if (typeof PublicKeyCredential === "undefined") return;

      // Check user has no passkey
      const userId = localStorage.getItem("ggid_user_id");
      if (!userId) return;

      try {
        const has = await userHasPasskey(userId);
        if (cancelled) return;
        if (!has) {
          setShow(true);
        }
      } catch {
        // Silently skip on error
      }
    }
    check();
    return () => { cancelled = true; };
  }, []);

  const handleBind = async () => {
    setStatus("loading");
    setErrorMsg("");
    const token = localStorage.getItem("ggid_access_token") || "";
    const userId = localStorage.getItem("ggid_user_id") || "";
    if (!token || !userId) {
      setStatus("error");
      setErrorMsg("Not logged in");
      return;
    }
    try {
      const ok = await offerPasskeyUpgrade({ accessToken: token, userId });
      if (ok) {
        setStatus("success");
        setTimeout(() => setShow(false), 3000);
      } else {
        setStatus("error");
        setErrorMsg("Passkey creation was cancelled or failed. Try again.");
      }
    } catch {
      setStatus("error");
      setErrorMsg("Something went wrong. Please try again.");
    }
  };

  const handleDismiss = () => {
    localStorage.setItem(DISMISS_KEY, Date.now().toString());
    setShow(false);
  };

  if (!show) return null;

  return (
    <div className="rounded-xl border border-blue-200 bg-gradient-to-r from-blue-50 to-indigo-50 p-4 dark:border-blue-900 dark:from-blue-950 dark:to-indigo-950">
      <div className="flex items-start gap-3">
        <div className="flex h-10 w-10 shrink-0 items-center justify-center rounded-full bg-blue-100 dark:bg-blue-900">
          <Fingerprint className="h-5 w-5 text-blue-600 dark:text-blue-400" />
        </div>
        <div className="flex-1">
          <h3 className="text-sm font-semibold text-gray-900 dark:text-white">
            Set up Passkey for this device
          </h3>
          <p className="mt-0.5 text-sm text-gray-600 dark:text-gray-400">
            Next time, sign in instantly with fingerprint or Face ID — no password needed.
          </p>

          {status === "success" ? (
            <div className="mt-3 flex items-center gap-2 text-sm text-green-600">
              <CheckCircle2 className="h-4 w-4" /> Passkey created successfully!
            </div>
          ) : status === "error" ? (
            <div className="mt-3">
              <div className="flex items-center gap-2 text-sm text-red-600">
                <AlertCircle className="h-4 w-4" /> {errorMsg}
              </div>
              <div className="mt-2 flex gap-2">
                <button onClick={handleBind} className="rounded-lg bg-blue-600 px-3 py-1.5 text-xs font-medium text-white hover:bg-blue-700">
                  Retry
                </button>
                <button onClick={handleDismiss} className="rounded-lg border border-gray-300 px-3 py-1.5 text-xs dark:border-gray-700">
                  Not now
                </button>
              </div>
            </div>
          ) : (
            <div className="mt-3 flex gap-2">
              <button
                onClick={handleBind}
                disabled={status === "loading"}
                className="flex items-center gap-1.5 rounded-lg bg-blue-600 px-4 py-1.5 text-sm font-medium text-white hover:bg-blue-700 disabled:opacity-50"
              >
                {status === "loading" ? <Loader2 className="h-4 w-4 animate-spin" /> : <Fingerprint className="h-4 w-4" />}
                Bind Passkey
              </button>
              <button onClick={handleDismiss} className="rounded-lg border border-gray-300 px-3 py-1.5 text-sm text-gray-600 hover:bg-gray-50 dark:border-gray-700 dark:text-gray-400">
                Not now
              </button>
            </div>
          )}
        </div>
        <button onClick={handleDismiss} className="text-gray-400 hover:text-gray-600 dark:hover:text-gray-300">
          <X className="h-4 w-4" />
        </button>
      </div>
    </div>
  );
}
