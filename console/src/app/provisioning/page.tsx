"use client";

import { useEffect, useState, useCallback } from "react";
import { useApi } from "@/lib/api";
import {
  Cloud,
  Plus,
  Trash2,
  Server,
  Users,
  CheckCircle2,
  Clock,
  XCircle,
  Loader2,
  Settings2,
  Database,
  Mail,
  Shield,
  Layers,
} from "lucide-react";
import { useTranslations } from "@/lib/i18n";

// ===== Types =====

interface EnvironmentInfo {
  kubernetesVersion: string;
  operatorNamespace: string;
  availableNamespaces: string[];
  gatewayURL: string;
  databaseDrivers: string[];
  defaultDatabase: {
    driver: string;
    host: string;
    port: number;
    name: string;
  };
  defaultReplicas: number;
  defaultPlan: string;
  defaultSslMode: string;
  existingInstances: number;
  existingTenants: number;
  idpProviders: string[];
}

interface InstanceInfo {
  name: string;
  organizationName: string;
  namespace: string;
  replicas: number;
  database: {
    driver: string;
    host: string;
    port: number;
    name: string;
    sslMode: string;
  };
  adminEmail: string;
  phase: string;
  tenantId: string;
  helmRelease: string;
  createdAt: string;
}

interface TenantInfo {
  name: string;
  slug: string;
  plan: string;
  maxUsers: number;
  adminEmail: string;
  ggidInstanceRef: string;
  phase: string;
  tenantId: string;
  gatewayUrl: string;
  createdAt: string;
}

type Tab = "instances" | "tenants";

// ===== Main Component =====

