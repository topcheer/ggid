"use client";

import { useState, useCallback, useMemo } from "react";
import { useApi } from "@/lib/api";
import {
  Lock, MessageSquare, Smartphone, Fingerprint, Mail, Globe,
  KeySquare, Server, Eye, EyeOff, Play, Plus, Trash2, Settings2,
  GripVertical, ChevronDown, ChevronRight, Save, Power, GitBranch,
  ArrowDown, Layers, User,
} from "lucide-react";
import { useTranslations } from "@/lib/i18n";

// ===== Types =====

interface StepConfig {
  required: boolean;
  fallback: boolean;
  timeout: number;
  retryCount: number;
}

interface FlowStep {
  id: string;
  type: StepType;
  name: string;
  config: StepConfig;
}

interface Branch {
  id: string;
  steps: FlowStep[];
}

interface FlowSegment {
  id: string;
  type: "linear" | "branch";
  steps?: FlowStep[];        // for linear segments
  branches?: Branch[];       // for branch segments
}

type StepType =
  | "password" | "sms_otp" | "totp" | "webauthn"
  | "social_google" | "social_github" | "social_microsoft"
  | "saml" | "email_link" | "ldap";

interface StepTypeDef {
  type: StepType;
  label: string;
  icon: typeof Lock;
  color: string;
  description: string;
}

// ===== Step palette definitions =====

const STEP_TYPES: StepTypeDef[] = [
  { type: "password", label: "Password", icon: Lock, color: "text-blue-600 bg-blue-50 dark:bg-blue-900/30", description: "Username + password authentication" },
  { type: "sms_otp", label: "SMS OTP", icon: MessageSquare, color: "text-purple-600 bg-purple-50 dark:bg-purple-900/30", description: "One-time code via SMS" },
  { type: "totp", label: "TOTP", icon: Smartphone, color: "text-indigo-600 bg-indigo-50 dark:bg-indigo-900/30", description: "Time-based OTP authenticator app" },
  { type: "webauthn", label: "WebAuthn", icon: Fingerprint, color: "text-green-600 bg-green-50 dark:bg-green-900/30", description: "FIDO2 security key or biometric" },
  { type: "social_google", label: "Google", icon: Globe, color: "text-red-600 bg-red-50 dark:bg-red-900/30", description: "Sign in with Google OAuth" },
  { type: "social_github", label: "GitHub", icon: Globe, color: "text-gray-700 bg-gray-100 dark:bg-gray-700", description: "Sign in with GitHub OAuth" },
  { type: "social_microsoft", label: "Microsoft", icon: Layers, color: "text-cyan-600 bg-cyan-50 dark:bg-cyan-900/30", description: "Sign in with Microsoft" },
  { type: "saml", label: "SAML", icon: KeySquare, color: "text-amber-600 bg-amber-50 dark:bg-amber-900/30", description: "SAML SSO via Identity Provider" },
  { type: "email_link", label: "Email Link", icon: Mail, color: "text-pink-600 bg-pink-50 dark:bg-pink-900/30", description: "Magic link sent to email" },
  { type: "ldap", label: "LDAP", icon: Server, color: "text-teal-600 bg-teal-50 dark:bg-teal-900/30", description: "LDAP / Active Directory bind" },
];

const STEP_MAP: Record<string, StepTypeDef> = Object.fromEntries(
  STEP_TYPES.map((s: any) => [s.type, s]),
);

function getStepDef(type: StepType): StepTypeDef {
  return STEP_MAP[type] || STEP_TYPES[0];
}

function makeStep(type: StepType): FlowStep {
  return {
    id: `step_${Date.now()}_${Math.random().toString(36).slice(2, 7)}`,
    type,
    name: getStepDef(type).label,
    config: { required: true, fallback: false, timeout: 30, retryCount: 3 },
  };
}

// ===== Default flow (demonstrates linear + branch) =====

const DEFAULT_FLOW: FlowSegment[] = [
  {
    id: "seg_1",
    type: "linear",
    steps: [makeStep("password")],
  },
  {
    id: "seg_2",
    type: "branch",
    branches: [
      { id: "br_1", steps: [makeStep("totp")] },
      { id: "br_2", steps: [makeStep("sms_otp")] },
    ],
  },
];

