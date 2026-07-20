'use client';
import { ERPUser, hasPermission } from '../lib/auth';
import { ERPLayout } from './ERPLayout';
import Forbidden403 from './Forbidden403';

export function PermissionGate({ user, perm, children }: { user: ERPUser; perm: string; children: React.ReactNode }) {
  if (!hasPermission(user, perm)) return <ERPLayout user={user}><Forbidden403 perm={perm} /></ERPLayout>;
  return <>{children}</>;
}