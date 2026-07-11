/**
 * GGID React SDK — useAuditStream hook
 *
 * SSE-based real-time audit event stream with auto-reconnect.
 */

import { useState, useEffect, useCallback, useRef } from 'react';
import { useGGIDAuth } from './useGGIDAuth';

export interface StreamEvent {
  id: string;
  action: string;
  actor_id: string;
  actor_name: string;
  resource_type: string;
  resource_id: string;
  result: string;
  created_at: string;
  ip_address?: string;
}

export interface UseAuditStreamResult {
  events: StreamEvent[];
  isConnected: boolean;
  error: string | null;
  reconnect: () => void;
  clear: () => void;
}

export function useAuditStream(maxEvents = 100): UseAuditStreamResult {
  const { getAccessToken, isAuthenticated } = useGGIDAuth();
  const apiBaseUrl = typeof window !== 'undefined' ? localStorage.getItem('ggid_api_base') || '' : '';
  const tenantId = typeof window !== 'undefined' ? localStorage.getItem('ggid_tenant_id') || '' : '';

  const [events, setEvents] = useState<StreamEvent[]>([]);
  const [isConnected, setIsConnected] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const eventSourceRef = useRef<EventSource | null>(null);
  const reconnectTimer = useRef<ReturnType<typeof setTimeout> | null>(null);

  const connect = useCallback(() => {
    const tok = getAccessToken();
    if (!tok || typeof window === 'undefined' || !window.EventSource) return;

    // Close existing connection
    if (eventSourceRef.current) {
      eventSourceRef.current.close();
    }

    // SSE doesn't support custom headers; pass token as query param
    const url = `${apiBaseUrl}/api/v1/audit/stream?token=${encodeURIComponent(tok)}&tenant_id=${encodeURIComponent(tenantId)}`;
    const es = new EventSource(url);
    eventSourceRef.current = es;

    es.onopen = () => {
      setIsConnected(true);
      setError(null);
    };

    es.onmessage = (msg) => {
      try {
        const event: StreamEvent = JSON.parse(msg.data);
        setEvents((prev) => [event, ...prev].slice(0, maxEvents));
      } catch {
        // Ignore malformed messages
      }
    };

    es.onerror = () => {
      setIsConnected(false);
      setError('Connection lost. Reconnecting...');
      es.close();

      // Auto-reconnect after 5s
      if (reconnectTimer.current) clearTimeout(reconnectTimer.current);
      reconnectTimer.current = setTimeout(() => connect(), 5000);
    };
  }, [getAccessToken, apiBaseUrl, tenantId, maxEvents]);

  useEffect(() => {
    if (isAuthenticated) {
      connect();
    }
    return () => {
      if (eventSourceRef.current) eventSourceRef.current.close();
      if (reconnectTimer.current) clearTimeout(reconnectTimer.current);
    };
  }, [isAuthenticated, connect]);

  const clear = useCallback(() => setEvents([]), []);

  return {
    events,
    isConnected,
    error,
    reconnect: connect,
    clear,
  };
}
