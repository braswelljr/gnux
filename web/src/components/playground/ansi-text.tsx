import { memo } from 'react';
import { parseAnsi } from 'lib/playground/ansi.ts';
import { cn } from 'lib/utils';

/**
 * The 16 ANSI colors. Each pair mirrors across the scale (700 with 300, 600
 * with 400), so a color keeps its distance from the background in both themes.
 */
const COLORS = [
  'text-neutral-950 dark:text-neutral-50',
  'text-red-700 dark:text-red-300',
  'text-green-700 dark:text-green-300',
  'text-yellow-700 dark:text-yellow-300',
  'text-blue-700 dark:text-blue-300',
  'text-fuchsia-700 dark:text-fuchsia-300',
  'text-cyan-700 dark:text-cyan-300',
  'text-neutral-700 dark:text-neutral-300',
  'text-neutral-500',
  'text-red-600 dark:text-red-400',
  'text-green-600 dark:text-green-400',
  'text-yellow-600 dark:text-yellow-400',
  'text-blue-600 dark:text-blue-400',
  'text-fuchsia-600 dark:text-fuchsia-400',
  'text-cyan-600 dark:text-cyan-400',
  'text-neutral-900 dark:text-neutral-100'
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
