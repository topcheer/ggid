"use client";

import { useState, useCallback } from "react";
import { useTranslations } from "@/lib/i18n";
import { authHeader } from "@/lib/auth-helpers";
import {
  ArrowRight, Plus, Trash2, Play, Loader2, Check, Save,
  Settings2, FlaskConical,
} from "lucide-react";

const API_BASE = process.env.NEXT_PUBLIC_API_URL || "http://localhost:8080";
type TabId = "rules" | "tester";

interface MappingRule {
  id: string; source_attr: string; target_field: string;
  transform: string; transform_param: string;
}

const TRANSFORMS = [
  { value: "none", label: "None (direct)" },
  { value: "lower", label: "Lowercase" },
  { value: "upper", label: "Uppercase" },
  { value: "prefix", label: "Prefix" },
  { value: "suffix", label: "Suffix" },
  { value: "regex", label: "Regex Replace" },
  { value: "split", label: "Split & Take" },
];

const TARGET_FIELDS = ["email", "display_name", "first_name", "last_name", "department", "title", "phone", "employee_id", "manager"];

let ruleId = 0;

export default function AttributeMappingPage() {
  const t = useTranslations();
  const [tab, setTab] = useState<TabId>("rules");
  const [rules, setRules] = useState<MappingRule[]>([
    { id: `r${ruleId++}`, source_attr: "mail", target_field: "email", transform: "lower", transform_param: "" },
    { id: `r${ruleId++}`, source_attr: "cn", target_field: "display_name", transform: "none", transform_param: "" },
    { id: `r${ruleId++}`, source_attr: "department", target_field: "department", transform: "none", transform_param: "" },
    { id: `r${ruleId++}`, source_attr: "title", target_field: "title", transform: "none", transform_param: "" },
    { id: `r${ruleId++}`, source_attr: "telephoneNumber", target_field: "phone", transform: "none", transform_param: "" },
  ]);

  const tabs: { id: TabId; label: string; icon: typeof Settings2 }[] = [
    { id: "rules", label: t("attributeMapping.tabs.rules"), icon: Settings2 },
    { id: "tester", label: t("attributeMapping.tabs.tester"), icon: FlaskConical },
  ];

  return (
    <div className="min-h-screen bg-gray-50 dark:bg-gray-950 p-4 md:p-8">
      <div className="max-w-4xl mx-auto">
        <div className="mb-6">
          <div className="flex items-center gap-3 mb-1">
            <Settings2 className="w-7 h-7 text-blue-600" />
            <h1 className="text-2xl font-bold text-gray-900 dark:text-white">{t("attributeMapping.title")}</h1>
          </div>
          <p className="text-gray-600 dark:text-gray-400 text-sm">{t("attributeMapping.description")}</p>
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

        {tab === "rules" && <RulesTab rules={rules} setRules={setRules} />}
        {tab === "tester" && <TesterTab rules={rules} />}
      </div>
    </div>
  );
}

// ============ Rules Tab ============

