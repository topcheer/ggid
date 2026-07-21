import { useState, useCallback } from "react";
export interface EmailContent { templates: Record<string, string>; variables: string[]; }
export function useEmailTemplateEditor(baseUrl: string = "") {
  const [templates, setTemplates] = useState<Record<string, string>>({});
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const fetchTemplates = useCallback(async () => { setLoading(true); setError(null); try { const res = await fetch(baseUrl + "/api/v1/notification/email-templates"); if (!res.ok) throw new Error("HTTP " + res.status); setTemplates(await res.json()); } catch (e: any) { setError(e.message); } finally { setLoading(false); } }, [baseUrl]);
  const saveTemplate = useCallback(async (name: string, lang: string, content: string) => { setLoading(true); setError(null); try { const res = await fetch(baseUrl + "/api/v1/notification/email-templates", { method: "PUT", headers: { "Content-Type": "application/json" }, body: JSON.stringify({ name, lang, content }) }); if (!res.ok) throw new Error("HTTP " + res.status); return true; } catch (e: any) { setError(e.message); return false; } finally { setLoading(false); } }, [baseUrl]);
  const sendTest = useCallback(async (template: string, content: string, email: string) => { setLoading(true); setError(null); try { const res = await fetch(baseUrl + "/api/v1/notification/email-test", { method: "POST", headers: { "Content-Type": "application/json" }, body: JSON.stringify({ template, content, email }) }); if (!res.ok) throw new Error("HTTP " + res.status); return true; } catch (e: any) { setError(e.message); return false; } finally { setLoading(false); } }, [baseUrl]);
  return { templates, loading, error, fetchTemplates, saveTemplate, sendTest };
}
