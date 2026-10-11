import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CodeBlock,
  Heading,
  Reveal,
  Section
} from '@/components/ui';
import { installMethods } from '@/config/site';

/** How to get gnux onto a machine, each way copyable. */
export function Install() {
  return (
    <Section aria-labelledby="install-title">
      <Reveal className="max-w-2xl">
        <p className="font-mono text-sm text-lit">One static binary</p>
        <Heading
          level={2}
          id="install-title"
          className="mt-2"
        >
          Install
        </Heading>
        <p className="mt-3 text-pretty text-fd-muted-foreground">
          No cgo and no runtime dependencies. Link the tool names to the binary and each one is a
          command of its own.
        </p>
      </Reveal>
      <ul className="mt-10 grid gap-4">
        {installMethods.map((method, i) => (
          <li
            key={method.title}
            className="min-w-0"
          >
            <Reveal
              delay={i * 0.06}
              className="h-full"
            >
              <Card
                variant="plate"
                className="h-full lg:grid lg:grid-cols-[18rem_minmax(0,1fr)]"
              >
                <CardHeader className="lg:pb-5">
                  <Heading level={3}>{method.title}</Heading>
                  <CardDescription>{method.summary}</CardDescription>
                </CardHeader>
                <CardContent className="lg:p-5 lg:pl-0">
                  <CodeBlock
                    title={`${method.title} install`}
                    code={method.code}
                    className="flex-1 bg-fd-card"
                  />
                </CardContent>
              </Card>
            </Reveal>
          </li>
        ))}
      </ul>
    </Section>
  );
}
