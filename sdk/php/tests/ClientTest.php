<?php
declare(strict_types=1);

namespace Ggid\Sdk\Tests;

use Ggid\Sdk\GGIDClient;
use Ggid\Sdk\GGIDException;
use Ggid\Sdk\Claims;
use Ggid\Sdk\TokenResponse;
use Ggid\Sdk\Role;
use Ggid\Sdk\Permission;
use Ggid\Sdk\ABACResult;
use Ggid\Sdk\PermissionCheckResult;
use Ggid\Sdk\RequiresAuth;
use Ggid\Sdk\RequirePermission;
use Ggid\Sdk\RequireRole;
use Ggid\Sdk\MiddlewareResolver;
use GuzzleHttp\Client as GuzzleClient;
use GuzzleHttp\Handler\MockHandler;
use GuzzleHttp\HandlerStack;
use GuzzleHttp\Psr7\Response;
use GuzzleHttp\Psr7\Request;
use PHPUnit\Framework\TestCase;

/**
 * @covers \Ggid\Sdk\GGIDClient
 * @covers \Ggid\Sdk\Auth
 * @covers \Ggid\Sdk\RBAC
 * @covers \Ggid\Sdk\ABAC
 * @covers \Ggid\Sdk\Types
 * @covers \Ggid\Sdk\Middleware
 */
class ClientTest extends TestCase
{
    private function mockClient(array $responses, string $baseUrl = 'https://ggid.test', string $tenantId = 'tenant-123'): GGIDClient
    {
        $mock = new MockHandler($responses);
        $handlerStack = HandlerStack::create($mock);
        $guzzle = new GuzzleClient(['handler' => $handlerStack]);
        return new GGIDClient($baseUrl, $tenantId, $guzzle);
    }

    public function testLogin(): void
    {
        $client = $this->mockClient([
            new Response(200, [], json_encode([
                'access_token' => 'jwt-token-123',
                'refresh_token' => 'refresh-456',
                'expires_in' => 3600,
                'token_type' => 'Bearer',
            ])),
        ]);

        $result = $client->login('admin', 'Admin@123456');

        $this->assertEquals('jwt-token-123', $result['access_token']);
        $this->assertEquals('refresh-456', $result['refresh_token']);
        $this->assertEquals(3600, $result['expires_in']);
    }

    public function testRegister(): void
    {
        $client = $this->mockClient([
            new Response(201, [], json_encode([
                'id' => 'user-001',
                'username' => 'newuser',
                'email' => 'new@test.com',
            ])),
        ]);

        $result = $client->register('newuser', 'new@test.com', 'Password123!');

        $this->assertEquals('user-001', $result['id']);
        $this->assertEquals('newuser', $result['username']);
    }

    public function testListUsers(): void
    {
        $client = $this->mockClient([
            new Response(200, [], json_encode([
                ['id' => 'u1', 'username' => 'admin'],
                ['id' => 'u2', 'username' => 'user2'],
            ])),
        ]);

        $users = $client->listUsers('access-token');

        $this->assertCount(2, $users);
        $this->assertEquals('admin', $users[0]['username']);
    }

    public function testCreateRole(): void
    {
        $client = $this->mockClient([
            new Response(201, [], json_encode([
                'id' => 'role-001',
                'name' => 'Editor',
                'key' => 'editor',
                'description' => 'Content editor role',
            ])),
        ]);

        $result = $client->createRole('token', 'Editor', 'editor', 'Content editor role');

        $this->assertEquals('role-001', $result['id']);
        $this->assertEquals('editor', $result['key']);
    }

    public function testListRolesReturnsRoleObjects(): void
    {
        $client = $this->mockClient([
            new Response(200, [], json_encode([
                'roles' => [
                    ['id' => 'r1', 'name' => 'Admin', 'key' => 'admin'],
                    ['id' => 'r2', 'name' => 'User', 'key' => 'user'],
                ],
            ])),
        ]);

        $roles = $client->listRoles('token');

        $this->assertCount(2, $roles);
        $this->assertInstanceOf(Role::class, $roles[0]);
        $this->assertEquals('Admin', $roles[0]->name);
    }

    public function testCheckPermission(): void
    {
        $client = $this->mockClient([
            new Response(200, [], json_encode([
                'allowed' => true,
                'reason' => 'matched by role',
                'matched_by' => 'admin',
            ])),
        ]);

        $result = $client->checkPermission('token', 'products', 'read');

        $this->assertInstanceOf(PermissionCheckResult::class, $result);
        $this->assertTrue($result->allowed);
        $this->assertEquals('matched by role', $result->reason);
    }

    public function testCheckPermissionDenied(): void
    {
        $client = $this->mockClient([
            new Response(200, [], json_encode([
                'allowed' => false,
                'reason' => 'no matching policy',
            ])),
        ]);

        $result = $client->checkPermission('token', 'products', 'delete');

        $this->assertFalse($result->allowed);
    }

