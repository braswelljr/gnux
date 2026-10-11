import { type ComponentProps } from 'react';
import { cn } from 'lib/utils';

const variants = {
  plain: 'border bg-fd-card',
  /** Recessed, like the plate keys sit in: for a group of things to read or copy. */
  plate: 'plate',
  /** A keycap that goes down when pressed: for a card that is a link. */
  key: 'keycap hover:brightness-[1.03] dark:hover:brightness-110'
};

type CardProps = ComponentProps<'article'> & { variant?: keyof typeof variants };

/** A bordered surface. Compose it: `Card > CardHeader + CardContent`. */
export function Card({ className, variant = 'plain', ...props }: CardProps) {
  return (
    <article
      className={cn(
        'flex min-w-0 flex-col rounded-xl text-fd-card-foreground',
        variants[variant],
        className
      )}
      {...props}
    />
  );
}

export function CardHeader({ className, ...props }: ComponentProps<'header'>) {
  return (
    <header
      className={cn('flex flex-col gap-1.5 p-5 pb-3', className)}
      {...props}
    />
  );
}

export function CardDescription({ className, ...props }: ComponentProps<'p'>) {
  return (
    <p
      className={cn('text-sm text-pretty text-fd-muted-foreground', className)}
      {...props}
    />
  );
}

export function CardContent({ className, ...props }: ComponentProps<'div'>) {
  return (
    <div
      className={cn('flex min-w-0 flex-1 flex-col gap-3 p-5 pt-0', className)}
      {...props}
    />
  );
}
