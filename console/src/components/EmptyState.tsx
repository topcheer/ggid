"use client";

import { ReactNode } from "react";
import { LucideIcon } from "lucide-react";

interface EmptyStateProps {
  icon: LucideIcon;
  title: string;
  description?: string;
  action?: {
    label: string;
    onClick: () => void;
    icon?: LucideIcon;
  };
  children?: ReactNode;
  className?: string;
}

/**
 * Unified empty state component.
 * Usage:
 *   <EmptyState icon={Users} title="No users found" description="Add your first user to get started" action={{ label: "Add User", onClick: handleClick, icon: Plus }} />
 */
export function EmptyState({ icon: Icon, title, description, action, children, className }: EmptyStateProps) {
  return (
    <div role="status"
    className={`flex flex-col items-center justify-center py-12 px-4 text-center ${className || ""}`}>
      <div className="w-16 h-16 rounded-2xl bg-gray-100 dark:bg-gray-800 flex items-center justify-center mb-4">
        <Icon className="w-8 h-8 text-gray-300 dark:text-gray-600" />
      </div>
      <h3 className="text-sm font-semibold text-gray-900 dark:text-white mb-1">{title}</h3>
      {description && <p className="text-xs text-gray-500 dark:text-gray-400 max-w-sm">{description}</p>}
      {action && (
        <button
          onClick={action.onClick}
          className="mt-4 flex items-center gap-1.5 px-4 py-2 bg-blue-600 hover:bg-blue-700 text-white rounded-lg text-sm font-medium transition-colors"
        >
          {action.icon && <action.icon className="w-4 h-4" />}
          {action.label}
        </button>
      )}
      {children}
    </div>
  );
}

/**
 * Inline loading spinner state.
 */
export function LoadingState({ className }: { className?: string }) {
  return (
    <div role="status"
    className={`flex items-center justify-center py-12 ${className || ""}`}>
      <div className="w-8 h-8 border-2 border-blue-600 border-t-transparent rounded-full animate-spin" />
    </div>
  );
}

/**
 * Error state with retry.
 */
export function ErrorState({ onRetry, message }: { onRetry?: () => void; message?: string }) {
  return (
    <div role="alert"
    className={`flex flex-col items-center justify-center py-12 px-4 text-center`}>
      <div className="w-16 h-16 rounded-2xl bg-red-100 dark:bg-red-950/30 flex items-center justify-center mb-4">
        <svg className="w-8 h-8 text-red-400" fill="none" viewBox="0 0 24 24" stroke="currentColor" strokeWidth={1.5}>
          <path strokeLinecap="round" strokeLinejoin="round" d="M12 9v3.75m9-.75a9 9 0 11-18 0 9 9 0 0118 0zm-9 3.75h.008v.008H12v-.008z" />
        </svg>
      </div>
      <h3 className="text-sm font-semibold text-gray-900 dark:text-white mb-1">{message || "Something went wrong"}</h3>
      {onRetry && (
        <button onClick={onRetry} className="mt-3 px-4 py-2 bg-gray-100 dark:bg-gray-800 hover:bg-gray-200 dark:hover:bg-gray-700 text-gray-700 dark:text-gray-300 rounded-lg text-sm font-medium">
          Try again
        </button>
      )}
    </div>
  );
}
