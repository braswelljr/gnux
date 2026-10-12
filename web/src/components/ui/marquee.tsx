import { type ComponentProps, type ReactNode } from 'react';
import { cn } from 'lib/utils';

type MarqueeProps = ComponentProps<'div'> & {
  /** The items. Wrap a group of them in a list for assistive technology. */
  children: ReactNode;
  /** Slide the other way. */
  reverse?: boolean;
  /** Hold still while the pointer is over it or focus is in it. */
  pauseOnHover?: boolean;
  /** Slide up instead of across. */
  vertical?: boolean;
  /** How many copies of the items fill the track; more for short content. */
  repeat?: number;
};

/**
 * Items sliding past in a loop: the children repeated `repeat` times, each
 * copy moving by its own width plus the gap. Set `--duration` and `--gap` in
 * `style` to tune it. Only the first copy is exposed: the repeats are hidden
 * from assistive technology and unreachable by keyboard, so every item is read
 * once. Under reduced motion it stands still, shows one copy and scrolls by
 * hand. Stop it from outside with
 * `data-paused="true"` on an ancestor.
 */
export function Marquee({
  className,
  reverse = false,
  pauseOnHover = false,
  children,
  vertical = false,
  repeat = 4,
  ...props
}: MarqueeProps) {
  return (
    <div
      {...props}
      className={cn(
        'group flex gap-(--gap) overflow-hidden p-2 [--duration:40s] [--gap:1rem] motion-reduce:overflow-auto',
        vertical ? 'flex-col' : 'flex-row',
        className
      )}
    >
      {Array.from({ length: Math.max(2, repeat) }, (_, i) => (
        <div
          key={i}
          aria-hidden={i > 0 || undefined}
          inert={i > 0}
          className={cn(
            'flex shrink-0 justify-around gap-(--gap)',
            // Standing still, the repeats would only show the items again.
            i > 0 && 'motion-reduce:hidden',
            vertical ? 'animate-marquee-vertical flex-col' : 'animate-marquee flex-row',
            pauseOnHover &&
              'group-focus-within:[animation-play-state:paused] group-hover:[animation-play-state:paused]',
            reverse && '[animation-direction:reverse]'
          )}
        >
          {children}
        </div>
      ))}
    </div>
  );
}
