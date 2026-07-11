"use client";

import { useState, useCallback } from "react";
import { Check, Copy } from "lucide-react";

interface CopyButtonProps {
  /** The text to copy to clipboard */
  value: string;
  /** Optional label to show alongside the icon (e.g. "Copy API Key") */
  label?: string;
  /** Tooltip text shown on hover */
  title?: string;
  /** Visual variant */
  variant?: "icon" | "button" | "ghost";
  /** Additional CSS classes */
  className?: string;
  /** Mask the value display (for secrets) */
  masked?: boolean;
}

export function CopyButton({
  value,
  label,
  title,
  variant = "icon",
  className = "",
  masked = false,
}: CopyButtonProps) {
  const [copied, setCopied] = useState(false);

  const handleCopy = useCallback(async () => {
    try {
      if (navigator.clipboard && navigator.clipboard.writeText) {
        await navigator.clipboard.writeText(value);
      } else {
        // Fallback for browsers without Clipboard API
        const textarea = document.createElement("textarea");
        textarea.value = value;
        textarea.style.position = "fixed";
        textarea.style.opacity = "0";
        document.body.appendChild(textarea);
        textarea.select();
        document.execCommand("copy");
        document.body.removeChild(textarea);
      }
      setCopied(true);
      setTimeout(() => setCopied(false), 2000);
    } catch {
      // Silently fail — clipboard may be blocked
    }
  }, [value]);

  const baseCls =
    "inline-flex items-center gap-1.5 transition-colors select-none";
  const variantCls =
    variant === "icon"
      ? "p-1.5 rounded-md text-gray-400 hover:text-gray-600 hover:bg-gray-100 dark:hover:bg-gray-700 dark:hover:text-gray-200"
      : variant === "button"
        ? "px-3 py-1.5 rounded-lg text-xs font-medium border border-gray-300 text-gray-700 hover:bg-gray-50 dark:border-gray-600 dark:text-gray-300 dark:hover:bg-gray-700"
        : "px-2 py-1 text-xs text-gray-400 hover:text-brand-600 dark:hover:text-brand-400";

  const displayValue = masked
    ? value.length > 12
      ? `${value.slice(0, 8)}••••${value.slice(-4)}`
      : "••••••••"
    : value;

  return (
    <button
      type="button"
      onClick={handleCopy}
      title={title || "Copy to clipboard"}
      className={`${baseCls} ${variantCls} ${className} ${
        copied ? "text-green-500 dark:text-green-400" : ""
      }`}
    >
      {copied ? (
        <Check className="h-3.5 w-3.5" />
      ) : (
        <Copy className="h-3.5 w-3.5" />
      )}
      {label && <span>{copied ? "Copied!" : label}</span>}
      {variant === "ghost" && !label && displayValue && (
        <span className="font-mono text-xs">{displayValue}</span>
      )}
    </button>
  );
}