export default function ProvisioningPage() {
  const t = useTranslations();

  const { apiFetch } = useApi();
  const [tab, setTab] = useState<Tab>("instances");
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [msg, setMsg] = useState<string | null>(null);

  // Environment
  const [env, setEnv] = useState<EnvironmentInfo | null>(null);
  const [envLoading, setEnvLoading] = useState(true);

  // Data
  const [instances, setInstances] = useState<InstanceInfo[]>([]);
  const [tenants, setTenants] = useState<TenantInfo[]>([]);

  // UI
  const [showCreate, setShowCreate] = useState(false);

  // ===== Load environment =====
  const loadEnv = useCallback(async () => {
    setEnvLoading(true);
    try {
      const data = await apiFetch<EnvironmentInfo>("/api/v1/provisioning/environment");
      setEnv(data);
    } catch {
      // Operator might not be running — use fallback defaults
      setEnv({
        kubernetesVersion: "unknown",
        operatorNamespace: "ggid-system",
        availableNamespaces: ["default", "ggid-system"],
        gatewayURL: "",
        databaseDrivers: ["postgres", "mysql", "sqlite"],
        defaultDatabase: { driver: "postgres", host: "localhost", port: 5432, name: "ggid" },
        defaultReplicas: 2,
        defaultPlan: "starter",
        defaultSslMode: "require",
        existingInstances: 0,
        existingTenants: 0,
        idpProviders: ["saml", "oidc", "ldap"],
      });
    } finally {
      setEnvLoading(false);
    }
  }, [apiFetch]);

  // ===== Load instances =====
  const loadInstances = useCallback(async () => {
    setLoading(true);
    setError(null);
    try {
      const data = await apiFetch<{ instances?: InstanceInfo[] }>("/api/v1/provisioning/instances");
      setInstances(data.instances || []);
    } catch (err) {
      setError(err instanceof Error ? err.message : "Failed to load instances. Operator may not be deployed.");
      setInstances([]);
    } finally {
      setLoading(false);
    }
  }, [apiFetch]);

  // ===== Load tenants =====
  const loadTenants = useCallback(async () => {
    setLoading(true);
    setError(null);
    try {
      const data = await apiFetch<{ tenants?: TenantInfo[] }>("/api/v1/provisioning/tenants");
      setTenants(data.tenants || []);
    } catch (err) {
      setError(err instanceof Error ? err.message : "Failed to load tenants. Operator may not be deployed.");
      setTenants([]);
    } finally {
      setLoading(false);
    }
  }, [apiFetch]);

  useEffect(() => {
    loadEnv();
  }, [loadEnv]);

  useEffect(() => {
    if (tab === "instances") {
      loadInstances();
    } else {
      loadTenants();
    }
  }, [tab, loadInstances, loadTenants]);

  // Auto-dismiss messages
  useEffect(() => {
    if (msg) {
      const t = setTimeout(() => setMsg(null), 4000);
      return () => clearTimeout(t);
    }
  }, [msg]);

  // ===== Handlers =====
  const handleDeleteInstance = async (name: string) => {
    if (!confirm(`Delete dedicated instance "${name}"? This will uninstall the Helm release and delete the namespace.`)) return;
    try {
      await apiFetch(`/api/v1/provisioning/instances/${name}`, { method: "DELETE" });
      setMsg(`Instance "${name}" deletion initiated`);
      loadInstances();
    } catch (err) {
      setError(err instanceof Error ? err.message : "Failed to delete instance");
    }
  };

  const handleDeleteTenant = async (name: string) => {
    if (!confirm(`Delete shared tenant "${name}"? This will remove the tenant from the shared instance.`)) return;
    try {
      await apiFetch(`/api/v1/provisioning/tenants/${name}`, { method: "DELETE" });
      setMsg(`Tenant "${name}" deletion initiated`);
      loadTenants();
    } catch (err) {
      setError(err instanceof Error ? err.message : "Failed to delete tenant");
    }
  };

  const refresh = () => {
    loadEnv();
    if (tab === "instances") loadInstances();
    else loadTenants();
  };

  // ===== Render =====
  return (
    <div>
      {/* Header */}
      <div className="mb-6 flex items-center justify-between">
        <div>
          <h1 className="text-2xl font-bold dark:text-gray-100 flex items-center gap-2">
            <Cloud className="h-6 w-6 text-brand-600" />
            Provisioning
          </h1>
          <p className="mt-1 text-sm text-gray-500 dark:text-gray-400">
            Manage dedicated IAM instances and shared tenants via K8s Operator
          </p>
        </div>
        <div className="flex items-center gap-2">
          <button
            onClick={refresh}
            className="flex items-center gap-1.5 rounded-lg border border-gray-300 px-3 py-2 text-sm font-medium text-gray-600 hover:bg-gray-50 dark:border-gray-600 dark:text-gray-300 dark:hover:bg-gray-700"
          >
            <Loader2 className="h-4 w-4" />
            Refresh
          </button>
          <button
            onClick={() => { setShowCreate(!showCreate); setError(null); }}
            className="flex items-center gap-1.5 rounded-lg bg-brand-600 px-4 py-2 text-sm font-medium text-white hover:bg-brand-700"
          >
            <Plus className="h-4 w-4" />
            New {tab === "instances" ? "Instance" : "Tenant"}
          </button>
        </div>
      </div>

      {/* Environment Banner */}
      {env && !envLoading && (
        <EnvBanner env={env} />
      )}

      {/* Messages */}
      {msg && (
        <div role="status" className="mb-4 rounded-lg border border-green-200 bg-green-50 p-3 text-sm text-green-700">
          {msg}
        </div>
      )}
      {error && (
        <div className="mb-4 rounded-lg border border-red-200 bg-red-50 p-3 text-sm text-red-700">
          {error}
          <p className="mt-1 text-xs">
            The K8s Operator may not be deployed. Deploy it with:
            <code className="ml-1 rounded bg-red-100 px-1 py-0.5 text-xs">kubectl apply -f deploy/operator/config/</code>
          </p>
        </div>
      )}

      {/* Create Forms */}
      {showCreate && tab === "instances" && env && (
        <CreateInstanceForm
          env={env}
          onClose={() => setShowCreate(false)}
          onCreated={() => { setShowCreate(false); setMsg("Instance provisioning initiated"); loadInstances(); }}
          apiFetch={apiFetch}
        />
      )}
      {showCreate && tab === "tenants" && env && (
        <CreateTenantForm
          env={env}
          instances={instances}
          onClose={() => setShowCreate(false)}
          onCreated={() => { setShowCreate(false); setMsg("Tenant provisioning initiated"); loadTenants(); }}
          apiFetch={apiFetch}
        />
      )}

      {/* Tabs */}
      <div className="mb-4 flex gap-2 border-b border-gray-200 dark:border-gray-700">
        <TabButton active={tab === "instances"} onClick={() => setTab("instances")} icon={Server} label={`Dedicated Instances (${instances.length})`} />
        <TabButton active={tab === "tenants"} onClick={() => setTab("tenants")} icon={Users} label={`Shared Tenants (${tenants.length})`} />
      </div>

      {/* Content */}
      {loading ? (
        <div className="flex items-center gap-2 text-gray-500">
          <Loader2 className="h-4 w-4 animate-spin" />
          Loading...
        </div>
      ) : tab === "instances" ? (
        instances.length === 0 ? (
          <EmptyState
            icon={Server}
            title="No dedicated instances"
            subtitle="Create a dedicated IAM instance for a large customer with isolated resources."
          />
        ) : (
          <InstanceTable instances={instances} onDelete={handleDeleteInstance} />
        )
      ) : tenants.length === 0 ? (
        <EmptyState
          icon={Users}
          title="No shared tenants"
          subtitle="Create a tenant within a shared GGID instance for multi-tenant isolation."
        />
      ) : (
        <TenantTable tenants={tenants} onDelete={handleDeleteTenant} />
      )}
    </div>
  );
}

