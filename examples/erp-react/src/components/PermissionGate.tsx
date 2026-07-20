'use client';
import { useState } from 'react';
import { ERPUser, hasPermission, ERPLayout } from '@/lib/auth';
import { Forbidden403 } from '@/components/Forbidden403';

export function PermissionGate({ user, perm, children }: { user: ERPUser; perm: string; children: React.ReactNode }) {
  if (!hasPermission(user, perm)) return <ERPLayout user={user}><Forbidden403 perm={perm} /></ERPLayout>;
  return <>{children}</>;
}
