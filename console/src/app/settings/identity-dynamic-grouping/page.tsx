"use client";

import { useState } from "react";
import { useIdentityDynamicGrouping } from "@ggid/sdk-react";
import { Users, Layers, GitBranch, Play, Eye } from "lucide-react";
import { useTranslations } from "@/lib/i18n";

export default function IdentityDynamicGroupingPage() {
  const { data, loading, error, refresh, evaluatePreview } = useIdentityDynamicGrouping();
  const [previewGroup, setPreviewGroup] = useState("");
  const t = useTranslations();

  if (loading) return <div className="p-8 text-gray-400">{t("idDynamicGrouping.loading")}</div>;
  if (error) return <div className="p-8 text-red-400">Error: {error}</div>;

  const group = previewGroup
    ? (data?.group_rules ?? []).find((g: any) => g.group_name === previewGroup)
    : (data?.group_rules ?? [])[0];

  return (
    <div className="min-h-screen bg-gray-950 text-white p-8">
      <div className="flex items-center justify-between mb-8">
        <div>
          <h1 className="text-2xl font-bold">{t("idDynamicGrouping.title")}</h1>
          <p className="text-sm text-gray-400 mt-1">{t("idDynamicGrouping.subtitle")}</p>
        </div>
        <button
          onClick={refresh}
          className="px-4 py-2 bg-blue-600 hover:bg-blue-700 rounded-lg text-sm font-medium transition"
         aria-label="Action">
          {t("idDynamicGrouping.refresh")}
        </button>
      </div>

      {/* Summary */}
      <div className="grid grid-cols-1 md:grid-cols-4 gap-4 mb-6">
        <div className="bg-gray-900 rounded-xl p-4">
          <div className="flex items-center gap-2 mb-1 text-blue-400">
            <Layers className="w-4 h-4" />
            <span className="text-xs text-gray-400">{t("idDynamicGrouping.totalGroups")}</span>
          </div>
          <p className="text-2xl font-bold">{data?.group_rules?.length ?? 0}</p>
        </div>
        <div className="bg-gray-900 rounded-xl p-4">
          <div className="flex items-center gap-2 mb-1 text-green-400">
            <Users className="w-4 h-4" />
            <span className="text-xs text-gray-400">{t("idDynamicGrouping.dynamicMembers")}</span>
          </div>
          <p className="text-2xl font-bold">{(data?.group_rules ?? []).reduce((a: any, g: any) => a + g.member_count, 0)}</p>
        </div>
        <div className="bg-gray-900 rounded-xl p-4">
          <div className="flex items-center gap-2 mb-1 text-purple-400">
            <Play className="w-4 h-4" />
            <span className="text-xs text-gray-400">{t("idDynamicGrouping.evaluationFreq")}</span>
          </div>
          <p className="text-lg font-bold capitalize">{data?.evaluation_frequency ?? "real-time"}</p>
        </div>
        <div className="bg-gray-900 rounded-xl p-4">
          <div className="flex items-center gap-2 mb-1 text-yellow-400">
            <GitBranch className="w-4 h-4" />
            <span className="text-xs text-gray-400">{t("idDynamicGrouping.conflictResolution")}</span>
          </div>
          <p className="text-lg font-bold capitalize">{data?.conflict_resolution?.replace(/_/g, " ") ?? "priority"}</p>
        </div>
      </div>

      <div className="grid grid-cols-1 lg:grid-cols-2 gap-6">
        {/* Group Rules Table */}
        <div className="bg-gray-900 rounded-xl p-6">
          <h2 className="text-lg font-semibold mb-4">{t("idDynamicGrouping.groupRules")}</h2>
          <div className="space-y-2">
            {(data?.group_rules ?? []).map((g: any) => (
              <div
                key={g.group_name}
                className={"bg-gray-800 rounded-lg p-3 cursor-pointer transition " + (previewGroup === g.group_name ? "border border-blue-500" : "hover:border hover:border-gray-600")}
                onClick={() => { setPreviewGroup(g.group_name); evaluatePreview(g.group_name); }}
              >
                <div className="flex items-center justify-between mb-1">
                  <p className="text-sm font-medium">{g.group_name}</p>
                  <span
                    className={"text-xs px-2 py-0.5 rounded " + (
                      g.membership_type === "dynamic" ? "bg-green-900 text-green-300" :
                      g.membership_type === "hybrid" ? "bg-yellow-900 text-yellow-300" :
                      "bg-gray-700 text-gray-300"
                    )}
                  >
                    {g.membership_type}
                  </span>
                </div>
                <p className="text-xs text-gray-400 font-mono">{g.rule_expression}</p>
                <p className="text-xs text-gray-500 mt-1">{g.member_count} members</p>
              </div>
            ))}
          </div>
        </div>

        {/* Live Membership Preview */}
        <div className="bg-gray-900 rounded-xl p-6">
          <h2 className="text-lg font-semibold flex items-center gap-2 mb-4">
            <Eye className="w-5 h-5 text-blue-400" />
            Live Membership Preview
          </h2>
          {group && (
            <>
              <div className="bg-gray-800 rounded-lg p-3 mb-3">
                <p className="text-sm font-medium mb-1">{group.group_name}</p>
                <code className="text-xs text-gray-400 font-mono block mb-2">{group.rule_expression}</code>
              </div>
              <div className="bg-gray-800 rounded-lg p-3">
                <p className="text-xs text-gray-400 mb-2">{t("idDynamicGrouping.matchedMembers")} ({group.member_count}):</p>
                <div className="space-y-1 max-h-48 overflow-y-auto">
                  {(group.preview_members ?? []).map((m: any, i: number) => (
                    <div key={i} className="flex items-center gap-2 bg-gray-900 rounded p-2">
                      <Users className="w-3 h-3 text-gray-400" />
                      <span className="text-xs font-medium">{m.username}</span>
                      <span className="text-xs text-gray-500 ml-auto">{m.matched_attribute}</span>
                    </div>
                  ))}
                </div>
              </div>
            </>
          )}
        </div>
      </div>

      {/* Rule Builder */}
      <div className="bg-gray-900 rounded-xl p-6 mt-6">
        <h2 className="text-lg font-semibold mb-4">{t("idDynamicGrouping.ruleBuilder")}</h2>
        <div className="flex items-center gap-2 flex-wrap">
          <select aria-label="Select option" className="text-xs bg-gray-800 border border-gray-700 rounded-lg px-2 py-1.5">
            <option>department</option>
            <option>title</option>
            <option>location</option>
            <option>manager</option>
            <option>cost_center</option>
          </select>
          <select aria-label="Select option" className="text-xs bg-gray-800 border border-gray-700 rounded-lg px-2 py-1.5">
            <option>equals</option>
            <option>contains</option>
            <option>in</option>
            <option>not_equals</option>
          </select>
          <input
            type="text"
            placeholder="value"
            className="text-xs bg-gray-800 border border-gray-700 rounded-lg px-2 py-1.5 w-32"
          />
          <button className="text-xs px-3 py-1.5 bg-blue-600 hover:bg-blue-700 rounded-lg font-medium">+ AND</button>
        </div>
      </div>
    </div>
  );
}
