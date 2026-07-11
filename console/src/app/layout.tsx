import type { Metadata, Viewport } from "next";
import "./globals.css";
import { Sidebar } from "@/components/sidebar";
import { AuthGuard } from "@/components/auth-guard";
import { ThemeProvider } from "@/lib/theme";
import { I18nProvider } from "@/lib/i18n";
import { ToastProvider } from "@/components/Toast";
import PWARegister from "@/components/PWARegister";

export const metadata: Metadata = {
  title: "GGID Console",
  description: "GGID Identity & Access Management Console",
  manifest: "/manifest.json",
  appleWebApp: {
    capable: true,
    title: "GGID Console",
    statusBarStyle: "default",
  },
};

export const viewport: Viewport = {
  themeColor: "#4f46e5",
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
        <ThemeProvider>
          <I18nProvider>
            <ToastProvider>
            <AuthGuard>
              <div className="flex h-screen dark:bg-gray-950">
                <Sidebar />
                <main className="flex-1 overflow-auto">
                  <div className="p-4 md:p-6">{children}</div>
                </main>
              </div>
            </AuthGuard>
            </ToastProvider>
          </I18nProvider>
        </ThemeProvider>
      </body>
    </html>
  );
}