// ===== Environment Banner =====

function EnvBanner({ env }: { env: EnvironmentInfo }) {
  return (
    <div className="mb-6 rounded-xl border border-blue-200 bg-blue-50 p-4 dark:border-blue-800 dark:bg-blue-950/30">
      <div className="flex items-center gap-2 mb-3">
        <Settings2 className="h-4 w-4 text-blue-600" />
        <span className="text-sm font-semibold text-blue-900 dark:text-blue-300">Detected Environment</span>
      </div>
      <div className="grid grid-cols-2 gap-3 sm:grid-cols-4 lg:grid-cols-6">
        <EnvStat label="K8s Version" value={env.kubernetesVersion} icon={Cloud} />
        <EnvStat label="Operator NS" value={env.operatorNamespace} icon={Layers} />
        <EnvStat label="Gateway URL" value={env.gatewayURL || "not configured"} icon={Server} />
        <EnvStat label="DB Drivers" value={env.databaseDrivers.join(", ")} icon={Database} />
        <EnvStat label="Instances" value={String(env.existingInstances)} icon={Server} />
        <EnvStat label="Tenants" value={String(env.existingTenants)} icon={Users} />
      </div>
    </div>
  );
}

function EnvStat({ label, value, icon: Icon }: { label: string; value: string; icon: React.ElementType }) {
  return (
    <div className="flex flex-col gap-1">
      <div className="flex items-center gap-1 text-xs text-blue-600 dark:text-blue-400">
        <Icon className="h-3 w-3" />
        {label}
      </div>
      <span className="truncate text-sm font-medium text-blue-900 dark:text-blue-200" title={value}>
        {value}
      </span>
    </div>
  );
}

// ===== Instance Table =====

function InstanceTable({ instances, onDelete }: { instances: InstanceInfo[]; onDelete: (name: string) => void }) {
  return (
    <div className="overflow-x-auto rounded-xl border border-gray-200 bg-white shadow-sm dark:border-gray-700 dark:bg-gray-900">
      <table className="w-full">
        <thead className="border-b border-gray-200 bg-gray-50 dark:border-gray-700 dark:bg-gray-800">
          <tr>
            <th scope="col" className="px-4 py-3 text-left text-xs font-medium uppercase text-gray-500">Name</th>
            <th scope="col" className="px-4 py-3 text-left text-xs font-medium uppercase text-gray-500">Organization</th>
            <th scope="col" className="px-4 py-3 text-left text-xs font-medium uppercase text-gray-500">Namespace</th>
            <th scope="col" className="px-4 py-3 text-left text-xs font-medium uppercase text-gray-500">Database</th>
            <th scope="col" className="px-4 py-3 text-left text-xs font-medium uppercase text-gray-500">Replicas</th>
            <th scope="col" className="px-4 py-3 text-left text-xs font-medium uppercase text-gray-500">Phase</th>
            <th scope="col" className="px-4 py-3 text-left text-xs font-medium uppercase text-gray-500">Tenant ID</th>
            <th scope="col" className="px-4 py-3 text-right text-xs font-medium uppercase text-gray-500">Actions</th>
          </tr>
        </thead>
        <tbody className="divide-y divide-gray-100 dark:divide-gray-700">
          {instances.map((inst: any) => (
            <tr key={inst.name} className="hover:bg-gray-50 dark:hover:bg-gray-800">
              <td className="px-4 py-3">
                <div className="flex items-center gap-2">
                  <Server className="h-4 w-4 text-brand-600" />
                  <span className="text-sm font-medium">{inst.name}</span>
                </div>
              </td>
              <td className="px-4 py-3 text-sm text-gray-600 dark:text-gray-400">{inst.organizationName}</td>
              <td className="px-4 py-3">
                <code className="rounded bg-gray-100 px-1.5 py-0.5 text-xs text-gray-600 dark:bg-gray-700 dark:text-gray-300">
                  {inst.namespace}
                </code>
              </td>
              <td className="px-4 py-3 text-sm">
                <div className="flex items-center gap-1">
                  <Database className="h-3 w-3 text-gray-400" />
                  <span>{inst.database.driver}</span>
                  <span className="text-xs text-gray-400">:{inst.database.port}</span>
                </div>
              </td>
              <td className="px-4 py-3 text-sm">{inst.replicas}</td>
              <td className="px-4 py-3">
                <PhaseBadge phase={inst.phase} />
              </td>
              <td className="px-4 py-3">
                {inst.tenantId ? (
                  <code className="text-xs text-gray-500">{inst.tenantId.slice(0, 12)}...</code>
                ) : (
                  <span className="text-xs text-gray-400">-</span>
                )}
              </td>
              <td className="px-4 py-3 text-right">
                <button
                  onClick={() => onDelete(inst.name)}
                  className="text-gray-400 hover:text-red-500"
                  title="Delete instance"
                >
                  <Trash2 className="h-4 w-4" />
                </button>
              </td>
            </tr>
          ))}
        </tbody>
      </table>
    </div>
  );
}

