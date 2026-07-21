/**
 * GGID React SDK — LogoutButton component
 *
 * Pre-built logout button with optional redirect.
 *
 * Usage:
 * <LogoutButton />
 * <LogoutButton redirectAfterLogout="/goodbye" />
 * <LogoutButton label="Sign Out" variant="danger" />
 */

import React from 'react';
import { useGGIDAuth } from './useGGIDAuth';

export interface LogoutButtonProps {
  /** Label text (default: "Logout") */
  label?: string;
  /** Redirect path after logout (default: "/login") */
  redirectAfterLogout?: string;
  /** Visual variant */
  variant?: 'default' | 'danger' | 'ghost';
  /** Custom className */
  className?: string;
  /** Show icon */
  showIcon?: boolean;
  /** Disabled state */
  disabled?: boolean;
}

export function LogoutButton({
  label = 'Logout',
  redirectAfterLogout = '/login',
  variant = 'default',
  className = '',
  showIcon = true,
  disabled = false,
}: LogoutButtonProps) {
  const { logout } = useGGIDAuth();

  const handleClick = () => {
    logout();
    if (typeof window !== 'undefined') {
      window.location.href = redirectAfterLogout;
    }
  };

  const baseStyle: React.CSSProperties = {
    display: 'inline-flex',
    alignItems: 'center',
    gap: 6,
    padding: '8px 16px',
    borderRadius: 6,
    fontSize: 14,
    fontWeight: 500,
    cursor: disabled ? 'not-allowed' : 'pointer',
    opacity: disabled ? 0.5 : 1,
    border: 'none',
    transition: 'all 0.15s ease',
  };

  const variantStyles: Record<string, React.CSSProperties> = {
    default: {
      background: '#6366f1',
      color: '#fff',
    },
    danger: {
      background: '#ef4444',
      color: '#fff',
    },
    ghost: {
      background: 'transparent',
      color: '#6b7280',
      border: '1px solid #e5e7eb',
    },
  };

  const style = { ...baseStyle, ...variantStyles[variant] };

  return (
    <button
      type="button"
      onClick={handleClick}
      disabled={disabled}
      className={className}
      style={style}
    >
      {showIcon && (
        <svg
          width="16"
          height="16"
          viewBox="0 0 24 24"
          fill="none"
          stroke="currentColor"
          strokeWidth="2"
          strokeLinecap="round"
          strokeLinejoin="round"
        >
          <path d="M9 21H5a2 2 0 0 1-2-2V5a2 2 0 0 1 2-2h4" />
          <polyline points="16 17 21 12 16 7" />
          <line x1="21" y1="12" x2="9" y2="12" />
        </svg>
      )}
      {label}
    </button>
  );
}
