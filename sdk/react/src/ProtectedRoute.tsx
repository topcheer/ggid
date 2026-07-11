/**
 * GGID React SDK — ProtectedRoute component
 * Redirects to loginPath if not authenticated
 */

import { type ReactNode } from 'react';
import { useGGIDAuth } from './useGGIDAuth';

export function ProtectedRoute({
  children,
  loginPath = '/login',
}: {
  children: ReactNode;
  loginPath?: string;
}) {
  const { isAuthenticated, isLoading } = useGGIDAuth();

  if (isLoading) {
    return (
      <div className="flex min-h-screen items-center justify-center">
        <div className="h-8 w-8 animate-spin rounded-full border-4 border-gray-300 border-t-blue-600" />
      </div>
    );
  }

  if (!isAuthenticated) {
    if (typeof window !== 'undefined') {
      window.location.href = loginPath;
    }
    return null;
  }

  return <>{children}</>;
}