    public function testAssignRole(): void
    {
        $client = $this->mockClient([
            new Response(200, [], json_encode(['success' => true])),
        ]);

        $result = $client->assignRole('token', 'user-001', 'role-001');

        $this->assertTrue($result['success']);
    }

    public function testRevokeRole(): void
    {
        $client = $this->mockClient([
            new Response(204, [], ''),
        ]);

        // Should not throw
        $client->revokeRole('token', 'user-001', 'role-001');
        $this->assertTrue(true);
    }

    public function testGetUserRoles(): void
    {
        $client = $this->mockClient([
            new Response(200, [], json_encode([
                ['id' => 'r1', 'name' => 'Admin', 'key' => 'admin'],
            ])),
        ]);

        $roles = $client->getUserRoles('token', 'user-001');

        $this->assertCount(1, $roles);
        $this->assertInstanceOf(Role::class, $roles[0]);
        $this->assertEquals('admin', $roles[0]->key);
    }

    public function testListPermissions(): void
    {
        $client = $this->mockClient([
            new Response(200, [], json_encode([
                ['id' => 'p1', 'name' => 'Read Products', 'resource' => 'products', 'action' => 'read'],
            ])),
        ]);

        $permissions = $client->listPermissions('token');

        $this->assertCount(1, $permissions);
        $this->assertInstanceOf(Permission::class, $permissions[0]);
        $this->assertEquals('products', $permissions[0]->resource);
    }

    public function testEvaluateABAC(): void
    {
        $client = $this->mockClient([
            new Response(200, [], json_encode([
                'allowed' => true,
                'reason' => 'matched ABAC rule',
                'matched_rules' => ['rule-001'],
            ])),
        ]);

        $result = $client->evaluateABAC(
            'token',
            'transfer',
            'inventory',
            'user-001',
            [['field' => 'warehouse', 'operator' => 'eq', 'value' => 'WH-001']],
        );

        $this->assertInstanceOf(ABACResult::class, $result);
        $this->assertTrue($result->allowed);
        $this->assertCount(1, $result->matchedRules);
    }

    public function testCheckPolicy(): void
    {
        $client = $this->mockClient([
            new Response(200, [], json_encode([
                'allowed' => false,
                'reason' => 'no matching policy',
            ])),
        ]);

        $result = $client->checkPolicy('token', 'user-001', 'inventory', 'delete', ['dept' => 'sales']);

        $this->assertFalse($result->allowed);
        $this->assertEquals('no matching policy', $result->reason);
    }

    public function testGetDiscovery(): void
    {
        $client = $this->mockClient([
            new Response(200, [], json_encode([
                'issuer' => 'https://ggid.test',
                'authorization_endpoint' => 'https://ggid.test/api/v1/oauth/authorize',
                'token_endpoint' => 'https://ggid.test/api/v1/oauth/token',
                'jwks_uri' => 'https://ggid.test/.well-known/jwks.json',
                'userinfo_endpoint' => 'https://ggid.test/api/v1/oauth/userinfo',
            ])),
        ]);

        $discovery = $client->getDiscovery();

        $this->assertEquals('https://ggid.test', $discovery['issuer']);
        $this->assertArrayHasKey('jwks_uri', $discovery);
    }

    public function testGetAuthorizeUrl(): void
    {
        $client = $this->mockClient([]);

        $url = $client->getAuthorizeUrl('client-123', 'https://app.test/callback', 'openid profile', 'state-xyz');

        $this->assertStringContainsString('client_id=client-123', $url);
        $this->assertStringContainsString('redirect_uri=', $url);
        $this->assertStringContainsString('state=state-xyz', $url);
        $this->assertStringContainsString('response_type=code', $url);
    }

    public function testExchangeCode(): void
    {
        $client = $this->mockClient([
            new Response(200, [], json_encode([
                'access_token' => 'access-123',
                'refresh_token' => 'refresh-456',
                'id_token' => 'id-789',
                'expires_in' => 3600,
                'token_type' => 'Bearer',
            ])),
        ]);

        $result = $client->exchangeCode('auth-code', 'https://app.test/callback', 'client-123', 'secret');

        $this->assertInstanceOf(TokenResponse::class, $result);
        $this->assertEquals('access-123', $result->accessToken);
        $this->assertEquals('refresh-456', $result->refreshToken);
        $this->assertEquals('id-789', $result->idToken);
    }

    public function testApiErrorThrowsException(): void
    {
        $client = $this->mockClient([
            new Response(404, [], json_encode(['error' => 'not_found'])),
        ]);

        $this->expectException(GGIDException::class);
        $client->getUser('token', 'nonexistent');
    }

    public function testApiErrorPreservesStatusCode(): void
    {
        $client = $this->mockClient([
            new Response(403, [], json_encode(['error' => 'forbidden'])),
        ]);

        try {
            $client->deleteUser('token', 'user-001');
            $this->fail('Expected GGIDException');
        } catch (GGIDException $e) {
            $this->assertEquals(403, $e->getStatusCode());
        }
    }

