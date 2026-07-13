<?php
declare(strict_types=1);

namespace Ggid\Sdk;

use ReflectionClass;
use ReflectionMethod;

// ─── PHP 8 Attributes ──────────────────────────────────────────────

/**
 * Attribute: require authenticated user.
 */
#[\Attribute(\Attribute::TARGET_METHOD | \Attribute::TARGET_CLASS)]
class RequiresAuth {}

/**
 * Attribute: require specific permission.
 */
#[\Attribute(\Attribute::TARGET_METHOD | \Attribute::TARGET_CLASS)]
class RequirePermission
{
    public function __construct(
        public readonly string $resource,
        public readonly string $action,
    ) {}
}

/**
 * Attribute: require specific role.
 */
#[\Attribute(\Attribute::TARGET_METHOD | \Attribute::TARGET_CLASS)]
class RequireRole
{
    public function __construct(
        public readonly string $role,
    ) {}
}

/**
 * Middleware resolver — inspects controller method attributes and enforces auth.
 *
 * Usage with any framework that provides the controller class + method name:
 *
 *   $resolver = new MiddlewareResolver($ggidClient);
 *   $resolver->resolve($controllerClass, $methodName, $requestToken);
 */
class MiddlewareResolver
{
    public function __construct(
        private readonly GGIDClient $client,
    ) {}

    /**
     * Resolve and enforce attributes on a controller method.
     *
     * @param string $controllerClass Fully-qualified class name
     * @param string $methodName Controller method name
     * @param string|null $token Bearer token from the request
     * @return Claims The verified claims (for use in the controller)
     * @throws GGIDException if auth/permission/role check fails
     */
    public function resolve(string $controllerClass, string $methodName, ?string $token): Claims
    {
        $reflection = new ReflectionClass($controllerClass);
        $method = $reflection->getMethod($methodName);

        // Gather attributes from both class and method (method takes priority)
        $attrs = array_merge(
            $reflection->getAttributes(),
            $method->getAttributes(),
        );

        $requiresAuth = false;
        $requirePermission = null;
        $requireRole = null;

        foreach ($attrs as $attr) {
            $instance = $attr->newInstance();
            if ($instance instanceof RequiresAuth) {
                $requiresAuth = true;
            } elseif ($instance instanceof RequirePermission) {
                $requirePermission = $instance;
            } elseif ($instance instanceof RequireRole) {
                $requireRole = $instance;
            }
        }

        if (!$requiresAuth && $requirePermission === null && $requireRole === null) {
            // No auth attributes — skip
            return Claims::fromArray([]);
        }

        // Verify token
        if ($token === null || $token === '') {
            throw new GGIDException('Authentication required: no bearer token provided', 401);
        }

        $claims = $this->client->verifyToken($token);

        // Check role
        if ($requireRole !== null && !$claims->hasRole($requireRole->role)) {
            throw new GGIDException("Required role not found: {$requireRole->role}", 403);
        }

        // Check permission
        if ($requirePermission !== null) {
            $result = $this->client->checkPermission($token, $requirePermission->resource, $requirePermission->action);
            if (!$result->allowed) {
                throw new GGIDException(
                    "Permission denied: {$requirePermission->resource}:{$requirePermission->action}",
                    403
                );
            }
        }

        return $claims;
    }
}

/**
 * Callable middleware for Slim/PSR-15 style frameworks.
 *
 * Usage in Slim:
 *   $app->add(new AuthMiddleware($ggidClient));
 *
 * Usage as callable:
 *   $middleware = AuthMiddleware::create($ggidClient);
 *   $middleware($request, $handler);
 */
class AuthMiddleware
{
    public function __construct(
        private readonly GGIDClient $client,
    ) {}

    /**
     * Create a callable middleware closure.
     */
    public static function create(GGIDClient $client): callable
    {
        return function (object $request, callable $next) use ($client): mixed {
            $token = self::extractToken($request);
            if ($token === null) {
                return self::unauthorizedResponse('Missing or invalid Authorization header');
            }
            try {
                $claims = $client->verifyToken($token);
            } catch (GGIDException $e) {
                return self::unauthorizedResponse($e->getMessage());
            }
            // Attach claims to request for downstream use
            if (method_exists($request, 'withAttribute')) {
                $request = $request->withAttribute('ggid_claims', $claims);
                $request = $request->withAttribute('ggid_token', $token);
            }
            return $next($request);
        };
    }

    /**
     * Create a permission-checking middleware closure.
     */
    public static function requirePermission(GGIDClient $client, string $resource, string $action): callable
    {
        return function (object $request, callable $next) use ($client, $resource, $action): mixed {
            $token = self::extractToken($request);
            if ($token === null) {
                return self::unauthorizedResponse('Missing or invalid Authorization header');
            }
            $result = $client->checkPermission($token, $resource, $action);
            if (!$result->allowed) {
                return self::forbiddenResponse("Permission denied: {$resource}:{$action}");
            }
            return $next($request);
        };
    }

    /**
     * Create a role-checking middleware closure.
     */
    public static function requireRole(GGIDClient $client, string $role): callable
    {
        return function (object $request, callable $next) use ($client, $role): mixed {
            $token = self::extractToken($request);
            if ($token === null) {
                return self::unauthorizedResponse('Missing or invalid Authorization header');
            }
            try {
                $claims = $client->verifyToken($token);
            } catch (GGIDException $e) {
                return self::unauthorizedResponse($e->getMessage());
            }
            if (!$claims->hasRole($role)) {
                return self::forbiddenResponse("Required role: {$role}");
            }
            return $next($request);
        };
    }

    /**
     * Extract Bearer token from request headers.
     */
    private static function extractToken(object $request): ?string
    {
        $authHeader = null;
        if (method_exists($request, 'getHeaderLine')) {
            $authHeader = $request->getHeaderLine('Authorization');
        } elseif (method_exists($request, 'getHeader')) {
            $headers = $request->getHeader('Authorization');
            $authHeader = is_array($headers) ? ($headers[0] ?? '') : (string) $headers;
        } elseif (isset($_SERVER['HTTP_AUTHORIZATION'])) {
            $authHeader = $_SERVER['HTTP_AUTHORIZATION'];
        } elseif (isset($_SERVER['REDIRECT_HTTP_AUTHORIZATION'])) {
            $authHeader = $_SERVER['REDIRECT_HTTP_AUTHORIZATION'];
        }

        if ($authHeader && preg_match('/Bearer\s+(.+)/i', $authHeader, $matches)) {
            return trim($matches[1]);
        }
        return null;
    }

    private static function unauthorizedResponse(string $message): mixed
    {
        if (class_exists('\\Slim\\Psr7\\Response')) {
            $resp = new \Slim\Psr7\Response(401);
            $resp->getBody()->write(json_encode(['error' => 'unauthorized', 'message' => $message]));
            return $resp->withHeader('Content-Type', 'application/json');
        }
        http_response_code(401);
        return json_encode(['error' => 'unauthorized', 'message' => $message]);
    }

    private static function forbiddenResponse(string $message): mixed
    {
        if (class_exists('\\Slim\\Psr7\\Response')) {
            $resp = new \Slim\Psr7\Response(403);
            $resp->getBody()->write(json_encode(['error' => 'forbidden', 'message' => $message]));
            return $resp->withHeader('Content-Type', 'application/json');
        }
        http_response_code(403);
        return json_encode(['error' => 'forbidden', 'message' => $message]);
    }
}