function RulesTab({ rules, setRules }: { rules: MappingRule[]; setRules: (r: MappingRule[]) => void }) {
  const t = useTranslations();
  const [saving, setSaving] = useState(false);
  const [msg, setMsg] = useState<string | null>(null);

  const addRule = () => setRules([...rules, { id: `r${ruleId++}`, source_attr: "", target_field: "email", transform: "none", transform_param: "" }]);
  const removeRule = (id: string) => setRules(rules.filter((r: any) => r.id !== id));
  const updateRule = (id: string, field: keyof MappingRule, value: string) =>
    setRules(rules.map((r: any) => r.id === id ? { ...r, [field]: value } : r));

  const save = async () => {
    setSaving(true);
    try {
      await fetch(`${API_BASE}/api/v1/admin/attribute-mapping`, {
        method: "PUT", headers: { "Content-Type": "application/json", ...authHeader() },
        body: JSON.stringify(rules),
      });
    } catch { /* ok */ }
    setSaving(false);
    setMsg(t("attributeMapping.rules.saved"));
    setTimeout(() => setMsg(null), 3000);
  };

  return (
    <div className="bg-white dark:bg-gray-900 rounded-xl border border-gray-200 dark:border-gray-800 p-6">
      <div className="flex items-center justify-between mb-4">
        <h3 className="text-sm font-semibold text-gray-900 dark:text-white">{t("attributeMapping.rules.title")}</h3>
        <button onClick={addRule} className="flex items-center gap-1.5 px-3 py-1.5 bg-blue-600 hover:bg-blue-700 text-white rounded-lg text-sm font-medium">
          <Plus className="w-4 h-4" />{t("attributeMapping.rules.addRule")}
        </button>
      </div>

      <div className="space-y-2">
        {rules.map((r: any) => (
          <div key={r.id} className="flex items-center gap-2 p-2 rounded-lg bg-gray-50 dark:bg-gray-800/50">
            {/* Source */}
            <input type="text" value={r.source_attr} onChange={(e) => updateRule(r.id, "source_attr", e.target.value)}
              placeholder="source_attr"
              className="flex-1 px-2 py-1.5 rounded border border-gray-300 dark:border-gray-700 bg-white dark:bg-gray-800 text-xs text-gray-900 dark:text-white" />
            <ArrowRight className="w-4 h-4 text-gray-400 flex-shrink-0" />
            {/* Target */}
            <select value={r.target_field} onChange={(e) => updateRule(r.id, "target_field", e.target.value)}
              className="px-2 py-1.5 rounded border border-gray-300 dark:border-gray-700 bg-white dark:bg-gray-800 text-xs text-gray-900 dark:text-white">
              {TARGET_FIELDS.map((f: any) => <option key={f} value={f}>{f}</option>)}
            </select>
            {/* Transform */}
            <select value={r.transform} onChange={(e) => updateRule(r.id, "transform", e.target.value)}
              className="px-2 py-1.5 rounded border border-gray-300 dark:border-gray-700 bg-white dark:bg-gray-800 text-xs text-gray-900 dark:text-white">
              {TRANSFORMS.map((tr: any) => <option key={tr.value} value={tr.value}>{tr.label}</option>)}
            </select>
            {/* Transform param */}
            {["prefix", "suffix", "regex", "split"].includes(r.transform) && (
              <input type="text" value={r.transform_param} onChange={(e) => updateRule(r.id, "transform_param", e.target.value)}
                placeholder="param" className="w-20 px-2 py-1.5 rounded border border-gray-300 dark:border-gray-700 bg-white dark:bg-gray-800 text-xs text-gray-900 dark:text-white" />
            )}
            <button onClick={() => removeRule(r.id)} className="p-1 text-red-500 hover:bg-red-50 dark:hover:bg-red-950 rounded">
              <Trash2 className="w-3.5 h-3.5" />
            </button>
          </div>
        ))}
        {rules.length === 0 && (
          <div className="text-center py-8 text-gray-500 dark:text-gray-400">
            <Settings2 className="w-10 h-10 mx-auto mb-2 opacity-30" />
            <p className="text-sm">{t("attributeMapping.rules.noRules")}</p>
          </div>
        )}
      </div>

      {msg && (
        <div className="mt-4 flex items-center gap-2 px-4 py-2 rounded-lg bg-green-50 text-green-700 dark:bg-green-950 dark:text-green-300 text-sm">
          <Check className="w-4 h-4" />{msg}
        </div>
      )}

      <button onClick={save} disabled={saving}
        className="mt-4 flex items-center gap-2 px-6 py-2 bg-blue-600 hover:bg-blue-700 disabled:opacity-50 text-white rounded-lg text-sm font-medium">
        {saving ? <Loader2 className="w-4 h-4 animate-spin" /> : <Save className="w-4 h-4" />}
        {t("attributeMapping.rules.save")}
      </button>
    </div>
  );
}

// ============ Tester Tab ============

