import Link from 'next/link';
import { type ToolSummary } from 'lib/source';
import { Card, Heading, Reveal, Section } from '@/components/ui';

/** Every tool, each linked to its reference page. The list comes in from the docs. */
export function ToolGrid({ tools }: { tools: ToolSummary[] }) {
  return (
    <Section aria-labelledby="tools-title">
      <Reveal className="max-w-2xl">
        <p className="font-mono text-sm text-lit">GNU-compatible</p>
        <Heading
          level={2}
          id="tools-title"
          className="mt-2"
        >
          {tools.length} tools, and the names they answer to
        </Heading>
        <p className="mt-3 text-pretty text-fd-muted-foreground">
          Short flags combine, long flags take values, and diagnostics read as the originals do.
          Each tool runs as its own name, as its GNU name, or as a subcommand of gnux.
        </p>
      </Reveal>
      <ul className="mt-10 grid grid-cols-1 gap-3 rounded-2xl plate p-3 pb-4 sm:grid-cols-2 sm:p-4 sm:pb-5 lg:grid-cols-4">
        {tools.map((tool) => (
          <li
            key={tool.name}
            className="min-w-0"
          >
            <Card
              variant="key"
              className="h-full rounded-xl p-4"
            >
              <div className="flex items-baseline justify-between gap-2">
                <Heading
                  level={3}
                  size="base"
                  className="font-mono"
                >
                  <Link
                    href={tool.url}
                    className="outline-none after:absolute after:inset-0 after:rounded-xl focus-visible:after:ring-2 focus-visible:after:ring-fd-ring focus-visible:after:ring-offset-2 focus-visible:after:ring-offset-fd-background"
                  >
                    {tool.name}
                  </Link>
                </Heading>
                <span className="font-mono text-xs text-lit">
                  <span className="sr-only">GNU name: </span>
                  {tool.gnu}
                </span>
              </div>
              <p className="mt-2 text-sm text-pretty text-fd-muted-foreground">
                {tool.description}
              </p>
            </Card>
          </li>
        ))}
      </ul>
    </Section>
  );
}