// ===== Tenant Table =====

function TenantTable({ tenants, onDelete }: { tenants: TenantInfo[]; onDelete: (name: string) => void }) {
  return (
    <div className="overflow-x-auto rounded-xl border border-gray-200 bg-white shadow-sm dark:border-gray-700 dark:bg-gray-900">
      <table className="w-full">
        <thead className="border-b border-gray-200 bg-gray-50 dark:border-gray-700 dark:bg-gray-800">
          <tr>
            <th scope="col" className="px-4 py-3 text-left text-xs font-medium uppercase text-gray-500">Name</th>
            <th scope="col" className="px-4 py-3 text-left text-xs font-medium uppercase text-gray-500">Slug</th>
            <th scope="col" className="px-4 py-3 text-left text-xs font-medium uppercase text-gray-500">Plan</th>
            <th scope="col" className="px-4 py-3 text-left text-xs font-medium uppercase text-gray-500">Max Users</th>
            <th scope="col" className="px-4 py-3 text-left text-xs font-medium uppercase text-gray-500">Admin Email</th>
            <th scope="col" className="px-4 py-3 text-left text-xs font-medium uppercase text-gray-500">Phase</th>
            <th scope="col" className="px-4 py-3 text-left text-xs font-medium uppercase text-gray-500">Gateway URL</th>
            <th scope="col" className="px-4 py-3 text-right text-xs font-medium uppercase text-gray-500">Actions</th>
          </tr>
        </thead>
        <tbody className="divide-y divide-gray-100 dark:divide-gray-700">
          {tenants.map((t: any) => (
            <tr key={t.name} className="hover:bg-gray-50 dark:hover:bg-gray-800">
              <td className="px-4 py-3">
                <div className="flex items-center gap-2">
                  <Users className="h-4 w-4 text-purple-600" />
                  <span className="text-sm font-medium">{t.name}</span>
                </div>
              </td>
              <td className="px-4 py-3">
                <code className="rounded bg-gray-100 px-1.5 py-0.5 text-xs text-gray-600 dark:bg-gray-700 dark:text-gray-300">
                  {t.slug}
                </code>
              </td>
              <td className="px-4 py-3">
                <PlanBadge plan={t.plan} />
              </td>
              <td className="px-4 py-3 text-sm">{t.maxUsers.toLocaleString()}</td>
              <td className="px-4 py-3 text-sm text-gray-600 dark:text-gray-400">{t.adminEmail}</td>
              <td className="px-4 py-3">
                <PhaseBadge phase={t.phase} />
              </td>
              <td className="px-4 py-3">
                {t.gatewayUrl ? (
                  <a
                    href={t.gatewayUrl}
                    target="_blank"
                    rel="noopener noreferrer"
                    className="text-xs text-brand-600 hover:underline"
                  >
                    {t.gatewayUrl}
                  </a>
                ) : (
                  <span className="text-xs text-gray-400">-</span>
                )}
              </td>
              <td className="px-4 py-3 text-right">
                <button
                  onClick={() => onDelete(t.name)}
                  className="text-gray-400 hover:text-red-500"
                  title="Delete tenant"
                >
                  <Trash2 className="h-4 w-4" />
                </button>
              </td>
            </tr>
          ))}
        </tbody>
      </table>
    </div>
  );
}