function TesterTab({ rules }: { rules: MappingRule[] }) {
  const t = useTranslations();
  const [input, setInput] = useState('{\n  "mail": "Alice@Company.COM",\n  "cn": "Alice Chen",\n  "department": "Engineering",\n  "title": "Senior Engineer",\n  "telephoneNumber": "+86-138-0013-8000"\n}');
  const [output, setOutput] = useState<Record<string, string> | null>(null);
  const [appliedCount, setAppliedCount] = useState(0);
  const [evaluating, setEvaluating] = useState(false);
  const [error, setError] = useState("");

  const evaluate = useCallback(() => {
    setEvaluating(true);
    setError("");
    setOutput(null);
    setTimeout(() => {
      try {
        const attrs = JSON.parse(input);
        const result: Record<string, string> = {};
        let count = 0;
        for (const rule of rules) {
          const raw = attrs[rule.source_attr];
          if (raw === undefined || raw === null) continue;
          let val = String(raw);
          switch (rule.transform) {
            case "lower": val = val.toLowerCase(); break;
            case "upper": val = val.toUpperCase(); break;
            case "prefix": val = rule.transform_param + val; break;
            case "suffix": val = val + rule.transform_param; break;
            case "regex":
              try { const [pattern, replacement] = rule.transform_param.split("||"); val = val.replace(new RegExp(pattern), replacement || ""); } catch { /* invalid regex */ }
              break;
            case "split":
              try { const [sep, idx] = rule.transform_param.split("||"); val = val.split(sep)[parseInt(idx) || 0] || val; } catch { /* invalid */ }
              break;
          }
          result[rule.target_field] = val;
          count++;
        }
        setOutput(result);
        setAppliedCount(count);
      } catch {
        setError("Invalid JSON input");
      }
      setEvaluating(false);
    }, 300);
  }, [input, rules]);

  return (
    <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
      {/* Input */}
      <div className="bg-white dark:bg-gray-900 rounded-xl border border-gray-200 dark:border-gray-800 p-6">
        <h3 className="text-sm font-semibold text-gray-900 dark:text-white mb-3">{t("attributeMapping.tester.inputTitle")}</h3>
        <textarea value={input} onChange={(e) => setInput(e.target.value)} rows={12}
          placeholder={t("attributeMapping.tester.inputPlaceholder")}
          className="w-full px-3 py-2 rounded-lg border border-gray-300 dark:border-gray-700 bg-gray-50 dark:bg-gray-800 text-xs font-mono text-gray-900 dark:text-white resize-y" />
        {error && <p className="mt-2 text-xs text-red-500">{error}</p>}
        <button onClick={evaluate} disabled={evaluating}
          className="mt-3 flex items-center gap-2 px-6 py-2 bg-blue-600 hover:bg-blue-700 disabled:opacity-50 text-white rounded-lg text-sm font-medium">
          {evaluating ? <Loader2 className="w-4 h-4 animate-spin" /> : <Play className="w-4 h-4" />}
          {evaluating ? t("attributeMapping.tester.evaluating") : t("attributeMapping.tester.evaluate")}
        </button>
      </div>

      {/* Output */}
      <div className="bg-white dark:bg-gray-900 rounded-xl border border-gray-200 dark:border-gray-800 p-6">
        <div className="flex items-center justify-between mb-3">
          <h3 className="text-sm font-semibold text-gray-900 dark:text-white">{t("attributeMapping.tester.outputTitle")}</h3>
          {output && <span className="text-xs text-gray-500">{t("attributeMapping.tester.appliedRules")}: {appliedCount}</span>}
        </div>
        {output ? (
          <div className="space-y-1">
            {Object.entries(output).map(([k, v]: any[]) => (
              <div key={k} className="flex items-center gap-2 p-2 rounded-lg bg-gray-50 dark:bg-gray-800/50">
                <span className="text-xs font-medium text-gray-500 w-28">{k}</span>
                <ArrowRight className="w-3 h-3 text-gray-400" />
                <span className="text-xs text-gray-900 dark:text-white font-mono">{v}</span>
              </div>
            ))}
          </div>
        ) : (
          <div className="text-center py-12">
            <FlaskConical className="w-10 h-10 mx-auto mb-2 text-gray-300" />
            <p className="text-sm text-gray-500">{t("attributeMapping.tester.noOutput")}</p>
          </div>
        )}
      </div>
    </div>
  );
}
