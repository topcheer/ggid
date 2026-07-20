import type { Metadata, Viewport } from "next";
import "./globals.css";
import { AuthGuard } from "@/components/auth-guard";
import { ImpersonationBanner } from "@/components/impersonation-banner";
import { ThemeProvider } from "@/lib/theme";
import { I18nProvider } from "@/lib/i18n";
import { ToastProvider } from "@/components/Toast";
import { ConfirmProvider } from "@/components/ConfirmDialog";
import PWARegister from "@/components/PWARegister";

export const metadata: Metadata = {
  title: {
    default: "GGID Console",
    template: "%s | GGID Console",
  },
  description: "GGID Identity & Access Management Console",
  manifest: "/manifest.json",
  icons: {
    icon: [
      { url: "/favicon.ico", sizes: "any" },
      { url: "/icon.svg", type: "image/svg+xml" },
    ],
    apple: "/apple-icon.png",
  },
  appleWebApp: {
    capable: true,
    title: "GGID Console",
    statusBarStyle: "default",
  },
};

export const viewport: Viewport = {
  themeColor: "#4f46e5",
  width: "device-width",
  initialScale: 1,
  maximumScale: 5,
};

export default function RootLayout({
  children,
}: {
  children: React.ReactNode;
}) {
  return (
    <html lang="en" suppressHydrationWarning>
      <head>
        <script dangerouslySetInnerHTML={{
          __html: `(function(){try{var d=localStorage.getItem('darkMode');var m=window.matchMedia('(prefers-color-scheme: dark)').matches;if(d==='dark'||((!d||d==='system')&&m)){document.documentElement.classList.add('dark')}}catch(e){}})()`,
        }} />
      </head>
      <body>
        <PWARegister />
        <a href="#main-content" className="skip-link">Skip to content</a>
        <ThemeProvider>
          <I18nProvider>
            <ToastProvider>
              <ConfirmProvider>
                <AuthGuard>
                  <ImpersonationBanner />
                  {children}
                </AuthGuard>
              </ConfirmProvider>
            </ToastProvider>
          </I18nProvider>
        </ThemeProvider>
      </body>
    </html>
  );
}
