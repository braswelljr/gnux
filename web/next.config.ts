import { availableParallelism } from 'node:os';
import { type NextConfig } from 'next';
import { createMDX } from 'fumadocs-mdx/next';

/**
 * How many workers render the static pages. Next counts the host's CPUs, so in
 * a build container (Vercel's has 4 CPUs and 8 GB) it started 14 workers, which
 * ran out of memory and were killed. availableParallelism() counts the CPUs
 * this process may run on; four workers are as fast as more, for this site.
 *
 * `pnpm build` also uses webpack (`next build --webpack`), not Turbopack:
 * Turbopack compiles the MDX in a pool of loader processes that stay alive
 * while the pages are rendered, and in a 4-CPU, 8 GB container the build
 * peaked above 8 GB with it and at 4.4 GB with webpack. `next dev` keeps
 * Turbopack, where memory is not the limit.
 */
const buildWorkers = Math.max(1, Math.min(availableParallelism() - 1, 4));

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
  experimental: {
    cpus: buildWorkers
  },
  async headers() {
    return [
      { source: '/:path*', headers: securityHeaders },
      // The YAML reference is read by scripts and AI tools, and should show
      // as text in a browser rather than download.
      {
        source: '/reference/yaml/:path*',
        headers: [{ key: 'Content-Type', value: 'text/yaml; charset=utf-8' }]
      }
    ];
  },
  async redirects() {
    // Until 1.4.0 the docs of each minor release had a folder of their own
    // (/docs/v1.1/...). Now each older major version has one (/docs/v0/...),
    // and the latest docs are the newest major version's, so a link into a
    // v1.x folder leads to the same page of the latest docs.
    return [
      { source: '/docs/:version(v1\\.\\d+)', destination: '/docs', permanent: true },
      { source: '/docs/:version(v1\\.\\d+)/:path*', destination: '/docs/:path*', permanent: true }
    ];
  }
};

const withMDX = createMDX({});
export default withMDX(nextConfig);
