/** @type {import('next').NextConfig} */
const nextConfig = {
  output: 'standalone',
  // Proxy API requests to the Gateway in development
  async rewrites() {
    return [
      {
        source: '/api/:path*',
        destination: `${process.env.GATEWAY_URL || 'http://localhost:8080'}/api/:path*`,
      },
    ]
  },
}

export default nextConfig
