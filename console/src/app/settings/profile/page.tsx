"use client";

import { useState, useEffect, useCallback, useRef } from "react";
import { useApi } from "@/lib/api";
import { useI18n } from "@/lib/i18n";
import {
  User,
  Mail,
  Clock,
  Globe,
  Save,
  RotateCcw,
  Camera,
  Check,
} from "lucide-react";

const TIMEZONES = [
  "UTC",
  "America/New_York",
  "America/Chicago",
  "America/Denver",
  "America/Los_Angeles",
  "America/Sao_Paulo",
  "Europe/London",
  "Europe/Paris",
  "Europe/Berlin",
  "Europe/Moscow",
  "Asia/Dubai",
  "Asia/Shanghai",
  "Asia/Tokyo",
  "Asia/Seoul",
  "Asia/Singapore",
  "Australia/Sydney",
  "Pacific/Auckland",
];

const LANGUAGES = [
  { value: "en", label: "English" },
  { value: "zh", label: "中文" },
  { value: "es", label: "Español" },
  { value: "fr", label: "Français" },
  { value: "de", label: "Deutsch" },
  { value: "ja", label: "日本語" },
  { value: "pt", label: "Português" },
];

interface ProfileData {
  display_name: string;
  bio: string;
  contact_email: string;
  timezone: string;
  language: string;
  avatar_url: string;
}

const DEFAULT_PROFILE: ProfileData = {
  display_name: "",
  bio: "",
  contact_email: "",
  timezone: "UTC",
  language: "en",
  avatar_url: "",
};

