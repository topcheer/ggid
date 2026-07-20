<?php
/**
 * GGID SDK - WebAuthn / Passkey utilities (PHP)
 *
 * Server-side helpers for WebAuthn credential encoding/decoding.
 */

namespace GGID;

class WebAuthn
{
    /**
     * Register a passkey by calling GGID API.
     * Note: browser-side navigator.credentials.create() must happen on client.
     *
     * @param string $apiBaseUrl GGID API base URL
     * @param string $authToken JWT access token
     * @param string $userId User ID
     * @param string|null $tenantId Tenant ID
     * @param resource|null $curl Custom curl handle
     * @return bool
     */
    public static function registerPasskey(
        string $apiBaseUrl,
        string $authToken,
        string $userId,
        ?string $tenantId = null,
        $curl = null
    ): bool {
        $headers = ["Authorization: Bearer $authToken", "Content-Type: application/json"];
        if ($tenantId) $headers[] = "X-Tenant-ID: $tenantId";

        $body = json_encode(['user_id' => $userId]);
        $ch = $curl ?: curl_init();
        curl_setopt($ch, CURLOPT_URL, "$apiBaseUrl/api/v1/auth/webauthn/register/begin");
        curl_setopt($ch, CURLOPT_POST, true);
        curl_setopt($ch, CURLOPT_POSTFIELDS, $body);
        curl_setopt($ch, CURLOPT_HTTPHEADER, $headers);
        curl_setopt($ch, CURLOPT_RETURNTRANSFER, true);
        $result = curl_exec($ch);
        if (!$curl) curl_close($ch);
        return $result !== false;
    }

    /** Encode bytes to base64url string. */
    public static function bufferToBase64Url(string $data): string
    {
        return rtrim(strtr(base64_encode($data), '+/', '-_'), '=');
    }

    /** Decode base64url string to bytes. */
    public static function base64UrlToBuffer(string $b64url): string
    {
        $padded = strtr($b64url, '-_', '+/');
        $pad = strlen($padded) % 4;
        if ($pad) $padded .= str_repeat('=', 4 - $pad);
        return base64_decode($padded);
    }
}

/**
 * GGID SDK - User Management CRUD (PHP)
 */
class UserManagement
{
    private string $apiBaseUrl;
    private string $authToken;
    private ?string $tenantId;

    public function __construct(string $apiBaseUrl, string $authToken, ?string $tenantId = null)
    {
        $this->apiBaseUrl = $apiBaseUrl;
        $this->authToken = $authToken;
        $this->tenantId = $tenantId;
    }

    private function request(string $method, string $path, ?array $body = null): array
    {
        $headers = ["Authorization: Bearer {$this->authToken}", "Content-Type: application/json"];
        if ($this->tenantId) $headers[] = "X-Tenant-ID: {$this->tenantId}";

        $ch = curl_init();
        curl_setopt($ch, CURLOPT_URL, "{$this->apiBaseUrl}{$path}");
        curl_setopt($ch, CURLOPT_CUSTOMREQUEST, $method);
        curl_setopt($ch, CURLOPT_HTTPHEADER, $headers);
        curl_setopt($ch, CURLOPT_RETURNTRANSFER, true);
        if ($body) curl_setopt($ch, CURLOPT_POSTFIELDS, json_encode($body));

        $result = curl_exec($ch);
        $code = curl_getinfo($ch, CURLINFO_HTTP_CODE);
        curl_close($ch);

        if ($code >= 400) {
            throw new \RuntimeException("GGID API error: $code");
        }
        return json_decode($result, true) ?: [];
    }

    public function createUser(string $username, string $email, ?string $password = null): array
    {
        return $this->request('POST', '/api/v1/users', [
            'username' => $username,
            'email' => $email,
            'password' => $password,
        ]);
    }

    public function getUser(string $userId): array
    {
        return $this->request('GET', "/api/v1/users/$userId");
    }

    public function listUsers(int $page = 1, int $pageSize = 20): array
    {
        return $this->request('GET', "/api/v1/users?page=$page&page_size=$pageSize");
    }

    public function updateUser(string $userId, array $updates): array
    {
        return $this->request('PATCH', "/api/v1/users/$userId", $updates);
    }

    public function deleteUser(string $userId): bool
    {
        $result = $this->request('DELETE', "/api/v1/users/$userId");
        return isset($result['status']) || isset($result['id']);
    }
}
