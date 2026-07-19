"use client";

import { useUserRole } from "@/lib/api";
import { ShieldX } from "lucide-react";

export function PermissionGuard({
  requiredScope,
  children,
}: {
  requiredScope: string;
  children: React.ReactNode;
}) {
  const { scopes, isPlatformAdmin } = useUserRole();

  // Platform admins can access everything
  if (isPlatformAdmin) return <>{children}</>;

  // Check if user has the required scope
  if (!scopes.includes(requiredScope)) {
    return (
      <div className="flex flex-col items-center justify-center py-20">
        <ShieldX className="h-12 w-12 text-gray-400" />
        <h2 className="mt-4 text-lg font-semibold text-gray-700 dark:text-gray-300">
          Access Denied
        </h2>
        <p className="mt-1 text-sm text-gray-500">
          You need the <code className="rounded bg-gray-100 px-1.5 py-0.5 text-xs dark:bg-gray-800">{requiredScope}</code> role to access this page.
        </p>
      </div>
    );
  }

  return <>{children}</>;
}
