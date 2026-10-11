'use client';

import { useId, useState } from 'react';
import Link from 'next/link';
import { ChevronDown } from 'lucide-react';
import { cn } from 'lib/utils';
import { Button, Heading } from '@/components/ui';
import { type PlaygroundExample } from '@/config/playground';

type SidePanelProps = {
  /** Tool names from the running build; empty until it has loaded. */
  tools: string[];
  version?: string;
  examples: PlaygroundExample[];
  /** True while a command is running, when another cannot start. */
  busy: boolean;
  onRun: (command: string) => void;
  className?: string;
};

/**
 * Examples and the tool list. On small screens it folds away under a button
 * so the terminal keeps the room. Starting over is the `/refresh` command.
 */
export function SidePanel(props: SidePanelProps) {
  const { tools, version, examples, busy, onRun } = props;
  const [open, setOpen] = useState(false);
  const panelId = useId();

  return (
    <aside
      aria-label="Examples and tools"
      className={cn('flex min-h-0 flex-col gap-3', props.className)}
    >
      <div className="flex items-center gap-2 lg:hidden">
        <Button
          onClick={() => setOpen((v) => !v)}
          aria-expanded={open}
          aria-controls={panelId}
          className="flex-1 justify-between"
        >
          Examples and tools
          <ChevronDown
            aria-hidden="true"
            className={cn(
              'transition-transform duration-150 motion-reduce:transition-none',
              open && 'rotate-180'
            )}
          />
        </Button>
      </div>

      <div
        id={panelId}
        className={cn(
          'min-h-0 flex-col gap-5 overflow-y-auto rounded-xl border bg-fd-card p-4 lg:flex lg:flex-1',
          open ? 'flex' : 'hidden'
        )}
      >
        <section aria-labelledby={`${panelId}-examples`}>
          <Heading
            level={2}
            size="sm"
            id={`${panelId}-examples`}
            className="mb-2"
          >
            Try a command
          </Heading>
          {examples.length === 0 ? (
            <p className="text-sm text-fd-muted-foreground">
              Examples appear once gnux has loaded.
            </p>
          ) : (
            <>
              {busy && (
                <p className="mb-2 text-xs text-fd-muted-foreground">
                  A command is running. Examples are available again when it finishes.
                </p>
              )}
              <ul className="flex flex-col gap-1">
                {examples.map((example) => (
                  <li key={example.command}>
                    <Button
                      variant="ghost"
                      disabled={busy}
                      onClick={() => {
                        onRun(example.command);
                        setOpen(false);
                      }}
                      className="h-auto w-full flex-col items-start gap-0.5 px-2 py-1.5 text-left whitespace-normal"
                    >
                      <span className="text-xs font-normal">{example.label}</span>
                      <code className="font-mono text-xs font-normal break-all text-fd-foreground">
                        {example.command}
                      </code>
                    </Button>
                  </li>
                ))}
              </ul>
            </>
          )}
        </section>

        <section aria-labelledby={`${panelId}-tools`}>
          <Heading
            level={2}
            size="sm"
            id={`${panelId}-tools`}
            className="mb-2"
          >
            Tools{tools.length > 0 && ` (${tools.length})`}
          </Heading>
          {tools.length === 0 ? (
            <p className="text-sm text-fd-muted-foreground">
              The tool list appears once gnux has loaded.
            </p>
          ) : (
            <ul className="flex flex-wrap gap-1.5">
              {tools.map((tool) => (
                <li key={tool}>
                  <Link
                    href={`/docs/tools/${tool}`}
                    className="inline-block rounded-md border px-2 py-0.5 font-mono text-xs hover:bg-fd-accent hover:text-fd-accent-foreground"
                  >
                    {tool}
                  </Link>
                </li>
              ))}
            </ul>
          )}
          <p className="mt-2 text-xs text-fd-muted-foreground">
            Each tool links to its reference page.
            {version && (
              <>
                {' '}
                Running gnux <span className="font-mono">{version}</span>.
              </>
            )}
          </p>
        </section>
      </div>
    </aside>
  );
}
