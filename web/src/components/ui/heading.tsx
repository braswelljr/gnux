import { type ComponentProps } from 'react';
import { cn } from 'lib/utils';

const sizes = {
  /** The one hero headline on the landing page. Steps of the scale, never a pixel value. */
  display: 'text-4xl font-bold tracking-tighter sm:text-5xl lg:text-6xl',
  '3xl': 'text-3xl font-bold tracking-tight',
  '2xl': 'text-2xl font-bold tracking-tight',
  xl: 'text-xl font-bold tracking-tight',
  lg: 'text-lg font-bold',
  base: 'text-base font-bold',
  sm: 'text-sm font-bold'
};

type HeadingProps = ComponentProps<'h1'> & {
  /** The heading's place in the page outline. */
  level: 1 | 2 | 3 | 4;
  /** How large it reads, chosen separately so the outline never follows the type size. */
  size?: keyof typeof sizes;
};

const defaults = { 1: '3xl', 2: '2xl', 3: 'lg', 4: 'base' } as const;

export function Heading({ level, size, className, ...props }: HeadingProps) {
  const Tag = `h${level}` as const;
  return (
    <Tag
      className={cn('text-balance', sizes[size ?? defaults[level]], className)}
      {...props}
    />
  );
}
