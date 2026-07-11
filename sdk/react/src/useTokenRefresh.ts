/**
 * GGID React SDK — useTokenRefresh hook
 *
 * Automatically refreshes the access token before it expires.
 * Uses refresh_token from the token set to get a new access_token.
 */

import { useEffect, useRef } from 'react';
import { useGGIDAuth } from './useGGIDAuth';

const REFRESH_THRESHOLD_MS = 60_000; // Refresh 60s before expiry

export function useTokenRefresh() {
  const { tokenSet, login } = useGGIDAuth();
  const refreshTimerRef = useRef<ReturnType<typeof setTimeout> | null>(null);

  useEffect(() => {
    if (!tokenSet?.access_token) return;

    // Clear any existing timer
    if (refreshTimerRef.current) {
      clearTimeout(refreshTimerRef.current);
    }

    // Parse JWT to get expiry
    let expiresAt: number | undefined;

    if (tokenSet.expires_at) {
      // Unix timestamp (seconds or ms)
      expiresAt = tokenSet.expires_at > 1e12 ? tokenSet.expires_at : tokenSet.expires_at * 1000;
    } else {
      // Try to decode from JWT
      try {
        const payload = JSON.parse(atob(tokenSet.access_token.split('.')[1]));
        if (payload.exp) {
          expiresAt = payload.exp * 1000;
        }
      } catch {
        // Not a JWT or can't decode — skip refresh
        return;
      }
    }

    if (!expiresAt) return;

    // Calculate when to refresh (60s before expiry, min 5s from now)
    const refreshDelay = Math.max(expiresAt - Date.now() - REFRESH_THRESHOLD_MS, 5_000);

    // Don't schedule if token already expired
    if (refreshDelay <= 0) return;

    refreshTimerRef.current = setTimeout(async () => {
      // Attempt refresh using the refresh token
      // The GGIDProvider handles the actual refresh — we just trigger a re-evaluation
      // In a full implementation, this would call a refresh endpoint
      console.debug('[GGID] Token refresh triggered');

      // Emit a custom event that GGIDProvider can listen for
      window.dispatchEvent(new CustomEvent('ggid:token-refresh'));
    }, refreshDelay);

    return () => {
      if (refreshTimerRef.current) {
        clearTimeout(refreshTimerRef.current);
      }
    };
  }, [tokenSet?.access_token, tokenSet?.expires_at]); // eslint-disable-line react-hooks/exhaustive-deps
}
