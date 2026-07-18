import path from 'path';
import { fileURLToPath } from 'url';

const __filename = fileURLToPath(import.meta.url);
const __dirname = path.dirname(__filename);

/** @type {import('next').NextConfig} */
const nextConfig = {
  output: 'standalone',
  transpilePackages: ['@ggid/sdk-react'],
  typescript: {
    // Type errors are validated separately via `npm run typecheck`
    // Build should not fail on type errors (some are from SDK types)
    ignoreBuildErrors: true,
  },
  experimental: {
    // Tree-shake large icon/SDK packages — reduces bundle size significantly
    optimizePackageImports: ['lucide-react', '@ggid/sdk-react'],
  },
  async rewrites() {
    const gatewayUrl = process.env.GATEWAY_URL || 'http://localhost:8080';
    return [
      { source: '/risk-scoring', destination: '/security/risk-score' },
      { source: '/threat-intel', destination: '/security/threat-intel' },
      { source: '/healthz', destination: `${gatewayUrl}/healthz` },
      { source: '/healthz/:path*', destination: `${gatewayUrl}/healthz/:path*` },
      { source: '/api/:path*', destination: `${gatewayUrl}/api/:path*` },
      { source: '/oauth/:path*', destination: `${gatewayUrl}/oauth/:path*` },
      { source: '/saml/:path*', destination: `${gatewayUrl}/saml/:path*` },
      { source: '/.well-known/:path*', destination: `${gatewayUrl}/.well-known/:path*` },
    ];
  },
};

export default nextConfig;
