import React, { useState, useEffect, useCallback } from 'react';
import {
  StyleSheet, Text, View, TouchableOpacity, ActivityIndicator,
  ScrollView, SafeAreaView, Alert,
} from 'react-native';
import { StatusBar } from 'expo-status-bar';
import * as WebBrowser from 'expo-web-browser';
import {
  login, logout, getStoredToken, getStoredUserInfo, parseJWT,
  clearSession, GGID_CONFIG,
  type UserInfo, type JWTCclaims,
} from './lib/ggid';

// Ensure WebBrowser completes when redirect returns
WebBrowser.maybeCompleteAuthSession();

export default function App() {
  const [loading, setLoading] = useState(true);
  const [loggingIn, setLoggingIn] = useState(false);
  const [userInfo, setUserInfo] = useState<UserInfo | null>(null);
  const [claims, setClaims] = useState<JWTCclaims | null>(null);
  const [token, setToken] = useState<string | null>(null);

  // Check for existing session on app start
  useEffect(() => {
    (async () => {
      const storedToken = await getStoredToken();
      const storedUser = await getStoredUserInfo();
      if (storedToken && storedUser) {
        setToken(storedToken);
        setUserInfo(storedUser);
        setClaims(parseJWT(storedToken));
      }
      setLoading(false);
    })();
  }, []);

  const handleLogin = useCallback(async () => {
    setLoggingIn(true);
    try {
      const { token: tok, userInfo: user } = await login();
      setToken(tok.access_token);
      setUserInfo(user);
      setClaims(parseJWT(tok.access_token));
    } catch (err: any) {
      Alert.alert('Login Failed', err?.message || 'Unknown error');
    } finally {
      setLoggingIn(false);
    }
  }, []);

  const handleLogout = useCallback(async () => {
    await logout();
    setToken(null);
    setUserInfo(null);
    setClaims(null);
  }, []);

  // ─── Loading Screen ─────────────────────────────────────
  if (loading) {
    return (
      <View style={styles.center}>
        <ActivityIndicator size="large" color="#4f46e5" />
      </View>
    );
  }

  // ─── Logged Out: Login Screen ──────────────────────────
  if (!token || !userInfo) {
    return (
      <SafeAreaView style={styles.container}>
        <StatusBar style="light" />
        <View style={styles.loginContainer}>
          <View style={styles.logoCircle}>
            <Text style={styles.logoText}>GGID</Text>
          </View>
          <Text style={styles.title}>GGID Mobile Demo</Text>
          <Text style={styles.subtitle}>OAuth 2.0 + OpenID Connect</Text>
          <Text style={styles.description}>
            Sign in with your GGID account to view your profile and JWT claims.
          </Text>
          <TouchableOpacity
            style={styles.loginButton}
            onPress={handleLogin}
            disabled={loggingIn}
          >
            {loggingIn ? (
              <ActivityIndicator color="#fff" />
            ) : (
              <Text style={styles.loginButtonText}>Sign In with GGID</Text>
            )}
          </TouchableOpacity>
          <Text style={styles.hint}>
            Server: {GGID_CONFIG.url}{'\n'}
            Client: {GGID_CONFIG.clientId}
          </Text>
        </View>
      </SafeAreaView>
    );
  }

  // ─── Logged In: User Profile Screen ─────────────────────
  return (
    <SafeAreaView style={styles.container}>
      <StatusBar style="light" />
      <ScrollView style={styles.scrollView}>
        {/* Header */}
        <View style={styles.header}>
          <View style={styles.avatar}>
            <Text style={styles.avatarText}>
              {(userInfo.name || userInfo.preferred_username || '?')[0]?.toUpperCase()}
            </Text>
          </View>
          <Text style={styles.userName}>{userInfo.name || 'Unknown User'}</Text>
          <Text style={styles.userEmail}>{userInfo.email || 'No email'}</Text>
          {userInfo.email_verified && (
            <View style={styles.verifiedBadge}>
              <Text style={styles.verifiedText}>✓ Verified</Text>
            </View>
          )}
        </View>

        {/* User Info Card */}
        <View style={styles.card}>
          <Text style={styles.cardTitle}>User Information</Text>
          <InfoRow label="Subject ID" value={userInfo.sub} />
          <InfoRow label="Username" value={userInfo.preferred_username || '-'} />
          <InfoRow label="Email" value={userInfo.email || '-'} />
          <InfoRow label="Tenant ID" value={userInfo.tenant_id || GGID_CONFIG.tenantId} />
          <InfoRow label="Name" value={userInfo.name || '-'} />
        </View>

        {/* JWT Claims Card */}
        {claims && (
          <View style={styles.card}>
            <Text style={styles.cardTitle}>JWT Claims</Text>
            {Object.entries(claims).map(([key, value]) => (
              <InfoRow
                key={key}
                label={key}
                value={typeof value === 'object' ? JSON.stringify(value) : String(value)}
              />
            ))}
          </View>
        )}

        {/* Token Preview */}
        <View style={styles.card}>
          <Text style={styles.cardTitle}>Access Token (truncated)</Text>
          <Text style={styles.tokenText}>
            {token.substring(0, 50)}...{token.substring(token.length - 20)}
          </Text>
        </View>

        {/* Logout Button */}
        <TouchableOpacity style={styles.logoutButton} onPress={handleLogout}>
          <Text style={styles.logoutButtonText}>Sign Out</Text>
        </TouchableOpacity>

        <Text style={styles.footer}>GGID Mobile Demo v1.0.0</Text>
      </ScrollView>
    </SafeAreaView>
  );
}

