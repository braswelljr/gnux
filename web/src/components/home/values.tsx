import { Boxes, Globe, Wrench } from 'lucide-react';
import { Card, CardDescription, CardHeader, Heading, Reveal, Section } from '@/components/ui';

const values = [
  {
    icon: Boxes,
    title: 'One static binary',
    description:
      'Every tool in one file, built without cgo. Nothing to install alongside it, nothing that breaks when the system upgrades.'
  },
  {
    icon: Globe,
    title: 'The same everywhere',
    description:
      'Linux, macOS, FreeBSD, Windows and the browser run the same code, so a script prints the same output and exits with the same status on each.'
  },
  {
    icon: Wrench,
    title: 'Fixes, not just copies',
    description:
      'GNU flags and messages, minus its rough edges: dux reads trees in parallel, recursive chmodx refuses /, nicex and nohupx really work on Windows. Every difference is documented.'
  }
];

/** Why gnux rather than a port: three promises, each kept in the code. */
export function Values() {
  return (
    <Section aria-labelledby="values-title">
      <Reveal className="max-w-2xl">
        <p className="font-mono text-sm text-lit">Why gnux</p>
        <Heading
          level={2}
          id="values-title"
          className="mt-2"
        >
          Compatible where it counts. Better where it can be.
        </Heading>
      </Reveal>
      <ul className="mt-10 grid gap-4 md:grid-cols-3">
        {values.map(({ icon: Icon, title, description }, i) => (
          <li
            key={title}
            className="min-w-0"
          >
            <Reveal
              delay={i * 0.06}
              className="h-full"
            >
              <Card
                variant="plate"
                className="h-full"
              >
                <CardHeader className="gap-3 pb-5">
                  <Icon
                    aria-hidden="true"
                    className="size-6 text-lit"
                  />
                  <Heading level={3}>{title}</Heading>
                  <CardDescription>{description}</CardDescription>
                </CardHeader>
              </Card>
            </Reveal>
          </li>
        ))}
      </ul>
    </Section>
  );
}
