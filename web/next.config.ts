import type { NextConfig } from "next";

const nextConfig: NextConfig = {
  // Separate build output to avoid conflicts with dev server
  distDir: "build",
};

export default nextConfig;
