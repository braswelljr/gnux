'use client';

import { useRef } from 'react';
import { cn } from 'lib/utils';
import { useInView } from '@/hooks/use-in-view';

type AnimatedBackgroundProps = {
  /** `hero` is the full aurora; `subtle` is a quieter wash for inner pages and sections. */
  variant?: 'hero' | 'subtle';
  className?: string;
};

/** One keycap outline on a 48px pitch: the tile the backdrop's grid repeats. */
const KEY_TILE = `url("data:image/svg+xml,${encodeURIComponent(
  '<svg xmlns="http://www.w3.org/2000/svg" width="48" height="48"><rect x="3.5" y="3.5" width="41" height="41" rx="8" fill="none" stroke="black" stroke-width="1"/></svg>'
)}")`;

/**
 * The ambient layer behind a section: slow aurora glows over a faint grid of
 * keycap outlines that fades toward the edges, as if the page were a plate
 * with the keys lifted off. Fill the positioned parent with it. It is
 * decorative, pauses off screen, and holds still under reduced motion.
 */
export function AnimatedBackground({ variant = 'hero', className }: AnimatedBackgroundProps) {
  const ref = useRef<HTMLDivElement>(null);
  const inView = useInView(ref);
  const hero = variant === 'hero';

  return (
    <div
      ref={ref}
      aria-hidden="true"
      data-paused={!inView}
      className={cn('pointer-events-none absolute inset-0 -z-10 overflow-hidden', className)}
    >
      <div
        className={cn(
          'absolute -top-1/3 left-1/2 aspect-square w-[60rem] max-w-[160%] -translate-x-1/2',
          hero ? 'opacity-45 dark:opacity-35' : 'opacity-25 dark:opacity-20'
        )}
      >
        <div className="absolute top-[4%] left-[2%] size-3/5 animate-aurora aurora-glow-cyan opacity-80" />
        <div className="absolute top-[14%] right-[0%] size-3/5 animate-aurora aurora-glow-violet [animation-delay:-7s]" />
        {hero && (
          <div className="absolute top-[34%] left-[26%] size-1/2 animate-aurora aurora-glow-amber opacity-70 [animation-delay:-14s]" />
        )}
      </div>
      <div
        className={cn(
          'absolute inset-0 bg-fd-foreground',
          hero ? 'opacity-[0.07] dark:opacity-[0.09]' : 'opacity-[0.05]'
        )}
        style={{
          maskImage: `${KEY_TILE}, radial-gradient(ellipse 70% 60% at 50% 30%, black, transparent)`,
          maskComposite: 'intersect',
          WebkitMaskImage: `${KEY_TILE}, radial-gradient(ellipse 70% 60% at 50% 30%, black, transparent)`,
          WebkitMaskComposite: 'source-in'
        }}
      />
    </div>
  );
}
