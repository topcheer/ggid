# OAuth Scope Design Guide

Best practices for OAuth scope design — granularity, naming, consent UX, hierarchy, delegation, versioning, deprecation.

## Naming Conventions

```
<resource>:<action>

users:read      users:write      users:delete
roles:read      roles:write
audit:read      audit:export
policy:check    policy:write
org:read        org:write
```

## Granularity

| Level | Example | Pros | Cons |
|-------|---------|------|------|
| Fine | `users:email:read` | Minimal disclosure | Scope explosion |
| Medium | `users:read` | Balanced | May over-share |
| Coarse | `read` | Simple | Violates least privilege |

**Recommendation**: Medium granularity (`resource:action`).

## Scope Hierarchy

```
users:write ⊃ users:read      (write implies read)
admin ⊃ users:write ⊃ users:read
```

GGID automatically grants implied scopes.

## Consent UX

```
App "MyApp" requests:
  ☑ Read your profile (profile)
  ☑ Read your email (email)
  ☐ Manage users (users:write)

[Deny]  [Allow]
```

- Group related scopes for UX
- Explain each scope in plain language
- Allow partial consent (except required scopes)

## Delegated Scopes

Agents can only request subset of their registered scopes:
```
Agent registered: [users:read, audit:read]
Token exchange request: [users:read]  ← OK
Token exchange request: [users:write] ← DENIED (not registered)
```

## Versioning

For breaking scope changes:
```
v1: users:read (returns all fields)
v2: users:read (returns limited PII)

Support both during transition, deprecate v1.
```

## Deprecation

1. Mark scope as deprecated in discovery doc
2. Log warning when deprecated scope used
3. Notify developers
4. Remove after 6 months

## See Also

- [OAuth API](../api/oauth.md)
- [Delegation Guide](delegation-guide.md)
- [OAuth 2.1 Migration](oauth-migration-guide.md)
