import { type ComponentProps } from 'react';
import { cn } from 'lib/utils';

/**
 * The cuts that draw the mark's figure, on a 32 by 32 grid. The tile is a
 * keycap seen face on: the line near its foot is where the face meets the
 * skirt. On the face sit a terminal prompt and a spanner: a shell, and the
 * tools it runs. It is original work, not derived from GNU Project artwork.
 */
export const LOGO_CUTS = {
  /** Stroked cuts: the keycap's skirt line, the prompt, the spanner's shaft. */
  strokes: [
    { d: 'M4 26.5H28', width: 1.4 },
    { d: 'M7.5 7.5 11.5 11 7.5 14.5', width: 2.4 },
    { d: 'M10.5 22 18.5 14', width: 2.8 }
  ],
  /** The spanner's head, cut as a disc... */
  head: { cx: 21, cy: 11.5, r: 4.4 },
  /** ...with its open jaw put back. */
  jaw: { d: 'M21 11.5 26.5 6', width: 3.4 }
} as const;

/**
 * The mark as a standalone SVG document in one flat color, for places that
 * take a file rather than a component: the Open Graph card.
 */
export function logoSvg(fill: string) {
  const { strokes, head, jaw } = LOGO_CUTS;
  const cuts = strokes.map((s) => `<path d="${s.d}" stroke-width="${s.width}"/>`).join('');
  return `<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 32 32"><mask id="m"><rect width="32" height="32" fill="#fff"/><g fill="none" stroke="#000" stroke-linecap="round" stroke-linejoin="round">${cuts}</g><circle cx="${head.cx}" cy="${head.cy}" r="${head.r}" fill="#000"/><path d="${jaw.d}" stroke="#fff" stroke-width="${jaw.width}"/></mask><rect width="32" height="32" rx="8" fill="${fill}" mask="url(#m)"/></svg>`;
}

type LogoProps = Omit<ComponentProps<'span'>, 'children'> & {
  /** The accessible name. Omit it where the written name sits beside the mark, and it is hidden. */
  title?: string;
};

/** The mark as a mask: opaque where the tile is, clear where it is cut. */
const MASK = `url("data:image/svg+xml,${encodeURIComponent(logoSvg('#000'))}")`;

/**
 * The gnux mark: one flat shape in `currentColor`, a keycap tile with a
 * prompt and a spanner cut out of it. It is black on light and white on dark
 * by following the text color, and reads from 16px up.
 *
 * It is painted through a CSS mask rather than drawn as inline SVG. An inline
 * SVG mask needs an id, and a layout that renders the same element twice (the
 * docs header and sidebar both do) would repeat it; the copy in the hidden
 * header then wins and the visible mark loses its cuts.
 */
export function Logo({ title, className, style, ...props }: LogoProps) {
  return (
    <span
      role={title ? 'img' : undefined}
      aria-hidden={title ? undefined : true}
      aria-label={title}
      className={cn('inline-block size-6 shrink-0 bg-current', className)}
      style={{
        maskImage: MASK,
        maskSize: '100% 100%',
        WebkitMaskImage: MASK,
        WebkitMaskSize: '100% 100%',
        ...style
      }}
      {...props}
    />
  );
}

/** The mark beside the written name, for headers and footers. */
export function LogoLockup({ className, ...props }: ComponentProps<'span'>) {
  return (
    <span
      className={cn('inline-flex items-center gap-2 text-fd-foreground', className)}
      {...props}
    >
      <Logo className="size-[1.4em]" />
      <span className="font-semibold tracking-tight">gnux</span>
    </span>
  );
}