    public function testClaimsFromToken(): void
    {
        $payload = [
            'sub' => 'user-123',
            'tenant_id' => 'tenant-001',
            'roles' => ['admin', 'editor'],
            'scope' => 'read write',
            'exp' => time() + 3600,
            'iat' => time(),
            'iss' => 'https://ggid.test',
            'email' => 'admin@test.com',
        ];

        $claims = Claims::fromArray($payload);

        $this->assertEquals('user-123', $claims->userId);
        $this->assertTrue($claims->hasRole('admin'));
        $this->assertTrue($claims->hasScope('read'));
        $this->assertFalse($claims->isExpired());
        $this->assertEquals('admin@test.com', $claims->email);
    }

    public function testClaimsExpiredCheck(): void
    {
        $claims = new Claims(
            userId: 'u1',
            tenantId: 't1',
            roles: [],
            scope: '',
            exp: 1000,
            iat: 900,
            iss: 'test',
        );

        $this->assertTrue($claims->isExpired(2000));
    }

    // ─── Middleware Tests ──────────────────────────────────────────

    public function testRequirePermissionAttributeExists(): void
    {
        $reflection = new \ReflectionClass(RequirePermission::class);
        $attrs = $reflection->getAttributes(\Attribute::class);
        $this->assertNotEmpty($attrs);
    }

    public function testRequireRoleAttributeExists(): void
    {
        $reflection = new \ReflectionClass(RequireRole::class);
        $attrs = $reflection->getAttributes(\Attribute::class);
        $this->assertNotEmpty($attrs);
    }

    public function testRequiresAuthAttributeExists(): void
    {
        $reflection = new \ReflectionClass(RequiresAuth::class);
        $attrs = $reflection->getAttributes(\Attribute::class);
        $this->assertNotEmpty($attrs);
    }

    public function testMiddlewareResolverWithNoAttributes(): void
    {
        $client = $this->mockClient([]);
        $resolver = new MiddlewareResolver($client);

        // A test controller with no attributes
        $claims = $resolver->resolve(NoAuthController::class, 'index', null);

        $this->assertEmpty($claims->userId);
    }

    public function testMiddlewareResolverRequiresToken(): void
    {
        $client = $this->mockClient([]);
        $resolver = new MiddlewareResolver($client);

        $this->expectException(GGIDException::class);
        $resolver->resolve(AuthedController::class, 'dashboard', null);
    }

    // ─── Webhook Tests ──────────────────────────────────────────

    public function testListWebhooks(): void
    {
        $client = $this->mockClient([
            new Response(200, [], json_encode([
                ['id' => 'wh-1', 'url' => 'https://example.com/hook', 'events' => ['user.created']],
                ['id' => 'wh-2', 'url' => 'https://example.com/hook2', 'events' => ['role.assigned']],
            ])),
        ]);

        $result = $client->listWebhooks('token');

        $this->assertCount(2, $result);
        $this->assertEquals('wh-1', $result[0]['id']);
        $this->assertEquals('https://example.com/hook', $result[0]['url']);
    }

    public function testCreateWebhook(): void
    {
        $client = $this->mockClient([
            new Response(201, [], json_encode([
                'id' => 'wh-3',
                'url' => 'https://example.com/hook3',
                'events' => ['user.created', 'user.deleted'],
            ])),
        ]);

        $result = $client->createWebhook('token', 'https://example.com/hook3', ['user.created', 'user.deleted']);

        $this->assertEquals('wh-3', $result['id']);
        $this->assertEquals('https://example.com/hook3', $result['url']);
        $this->assertCount(2, $result['events']);
    }

    public function testDeleteWebhook(): void
    {
        $client = $this->mockClient([
            new Response(204, [], ''),
        ]);

        // Should not throw
        $client->deleteWebhook('token', 'wh-1');
        $this->assertTrue(true);
    }

    // ─── Introspect Tests ───────────────────────────────────────

    public function testIntrospectTokenActive(): void
    {
        $client = $this->mockClient([
            new Response(200, [], json_encode([
                'active' => true,
                'sub' => 'user-1',
                'exp' => 1700000000,
                'scope' => 'openid profile',
            ])),
        ]);

        $result = $client->introspectToken('jwt-token', 'client-id', 'secret');

        $this->assertTrue($result['active']);
        $this->assertEquals('user-1', $result['sub']);
        $this->assertEquals('openid profile', $result['scope']);
    }

    public function testIntrospectTokenInactive(): void
    {
        $client = $this->mockClient([
            new Response(200, [], json_encode(['active' => false])),
        ]);

        $result = $client->introspectToken('revoked-token', 'client-id', 'secret');

        $this->assertFalse($result['active']);
    }
}

// Test controllers for middleware
class NoAuthController
{
    public function index(): void {}
}

class AuthedController
{
    #[RequiresAuth]
    public function dashboard(): void {}
}

class AdminController
{
    #[RequiresAuth]
    #[RequireRole('admin')]
    public function adminPanel(): void {}
}

class ProductController
{
    #[RequiresAuth]
    #[RequirePermission('products', 'read')]
    public function list(): void {}
}
