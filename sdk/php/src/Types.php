<?php
declare(strict_types=1);

namespace Ggid\Sdk;

/**
 * JWT Claims decoded from a verified access token.
 */
final class Claims
{
    public function __construct(
        public readonly string $userId,
        public readonly string $tenantId,
        public readonly array $roles,
        public readonly string $scope,
        public readonly int $exp,
        public readonly int $iat,
        public readonly string $iss,
        public readonly ?string $sub = null,
        public readonly ?string $email = null,
        public readonly ?string $name = null,
    ) {}

    /**
     * Build a Claims instance from a raw JWT payload array.
     */
    public static function fromArray(array $payload): self
    {
        return new self(
            userId: $payload['sub'] ?? $payload['user_id'] ?? '',
            tenantId: $payload['tenant_id'] ?? '',
            roles: $payload['roles'] ?? [],
            scope: $payload['scope'] ?? '',
            exp: $payload['exp'] ?? 0,
            iat: $payload['iat'] ?? 0,
            iss: $payload['iss'] ?? '',
            sub: $payload['sub'] ?? null,
            email: $payload['email'] ?? null,
            name: $payload['name'] ?? null,
        );
    }

    /**
     * Check whether the token has expired.
     */    public function isExpired(int $now = null): bool
    {
        return ($now ?? time()) >= $this->exp;
    }

    /**
     * Check whether the user has a specific role.
     */
    public function hasRole(string $role): bool
    {
        return in_array($role, $this->roles, true);
    }

    /**
     * Check whether the token grants a specific scope.
     */
    public function hasScope(string $scope): bool
    {
        $scopes = explode(' ', $this->scope);
        return in_array($scope, $scopes, true);
    }
}

/**
 * UserInfo returned by the OIDC /userinfo endpoint.
 */
final class UserInfo
{
    public function __construct(
        public readonly string $sub,
        public readonly ?string $name = null,
        public readonly ?string $email = null,
        public readonly array $roles = [],
        public readonly ?string $picture = null,
    ) {}

    public static function fromArray(array $data): self
    {
        return new self(
            sub: $data['sub'] ?? '',
            name: $data['name'] ?? null,
            email: $data['email'] ?? null,
            roles: $data['roles'] ?? [],
            picture: $data['picture'] ?? null,
        );
    }
}

/**
 * Token response from OAuth token exchange or refresh.
 */
final class TokenResponse
{
    public function __construct(
        public readonly string $accessToken,
        public readonly ?string $refreshToken,
        public readonly ?string $idToken,
        public readonly int $expiresIn,
        public readonly string $tokenType = 'Bearer',
    ) {}

    public static function fromArray(array $data): self
    {
        return new self(
            accessToken: $data['access_token'] ?? '',
            refreshToken: $data['refresh_token'] ?? null,
            idToken: $data['id_token'] ?? null,
            expiresIn: $data['expires_in'] ?? 0,
            tokenType: $data['token_type'] ?? 'Bearer',
        );
    }
}

/**
 * Role model.
 */
final class Role
{
    public function __construct(
        public readonly string $id,
        public readonly string $name,
        public readonly string $key,
        public readonly ?string $description = null,
    ) {}

    public static function fromArray(array $data): self
    {
        return new self(
            id: $data['id'] ?? '',
            name: $data['name'] ?? '',
            key: $data['key'] ?? '',
            description: $data['description'] ?? null,
        );
    }
}

/**
 * Permission model.
 */
final class Permission
{
    public function __construct(
        public readonly string $id,
        public readonly string $name,
        public readonly string $resource,
        public readonly string $action,
        public readonly ?string $description = null,
        /** @var Permission[] */
        public readonly array $children = [],
    ) {}

    public static function fromArray(array $data): self
    {
        $children = [];
        foreach ($data['children'] ?? [] as $child) {
            if (is_array($child)) {
                $children[] = self::fromArray($child);
            }
        }
        return new self(
            id: $data['id'] ?? '',
            name: $data['name'] ?? '',
            resource: $data['resource'] ?? '',
            action: $data['action'] ?? '',
            description: $data['description'] ?? null,
            children: $children,
        );
    }
}

/**
 * ABAC evaluation result.
 */
final class ABACResult
{
    public function __construct(
        public readonly bool $allowed,
        public readonly string $reason,
        public readonly array $matchedRules = [],
        public readonly ?string $decision = null,
    ) {}

    public static function fromArray(array $data): self
    {
        return new self(
            allowed: $data['allowed'] ?? $data['matched'] ?? false,
            reason: $data['reason'] ?? '',
            matchedRules: $data['matched_rules'] ?? [],
            decision: $data['decision'] ?? null,
        );
    }
}

/**
 * Permission check result.
 */
final class PermissionCheckResult
{
    public function __construct(
        public readonly bool $allowed,
        public readonly string $reason = '',
        public readonly ?string $matchedBy = null,
    ) {}

    public static function fromArray(array $data): self
    {
        return new self(
            allowed: $data['allowed'] ?? false,
            reason: $data['reason'] ?? '',
            matchedBy: $data['matched_by'] ?? null,
        );
    }
}
