'use client';

import { useCallback, useEffect, useMemo, useRef, useState } from 'react';
import { LoaderCircle, TriangleAlert } from 'lucide-react';
import { parseAsString, useQueryState } from 'nuqs';
import { completeSlash, helpText, parseSlash, type ToolHelp } from 'lib/playground/commands.ts';
import { complete } from 'lib/playground/complete.ts';
import { type GnuxSession } from 'lib/playground/runtime.ts';
import { HOME } from 'lib/playground/seed.ts';
import { cn } from 'lib/utils';
import { SidePanel } from '@/components/playground/side-panel';
import { Terminal } from '@/components/playground/terminal';
import { Button, TerminalWindow } from '@/components/ui';
import { builtins, examples, type PlaygroundExample } from '@/config/playground';
import { useGnux } from '@/hooks/use-gnux';
import { useTerminal } from '@/hooks/use-terminal';

const GRID = 'grid gap-4 lg:grid-cols-[minmax(0,1fr)_19rem]';
const FRAME = 'h-[62dvh] min-h-80 lg:h-[70dvh]';
const PANEL = 'lg:h-[70dvh]';

/** The longest command a link may carry. */
const MAX_COMMAND = 500;

/**
 * A URL parameter is input from outside: anyone can write the link. It is only
 * ever placed in the prompt for the reader to run or discard, never run on
 * arrival, and it is cut to one printable line first.
 */
function readCommand(value: unknown): string {
  if (typeof value !== 'string') return '';
  return value
    .replace(/\p{Cc}/gu, ' ')
    .trim()
    .slice(0, MAX_COMMAND);
}

/** The examples as `/examples` lists them: numbered, so `/examples N` can run one. */
function exampleList(available: PlaygroundExample[]) {
  const width = Math.max(...available.map((e) => e.label.length));
  return `${available
    .map((e, i) => `  ${String(i + 1).padStart(2)}  ${e.label.padEnd(width)}  ${e.command}`)
    .join('\n')}\nRun one with /examples N.\n`;
}

/**
 * Full screen for the terminal: the browser's own where it allows it, and the
 * terminal laid over the page either way, so it works where the API does not
 * (iOS Safari). Esc leaves it in both cases.
 */
function useFullscreen(target: React.RefObject<HTMLElement | null>) {
  const [full, setFull] = useState(false);

  useEffect(() => {
    if (!full) return;
    // The browser leaving full screen (its own Esc) leaves ours too.
    const onChange = () => {
      if (!document.fullscreenElement) setFull(false);
    };
    const onKey = (event: KeyboardEvent) => {
      if (event.key === 'Escape' && !document.fullscreenElement) setFull(false);
    };
    document.addEventListener('fullscreenchange', onChange);
    document.addEventListener('keydown', onKey);
    return () => {
      document.removeEventListener('fullscreenchange', onChange);
      document.removeEventListener('keydown', onKey);
    };
  }, [full]);

  const toggle = useCallback(() => {
    if (full) {
      setFull(false);
      if (document.fullscreenElement) void document.exitFullscreen().catch(() => {});
      return;
    }
    setFull(true);
    void target.current?.requestFullscreen?.().catch(() => {});
  }, [full, target]);

  return [full, toggle] as const;
}

type WorkspaceProps = {
  session: GnuxSession;
  /** What each tool does, for `/help`. */
  toolHelp: ToolHelp[];
  /** The command in the URL when the session started. */
  initialCommand: string;
  onCommand: (command: string) => void;
  onReset: () => void;
};