// ===== Create Instance Form =====

function CreateInstanceForm({
  env,
  onClose,
  onCreated,
  apiFetch,
}: {
  env: EnvironmentInfo;
  onClose: () => void;
  onCreated: () => void;
  apiFetch: <T>(path: string, options?: RequestInit) => Promise<T>;
}) {
  const [submitting, setSubmitting] = useState(false);
  const [formError, setFormError] = useState<string | null>(null);
  const [showAdvanced, setShowAdvanced] = useState(false);

  const handleSubmit = async (e: React.FormEvent<HTMLFormElement>) => {
    e.preventDefault();
    setSubmitting(true);
    setFormError(null);

    const fd = new FormData(e.currentTarget);
    const body: Record<string, unknown> = {
      name: fd.get("name"),
      organizationName: fd.get("organizationName"),
      adminEmail: fd.get("adminEmail"),
      namespace: fd.get("namespace") || undefined,
      replicas: parseInt(fd.get("replicas") as string) || env.defaultReplicas,
      database: {
        driver: fd.get("dbDriver"),
        host: fd.get("dbHost"),
        port: parseInt(fd.get("dbPort") as string) || env.defaultDatabase.port,
        name: fd.get("dbName") || env.defaultDatabase.name,
        sslMode: fd.get("dbSslMode") || env.defaultSslMode,
      },
    };

    // Optional IdP config
    const idpProvider = fd.get("idpProvider") as string;
    if (idpProvider) {
      body.idpConfig = {
        provider: idpProvider,
        entityId: fd.get("idpEntityId") as string,
        ssoUrl: fd.get("idpSsoUrl") as string,
        certificate: fd.get("idpCert") as string || undefined,
      };
    }

    try {
      await apiFetch("/api/v1/provisioning/instances", {
        method: "POST",
        body: JSON.stringify(body),
      });
      onCreated();
    } catch (err) {
      setFormError(err instanceof Error ? err.message : "Failed to create instance");
    } finally {
      setSubmitting(false);
    }
  };

  return (
    <form onSubmit={handleSubmit} className="mb-6 rounded-xl border border-gray-200 bg-white p-6 shadow-sm dark:border-gray-700 dark:bg-gray-800">
      <div className="mb-4 flex items-center justify-between">
        <h3 className="flex items-center gap-2 text-sm font-semibold">
          <Server className="h-4 w-4 text-brand-600" />
          Create Dedicated Instance
        </h3>
        <button type="button" onClick={onClose} className="text-gray-400 hover:text-gray-600">✕</button>
      </div>

      {formError && (
        <div className="mb-4 rounded-lg border border-red-200 bg-red-50 p-2 text-sm text-red-700">{formError}</div>
      )}

      {/* Basic fields */}
      <div className="grid gap-4 sm:grid-cols-2">
        <Field label="Instance Name" name="name" placeholder="acme-corp" required hint="K8s resource name (lowercase, hyphens)" />
        <Field label="Organization Name" name="organizationName" placeholder="ACME Corporation" required />
        <Field label="Admin Email" name="adminEmail" type="email" placeholder="admin@acme.com" required />
        <Field
          label="Namespace"
          name="namespace"
          placeholder={`ggid-acme-corp (auto)`}
          hint="Defaults to ggid-<name>"
        />
      </div>

      {/* Database config */}
      <div className="mt-4">
        <div className="mb-2 flex items-center gap-2 text-xs font-semibold uppercase text-gray-500">
          <Database className="h-3 w-3" />
          Database Configuration
        </div>
        <div className="grid gap-4 sm:grid-cols-2 lg:grid-cols-4">
          <SelectField
            label="Driver"
            name="dbDriver"
            defaultValue={env.defaultDatabase.driver}
            options={env.databaseDrivers.map((d: any) => ({ value: d, label: d }))}
          />
          <Field
            label="DB Host"
            name="dbHost"
            defaultValue={env.defaultDatabase.host}
            placeholder="ggid-postgresql"
            required
          />
          <Field
            label="DB Port"
            name="dbPort"
            type="number"
            defaultValue={String(env.defaultDatabase.port)}
            placeholder="5432"
          />
          <Field
            label="DB Name"
            name="dbName"
            defaultValue={env.defaultDatabase.name}
            placeholder="ggid"
          />
        </div>
        <div className="mt-2 grid gap-4 sm:grid-cols-2">
          <SelectField
            label="SSL Mode (PostgreSQL)"
            name="dbSslMode"
            defaultValue={env.defaultSslMode}
            options={[
              { value: "disable", label: "disable" },
              { value: "require", label: "require" },
              { value: "verify-ca", label: "verify-ca" },
              { value: "verify-full", label: "verify-full" },
            ]}
          />
          <Field
            label="Replicas"
            name="replicas"
            type="number"
            defaultValue={String(env.defaultReplicas)}
            placeholder="2"
          />
        </div>
      </div>

      {/* Advanced: IdP config */}
      <div className="mt-4">
        <button
          type="button"
          onClick={() => setShowAdvanced(!showAdvanced)}
          className="flex items-center gap-1 text-xs font-medium text-brand-600 hover:underline"
        >
          <Shield className="h-3 w-3" />
          {showAdvanced ? "Hide" : "Show"} External IdP Configuration (optional)
        </button>
        {showAdvanced && (
          <div className="mt-3 grid gap-4 sm:grid-cols-2">
            <SelectField
              label="IdP Provider"
              name="idpProvider"
              options={[{ value: "", label: "None" }, ...env.idpProviders.map((p: any) => ({ value: p, label: p.toUpperCase() }))]}
            />
            <Field label="Entity ID / Issuer" name="idpEntityId" placeholder="https://idp.example.com" />
            <Field label="SSO URL" name="idpSsoUrl" placeholder="https://idp.example.com/sso" />
            <Field label="Certificate (PEM)" name="idpCert" placeholder="MIIB..." hint="Base64-encoded X.509 cert" />
          </div>
        )}
      </div>

      {/* Defaults hint */}
      <div className="mt-4 rounded-lg bg-gray-50 p-2 text-xs text-gray-500 dark:bg-gray-900 dark:text-gray-400">
        <strong>Smart defaults applied:</strong> {env.defaultDatabase.driver} on {env.defaultDatabase.host}:{env.defaultDatabase.port},
        {" "}{env.defaultReplicas} replicas, SSL={env.defaultSslMode}. Detected {env.availableNamespaces.length} namespaces.
      </div>

      <button
        type="submit"
        disabled={submitting}
        className="mt-4 flex items-center gap-2 rounded-lg bg-brand-600 px-4 py-2 text-sm font-medium text-white hover:bg-brand-700 disabled:opacity-50"
      >
        {submitting ? <Loader2 className="h-4 w-4 animate-spin" /> : <Plus className="h-4 w-4" />}
        Create Instance
      </button>
    </form>
  );
}

