import path from 'path';
import { fileURLToPath } from 'url';

const __filename = fileURLToPath(import.meta.url);
const __dirname = path.dirname(__filename);

/** @type {import('next').NextConfig} */
const nextConfig = {
  output: 'standalone',
  transpilePackages: ['@ggid/sdk-react'],
  async rewrites() {
    const gatewayUrl = process.env.GATEWAY_URL || 'http://localhost:8080';
    return [
      { source: '/api/:path*', destination: `${gatewayUrl}/api/:path*` },
      { source: '/oauth/:path*', destination: `${gatewayUrl}/oauth/:path*` },
      { source: '/saml/:path*', destination: `${gatewayUrl}/saml/:path*` },
      { source: '/.well-known/:path*', destination: `${gatewayUrl}/.well-known/:path*` },
    ];
  },
};

export default nextConfig;
