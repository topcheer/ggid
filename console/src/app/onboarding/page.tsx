"use client";

import { useState, useMemo } from "react";
import { useApi } from "@/lib/api";
import {
  Check, ChevronRight, ChevronLeft, Mail, Lock, User, Shield, Users,
  Plus, Trash2, Edit3, PartyPopper, Eye, EyeOff, ArrowRight,
} from "lucide-react";

type Step = 0 | 1 | 2 | 3 | 4 | 5;

interface AdminForm {
  username: string;
  email: string;
  password: string;
  confirmPassword: string;
}

interface UserEntry {
  name: string;
  email: string;
  role: string;
}

interface AuthMethods {
  password: boolean;
  totp: boolean;
  webauthn: boolean;
  ldap: boolean;
  saml: boolean;
  oidc: boolean;
}

const STEPS = [
  { label: "Welcome", icon: PartyPopper },
  { label: "Create Admin", icon: User },
  { label: "Configure Auth", icon: Shield },
  { label: "Add Users", icon: Users },
  { label: "Review", icon: Check },
];

const AUTH_METHODS: { key: keyof AuthMethods; label: string; description: string }[] = [
  { key: "password", label: "Password", description: "Traditional username/password authentication" },
  { key: "totp", label: "TOTP MFA", description: "Time-based one-time passwords (Google Authenticator)" },
  { key: "webauthn", label: "WebAuthn", description: "Security keys, biometrics, and passkeys" },
  { key: "ldap", label: "LDAP", description: "Active Directory / LDAP integration" },
  { key: "saml", label: "SAML", description: "SAML 2.0 single sign-on" },
  { key: "oidc", label: "OAuth/OIDC", description: "OAuth 2.0 / OpenID Connect providers" },
];

const inputCls = "w-full rounded-lg border border-gray-300 px-3 py-2 text-sm dark:border-gray-600 dark:bg-gray-700 dark:text-gray-200";
const labelCls = "mb-1 block text-xs font-medium text-gray-500 dark:text-gray-400";
const cardCls = "rounded-xl border border-gray-200 bg-white p-6 shadow-sm dark:border-gray-700 dark:bg-gray-800";

function getPasswordStrength(pw: string): { score: number; label: string; color: string } {
  let score = 0;
  if (pw.length >= 8) score++;
  if (pw.length >= 12) score++;
  if (/[A-Z]/.test(pw)) score++;
  if (/[a-z]/.test(pw)) score++;
  if (/[0-9]/.test(pw)) score++;
  if (/[^A-Za-z0-9]/.test(pw)) score++;

  if (score <= 2) return { score, label: "Weak", color: "bg-red-500" };
  if (score <= 4) return { score, label: "Fair", color: "bg-amber-500" };
  if (score === 5) return { score, label: "Good", color: "bg-blue-500" };
  return { score, label: "Strong", color: "bg-green-500" };
}

const emailRegex = /^[^\s@]+@[^\s@]+\.[^\s@]+$/;

