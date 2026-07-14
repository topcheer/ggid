# Console Development Guide

Next.js 15 setup, page structure, hook patterns, API proxy, auth flow, i18n, component library, and testing.

## Project Setup

```bash
# Console is Next.js 15 with App Router
cd console
npm install
npm run dev  # http://localhost:3000
```

### Tech Stack

| Layer | Technology |
|-------|-----------|
| Framework | Next.js 15 (App Router) |
| UI | React 19, Tailwind CSS, Radix UI |
| Data fetching | SWR (via @ggid/react-sdk) |
| i18n | next-intl |
| Auth | GGID OAuth/OIDC |
| Charts | Recharts |

### Directory Structure

```
console/
├── src/
│   ├── app/                    # Next.js App Router pages
│   │   ├── layout.tsx          # Root layout (providers)
│   │   ├── page.tsx            # Dashboard
│   │   ├── users/              # User management
│   │   ├── roles/              # Role management
│   │   ├── organizations/      # Organization management
│   │   ├── audit/              # Audit log viewer
│   │   ├── settings/           # Settings pages
│   │   │   ├── sso/            # SSO configuration
│   │   │   ├── oauth-clients/  # OAuth client management
│   │   │   ├── certificates/   # Certificate management
│   │   │   └── ...
│   │   └── agents/             # AI Agent management
│   ├── components/             # Shared UI components
│   │   ├── ui/                 # Base components (button, table, etc.)
│   │   ├── layout/             # Layout components (sidebar, header)
│   │   └── forms/              # Form components
│   └── lib/                    # Utilities
│       ├── api.ts              # API client
│       └── auth.ts             # Auth helpers
├── messages/                   # i18n translation files
│   ├── en.json
│   └── zh.json
├── next.config.ts
└── tailwind.config.ts
```

## Page Structure

### Standard Page Pattern

```tsx
// app/users/page.tsx
'use client';
import { useUsers, useCreateUser } from '@ggid/react-sdk';
import { DataTable } from '@/components/ui/data-table';
import { PageHeader } from '@/components/layout/page-header';

export default function UsersPage() {
  const { users, loading, error, mutate } = useUsers({ perPage: 50 });
  const { createUser } = useCreateUser();

  return (
    <div className="space-y-6">
      <PageHeader
        title="Users"
        description="Manage user accounts and permissions"
        action={<CreateUserDialog onCreate={createUser} />}
      />
      {error && <ErrorBanner error={error} />}
      <DataTable
        data={users}
        loading={loading}
        columns={[
          { key: 'displayName', label: 'Name' },
          { key: 'email', label: 'Email' },
          { key: 'status', label: 'Status' },
          { key: 'createdAt', label: 'Created' },
        ]}
      />
    </div>
  );
}
```

### Settings Page Pattern

```tsx
// app/settings/sso/page.tsx
'use client';
import { useSsoConfig, useUpdateSsoConfig } from '@ggid/react-sdk';

export default function SsoSettingsPage() {
  const { config, loading } = useSsoConfig();
  const { update } = useUpdateSsoConfig();

  return (
    <SettingsLayout>
      <ConfigForm
        config={config}
        loading={loading}
        onSave={update}
      />
    </SettingsLayout>
  );
}
```

## Hook Patterns

### Data Hooks (SWR-based)

```tsx
// sdk/react/src/useUsers.ts
import useSWR from 'swr';

export function useUsers(opts?: { page?: number; perPage?: number }) {
  const { data, error, mutate } = useSWR(
    ['/api/v1/users', opts],
    () => apiClient.users.list(opts),
    { revalidateOnFocus: false, refreshInterval: 30000 }
  );
  
  return {
    users: data?.users ?? [],
    total: data?.total ?? 0,
    loading: !data && !error,
    error,
    mutate,
  };
}
```

### Mutation Hooks

```tsx
export function useCreateUser() {
  const { mutate: revalidate } = useUsers();
  
  const createUser = async (input: CreateUserInput) => {
    const user = await apiClient.users.create(input);
    await revalidate();  // Refresh list
    return user;
  };
  
  return { createUser };
}
```

### Auth Hook

```tsx
export function useAuth() {
  const { data, error } = useSWR('/auth/session', fetchSession);
  return {
    user: data?.user,
    isAuthenticated: !!data?.user,
    loading: !data && !error,
  };
}
```

## API Proxy

Next.js API routes proxy requests to the gateway, injecting auth:

