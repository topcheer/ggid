// GGID React SDK Quickstart — 5 minute integration
//
// Prerequisites:
//   1. GGID running on localhost:8080
//   2. npm install @ggid/react
//   3. Wrap your app in <GGIDProvider>
//
import { GGIDProvider, useGGIDAuth, LogoutButton } from "@ggid/react";

// --- 1. Wrap your app ---
export default function App() {
  return (
    <GGIDProvider baseUrl="http://localhost:8080">
      <Dashboard />
    </GGIDProvider>
  );
}

// --- 2. Use auth + API in any component ---
function Dashboard() {
  const { user, login, isLoading } = useGGIDAuth();

  // Show login form if not authenticated
  if (!user) {
    return (
      <form onSubmit={(e) => {
        e.preventDefault();
        const f = new FormData(e.currentTarget);
        login(f.get("email") as string, f.get("password") as string);
      }}>
        <input name="email" placeholder="admin@ggid.dev" defaultValue="admin@ggid.dev" />
        <input name="password" type="password" defaultValue="Admin@123456" />
        <button type="submit" disabled={isLoading}>Login</button>
      </form>
    );
  }

  // Authenticated — show user info + logout
  return (
    <div>
      <h1>Welcome, {user.display_name}!</h1>
      <p>Email: {user.email}</p>
      <LogoutButton />
    </div>
  );
}
