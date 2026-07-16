"use client";
import { useEffect, useState } from "react";
import { useTranslations } from "@/lib/i18n";
import { useOAuthDynamicClientReg, OAuthDynamicClientRegConfig, RegisteredClient } from "@ggid/sdk-react";

export default function OAuthDynamicClientRegPage() {

  const { config, loading, error, fetchConfig, registerClient, deleteClient } = useOAuthDynamicClientReg();
  const [form, setForm] = useState<OAuthDynamicClientRegConfig | null>(null);
  const [showModal, setShowModal] = useState(false);
  const [newClient, setNewClient] = useState({ client_name: "", grant_types: "authorization_code", redirect_uris: "" });
  const t = useTranslations();

  useEffect(() => { fetchConfig(); }, [fetchConfig]);
  useEffect(() => { if (config) setForm(config); }, [config]);

  const handleRegister = async () => {
    await registerClient({
      client_name: newClient.client_name,
      grant_types: newClient.grant_types.split(",").map((s) => s.trim()),
      redirect_uris: newClient.redirect_uris.split(",").map((s) => s.trim()),
    } as Partial<RegisteredClient>);
    setShowModal(false);
    setNewClient({ client_name: "", grant_types: "authorization_code", redirect_uris: "" });
  };

  const handleDelete = async (clientId: string) => {
    if (!confirm(`Delete client ${clientId}?`)) return;
    await deleteClient(clientId);
  };

  if (loading && !form) return <div className="p-8">{t("oauthDCR.loading")}</div>;
  if (error) return <div className="p-8 text-red-600">Error: {error}</div>;
  if (!form) return <div className="p-8">{t("oauthDCR.noData")}</div>;

  return (
    <div className="p-8 space-y-6 max-w-4xl">
      <h1 className="text-2xl font-bold">{t("oauthDCR.title")}</h1>
      <p className="text-gray-600">{t("oauthDCR.subtitle")}</p>

      <div className="bg-white rounded-lg p-6 shadow">
        <label className="block text-sm font-medium mb-1">{t("oauthDCR.registrationEndpoint")}</label>
        <input aria-label="form" type="text" value={form.registration_endpoint} readOnly className="border rounded px-3 py-2 w-full bg-gray-50" />
        <div className="mt-3 flex items-center gap-3">
          <input aria-label="Toggle" type="checkbox" checked={form.software_statement_enabled} readOnly className="w-4 h-4" />
          <label>Software Statement Enabled</label>
        </div>
      </div>

      <div className="bg-white rounded-lg p-6 shadow">
        <div className="flex justify-between items-center mb-4">
          <h2 className="text-lg font-semibold">{t("oauthDCR.registeredClients")}</h2>
          <button onClick={() => setShowModal(true)} className="px-4 py-1 bg-green-600 text-white rounded text-sm hover:bg-green-700">+ Register Client</button>
        </div>
        <table className="w-full text-sm">
          <thead>
            <tr className="border-b text-left">
              <th scope="col" className="py-2">{t("oauthDCR.clientId")}</th>
              <th scope="col">{t("oauthDCR.name")}</th>
              <th scope="col">{t("oauthDCR.grantTypes")}</th>
              <th scope="col">{t("oauthDCR.redirectUris")}</th>
              <th scope="col">{t("oauthDCR.created")}</th>
              <th scope="col">{t("oauthDCR.actions")}</th>
            </tr>
          </thead>
          <tbody>
            {form.registered_clients.map((c: RegisteredClient, i: number) => (
              <tr key={i} className="border-b">
                <td className="py-2 font-mono text-xs">{c.client_id}</td>
                <td>{c.client_name}</td>
                <td>{c.grant_types.join(", ")}</td>
                <td className="break-all max-w-[200px] truncate">{c.redirect_uris.join(", ")}</td>
                <td className="text-xs text-gray-500">{c.created_at}</td>
                <td>
                  <button onClick={() => handleDelete(c.client_id)} className="text-red-600 hover:text-red-800 text-xs">{t("oauthDCR.delete")}</button>
                </td>
              </tr>
            ))}
          </tbody>
        </table>
      </div>

      {showModal && (
        <div className="fixed inset-0 bg-black/50 flex items-center justify-center z-50">
          <div className="bg-white rounded-lg p-6 w-96 space-y-4">
            <h2 className="text-lg font-bold">{t("oauthDCR.registerNew")}</h2>
            <div>
              <label className="block text-sm font-medium mb-1">{t("oauthDCR.clientName")}</label>
              <input aria-label="new Client" type="text" value={newClient.client_name} onChange={(e) => setNewClient({ ...newClient, client_name: e.target.value })} className="border rounded px-3 py-2 w-full" />
            </div>
            <div>
              <label className="block text-sm font-medium mb-1">{t("oauthDCR.grantTypesPlaceholder")}</label>
              <input aria-label="new Client" type="text" value={newClient.grant_types} onChange={(e) => setNewClient({ ...newClient, grant_types: e.target.value })} className="border rounded px-3 py-2 w-full" />
            </div>
            <div>
              <label className="block text-sm font-medium mb-1">{t("oauthDCR.redirectUrisPlaceholder")}</label>
              <input aria-label="new Client" type="text" value={newClient.redirect_uris} onChange={(e) => setNewClient({ ...newClient, redirect_uris: e.target.value })} className="border rounded px-3 py-2 w-full" />
            </div>
            <div className="flex gap-3 justify-end">
              <button onClick={() => setShowModal(false)} className="px-4 py-2 border rounded">{t("oauthDCR.cancel")}</button>
              <button aria-label="action" onClick={handleRegister} className="px-4 py-2 bg-blue-600 text-white rounded">{t("oauthDCR.register")}</button>
            </div>
          </div>
        </div>
      )}
    </div>
  );
}
