import { type ComponentProps } from 'react';
import { cn } from 'lib/utils';

/** The page's content column: one width and one gutter, everywhere. */
export function Container({ className, ...props }: ComponentProps<'div'>) {
  return (
    <div
      className={cn('mx-auto w-full max-w-6xl px-4 sm:px-6', className)}
      {...props}
    />
  );
}

/** A titled region of a page. Give it `aria-labelledby` pointing at its heading. */
export function Section({ className, children, ...props }: ComponentProps<'section'>) {
  return (
    <section
      className={cn('py-12 sm:py-16', className)}
      {...props}
    >
      <Container>{children}</Container>
    </section>
  );
}
