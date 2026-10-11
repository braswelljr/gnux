import { type ComponentProps } from 'react';
import { cn } from 'lib/utils';

const base =
  'inline-flex shrink-0 items-center justify-center gap-2 rounded-lg font-medium whitespace-nowrap outline-none focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-fd-ring disabled:pointer-events-none disabled:opacity-50 [&_svg]:pointer-events-none [&_svg]:shrink-0';

/**
 * Primary and secondary are keycaps that go down when pressed: primary wears
 * the accent colorway, as Return does on a board. Ghost is a bare legend, for
 * small controls inside another surface.
 */
const variants = {
  primary: 'keycap keycap-accent hover:brightness-105',
  secondary: 'keycap hover:brightness-[1.03] dark:hover:brightness-110',
  ghost:
    'text-fd-muted-foreground transition-colors duration-150 hover:bg-fd-accent hover:text-fd-accent-foreground motion-reduce:transition-none'
};

/**
 * The icon sizes set the box and the glyph together, so an icon-only button is
 * sized with `size="icon"` or `"icon-sm"`, never squared off with a className.
 */
const sizes = {
  sm: 'h-8 px-3 text-sm [&_svg]:size-4',
  md: 'h-10 px-4 text-sm [&_svg]:size-4',
  lg: 'h-11 px-5 text-base [&_svg]:size-5',
  'icon-sm': 'size-8 [&_svg]:size-4',
  icon: 'size-10 [&_svg]:size-5'
};

export type ButtonVariant = keyof typeof variants;
export type ButtonSize = keyof typeof sizes;

type VariantProps = { variant?: ButtonVariant; size?: ButtonSize; className?: string };

/** The classes of a button, for an element that is not one: a link that looks like an action. */
export function buttonVariants({ variant = 'secondary', size = 'md', className }: VariantProps) {
  return cn(base, variants[variant], sizes[size], className);
}

export type ButtonProps = ComponentProps<'button'> & VariantProps;

/**
 * An action. An icon-only button needs an `aria-label`, and its icon
 * `aria-hidden`, because the glyph is not a name.
 */
export function Button({ variant, size, className, type = 'button', ...props }: ButtonProps) {
  return (
    <button
      type={type}
      className={buttonVariants({ variant, size, className })}
      {...props}
    />
  );
}
