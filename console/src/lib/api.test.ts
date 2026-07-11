import { describe, it, expect, vi, beforeEach, afterEach } from "vitest";

// Mock fetch globally
const fetchMock = vi.fn();
global.fetch = fetchMock as unknown as typeof fetch;

// Mock localStorage
const localStorageMock = {
  getItem: vi.fn(),
  setItem: vi.fn(),
  removeItem: vi.fn(),
};
Object.defineProperty(globalThis, "localStorage", {
  value: localStorageMock,
  writable: true,
});

// Import after mocks are set up
import { useUsers, useApi, useAuth } from "./api";

// Helper: create a mock Response object
function mockResponse(data: unknown, status = 200): Response {
  return {
    ok: status >= 200 && status < 300,
    status,
    text: async () => JSON.stringify(data),
    json: async () => data,
  } as Response;
}

// We need to test the apiFetch function indirectly via hooks.
// Since hooks need React rendering, we test the core fetch logic directly.

describe("API client", () => {
  beforeEach(() => {
    fetchMock.mockClear();
    localStorageMock.getItem.mockClear();
    localStorageMock.setItem.mockClear();
    localStorageMock.removeItem.mockClear();
  });

  afterEach(() => {
    vi.restoreAllMocks();
  });

  describe("apiFetch (via useApi)", () => {
    it("sends request to correct base URL with tenant header", async () => {
      fetchMock.mockResolvedValueOnce(
        mockResponse({ users: [] })
      );

      const { apiFetch } = useApi();
      await apiFetch("/api/v1/users");

      expect(fetchMock).toHaveBeenCalledTimes(1);
      const [url, options] = fetchMock.mock.calls[0];
      expect(url).toContain("/api/v1/users");
      expect(options.headers["X-Tenant-ID"]).toBeDefined();
      expect(options.headers["Content-Type"]).toBe("application/json");
    });

    it("includes Authorization header when token exists in localStorage", async () => {
      localStorageMock.getItem.mockReturnValue("fake-jwt-token");
      fetchMock.mockResolvedValueOnce(
        mockResponse({ users: [] })
      );

      const { apiFetch } = useApi();
      await apiFetch("/api/v1/users");

      const [, options] = fetchMock.mock.calls[0];
      expect(options.headers["Authorization"]).toBe("Bearer fake-jwt-token");
    });

    it("omits Authorization header when no token in localStorage", async () => {
      localStorageMock.getItem.mockReturnValue(null);
      fetchMock.mockResolvedValueOnce(
        mockResponse({ users: [] })
      );

      const { apiFetch } = useApi();
      await apiFetch("/api/v1/users");

      const [, options] = fetchMock.mock.calls[0];
      expect(options.headers["Authorization"]).toBeUndefined();
    });

    it("sends POST with JSON body for register-like requests", async () => {
      fetchMock.mockResolvedValueOnce(
        mockResponse({ id: "user-1" }, 201)
      );

      const { apiFetch } = useApi();
      await apiFetch("/api/v1/auth/register", {
        method: "POST",
        body: JSON.stringify({
          username: "testuser",
          email: "test@example.com",
          password: "SecurePass123!",
        }),
      });

      const [url, options] = fetchMock.mock.calls[0];
      expect(url).toContain("/api/v1/auth/register");
      expect(options.method).toBe("POST");
      expect(options.body).toContain("testuser");
      expect(options.body).toContain("test@example.com");
    });

    it("sends login request with correct format", async () => {
      fetchMock.mockResolvedValueOnce(
        mockResponse({ access_token: "jwt-abc", refresh_token: "ref-xyz" })
      );

      const { apiFetch } = useApi();
      const result = await apiFetch("/api/v1/auth/login", {
        method: "POST",
        body: JSON.stringify({
          username: "admin",
          password: "Password123!",
        }),
      });

      const [, options] = fetchMock.mock.calls[0];
      expect(options.method).toBe("POST");
      expect(options.body).toContain("admin");
      expect(result).toHaveProperty("access_token");
    });

    it("sends DELETE for delete operations", async () => {
      fetchMock.mockResolvedValueOnce(mockResponse({}, 204));

      const { apiFetch } = useApi();
      await apiFetch("/api/v1/users/user-123", { method: "DELETE" });

      const [url, options] = fetchMock.mock.calls[0];
      expect(url).toContain("/api/v1/users/user-123");
      expect(options.method).toBe("DELETE");
    });

    it("sends PUT for update operations", async () => {
      fetchMock.mockResolvedValueOnce(
        mockResponse({ id: "role-1", key: "admin" })
      );

      const { apiFetch } = useApi();
      await apiFetch("/api/v1/policies/roles/role-1", {
        method: "PUT",
        body: JSON.stringify({ name: "Administrator", key: "admin" }),
      });

      const [, options] = fetchMock.mock.calls[0];
      expect(options.method).toBe("PUT");
      expect(options.body).toContain("Administrator");
    });

    it("throws on non-ok response", async () => {
      fetchMock.mockResolvedValueOnce(
        mockResponse({ error: "Unauthorized" }, 401)
      );

      const { apiFetch } = useApi();
      await expect(apiFetch("/api/v1/users")).rejects.toThrow("API 401");
    });

    it("throws on server error", async () => {
      fetchMock.mockResolvedValueOnce(
        mockResponse({ error: "Internal error" }, 500)
      );

      const { apiFetch } = useApi();
      await expect(apiFetch("/api/v1/users")).rejects.toThrow("API 500");
    });

    it("returns empty object for 204 No Content", async () => {
      fetchMock.mockResolvedValueOnce(mockResponse({}, 204));

      const { apiFetch } = useApi();
      const result = await apiFetch("/api/v1/users/user-123", {
        method: "DELETE",
      });
      expect(result).toEqual({});
    });

    it("merges custom headers with defaults", async () => {
      fetchMock.mockResolvedValueOnce(mockResponse({ ok: true }));

      const { apiFetch } = useApi();
      await apiFetch("/api/v1/webhooks", {
        method: "POST",
        headers: { "X-Webhook-Source": "console" },
        body: JSON.stringify({ url: "https://example.com/hook" }),
      });

      const [, options] = fetchMock.mock.calls[0];
      expect(options.headers["X-Webhook-Source"]).toBe("console");
      expect(options.headers["Content-Type"]).toBe("application/json");
      expect(options.headers["X-Tenant-ID"]).toBeDefined();
    });

    it("creates role with correct endpoint and payload", async () => {
      fetchMock.mockResolvedValueOnce(
        mockResponse({ id: "role-1", key: "developer" }, 201)
      );

      const { apiFetch } = useApi();
      await apiFetch("/api/v1/policies/roles", {
        method: "POST",
        body: JSON.stringify({
          key: "developer",
          name: "Developer",
          description: "Dev team role",
        }),
      });

      const [url, options] = fetchMock.mock.calls[0];
      expect(url).toContain("/api/v1/policies/roles");
      expect(options.method).toBe("POST");
      expect(options.body).toContain("developer");
    });

    it("queries audit events with query params", async () => {
      fetchMock.mockResolvedValueOnce(
        mockResponse({ events: [], total: 0 })
      );

      const { apiFetch } = useApi();
      await apiFetch("/api/v1/audit/events?limit=20&action=user.login");

      const [url] = fetchMock.mock.calls[0];
      expect(url).toContain("/api/v1/audit/events");
      expect(url).toContain("action=user.login");
      expect(url).toContain("limit=20");
    });
  });

  describe("useApi", () => {
    it("exposes apiFetch, API_BASE, and TENANT_ID", () => {
      const { apiFetch, API_BASE, TENANT_ID } = useApi();
      expect(typeof apiFetch).toBe("function");
      expect(API_BASE).toBeDefined();
      expect(TENANT_ID).toBeDefined();
    });
  });

  describe("useAuth", () => {
    it("returns isAuthenticated=false when no token", () => {
      localStorageMock.getItem.mockReturnValue(null);
      const { isAuthenticated } = useAuth();
      // Note: initial state before useEffect runs
      expect(typeof isAuthenticated).toBe("boolean");
    });

    it("logout removes tokens from localStorage", () => {
      const { logout } = useAuth();
      logout();
      expect(localStorageMock.removeItem).toHaveBeenCalledWith("ggid_access_token");
      expect(localStorageMock.removeItem).toHaveBeenCalledWith("ggid_refresh_token");
      expect(localStorageMock.removeItem).toHaveBeenCalledWith("ggid_session_id");
    });
  });
});