// ===== Create Tenant Form =====

function CreateTenantForm({
  env,
  instances,
  onClose,
  onCreated,
  apiFetch,
}: {
  env: EnvironmentInfo;
  instances: InstanceInfo[];
  onClose: () => void;
  onCreated: () => void;
  apiFetch: <T>(path: string, options?: RequestInit) => Promise<T>;
}) {
  const [submitting, setSubmitting] = useState(false);
  const [formError, setFormError] = useState<string | null>(null);

  const handleSubmit = async (e: React.FormEvent<HTMLFormElement>) => {
    e.preventDefault();
    setSubmitting(true);
    setFormError(null);

    const fd = new FormData(e.currentTarget);
    const body: Record<string, unknown> = {
      name: fd.get("name"),
      slug: fd.get("slug"),
      adminEmail: fd.get("adminEmail"),
      adminPassword: fd.get("adminPassword") || undefined,
      plan: fd.get("plan") || env.defaultPlan,
      maxUsers: parseInt(fd.get("maxUsers") as string) || 1000,
      ggidInstanceRef: fd.get("ggidInstanceRef") || undefined,
    };

    try {
      await apiFetch("/api/v1/provisioning/tenants", {
        method: "POST",
        body: JSON.stringify(body),
      });
      onCreated();
    } catch (err) {
      setFormError(err instanceof Error ? err.message : "Failed to create tenant");
    } finally {
      setSubmitting(false);
    }
  };

  const planLimits: Record<string, number> = {
    free: 100,
    starter: 1000,
    pro: 10000,
    enterprise: 100000,
  };

  return (
    <form onSubmit={handleSubmit} className="mb-6 rounded-xl border border-gray-200 bg-white p-6 shadow-sm dark:border-gray-700 dark:bg-gray-800">
      <div className="mb-4 flex items-center justify-between">
        <h3 className="flex items-center gap-2 text-sm font-semibold">
          <Users className="h-4 w-4 text-purple-600" />
          Create Shared Tenant
        </h3>
        <button type="button" onClick={onClose} className="text-gray-400 hover:text-gray-600">✕</button>
      </div>

      {formError && (
        <div className="mb-4 rounded-lg border border-red-200 bg-red-50 p-2 text-sm text-red-700">{formError}</div>
      )}

      <div className="grid gap-4 sm:grid-cols-2">
        <Field label="Tenant Name" name="name" placeholder="ACME Inc" required />
        <Field label="Slug" name="slug" placeholder="acme" required hint="URL-friendly identifier (unique)" />
        <Field label="Admin Email" name="adminEmail" type="email" placeholder="admin@acme.com" required />
        <Field label="Admin Password" name="adminPassword" type="password" placeholder="••••••••" hint="Leave empty to auto-generate" />
      </div>

      <div className="mt-4 grid gap-4 sm:grid-cols-3">
        <SelectField
          label="Plan"
          name="plan"
          defaultValue={env.defaultPlan}
          options={[
            { value: "free", label: "Free (100 users)" },
            { value: "starter", label: "Starter (1,000 users)" },
            { value: "pro", label: "Pro (10,000 users)" },
            { value: "enterprise", label: "Enterprise (100,000 users)" },
          ]}
          onChange={(e) => {
            const maxField = document.querySelector('input[name="maxUsers"]') as HTMLInputElement;
            if (maxField && planLimits[e.target.value]) {
              maxField.value = String(planLimits[e.target.value]);
            }
          }}
        />
        <Field
          label="Max Users"
          name="maxUsers"
          type="number"
          defaultValue={String(1000)}
          placeholder="1000"
        />
        {instances.length > 0 && (
          <SelectField
            label="Instance (optional)"
            name="ggidInstanceRef"
            options={[
              { value: "", label: "Default shared instance" },
              ...instances.map((i: any) => ({ value: i.name, label: `${i.name} (${i.namespace})` })),
            ]}
          />
        )}
      </div>

      {/* Defaults hint */}
      <div className="mt-4 rounded-lg bg-gray-50 p-2 text-xs text-gray-500 dark:bg-gray-900 dark:text-gray-400">
        <strong>Smart defaults applied:</strong> Plan={env.defaultPlan}, Gateway={env.gatewayURL || "auto-detect"}.
        {" "}Tenant will be provisioned in the shared instance with RLS isolation.
      </div>

      <button
        type="submit"
        disabled={submitting}
        className="mt-4 flex items-center gap-2 rounded-lg bg-brand-600 px-4 py-2 text-sm font-medium text-white hover:bg-brand-700 disabled:opacity-50"
      >
        {submitting ? <Loader2 className="h-4 w-4 animate-spin" /> : <Plus className="h-4 w-4" />}
        Create Tenant
      </button>
    </form>
  );
}

