"use client";

import { useState } from "react";
import { useTranslations } from "@/lib/i18n";
import {
  BookOpen, Code, Terminal, ArrowRight, Zap, Globe,
  Shield, Users, KeyRound, Check, Copy,
} from "lucide-react";

type TabId = "gettingStarted" | "api" | "sdk";

export default function DocsPage() {
  const t = useTranslations();
  const [tab, setTab] = useState<TabId>("gettingStarted");

  const tabs: { id: TabId; label: string; icon: typeof BookOpen }[] = [
    { id: "gettingStarted", label: t("docs.tabs.gettingStarted"), icon: BookOpen },
    { id: "api", label: t("docs.tabs.api"), icon: Terminal },
    { id: "sdk", label: t("docs.tabs.sdk"), icon: Code },
  ];

  return (
    <div className="min-h-screen bg-gray-50 dark:bg-gray-950 p-4 md:p-8">
      <div className="max-w-4xl mx-auto">
        <div className="mb-6">
          <div className="flex items-center gap-3 mb-1">
            <BookOpen className="w-7 h-7 text-blue-600" />
            <h1 className="text-2xl font-bold text-gray-900 dark:text-white">{t("docs.title")}</h1>
          </div>
          <p className="text-gray-600 dark:text-gray-400 text-sm">{t("docs.description")}</p>
        </div>

        <div className="flex gap-1 mb-6 bg-gray-200 dark:bg-gray-800 rounded-lg p-1">
          {tabs.map(({ id, label, icon: Icon }) => (
            <button key={id} onClick={() => setTab(id)}
              className={`flex items-center gap-2 px-4 py-2 rounded-md text-sm font-medium transition-colors ${
                tab === id ? "bg-white dark:bg-gray-700 text-blue-600 dark:text-blue-400 shadow-sm" : "text-gray-600 dark:text-gray-400 hover:text-gray-900 dark:hover:text-white"
              }`}>
              <Icon className="w-4 h-4" />{label}
            </button>
          ))}
        </div>

        {tab === "gettingStarted" && <GettingStartedTab />}
        {tab === "api" && <APITab />}
        {tab === "sdk" && <SDKTab />}
      </div>
    </div>
  );
}

// ============ Getting Started ============

function GettingStartedTab() {
  const t = useTranslations();
  const steps = [
    { title: t("docs.gettingStarted.step1Title"), body: t("docs.gettingStarted.step1Body"), icon: Rocket },
    { title: t("docs.gettingStarted.step2Title"), body: t("docs.gettingStarted.step2Body"), icon: Globe },
    { title: t("docs.gettingStarted.step3Title"), body: t("docs.gettingStarted.step3Body"), icon: Users },
    { title: t("docs.gettingStarted.step4Title"), body: t("docs.gettingStarted.step4Body"), icon: Shield },
    { title: t("docs.gettingStarted.step5Title"), body: t("docs.gettingStarted.step5Body"), icon: Code },
  ];

  return (
    <div className="bg-white dark:bg-gray-900 rounded-xl border border-gray-200 dark:border-gray-800 p-6 md:p-8">
      <h2 className="text-lg font-bold text-gray-900 dark:text-white mb-2">{t("docs.gettingStarted.title")}</h2>
      <div className="flex items-center gap-2 mb-6 text-sm text-gray-500">
        <Check className="w-4 h-4 text-green-500" />
        {t("docs.gettingStarted.prerequisites")}: Docker or Go 1.22+
      </div>

      <div className="relative">
        <div className="absolute left-5 top-0 bottom-0 w-0.5 bg-gray-200 dark:bg-gray-800" />
        <div className="space-y-6">
          {steps.map((step, i) => {
            const Icon = step.icon;
            return (
              <div key={i} className="relative pl-14">
                <div className="absolute left-2 top-0 w-9 h-9 rounded-full bg-gradient-to-br from-blue-500 to-purple-600 flex items-center justify-center ring-4 ring-gray-50 dark:ring-gray-950">
                  <Icon className="w-4 h-4 text-white" />
                </div>
                <h3 className="text-sm font-bold text-gray-900 dark:text-white mb-1">{step.title}</h3>
                <div className="text-sm text-gray-600 dark:text-gray-400 whitespace-pre-line">{step.body}</div>
                {i === 0 && (
                  <div className="mt-2 p-3 rounded-lg bg-gray-900 dark:bg-gray-800 text-xs font-mono text-green-400 overflow-x-auto">
                    <span className="text-gray-500">$ </span>docker run -d -p 8080:8080 -p 3000:3000 ggid/ggid-all-in-one:latest
                  </div>
                )}
              </div>
            );
          })}
        </div>
      </div>
    </div>
  );
}

