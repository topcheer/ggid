/**
 * GGID React SDK — useThreatFeed hook
 *
 * Real-time threat intelligence feed via SSE.
 *
 * Usage:
 *   const { threats, isConnected, paused, pause, resume } = useThreatFeed();
 */

import { useState, useEffect, useCallback, useRef } from 'react';
import { useGGIDAuth } from './useGGIDAuth';

export interface ThreatEvent {
  id: string;
  severity: 'info' | 'low' | 'medium' | 'high' | 'critical';
  type: string;
  description: string;
  source_ip: string;
  indicators: string[];
  target: string;
  source: string;
  created_at: string;
}

export interface UseThreatFeedResult {
  threats: ThreatEvent[];
  isConnected: boolean;
  paused: boolean;
  error: string | null;
  pause: () => void;
  resume: () => void;
  clear: () => void;
}

export function useThreatFeed(maxEvents = 100): UseThreatFeedResult {
  const { getAccessToken, isAuthenticated } = useGGIDAuth();
  const apiBaseUrl = typeof window !== 'undefined' ? localStorage.getItem('ggid_api_base') || '' : '';
  const tenantId = typeof window !== 'undefined' ? localStorage.getItem('ggid_tenant_id') || '' : '';

  const [threats, setThreats] = useState<ThreatEvent[]>([]);
  const [isDemoData, setIsDemoData] = useState(true);
  const [isConnected, setIsConnected] = useState(false);
  const [paused, setPaused] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const esRef = useRef<EventSource | null>(null);
  const pausedRef = useRef(false);
  const reconnectRef = useRef<ReturnType<typeof setTimeout> | null>(null);

  const connect = useCallback(() => {
    const tok = getAccessToken();
    if (!tok || typeof window === 'undefined' || !window.EventSource) return;
    if (esRef.current) esRef.current.close();
    const url = `${apiBaseUrl}/api/v1/audit/threat-feed/stream?token=${encodeURIComponent(tok)}&tenant_id=${encodeURIComponent(tenantId)}`;
    const es = new EventSource(url);
    esRef.current = es;
    es.onopen = () => { setIsConnected(true); setError(null); };
    es.onmessage = (msg) => {
      if (pausedRef.current) return;
      try {
        const threat: ThreatEvent = JSON.parse(msg.data);
        setThreats((prev: any) => [threat, ...prev].slice(0, maxEvents));
      } catch { /* ignore */ }
    };
    es.onerror = () => {
      setIsConnected(false);
      es.close();
      reconnectRef.current = setTimeout(() => connect(), 5000);
    };
  }, [getAccessToken, apiBaseUrl, tenantId, maxEvents]);

  useEffect(() => {
    if (isAuthenticated) connect();
    return () => { if (esRef.current) esRef.current.close(); if (reconnectRef.current) clearTimeout(reconnectRef.current); };
  }, [isAuthenticated, connect]);

  const pause = useCallback(() => { setPaused(true); pausedRef.current = true; }, []);
  const resume = useCallback(() => { setPaused(false); pausedRef.current = false; }, []);
  const clear = useCallback(() => setThreats([]), []);

  return { threats, isConnected, paused, error, pause, resume, clear };
}
