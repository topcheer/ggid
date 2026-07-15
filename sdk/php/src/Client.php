<?php
declare(strict_types=1);

namespace Ggid\Sdk;

use GuzzleHttp\Client as GuzzleClient;
use GuzzleHttp\ClientInterface;
use GuzzleHttp\Exception\RequestException;

/**
 * GGID SDK Exception.
 */
class GGIDException extends \RuntimeException
{
    private ?int $statusCode;
    private mixed $body;

    public function __construct(string $message, int $statusCode = 0, mixed $body = null, ?\Throwable $previous = null)
    {
        parent::__construct($message, 0, $previous);
        $this->statusCode = $statusCode;
        $this->body = $body;
    }

    public function getStatusCode(): ?int
    {
        return $this->statusCode;
    }

    public function getBody(): mixed
    {
        return $this->body;
    }
}

/**
 * Main GGID API client.
 *
 * Usage:
 *   $ggid = new GGIDClient('https://ggid.iot2.win', 'tenant-uuid');
 *   $tokens = $ggid->login('admin', 'Admin@123456');
 *   $claims = $ggid->verifyToken($tokens['access_token']);
 *   $allowed = $ggid->checkPermission($token, 'products', 'read');
 */
class GGIDClient
{
    use RBAC;
    use ABAC;

    private string $baseUrl;
    private string $tenantId;
    private ClientInterface $httpClient;
    private ?Auth $auth = null;

    /**
     * @param string $baseUrl GGID gateway URL (e.g. https://ggid.iot2.win)
     * @param string $tenantId Tenant UUID
     * @param ClientInterface|null $httpClient Optional custom HTTP client
     * @param int $timeout Request timeout in seconds
     */
    public function __construct(
        string $baseUrl,
        string $tenantId = '00000000-0000-0000-0000-000000000001',
        ?ClientInterface $httpClient = null,
        int $timeout = 30,
    ) {
        $this->baseUrl = rtrim($baseUrl, '/');
        $this->tenantId = $tenantId;
        $this->httpClient = $httpClient ?? new GuzzleClient([
            'timeout' => $timeout,
            'connect_timeout' => 10,
        ]);
    }

    // ─── JWT Verification ──────────────────────────────────────────

    /**
     * Verify a JWT access token and return decoded claims.
     *
     * @throws GGIDException on invalid or expired token
     */
    public function verifyToken(string $jwt): Claims
    {
        return $this->getAuth()->verifyToken($jwt);
    }

    // ─── Authentication ────────────────────────────────────────────

    /**
     * Register a new user.
     */
    public function register(string $username, string $email, string $password): array
    {
        return $this->request('POST', '/api/v1/auth/register', [
            'username' => $username,
            'email' => $email,
            'password' => $password,
        ]);
    }

    /**
     * Login and obtain tokens.
     */
    public function login(string $username, string $password): array
    {
        return $this->request('POST', '/api/v1/auth/login', [
            'username' => $username,
            'password' => $password,
        ]);
    }

    // ─── User Management ──────────────────────────────────────────

    public function getUser(string $token, string $userId): array
    {
        return $this->request('GET', "/api/v1/users/{$userId}", null, $token);
    }

    public function listUsers(string $token, array $params = []): array
    {
        return $this->request('GET', '/api/v1/users', null, $token, $params);
    }

    public function createUser(string $token, array $data): array
    {
        return $this->request('POST', '/api/v1/users', $data, $token);
    }

    public function updateUser(string $token, string $userId, array $data): array
    {
        return $this->request('PUT', "/api/v1/users/{$userId}", $data, $token);
    }

    public function deleteUser(string $token, string $userId): void
    {
        $this->request('DELETE', "/api/v1/users/{$userId}", null, $token);
    }

    // ─── OAuth/OIDC ───────────────────────────────────────────────

    public function getDiscovery(): array
    {
        return $this->getAuth()->getDiscovery();
    }

    public function getJwks(): array
    {
        return $this->getAuth()->getJwks();
    }

