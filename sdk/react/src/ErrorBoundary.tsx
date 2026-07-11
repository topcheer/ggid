/**
 * GGID React SDK — ErrorBoundary
 *
 * Catches authentication-related errors and provides a fallback UI.
 *
 * Usage:
 *   <ErrorBoundary fallback={<LoginScreen />}>
 *     <ProtectedRoute><App /></ProtectedRoute>
 *   </ErrorBoundary>
 */

import { Component, type ReactNode } from 'react';

interface ErrorBoundaryProps {
  children: ReactNode;
  fallback?: ReactNode;
  onError?: (error: Error, errorInfo: React.ErrorInfo) => void;
  /** Called when user clicks "retry" — typically logout + redirect */
  onRetry?: () => void;
}

interface ErrorBoundaryState {
  hasError: boolean;
  error: Error | null;
}

export class ErrorBoundary extends Component<ErrorBoundaryProps, ErrorBoundaryState> {
  constructor(props: ErrorBoundaryProps) {
    super(props);
    this.state = { hasError: false, error: null };
  }

  static getDerivedStateFromError(error: Error): ErrorBoundaryState {
    return { hasError: true, error };
  }

  componentDidCatch(error: Error, errorInfo: React.ErrorInfo) {
    // Call optional onError callback for logging
    this.props.onError?.(error, errorInfo);

    // Detect auth errors and log them
    const isAuthError =
      error.message.includes('401') ||
      error.message.includes('403') ||
      error.message.includes('token') ||
      error.message.includes('unauthorized') ||
      error.message.includes('session expired');

    if (isAuthError) {
      console.warn('[GGID] Auth error caught:', error.message);
    }
  }

  handleRetry = () => {
    this.setState({ hasError: false, error: null });
    this.props.onRetry?.();
  };

  render() {
    if (this.state.hasError) {
      if (this.props.fallback) {
        return this.props.fallback;
      }

      return (
        <div
          style={{
            display: 'flex',
            flexDirection: 'column',
            alignItems: 'center',
            justifyContent: 'center',
            minHeight: '100vh',
            padding: '2rem',
            fontFamily: 'system-ui, sans-serif',
          }}
        >
          <div
            style={{
              maxWidth: 400,
              textAlign: 'center',
            }}
          >
            <h2 style={{ fontSize: '1.25rem', fontWeight: 600, marginBottom: '0.5rem' }}>
              Something went wrong
            </h2>
            <p style={{ fontSize: '0.875rem', color: '#6b7280', marginBottom: '1.5rem' }}>
              {this.state.error?.message || 'An unexpected error occurred.'}
            </p>
            <button
              onClick={this.handleRetry}
              style={{
                padding: '0.5rem 1.5rem',
                borderRadius: '0.5rem',
                backgroundColor: '#4f46e5',
                color: 'white',
                fontSize: '0.875rem',
                fontWeight: 500,
                border: 'none',
                cursor: 'pointer',
              }}
            >
              Try Again
            </button>
          </div>
        </div>
      );
    }

    return this.props.children;
  }
}
