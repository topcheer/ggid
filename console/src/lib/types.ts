/**
 * Centralized TypeScript types for all API responses.
 * Used across console pages for type-safe fetch calls.
 */

export interface User {
  id: string;
  tenant_id: string;
  username: string;
  email: string;
  phone: string;
  status: string;
  email_verified: boolean;
  display_name: string;
  locale: string;
  timezone: string;
  created_at: string;
  updated_at: string;
}

export interface Role {
  id: string;
  tenant_id: string;
  key: string;
  name: string;
  description: string;
  permissions: string[];
  created_at: string;
  updated_at: string;
}

export interface Organization {
  id: string;
  tenant_id: string;
  name: string;
  description: string;
  parent_id: string | null;
  member_count: number;
  created_at: string;
  updated_at: string;
}

export interface AuditEvent {
  id: string;
  tenant_id: string;
  event_type: string;
  actor_id: string;
  actor_name: string;
  action: string;
  resource_type: string;
  resource_id: string;
  ip_address: string;
  user_agent: string;
  severity: string;
  result: string;
  timestamp: string;
  metadata: Record<string, unknown>;
}

export interface Policy {
  id: string;
  tenant_id: string;
  name: string;
  description: string;
  effect: "allow" | "deny";
  conditions: Record<string, unknown>;
  priority: number;
  enabled: boolean;
  created_at: string;
  updated_at: string;
}

export interface Group {
  id: string;
  tenant_id: string;
  name: string;
  description: string;
  parent_id: string | null;
  member_count: number;
  roles: string[];
  created_at: string;
  updated_at: string;
}

export interface Session {
  id: string;
  user_id: string;
  user_name: string;
  ip_address: string;
  device: string;
  location: string;
  last_active: string;
  expires_at: string;
  status: "active" | "revoked" | "expired";
}

export interface OAuthClient {
  id: string;
  client_id: string;
  name: string;
  client_type: "confidential" | "public";
  grant_types: string[];
  redirect_uris: string[];
  scopes: string[];
  token_lifetime: number;
  enabled: boolean;
  created_at: string;
}

export interface ApiKey {
  id: string;
  name: string;
  key_prefix: string;
  scopes: string[];
  expires_at: string | null;
  last_used: string | null;
  status: "active" | "expired" | "revoked";
  created_at: string;
}

export interface Webhook {
  id: string;
  url: string;
  description: string;
  events: string[];
  active: boolean;
  created_at: string;
}

export interface PageResult<T> {
  items?: T[];
  users?: T[];
  total?: number;
  total_count?: number;
}

export interface HealthStatus {
  service: string;
  status: "healthy" | "degraded" | "down";
  latency_ms: number;
  details?: string;
}

export type ApiResult<T> =
  | { ok: true; data: T }
  | { ok: false; error: string };
