"use client";

import { useEffect } from "react";
import { RefreshCw, AlertTriangle } from "lucide-react";
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
    <div className="flex min-h-screen flex-col items-center justify-center bg-gray-50 px-4 dark:bg-gray-950">
      <div className="flex h-20 w-20 items-center justify-center rounded-full bg-red-50 dark:bg-red-950/30">
        <AlertTriangle className="h-10 w-10 text-red-500" />
      </div>
      <h1 className="mt-6 text-xl font-bold text-gray-900 dark:text-white">
        {t("common.somethingWentWrong")}
      </h1>
      <p className="mt-2 max-w-md text-center text-sm text-gray-500 dark:text-gray-400">
        {t("common.unexpectedError")}
      </p>
      {error.digest && (
        <p className="mt-2 font-mono text-xs text-gray-400">{t("common.errorId")}: {error.digest}</p>
      )}
      <button
        onClick={reset}
        className="mt-6 flex items-center gap-2 rounded-lg bg-brand-600 px-5 py-2.5 text-sm font-medium text-white hover:bg-brand-700 transition"
      >
        <RefreshCw className="h-4 w-4" />
        {t("common.reload")}
      </button>
    </div>
  );
}
