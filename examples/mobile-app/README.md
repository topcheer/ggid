# GGID Mobile Demo (Expo + React Native)

A mobile app demonstrating GGID OAuth 2.0 + OpenID Connect login flow.

## Features

- OAuth authorization code flow with PKCE
- User info display from `/api/v1/oauth/userinfo`
- JWT claims visualization
- Secure token storage via `expo-secure-store`
- Dark theme UI

## Prerequisites

- Node.js 18+
- Expo CLI: `npm install -g expo-cli`
- Expo Go app on your phone (or iOS Simulator / Android Emulator)

## Setup

```bash
cd examples/mobile-app
npm install
```

## Configuration

Environment variables are set via `app.json` or `.env`:

| Variable | Default | Description |
|----------|---------|-------------|
| `EXPO_PUBLIC_GGID_URL` | `https://ggid.iot2.win` | GGID server URL |
| `EXPO_PUBLIC_CLIENT_ID` | `gcid__sbYZX3_2aJ4eDz-Oy1qRQ` | OAuth client ID |
| `EXPO_PUBLIC_REDIRECT_URI` | `exp://localhost:8081/+redirect` | Redirect URI |

Create `.env` file to override:

```
EXPO_PUBLIC_GGID_URL=https://ggid.iot2.win
EXPO_PUBLIC_CLIENT_ID=gcid__sbYZX3_2aJ4eDz-Oy1qRQ
EXPO_PUBLIC_REDIRECT_URI=exp://localhost:8081/+redirect
```

## Run

```bash
npx expo start
```

Then scan the QR code with Expo Go (iOS/Android) or press:
- `i` — open in iOS Simulator
- `a` — open in Android Emulator
- `w` — open in web browser

## OAuth Flow

1. User taps "Sign In with GGID"
2. Browser opens to `https://ggid.iot2.win/oauth/authorize` with PKCE challenge
3. User logs in at GGID
4. Browser redirects back to app with authorization code
5. App exchanges code for access token at `/api/v1/oauth/token`
6. App fetches user info at `/api/v1/oauth/userinfo`
7. User profile and JWT claims are displayed

## File Structure

```
mobile-app/
├── App.tsx           # Main React component (login + profile screens)
├── app.json          # Expo configuration
├── package.json      # Dependencies
├── tsconfig.json     # TypeScript config
├── lib/
│   └── ggid.ts       # GGID OAuth client (authorize, token exchange, userinfo)
└── README.md         # This file
```

## GGID OAuth Endpoints Used

| Endpoint | Method | Description |
|----------|--------|-------------|
| `/oauth/authorize` | GET (browser) | Authorization code + PKCE |
| `/api/v1/oauth/token` | POST | Code → access_token exchange |
| `/api/v1/oauth/userinfo` | GET | Get user profile |

## License

Apache 2.0
