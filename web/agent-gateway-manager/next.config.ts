import path from "path";
import type { NextConfig } from "next";

const nextConfig: NextConfig = {
  webpack(config, { dir }) {
    config.resolve.modules = [path.join(dir, "node_modules"), ...config.resolve.modules];
    return config;
  },
};

export default nextConfig;
