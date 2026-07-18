"use client";

import { useEffect } from "react";
import { RefreshCw, AlertTriangle, Home, ExternalLink } from "lucide-react";
import { useTranslations } from "@/lib/i18n";

export default function Error({
  error,
  reset,
}: {
  error: Error & { digest?: string };
  reset: () => void;
}) {
  const t = useTranslations();
  useEffect(() => {
    console.error(error);
  }, [error]);

  return (
    <div className="flex min-h-screen flex-col items-center justify-center bg-gradient-to-br from-red-50 via-white to-orange-50 dark:from-gray-950 dark:via-gray-900 dark:to-red-950 px-4">
      {/* Icon */}
      <div className="flex h-16 w-16 items-center justify-center rounded-2xl bg-red-100 dark:bg-red-950/30 shadow-lg mb-6">
        <AlertTriangle className="h-8 w-8 text-red-500" />
      </div>

      <h1 className="text-2xl font-bold text-gray-900 dark:text-white">
        {t("emptyState.error") || "Something went wrong"}
      </h1>
      <p className="mt-1 text-sm text-gray-500 dark:text-gray-400 text-center max-w-sm">
        An unexpected error occurred. Your data is safe — try refreshing the page.
      </p>

      {/* Error digest */}
      {error.digest && (
        <code className="mt-3 px-3 py-1 rounded-lg bg-gray-100 dark:bg-gray-800 text-xs font-mono text-gray-500">
          Error ID: {error.digest}
        </code>
      )}

      {/* Actions */}
      <div className="flex items-center gap-3 mt-6">
        <button onClick={reset}
          className="flex items-center gap-2 rounded-xl bg-gradient-to-r from-blue-600 to-purple-600 px-5 py-2.5 text-sm font-medium text-white hover:opacity-90 shadow-md transition-opacity">
          <RefreshCw className="h-4 w-4" />{t("emptyState.tryAgain") || "Try again"}
        </button>
        <a href="/dashboard"
          className="flex items-center gap-2 rounded-xl bg-white dark:bg-gray-800 border border-gray-200 dark:border-gray-700 px-5 py-2.5 text-sm font-medium text-gray-600 dark:text-gray-400 hover:border-gray-300">
          <Home className="h-4 w-4" />Dashboard
        </a>
      </div>

      {/* Feedback link */}
      <a href="https://github.com/topcheer/ggid/issues/new" target="_blank" rel="noopener noreferrer"
        className="mt-4 flex items-center gap-1 text-xs text-gray-400 hover:text-gray-600 dark:hover:text-gray-300">
        Report this issue <ExternalLink className="w-3 h-3" />
      </a>
    </div>
  );
}