/** A running session: the terminal and the panel that drives it. */
function Workspace({ session, toolHelp, initialCommand, onCommand, onReset }: WorkspaceProps) {
  const { fs } = session;
  const banner = `gnux ${session.version}, ${session.tools.length} tools. Type /help for every command, or pick an example.\n`;
  const terminal = useTerminal(session, banner);
  const inputRef = useRef<HTMLInputElement>(null);
  const root = useRef<HTMLDivElement>(null);
  const [fullscreen, toggleFullscreen] = useFullscreen(root);

  // The prompt keeps the focus across the change of layout. Only a change:
  // on the first render the focus stays wherever the visitor has put it.
  const laidOut = useRef(fullscreen);
  useEffect(() => {
    if (laidOut.current === fullscreen) return;
    laidOut.current = fullscreen;
    inputRef.current?.focus({ preventScroll: true });
  }, [fullscreen]);

  const tools = useMemo(() => [...session.tools].sort(), [session]);
  const available = useMemo(
    () => examples.filter((example) => example.needs.every((tool) => tools.includes(tool))),
    [tools]
  );
  const commands = useMemo(() => [...tools, ...builtins], [tools]);

  const onComplete = useCallback(
    (line: string, cursor: number) =>
      completeSlash(line, cursor) ?? complete({ line, cursor, commands, fs, home: HOME }),
    [commands, fs]
  );

  const { run, echo, print, clear, history } = terminal;

  const runShell = useCallback(
    (line: string) => {
      // The last command run is what the page is showing, so it goes in the URL.
      if (line.trim() !== '') onCommand(line.trim());
      void run(line);
    },
    [run, onCommand]
  );

  /** Runs a line: a `/` command is the page's own, anything else goes to gnux. */
  const runLine = useCallback(
    (line: string) => {
      const slash = parseSlash(line);
      if (!slash) {
        runShell(line);
        return;
      }
      echo(line);
      const [arg] = slash.args;
      switch (slash.name) {
        case 'help':
          print('stdout', helpText(tools, toolHelp));
          break;
        case 'examples': {
          if (arg === undefined) {
            print('stdout', exampleList(available));
            break;
          }
          const example = available[Number(arg) - 1];
          if (example) runShell(example.command);
          else print('stderr', `/examples: no example ${arg}; there are ${available.length}\n`);
          break;
        }
        case 'docs': {
          if (arg !== undefined && !tools.includes(arg)) {
            print('stderr', `/docs: no tool named ${arg}. /help lists them.\n`);
            break;
          }
          const path = arg === undefined ? '/docs' : `/docs/tools/${arg}`;
          window.open(path, '_blank', 'noopener');
          print('info', `Opened ${path} in a new tab.\n`);
          break;
        }
        case 'history':
          print(
            'stdout',
            history.length === 0
              ? 'Nothing run yet.\n'
              : `${history.map((h, i) => `  ${String(i + 1).padStart(3)}  ${h}`).join('\n')}\n`
          );
          break;
        case 'clear':
          clear();
          break;
        case 'fullscreen':
          toggleFullscreen();
          break;
        case 'refresh':
          onReset();
          break;
        default:
          print('stderr', `/${slash.word}: not a playground command. /help lists them.\n`);
      }
    },
    [runShell, echo, print, clear, history, tools, toolHelp, available, toggleFullscreen, onReset]
  );
  const runExample = useCallback(
    (command: string) => {
      runLine(command);
      inputRef.current?.focus({ preventScroll: true });
    },
    [runLine]
  );

  return (
    <div
      ref={root}
      className={cn(
        fullscreen ? 'fixed inset-0 z-50 flex flex-col bg-fd-background p-2 sm:p-4' : GRID
      )}
    >
      <Terminal
        className={fullscreen ? 'flex-1' : FRAME}
        entries={terminal.entries}
        history={terminal.history}
        running={terminal.running}
        cwd={terminal.cwd}
        home={HOME}
        initialValue={initialCommand}
        inputRef={inputRef}
        onRun={runLine}
        onClear={terminal.clear}
        onComplete={onComplete}
        onPrint={terminal.print}
        onRefresh={onReset}
        fullscreen={fullscreen}
        onFullscreen={toggleFullscreen}
      />
      {!fullscreen && (
        <SidePanel
          className={PANEL}
          tools={tools}
          version={session.version}
          examples={available}
          busy={terminal.running}
          onRun={runExample}
        />
      )}
    </div>
  );
}

/** What stands in for the terminal while gnux downloads, or when it cannot. */
function Placeholder({ error, onReset }: { error?: string; onReset: () => void }) {
  return (
    <div className={GRID}>
      <TerminalWindow
        as="section"
        aria-label="Terminal"
        title="guest@gnux"
        glow
        className={FRAME}
      >
        <div className="flex flex-1 flex-col items-center justify-center gap-3 p-6 text-center">
          {error === undefined ? (
            <div
              role="status"
              className="flex flex-col items-center gap-3"
            >
              <LoaderCircle
                aria-hidden="true"
                className="size-6 animate-spin text-fd-muted-foreground motion-reduce:animate-none"
              />
              <p className="text-sm font-medium">Loading gnux…</p>
              <p className="max-w-sm text-sm text-pretty text-fd-muted-foreground">
                Downloading the WebAssembly build, a few megabytes. Your browser caches it after the
                first visit.
              </p>
            </div>
          ) : (
            <div
              role="alert"
              className="flex flex-col items-center gap-3"
            >
              <TriangleAlert
                aria-hidden="true"
                className="size-6 text-red-600 dark:text-red-400"
              />
              <p className="text-sm font-medium">The playground could not start</p>
              <p className="max-w-md font-mono text-xs break-words text-fd-muted-foreground">
                {error}
              </p>
              <Button
                variant="primary"
                onClick={onReset}
              >
                Try again
              </Button>
            </div>
          )}
        </div>
      </TerminalWindow>
      <SidePanel
        className={PANEL}
        tools={[]}
        examples={[]}
        busy={false}
        onRun={() => {}}
      />
    </div>
  );
}

/**
 * The in-browser playground. gnux.wasm is fetched from an effect, so it loads
 * only in the browser and only when this component is on the page.
 */
export function Playground({ toolHelp }: { toolHelp: ToolHelp[] }) {
  const { state, generation, reset } = useGnux();
  // `?cmd=` is the one piece of state a reader can arrive with: a link to the
  // playground with a command ready at the prompt.
  const [cmd, setCmd] = useQueryState('cmd', parseAsString.withDefault(''));
  const onCommand = useCallback((command: string) => void setCmd(command), [setCmd]);

  if (state.status === 'ready') {
    return (
      <Workspace
        key={generation}
        session={state.session}
        toolHelp={toolHelp}
        initialCommand={readCommand(cmd)}
        onCommand={onCommand}
        onReset={reset}
      />
    );
  }
  return (
    <Placeholder
      error={state.status === 'error' ? state.message : undefined}
      onReset={reset}
    />
  );
}
