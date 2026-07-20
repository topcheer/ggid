package dev.ggid.sdk;

import java.util.List;

public class IntrospectionResult {
    public boolean active;
    public String scope;
    public String client_id;
    public String username;
    public String token_type;
    public long exp;
    public long iat;
    public String sub;
    public List<String> aud;
    public String iss;
    public String tenant_id;
    public String email;
    public List<String> roles;
    public List<String> permissions;  // Fine-grained permissions
}