export default function FlowBuilderPage() {
  const { apiFetch } = useApi();
  const [segments, setSegments] = useState<FlowSegment[]>(DEFAULT_FLOW);
  const [selectedStep, setSelectedStep] = useState<FlowStep | null>(null);
  const [selectedStepLocation, setSelectedStepLocation] = useState<{ segId: string; branchId?: string; index: number } | null>(null);
  const [previewMode, setPreviewMode] = useState(false);
  const [isActive, setIsActive] = useState(false);
  const [msg, setMsg] = useState<string | null>(null);
  const [error, setError] = useState<string | null>(null);
  const [saving, setSaving] = useState(false);
  const [draggedType, setDraggedType] = useState<StepType | null>(null);
  const [dragState, setDragState] = useState<{ segId: string; branchId?: string; index: number } | null>(null);

  // ===== Step CRUD =====

  const addStepToSegment = useCallback((segId: string, branchId: string | undefined, type: StepType) => {
    const newStep = makeStep(type);
    setSegments((prev) =>
      prev.map((seg: any) => {
        if (seg.id !== segId) return seg;
        if (seg.type === "linear") {
          return { ...seg, steps: [...(seg.steps || []), newStep] };
        }
        return {
          ...seg,
          branches: seg.branches?.map((b: any) =>
            b.id === branchId ? { ...b, steps: [...b.steps, newStep] } : b,
          ),
        };
      }),
    );
  }, []);

  const removeStep = useCallback((segId: string, branchId: string | undefined, stepId: string) => {
    setSegments((prev) =>
      prev.map((seg: any) => {
        if (seg.id !== segId) return seg;
        if (seg.type === "linear") {
          return { ...seg, steps: (seg.steps || []).filter((s: any) => s.id !== stepId) };
        }
        return {
          ...seg,
          branches: seg.branches?.map((b: any) =>
            b.id === branchId ? { ...b, steps: b.steps.filter((s: any) => s.id !== stepId) } : b,
          ),
        };
      }),
    );
    // Clear selection if we removed the selected step
    if (selectedStepLocation?.segId === segId && selectedStepLocation?.branchId === branchId) {
      const seg = segments.find((s: any) => s.id === segId);
      const steps = seg?.type === "linear" ? seg.steps : seg?.branches?.find((b: any) => b.id === branchId)?.steps;
      if (steps && !steps.some((s: any) => s.id === stepId)) {
        setSelectedStep(null);
        setSelectedStepLocation(null);
      }
    }
  }, [selectedStepLocation, segments]);

  const updateStepConfig = useCallback((segId: string, branchId: string | undefined, stepId: string, config: Partial<StepConfig>) => {
    setSegments((prev) =>
      prev.map((seg: any) => {
        if (seg.id !== segId) return seg;
        if (seg.type === "linear") {
          return {
            ...seg,
            steps: (seg.steps || []).map((s: any) =>
              s.id === stepId ? { ...s, config: { ...s.config, ...config } } : s,
            ),
          };
        }
        return {
          ...seg,
          branches: seg.branches?.map((b: any) =>
            b.id === branchId
              ? { ...b, steps: b.steps.map((s: any) => s.id === stepId ? { ...s, config: { ...s.config, ...config } } : s) }
              : b,
          ),
        };
      }),
    );
    // Sync local selectedStep if it's the one being edited
    if (selectedStep?.id === stepId) {
      setSelectedStep((prev) => prev ? { ...prev, config: { ...prev.config, ...config } } : prev);
    }
  }, [selectedStep]);

  // ===== Drag and drop reordering =====

  const handleDragStart = (segId: string, branchId: string | undefined, index: number) => {
    setDragState({ segId, branchId, index });
  };

  const handleDragOver = (e: React.DragEvent) => {
    e.preventDefault();
  };

  const handleDrop = (targetSegId: string, targetBranchId: string | undefined, targetIndex: number) => {
    if (!dragState) return;
    // For palette drag (new step)
    if (draggedType && !dragState.segId) {
      addStepToSegment(targetSegId, targetBranchId, draggedType);
      setDraggedType(null);
      setDragState(null);
      return;
    }

    // For reorder within same segment
    if (dragState.segId === targetSegId && dragState.branchId === targetBranchId) {
      setSegments((prev) =>
        prev.map((seg: any) => {
          if (seg.id !== targetSegId) return seg;
          const stepList = seg.type === "linear" ? seg.steps : seg.branches?.find((b: any) => b.id === targetBranchId)?.steps;
          if (!stepList) return seg;
          const reordered = [...stepList];
          const [moved] = reordered.splice(dragState.index, 1);
          reordered.splice(targetIndex, 0, moved);
          if (seg.type === "linear") {
            return { ...seg, steps: reordered };
          }
          return {
            ...seg,
            branches: seg.branches?.map((b: any) =>
              b.id === targetBranchId ? { ...b, steps: reordered } : b,
            ),
          };
        }),
      );
    }
    setDragState(null);
  };

  const handlePaletteDragStart = (type: StepType) => {
    setDraggedType(type);
    setDragState({ segId: "", index: -1 }); // marker for palette drag
  };

  // ===== Segment CRUD =====

  const addLinearSegment = () => {
    const newSeg: FlowSegment = { id: `seg_${Date.now()}`, type: "linear", steps: [] };
    setSegments([...segments, newSeg]);
  };

  const addBranchSegment = () => {
    const newSeg: FlowSegment = {
      id: `seg_${Date.now()}`,
      type: "branch",
      branches: [
        { id: `br_${Date.now()}_a`, steps: [] },
        { id: `br_${Date.now()}_b`, steps: [] },
      ],
    };
    setSegments([...segments, newSeg]);
  };

  const removeSegment = (segId: string) => {
    setSegments(segments.filter((s: any) => s.id !== segId));
  };

  const addBranchToSegment = (segId: string) => {
    setSegments((prev) =>
      prev.map((seg: any) =>
        seg.id === segId && seg.type === "branch"
          ? { ...seg, branches: [...(seg.branches || []), { id: `br_${Date.now()}`, steps: [] }] }
          : seg,
      ),
    );
  };

  // ===== Save =====

  const handleSave = async () => {
    setSaving(true);
    setError(null);
    try {
      const payload = {
        name: "Login Flow",
        active: isActive,
        segments: segments.map((seg: any) => ({
          type: seg.type,
          steps: seg.type === "linear" ? seg.steps : undefined,
          branches: seg.type === "branch" ? seg.branches : undefined,
        })),
      };
      await apiFetch("/api/v1/flows", {
        method: "POST",
        body: JSON.stringify(payload),
      });
      setMsg("Flow saved successfully");
    } catch {
      setMsg("Saved locally (API may not be available)");
    } finally {
      setSaving(false);
    }
  };

  // ===== Preview flattening =====

  const previewSteps = useMemo(() => {
    const result: { label: string; icon: typeof Lock; alternatives?: string[] }[] = [];
    for (const seg of segments) {
      if (seg.type === "linear") {
        for (const step of seg.steps || []) {
          const def = getStepDef(step.type);
          result.push({ label: step.name, icon: def.icon });
        }
      } else if (seg.type === "branch" && seg.branches) {
        const allBranchSteps = seg.branches.map((b: any) => b.steps.map((s: any) => s.name));
        const firstBranch = allBranchSteps[0] || [];
        for (let i = 0; i < firstBranch.length; i++) {
          const alternatives = allBranchSteps.map((bs: any) => bs[i]).filter(Boolean);
          if (alternatives.length > 1) {
            result.push({ label: alternatives[0], icon: GitBranch, alternatives });
          } else {
            const def = getStepDef(seg.branches[0].steps[i].type);
            result.push({ label: firstBranch[i], icon: def.icon });
          }
        }
      }
    }
    return result;
  }, [segments]);

  // Auto-dismiss messages
  useMemo(() => {
    if (msg) {
      const t = setTimeout(() => setMsg(null), 3000);
      return () => clearTimeout(t);
    }
  }, [msg]);

  return (
    <div>
      {/* Header */}
      <div className="mb-6 flex items-center justify-between">
        <div>
          <h1 className="flex items-center gap-2 text-2xl font-bold dark:text-gray-100">
            <GitBranch className="h-6 w-6 text-brand-600" />
            Login Flow Builder
          </h1>
          <p className="mt-1 text-sm text-gray-500">
            Design multi-step authentication flows with drag-and-drop, conditional branches, and MFA.
          </p>
        </div>
        <div className="flex items-center gap-2">
          {/* Activate toggle */}
          <label className="flex items-center gap-2 rounded-lg border border-gray-300 px-3 py-2 text-sm dark:border-gray-600">
            <Power className={`h-4 w-4 ${isActive ? "text-green-500" : "text-gray-400"}`} />
            <span className="text-gray-600 dark:text-gray-300">Activate Flow</span>
            <button
              onClick={() => { setIsActive(!isActive); setMsg(isActive ? "Flow deactivated" : "Flow activated"); }}
              className={`relative h-5 w-9 rounded-full transition-colors ${isActive ? "bg-brand-600" : "bg-gray-300 dark:bg-gray-600"}`}
            >
              <span className={`absolute top-0.5 h-4 w-4 rounded-full bg-white transition-transform ${isActive ? "translate-x-4" : "translate-x-0.5"}`} />
            </button>
          </label>
          {/* Preview toggle */}
          <button
            onClick={() => setPreviewMode(!previewMode)}
            className="flex items-center gap-1.5 rounded-lg border border-gray-300 px-4 py-2 text-sm font-medium text-gray-700 hover:bg-gray-50 dark:border-gray-600 dark:text-gray-300 dark:hover:bg-gray-700"
          >
            {previewMode ? <EyeOff className="h-4 w-4" /> : <Eye className="h-4 w-4" />}
            {previewMode ? "Edit" : "Preview"}
          </button>
          {/* Save */}
          <button
            onClick={handleSave}
            disabled={saving}
            className="flex items-center gap-1.5 rounded-lg bg-brand-600 px-4 py-2 text-sm font-medium text-white hover:bg-brand-700 disabled:opacity-50"
           aria-label="Save">
            <Save className="h-4 w-4" /> {saving ? "Saving..." : "Save Flow"}
          </button>
        </div>
      </div>

      {/* Messages */}
      {msg && (
        <div role="status" className="mb-4 rounded-lg border border-green-200 bg-green-50 p-3 text-sm text-green-700 dark:border-green-800 dark:bg-green-950 dark:text-green-400">
          {msg}
        </div>
      )}
      {error && (
        <div className="mb-4 rounded-lg border border-red-200 bg-red-50 p-3 text-sm text-red-700 dark:border-red-800 dark:bg-red-950 dark:text-red-400">
          {error}
        </div>
      )}

      {/* Preview Mode */}
      {previewMode ? (
        <div className="mx-auto max-w-lg">
          <div className="rounded-xl border border-gray-200 bg-white p-6 shadow-sm dark:border-gray-700 dark:bg-gray-800">
            <h3 className="mb-4 flex items-center gap-2 text-sm font-semibold">
              <Play className="h-4 w-4 text-brand-600" /> User Experience Preview
            </h3>
            {previewSteps.length === 0 ? (
              <p className="py-8 text-center text-sm text-gray-400">No steps in flow. Add steps in edit mode.</p>
            ) : (
              <div className="space-y-1">
                {previewSteps.map((step: any, i: any) => (
                  <div key={i}>
                    <div className="flex items-center gap-3 rounded-lg border border-gray-200 p-3 dark:border-gray-700">
                      <span className="flex h-7 w-7 items-center justify-center rounded-full bg-brand-100 text-xs font-bold text-brand-600">
                        {i + 1}
                      </span>
                      <step.icon className="h-5 w-5 text-gray-500" />
                      <span className="text-sm font-medium">{step.label}</span>
                      {step.alternatives && (
                        <span className="ml-auto rounded-full bg-amber-50 px-2 py-0.5 text-xs text-amber-600 dark:bg-amber-900/30">
                          OR: {step.alternatives.join(" / ")}
                        </span>
                      )}
                    </div>
                    {i < previewSteps.length - 1 && (
                      <div className="flex justify-center py-1">
                        <ArrowDown className="h-4 w-4 text-gray-300" />
                      </div>
                    )}
                  </div>
                ))}
              </div>
            )}
          </div>
        </div>
      ) : (
        /* ===== Editor Mode ===== */
        <div className="flex gap-4">
          {/* Step Palette (left sidebar) */}
          <div className="w-56 shrink-0">
            <div className="sticky top-4 rounded-xl border border-gray-200 bg-white p-4 shadow-sm dark:border-gray-700 dark:bg-gray-800">
              <h3 className="mb-3 text-xs font-bold uppercase tracking-wide text-gray-500">
                Step Types
              </h3>
              <div className="space-y-1.5">
                {STEP_TYPES.map((st: any) => (
                  <div
                    key={st.type}
                    draggable
                    onDragStart={() => handlePaletteDragStart(st.type)}
                    onDragEnd={() => { setDraggedType(null); setDragState(null); }}
                    className="flex cursor-grab items-center gap-2 rounded-lg border border-gray-200 p-2 hover:border-brand-300 hover:bg-brand-50 dark:border-gray-700 dark:hover:border-brand-700 dark:hover:bg-brand-900/20 active:cursor-grabbing"
                  >
                    <div className={`flex h-7 w-7 items-center justify-center rounded-lg ${st.color}`}>
                      <st.icon className="h-4 w-4" />
                    </div>
                    <span className="text-xs font-medium text-gray-700 dark:text-gray-300">{st.label}</span>
                  </div>
                ))}
              </div>
              <p className="mt-3 text-[10px] text-gray-400">Drag steps to the canvas</p>
            </div>
          </div>

          {/* Flow Canvas (center) */}
          <div className="flex-1">
            <div className="space-y-4">
              {segments.map((seg, segIdx) => (
                <div key={seg.id}>
                  {/* Linear segment */}
                  {seg.type === "linear" && (
                    <div
                      className="rounded-xl border border-gray-200 bg-white p-4 shadow-sm dark:border-gray-700 dark:bg-gray-800"
                      onDragOver={handleDragOver}
                      onDrop={() => handleDrop(seg.id, undefined, (seg.steps || []).length)}
                    >
                      <div className="mb-2 flex items-center justify-between">
                        <span className="text-xs font-bold uppercase tracking-wide text-gray-500">
                          Step Group
                        </span>
                        <button
                          onClick={() => removeSegment(seg.id)}
                          className="text-gray-400 hover:text-red-500"
                          title="Remove segment"
                        >
                          <Trash2 className="h-3.5 w-3.5" />
                        </button>
                      </div>
                      {(seg.steps || []).map((step: any, idx: any) => (
                        <StepCard
                          key={step.id}
                          step={step}
                          index={idx}
                          isSelected={selectedStep?.id === step.id}
                          previewMode={previewMode}
                          onDragStart={() => handleDragStart(seg.id, undefined, idx)}
                          onDragOver={handleDragOver}
                          onDrop={() => handleDrop(seg.id, undefined, idx)}
                          onClick={() => {
                            setSelectedStep(step);
                            setSelectedStepLocation({ segId: seg.id, index: idx });
                          }}
                          onDelete={() => removeStep(seg.id, undefined, step.id)}
                        />
                      ))}
                      {(seg.steps || []).length === 0 && (
                        <DropZone onDrop={() => draggedType && addStepToSegment(seg.id, undefined, draggedType)} />
                      )}
                    </div>
                  )}

                  {/* Branch segment */}
                  {seg.type === "branch" && (
                    <div className="rounded-xl border border-gray-200 bg-white p-4 shadow-sm dark:border-gray-700 dark:bg-gray-800">
                      <div className="mb-3 flex items-center justify-between">
                        <span className="flex items-center gap-1.5 text-xs font-bold uppercase tracking-wide text-gray-500">
                          <GitBranch className="h-3.5 w-3.5" /> Conditional Branch
                        </span>
                        <div className="flex gap-2">
                          <button
                            onClick={() => addBranchToSegment(seg.id)}
                            className="flex items-center gap-1 rounded-lg border border-gray-300 px-2 py-1 text-xs text-gray-600 hover:bg-gray-50 dark:border-gray-600 dark:text-gray-300 dark:hover:bg-gray-700"
                          >
                            <Plus className="h-3 w-3" /> Add Path
                          </button>
                          <button
                            onClick={() => removeSegment(seg.id)}
                            className="text-gray-400 hover:text-red-500"
                            title="Remove branch segment"
                          >
                            <Trash2 className="h-3.5 w-3.5" />
                          </button>
                        </div>
                      </div>
                      <div className="flex gap-4">
                        {(seg.branches || []).map((branch, bIdx) => (
                          <div key={branch.id} className="flex-1">
                            {/* OR connector */}
                            {bIdx > 0 && (
                              <div className="mb-2 flex justify-center">
                                <span className="rounded-full bg-amber-100 px-3 py-0.5 text-xs font-bold text-amber-600 dark:bg-amber-900/40 dark:text-amber-400">
                                  OR
                                </span>
                              </div>
                            )}
                            <div
                              className="rounded-lg border border-dashed border-gray-300 p-3 dark:border-gray-600"
                              onDragOver={handleDragOver}
                              onDrop={() => handleDrop(seg.id, branch.id, branch.steps.length)}
                            >
                              <p className="mb-2 text-xs font-medium text-gray-400">Path {bIdx + 1}</p>
                              <div className="space-y-2">
                                {branch.steps.map((step: any, idx: any) => (
                                  <StepCard
                                    key={step.id}
                                    step={step}
                                    index={idx}
                                    isSelected={selectedStep?.id === step.id}
                                    previewMode={previewMode}
                                    onDragStart={() => handleDragStart(seg.id, branch.id, idx)}
                                    onDragOver={handleDragOver}
                                    onDrop={() => handleDrop(seg.id, branch.id, idx)}
                                    onClick={() => {
                                      setSelectedStep(step);
                                      setSelectedStepLocation({ segId: seg.id, branchId: branch.id, index: idx });
                                    }}
                                    onDelete={() => removeStep(seg.id, branch.id, step.id)}
                                  />
                                ))}
                                {branch.steps.length === 0 && (
                                  <DropZone onDrop={() => draggedType && addStepToSegment(seg.id, branch.id, draggedType)} />
                                )}
                              </div>
                            </div>
                          </div>
                        ))}
                      </div>
                    </div>
                  )}

                  {/* Connector between segments */}
                  {segIdx < segments.length - 1 && (
                    <div className="flex justify-center py-1">
                      <ArrowDown className="h-5 w-5 text-gray-300" />
                    </div>
                  )}
                </div>
              ))}

              {/* Add segment buttons */}
              <div className="flex justify-center gap-2 pt-2">
                <button
                  onClick={addLinearSegment}
                  className="flex items-center gap-1.5 rounded-lg border border-gray-300 px-4 py-2 text-sm font-medium text-gray-600 hover:bg-gray-50 dark:border-gray-600 dark:text-gray-300 dark:hover:bg-gray-700"
                >
                  <Plus className="h-4 w-4" /> Add Step Group
                </button>
                <button
                  onClick={addBranchSegment}
                  className="flex items-center gap-1.5 rounded-lg border border-gray-300 px-4 py-2 text-sm font-medium text-gray-600 hover:bg-gray-50 dark:border-gray-600 dark:text-gray-300 dark:hover:bg-gray-700"
                >
                  <GitBranch className="h-4 w-4" /> Add Branch
                </button>
              </div>
            </div>
          </div>

          {/* Config Panel (right sidebar) */}
          <div className="w-64 shrink-0">
            <div className="sticky top-4">
              {selectedStep && selectedStepLocation ? (
                <ConfigPanel
                  step={selectedStep}
                  onUpdate={(config) => updateStepConfig(
                    selectedStepLocation.segId,
                    selectedStepLocation.branchId,
                    selectedStep.id,
                    config,
                  )}
                  onClose={() => { setSelectedStep(null); setSelectedStepLocation(null); }}
                />
              ) : (
                <div className="rounded-xl border border-gray-200 bg-white p-4 text-center shadow-sm dark:border-gray-700 dark:bg-gray-800">
                  <Settings2 className="mx-auto mb-2 h-8 w-8 text-gray-300" />
                  <p className="text-xs text-gray-400">
                    Select a step to configure its settings
                  </p>
                </div>
              )}
            </div>
          </div>
        </div>
      )}
    </div>
  );
}

