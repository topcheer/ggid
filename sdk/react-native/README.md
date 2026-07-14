# GGID React Native SDK

Simplest IAM integration for React Native / Expo apps.

## Quick Start

```bash
npm install @ggid/react-native-sdk
```

### 1. Initialize

```tsx
import { useGGIDAuth } from '@ggid/react-native-sdk';

// In your App root:
useGGIDAuth.getState().init({
  baseUrl: 'https://ggid.iot2.win',
  tenantId: '00000000-0000-0000-0000-000000000001',
  clientId: 'gcid__sbYZX3_2aJ4eDz-Oy1qRQ',
  redirectUri: 'exp://localhost:8081/+redirect',
});
```

### 2. Login Button

```tsx
function LoginScreen() {
  const { login, loading, error } = useGGIDAuth();
  return (
    <Button title="GGID Login" onPress={login} loading={loading} />
  );
}
```

### 3. Show User Info

```tsx
function Profile() {
  const user = useUser();
  return <Text>{user?.name} ({user?.email})</Text>;
}
```

### 4. Check Permission

```tsx
function AdminPanel() {
  const [allowed, setAllowed] = useState(false);
  const checkPermission = useGGIDAuth((s) => s.checkPermission);

  useEffect(() => {
    checkPermission('admin', 'read').then(setAllowed);
  }, []);

  if (!allowed) return <Text>Access denied</Text>;
  return <AdminContent />;
}
```

### 5. API Calls with Auth Header

```tsx
const token = useToken();
fetch('https://api.example.com/data', {
  headers: { Authorization: token },
});
```

## API

| Method | Description |
|--------|-------------|
| `useGGIDAuth().init(config)` | Initialize SDK with GGID config |
| `useGGIDAuth().login()` | Start OAuth flow |
| `useGGIDAuth().logout()` | Clear session |
| `useGGIDAuth().restore()` | Restore session from secure storage |
| `useGGIDAuth().checkPermission(resource, action)` | RBAC permission check |
| `useUser()` | Get current user |
| `useIsAuthenticated()` | Check if logged in |
| `useToken()` | Get access token |

## Features

- OAuth Authorization Code + PKCE flow
- Secure token storage (expo-secure-store)
- Automatic token refresh
- RBAC permission checking via GGID policy engine
- Zustand state management
- TypeScript support

## Dependencies

- expo-auth-session
- expo-web-browser
- expo-secure-store
- zustand
