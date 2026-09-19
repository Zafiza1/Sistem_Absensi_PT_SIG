import type { NextConfig } from "next";

const nextConfig: NextConfig = {
  /* config options here */
  // Allow the dev server's assets/HMR to be fetched when the dashboard is
  // opened from another machine on the LAN via this host's IP, not just
  // localhost (Next.js blocks cross-origin dev requests by default).
  allowedDevOrigins: ["10.10.20.7"],
};

export default nextConfig;
