import { memo } from 'react';
import { parseAnsi } from 'lib/playground/ansi.ts';
import { cn } from 'lib/utils';

/**
 * The 16 ANSI colors, from the terminal's palette (`.term-palette` in
 * main.css): the page's colors, or those that suit a theme's own background.
 */
const COLORS = [
  'text-(--ansi-0)',
  'text-(--ansi-1)',
  'text-(--ansi-2)',
  'text-(--ansi-3)',
  'text-(--ansi-4)',
  'text-(--ansi-5)',
  'text-(--ansi-6)',
  'text-(--ansi-7)',
  'text-(--ansi-8)',
  'text-(--ansi-9)',
  'text-(--ansi-10)',
  'text-(--ansi-11)',
  'text-(--ansi-12)',
  'text-(--ansi-13)',
  'text-(--ansi-14)',
  'text-(--ansi-15)'
];

/** Renders tool output, turning ANSI color escapes into styled spans. */
export const AnsiText = memo(function AnsiText({ text }: { text: string }) {
  // Most output has no escapes at all; skip the parser for it.
  if (!text.includes('\x1b')) return <>{text}</>;

  return (
    <>
      {parseAnsi(text).map((span, i) => (
        <span
          key={i}
          // A theme's exact color comes as data, not as a class.
          style={span.color === undefined ? undefined : { color: span.color }}
          className={cn(
            span.fg !== undefined && COLORS[span.fg],
            span.bold && 'font-bold',
            span.dim && 'opacity-70',
            span.underline && 'underline'
          )}
        >
          {span.text}
        </span>
      ))}
    </>
  );
});