// ============ API Reference ============

function APITab() {
  const t = useTranslations();

  return (
    <div className="space-y-4">
      <div className="bg-white dark:bg-gray-900 rounded-xl border border-gray-200 dark:border-gray-800 p-6">
        <h2 className="text-lg font-bold text-gray-900 dark:text-white mb-2">{t("docs.api.title")}</h2>
        <p className="text-sm text-gray-500 mb-4">{t("docs.api.description")}</p>

        <a href="/docs/swagger" className="flex items-center gap-3 p-4 rounded-xl bg-gradient-to-r from-blue-50 to-purple-50 dark:from-blue-950/20 dark:to-purple-950/20 border border-blue-200 dark:border-blue-900 hover:border-blue-400 transition-colors mb-4">
          <div className="w-10 h-10 rounded-lg bg-blue-600 flex items-center justify-center">
            <Terminal className="w-5 h-5 text-white" />
          </div>
          <div className="flex-1">
            <h3 className="text-sm font-bold text-gray-900 dark:text-white">{t("docs.api.openSwagger")}</h3>
            <p className="text-xs text-gray-500">{t("docs.api.swaggerDesc")}</p>
          </div>
          <ArrowRight className="w-5 h-5 text-blue-600" />
        </a>

        <div className="grid grid-cols-1 md:grid-cols-2 gap-3 mb-4">
          <div className="p-3 rounded-lg bg-gray-50 dark:bg-gray-800/50">
            <span className="text-xs text-gray-500">{t("docs.api.baseUrl")}</span>
            <code className="block text-sm font-mono text-gray-900 dark:text-white mt-1">http://localhost:8080</code>
          </div>
          <div className="p-3 rounded-lg bg-gray-50 dark:bg-gray-800/50">
            <span className="text-xs text-gray-500">{t("docs.api.auth")}</span>
            <p className="text-xs text-gray-600 dark:text-gray-400 mt-1">{t("docs.api.authDesc")}</p>
          </div>
        </div>

        <div>
          <span className="text-xs font-medium text-gray-500 mb-2 block">{t("docs.api.sampleRequest")}</span>
          <div className="p-4 rounded-lg bg-gray-900 dark:bg-gray-800 text-xs font-mono overflow-x-auto">
            <div className="text-gray-500"># Login</div>
            <div className="text-green-400">curl -X POST http://localhost:8080/api/v1/auth/login \</div>
            <div className="text-green-400 ml-4">-H "Content-Type: application/json" \</div>
            <div className="text-green-400 ml-4">-d {"'\"email\":\"admin@ggid.dev\",\"password\":\"Admin@123456\"'"}</div>
            <div className="text-gray-500 mt-2"># List Users</div>
            <div className="text-blue-400">curl http://localhost:8080/api/v1/users \</div>
            <div className="text-blue-400 ml-4">-H "Authorization: Bearer {'<TOKEN>'}"</div>
          </div>
        </div>
      </div>
    </div>
  );
}

// ============ SDK Examples ============

