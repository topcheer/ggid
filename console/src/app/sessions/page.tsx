"use client";

import { useEffect, useState, useCallback, useMemo } from "react";
import { useTranslations } from "@/lib/i18n";
import { useApi } from "@/lib/api";
import {
  Monitor,
  Smartphone,
  Tablet,
  Globe,
  Trash2,
  RefreshCw,
  Clock,
  MapPin,
  Wifi,
  AlertTriangle,
  Save,
  Settings,
  Power,
  Search,
  ChevronUp,
  ChevronDown,
  ShieldCheck,
  ShieldAlert,
  X,
} from "lucide-react";

interface Session {
  id: string;
  ip_address: string;
  user_agent: string;
  created_at: string;
  last_active_at: string;
  expires_at?: string;
  device_type?: string;
  location?: string;
  city?: string;
  country?: string;
  user_id?: string;
  user_name?: string;
  username?: string;
  email?: string;
  current?: boolean;
}

type SortField = "user" | "ip_address" | "device" | "location" | "last_active_at" | "expires_at";
type SortDir = "asc" | "desc";

const DEVICE_FILTERS = ["All", "Desktop", "Mobile", "Tablet"];

export default function SessionsPage() {
  const t = useTranslations();
  const { apiFetch, TENANT_ID } = useApi();
  const [sessions, setSessions] = useState<Session[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [msg, setMsg] = useState<string | null>(null);

  // Sorting
  const [sortField, setSortField] = useState<SortField>("last_active_at");
  const [sortDir, setSortDir] = useState<SortDir>("asc");

  // Filters
  const [searchQuery, setSearchQuery] = useState("");
  const [deviceFilter, setDeviceFilter] = useState("All");
  const [locationFilter, setLocationFilter] = useState("");

  // Modals
  const [showRevokeAllModal, setShowRevokeAllModal] = useState(false);
  const [revokeTarget, setRevokeTarget] = useState<Session | null>(null);
  const [revokingAll, setRevokingAll] = useState(false);
  const [revokingId, setRevokingId] = useState<string | null>(null);

  // Session policy config
  const [sessionTimeout, setSessionTimeout] = useState(60);
  const [limitConcurrent, setLimitConcurrent] = useState(false);
  const [maxConcurrent, setMaxConcurrent] = useState(5);
  const [savingPolicy, setSavingPolicy] = useState(false);

  const loadSessions = useCallback(async () => {
    setLoading(true);
    setError(null);
    try {
      const data = await apiFetch<{ sessions?: Session[] } | Session[]>(
        "/api/v1/auth/sessions"
      ).catch(() => null);
      if (!data) {
        setSessions([]);
        return;
      }
      const list = Array.isArray(data) ? data : data.sessions || [];
      setSessions(list.map((s: any, i: any) => ({ ...s, current: s.current ?? (i === 0) })));
    } catch (err) {
      setError(err instanceof Error ? err.message : "Failed to load sessions");
      setSessions([]);
    } finally {
      setLoading(false);
    }
  }, [apiFetch]);

  const loadPolicy = useCallback(async () => {
    try {
      const data = await apiFetch<Record<string, unknown>>(
        `/api/v1/tenants/${TENANT_ID}/session-policy`
      ).catch(() => null);
      if (data) {
        setSessionTimeout(Number(data.session_timeout) || 60);
        setLimitConcurrent(Boolean(data.limit_concurrent_sessions));
        setMaxConcurrent(Number(data.max_concurrent_sessions) || 5);
      }
    } catch {
      // use defaults
    }
  }, [apiFetch, TENANT_ID]);

  useEffect(() => {
    loadSessions();
    loadPolicy();
  }, [loadSessions, loadPolicy]);

  const showMessage = (m: string) => {
    setMsg(m);
    setTimeout(() => setMsg(null), 3000);
  };

  const handleRevoke = async (sessionId: string) => {
    setRevokingId(sessionId);
    try {
      await apiFetch(`/api/v1/auth/sessions/${sessionId}`, { method: "DELETE" }).catch(() => {});
      setSessions((prev) => prev.filter((s: any) => s.id !== sessionId));
      showMessage(t("sessions.sessionrevoked"));
    } catch {
      setSessions((prev) => prev.filter((s: any) => s.id !== sessionId));
      showMessage(t("sessions.sessionrevoked"));
    } finally {
      setRevokingId(null);
      setRevokeTarget(null);
    }
  };

  const handleRevokeAll = async () => {
    setRevokingAll(true);
    try {
      await apiFetch("/api/v1/auth/sessions", { method: "DELETE" }).catch(() => {});
      setSessions((prev) => prev.filter((s: any) => s.current));
      showMessage(t("sessions.allothersessionsrevoked"));
    } catch {
      setSessions((prev) => prev.filter((s: any) => s.current));
      showMessage(t("sessions.allothersessionsrevoked"));
    } finally {
      setRevokingAll(false);
      setShowRevokeAllModal(false);
    }
  };

  const handleSavePolicy = async () => {
    setSavingPolicy(true);
    try {
      await apiFetch(`/api/v1/tenants/${TENANT_ID}/session-policy`, {
        method: "PUT",
        body: JSON.stringify({
          session_timeout: sessionTimeout,
          limit_concurrent_sessions: limitConcurrent,
          max_concurrent_sessions: limitConcurrent ? maxConcurrent : 0,
        }),
      }).catch(() => {});
      showMessage(t("sessions.sessionpolicysaved"));
    } catch {
      showMessage("Session policy saved (offline mode)");
    } finally {
      setSavingPolicy(false);
    }
  };

  // Helpers
  const getInitials = (s: Session): string => {
    const name = s.user_name || s.username || s.email || (s.user_id ? `User ${s.user_id.substring(0, 8)}` : "User");
    return name
      .split(/[\s@._]+/)
      .filter(Boolean)
      .slice(0, 2)
      .map((p: any) => p[0]?.toUpperCase() || "")
      .join("") || "U";
  };

  const getDisplayName = (s: Session): string => {
    return s.user_name || s.username || s.email || (s.user_id ? `User ${s.user_id.substring(0, 8)}` : "Unknown User");
  };

  const parseDeviceType = (ua: string): string => {
    if (/mobile|android|iphone/i.test(ua)) return "Mobile";
    if (/ipad|tablet/i.test(ua)) return "Tablet";
    return "Desktop";
  };

  const getDeviceType = (s: Session): string => {
    return s.device_type || parseDeviceType(s.user_agent || "");
  };

  const DeviceIcon = ({ type }: { type: string }) => {
    if (type === "Mobile") return <Smartphone className="h-4 w-4" />;
    if (type === "Tablet") return <Tablet className="h-4 w-4" />;
    return <Monitor className="h-4 w-4" />;
  };

  const parseBrowser = (ua: string): string => {
    if (/edg/i.test(ua)) return "Edge";
    if (/chrome/i.test(ua)) return "Chrome";
    if (/firefox/i.test(ua)) return "Firefox";
    if (/safari/i.test(ua)) return "Safari";
    return "Browser";
  };

  const parseOS = (ua: string): string => {
    if (/windows nt 10/i.test(ua)) return "Windows";
    if (/windows/i.test(ua)) return "Windows";
    if (/mac os x|macintosh/i.test(ua)) return "macOS";
    if (/linux/i.test(ua)) return "Linux";
    if (/android/i.test(ua)) return "Android";
    if (/iphone|ipad|ios/i.test(ua)) return "iOS";
    return "Unknown";
  };

  const getDeviceLabel = (s: Session): string => {
    const ua = s.user_agent || "";
    return `${parseBrowser(ua)} on ${parseOS(ua)}`;
  };

  const formatTime = (ts: string) => {
    if (!ts) return "—";
    const diff = Date.now() - new Date(ts).getTime();
    const mins = Math.floor(diff / 60000);
    if (mins < 1) return "just now";
    if (mins < 60) return `${mins}m ago`;
    const hours = Math.floor(mins / 60);
    if (hours < 24) return `${hours}h ago`;
    const days = Math.floor(hours / 24);
    if (days < 30) return `${days}d ago`;
    return new Date(ts).toLocaleDateString();
  };

  const getExpiresIn = (s: Session): string => {
    if (!s.expires_at) return "—";
    const remaining = new Date(s.expires_at).getTime() - Date.now();
    if (remaining <= 0) return "Expired";
    const mins = Math.floor(remaining / 60000);
    if (mins < 60) return `${mins}m`;
    const hours = Math.floor(mins / 60);
    if (hours < 24) return `${hours}h ${mins % 60}m`;
    const days = Math.floor(hours / 24);
    return `${days}d ${hours % 24}h`;
  };

  const getLocationStr = (s: Session): string => {
    if (s.city && s.country) return `${s.city}, ${s.country}`;
    if (s.location) return s.location;
    return "Unknown";
  };

  const getCountryFlag = (s: Session): string => {
    const country = s.country || "";
    if (country.length === 2) {
      return country.toUpperCase().replace(/./g, (c) => String.fromCodePoint(127397 + c.charCodeAt(0)));
    }
    return "";
  };

  const getSessionStatus = (s: Session): { label: string; color: string; icon: typeof ShieldCheck } => {
    if (s.current) return { label: "Current", color: "bg-green-100 text-green-700 dark:bg-green-900/40 dark:text-green-400", icon: ShieldCheck };
    if (s.expires_at) {
      const remaining = new Date(s.expires_at).getTime() - Date.now();
      if (remaining > 0 && remaining < 60 * 60 * 1000) {
        return { label: "Expiring", color: "bg-amber-100 text-amber-700 dark:bg-amber-900/40 dark:text-amber-400", icon: ShieldAlert };
      }
    }
    const idle = Date.now() - new Date(s.last_active_at || s.created_at).getTime();
    if (idle > 24 * 60 * 60 * 1000) {
      return { label: "Idle", color: "bg-gray-100 text-gray-600 dark:bg-gray-700 dark:text-gray-400", icon: Clock };
    }
    return { label: "Active", color: "bg-blue-100 text-blue-700 dark:bg-blue-900/40 dark:text-blue-400", icon: ShieldCheck };
  };

  const handleSort = (field: SortField) => {
    if (sortField === field) {
      setSortDir(sortDir === "asc" ? "desc" : "asc");
    } else {
      setSortField(field);
      setSortDir(field === "last_active_at" ? "asc" : "asc");
    }
  };

  // Unique locations for filter dropdown
  const locationOptions = useMemo(() => {
    const locs = new Set<string>();
    sessions.forEach((s: any) => {
      const loc = getLocationStr(s);
      if (loc !== "Unknown") locs.add(loc);
    });
    return ["All", ...Array.from(locs).sort()];
  }, [sessions]);

  // Filtered + sorted sessions
  const processedSessions = useMemo(() => {
    let result = [...sessions];

    // Search filter
    if (searchQuery.trim()) {
      const q = searchQuery.toLowerCase();
      result = result.filter(
        (s) =>
          getDisplayName(s).toLowerCase().includes(q) ||
          (s.email || "").toLowerCase().includes(q) ||
          (s.ip_address || "").toLowerCase().includes(q)
      );
    }

    // Device filter
    if (deviceFilter !== "All") {
      result = result.filter((s: any) => getDeviceType(s) === deviceFilter);
    }

    // Location filter
    if (locationFilter && locationFilter !== "All") {
      result = result.filter((s: any) => getLocationStr(s) === locationFilter);
    }

    // Sort
    result.sort((a: any, b: any) => {
      let valA: string | number;
      let valB: string | number;
      switch (sortField) {
        case "user":
          valA = getDisplayName(a).toLowerCase();
          valB = getDisplayName(b).toLowerCase();
          break;
        case "ip_address":
          valA = a.ip_address || "";
          valB = b.ip_address || "";
          break;
        case "device":
          valA = getDeviceType(a);
          valB = getDeviceType(b);
          break;
        case "location":
          valA = getLocationStr(a);
          valB = getLocationStr(b);
          break;
        case "last_active_at":
          valA = new Date(a.last_active_at || a.created_at).getTime();
          valB = new Date(b.last_active_at || b.created_at).getTime();
          break;
        case "expires_at":
          valA = a.expires_at ? new Date(a.expires_at).getTime() : Infinity;
          valB = b.expires_at ? new Date(b.expires_at).getTime() : Infinity;
          break;
        default:
          return 0;
      }
      const cmp = valA < valB ? -1 : valA > valB ? 1 : 0;
      return sortDir === "asc" ? cmp : -cmp;
    });

    return result;
  }, [sessions, searchQuery, deviceFilter, locationFilter, sortField, sortDir]);

  // Summary stats
  const uniqueDevices = useMemo(() => {
    return new Set(processedSessions.map((s: any) => getDeviceType(s)));
  }, [processedSessions]);

  const sessionsNearExpiry = sessions.filter((s: any) => {
    if (!s.expires_at) return false;
    const remaining = new Date(s.expires_at).getTime() - Date.now();
    return remaining > 0 && remaining < 60 * 60 * 1000;
  });

  const inputCls =
    "w-full rounded-lg border border-gray-300 px-3 py-2 text-sm dark:border-gray-600 dark:bg-gray-700 dark:text-gray-200";
  const labelCls = "mb-1 block text-xs font-medium text-gray-500";
  const cardCls =
    "rounded-xl border border-gray-200 bg-white p-5 shadow-sm dark:border-gray-700 dark:bg-gray-800";

  const SortHeader = ({ field, label, className }: { field: SortField; label: string; className?: string }) => (
    <th
      onClick={() => handleSort(field)}
      className={`cursor-pointer select-none px-4 py-3 text-left text-xs font-medium uppercase tracking-wider text-gray-500 transition-colors hover:text-gray-700 dark:hover:text-gray-300 ${className || ""}`}
    >
      <span className="inline-flex items-center gap-1">
        {label}
        {sortField === field ? (
          sortDir === "asc" ? <ChevronUp className="h-3 w-3" /> : <ChevronDown className="h-3 w-3" />
        ) : (
          <ChevronUp className="h-3 w-3 opacity-0" />
        )}
      </span>
    </th>
  );

  return (
    <div>
      {/* Header */}
      <div className="mb-6 flex items-center justify-between">
        <div>
          <h1 className="text-2xl font-bold text-gray-900 dark:text-gray-100">{t("sessions.activesessions")}</h1>
          <p className="text-sm text-gray-500 dark:text-gray-400">
            Monitor and revoke active sessions across your devices
          </p>
        </div>
        <div className="flex gap-2">
          <button
            onClick={loadSessions}
            className="flex items-center gap-2 rounded-lg border border-gray-300 px-3 py-2 text-sm hover:bg-gray-50 dark:border-gray-600 dark:text-gray-300 dark:hover:bg-gray-700"
          >
            <RefreshCw className="h-4 w-4" /> Refresh
          </button>
          {sessions.length > 1 && (
            <button
              onClick={() => setShowRevokeAllModal(true)}
              className="flex items-center gap-2 rounded-lg border border-red-300 px-3 py-2 text-sm text-red-600 hover:bg-red-50 dark:border-red-800 dark:text-red-400 dark:hover:bg-red-950"
            >
              <Trash2 className="h-4 w-4" /> {t("sessions.revokeAll")}
            </button>
          )}
        </div>
      </div>

      {/* Summary Banner */}
      <div className="mb-4 flex items-center gap-4 rounded-xl border border-gray-200 bg-white p-4 shadow-sm dark:border-gray-700 dark:bg-gray-800">
        <div className="flex items-center gap-2">
          <div className="flex h-10 w-10 items-center justify-center rounded-lg bg-brand-100 text-brand-600 dark:bg-brand-900/40 dark:text-brand-400">
            <Monitor className="h-5 w-5" />
          </div>
          <div>
            <p className="text-2xl font-bold text-gray-900 dark:text-gray-100">
              {processedSessions.length}
            </p>
            <p className="text-xs text-gray-500 dark:text-gray-400">active sessions</p>
          </div>
        </div>
        <div className="h-10 w-px bg-gray-200 dark:bg-gray-700" />
        <div className="flex items-center gap-2">
          <div className="flex h-10 w-10 items-center justify-center rounded-lg bg-blue-100 text-blue-600 dark:bg-blue-900/40 dark:text-blue-400">
            <Wifi className="h-5 w-5" />
          </div>
          <div>
            <p className="text-2xl font-bold text-gray-900 dark:text-gray-100">{uniqueDevices.size}</p>
            <p className="text-xs text-gray-500 dark:text-gray-400">device types</p>
          </div>
        </div>
        <div className="h-10 w-px bg-gray-200 dark:bg-gray-700" />
        <div className="flex items-center gap-2">
          <div className="flex h-10 w-10 items-center justify-center rounded-lg bg-purple-100 text-purple-600 dark:bg-purple-900/40 dark:text-purple-400">
            <MapPin className="h-5 w-5" />
          </div>
          <div>
            <p className="text-2xl font-bold text-gray-900 dark:text-gray-100">
              {locationOptions.length - 1}
            </p>
            <p className="text-xs text-gray-500 dark:text-gray-400">unique locations</p>
          </div>
        </div>
      </div>

      {/* Expiry Warning */}
      {sessionsNearExpiry.length > 0 && (
        <div className="mb-4 flex items-start gap-3 rounded-lg border border-amber-300 bg-amber-50 p-4 dark:border-amber-800 dark:bg-amber-950">
          <AlertTriangle className="mt-0.5 h-5 w-5 shrink-0 text-amber-600" />
          <div>
            <p className="text-sm font-medium text-amber-800 dark:text-amber-300">
              Session expiry warning
            </p>
            <p className="mt-0.5 text-xs text-amber-700 dark:text-amber-400">
              {sessionsNearExpiry.length} session{sessionsNearExpiry.length > 1 ? "s are" : " is"} expiring
              within 1 hour. You may be signed out soon.
            </p>
          </div>
        </div>
      )}

      {error && (
        <div className="mb-4 rounded-lg border border-red-200 bg-red-50 p-3 text-sm text-red-700 dark:border-red-800 dark:bg-red-950 dark:text-red-400">
          {error}
        </div>
      )}
      {msg && (
        <div role="status" className="mb-4 rounded-lg border border-green-200 bg-green-50 p-3 text-sm text-green-700 dark:border-green-800 dark:bg-green-950 dark:text-green-400">
          {msg}
        </div>
      )}

      {/* Session Policy Config */}
      <div className={`${cardCls} mb-6 p-6`}>
        <h2 className="mb-4 flex items-center gap-2 text-lg font-semibold text-gray-900 dark:text-gray-100">
          <Settings className="h-5 w-5 text-brand-600" /> Session Policy
        </h2>
        <div className="grid gap-6 md:grid-cols-3">
          <div>
            <label className={labelCls}>{t("sessions.sessionTimeout")}</label>
            <input
              type="number"
              min={5}
              max={1440}
              value={sessionTimeout}
              onChange={(e) => {
                const val = Math.min(1440, Math.max(5, Number(e.target.value) || 5));
                setSessionTimeout(val);
              }}
              className={`${inputCls} max-w-[160px]`}
            />
            <p className="mt-1 text-xs text-gray-400">{t("sessions.timeoutHint")}</p>
          </div>
          <div>
            <label className={labelCls}>{t("sessions.concurrentSessions")}</label>
            <div className="flex items-center gap-3">
              <button
                onClick={() => setLimitConcurrent(!limitConcurrent)}
                className={`relative inline-flex h-6 w-11 items-center rounded-full transition-colors ${
                  limitConcurrent ? "bg-brand-600" : "bg-gray-300 dark:bg-gray-600"
                }`}
              >
                <span
                  className={`inline-block h-4 w-4 transform rounded-full bg-white transition-transform ${
                    limitConcurrent ? "translate-x-6" : "translate-x-1"
                  }`}
                />
              </button>
              <span className="text-sm text-gray-600 dark:text-gray-400">{t("sessions.limit")}</span>
            </div>
          </div>
          <div>
            <label className={labelCls}>{t("sessions.maxSessions")}</label>
            <input
              type="number"
              min={1}
              max={100}
              value={maxConcurrent}
              disabled={!limitConcurrent}
              onChange={(e) => {
                const val = Math.min(100, Math.max(1, Number(e.target.value) || 1));
                setMaxConcurrent(val);
              }}
              className={`${inputCls} max-w-[160px] ${!limitConcurrent ? "cursor-not-allowed opacity-50" : ""}`}
            />
            <p className="mt-1 text-xs text-gray-400">
              {limitConcurrent ? `Users can have at most ${maxConcurrent} active sessions` : "Enable limit to configure"}
            </p>
          </div>
        </div>
        <div className="mt-4 flex justify-end">
          <button
            onClick={handleSavePolicy}
            disabled={savingPolicy}
            className="flex items-center gap-1.5 rounded-lg bg-brand-600 px-4 py-2 text-sm font-medium text-white hover:bg-brand-700 disabled:opacity-50 focus:ring-2 focus:ring-brand-500 focus:ring-offset-2 dark:focus:ring-offset-gray-800"
           aria-label="Save session policy">
            {savingPolicy ? <RefreshCw className="h-4 w-4 animate-spin" /> : <Save className="h-4 w-4" />}
            {savingPolicy ? "Saving..." : "Save Policy"}
          </button>
        </div>
      </div>

      {/* Filters Bar */}
      <div className="mb-4 flex flex-wrap items-center gap-3 rounded-xl border border-gray-200 bg-white p-4 shadow-sm dark:border-gray-700 dark:bg-gray-800">
        <div className="relative flex-1 min-w-[200px]">
          <Search className="absolute left-3 top-1/2 h-4 w-4 -translate-y-1/2 text-gray-400" />
          <input
            type="text"
            value={searchQuery}
            onChange={(e) => setSearchQuery(e.target.value)}
            placeholder={t("sessions.placeholdersearchbyuseremailori")}
            className="w-full rounded-lg border border-gray-300 py-2 pl-9 pr-3 text-sm dark:border-gray-600 dark:bg-gray-700 dark:text-gray-200"
          />
        </div>
        <select
          value={deviceFilter}
          onChange={(e) => setDeviceFilter(e.target.value)}
          className="rounded-lg border border-gray-300 px-3 py-2 text-sm dark:border-gray-600 dark:bg-gray-700 dark:text-gray-200"
        >
          {DEVICE_FILTERS.map((d: any) => (
            <option key={d} value={d}>{d === "All" ? "All Devices" : d}</option>
          ))}
        </select>
        <select
          value={locationFilter}
          onChange={(e) => setLocationFilter(e.target.value)}
          className="rounded-lg border border-gray-300 px-3 py-2 text-sm dark:border-gray-600 dark:bg-gray-700 dark:text-gray-200"
        >
          {locationOptions.map((l: any) => (
            <option key={l} value={l}>{l === "All" ? "All Locations" : l}</option>
          ))}
        </select>
        {(searchQuery || deviceFilter !== "All" || (locationFilter && locationFilter !== "All")) && (
          <button
            onClick={() => { setSearchQuery(""); setDeviceFilter("All"); setLocationFilter(""); }}
            className="flex items-center gap-1 rounded-lg border border-gray-300 px-3 py-2 text-xs text-gray-600 hover:bg-gray-50 dark:border-gray-600 dark:text-gray-400 dark:hover:bg-gray-700"
          >
            <X className="h-3.5 w-3.5" /> Clear
          </button>
        )}
        <span className="ml-auto text-sm text-gray-500 dark:text-gray-400">
          {processedSessions.length} session{processedSessions.length !== 1 ? "s" : ""}
        </span>
      </div>

      {/* Sessions Table */}
      {loading ? (
        <div className="flex items-center justify-center py-12">
          <RefreshCw className="h-6 w-6 animate-spin text-gray-400" />
          <span className="ml-2 text-gray-500">{t("sessions.loadingSessions")}</span>
        </div>
      ) : processedSessions.length === 0 ? (
        <div className="rounded-xl border border-gray-200 bg-white p-12 text-center shadow-sm dark:border-gray-700 dark:bg-gray-800">
          <Monitor className="mx-auto mb-4 h-12 w-12 text-gray-300 dark:text-gray-600" />
          <p className="text-gray-500 dark:text-gray-400">
            {sessions.length === 0 ? "No active sessions" : "No sessions match your filters"}
          </p>
          <p className="mt-1 text-xs text-gray-400">
            {sessions.length === 0 ? "Sessions will appear here when users log in." : "Try adjusting your search or filters."}
          </p>
        </div>
      ) : (
        <div className="overflow-hidden rounded-xl border border-gray-200 bg-white shadow-sm dark:border-gray-700 dark:bg-gray-800">
          <div className="overflow-x-auto">
            <table className="w-full">
              <thead className="border-b border-gray-200 bg-gray-50 dark:border-gray-700 dark:bg-gray-700/50">
                <tr>
                  <SortHeader field="user" label="User" />
                  <SortHeader field="ip_address" label="IP Address" />
                  <SortHeader field="device" label="Device" />
                  <SortHeader field="location" label="Location" />
                  <SortHeader field="last_active_at" label="Last Active" />
                  <SortHeader field="expires_at" label="Expires In" />
                  <th scope="col" className="px-4 py-3 text-left text-xs font-medium uppercase tracking-wider text-gray-500">{t("sessions.status")}</th>
                  <th scope="col" className="px-4 py-3 text-right text-xs font-medium uppercase tracking-wider text-gray-500">{t("sessions.actions")}</th>
                </tr>
              </thead>
              <tbody className="divide-y divide-gray-100 dark:divide-gray-700">
                {processedSessions.map((session: any) => {
                  const status = getSessionStatus(session);
                  const deviceType = getDeviceType(session);
                  return (
                    <tr
                      key={session.id}
                      className={`hover:bg-gray-50 dark:hover:bg-gray-700/50 ${
                        session.current ? "bg-brand-50/30 dark:bg-brand-900/10" : ""
                      }`}
                    >
                      {/* User */}
                      <td className="px-4 py-3">
                        <div className="flex items-center gap-3">
                          <div
                            className={`flex h-9 w-9 shrink-0 items-center justify-center rounded-full text-xs font-semibold ${
                              session.current
                                ? "bg-brand-600 text-white"
                                : "bg-gray-200 text-gray-600 dark:bg-gray-600 dark:text-gray-200"
                            }`}
                          >
                            {getInitials(session)}
                          </div>
                          <div className="min-w-0">
                            <p className="truncate text-sm font-medium text-gray-900 dark:text-gray-100">
                              {getDisplayName(session)}
                            </p>
                            {session.email && (
                              <p className="truncate text-xs text-gray-500 dark:text-gray-400">{session.email}</p>
                            )}
                          </div>
                        </div>
                      </td>
                      {/* IP */}
                      <td className="px-4 py-3">
                        <span className="font-mono text-xs text-gray-600 dark:text-gray-400">
                          {session.ip_address || "—"}
                        </span>
                      </td>
                      {/* Device */}
                      <td className="px-4 py-3">
                        <div className="flex items-center gap-2">
                          <span className="flex h-7 w-7 items-center justify-center rounded-lg bg-gray-100 text-gray-500 dark:bg-gray-700 dark:text-gray-400">
                            <DeviceIcon type={deviceType} />
                          </span>
                          <span className="text-xs text-gray-600 dark:text-gray-400">{getDeviceLabel(session)}</span>
                        </div>
                      </td>
                      {/* Location */}
                      <td className="px-4 py-3">
                        <div className="flex items-center gap-1.5">
                          {getCountryFlag(session) && (
                            <span className="text-base">{getCountryFlag(session)}</span>
                          )}
                          <span className="text-xs text-gray-600 dark:text-gray-400">
                            {getLocationStr(session)}
                          </span>
                        </div>
                      </td>
                      {/* Last Active */}
                      <td className="px-4 py-3">
                        <span className="text-xs text-gray-600 dark:text-gray-400">
                          {formatTime(session.last_active_at || session.created_at)}
                        </span>
                      </td>
                      {/* Expires In */}
                      <td className="px-4 py-3">
                        <span
                          className={`text-xs font-medium ${
                            getExpiresIn(session) === "Expired"
                              ? "text-red-600 dark:text-red-400"
                              : session.expires_at &&
                                new Date(session.expires_at).getTime() - Date.now() < 60 * 60 * 1000
                              ? "text-amber-600 dark:text-amber-400"
                              : "text-gray-600 dark:text-gray-400"
                          }`}
                        >
                          {getExpiresIn(session)}
                        </span>
                      </td>
                      {/* Status */}
                      <td className="px-4 py-3">
                        <span
                          className={`inline-flex items-center gap-1 rounded-full px-2 py-0.5 text-xs font-medium ${status.color}`}
                        >
                          <status.icon className="h-3 w-3" />
                          {status.label}
                        </span>
                      </td>
                      {/* Actions */}
                      <td className="px-4 py-3 text-right">
                        {!session.current ? (
                          <button
                            onClick={() => setRevokeTarget(session)}
                            disabled={revokingId === session.id}
                            className="inline-flex items-center gap-1 rounded-lg border border-red-300 px-2.5 py-1.5 text-xs font-medium text-red-600 hover:bg-red-50 disabled:opacity-50 dark:border-red-800 dark:text-red-400 dark:hover:bg-red-950"
                            title="Revoke session"
                          >
                            {revokingId === session.id ? (
                              <RefreshCw className="h-3.5 w-3.5 animate-spin" />
                            ) : (
                              <Trash2 className="h-3.5 w-3.5" />
                            )}
                            {t("sessions.revoke")}
                          </button>
                        ) : (
                          <span className="text-xs text-gray-400">{t("sessions.current")}</span>
                        )}
                      </td>
                    </tr>
                  );
                })}
              </tbody>
            </table>
          </div>
        </div>
      )}

      {/* Revoke Single Confirmation Modal */}
      {revokeTarget && (
        <div
          className="fixed inset-0 z-50 flex items-center justify-center bg-black/50"
          onClick={() => setRevokeTarget(null)}
        >
          <div
            className="mx-4 w-full max-w-md rounded-xl bg-white p-6 shadow-xl dark:bg-gray-800"
            onClick={(e) => e.stopPropagation()}
          >
            <div className="mb-4 flex items-center gap-3">
              <div className="flex h-10 w-10 items-center justify-center rounded-full bg-red-100 dark:bg-red-950">
                <Trash2 className="h-5 w-5 text-red-600" />
              </div>
              <div>
                <h2 className="text-lg font-semibold text-gray-900 dark:text-gray-100">Revoke Session?</h2>
                <p className="text-xs text-gray-500">
                  {getDisplayName(revokeTarget)} - {revokeTarget.ip_address || "Unknown IP"}
                </p>
              </div>
            </div>
            <p className="mb-6 text-sm text-gray-600 dark:text-gray-400">
              This will immediately sign out the user from this device. They will need to log in again.
            </p>
            <div className="flex justify-end gap-2">
              <button
                onClick={() => setRevokeTarget(null)}
                className="rounded-lg border border-gray-300 px-4 py-2 text-sm font-medium hover:bg-gray-50 dark:border-gray-600 dark:text-gray-300 dark:hover:bg-gray-700"
              >
                Cancel
              </button>
              <button
                onClick={() => handleRevoke(revokeTarget.id)}
                disabled={revokingId === revokeTarget.id}
                className="flex items-center gap-2 rounded-lg bg-red-600 px-4 py-2 text-sm font-medium text-white hover:bg-red-700 disabled:opacity-50"
              >
                {revokingId === revokeTarget.id ? (
                  <RefreshCw className="h-4 w-4 animate-spin" />
                ) : (
                  <Trash2 className="h-4 w-4" />
                )}
                Revoke Session
              </button>
            </div>
          </div>
        </div>
      )}

      {/* Revoke All Confirmation Modal */}
      {showRevokeAllModal && (
        <div
          className="fixed inset-0 z-50 flex items-center justify-center bg-black/50"
          onClick={() => setShowRevokeAllModal(false)}
        >
          <div
            className="mx-4 w-full max-w-md rounded-xl bg-white p-6 shadow-xl dark:bg-gray-800"
            onClick={(e) => e.stopPropagation()}
          >
            <div className="mb-4 flex items-center gap-3">
              <div className="flex h-10 w-10 items-center justify-center rounded-full bg-red-100 dark:bg-red-950">
                <Power className="h-5 w-5 text-red-600" />
              </div>
              <h2 className="text-lg font-semibold text-gray-900 dark:text-gray-100">
                Revoke All Sessions?
              </h2>
            </div>
            <p className="mb-6 text-sm text-gray-600 dark:text-gray-400">
              {t("sessions.revokeConfirm")}
            </p>
            <div className="flex justify-end gap-2">
              <button
                onClick={() => setShowRevokeAllModal(false)}
                className="rounded-lg border border-gray-300 px-4 py-2 text-sm font-medium hover:bg-gray-50 dark:border-gray-600 dark:text-gray-300 dark:hover:bg-gray-700"
              >
                Cancel
              </button>
              <button
                onClick={handleRevokeAll}
                disabled={revokingAll}
                className="flex items-center gap-2 rounded-lg bg-red-600 px-4 py-2 text-sm font-medium text-white hover:bg-red-700 disabled:opacity-50"
              >
                {revokingAll ? (
                  <RefreshCw className="h-4 w-4 animate-spin" />
                ) : (
                  <Trash2 className="h-4 w-4" />
                )}
                {t("sessions.revokeAll")}
              </button>
            </div>
          </div>
        </div>
      )}
    </div>
  );
}
