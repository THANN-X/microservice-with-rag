/** @type {import('next').NextConfig} */
const nextConfig = {
  images: {
    remotePatterns: [
      {
        protocol: "https",
        hostname: "**",
      },
    ],
  },
  async rewrites() {
    // All API calls route through BFF gateway at :8080
    // BFF handles auth, CORS, and proxying to individual services
    const BFF = process.env.BFF_URL || "http://localhost:8080";
    return [
      { source: "/api/auth/:path*", destination: `${BFF}/api/auth/:path*` },
      { source: "/api/products/:path*", destination: `${BFF}/api/products/:path*` },
      { source: "/api/categories/:path*", destination: `${BFF}/api/categories/:path*` },
      { source: "/api/attributes/:path*", destination: `${BFF}/api/attributes/:path*` },
      { source: "/api/cart/:path*", destination: `${BFF}/api/cart/:path*` },
      { source: "/api/orders/:path*", destination: `${BFF}/api/orders/:path*` },
      { source: "/api/catalog/:path*", destination: `${BFF}/api/catalog/:path*` },
      { source: "/api/order-history/:path*", destination: `${BFF}/api/order-history/:path*` },
      { source: "/chat", destination: `${BFF}/chat` },
    ];
  },
};

export default nextConfig;
