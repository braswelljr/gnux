import { type ComponentProps } from 'react';
import { cn } from 'lib/utils';

type MonitorProps = ComponentProps<'div'> & {
  /** Draw the neck and foot under the screen. Leave it off where there is no desk to stand on. */
  stand?: boolean;
  /** Classes for the stand, such as hiding it on small screens. */
  standClassName?: string;
  /** Classes for the screen, where the children are shown on the wallpaper. */
  screenClassName?: string;
};

/**
 * A desktop monitor: a dark bezel around a screen with a wallpaper and a soft
 * glare, a chin with its power light, and optionally a stand. Whatever is put
 * inside is shown on the screen, like a window on a desktop. The hardware is
 * decorative, so only the children reach assistive technology.
 */
export function Monitor({
  stand = true,
  className,
  screenClassName,
  standClassName,
  children,
  ...props
}: MonitorProps) {
  return (
    <div
      className={cn('mx-auto w-full', className)}
      {...props}
    >
      <div className="rounded-[1.25rem] p-1.5 monitor-bezel sm:rounded-[1.6rem] sm:p-2.5">
        <div
          className={cn(
            'relative isolate overflow-hidden rounded-[0.8rem] p-3 monitor-screen sm:rounded-[1rem] sm:p-6',
            screenClassName
          )}
        >
          {children}
          {/* Light falling on the glass, over everything on the screen. */}
          <span
            aria-hidden="true"
            className="pointer-events-none absolute inset-0 z-20 bg-[linear-gradient(115deg,rgb(255_255_255/0.07)_0%,rgb(255_255_255/0.02)_35%,transparent_36%)]"
          />
        </div>
        <div
          aria-hidden="true"
          className="flex h-5 items-center justify-center sm:h-7"
        >
          <span className="size-1.5 rounded-full bg-lit-face shadow-[0_0_6px_var(--cap-accent-face)]" />
        </div>
      </div>
      {stand && (
        <div
          aria-hidden="true"
          className={cn('flex flex-col items-center', standClassName)}
        >
          <div className="h-10 w-28 [clip-path:polygon(18%_0,82%_0,100%_100%,0_100%)] monitor-stand sm:h-14 sm:w-36" />
          <div className="h-2.5 w-64 rounded-t-md rounded-b-sm monitor-stand sm:w-80" />
        </div>
      )}
    </div>
  );
}
