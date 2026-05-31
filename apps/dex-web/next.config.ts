import type { NextConfig } from "next";
import path from "path";

const nextConfig: NextConfig = {
  reactStrictMode: true,
  webpack: (config, { dir }) => {
    config.resolve.alias = {
      ...config.resolve.alias,
      "@react-native-async-storage/async-storage": path.join(
        dir,
        "lib/stub-async-storage.ts"
      ),
      "pino-pretty": path.join(dir, "lib/stub-pino-pretty.ts"),
    };
    return config;
  },
};

export default nextConfig;
