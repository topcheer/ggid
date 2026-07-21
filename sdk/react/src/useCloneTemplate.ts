import { useState, useCallback } from "react";

export interface TemplateData {
  roles: string[];
  groups: string[];
  permissions: string[];
  org_id: string;
  attributes: Record<string, string>;
}

export interface SavedTemplate {
  id: string;
  name: string;
  description: string;
  source_user: string;
  data: TemplateData;
  created_at: string;
}

export function useCloneTemplate(baseUrl: string = "") {
  const [templates, setTemplates] = useState<SavedTemplate[]>([]);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const fetchTemplates = useCallback(async () => {
    setLoading(true);
    setError(null);
    try {
      const res = await fetch(`${baseUrl}/api/v1/identity/clone-templates`);
      if (!res.ok) throw new Error(`HTTP ${res.status}`);
      const data = await res.json();
      setTemplates(data.templates || data || []);
    } catch (e: any) {
      setError(e.message);
    } finally {
      setLoading(false);
    }
  }, [baseUrl]);

  const previewTemplate = useCallback(async (userId: string) => {
    setLoading(true);
    setError(null);
    try {
      const res = await fetch(`${baseUrl}/api/v1/identity/users/${userId}/template`);
      if (!res.ok) throw new Error(`HTTP ${res.status}`);
      const data: TemplateData = await res.json();
      return data;
    } catch (e: any) {
      setError(e.message);
      return null;
    } finally {
      setLoading(false);
    }
  }, [baseUrl]);

  const createTemplate = useCallback(async (name: string, description: string, sourceUser: string, data: TemplateData) => {
    setLoading(true);
    setError(null);
    try {
      const res = await fetch(`${baseUrl}/api/v1/identity/clone-templates`, {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ name, description, source_user: sourceUser, data }),
      });
      if (!res.ok) throw new Error(`HTTP ${res.status}`);
      await fetchTemplates();
      return true;
    } catch (e: any) {
      setError(e.message);
      return false;
    } finally {
      setLoading(false);
    }
  }, [baseUrl, fetchTemplates]);

  const applyTemplate = useCallback(async (templateId: string, username: string, email?: string) => {
    setLoading(true);
    setError(null);
    try {
      const res = await fetch(`${baseUrl}/api/v1/identity/clone-templates/apply`, {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ template_id: templateId, username, email }),
      });
      if (!res.ok) throw new Error(`HTTP ${res.status}`);
      return true;
    } catch (e: any) {
      setError(e.message);
      return false;
    } finally {
      setLoading(false);
    }
  }, [baseUrl]);

  return { templates, loading, error, fetchTemplates, previewTemplate, createTemplate, applyTemplate };
}