    public function getAuthorizeUrl(
        string $clientId,
        string $redirectUri,
        string $scope = 'openid profile email',
        string $state = '',
    ): string {
        return $this->getAuth()->getAuthorizeUrl($clientId, $redirectUri, $scope, $state);
    }

    public function exchangeCode(
        string $code,
        string $redirectUri,
        string $clientId,
        string $clientSecret,
    ): TokenResponse {
        return $this->getAuth()->exchangeCode($code, $redirectUri, $clientId, $clientSecret);
    }

    public function refreshToken(
        string $refreshToken,
        string $clientId,
        string $clientSecret,
    ): TokenResponse {
        return $this->getAuth()->refreshToken($refreshToken, $clientId, $clientSecret);
    }

    public function getUserInfo(string $accessToken): UserInfo
    {
        return $this->getAuth()->getUserInfo($accessToken);
    }

    public function revokeToken(string $token, string $clientId, string $clientSecret): void
    {
        $this->getAuth()->revokeToken($token, $clientId, $clientSecret);
    }

    public function introspectToken(string $token, string $clientId, string $clientSecret): array
    {
        return $this->getAuth()->introspectToken($token, $clientId, $clientSecret);
    }

    // ─── Roles CRUD ───────────────────────────────────────────────

    public function createRole(string $token, string $name, string $key, string $description = ''): array
    {
        return $this->request('POST', '/api/v1/roles', [
            'name' => $name,
            'key' => $key,
            'description' => $description,
        ], $token);
    }

    public function getRole(string $token, string $roleId): array
    {
        return $this->request('GET', "/api/v1/roles/{$roleId}", null, $token);
    }

    public function updateRole(string $token, string $roleId, ?string $name = null, ?string $description = null): array
    {
        $body = [];
        if ($name !== null) {
            $body['name'] = $name;
        }
        if ($description !== null) {
            $body['description'] = $description;
        }
        return $this->request('PUT', "/api/v1/roles/{$roleId}", $body, $token);
    }

    public function deleteRole(string $token, string $roleId): void
    {
        $this->request('DELETE', "/api/v1/roles/{$roleId}", null, $token);
    }

    // ─── Webhooks ────────────────────────────────────────────────

    /**
     * List all webhooks in the tenant.
     */
    public function listWebhooks(string $token): array
    {
        return $this->request('GET', '/api/v1/webhooks', null, $token);
    }

    /**
     * Create a new webhook.
     *
     * @param string $token Admin token
     * @param string $url Webhook endpoint URL
     * @param array $events Event types to subscribe to (e.g. ['user.created', 'user.deleted'])
     */
    public function createWebhook(string $token, string $url, array $events): array
    {
        return $this->request('POST', '/api/v1/webhooks', [
            'url' => $url,
            'events' => $events,
        ], $token);
    }

    /**
     * Delete a webhook by ID.
     */
    public function deleteWebhook(string $token, string $webhookId): void
    {
        $this->request('DELETE', "/api/v1/webhooks/{$webhookId}", null, $token);
    }

    // ─── Agent Identity ───────────────────────────────────────────

    /**
     * Register a new AI agent.
     */
    public function registerAgent(
        string $token,
        string $name,
        string $type,
        array $allowedScopes,
        string $ownerUserId = '',
        string $description = '',
        int $maxDelegationDepth = 3,
        int $rateLimitPerMin = 60,
    ): array {
        return $this->request('POST', '/api/v1/agents/register', [
            'name' => $name,
            'type' => $type,
            'owner_user_id' => $ownerUserId,
            'description' => $description,
            'allowed_scopes' => $allowedScopes,
            'max_delegation_depth' => $maxDelegationDepth,
            'rate_limit_per_min' => $rateLimitPerMin,
        ], $token);
    }

    /**
     * List all agents in the tenant.
     */
    public function listAgents(string $token): array
    {
        return $this->request('GET', '/api/v1/agents', null, $token);
    }