// ─── Helper Component ───────────────────────────────────────
function InfoRow({ label, value }: { label: string; value: string }) {
  return (
    <View style={styles.infoRow}>
      <Text style={styles.infoLabel}>{label}</Text>
      <Text style={styles.infoValue} numberOfLines={2}>{value}</Text>
    </View>
  );
}

// ─── Styles ─────────────────────────────────────────────────
const styles = StyleSheet.create({
  container: {
    flex: 1,
    backgroundColor: '#0f172a',
  },
  center: {
    flex: 1,
    justifyContent: 'center',
    alignItems: 'center',
    backgroundColor: '#0f172a',
  },
  loginContainer: {
    flex: 1,
    justifyContent: 'center',
    alignItems: 'center',
    paddingHorizontal: 32,
  },
  logoCircle: {
    width: 80,
    height: 80,
    borderRadius: 40,
    backgroundColor: '#4f46e5',
    justifyContent: 'center',
    alignItems: 'center',
    marginBottom: 24,
  },
  logoText: {
    color: '#fff',
    fontSize: 22,
    fontWeight: 'bold',
  },
  title: {
    fontSize: 28,
    fontWeight: 'bold',
    color: '#f1f5f9',
    marginBottom: 8,
  },
  subtitle: {
    fontSize: 16,
    color: '#818cf8',
    marginBottom: 24,
  },
  description: {
    fontSize: 14,
    color: '#94a3b8',
    textAlign: 'center',
    marginBottom: 32,
    lineHeight: 20,
  },
  loginButton: {
    backgroundColor: '#4f46e5',
    paddingHorizontal: 32,
    paddingVertical: 16,
    borderRadius: 12,
    width: '100%',
    alignItems: 'center',
  },
  loginButtonText: {
    color: '#fff',
    fontSize: 16,
    fontWeight: '600',
  },
  hint: {
    fontSize: 12,
    color: '#64748b',
    marginTop: 24,
    textAlign: 'center',
    lineHeight: 18,
  },
  scrollView: {
    flex: 1,
    padding: 16,
  },
  header: {
    alignItems: 'center',
    paddingVertical: 24,
  },
  avatar: {
    width: 72,
    height: 72,
    borderRadius: 36,
    backgroundColor: '#4f46e5',
    justifyContent: 'center',
    alignItems: 'center',
    marginBottom: 12,
  },
  avatarText: {
    color: '#fff',
    fontSize: 28,
    fontWeight: 'bold',
  },
  userName: {
    fontSize: 22,
    fontWeight: 'bold',
    color: '#f1f5f9',
    marginBottom: 4,
  },
  userEmail: {
    fontSize: 14,
    color: '#94a3b8',
    marginBottom: 8,
  },
  verifiedBadge: {
    backgroundColor: 'rgba(34, 197, 94, 0.15)',
    paddingHorizontal: 10,
    paddingVertical: 4,
    borderRadius: 12,
  },
  verifiedText: {
    color: '#22c55e',
    fontSize: 12,
    fontWeight: '600',
  },
  card: {
    backgroundColor: '#1e293b',
    borderRadius: 12,
    padding: 16,
    marginBottom: 16,
  },
  cardTitle: {
    fontSize: 14,
    fontWeight: '700',
    color: '#818cf8',
    marginBottom: 12,
    textTransform: 'uppercase',
  },
  infoRow: {
    flexDirection: 'row',
    justifyContent: 'space-between',
    paddingVertical: 6,
    borderBottomWidth: 1,
    borderBottomColor: '#334155',
  },
  infoLabel: {
    fontSize: 13,
    color: '#94a3b8',
    flex: 0.4,
  },
  infoValue: {
    fontSize: 13,
    color: '#e2e8f0',
    flex: 0.6,
    textAlign: 'right',
    fontFamily: 'monospace',
  },
  tokenText: {
    fontSize: 11,
    color: '#64748b',
    fontFamily: 'monospace',
    lineHeight: 16,
  },
  logoutButton: {
    backgroundColor: '#dc2626',
    paddingHorizontal: 24,
    paddingVertical: 14,
    borderRadius: 12,
    alignItems: 'center',
    marginBottom: 16,
  },
  logoutButtonText: {
    color: '#fff',
    fontSize: 16,
    fontWeight: '600',
  },
  footer: {
    fontSize: 12,
    color: '#475569',
    textAlign: 'center',
    paddingBottom: 24,
  },
});