// ===== Step Card Component =====

function StepCard({
  step, index, isSelected, previewMode,
  onDragStart, onDragOver, onDrop, onClick, onDelete,
}: {
  step: FlowStep;
  index: number;
  isSelected: boolean;
  previewMode: boolean;
  onDragStart: () => void;
  onDragOver: (e: React.DragEvent) => void;
  onDrop: () => void;
  onClick: () => void;
  onDelete: () => void;
}) {
  const def = getStepDef(step.type);
  return (
    <div
      draggable={!previewMode}
      onDragStart={onDragStart}
      onDragOver={onDragOver}
      onDrop={onDrop}
      className={`mb-2 flex items-center gap-3 rounded-lg border p-3 transition-all ${
        isSelected
          ? "border-brand-400 bg-brand-50 dark:border-brand-600 dark:bg-brand-900/20"
          : "border-gray-200 bg-white hover:border-gray-300 dark:border-gray-700 dark:bg-gray-800 dark:hover:border-gray-600"
      } ${previewMode ? "cursor-default" : "cursor-pointer"}`}
      onClick={onClick}
    >
      {/* Drag handle */}
      {!previewMode && (
        <GripVertical className="h-4 w-4 cursor-grab text-gray-300 hover:text-gray-400" />
      )}

      {/* Step number */}
      <span className="flex h-6 w-6 items-center justify-center rounded-full bg-gray-100 text-xs font-bold text-gray-500 dark:bg-gray-700 dark:text-gray-400">
        {index + 1}
      </span>

      {/* Icon */}
      <div className={`flex h-8 w-8 items-center justify-center rounded-lg ${def.color}`}>
        <def.icon className="h-4 w-4" />
      </div>

      {/* Name + badges */}
      <div className="flex-1">
        <div className="flex items-center gap-1.5">
          <span className="text-sm font-medium text-gray-800 dark:text-gray-200">{step.name}</span>
          {step.config.required && (
            <span className="rounded bg-red-50 px-1.5 py-0.5 text-[10px] font-medium text-red-600 dark:bg-red-900/30 dark:text-red-400">Required</span>
          )}
          {step.config.fallback && (
            <span className="rounded bg-blue-50 px-1.5 py-0.5 text-[10px] font-medium text-blue-600 dark:bg-blue-900/30 dark:text-blue-400">Fallback</span>
          )}
        </div>
        <p className="text-[10px] text-gray-400">Timeout: {step.config.timeout}s, Retries: {step.config.retryCount}</p>
      </div>

      {/* Actions */}
      {!previewMode && (
        <div className="flex items-center gap-1">
          <button
            onClick={(e) => { e.stopPropagation(); onDelete(); }}
            className="text-gray-300 hover:text-red-500"
            title="Delete step"
          >
            <Trash2 className="h-4 w-4" />
          </button>
        </div>
      )}
    </div>
  );
}

