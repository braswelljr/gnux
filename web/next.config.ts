import { type NextConfig } from 'next';
import { createMDX } from 'fumadocs-mdx/next';

/**
 * Baseline response headers for every route.
 *
 * There is no Content-Security-Policy here on purpose: the theme script is
 * inline and the playground compiles WebAssembly, so a useful policy needs a
 * per-request nonce, which would turn every static page dynamic.
 */
const securityHeaders = [
  { key: 'X-Content-Type-Options', value: 'nosniff' },
  { key: 'X-Frame-Options', value: 'SAMEORIGIN' },
  { key: 'Referrer-Policy', value: 'strict-origin-when-cross-origin' },
  { key: 'Permissions-Policy', value: 'camera=(), microphone=(), geolocation=()' }
];

const nextConfig: NextConfig = {
  reactStrictMode: true,
  // `next dev` would otherwise append its own block to AGENTS.md on every run;
  // that file is written by hand here, and the block breaks the house rules.
  agentRules: false,
  async headers() {
    return [{ source: '/:path*', headers: securityHeaders }];
  }
};

const withMDX = createMDX({});
export default withMDX(nextConfig);