function SDKTab() {
  const t = useTranslations();
  const [active, setActive] = useState<"go" | "react" | "python" | "curl">("go");
  const [copied, setCopied] = useState(false);

  const sdks: { id: typeof active; label: string; desc: string; icon: typeof Code }[] = [
    { id: "go", label: t("docs.sdk.go"), desc: t("docs.sdk.goDesc"), icon: Code },
    { id: "react", label: t("docs.sdk.react"), desc: t("docs.sdk.reactDesc"), icon: Globe },
    { id: "python", label: t("docs.sdk.python"), desc: t("docs.sdk.pythonDesc"), icon: KeyRound },
    { id: "curl", label: t("docs.sdk.curl"), desc: t("docs.sdk.curlDesc"), icon: Terminal },
  ];

  const codeExamples: Record<string, string> = {
    go: `package main

import ggid "github.com/ggid/ggid/sdk/go"

func main() {
    client := ggid.NewClient("http://localhost:8080")
    token, _ := client.Login(ctx, &ggid.LoginRequest{
        Email: "admin@ggid.dev",
        Password: "Admin@123456",
    })
    user, _ := client.GetUser(ctx, token.UserID, token.AccessToken)
    fmt.Println(user.Email)
}`,
    react: `import { GGIDProvider, useGGIDAuth } from "@ggid/react";

function App() {
  return (
    <GGIDProvider baseUrl="http://localhost:8080">
      <Dashboard />
    </GGIDProvider>
  );
}

function Dashboard() {
  const { user, login } = useGGIDAuth();
  // user is null until login completes
}`,
    python: `from ggid import GGIDClient

client = GGIDClient(base_url="http://localhost:8080")
token = client.login(email="admin@ggid.dev", password="Admin@123456")

users = client.list_users(access_token=token["access_token"])
for u in users:
    print(u["email"])`,
    curl: `# Login
TOKEN=$(curl -s -X POST http://localhost:8080/api/v1/auth/login \\
  -H "Content-Type: application/json" \\
  -d '{"email":"admin@ggid.dev","password":"Admin@123456"}' \\
  | jq -r '.access_token')

# List users
curl -s http://localhost:8080/api/v1/users \\
  -H "Authorization: Bearer $TOKEN" | jq`,
  };

  const copy = () => {
    navigator.clipboard.writeText(codeExamples[active]);
    setCopied(true);
    setTimeout(() => setCopied(false), 2000);
  };

  return (
    <div className="space-y-4">
      <div className="bg-white dark:bg-gray-900 rounded-xl border border-gray-200 dark:border-gray-800 p-6">
        <h2 className="text-lg font-bold text-gray-900 dark:text-white mb-2">{t("docs.sdk.title")}</h2>
        <p className="text-sm text-gray-500 mb-4">{t("docs.sdk.description")}</p>

        {/* SDK selector */}
        <div className="grid grid-cols-2 md:grid-cols-4 gap-2 mb-4">
          {sdks.map((s) => {
            const Icon = s.icon;
            const isActive = active === s.id;
            return (
              <button key={s.id} onClick={() => setActive(s.id)}
                className={`flex flex-col items-start gap-1 p-3 rounded-lg border-2 text-left transition-all ${
                  isActive ? "border-blue-500 bg-blue-50 dark:bg-blue-950/20" : "border-gray-200 dark:border-gray-700 hover:border-gray-300"
                }`}>
                <Icon className={`w-5 h-5 ${isActive ? "text-blue-600" : "text-gray-400"}`} />
                <span className="text-sm font-bold text-gray-900 dark:text-white">{s.label}</span>
                <span className="text-xs text-gray-400">{s.desc}</span>
              </button>
            );
          })}
        </div>

        {/* Code block */}
        <div className="relative">
          <button onClick={copy} className="absolute right-3 top-3 p-1.5 rounded bg-gray-700 hover:bg-gray-600 text-gray-300 text-xs">
            {copied ? <Check className="w-3.5 h-3.5" /> : <Copy className="w-3.5 h-3.5" />}
          </button>
          <pre className="p-4 rounded-lg bg-gray-900 dark:bg-gray-800 text-xs font-mono text-gray-300 overflow-x-auto max-h-96">
            {codeExamples[active]}
          </pre>
        </div>

        {/* Full example link */}
        <a href={`https://github.com/topcheer/ggid/tree/main/sdk/${active === "curl" ? "curl" : active}/examples`} target="_blank" rel="noopener"
          className="mt-3 inline-flex items-center gap-1 text-sm text-blue-600 hover:underline">
          {t("docs.sdk.viewFullExample")} <ArrowRight className="w-3 h-3" />
        </a>
      </div>
    </div>
  );
}
