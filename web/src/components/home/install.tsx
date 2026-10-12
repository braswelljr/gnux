import Link from 'next/link';
import { ArrowRight } from 'lucide-react';
import { buttonVariants, CodeBlock, GridCell, Heading, Reveal, Section } from '@/components/ui';
import { installMethods } from '@/config/site';

/** How to get gnux onto a machine, each way copyable. */
export function Install() {
  return (
    <Section
      aria-labelledby="install-title"
      index={5}
      label="Install"
      aside="One static binary"
      bleed={
        <>
          <ul className="grid gap-px border-y border-line bg-line">
            {installMethods.map((method, i) => (
              <li
                key={method.title}
                className="min-w-0"
              >
                <GridCell
                  label={method.title}
                  index={i + 1}
                >
                  <Reveal
                    delay={i * 0.06}
                    className="grid gap-4 p-4 sm:p-6 lg:grid-cols-[18rem_minmax(0,1fr)] lg:gap-8"
                  >
                    <div>
                      <Heading level={3}>{method.title}</Heading>
                      <p className="mt-2 text-sm text-pretty text-fd-muted-foreground">
                        {method.summary}
                      </p>
                    </div>
                    <CodeBlock
                      title={`${method.title} install`}
                      code={method.code}
                      className="bg-fd-card"
                    />
                  </Reveal>
                </GridCell>
              </li>
            ))}
          </ul>
          <div className="px-4 py-6 sm:px-6">
            <Link
              href="/install"
              className={buttonVariants({ variant: 'secondary' })}
            >
              Every platform: Homebrew, winget, Scoop, GitHub Actions
              <ArrowRight aria-hidden="true" />
            </Link>
          </div>
        </>
      }
    >
      <Reveal className="max-w-2xl">
        <Heading
          level={2}
          id="install-title"
        >
          Install
        </Heading>
        <p className="mt-3 text-pretty text-fd-muted-foreground">
          No cgo and no runtime dependencies. Link the tool names to the binary and each one is a
          command of its own.
        </p>
      </Reveal>
    </Section>
  );
}
