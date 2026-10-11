import { cn } from 'lib/utils';
import { CopyButton } from '@/components/ui/copy-button';

type CodeBlockProps = {
  code: string;
  /** Shown above the code, and the block's accessible name: "main.go", "Shell". */
  title: string;
  /** What the copy button copies, when it should differ from what is shown. */
  copyValue?: string;
  className?: string;
};

/**
 * A short code sample with its title and a copy button. Samples on the landing
 * page are a few lines of shell or Go, so they are set plainly rather than
 * sent through the syntax highlighter the documentation uses.
 */
export function CodeBlock({ code, title, copyValue, className }: CodeBlockProps) {
  return (
    <figure
      className={cn(
        'min-w-0 overflow-hidden rounded-lg border bg-fd-secondary/50 text-fd-secondary-foreground',
        className
      )}
    >
      <figcaption className="flex items-center justify-between gap-2 border-b py-1 pr-1 pl-3 text-xs font-medium text-fd-muted-foreground">
        {title}
        <CopyButton
          value={copyValue ?? code}
          label={`Copy ${title}`}
        />
      </figcaption>
      {/* Focusable so a keyboard user can scroll a sample wider than the screen. */}
      <pre
        tabIndex={0}
        className="overflow-x-auto p-3 font-mono text-sm leading-relaxed outline-none focus-visible:ring-2 focus-visible:ring-fd-ring focus-visible:ring-inset"
      >
        <code>{code}</code>
      </pre>
    </figure>
  );
}
