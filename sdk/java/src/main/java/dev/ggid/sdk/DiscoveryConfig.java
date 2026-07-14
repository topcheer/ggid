package dev.ggid.sdk;

import java.util.List;

public class DiscoveryConfig {
    public String issuer;
    public String authorization_endpoint;
    public String token_endpoint;
    public String userinfo_endpoint;
    public String jwks_uri;
    public String end_session_endpoint;
    public String revocation_endpoint;
    public String introspection_endpoint;
    public String registration_endpoint;
    public String device_authorization_endpoint;
    public List<String> scopes_supported;
    public List<String> response_types_supported;
    public List<String> grant_types_supported;
    public List<String> subject_types_supported;
    public List<String> id_token_signing_alg_values_supported;
    public List<String> token_endpoint_auth_methods_supported;
    public List<String> claims_supported;
    public List<String> code_challenge_methods_supported;
}
