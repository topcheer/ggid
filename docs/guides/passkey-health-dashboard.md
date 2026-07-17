# Passkey Health Dashboard — User Guide

> Feature: F-47 Passkey Health Dashboard (KB-024)
> Location: **Settings > Passkey Health** (`/settings/passkey-health`)

## What It Does

The Passkey Health Dashboard provides a centralized view of passkey adoption, device health, MFA enrollment statistics, and enforcement policy across your organization. Administrators can monitor the transition from traditional passwords to passkey-based authentication and take corrective action on unhealthy credentials.

## How to Access

1. Log in to the GGID Admin Console.
2. Navigate to **Settings** in the sidebar.
3. Click **Passkey Health**.

Alternatively, go to `/settings/passkey-health` directly.

## Tabs and Sections

### 1. Overview

Displays high-level KPIs at a glance:

- **Active Passkeys**: Total number of passkeys currently active across all users.
- **Revoked Passkeys**: Passkeys that have been manually revoked or expired.
- **Registration Sessions**: Ongoing or pending WebAuthn registration flows.
- **Authentication Sessions**: Active WebAuthn authentication events.
- **Sparkline Trend**: 14-day authentication activity chart.

**Workflow — Review daily health:**
1. Open the Overview tab.
2. Check if Active Passkeys is growing over time.
3. Investigate any spike in Revoked Passkeys (may indicate a compromised device).

### 2. Health

Detailed per-credential health status:

- **Healthy**: Credential is functioning correctly.
- **Warning**: Credential hasn't been used in 30+ days.
- **Critical**: Credential hasn't been used in 90+ days or the device is unreachable.

For each credential you can see:
- Username and email
- Device type (mobile, desktop, tablet, security key)
- Last used timestamp
- Health status badge

**Workflow — Identify stale credentials:**
1. Go to the Health tab.
2. Filter by status "Critical".
3. Contact affected users or revoke stale credentials.

### 3. Devices

Device breakdown by type:

- **Mobile**: iOS/Android phones
- **Desktop**: macOS/Windows/Linux
- **Tablet**: iPads and Android tablets
- **Security Key**: Hardware keys (YubiKey, Titan, etc.)

Each device card shows the count, last-seen date, and a quick-action menu.

### 4. Policy

MFA enforcement configuration:

- **Required for Admins**: Toggle whether administrators must use MFA.
- **Required for All Users**: Toggle global MFA requirement.
- **Grace Period (days)**: Number of days new users have before MFA is enforced.
- **Enforced Users**: Count of users currently subject to enforcement.

**Workflow — Enable MFA for all users:**
1. Go to the Policy tab.
2. Toggle **Required for All Users** to ON.
3. Set a reasonable **Grace Period** (e.g., 7 days).
4. Monitor the **Enforced Users** count over the next week.

## API Endpoints

The dashboard calls these backend endpoints:

| Endpoint | Method | Purpose |
|----------|--------|--------|
| `/api/v1/auth/passkeys/status` | GET | Active/revoked counts, session stats |
| `/api/v1/auth/mfa/enrollment-stats` | GET | MFA enrollment rates, method distribution |

## Troubleshooting

| Problem | Cause | Solution |
|---------|-------|--------|
| Dashboard shows "Failed to load" | Auth service unreachable | Check `ggid-auth` pod status: `kubectl get pod -n ggid -l app=ggid-auth` |
| Active passkeys is 0 | No users have enrolled | Direct users to Settings > Passkeys to register |
| MFA stats not updating | Enrollment stats endpoint timeout | Restart auth service or check DB connectivity |
| Cannot toggle policy | Insufficient permissions | Ensure your role has `mfa:policy:write` scope |

## Best Practices

- **Review weekly**: Check the Overview tab at least once a week.
- **Revoke critical credentials**: Don't let stale passkeys linger.
- **Set reasonable grace periods**: 7-14 days for new user onboarding.
- **Monitor method distribution**: If users are only using SMS OTP, encourage passkey migration.