```ts
// app/api/[...path]/route.ts
import { cookies } from 'next/headers';

export async function GET(request: Request, { params }: { params: { path: string[] } }) {
  return proxyRequest(request, params);
}

async function proxyRequest(request: Request, params: { path: string[] }) {
  const cookieStore = cookies();
  const accessToken = cookieStore.get('access_token')?.value;
  
  const path = '/api/v1/' + params.path.join('/');
  const url = `${process.env.GATEWAY_URL}${path}`;
  
  const headers = new Headers(request.headers);
  headers.set('Authorization', `Bearer ${accessToken}`);
  headers.set('X-Tenant-ID', process.env.TENANT_ID!);
  
  const response = await fetch(url, { headers });
  return new Response(response.body, {
    status: response.status,
    headers: response.headers,
  });
}
```

## Auth Flow

```
1. User visits console.ggid.dev
2. Middleware checks for access_token cookie
   ├── Valid → Render page
   └── Missing/Expired → Redirect to /login
3. Login page → POST /api/auth/login
4. Server exchanges credentials for tokens
5. Set httpOnly cookies: access_token (15min), refresh_token (7d)
6. Redirect to dashboard
```

### Middleware

```ts
// middleware.ts
import { NextResponse } from 'next/server';

export function middleware(request: NextRequest) {
  const token = request.cookies.get('access_token')?.value;
  const isLoginPage = request.nextUrl.pathname === '/login';
  
  if (!token && !isLoginPage) {
    return NextResponse.redirect(new URL('/login', request.url));
  }
  if (token && isLoginPage) {
    return NextResponse.redirect(new URL('/', request.url));
  }
  return NextResponse.next();
}

export const config = {
  matcher: ['/((?!api|_next|favicon|login).*)'],
};
```

## i18n

### Provider Setup

```tsx
// app/layout.tsx
import { NextIntlClientProvider } from 'next-intl';
import { getLocale } from 'next-intl/server';

export default async function RootLayout({ children }) {
  const locale = await getLocale();
  return (
    <html lang={locale}>
      <body>
        <NextIntlClientProvider locale={locale}>
          <GGIDProvider gatewayURL={process.env.NEXT_PUBLIC_GATEWAY_URL!}>
            {children}
          </GGIDProvider>
        </NextIntlClientProvider>
      </body>
    </html>
  );
}
```

### Usage

```tsx
import { useTranslations } from 'next-intl';

function UsersPage() {
  const t = useTranslations('users');
  return <h1>{t('page.title')}</h1>;
}
```

### Translation Files

```json
// messages/en.json
{
  "users": {
    "page": { "title": "Users", "description": "Manage user accounts" },
    "actions": { "create": "Create User", "delete": "Delete" },
    "status": { "active": "Active", "suspended": "Suspended" }
  }
}
```

## Component Library

### Base Components

| Component | Purpose |
|-----------|---------|
| `Button` | Primary, secondary, danger variants |
| `DataTable` | Sortable, paginated table |
| `Dialog` | Modal dialogs |
| `FormField` | Input with label, error, help text |
| `Toast` | Notifications (sonner) |
| `Badge` | Status indicators |
| `Tabs` | Tab navigation |
| `Select` | Dropdown selection |
| `Checkbox` | Toggle input |

### Layout Components

| Component | Purpose |
|-----------|---------|
| `AppShell` | Sidebar + header + content area |
| `Sidebar` | Navigation menu |
| `PageHeader` | Title + description + actions |
| `SettingsLayout` | Settings navigation sidebar |

## Testing

### Component Tests

```tsx
import { render, screen } from '@testing-library/react';
import { UsersPage } from './page';

jest.mock('@ggid/react-sdk', () => ({
  useUsers: () => ({
    users: [{ id: '1', displayName: 'Test', email: 'test@corp.com' }],
    loading: false,
  }),
}));

test('renders user list', () => {
  render(<UsersPage />);
  expect(screen.getByText('Test')).toBeInTheDocument();
});
```

### E2E Tests

```typescript
// Playwright
test('create user flow', async ({ page }) => {
  await page.goto('/users');
  await page.click('text=Create User');
  await page.fill('[name=email]', 'newuser@corp.com');
  await page.fill('[name=displayName]', 'New User');
  await page.click('text=Save');
  await expect(page.locator('text=New User')).toBeVisible();
});
```

## See Also

- [SDK Integration Guide](sdk-integration-guide.md)
- [Gateway Architecture](gateway-architecture.md)
- Frontend i18n Sprint