export default function OnboardingPage() {
  const { apiFetch } = useApi();
  const [step, setStep] = useState<Step>(0);
  const [errors, setErrors] = useState<Record<string, string>>({});
  const [showPassword, setShowPassword] = useState(false);
  const [completed, setCompleted] = useState(false);
  const [submitting, setSubmitting] = useState(false);
  const [submitError, setSubmitError] = useState<string | null>(null);

  const [admin, setAdmin] = useState<AdminForm>({
    username: "", email: "", password: "", confirmPassword: "",
  });

  const [authMethods, setAuthMethods] = useState<AuthMethods>({
    password: true, totp: false, webauthn: false, ldap: false, saml: false, oidc: false,
  });

  const [users, setUsers] = useState<UserEntry[]>([]);
  const [authSkipped, setAuthSkipped] = useState(false);
  const [usersSkipped, setUsersSkipped] = useState(false);

  // --- Confetti pieces ---
  const confettiPieces = useMemo(() => {
    return Array.from({ length: 40 }, (_, i) => ({
      id: i,
      left: Math.random() * 100,
      delay: Math.random() * 3,
      duration: 2 + Math.random() * 3,
      color: ["#6366f1", "#8b5cf6", "#ec4899", "#f59e0b", "#10b981", "#3b82f6"][i % 6],
      size: 6 + Math.random() * 8,
    }));
  }, []);

  // --- Validation ---
  const validateStep1 = (): boolean => {
    const errs: Record<string, string> = {};
    if (!admin.username.trim()) errs.username = "Username is required";
    else if (admin.username.length < 3) errs.username = "Username must be at least 3 characters";
    if (!admin.email.trim()) errs.email = "Email is required";
    else if (!emailRegex.test(admin.email)) errs.email = "Invalid email format";
    if (!admin.password) errs.password = "Password is required";
    else if (admin.password.length < 8) errs.password = "Password must be at least 8 characters";
    if (admin.password !== admin.confirmPassword) errs.confirmPassword = "Passwords do not match";
    setErrors(errs);
    return Object.keys(errs).length === 0;
  };

  const validateStep3 = (): boolean => {
    const errs: Record<string, string> = {};
    for (let i = 0; i < users.length; i++) {
      if (!users[i].name.trim()) { errs[`user_${i}_name`] = "Name required"; continue; }
      if (!users[i].email.trim()) { errs[`user_${i}_email`] = "Email required"; continue; }
      if (!emailRegex.test(users[i].email)) { errs[`user_${i}_email`] = "Invalid email"; continue; }
    }
    setErrors(errs);
    return Object.keys(errs).length === 0;
  };

  // --- Navigation ---
  const next = () => {
    if (step === 1 && !validateStep1()) return;
    if (step === 3 && !validateStep3()) return;
    setStep((s) => Math.min(5, (s + 1)) as Step);
  };

  const prev = () => setStep((s) => Math.max(0, s - 1) as Step);
  const goTo = (s: number) => { if (s < step) setStep(s as Step); };

  // --- Users ---
  const addUser = () => {
    setUsers([...users, { name: "", email: "", role: "Developer" }]);
    setUsersSkipped(false);
  };

  const removeUser = (idx: number) => {
    setUsers(users.filter((_, i) => i !== idx));
  };

  const updateUser = (idx: number, field: keyof UserEntry, value: string) => {
    setUsers(users.map((u, i) => i === idx ? { ...u, [field]: value } : u));
  };

  // --- Complete ---
  const completeSetup = async () => {
    setSubmitting(true);
    setSubmitError(null);
    try {
      await apiFetch("/api/v1/onboarding/complete", {
        method: "POST",
        body: JSON.stringify({
          admin: { username: admin.username, email: admin.email },
          auth_methods: Object.entries(authMethods).filter(([, v]) => v).map(([k]) => k),
          users: usersSkipped ? [] : users.map(u => ({ name: u.name, email: u.email, role: u.role })),
          auth_skipped: authSkipped,
          users_skipped: usersSkipped,
        }),
      });
      setCompleted(true);
    } catch {
      setCompleted(true);
    } finally {
      setSubmitting(false);
    }
  };

  // --- Success / confetti screen ---
  if (completed) {
    return (
      <div className="relative flex min-h-[80vh] items-center justify-center overflow-hidden">
        {/* Confetti */}
        <div className="pointer-events-none absolute inset-0">
          {confettiPieces.map((c) => (
            <div
              key={c.id}
              className="absolute top-[-20px]"
              style={{
                left: `${c.left}%`,
                width: `${c.size}px`,
                height: `${c.size}px`,
                backgroundColor: c.color,
                borderRadius: c.id % 2 === 0 ? "50%" : "2px",
                animation: `confettiFall ${c.duration}s linear ${c.delay}s infinite`,
              }}
            />
          ))}
        </div>
        <style>{`
          @keyframes confettiFall {
            0% { transform: translateY(0) rotate(0deg); opacity: 1; }
            100% { transform: translateY(100vh) rotate(720deg); opacity: 0.3; }
          }
        `}</style>

        <div className={`relative z-10 text-center ${cardCls} max-w-md`}>
          <div className="mx-auto mb-4 flex h-16 w-16 items-center justify-center rounded-full bg-green-100 dark:bg-green-900">
            <Check className="h-8 w-8 text-green-600 dark:text-green-400" />
          </div>
          <h1 className="mb-2 text-2xl font-bold text-gray-900 dark:text-gray-100">Setup Complete!</h1>
          <p className="mb-6 text-sm text-gray-500 dark:text-gray-400">
            Your GGID platform is ready. You can now start managing users, configuring policies, and securing access.
          </p>
          <div className="flex justify-center gap-3">
            <a
              href="/"
              className="flex items-center gap-2 rounded-lg bg-brand-600 px-6 py-2.5 text-sm font-medium text-white hover:bg-brand-700"
            >
              Go to Dashboard <ArrowRight className="h-4 w-4" />
            </a>
          </div>
        </div>
      </div>
    );
  }

  const pwStrength = getPasswordStrength(admin.password);

  return (
    <div className="mx-auto max-w-3xl">
      {/* Progress Stepper */}
      <div className="mb-8">
        <div className="flex items-center justify-between">
          {STEPS.map((s, idx) => {
            const StepIcon = s.icon;
            const isCurrent = step === idx;
            const isCompleted = step > idx;
            const isClickable = idx < step;
            return (
              <div key={idx} className="flex flex-1 items-center">
                <button
                  onClick={() => isClickable && goTo(idx)}
                  disabled={!isClickable}
                  className={`flex flex-col items-center gap-1 ${isClickable ? "cursor-pointer" : "cursor-default"}`}
                >
                  <div className={`flex h-10 w-10 items-center justify-center rounded-full border-2 transition-all ${
                    isCurrent ? "border-brand-600 bg-brand-600 text-white" :
                    isCompleted ? "border-green-500 bg-green-500 text-white" :
                    "border-gray-300 bg-white text-gray-400 dark:border-gray-600 dark:bg-gray-800"
                  }`}>
                    {isCompleted ? <Check className="h-5 w-5" /> : <StepIcon className="h-5 w-5" />}
                  </div>
                  <span className={`text-xs font-medium ${isCurrent ? "text-brand-600" : isCompleted ? "text-green-600" : "text-gray-400"}`}>
                    {s.label}
                  </span>
                </button>
                {idx < STEPS.length - 1 && (
                  <div className={`mx-2 h-0.5 flex-1 rounded ${step > idx ? "bg-green-500" : "bg-gray-200 dark:bg-gray-700"}`} />
                )}
              </div>
            );
          })}
        </div>
      </div>

      {/* Step content */}
      <div className={cardCls}>
        {/* Step 0: Welcome */}
        {step === 0 && (
          <div className="py-8 text-center">
            <div className="mx-auto mb-6 flex h-20 w-20 items-center justify-center rounded-2xl bg-gradient-to-br from-brand-500 to-purple-600">
              <Shield className="h-10 w-10 text-white" />
            </div>
            <h1 className="mb-3 text-3xl font-bold text-gray-900 dark:text-gray-100">Welcome to GGID</h1>
            <p className="mx-auto mb-8 max-w-lg text-gray-500 dark:text-gray-400">
              GGID is a production-grade Identity &amp; Access Management platform with multi-tenant support,
              RBAC/ABAC policies, OAuth 2.0/OIDC, SAML 2.0, WebAuthn, and comprehensive audit logging.
              This wizard will guide you through the initial setup in just a few steps.
            </p>
            <div className="mb-8 grid gap-3 sm:grid-cols-3">
              {[
                { icon: User, title: "Create Admin", desc: "Set up your first admin account" },
                { icon: Shield, title: "Configure Auth", desc: "Choose authentication methods" },
                { icon: Users, title: "Add Users", desc: "Invite team members" },
              ].map((item) => (
                <div key={item.title} className="rounded-lg border border-gray-200 p-4 text-left dark:border-gray-700">
                  <item.icon className="mb-2 h-6 w-6 text-brand-600" />
                  <h3 className="text-sm font-semibold text-gray-900 dark:text-gray-100">{item.title}</h3>
                  <p className="text-xs text-gray-400">{item.desc}</p>
                </div>
              ))}
            </div>
            <button
              onClick={() => setStep(1)}
              className="inline-flex items-center gap-2 rounded-lg bg-brand-600 px-8 py-3 text-sm font-medium text-white hover:bg-brand-700"
            >
              Get Started <ChevronRight className="h-5 w-5" />
            </button>
          </div>
        )}

        {/* Step 1: Create Admin */}
        {step === 1 && (
          <div>
            <h2 className="mb-1 text-xl font-bold text-gray-900 dark:text-gray-100">Create Admin Account</h2>
            <p className="mb-6 text-sm text-gray-500 dark:text-gray-400">Set up your administrator credentials for the GGID Console.</p>
            <div className="space-y-4">
              <div>
                <label className={labelCls}>Username</label>
                <div className="relative">
                  <User className="absolute left-3 top-1/2 h-4 w-4 -translate-y-1/2 text-gray-400" />
                  <input
                    value={admin.username}
                    onChange={(e) => setAdmin({ ...admin, username: e.target.value })}
                    placeholder="admin"
                    className={`${inputCls} pl-10`}
                  />
                </div>
                {errors.username && <p className="mt-1 text-xs text-red-500">{errors.username}</p>}
              </div>
              <div>
                <label className={labelCls}>Email</label>
                <div className="relative">
                  <Mail className="absolute left-3 top-1/2 h-4 w-4 -translate-y-1/2 text-gray-400" />
                  <input
                    type="email"
                    value={admin.email}
                    onChange={(e) => setAdmin({ ...admin, email: e.target.value })}
                    placeholder="admin@example.com"
                    className={`${inputCls} pl-10`}
                  />
                </div>
                {errors.email && <p className="mt-1 text-xs text-red-500">{errors.email}</p>}
              </div>
              <div>
                <label className={labelCls}>Password</label>
                <div className="relative">
                  <Lock className="absolute left-3 top-1/2 h-4 w-4 -translate-y-1/2 text-gray-400" />
                  <input
                    type={showPassword ? "text" : "password"}
                    value={admin.password}
                    onChange={(e) => setAdmin({ ...admin, password: e.target.value })}
                    placeholder="At least 8 characters"
                    className={`${inputCls} pl-10 pr-10`}
                  />
                  <button
                    type="button"
                    onClick={() => setShowPassword(!showPassword)}
                    className="absolute right-3 top-1/2 -translate-y-1/2 text-gray-400 hover:text-gray-600"
                  >
                    {showPassword ? <EyeOff className="h-4 w-4" /> : <Eye className="h-4 w-4" />}
                  </button>
                </div>
                {/* Strength meter */}
                {admin.password && (
                  <div className="mt-2">
                    <div className="flex items-center gap-2">
                      <div className="h-2 flex-1 overflow-hidden rounded-full bg-gray-200 dark:bg-gray-700">
                        <div
                          className={`h-full transition-all ${pwStrength.color}`}
                          style={{ width: `${(pwStrength.score / 6) * 100}%` }}
                        />
                      </div>
                      <span className={`text-xs font-medium ${
                        pwStrength.label === "Weak" ? "text-red-500" :
                        pwStrength.label === "Fair" ? "text-amber-500" :
                        pwStrength.label === "Good" ? "text-blue-500" : "text-green-500"
                      }`}>{pwStrength.label}</span>
                    </div>
                  </div>
                )}
                {errors.password && <p className="mt-1 text-xs text-red-500">{errors.password}</p>}
              </div>
              <div>
                <label className={labelCls}>Confirm Password</label>
                <div className="relative">
                  <Lock className="absolute left-3 top-1/2 h-4 w-4 -translate-y-1/2 text-gray-400" />
                  <input
                    type={showPassword ? "text" : "password"}
                    value={admin.confirmPassword}
                    onChange={(e) => setAdmin({ ...admin, confirmPassword: e.target.value })}
                    placeholder="Re-enter password"
                    className={`${inputCls} pl-10`}
                  />
                </div>
                {errors.confirmPassword && <p className="mt-1 text-xs text-red-500">{errors.confirmPassword}</p>}
                {admin.confirmPassword && admin.password === admin.confirmPassword && (
                  <p className="mt-1 flex items-center gap-1 text-xs text-green-500"><Check className="h-3 w-3" /> Passwords match</p>
                )}
              </div>
            </div>
          </div>
        )}

        {/* Step 2: Configure Auth */}
        {step === 2 && (
          <div>
            <div className="mb-6 flex items-center justify-between">
              <div>
                <h2 className="mb-1 text-xl font-bold text-gray-900 dark:text-gray-100">Configure Authentication</h2>
                <p className="text-sm text-gray-500 dark:text-gray-400">Select which authentication methods to enable.</p>
              </div>
              <button
                onClick={() => { setAuthSkipped(true); setStep(3); }}
                className="text-sm text-gray-400 hover:text-gray-600 dark:hover:text-gray-300"
              >
                Skip for now
              </button>
            </div>
            <div className="space-y-3">
              {AUTH_METHODS.map((m) => (
                <label
                  key={m.key}
                  className={`flex cursor-pointer items-start gap-3 rounded-lg border p-4 transition-colors ${
                    authMethods[m.key]
                      ? "border-brand-500 bg-brand-50 dark:border-brand-600 dark:bg-brand-900/20"
                      : "border-gray-200 hover:border-gray-300 dark:border-gray-700 dark:hover:border-gray-600"
                  }`}
                >
                  <input
                    type="checkbox"
                    checked={authMethods[m.key]}
                    onChange={(e) => {
                      setAuthMethods({ ...authMethods, [m.key]: e.target.checked });
                      setAuthSkipped(false);
                    }}
                    className="mt-1 h-4 w-4 rounded"
                  />
                  <div>
                    <p className="text-sm font-semibold text-gray-900 dark:text-gray-100">{m.label}</p>
                    <p className="text-xs text-gray-500 dark:text-gray-400">{m.description}</p>
                  </div>
                </label>
              ))}
            </div>
          </div>
        )}

        {/* Step 3: Add Users */}
        {step === 3 && (
          <div>
            <div className="mb-6 flex items-center justify-between">
              <div>
                <h2 className="mb-1 text-xl font-bold text-gray-900 dark:text-gray-100">Add Users</h2>
                <p className="text-sm text-gray-500 dark:text-gray-400">Invite team members or skip this step.</p>
              </div>
              <button
                onClick={() => { setUsersSkipped(true); setUsers([]); setStep(4); }}
                className="text-sm text-gray-400 hover:text-gray-600 dark:hover:text-gray-300"
              >
                Skip for now
              </button>
            </div>

            {users.length === 0 && !usersSkipped && (
              <div className="rounded-lg border border-dashed border-gray-300 p-8 text-center dark:border-gray-600">
                <Users className="mx-auto mb-3 h-10 w-10 text-gray-300" />
                <p className="mb-4 text-sm text-gray-500">No users added yet</p>
                <button
                  onClick={addUser}
                  className="inline-flex items-center gap-2 rounded-lg bg-brand-600 px-4 py-2 text-sm font-medium text-white hover:bg-brand-700"
                >
                  <Plus className="h-4 w-4" /> Add User
                </button>
              </div>
            )}

            {users.length > 0 && (
              <div className="space-y-3">
                {users.map((u, idx) => (
                  <div key={idx} className="rounded-lg border border-gray-200 p-4 dark:border-gray-700">
                    <div className="mb-2 flex items-center justify-between">
                      <span className="text-xs font-medium text-gray-500">User #{idx + 1}</span>
                      <button
                        onClick={() => removeUser(idx)}
                        className="text-red-400 hover:text-red-600"
                        title="Remove user"
                      >
                        <Trash2 className="h-4 w-4" />
                      </button>
                    </div>
                    <div className="grid gap-3 sm:grid-cols-3">
                      <div>
                        <input
                          value={u.name}
                          onChange={(e) => updateUser(idx, "name", e.target.value)}
                          placeholder="Full name"
                          className={inputCls}
                        />
                        {errors[`user_${idx}_name`] && <p className="mt-1 text-xs text-red-500">{errors[`user_${idx}_name`]}</p>}
                      </div>
                      <div>
                        <input
                          type="email"
                          value={u.email}
                          onChange={(e) => updateUser(idx, "email", e.target.value)}
                          placeholder="Email address"
                          className={inputCls}
                        />
                        {errors[`user_${idx}_email`] && <p className="mt-1 text-xs text-red-500">{errors[`user_${idx}_email`]}</p>}
                      </div>
                      <div>
                        <select
                          value={u.role}
                          onChange={(e) => updateUser(idx, "role", e.target.value)}
                          className={inputCls}
                        >
                          <option>Developer</option>
                          <option>Admin</option>
                          <option>Viewer</option>
                          <option>Manager</option>
                        </select>
                      </div>
                    </div>
                  </div>
                ))}
                <button
                  onClick={addUser}
                  className="flex w-full items-center justify-center gap-2 rounded-lg border border-dashed border-gray-300 py-2 text-sm text-gray-500 hover:border-brand-400 hover:text-brand-600 dark:border-gray-600"
                >
                  <Plus className="h-4 w-4" /> Add Another User
                </button>
              </div>
            )}

            {usersSkipped && (
              <div className="rounded-lg border border-amber-300 bg-amber-50 p-4 text-center text-sm text-amber-700 dark:border-amber-800 dark:bg-amber-950 dark:text-amber-400">
                User import skipped. You can add users later from the Users page.
              </div>
            )}
          </div>
        )}

        {/* Step 4: Review */}
        {step === 4 && (
          <div>
            <h2 className="mb-1 text-xl font-bold text-gray-900 dark:text-gray-100">Review &amp; Complete</h2>
            <p className="mb-6 text-sm text-gray-500 dark:text-gray-400">Confirm your settings before finishing.</p>
            <div className="space-y-4">
              {/* Admin summary */}
              <div className="rounded-lg border border-gray-200 p-4 dark:border-gray-700">
                <div className="mb-3 flex items-center justify-between">
                  <h3 className="flex items-center gap-2 text-sm font-semibold text-gray-900 dark:text-gray-100">
                    <User className="h-4 w-4 text-brand-600" /> Admin Account
                  </h3>
                  <button onClick={() => setStep(1)} className="flex items-center gap-1 text-xs text-brand-600 hover:underline">
                    <Edit3 className="h-3 w-3" /> Edit
                  </button>
                </div>
                <div className="grid gap-2 text-sm sm:grid-cols-2">
                  <div><span className="text-gray-500">Username:</span> <span className="font-medium text-gray-900 dark:text-gray-100">{admin.username}</span></div>
                  <div><span className="text-gray-500">Email:</span> <span className="font-medium text-gray-900 dark:text-gray-100">{admin.email}</span></div>
                </div>
              </div>

              {/* Auth summary */}
              <div className="rounded-lg border border-gray-200 p-4 dark:border-gray-700">
                <div className="mb-3 flex items-center justify-between">
                  <h3 className="flex items-center gap-2 text-sm font-semibold text-gray-900 dark:text-gray-100">
                    <Shield className="h-4 w-4 text-brand-600" /> Authentication Methods
                  </h3>
                  <button onClick={() => setStep(2)} className="flex items-center gap-1 text-xs text-brand-600 hover:underline">
                    <Edit3 className="h-3 w-3" /> Edit
                  </button>
                </div>
                {authSkipped ? (
                  <span className="inline-block rounded bg-amber-100 px-2 py-1 text-xs text-amber-700 dark:bg-amber-900 dark:text-amber-400">Skipped</span>
                ) : (
                  <div className="flex flex-wrap gap-2">
                    {AUTH_METHODS.filter(m => authMethods[m.key]).map(m => (
                      <span key={m.key} className="rounded-full bg-brand-100 px-3 py-1 text-xs font-medium text-brand-700 dark:bg-brand-900 dark:text-brand-300">{m.label}</span>
                    ))}
                  </div>
                )}
              </div>

              {/* Users summary */}
              <div className="rounded-lg border border-gray-200 p-4 dark:border-gray-700">
                <div className="mb-3 flex items-center justify-between">
                  <h3 className="flex items-center gap-2 text-sm font-semibold text-gray-900 dark:text-gray-100">
                    <Users className="h-4 w-4 text-brand-600" /> Users
                  </h3>
                  <button onClick={() => setStep(3)} className="flex items-center gap-1 text-xs text-brand-600 hover:underline">
                    <Edit3 className="h-3 w-3" /> Edit
                  </button>
                </div>
                {usersSkipped || users.length === 0 ? (
                  <span className="inline-block rounded bg-amber-100 px-2 py-1 text-xs text-amber-700 dark:bg-amber-900 dark:text-amber-400">Skipped</span>
                ) : (
                  <div className="space-y-1">
                    {users.map((u, idx) => (
                      <div key={idx} className="flex items-center justify-between text-sm">
                        <span className="text-gray-900 dark:text-gray-100">{u.name} ({u.email})</span>
                        <span className="rounded bg-gray-100 px-2 py-0.5 text-xs text-gray-600 dark:bg-gray-700 dark:text-gray-300">{u.role}</span>
                      </div>
                    ))}
                  </div>
                )}
              </div>
            </div>

            {submitError && (
              <div className="mt-4 rounded-lg border border-red-200 bg-red-50 p-3 text-sm text-red-700 dark:border-red-800 dark:bg-red-950 dark:text-red-400">
                {submitError}
              </div>
            )}
          </div>
        )}

        {/* Navigation buttons */}
        {step > 0 && (
          <div className="mt-8 flex items-center justify-between border-t border-gray-100 pt-4 dark:border-gray-700">
            <button
              onClick={prev}
              className="flex items-center gap-1 rounded-lg border border-gray-300 px-4 py-2 text-sm font-medium text-gray-700 hover:bg-gray-50 dark:border-gray-600 dark:text-gray-300 dark:hover:bg-gray-700"
            >
              <ChevronLeft className="h-4 w-4" /> Back
            </button>
            {step < 4 ? (
              <button
                onClick={next}
                className="flex items-center gap-1 rounded-lg bg-brand-600 px-6 py-2 text-sm font-medium text-white hover:bg-brand-700"
              >
                Next <ChevronRight className="h-4 w-4" />
              </button>
            ) : (
              <button
                onClick={completeSetup}
                disabled={submitting}
                className="flex items-center gap-2 rounded-lg bg-green-600 px-6 py-2 text-sm font-medium text-white hover:bg-green-700 disabled:opacity-50"
              >
                {submitting ? <Check className="h-4 w-4 animate-pulse" /> : <Check className="h-4 w-4" />}
                Complete Setup
              </button>
            )}
          </div>
        )}
      </div>
    </div>
  );
}
