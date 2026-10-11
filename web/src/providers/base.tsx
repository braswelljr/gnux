'use client';

import { RootProvider } from 'fumadocs-ui/provider/next';
import { MotionConfig } from 'motion/react';
import { NuqsAdapter } from 'nuqs/adapters/next/app';

/**
 * Client providers for the whole site: Fumadocs' theme and search, URL state,
 * and a motion default that honors the reader's reduced-motion setting.
 */
export default function Base({ children }: { children?: React.ReactNode }) {
  return (
    <RootProvider>
      <NuqsAdapter>
        <MotionConfig reducedMotion="user">{children}</MotionConfig>
      </NuqsAdapter>
    </RootProvider>
  );
}
