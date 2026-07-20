<?php
/**
 * GGID OAuth 2.0 Demo (PHP)
 * Run: GGID_URL=... CLIENT_ID=... php -S localhost:3000 index.php
 */
require __DIR__ . '/../../vendor/autoload.php';

use GGID\Client;

$ggidUrl = getenv('GGID_URL') ?: 'http://localhost:8080';
$clientId = getenv('CLIENT_ID') ?: '';
$clientSecret = getenv('CLIENT_SECRET') ?: '';
$redirectUri = getenv('REDIRECT_URI') ?: 'http://localhost:3000/callback';

$path = parse_url($_SERVER['REQUEST_URI'], PHP_URL_PATH);

if ($path === '/') {
    $authUrl = "$ggidUrl/api/v1/oauth/authorize?" . http_build_query([
        'response_type' => 'code', 'client_id' => $clientId,
        'redirect_uri' => $redirectUri, 'scope' => 'openid profile email', 'state' => 'demo',
    ]);
    echo "<h1>GGID OAuth Demo</h1><a href='$authUrl'>Login with GGID</a>";
} elseif ($path === '/callback') {
    $code = $_GET['code'] ?? '';
    if (!$code) { http_response_code(400); exit('Missing code'); }
    $ch = curl_init("$ggidUrl/api/v1/oauth/token");
    curl_setopt_array($ch, [
        CURLOPT_POST => true, CURLOPT_RETURNTRANSFER => true,
        CURLOPT_POSTFIELDS => http_build_query([
            'grant_type' => 'authorization_code', 'code' => $code,
            'redirect_uri' => $redirectUri, 'client_id' => $clientId, 'client_secret' => $clientSecret,
        ]),
        CURLOPT_HTTPHEADER => ['Content-Type: application/x-www-form-urlencoded'],
    ]);
    $tokens = json_decode(curl_exec($ch), true); curl_close($ch);

    $ch = curl_init("$ggidUrl/api/v1/oauth/userinfo");
    curl_setopt_array($ch, [
        CURLOPT_RETURNTRANSFER => true,
        CURLOPT_HTTPHEADER => ["Authorization: Bearer {$tokens['access_token']}"],
    ]);
    $user = json_decode(curl_exec($ch), true); curl_close($ch);

    echo '<h1>OAuth Success</h1><pre>' . json_encode(compact('tokens', 'user'), JSON_PRETTY_PRINT) . '</pre>';
}
