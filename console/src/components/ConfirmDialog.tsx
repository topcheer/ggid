"use client";

import { useState, createContext, useContext, useCallback, ReactNode } from "react";
import { AlertTriangle, X, Loader2 } from "lucide-react";

interface ConfirmOptions {
  title: string;
  description?: string;
  confirmLabel?: string;
  cancelLabel?: string;
  variant?: "danger" | "warning" | "info";
  onConfirm: () => void | Promise<void>;
}

interface ConfirmContextValue {
  confirm: (opts: ConfirmOptions) => void;
}

const ConfirmContext = createContext<ConfirmContextValue>({
  confirm: () => {},
});

export function useConfirm() {
  return useContext(ConfirmContext);
}

const variantConfig = {
  danger: {
    icon: AlertTriangle,
    iconBg: "bg-red-100 dark:bg-red-950/50",
    iconColor: "text-red-500",
    buttonBg: "bg-red-600 hover:bg-red-700",
    border: "border-red-200 dark:border-red-900",
  },
  warning: {
    icon: AlertTriangle,
    iconBg: "bg-amber-100 dark:bg-amber-950/50",
    iconColor: "text-amber-500",
    buttonBg: "bg-amber-600 hover:bg-amber-700",
    border: "border-amber-200 dark:border-amber-900",
  },
  info: {
    icon: AlertTriangle,
    iconBg: "bg-blue-100 dark:bg-blue-950/50",
    iconColor: "text-blue-500",
    buttonBg: "bg-blue-600 hover:bg-blue-700",
    border: "border-blue-200 dark:border-blue-900",
  },
};

export function ConfirmProvider({ children }: { children: ReactNode }) {
  const [dialog, setDialog] = useState<ConfirmOptions | null>(null);
  const [loading, setLoading] = useState(false);

  const confirm = useCallback((opts: ConfirmOptions) => {
    setDialog(opts);
  }, []);

  const handleConfirm = useCallback(async () => {
    if (!dialog) return;
    setLoading(true);
    try {
      await dialog.onConfirm();
    } finally {
      setLoading(false);
      setDialog(null);
    }
  }, [dialog]);

  const handleCancel = useCallback(() => {
    setDialog(null);
  }, []);

  return (
    <ConfirmContext.Provider value={{ confirm }}>
      {children}
      {dialog && (
        <ConfirmDialog
          options={dialog}
          loading={loading}
          onConfirm={handleConfirm}
          onCancel={handleCancel}
        />
      )}
    </ConfirmContext.Provider>
  );
}

function ConfirmDialog({
  options,
  loading,
  onConfirm,
  onCancel,
}: {
  options: ConfirmOptions;
  loading: boolean;
  onConfirm: () => void;
  onCancel: () => void;
}) {
  const variant = variantConfig[options.variant || "danger"];
  const Icon = variant.icon;

  return (
    <>
      {/* Backdrop */}
      <div
        className="fixed inset-0 z-[200] bg-black/40 backdrop-blur-sm"
        onClick={onCancel}
        aria-hidden="true"
      />

      {/* Dialog */}
      <div className="fixed left-1/2 top-1/2 z-[201] w-full max-w-sm -translate-x-1/2 -translate-y-1/2">
        <div className={`rounded-2xl border-2 bg-white dark:bg-gray-900 shadow-2xl ${variant.border}`} role="alertdialog" aria-modal="true" aria-labelledby="confirm-title" aria-describedby="confirm-desc">
          {/* Header */}
          <div className="flex items-start gap-3 p-6 pb-4">
            <div className={`flex h-10 w-10 items-center justify-center rounded-xl ${variant.iconBg} flex-shrink-0`}>
              <Icon className={`h-5 w-5 ${variant.iconColor}`} />
            </div>
            <div className="flex-1">
              <h3 id="confirm-title" className="text-base font-bold text-gray-900 dark:text-white">{options.title}</h3>
              {options.description && (
                <p id="confirm-desc" className="mt-1 text-sm text-gray-500 dark:text-gray-400">{options.description}</p>
              )}
            </div>
            <button onClick={onCancel} className="text-gray-400 hover:text-gray-600 dark:hover:text-gray-300" aria-label="Close dialog">
              <X className="h-4 w-4" />
            </button>
          </div>

          {/* Actions */}
          <div className="flex justify-end gap-2 px-6 pb-6">
            <button
              onClick={onCancel}
              disabled={loading}
              className="px-4 py-2 rounded-lg bg-gray-100 dark:bg-gray-800 text-gray-600 dark:text-gray-400 hover:bg-gray-200 dark:hover:bg-gray-700 text-sm font-medium disabled:opacity-50"
            >
              {options.cancelLabel || "Cancel"}
            </button>
            <button
              onClick={onConfirm}
              disabled={loading}
              className={`flex items-center gap-1.5 px-4 py-2 rounded-lg text-white text-sm font-medium disabled:opacity-50 ${variant.buttonBg}`}
            >
              {loading && <Loader2 className="h-3.5 w-3.5 animate-spin" />}
              {options.confirmLabel || "Confirm"}
            </button>
          </div>
        </div>
      </div>
    </>
  );
}
