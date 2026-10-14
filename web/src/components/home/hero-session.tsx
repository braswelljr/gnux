'use client';

import { useEffect, type RefObject } from 'react';
import { type GnuxSession } from 'lib/playground/runtime.ts';
import { cn } from 'lib/utils';
import { AnsiText } from '@/components/playground/ansi-text';
import { type GnuxState } from '@/hooks/use-gnux';
import { useTerminal } from '@/hooks/use-terminal';

/** Runs one line the visitor typed. */
export type Runner = (line: string) => void;

/** The screen shows its last lines only, so older entries need not be drawn at all. */
const TAIL = 40;

const BANNER =
  'This is gnux itself, running in your browser on files of its own.\nTry lsx -la, then catx README.txt. Enter runs the line.\n';

/**
 * What the visitor's own session has printed: the real gnux, the one the
 * playground runs, answering the lines typed on the hero's keyboard. It hands
 * `runnerRef` the function that runs a line, for as long as it is on the page.
 */
function Scrollback({
  session,
  runnerRef
}: {
  session: GnuxSession;
  runnerRef: RefObject<Runner | null>;
}) {
  const { entries, run } = useTerminal(session, BANNER);

  useEffect(() => {
    runnerRef.current = (line) => void run(line);
    return () => {
      runnerRef.current = null;
    };
  }, [run, runnerRef]);

  return (
    <>
      {entries.slice(-TAIL).map((entry) =>
        entry.kind === 'command' ? (
          <p
            key={entry.id}
            className="break-words whitespace-pre-wrap"
          >
            <span className="text-violet-700 select-none dark:text-violet-300">$ </span>
            {entry.text}
          </p>
        ) : (
          <p
            key={entry.id}
            className={cn(
              'break-words whitespace-pre-wrap text-fd-muted-foreground',
              entry.kind === 'stderr' && 'text-red-700 dark:text-red-300'
            )}
          >
            <AnsiText text={entry.text} />
          </p>
        )
      )}
    </>
  );
}

/**
 * The hero terminal's screen once the visitor has the keyboard: a note while
 * gnux downloads, the session's scrollback once it runs, or why it could not.
 */
export function HeroSession({
  state,
  runnerRef
}: {
  state: GnuxState;
  runnerRef: RefObject<Runner | null>;
}) {
  if (state.status === 'ready') {
    return (
      <Scrollback
        session={state.session}
        runnerRef={runnerRef}
      />
    );
  }
  return (
    <p className="break-words whitespace-pre-wrap text-fd-muted-foreground">
      {state.status === 'loading'
        ? 'Loading gnux, so that what you type runs...'
        : `gnux could not be loaded here (${state.message}). The playground has the same terminal.`}
    </p>
  );
}
