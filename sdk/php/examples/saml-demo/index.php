<?php
/**
 * GGID SAML SSO Demo with Permissions (PHP)
 * Run: GGID_URL=... php -S localhost:3103 index.php
 */
require __DIR__ . '/../../vendor/autoload.php';
use GGID\SAML;

$ggidUrl = getenv('GGID_URL') ?: 'http://localhost:8080';
$entityId = getenv('SP_ENTITY_ID') ?: 'http://localhost:3103/saml/metadata';
$acsUrl = getenv('ACS_URL') ?: 'http://localhost:3103/saml/acs';

// Demo user
$user = ['username' => 'demo_user', 'roles' => ['viewer'],
         'permissions' => ['dashboard:read', 'orders:read', 'inventory:read']];

function hasPermission($user, $perm) {
    return in_array('admin', $user['permissions']) || in_array($perm, $user['permissions']);
}

$path = parse_url($_SERVER['REQUEST_URI'], PHP_URL_PATH);

if ($path === '/') {
    echo renderDashboard($user);
} elseif ($path === '/saml/metadata') {
    header('Content-Type: application/xml');
    echo SAML::generateSPMetadata($entityId, $acsUrl);
} elseif ($path === '/login') {
    $url = SAML::buildAuthnRequestUrl("$ggidUrl/saml/sso", $entityId, $acsUrl, '/');
    header("Location: $url");
} elseif ($path === '/inventory') {
    if (!hasPermission($user, 'inventory:read')) { http_response_code(403); echo render403('inventory:read'); exit; }
    echo renderPage('Inventory', $user, hasPermission($user, 'inventory:write'));
} elseif ($path === '/orders') {
    if (!hasPermission($user, 'orders:read')) { http_response_code(403); echo render403('orders:read'); exit; }
    echo renderPage('Orders', $user, hasPermission($user, 'orders:write'));
}

function renderMenu($user) {
    $items = '<li><a href="/">Dashboard</a></li>';
    if (hasPermission($user, 'orders:read')) $items .= '<li><a href="/orders">Orders</a></li>';
    if (hasPermission($user, 'inventory:read')) $items .= '<li><a href="/inventory">Inventory</a></li>';
    return "<aside><h2>Menu</h2><ul>$items</ul><p>Roles: " . implode(', ', $user['roles']) . '</p></aside>';
}
function renderDashboard($user) {
    return '<html><body>' . renderMenu($user) . '<main><h1>Dashboard</h1><p>' . $user['username'] . '</p></main></body></html>';
}
function renderPage($title, $user, $canWrite) {
    $btn = $canWrite ? '<button>New</button>' : '<p>Read-only</p>';
    return '<html><body>' . renderMenu($user) . "<main><h1>$title</h1>$btn</main></body></html>";
}
function render403($perm) {
    return '<html><body><h1>403</h1><p>Need: ' . htmlspecialchars($perm) . '</p></body></html>';
}
