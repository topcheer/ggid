<?php
/**
 * GGID SAML SSO Demo (PHP)
 * Run: GGID_URL=... php -S localhost:3001 index.php
 */
require __DIR__ . '/../../vendor/autoload.php';

use GGID\SAML;

$ggidUrl = getenv('GGID_URL') ?: 'http://localhost:8080';
$entityId = getenv('SP_ENTITY_ID') ?: 'http://localhost:3001/saml/metadata';
$acsUrl = getenv('ACS_URL') ?: 'http://localhost:3001/saml/acs';

$path = parse_url($_SERVER['REQUEST_URI'], PHP_URL_PATH);

if ($path === '/') {
    echo '<h1>GGID SAML Demo</h1><a href="/login">Login with SAML SSO</a>';
} elseif ($path === '/saml/metadata') {
    header('Content-Type: application/xml');
    echo SAML::generateSPMetadata($entityId, $acsUrl);
} elseif ($path === '/login') {
    $ssoUrl = "$ggidUrl/saml/sso";
    $url = SAML::buildAuthnRequestUrl($ssoUrl, $entityId, $acsUrl, '/profile');
    header("Location: $url");
} elseif ($path === '/saml/acs' && $_SERVER['REQUEST_METHOD'] === 'POST') {
    $response = base64_decode($_POST['SAMLResponse'] ?? '');
    echo '<h1>SAML ACS</h1><pre>' . htmlspecialchars($response) . '</pre><a href="/profile">Continue</a>';
} elseif ($path === '/profile') {
    echo '<h1>Profile</h1><p>Authenticated via SAML SSO</p>';
}