// ===== Reusable UI Components =====

function Field({
  label,
  name,
  placeholder,
  required,
  type = "text",
  defaultValue,
  hint,
}: {
  label: string;
  name: string;
  placeholder?: string;
  required?: boolean;
  type?: string;
  defaultValue?: string;
  hint?: string;
}) {
  return (
    <div>
      <label className="mb-1 block text-xs font-medium text-gray-500 dark:text-gray-400">
        {label}
        {required && <span className="text-red-500"> *</span>}
      </label>
      <input
        name={name}
        type={type}
        required={required}
        placeholder={placeholder}
        defaultValue={defaultValue}
        className="w-full rounded-lg border border-gray-300 px-3 py-2 text-sm dark:border-gray-600 dark:bg-gray-700 dark:text-gray-200 focus:border-brand-500 focus:outline-none"
      />
      {hint && <p className="mt-1 text-xs text-gray-400">{hint}</p>}
    </div>
  );
}

function SelectField({
  label,
  name,
  defaultValue,
  options,
  onChange,
}: {
  label: string;
  name: string;
  defaultValue?: string;
  options: { value: string; label: string }[];
  onChange?: (e: React.ChangeEvent<HTMLSelectElement>) => void;
}) {
  return (
    <div>
      <label className="mb-1 block text-xs font-medium text-gray-500 dark:text-gray-400">
        {label}
      </label>
      <select
        name={name}
        defaultValue={defaultValue || ""}
        onChange={onChange}
        className="w-full rounded-lg border border-gray-300 px-3 py-2 text-sm dark:border-gray-600 dark:bg-gray-700 dark:text-gray-200 focus:border-brand-500 focus:outline-none"
      >
        {options.map((opt: any) => (
          <option key={opt.value} value={opt.value}>{opt.label}</option>
        ))}
      </select>
    </div>
  );
}