    /**
     * Exchange a user token for an agent-scoped token.
     */
    public function exchangeAgentToken(string $agentId, string $subjectToken, array $scopes = []): array
    {
        return $this->request('POST', '/api/v1/agents/token', [
            'agent_id' => $agentId,
            'subject_token' => $subjectToken,
            'scope' => $scopes,
        ]);
    }

    /**
     * Verify an agent token and return its claims.
     */
    public function verifyAgentToken(string $token): array
    {
        return $this->request('POST', '/api/v1/agents/verify', ['token' => $token]);
    }

    // ─── Access Request (IGA) ────────────────────────────────────

    /**
     * Create an access request.
     */
    public function createAccessRequest(
        string $token,
        string $userId,
        string $resource,
        string $action,
        string $reason = '',
    ): array {
        return $this->request('POST', '/api/v1/access-requests', [
            'user_id' => $userId,
            'resource' => $resource,
            'action' => $action,
            'reason' => $reason,
        ], $token);
    }

    /**
     * List access requests in the tenant.
     */
    public function listAccessRequests(string $token): array
    {
        return $this->request('GET', '/api/v1/access-requests', null, $token);
    }

    /**
     * Approve an access request.
     */
    public function approveAccessRequest(string $token, string $requestId, string $comment = ''): array
    {
        return $this->request('POST', "/api/v1/access-requests/{$requestId}/approve", [
            'comment' => $comment,
        ], $token);
    }

    /**
     * Reject an access request.
     */
    public function rejectAccessRequest(string $token, string $requestId, string $comment = ''): array
    {
        return $this->request('POST', "/api/v1/access-requests/{$requestId}/reject", [
            'comment' => $comment,
        ], $token);
    }

    // ─── Audit ────────────────────────────────────────────────────

    public function listAuditEvents(string $token, array $params = []): array
    {
        return $this->request('GET', '/api/v1/audit/events', null, $token, $params);
    }

    // ─── Internal ─────────────────────────────────────────────────

    /**
     * Core HTTP request method.
     *
     * @param string $method HTTP method (GET, POST, PUT, DELETE)
     * @param string $path API path (without base URL)
     * @param array|null $body JSON body for POST/PUT
     * @param string|null $token Bearer token for auth
     * @param array $params Query parameters
     * @return array Decoded JSON response
     * @throws GGIDException on HTTP error
     */
    protected function request(
        string $method,
        string $path,
        ?array $body = null,
        ?string $token = null,
        array $params = [],
    ): array {
        $url = $this->baseUrl . $path;
        $options = [
            'headers' => $this->buildHeaders($token),
        ];
        if (!empty($params)) {
            $options['query'] = $params;
        }
        if ($body !== null) {
            $options['json'] = $body;
        }

        try {
            $resp = $this->httpClient->request($method, $url, $options);
        } catch (RequestException $e) {
            $resp = $e->getResponse();
            $statusCode = $resp?->getStatusCode() ?? 0;
            $errorBody = null;
            if ($resp !== null) {
                $contents = $resp->getBody()->getContents();
                $errorBody = json_decode($contents, true) ?? $contents;
            }
            throw new GGIDException(
                "API error {$statusCode} for {$method} {$path}",
                $statusCode,
                $errorBody,
                $e
            );
        }

        $statusCode = $resp->getStatusCode();
        $contents = $resp->getBody()->getContents();
        if ($statusCode === 204 || $contents === '') {
            return [];
        }
        $decoded = json_decode($contents, true);
        return is_array($decoded) ? $decoded : [];
    }

    /**
     * Build HTTP headers for a request.
     */
    private function buildHeaders(?string $token): array
    {
        $headers = [
            'Content-Type' => 'application/json',
            'Accept' => 'application/json',
            'X-Tenant-ID' => $this->tenantId,
        ];
        if ($token !== null) {
            $headers['Authorization'] = 'Bearer ' . $token;
        }
        return $headers;
    }

    /**
     * Lazily create the Auth helper.
     */
    private function getAuth(): Auth
    {
        if ($this->auth === null) {
            $this->auth = new Auth($this->httpClient, $this->baseUrl);
        }
        return $this->auth;
    }
}
