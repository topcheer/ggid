/**
 * GGID React SDK — ProtectedRoute component
 *
 * Redirects to /login if user is not authenticated.
 *
 * Usage:
 *   <ProtectedRoute><Dashboard /></ProtectedRoute>
 *   <ProtectedRoute loginPath="/auth/signin"><Admin /></ProtectedRoute>
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

  // Show loading spinner while checking auth state
  if (isLoading) {
    return (
      <div style={{ display: 'flex', justifyContent: 'center', alignItems: 'center', minHeight: '100vh' }}>
        <div style={{
          width: 32,
          height: 32,
          border: '3px solid #e5e7eb',
          borderTopColor: '#4f46e5',
          borderRadius: '50%',
          animation: 'spin 0.8s linear infinite',
        }} />
        <style>{`@keyframes spin { to { transform: rotate(360deg); } }`}</style>
      </div>
    );
  }

  // Redirect to login if not authenticated
  if (!isAuthenticated) {
    if (typeof window !== 'undefined') {
      const currentPath = window.location.pathname + window.location.search;
      window.location.href = `${loginPath}?redirect=${encodeURIComponent(currentPath)}`;
    }
    return null;
  }

  return <>{children}</>;
}