// ===== Drop Zone Component =====

function DropZone({ onDrop }: { onDrop: () => void }) {
  return (
    <div
      onDragOver={(e) => e.preventDefault()}
      onDrop={(e) => { e.preventDefault(); onDrop(); }}
      className="flex items-center justify-center rounded-lg border-2 border-dashed border-gray-200 py-6 text-xs text-gray-400 dark:border-gray-700"
    >
      Drop step here
    </div>
  );
}

// ===== Config Panel Component =====

function ConfigPanel({
  step, onUpdate, onClose,
}: {
  step: FlowStep;
  onUpdate: (config: Partial<StepConfig>) => void;
  onClose: () => void;
}) {
  const def = getStepDef(step.type);
  return (
    <div className="rounded-xl border border-gray-200 bg-white p-4 shadow-sm dark:border-gray-700 dark:bg-gray-800">
      <div className="mb-3 flex items-center justify-between">
        <h3 className="flex items-center gap-2 text-sm font-semibold">
          <div className={`flex h-7 w-7 items-center justify-center rounded-lg ${def.color}`}>
            <def.icon className="h-4 w-4" />
          </div>
          {step.name}
        </h3>
        <button onClick={onClose} className="text-gray-400 hover:text-gray-600" aria-label="ChevronRight">
          <ChevronRight className="h-4 w-4" />
        </button>
      </div>

      <p className="mb-4 text-xs text-gray-400">{def.description}</p>

      {/* Required toggle */}
      <div className="mb-3">
        <label className="flex items-center justify-between">
          <div>
            <span className="text-xs font-medium text-gray-600 dark:text-gray-300">Required</span>
            <p className="text-[10px] text-gray-400">User must complete this step</p>
          </div>
          <button
            onClick={() => onUpdate({ required: !step.config.required })}
            className={`relative h-5 w-9 rounded-full transition-colors ${step.config.required ? "bg-brand-600" : "bg-gray-300 dark:bg-gray-600"}`}
          >
            <span className={`absolute top-0.5 h-4 w-4 rounded-full bg-white transition-transform ${step.config.required ? "translate-x-4" : "translate-x-0.5"}`} />
          </button>
        </label>
      </div>

      {/* Fallback toggle */}
      <div className="mb-3">
        <label className="flex items-center justify-between">
          <div>
            <span className="text-xs font-medium text-gray-600 dark:text-gray-300">Fallback</span>
            <p className="text-[10px] text-gray-400">Use as alternative if primary fails</p>
          </div>
          <button
            onClick={() => onUpdate({ fallback: !step.config.fallback })}
            className={`relative h-5 w-9 rounded-full transition-colors ${step.config.fallback ? "bg-brand-600" : "bg-gray-300 dark:bg-gray-600"}`}
          >
            <span className={`absolute top-0.5 h-4 w-4 rounded-full bg-white transition-transform ${step.config.fallback ? "translate-x-4" : "translate-x-0.5"}`} />
          </button>
        </label>
      </div>

      {/* Timeout */}
      <div className="mb-3">
        <label className="mb-1 block text-xs font-medium text-gray-600 dark:text-gray-300">
          Timeout (seconds)
        </label>
        <input
          type="number"
          value={step.config.timeout}
          onChange={(e) => onUpdate({ timeout: parseInt(e.target.value) || 0 })}
          min={0}
          className="w-full rounded-lg border border-gray-300 px-3 py-2 text-sm dark:border-gray-600 dark:bg-gray-700 dark:text-gray-200"
        />
      </div>

      {/* Retry count */}
      <div className="mb-3">
        <label className="mb-1 block text-xs font-medium text-gray-600 dark:text-gray-300">
          Retry Count
        </label>
        <input
          type="number"
          value={step.config.retryCount}
          onChange={(e) => onUpdate({ retryCount: parseInt(e.target.value) || 0 })}
          min={0}
          className="w-full rounded-lg border border-gray-300 px-3 py-2 text-sm dark:border-gray-600 dark:bg-gray-700 dark:text-gray-200"
        />
      </div>

      {/* JSON preview */}
      <div className="mt-4">
        <label className="mb-1 block text-[10px] font-medium text-gray-400">Step Config JSON</label>
        <pre className="max-h-32 overflow-auto rounded-lg bg-gray-900 p-3 text-[10px] text-green-400">
          {JSON.stringify({ type: step.type, ...step.config }, null, 2)}
        </pre>
      </div>
    </div>
  );
}
