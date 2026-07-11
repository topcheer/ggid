/**
 * GGID React SDK — useGGIDAuth hook
 */

import { useContext } from 'react';
import { GGIDAuthContext } from './GGIDProvider';
import type { GGIDAuthContextValue } from './types';

export function useGGIDAuth(): GGIDAuthContextValue {
  const ctx = useContext(GGIDAuthContext);
  if (!ctx) {
    throw new Error('useGGIDAuth must be used within a <GGIDProvider> component');
  }
  return ctx;
}
