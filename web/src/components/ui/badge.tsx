import { type ComponentProps } from 'react';
import { cn } from 'lib/utils';

/** A small modifier key of text: an eyebrow above a heading, a tag beside a name. */
export function Badge({ className, ...props }: ComponentProps<'span'>) {
  return (
    <span
      className={cn(
        'keycap inline-flex items-center gap-2 rounded-lg px-3 py-1 font-mono text-xs keycap-mod [--cap-depth:3px]',
        className
      )}
      {...props}
    />
  );
}
