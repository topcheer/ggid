/**
 * GGID React SDK — useTokenEvents hook
 *
 * SSE-based real-time token lifecycle event stream with pause/resume.
 *
 * Usage:
 *   const { events, isConnected, paused, pause, resume } = useTokenEvents();
 */

import { useState, useEffect, useCallback, useRef } from 'react';
import { useGGIDAuth } from './useGGIDAuth';

export interface TokenEvent {
  id: string;
  event_type: 'issued' | 'refreshed' | 'revoked' | 'expired' | 'introspected' | 'exchanged';
  token_type: 'access' | 'refresh' | 'id' | 'agent';
  client_id: string;
  user_id: string;
  scopes: string[];
  ip_address: string;
  created_at: string;
}

export interface UseTokenEventsResult {
  events: TokenEvent[];
  isConnected: boolean;
  paused: boolean;
  error: string | null;
  pause: () => void;
  resume: () => void;
  clear: () => void;
}

export function useTokenEvents(maxEvents = 100): UseTokenEventsResult {
  const { getAccessToken, isAuthenticated } = useGGIDAuth();
  const apiBaseUrl = typeof window !== 'undefined' ? localStorage.getItem('ggid_api_base') || '' : '';
  const tenantId = typeof window !== 'undefined' ? localStorage.getItem('ggid_tenant_id') || '' : '';

  const [events, setEvents] = useState<TokenEvent[]>([]);
  const [isDemoData, setIsDemoData] = useState(true);
  const [isConnected, setIsConnected] = useState(false);
  const [paused, setPaused] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const eventSourceRef = useRef<EventSource | null>(null);
  const pausedRef = useRef(false);
  const reconnectTimer = useRef<ReturnType<typeof setTimeout> | null>(null);
  const reconnectAttempts = useRef(0);

  const connect = useCallback(() => {
    const tok = getAccessToken();
    if (!tok || typeof window === 'undefined' || !window.EventSource) return;

    if (eventSourceRef.current) eventSourceRef.current.close();

    const url = `${apiBaseUrl}/api/v1/oauth/token-events/stream?token=${encodeURIComponent(tok)}&tenant_id=${encodeURIComponent(tenantId)}`;
    const es = new EventSource(url);
    eventSourceRef.current = es;

    es.onopen = () => { setIsConnected(true); setError(null); reconnectAttempts.current = 0; };

    es.onmessage = (msg) => {
      if (pausedRef.current) return;
      try {
        const event: TokenEvent = JSON.parse(msg.data);
        setEvents((prev: any) => [event, ...prev].slice(0, maxEvents));
      } catch { /* ignore */ }
    };

    es.onerror = () => {
      setIsConnected(false);
      reconnectAttempts.current += 1;
      setError(`Connection lost (attempt ${reconnectAttempts.current}). Reconnecting...`);
      es.close();
      const delay = Math.min(5000 * Math.pow(2, reconnectAttempts.current - 1), 30000);
      if (reconnectTimer.current) clearTimeout(reconnectTimer.current);
      reconnectTimer.current = setTimeout(() => connect(), delay);
    };
  }, [getAccessToken, apiBaseUrl, tenantId, maxEvents]);

  useEffect(() => {
    if (isAuthenticated) connect();
    return () => {
      if (eventSourceRef.current) eventSourceRef.current.close();
      if (reconnectTimer.current) clearTimeout(reconnectTimer.current);
    };
  }, [isAuthenticated, connect]);

  const pause = useCallback(() => { setPaused(true); pausedRef.current = true; }, []);
  const resume = useCallback(() => { setPaused(false); pausedRef.current = false; }, []);
  const clear = useCallback(() => setEvents([]), []);

  return { events, isConnected, paused, error, pause, resume, clear };
}