export default function ProfileSettingsPage() {
  const { apiFetch } = useApi();
  const fileInputRef = useRef<HTMLInputElement>(null);
  const [profile, setProfile] = useState<ProfileData>(DEFAULT_PROFILE);
  const [savedProfile, setSavedProfile] = useState<ProfileData>(DEFAULT_PROFILE);
  const [loading, setLoading] = useState(true);
  const [saving, setSaving] = useState(false);
  const [msg, setMsg] = useState<string | null>(null);
  const [error, setError] = useState<string | null>(null);

  // Load profile on mount
  useEffect(() => {
    const loadProfile = async () => {
      try {
        const data = await apiFetch<Record<string, unknown>>("/api/v1/users/me");
        const loaded: ProfileData = {
          display_name: (data.display_name as string) || (data.username as string) || "",
          bio: (data.bio as string) || "",
          contact_email: (data.contact_email as string) || "",
          timezone: (data.timezone as string) || "UTC",
          language: (data.locale as string) || (data.language as string) || "en",
          avatar_url: (data.avatar_url as string) || "",
        };
        setProfile(loaded);
        setSavedProfile(loaded);
      } catch {
        // Fallback to localStorage
        const stored = localStorage.getItem("ggid_profile_settings");
        if (stored) {
          try {
            const parsed = JSON.parse(stored) as ProfileData;
            setProfile(parsed);
            setSavedProfile(parsed);
          } catch {
            const name = localStorage.getItem("ggid_user_name") || "";
            const email = localStorage.getItem("ggid_user_email") || "";
            const fallback = {
              ...DEFAULT_PROFILE,
              display_name: name,
              contact_email: email,
              timezone: Intl.DateTimeFormat().resolvedOptions().timeZone || "UTC",
            };
            setProfile(fallback);
            setSavedProfile(fallback);
          }
        } else {
          const name = localStorage.getItem("ggid_user_name") || "";
          const email = localStorage.getItem("ggid_user_email") || "";
          const fallback = {
            ...DEFAULT_PROFILE,
            display_name: name,
            contact_email: email,
            timezone: Intl.DateTimeFormat().resolvedOptions().timeZone || "UTC",
          };
          setProfile(fallback);
          setSavedProfile(fallback);
        }
      } finally {
        setLoading(false);
      }
    };
    loadProfile();
  }, [apiFetch]);

  // Auto-dismiss messages
  useEffect(() => {
    if (msg) {
      const t = setTimeout(() => setMsg(null), 3000);
      return () => clearTimeout(t);
    }
  }, [msg]);

  const hasChanges = useCallback(() => {
    return JSON.stringify(profile) !== JSON.stringify(savedProfile);
  }, [profile, savedProfile]);

  const handleAvatarChange = (e: React.ChangeEvent<HTMLInputElement>) => {
    const file = e.target.files?.[0];
    if (!file) return;
    if (!file.type.startsWith("image/")) {
      setError("Please select an image file");
      return;
    }
    if (file.size > 2 * 1024 * 1024) {
      setError("Avatar image must be under 2MB");
      return;
    }
    const reader = new FileReader();
    reader.onload = () => {
      setProfile((prev) => ({ ...prev, avatar_url: reader.result as string }));
      setError(null);
    };
    reader.readAsDataURL(file);
  };

  const handleSave = async () => {
    setSaving(true);
    setError(null);
    try {
      await apiFetch("/api/v1/users/me", {
        method: "PUT",
        body: JSON.stringify({
          display_name: profile.display_name,
          bio: profile.bio,
          contact_email: profile.contact_email,
          timezone: profile.timezone,
          locale: profile.language,
          avatar_url: profile.avatar_url,
        }),
      });
      setSavedProfile(profile);
      setMsg("Profile saved successfully");
    } catch {
      // localStorage fallback
      localStorage.setItem("ggid_profile_settings", JSON.stringify(profile));
      setSavedProfile(profile);
      setMsg("Profile saved (offline mode)");
    } finally {
      setSaving(false);
    }
  };

  const handleReset = () => {
    setProfile(savedProfile);
    setError(null);
    setMsg("Changes reverted");
  };

  if (loading) {
    return (
      <div className="flex h-64 items-center justify-center">
        <div className="animate-spin rounded-full h-8 w-8 border-b-2 border-brand-600" />
      </div>
    );
  }

  return (
    <div className="max-w-3xl">
      <div className="mb-6 flex items-center gap-3">
        <User className="h-7 w-7 text-brand-600" />
        <div>
          <h1 className="text-2xl font-bold">Profile Settings</h1>
          <p className="text-sm text-gray-500">Manage your personal information and preferences</p>
        </div>
      </div>

      {msg && (
        <div role="status" className="mb-4 flex items-center gap-2 rounded-lg border border-green-200 bg-green-50 p-3 text-sm text-green-700">
          <Check className="h-4 w-4" /> {msg}
        </div>
      )}
      {error && (
        <div className="mb-4 rounded-lg border border-red-200 bg-red-50 p-3 text-sm text-red-700">
          {error}
        </div>
      )}

      <div className="space-y-6">
        {/* Avatar Upload */}
        <div className="rounded-xl border border-gray-200 bg-white p-6 shadow-sm">
          <h2 className="mb-4 text-sm font-semibold text-gray-700">Avatar</h2>
          <div className="flex items-center gap-4">
            <div className="relative">
              {profile.avatar_url ? (
                // eslint-disable-next-line @next/next/no-img-element
                <img
                  src={profile.avatar_url}
                  alt="Avatar"
                  className="h-20 w-20 rounded-full object-cover ring-2 ring-gray-200"
                />
              ) : (
                <div className="flex h-20 w-20 items-center justify-center rounded-full bg-brand-100 text-2xl font-bold text-brand-600 ring-2 ring-gray-200">
                  {profile.display_name.charAt(0).toUpperCase() || "?"}
                </div>
              )}
              <button
                onClick={() => fileInputRef.current?.click()}
                className="absolute -bottom-1 -right-1 flex h-7 w-7 items-center justify-center rounded-full bg-brand-600 text-white shadow-md hover:bg-brand-700"
                title="Upload avatar"
              >
                <Camera className="h-4 w-4" />
              </button>
            </div>
            <div>
              <button
                onClick={() => fileInputRef.current?.click()}
                className="rounded-lg border border-gray-300 px-3 py-1.5 text-sm font-medium text-gray-700 hover:bg-gray-50"
              >
                Upload Image
              </button>
              <p className="mt-1 text-xs text-gray-400">PNG, JPG up to 2MB</p>
            </div>
            <input
              ref={fileInputRef}
              type="file"
              accept="image/*"
              className="hidden"
              onChange={handleAvatarChange}
            />
          </div>
        </div>

        {/* Display Name */}
        <div className="rounded-xl border border-gray-200 bg-white p-6 shadow-sm">
          <label className="mb-1 block text-sm font-semibold text-gray-700">Display Name</label>
          <input
            type="text"
            value={profile.display_name}
            onChange={(e) => setProfile({ ...profile, display_name: e.target.value })}
            placeholder="Your display name"
            className="w-full rounded-lg border border-gray-300 px-3 py-2 text-sm focus:border-brand-500 focus:outline-none focus:ring-1 focus:ring-brand-500"
          />
        </div>

        {/* Bio */}
        <div className="rounded-xl border border-gray-200 bg-white p-6 shadow-sm">
          <div className="mb-1 flex items-center justify-between">
            <label className="block text-sm font-semibold text-gray-700">Bio</label>
            <span className={`text-xs ${profile.bio.length > 500 ? "text-red-500" : "text-gray-400"}`}>
              {profile.bio.length}/500
            </span>
          </div>
          <textarea
            value={profile.bio}
            onChange={(e) => setProfile({ ...profile, bio: e.target.value.slice(0, 500) })}
            placeholder="Tell us about yourself..."
            rows={4}
            className="w-full resize-none rounded-lg border border-gray-300 px-3 py-2 text-sm focus:border-brand-500 focus:outline-none focus:ring-1 focus:ring-brand-500"
          />
        </div>

        {/* Contact Email */}
        <div className="rounded-xl border border-gray-200 bg-white p-6 shadow-sm">
          <label className="mb-1 flex items-center gap-1.5 text-sm font-semibold text-gray-700">
            <Mail className="h-4 w-4" /> Contact Email
          </label>
          <p className="mb-2 text-xs text-gray-400">Separate from your authentication email</p>
          <input
            type="email"
            value={profile.contact_email}
            onChange={(e) => setProfile({ ...profile, contact_email: e.target.value })}
            placeholder="contact@example.com"
            className="w-full rounded-lg border border-gray-300 px-3 py-2 text-sm focus:border-brand-500 focus:outline-none focus:ring-1 focus:ring-brand-500"
          />
        </div>

        {/* Timezone & Language */}
        <div className="grid gap-6 sm:grid-cols-2">
          <div className="rounded-xl border border-gray-200 bg-white p-6 shadow-sm">
            <label className="mb-2 flex items-center gap-1.5 text-sm font-semibold text-gray-700">
              <Clock className="h-4 w-4" /> Timezone
            </label>
            <select
              value={profile.timezone}
              onChange={(e) => setProfile({ ...profile, timezone: e.target.value })}
              className="w-full rounded-lg border border-gray-300 px-3 py-2 text-sm focus:border-brand-500 focus:outline-none focus:ring-1 focus:ring-brand-500"
            >
              {TIMEZONES.map((tz: any) => (
                <option key={tz} value={tz}>
                  {tz}
                </option>
              ))}
            </select>
          </div>

          <div className="rounded-xl border border-gray-200 bg-white p-6 shadow-sm">
            <label className="mb-2 flex items-center gap-1.5 text-sm font-semibold text-gray-700">
              <Globe className="h-4 w-4" /> Language
            </label>
            <select
              value={profile.language}
              onChange={(e) => setProfile({ ...profile, language: e.target.value })}
              className="w-full rounded-lg border border-gray-300 px-3 py-2 text-sm focus:border-brand-500 focus:outline-none focus:ring-1 focus:ring-brand-500"
            >
              {LANGUAGES.map((lang: any) => (
                <option key={lang.value} value={lang.value}>
                  {lang.label}
                </option>
              ))}
            </select>
          </div>
        </div>

        {/* Action Buttons */}
        <div className="flex items-center gap-3">
          <button
            onClick={handleSave}
            disabled={saving || !hasChanges()}
            className="flex items-center gap-1.5 rounded-lg bg-brand-600 px-5 py-2 text-sm font-medium text-white hover:bg-brand-700 disabled:cursor-not-allowed disabled:opacity-50"
           aria-label="Save">
            <Save className="h-4 w-4" />
            {saving ? "Saving..." : "Save Changes"}
          </button>
          <button
            onClick={handleReset}
            disabled={!hasChanges()}
            className="flex items-center gap-1.5 rounded-lg border border-gray-300 px-5 py-2 text-sm font-medium text-gray-700 hover:bg-gray-50 disabled:cursor-not-allowed disabled:opacity-50"
          >
            <RotateCcw className="h-4 w-4" />
            Reset
          </button>
        </div>
      </div>
    </div>
  );
}
