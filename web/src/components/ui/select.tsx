import { type ComponentProps } from 'react';
import { ChevronsUpDown } from 'lucide-react';
import { cn } from 'lib/utils';

/**
 * A native select, styled as the site's controls are: the browser's own
 * list, keyboard handling and accessibility, with a chevron of ours.
 */
export function Select({ className, children, ...props }: ComponentProps<'select'>) {
  return (
    <span className={cn('relative inline-flex w-full items-center', className)}>
      <select
        className="w-full appearance-none rounded-lg border border-fd-border bg-fd-secondary py-1.5 pr-8 pl-3 text-sm text-fd-secondary-foreground transition-colors hover:bg-fd-accent focus-visible:ring-2 focus-visible:ring-fd-ring focus-visible:outline-none"
        {...props}
      >
        {children}
      </select>
      <ChevronsUpDown
        aria-hidden="true"
        className="pointer-events-none absolute right-2.5 size-4 text-fd-muted-foreground"
      />
    </span>
  );
}
