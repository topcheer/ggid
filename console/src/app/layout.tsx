import type { Metadata } from "next";
import "./globals.css";
import { Sidebar } from "@/components/sidebar";
import { AuthGuard } from "@/components/auth-guard";
import { ThemeProvider } from "@/lib/theme";
import { I18nProvider } from "@/lib/i18n";

export const metadata: Metadata = {
  title: "GGID Console",
  description: "GGID Identity & Access Management Console",
};

export default function RootLayout({
  children,
}: {
  children: React.ReactNode;
}) {
  return (
    <html lang="en" suppressHydrationWarning>
      <body>
        <ThemeProvider>
          <I18nProvider>
            <AuthGuard>
              <div className="flex h-screen">
                <Sidebar />
                <main className="flex-1 overflow-auto">
                  <div className="p-6">{children}</div>
                </main>
              </div>
            </AuthGuard>
          </I18nProvider>
        </ThemeProvider>
      </body>
    </html>
  );
}
