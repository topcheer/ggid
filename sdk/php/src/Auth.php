<?php
declare(strict_types=1);

namespace Ggid\Sdk;

use Firebase\JWT\JWT;
use Firebase\JWT\Key;
use Firebase\JWT\ExpiredException;
use Firebase\JWT\SignatureInvalidException;
use GuzzleHttp\ClientInterface;

class Auth
{
    private ?array $jwksCache = null;
    private int $jwksCacheTtl = 300; // 5 minutes
    private int $jwksCachedAt = 0;

    public function __construct(
        private readonly ClientInterface $httpClient,
        private readonly string $baseUrl,
    ) {}

    /**
     * Verify a JWT access token using JWKS.
     *
     * @throws GGIDException on invalid, expired, or unverifiable token
     */
    public function verifyToken(string $jwt): Claims
    {
        // Decode header to get kid
        $parts = explode('.', $jwt);
        if (count($parts) !== 3) {
            throw new GGIDException('Invalid token format: expected 3 segments');
        }
        $headerRaw = base64_decode(strtr($parts[0], '-_', '+/'), true);
        if ($headerRaw === false) {
            throw new GGIDException('Invalid token header encoding');
        }
        $header = json_decode($headerRaw, true);
        if (!is_array($header) || !isset($header['kid'])) {
            throw new GGIDException('Token header missing key ID (kid)');
        }

        $key = $this->getKeyForKid($header['kid']);

        try {
            $payload = JWT::decode($jwt, $key);
        } catch (ExpiredException $e) {
            throw new GGIDException('Token has expired', 0, $e);
        } catch (SignatureInvalidException $e) {
            throw new GGIDException('Token signature verification failed', 0, $e);
        } catch (\UnexpectedValueException $e) {
            throw new GGIDException('Token verification failed: ' . $e->getMessage(), 0, $e);
        }

        $payloadArray = json_decode(json_encode($payload), true);
        return Claims::fromArray($payloadArray);
    }

    /**
     * Fetch and cache JWKS from the GGID gateway.
     */
    public function getJwks(): array
    {
        $now = time();
        if ($this->jwksCache !== null && ($now - $this->jwksCachedAt) < $this->jwksCacheTtl) {
            return $this->jwksCache;
        }

        $resp = $this->httpClient->request('GET', $this->baseUrl . '/.well-known/jwks.json');
        $body = json_decode($resp->getBody()->getContents(), true);
        if (!is_array($body) || !isset($body['keys'])) {
            throw new GGIDException('Invalid JWKS response: missing keys array');
        }
        $this->jwksCache = $body;
        $this->jwksCachedAt = $now;
        return $body;
    }

    /**
     * Get OIDC discovery document.
     */
    public function getDiscovery(): array
    {
        $resp = $this->httpClient->request('GET', $this->baseUrl . '/.well-known/openid-configuration');
        return json_decode($resp->getBody()->getContents(), true);
    }

    /**
     * Build a full authorize URL for browser redirect.
     */
    public function getAuthorizeUrl(
        string $clientId,
        string $redirectUri,
        string $scope = 'openid profile email',
        string $state = '',
        ?string $codeChallenge = null,
    ): string {
        $params = [
            'response_type' => 'code',
            'client_id' => $clientId,
            'redirect_uri' => $redirectUri,
            'scope' => $scope,
        ];
        if ($state !== '') {
            $params['state'] = $state;
        }
        if ($codeChallenge !== null) {
            $params['code_challenge'] = $codeChallenge;
            $params['code_challenge_method'] = 'S256';
        }
        return $this->baseUrl . '/api/v1/oauth/authorize?' . http_build_query($params);
    }

    /**
     * Exchange an authorization code for tokens.
     */
    public function exchangeCode(
        string $code,
        string $redirectUri,
        string $clientId,
        string $clientSecret,
        ?string $codeVerifier = null,
    ): TokenResponse {
        $body = [
            'grant_type' => 'authorization_code',
            'code' => $code,
            'redirect_uri' => $redirectUri,
            'client_id' => $clientId,
            'client_secret' => $clientSecret,
        ];
        if ($codeVerifier !== null) {
            $body['code_verifier'] = $codeVerifier;
        }
        $resp = $this->httpClient->request('POST', $this->baseUrl . '/api/v1/oauth/token', [
            'json' => $body,
        ]);
        $data = json_decode($resp->getBody()->getContents(), true);
        return TokenResponse::fromArray($data);
    }

    /**
     * Refresh an access token using a refresh token.
     */
    public function refreshToken(
        string $refreshToken,
        string $clientId,
        string $clientSecret,
    ): TokenResponse {
        $resp = $this->httpClient->request('POST', $this->baseUrl . '/api/v1/oauth/token', [
            'json' => [
                'grant_type' => 'refresh_token',
                'refresh_token' => $refreshToken,
                'client_id' => $clientId,
                'client_secret' => $clientSecret,
            ],
        ]);
        $data = json_decode($resp->getBody()->getContents(), true);
        return TokenResponse::fromArray($data);
    }