function TabButton({
  active,
  onClick,
  icon: Icon,
  label,
}: {
  active: boolean;
  onClick: () => void;
  icon: React.ElementType;
  label: string;
}) {
  return (
    <button
      onClick={onClick}
      className={`flex items-center gap-1.5 px-4 py-2 text-sm font-medium ${
        active
          ? "border-b-2 border-brand-600 text-brand-600"
          : "text-gray-500 hover:text-gray-700 dark:text-gray-400 dark:hover:text-gray-200"
      }`}
     aria-label="Icon">
      <Icon className="h-4 w-4" />
      {label}
    </button>
  );
}

function PhaseBadge({ phase }: { phase: string }) {
  const config: Record<string, { color: string; icon: React.ElementType }> = {
    Ready: { color: "bg-green-100 text-green-700 dark:bg-green-900/30 dark:text-green-400", icon: CheckCircle2 },
    Provisioning: { color: "bg-blue-100 text-blue-700 dark:bg-blue-900/30 dark:text-blue-400", icon: Clock },
    Pending: { color: "bg-yellow-100 text-yellow-700 dark:bg-yellow-900/30 dark:text-yellow-400", icon: Clock },
    Failed: { color: "bg-red-100 text-red-700 dark:bg-red-900/30 dark:text-red-400", icon: XCircle },
    Deleting: { color: "bg-gray-100 text-gray-600 dark:bg-gray-700 dark:text-gray-400", icon: XCircle },
    Suspended: { color: "bg-orange-100 text-orange-700 dark:bg-orange-900/30 dark:text-orange-400", icon: XCircle },
  };

  const c = config[phase] || { color: "bg-gray-100 text-gray-600 dark:bg-gray-700 dark:text-gray-400", icon: Clock };
  const Icon = c.icon;

  return (
    <span className={`inline-flex items-center gap-1 rounded-full px-2 py-0.5 text-xs font-medium ${c.color}`}>
      <Icon className="h-3 w-3" />
      {phase || "Unknown"}
    </span>
  );
}

function PlanBadge({ plan }: { plan: string }) {
  const colors: Record<string, string> = {
    free: "bg-gray-100 text-gray-600 dark:bg-gray-700 dark:text-gray-400",
    starter: "bg-blue-100 text-blue-700 dark:bg-blue-900/30 dark:text-blue-400",
    pro: "bg-purple-100 text-purple-700 dark:bg-purple-900/30 dark:text-purple-400",
    enterprise: "bg-gold-100 text-gold-700 dark:bg-gold-900/30 dark:text-gold-400",
  };
  return (
    <span className={`rounded-full px-2 py-0.5 text-xs font-medium ${colors[plan] || colors.free}`}>
      {plan}
    </span>
  );
}

function EmptyState({
  icon: Icon,
  title,
  subtitle,
}: {
  icon: React.ElementType;
  title: string;
  subtitle: string;
}) {
  return (
    <div className="rounded-xl border border-gray-200 bg-white p-12 text-center shadow-sm dark:border-gray-700 dark:bg-gray-900">
      <Icon className="mx-auto mb-4 h-12 w-12 text-gray-300 dark:text-gray-600" />
      <p className="text-gray-500 dark:text-gray-400">{title}</p>
      <p className="mt-1 text-xs text-gray-400">{subtitle}</p>
    </div>
  );
}
