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
  severity: 'info' | 'warning' | 'critical';
  created_at: string;
  ip_address?: string;
}

export type SeverityFilter = 'all' | 'info' | 'warning' | 'critical' | 'warning+';

export interface UseAuditStreamResult {
  events: StreamEvent[];
  filteredEvents: StreamEvent[];
  isConnected: boolean;
  reconnectAttempts: number;
  error: string | null;
  severityFilter: SeverityFilter;
  setSeverityFilter: (filter: SeverityFilter) => void;
  reconnect: () => void;
  clear: () => void;
}

export function useAuditStream(maxEvents = 100): UseAuditStreamResult {
  const [severityFilter, setSeverityFilter] = useState<SeverityFilter>('all');
  const reconnectAttempts = useRef(0);
  const maxReconnectDelay = 30000;
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
        // Default severity to info if not provided
        if (!event.severity) event.severity = event.result === 'failure' || event.result === 'denied' ? 'warning' : 'info';
        setEvents((prev) => [event, ...prev].slice(0, maxEvents));
      } catch {
        // Ignore malformed messages
      }
    };

    es.onerror = () => {
      setIsConnected(false);
      reconnectAttempts.current += 1;
      setError(`Connection lost (attempt ${reconnectAttempts.current}). Reconnecting...`);
      es.close();

      // Exponential backoff: min(5s * 2^attempts, 30s)
      const delay = Math.min(5000 * Math.pow(2, reconnectAttempts.current - 1), maxReconnectDelay);
      if (reconnectTimer.current) clearTimeout(reconnectTimer.current);
      reconnectTimer.current = setTimeout(() => connect(), delay);
    };
  }, [getAccessToken, apiBaseUrl, tenantId, maxEvents]);

  useEffect(() => {
    if (isAuthenticated) {
      reconnectAttempts.current = 0;
      connect();
    }
    return () => {
      if (eventSourceRef.current) eventSourceRef.current.close();
      if (reconnectTimer.current) clearTimeout(reconnectTimer.current);
    };
  }, [isAuthenticated, connect]);

  const clear = useCallback(() => setEvents([]), []);

  // Severity filtering
  const filteredEvents = events.filter((e) => {
    if (severityFilter === 'all') return true;
    if (severityFilter === 'warning+') return e.severity === 'warning' || e.severity === 'critical';
    return e.severity === severityFilter;
  });

  return {
    events,
    filteredEvents,
    isConnected,
    reconnectAttempts: reconnectAttempts.current,
    error,
    severityFilter,
    setSeverityFilter,
    reconnect: connect,
    clear,
  };
}
