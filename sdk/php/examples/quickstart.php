<?php
/**
 * GGID PHP SDK — Quick Start Example
 *
 * Run: php examples/quickstart.php
 * (Requires: composer install first)
 */

require_once __DIR__ . '/../vendor/autoload.php';

use Ggid\Sdk\GGIDClient;
use Ggid\Sdk\GGIDException;

// ─── Configuration ──────────────────────────────────────────────────
$baseUrl = getenv('GGID_BASE_URL') ?: 'https://ggid.iot2.win';
$tenantId = getenv('GGID_TENANT_ID') ?: '00000000-0000-0000-0000-000000000001';
$username = getenv('GGID_USERNAME') ?: 'admin';
$password = getenv('GGID_PASSWORD') ?: 'Admin@123456';

echo "=== GGID PHP SDK Quick Start ===\n\n";

// ─── 1. Initialize ──────────────────────────────────────────────────
$ggid = new GGIDClient($baseUrl, $tenantId);
echo "1. Client initialized: {$baseUrl}\n";

// ─── 2. Login ───────────────────────────────────────────────────────
try {
    $tokens = $ggid->login($username, $password);
    echo "2. Login successful! Access token: " . substr($tokens['access_token'], 0, 20) . "...\n";
} catch (GGIDException $e) {
    echo "2. Login failed: " . $e->getMessage() . "\n";
    exit(1);
}

$accessToken = $tokens['access_token'];

// ─── 3. Get User Info ───────────────────────────────────────────────
try {
    $userInfo = $ggid->getUserInfo($accessToken);
    echo "3. User info: {$userInfo->sub} ({$userInfo->email})\n";
} catch (GGIDException $e) {
    echo "3. UserInfo failed: " . $e->getMessage() . "\n";
}

// ─── 4. Check Permission ────────────────────────────────────────────
try {
    $result = $ggid->checkPermission($accessToken, 'products', 'read');
    echo "4. Permission check (products:read): " . ($result->allowed ? 'ALLOWED' : 'DENIED') . "\n";
    if (!$result->allowed) {
        echo "   Reason: {$result->reason}\n";
    }
} catch (GGIDException $e) {
    echo "4. Permission check failed: " . $e->getMessage() . "\n";
}

// ─── 5. List Roles ──────────────────────────────────────────────────
try {
    $roles = $ggid->listRoles($accessToken);
    echo "5. Roles found: " . count($roles) . "\n";
    foreach (array_slice($roles, 0, 5) as $role) {
        echo "   - {$role->name} (key: {$role->key})\n";
    }
} catch (GGIDException $e) {
    echo "5. List roles failed: " . $e->getMessage() . "\n";
}

// ─── 6. List Users ──────────────────────────────────────────────────
try {
    $users = $ggid->listUsers($accessToken);
    echo "6. Users found: " . count($users) . "\n";
    foreach (array_slice($users, 0, 5) as $user) {
        $name = $user['username'] ?? $user['email'] ?? 'unknown';
        echo "   - {$name}\n";
    }
} catch (GGIDException $e) {
    echo "6. List users failed: " . $e->getMessage() . "\n";
}

// ─── 7. ABAC Evaluation ─────────────────────────────────────────────
try {
    $abacResult = $ggid->evaluateABAC(
        $accessToken,
        'transfer',
        'inventory',
        $tokens['user_id'] ?? 'user-001',
        [
            ['field' => 'warehouse', 'operator' => 'eq', 'value' => 'WH-001'],
        ],
    );
    echo "7. ABAC evaluation (inventory:transfer): " . ($abacResult->allowed ? 'ALLOWED' : 'DENIED') . "\n";
    if ($abacResult->allowed) {
        echo "   Matched rules: " . implode(', ', $abacResult->matchedRules) . "\n";
    }
} catch (GGIDException $e) {
    echo "7. ABAC evaluation failed: " . $e->getMessage() . "\n";
}

// ─── 8. Audit Events ────────────────────────────────────────────────
try {
    $events = $ggid->listAuditEvents($accessToken, ['limit' => 5]);
    $count = is_array($events) ? count($events) : 0;
    echo "8. Audit events: {$count} recent\n";
} catch (GGIDException $e) {
    echo "8. Audit query failed: " . $e->getMessage() . "\n";
}

echo "\n=== Done! ===\n";
