# API Key Lifecycle Management — User Guide

> Feature: F-46 API Key Lifecycle Management
> Location: **Settings > API Key Lifecycle** (`/settings/api-key-lifecycle`)

## What It Does

The API Key Lifecycle Management page lets administrators create, monitor, rotate, and revoke API keys used for programmatic access to GGID services. It provides a single view of all API keys with their usage statistics, scopes, expiration dates, and status.

## How to Access

1. Log in to the GGID Admin Console.
2. Navigate to **Settings** in the sidebar.
3. Click **API Key Lifecycle**.

Alternatively, go to `/settings/api-key-lifecycle` directly.

## Page Layout

### Key Table

The main table displays all API keys with the following columns:

| Column | Description |
|--------|-------------|
| **Name** | Human-readable label for the key |
| **Scopes** | Permission scopes (e.g., `read:users`, `write:audit`) |
| **Created** | Creation timestamp |
| **Expires** | Expiration date |
| **Last Used** | Most recent API call using this key |
| **Usage** | Total number of API calls |
| **Status** | `active`, `expired`, or `revoked` |
| **Actions** | Rotate / Revoke buttons |

## Workflows

### Create a New API Key

1. Click **Create Key** (top-right).
2. Fill in the form:
   - **Name**: A descriptive label (e.g., "CI/CD Pipeline Key").
   - **Scopes**: Comma-separated permission scopes (e.g., `read:users, write:audit`).
   - **Expires At**: Expiration date (recommended: 90 days).
3. Click **Create**.
4. The key is created and appears in the table.

> **Note:** The API key secret is shown only once at creation time. Store it securely immediately.

### Rotate an API Key

Rotation generates a new key secret while preserving the key name, scopes, and ID. The old secret becomes invalid immediately.

1. Find the key in the table.
2. Click **Rotate**.
3. The new secret is generated. Update your applications with the new secret.

**When to rotate:**
- Suspected compromise
- Regular security policy (e.g., every 90 days)
- Team member departure

### Revoke an API Key

Revocation permanently disables the key. It cannot be undone.

1. Find the key in the table.
2. Click **Revoke**.
3. Confirm the action.

**When to revoke:**
- Key is no longer needed
- Confirmed compromise
- Application decommissioned

### Monitor Key Usage

- Check the **Last Used** column to identify unused keys.
- Check the **Usage** count to understand API call volume.
- Keys with zero usage and past expiration should be revoked.

## API Endpoints

| Endpoint | Method | Purpose |
|----------|--------|--------|
| `/api/v1/auth/api-keys` | GET | List all API keys |
| `/api/v1/auth/api-keys` | POST | Create a new API key |
| `/api/v1/auth/api-keys/:id/rotate` | POST | Rotate (regenerate secret) |
| `/api/v1/auth/api-keys/:id` | DELETE | Revoke an API key |

## Troubleshooting

| Problem | Cause | Solution |
|---------|-------|--------|
| "API key creation not available yet" | Backend endpoint not deployed or misconfigured | Ensure `ggid-auth` pod is running and DB migration for api_keys table is complete |
| Keys list is empty | No keys created or auth failed | Check your auth token is valid; create a key first |
| Rotate button not visible | Key is expired or already revoked | Only active keys can be rotated. Create a new key instead |
| Usage count not updating | Backend not tracking API calls | Verify API key authentication middleware is active |

## Best Practices

- **Principle of least privilege**: Only grant the scopes an application needs.
- **Set expiration dates**: Never create keys without an expiration date. 90 days is recommended.
- **Rotate regularly**: Rotate keys every 90 days or per your security policy.
- **Revoke unused keys**: Review monthly and revoke keys with zero recent usage.
- **Never share keys**: Each application or service should have its own key.
- **Monitor usage spikes**: Sudden usage increases may indicate abuse.
