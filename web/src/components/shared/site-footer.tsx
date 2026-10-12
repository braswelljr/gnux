import Link from 'next/link';
import { Container, Logo } from '@/components/ui';
import { independence, project, siteConfig } from '@/config/site';

const linkClass =
  'rounded-sm underline-offset-4 outline-none hover:text-fd-foreground hover:underline focus-visible:ring-2 focus-visible:ring-fd-ring';

/** The foot of the landing page and the playground: where the project lives, and whose it is. */
export function SiteFooter() {
  return (
    <footer className="border-t py-8 text-sm text-fd-muted-foreground">
      <Container className="flex flex-col gap-4 sm:flex-row sm:items-center sm:justify-between">
        <div className="flex flex-col gap-2">
          <p className="flex items-center gap-2 font-mono font-semibold text-fd-foreground">
            <Logo className="size-5" />
            {siteConfig.shortName}
          </p>
          <p className="max-w-md text-xs text-pretty">{independence}</p>
        </div>
        <nav
          aria-label="Footer"
          className="flex flex-wrap gap-x-5 gap-y-2"
        >
          <Link
            href="/docs"
            className={linkClass}
          >
            Docs
          </Link>
          <Link
            href="/install"
            className={linkClass}
          >
            Install
          </Link>
          <Link
            href="/playground"
            className={linkClass}
          >
            Playground
          </Link>
          <a
            href={project.github}
            rel="noreferrer"
            className={linkClass}
          >
            GitHub
          </a>
          <a
            href={project.license}
            rel="noreferrer"
            className={linkClass}
          >
            License
          </a>
        </nav>
      </Container>
    </footer>
  );
}
