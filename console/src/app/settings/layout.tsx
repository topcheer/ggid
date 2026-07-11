import Link from "next/link";
import { Settings, User, Shield, Building2, Palette, Bell, Key, Globe, Lock, FileText, Database } from "lucide-react";

const settingsNav = [
  { href: "/settings/profile", label: "Profile", icon: User },
  { href: "/settings/security", label: "Security", icon: Shield },
  { href: "/settings/tenant", label: "Tenant", icon: Building2 },
  { href: "/settings/tenant-config", label: "Tenant Config", icon: Settings },
  { href: "/settings/branding", label: "Branding", icon: Palette },
  { href: "/settings/branding-custom", label: "Brand Custom", icon: Palette },
  { href: "/settings/notifications", label: "Notifications", icon: Bell },
  { href: "/settings/notifications/preview", label: "Notification Preview", icon: Bell },
  { href: "/settings/password", label: "Password Policy", icon: Lock },
  { href: "/settings/password-policy", label: "Password Rules", icon: Lock },
  { href: "/settings/ip-allowlist", label: "IP Allowlist", icon: Globe },
  { href: "/settings/webhooks", label: "Webhooks", icon: Bell },
  { href: "/settings/sso", label: "SSO", icon: Globe },
  { href: "/settings/certificates", label: "Certificates", icon: FileText },
  { href: "/settings/sessions", label: "Sessions", icon: Shield },
  { href: "/settings/api-keys", label: "API Keys", icon: Key },
  { href: "/settings/data", label: "Data", icon: Database },
];

export default function SettingsLayout({ children }: { children: React.ReactNode }) {
  return (
    <div className="flex gap-6">
      {/* Settings sidebar nav */}
      <aside className="hidden w-56 shrink-0 lg:block">
        <div className="sticky top-0 space-y-0.5">
          <h2 className="mb-3 flex items-center gap-2 px-3 text-sm font-bold text-gray-900 dark:text-white">
            <Settings className="h-4 w-4" /> Settings
          </h2>
          {settingsNav.map(({ href, label, icon: Icon }) => (
            <Link
              key={href}
              href={href}
              className="flex items-center gap-2 rounded-lg px-3 py-2 text-sm text-gray-600 hover:bg-gray-100 hover:text-gray-900 dark:text-gray-400 dark:hover:bg-gray-800 dark:hover:text-white transition"
            >
              <Icon className="h-4 w-4 shrink-0" />
              {label}
            </Link>
          ))}
        </div>
      </aside>
      {/* Content area */}
      <div className="min-w-0 flex-1">{children}</div>
    </div>
  );
}
