import Link from 'next/link';
import { type ToolSummary } from 'lib/source';
import { cn } from 'lib/utils';
import { GridCell, Heading, Reveal, Section } from '@/components/ui';

/**
 * Blank cells that complete the grid's last row, so its lines close instead of
 * the line color showing through: one at two columns when the count is odd,
 * enough for four columns at the widest.
 */
function Fillers({ count }: { count: number }) {
  const atTwo = count % 2;
  const atFour = (4 - (count % 4)) % 4;
  return Array.from({ length: atFour }, (_, i) => (
    <li
      key={i}
      aria-hidden="true"
      className={cn('hidden bg-fd-background/70', i < atTwo ? 'sm:block' : 'lg:block')}
    />
  ));
}

/** Every tool, each linked to its reference page. The list comes in from the docs. */
export function ToolGrid({ tools }: { tools: ToolSummary[] }) {
  return (
    <Section
      aria-labelledby="tools-title"
      index={4}
      label="Tools"
      aside="GNU-compatible"
      bleed={
        <ul className="grid grid-cols-1 gap-px border-t border-line bg-line sm:grid-cols-2 lg:grid-cols-4">
          {tools.map((tool, i) => (
            <li
              key={tool.name}
              className="min-w-0"
            >
              <GridCell
                label={
                  <span className="normal-case">
                    <span className="sr-only">GNU name: </span>
                    {tool.gnu}
                  </span>
                }
                index={i + 1}
                className="relative transition-colors hover:bg-fd-card"
              >
                <div className="p-4 sm:px-6">
                  <Heading
                    level={3}
                    size="base"
                    className="font-mono"
                  >
                    <Link
                      href={tool.url}
                      className="outline-none after:absolute after:inset-0 focus-visible:after:ring-2 focus-visible:after:ring-fd-ring focus-visible:after:ring-inset"
                    >
                      {tool.name}
                    </Link>
                  </Heading>
                  <p className="mt-2 text-sm text-pretty text-fd-muted-foreground">
                    {tool.description}
                  </p>
                </div>
              </GridCell>
            </li>
          ))}
          <Fillers count={tools.length} />
        </ul>
      }
    >
      <Reveal className="max-w-2xl">
        <Heading
          level={2}
          id="tools-title"
        >
          {tools.length} tools, and the names they answer to
        </Heading>
        <p className="mt-3 text-pretty text-fd-muted-foreground">
          Short flags combine, long flags take values, and diagnostics read as the originals do.
          Each tool runs as its own name, as its GNU name, or as a subcommand of gnux.
        </p>
      </Reveal>
    </Section>
  );
}
