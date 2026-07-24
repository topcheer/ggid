/**
 * Ensures a value is an array. Returns [] for null/undefined/non-array values.
 * Use when API responses might return null or objects instead of arrays.
 */
export function asArray<T>(value: unknown): T[] {
  if (Array.isArray(value)) return value as T[];
  if (value == null) return [];
  // Handle { items: [...] } or { data: [...] } wrappers
  if (typeof value === "object" && value !== null) {
    const obj = value as Record<string, unknown>;
    if (Array.isArray(obj.items)) return obj.items as T[];
    if (Array.isArray(obj.data)) return obj.data as T[];
    if (Array.isArray(obj.users)) return obj.users as T[];
    if (Array.isArray(obj.roles)) return obj.roles as T[];
    if (Array.isArray(obj.tenants)) return obj.tenants as T[];
    if (Array.isArray(obj.events)) return obj.events as T[];
    if (Array.isArray(obj.organizations)) return obj.organizations as T[];
  }
  return [];
}
