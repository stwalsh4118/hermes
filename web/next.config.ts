import type { NextConfig } from "next";

const nextConfig: NextConfig = {
  output: 'standalone', // Enables optimized Docker builds
  // Use separate build directories for dev vs production to avoid conflicts
  distDir: process.env.NODE_ENV === 'development' ? '.next-dev' : '.next',
};

export default nextConfig;