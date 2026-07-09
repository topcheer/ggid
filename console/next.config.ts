/** @type {import('next').NextConfig} */
const nextConfig = {
  output: 'standalone',
  async rewrites() {
    const gatewayUrl = process.env.GATEWAY_URL || 'http://gateway:8080';
    return [
      {
        source: '/api/:path*',
        destination: `${gatewayUrl}/api/:path*`,
      },
      {
        source: '/oauth/:path*',
        destination: `${gatewayUrl}/oauth/:path*`,
      },
    ];
  },
};

export default nextConfig;
