'use client';

import { useState, type CSSProperties } from 'react';
import Link from 'next/link';
import { Pause, Play } from 'lucide-react';
import type { ToolSummary } from 'lib/source';
import { Button, Container, Marquee } from '@/components/ui';
import { useReducedMotion } from '@/hooks/use-reduced-motion';

/** A tool as a marquee item: its name on a key, and what it does. */
function ToolChip({ tool }: { tool: ToolSummary }) {
  return (
    <Link
      href={tool.url}
      className="flex items-center gap-3 rounded-xl border bg-fd-card/80 py-2 pr-4 pl-2 text-sm outline-none hover:bg-fd-card focus-visible:ring-2 focus-visible:ring-fd-ring"
    >
      <span className="keycap rounded-lg px-2 py-1 font-mono text-xs font-semibold keycap-mod [--cap-depth:3px]">
        {tool.name}
      </span>
      <span className="max-w-72 truncate text-fd-muted-foreground">{tool.description}</span>
    </Link>
  );
}

/** One row of tools, as a list, on the marquee. */
function ToolRow({ tools, reverse }: { tools: ToolSummary[]; reverse?: boolean }) {
  return (
    <Marquee
      pauseOnHover
      reverse={reverse}
      repeat={2}
      style={{ '--duration': '120s', '--gap': '0.75rem' } as CSSProperties}
      className="p-0 py-1"
    >
      <ul className="flex gap-(--gap)">
        {tools.map((tool) => (
          <li
            key={tool.name}
            className="shrink-0"
          >
            <ToolChip tool={tool} />
          </li>
        ))}
      </ul>
    </Marquee>
  );
}

/**
 * The tools sliding past in two rows, each with what it does: the breadth of
 * the bundle at a glance, under the hero. Motion that lasts must be stoppable,
 * so besides pausing on hover and focus it has a pause button; under reduced
 * motion it stands still and the button is not needed.
 */
export function ToolMarquee({ tools }: { tools: ToolSummary[] }) {
  const [paused, setPaused] = useState(false);
  const reduced = useReducedMotion();
  const half = Math.ceil(tools.length / 2);

  return (
    <section
      aria-labelledby="marquee-title"
      className="py-6"
    >
      <Container className="mb-3 flex items-center justify-between gap-4">
        <h2
          id="marquee-title"
          className="font-mono text-sm text-lit"
        >
          {tools.length} tools, one binary
        </h2>
        {!reduced && (
          <Button
            variant="ghost"
            size="icon-sm"
            aria-label={paused ? 'Play the tool marquee' : 'Pause the tool marquee'}
            aria-pressed={paused}
            onClick={() => setPaused((p) => !p)}
          >
            {paused ? <Play aria-hidden="true" /> : <Pause aria-hidden="true" />}
          </Button>
        )}
      </Container>
      <div
        data-paused={paused}
        className="flex flex-col gap-2 [mask-image:linear-gradient(to_right,transparent,black_6%,black_94%,transparent)]"
      >
        <ToolRow tools={tools.slice(0, half)} />
        <ToolRow
          tools={tools.slice(half)}
          reverse
        />
      </div>
    </section>
  );
}