    /**
     * Get user info from the OIDC userinfo endpoint.
     */
    public function getUserInfo(string $accessToken): UserInfo
    {
        $resp = $this->httpClient->request('GET', $this->baseUrl . '/api/v1/oauth/userinfo', [
            'headers' => ['Authorization' => 'Bearer ' . $accessToken],
        ]);
        $data = json_decode($resp->getBody()->getContents(), true);
        return UserInfo::fromArray($data);
    }

    /**
     * Revoke a token (RFC 7009).
     */
    public function revokeToken(string $token, string $clientId, string $clientSecret): void
    {
        $this->httpClient->request('POST', $this->baseUrl . '/api/v1/oauth/revoke', [
            'json' => [
                'token' => $token,
                'client_id' => $clientId,
                'client_secret' => $clientSecret,
            ],
        ]);
    }

    /**
     * Introspect a token (RFC 7662).
     */
    public function introspectToken(string $token, string $clientId, string $clientSecret): array
    {
        $resp = $this->httpClient->request('POST', $this->baseUrl . '/api/v1/oauth/introspect', [
            'json' => [
                'token' => $token,
                'client_id' => $clientId,
                'client_secret' => $clientSecret,
            ],
        ]);
        return json_decode($resp->getBody()->getContents(), true);
    }

    /**
     * Resolve a Key object for the given key ID from JWKS.
     */
    private function getKeyForKid(string $kid): Key
    {
        $jwks = $this->getJwks();
        foreach ($jwks['keys'] as $keyData) {
            if (($keyData['kid'] ?? null) === $kid) {
                $alg = $keyData['alg'] ?? 'RS256';
                $pem = $this->jwkToPem($keyData);
                return new Key($pem, $alg);
            }
        }
        // Refresh JWKS once in case keys rotated
        $this->jwksCache = null;
        $jwks = $this->getJwks();
        foreach ($jwks['keys'] as $keyData) {
            if (($keyData['kid'] ?? null) === $kid) {
                $alg = $keyData['alg'] ?? 'RS256';
                $pem = $this->jwkToPem($keyData);
                return new Key($pem, $alg);
            }
        }
        throw new GGIDException("No matching key found for kid: {$kid}");
    }

    /**
     * Convert a JWK (RSA) to PEM format.
     */
    private function jwkToPem(array $jwk): string
    {
        // Handle RSA public keys with n and e parameters
        if (($jwk['kty'] ?? null) === 'RSA') {
            $n = $this->base64UrlDecode($jwk['n'] ?? '');
            $e = $this->base64UrlDecode($jwk['e'] ?? '');
            if ($n === '' || $e === '') {
                throw new GGIDException('Invalid RSA JWK: missing n or e');
            }
            return $this->rsaPublicToPem($n, $e);
        }
        // If x5c is present, use the first certificate
        if (isset($jwk['x5c'][0])) {
            return "-----BEGIN CERTIFICATE-----\n" . chunk_split($jwk['x5c'][0], 64, "\n") . "-----END CERTIFICATE-----\n";
        }
        throw new GGIDException('Unsupported JWK format');
    }

    private function base64UrlDecode(string $data): string
    {
        $padded = strtr($data, '-_', '+/');
        $pad = strlen($padded) % 4;
        if ($pad > 0) {
            $padded .= str_repeat('=', 4 - $pad);
        }
        $decoded = base64_decode($padded, true);
        return $decoded !== false ? $decoded : '';
    }

    private function rsaPublicToPem(string $modulus, string $exponent): string
    {
        // ASN.1 encode the RSA public key
        $modulus = ltrim($modulus, "\0");
        $asn1 = $this->asn1Sequence(
            $this->asn1Integer($modulus) . $this->asn1Integer($exponent)
        );
        $der = $this->asn1Sequence($this->asn1BitString($asn1));
        $b64 = base64_encode($der);
        $pem = "-----BEGIN PUBLIC KEY-----\n";
        $pem .= chunk_split($b64, 64, "\n");
        $pem .= "-----END PUBLIC KEY-----\n";
        return $pem;
    }

    private function asn1Length(int $length): string
    {
        if ($length < 0x80) {
            return chr($length);
        }
        $bytes = '';
        while ($length > 0) {
            $bytes = chr($length & 0xFF) . $bytes;
            $length >>= 8;
        }
        return chr(0x80 | strlen($bytes)) . $bytes;
    }

    private function asn1Sequence(string $content): string
    {
        return "\x30" . $this->asn1Length(strlen($content)) . $content;
    }

    private function asn1Integer(string $value): string
    {
        if (strlen($value) > 0 && (ord($value[0]) & 0x80)) {
            $value = "\x00" . $value;
        }
        return "\x02" . $this->asn1Length(strlen($value)) . $value;
    }

    private function asn1BitString(string $content): string
    {
        return "\x03" . $this->asn1Length(strlen($content) + 1) . "\x00" . $content;
    }
}
