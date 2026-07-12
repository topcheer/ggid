# WebAuthn Recovery Guide

Guide for handling lost WebAuthn devices, backup authenticators, recovery codes, and account recovery.

## Lost Device Flow

```
User loses security key / phone
  ↓
Option A: Use backup authenticator
Option B: Use recovery code
Option C: Admin-assisted recovery
  ↓
Re-enroll new device
  ↓
Revoke lost device credential
```

## Backup Authenticators

Recommend users enroll 2+ authenticators:

```bash
# Check user's enrolled devices
curl https://api.ggid.example.com/api/v1/webauthn/devices \
  -H "Authorization: Bearer $TOKEN"
```

**Recommendation**: 1 platform (Face ID/Touch ID) + 1 cross-platform (YubiKey).

## Recovery Codes

### Generate

```bash
curl -X POST https://api.ggid.example.com/api/v1/webauthn/recovery-codes \
  -H "Authorization: Bearer $TOKEN"
```

10 single-use codes, Argon2id hashed, shown once.

### Use Recovery Code

```bash
curl -X POST https://api.ggid.example.com/api/v1/webauthn/recovery \
  -d '{"code":"abc12-def34-ghi56","new_credential":{...}}'
```

## Account Recovery Verification

Admin-assisted recovery for users who lost all factors:

1. **Identity proofing**: Verify via email + phone + manager approval
2. **Admin clears WebAuthn**: `DELETE /api/v1/users/$ID/webauthn/devices`
3. **Force password reset**: User sets new password
4. **Re-enroll WebAuthn**: User registers new device
5. **Audit**: `user.recovery` event logged with admin ID

```bash
# Admin clears all WebAuthn devices for user
curl -X DELETE https://api.ggid.example.com/api/v1/users/$USER_ID/webauthn/devices \
  -H "Authorization: Bearer $ADMIN_TOKEN" \
  -d '{"reason":"Lost all authenticators","ticket":"HELP-1234"}'
```

## Multiple Device Management

```bash
# List devices
curl https://api.ggid.example.com/api/v1/webauthn/devices \
  -H "Authorization: Bearer $TOKEN"

# Remove specific device
curl -X DELETE https://api.ggid.example.com/api/v1/webauthn/devices/$CRED_ID \
  -H "Authorization: Bearer $TOKEN"

# Rename device
curl -X PUT https://api.ggid.example.com/api/v1/webauthn/devices/$CRED_ID \
  -H "Authorization: Bearer $TOKEN" \
  -d '{"name":"Work YubiKey"}'
```

## Best Practices

- [ ] Users enrolled with 2+ authenticators
- [ ] Recovery codes generated and stored safely
- [ ] Admin recovery procedure documented
- [ ] Identity proofing required for admin-assisted recovery
- [ ] All recovery events audited
- [ ] Lost devices revoked immediately

## See Also

- [WebAuthn Deploy](webauthn-deploy.md)
- [Passkey Conditional UI](passkey-conditional-ui.md)
- [MFA Setup](mfa-setup.md)
