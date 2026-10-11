'use client';

import {
  memo,
  useEffect,
  useId,
  useMemo,
  useRef,
  useState,
  type KeyboardEvent,
  type RefObject
} from 'react';
import { Maximize2, Minimize2 } from 'lucide-react';
import { suggest } from 'lib/playground/commands.ts';
import { type Completion } from 'lib/playground/complete.ts';
import { cn } from 'lib/utils';
import { AnsiText } from '@/components/playground/ansi-text';
import { ShareButton } from '@/components/playground/share-button';
import { Button, TerminalWindow } from '@/components/ui';
import { type Entry } from '@/hooks/use-terminal';

type TerminalProps = {
  entries: Entry[];
  history: string[];
  running: boolean;
  cwd: string;
  home: string;
  /** What the prompt starts with: a command that arrived in the URL. */
  initialValue: string;
  inputRef: RefObject<HTMLInputElement | null>;
  onRun: (line: string) => void;
  onClear: () => void;
  onComplete: (line: string, cursor: number) => Completion;
  onPrint: (kind: 'command' | 'info', text: string, cwd?: string) => void;
  /** Throw the session away and start a fresh one: `/refresh`, or Esc while a command hangs. */
  onRefresh: () => void;
  /** Whether the terminal fills the screen, and the control that toggles it. */
  fullscreen: boolean;
  onFullscreen: () => void;
  className?: string;
};

/** Shows the home directory as `~`, as a shell prompt does. */
function shortPath(path: string, home: string) {
  if (path === home) return '~';
  return path.startsWith(`${home}/`) ? `~${path.slice(home.length)}` : path;
}

function Prompt({ cwd, home }: { cwd: string; home: string }) {
  return (
    <span className="shrink-0 select-none">
      <span className="text-green-700 dark:text-green-300">guest@gnux</span>
      <span className="text-fd-muted-foreground">:</span>
      <span className="text-blue-700 dark:text-blue-300">{shortPath(cwd, home)}</span>
      <span className="text-fd-muted-foreground">$ </span>
    </span>
  );
}

const Line = memo(function Line({ entry, home }: { entry: Entry; home: string }) {
  if (entry.kind === 'command') {
    return (
      <div className="flex">
        <Prompt
          cwd={entry.cwd ?? home}
          home={home}
        />
        <span className="break-all whitespace-pre-wrap">{entry.text}</span>
      </div>
    );
  }
  return (
    <div
      className={cn(
        'break-words whitespace-pre-wrap',
        entry.kind === 'stderr' && 'text-red-700 dark:text-red-300',
        entry.kind === 'info' && 'text-fd-muted-foreground'
      )}
    >
      {/* Color alone must not carry the meaning: the stream is named for screen readers. */}
      {entry.kind === 'stderr' && <span className="sr-only">Error output: </span>}
      <AnsiText text={entry.text} />
    </div>
  );
});

/**
 * The terminal pane: scrollback above, one input line below. It holds only what
 * is being typed; everything that persists comes in through props.
 */
export function Terminal(props: TerminalProps) {
  const {
    entries,
    history,
    running,
    cwd,
    home,
    inputRef,
    onRun,
    onClear,
    onComplete,
    onPrint,
    onRefresh,
    fullscreen,
    onFullscreen
  } = props;
  const [value, setValue] = useState(props.initialValue);
  // Position in history while browsing it; null while editing a new line.
  const [cursor, setCursor] = useState<number | null>(null);
  const [draft, setDraft] = useState('');
  // The grey suggestion is only offered with the caret at the end of the line.
  const [atEnd, setAtEnd] = useState(true);
  const scroller = useRef<HTMLDivElement>(null);
  const inputId = useId();
  const hintId = useId();

  // Keep the newest output in view.
  useEffect(() => {
    const el = scroller.current;
    if (el) el.scrollTop = el.scrollHeight;
  }, [entries, running]);

  // Typing resumes where it left off once a command finishes.
  useEffect(() => {
    if (!running && document.activeElement === document.body) {
      inputRef.current?.focus({ preventScroll: true });
    }
  }, [running, inputRef]);

  // What the line could become: the rest of it, shown greyed after the caret.
  const suggestion = useMemo(
    () => (atEnd && !running ? suggest(value, history, onComplete(value, value.length)) : ''),
    [atEnd, running, value, history, onComplete]
  );

  function accept(input: HTMLInputElement) {
    const next = value + suggestion;
    edit(next);
    requestAnimationFrame(() => input.setSelectionRange(next.length, next.length));
  }

  function edit(next: string) {
    setValue(next);
    setCursor(null);
  }

  function submit() {
    onRun(value);
    edit('');
    setDraft('');
  }

  function recall(step: -1 | 1) {
    if (history.length === 0) return;
    if (step === -1) {
      const index = cursor === null ? history.length - 1 : Math.max(0, cursor - 1);
      if (cursor === null) setDraft(value);
      setCursor(index);
      setValue(history[index]);
    } else if (cursor !== null) {
      const index = cursor + 1;
      if (index >= history.length) {
        setCursor(null);
        setValue(draft);
      } else {
        setCursor(index);
        setValue(history[index]);
      }
    }
  }

  function completeAtCaret(input: HTMLInputElement) {
    const result = onComplete(value, input.selectionStart ?? value.length);
    // Nothing for completion to add, but history has a line that starts so.
    if (result.line === value && result.candidates.length === 0 && suggestion !== '') {
      accept(input);
      return;
    }
    if (result.candidates.length > 1) {
      onPrint('command', value, cwd);
      onPrint('info', `${result.candidates.join('  ')}\n`);
    }
    if (result.line === value) return;
    edit(result.line);
    // The caret can only be placed once React has written the new value.
    requestAnimationFrame(() => input.setSelectionRange(result.cursor, result.cursor));
  }

  function onKeyDown(event: KeyboardEvent<HTMLInputElement>) {
    const ctrl = event.ctrlKey && !event.altKey && !event.metaKey;
    if (running) {
      // A WebAssembly call cannot be interrupted, so a command that never
      // returns is stopped the only way there is: a fresh session.
      const copying = ctrl && window.getSelection()?.isCollapsed === false;
      if (event.key === 'Escape' || (ctrl && event.key.toLowerCase() === 'c' && !copying)) {
        event.preventDefault();
        onRefresh();
      }
      return;
    }
    const input = event.currentTarget;
    const caretAtEnd = input.selectionStart === value.length && input.selectionEnd === value.length;

    if (event.key === 'Enter') {
      event.preventDefault();
      submit();
    } else if (event.key === 'ArrowUp') {
      event.preventDefault();
      recall(-1);
    } else if (event.key === 'ArrowDown') {
      event.preventDefault();
      recall(1);
    } else if (event.key === 'ArrowRight' && caretAtEnd && suggestion !== '') {
      event.preventDefault();
      accept(input);
    } else if (event.key === 'Tab' && !event.shiftKey && !ctrl && !event.altKey && !event.metaKey) {
      // With nothing typed, Tab keeps its usual job of moving focus, so the
      // terminal is never a keyboard trap.
      if (value === '') return;
      event.preventDefault();
      completeAtCaret(event.currentTarget);
    } else if (ctrl && event.key.toLowerCase() === 'l') {
      event.preventDefault();
      onClear();
    } else if (ctrl && event.key.toLowerCase() === 'c' && value !== '') {
      // Leave Ctrl+C alone when it would copy a selection.
      if (input.selectionStart !== input.selectionEnd) return;
      event.preventDefault();
      onPrint('command', `${value}^C`, cwd);
      edit('');
    }
  }

  // What a shared link should carry: the line being typed, else the last one run.
  const shareable = value.trim() || history[history.length - 1] || '';

  return (
    <TerminalWindow
      as="section"
      aria-label="Terminal"
      title={`guest@gnux: ${shortPath(cwd, home)}`}
      actions={
        <Button
          variant="ghost"
          size="icon-sm"
          onClick={onFullscreen}
          aria-label={fullscreen ? 'Exit full screen' : 'Full screen'}
          title={fullscreen ? 'Exit full screen (/fullscreen)' : 'Full screen (/fullscreen)'}
          className="-my-1 size-7"
        >
          {fullscreen ? <Minimize2 aria-hidden="true" /> : <Maximize2 aria-hidden="true" />}
        </Button>
      }
      glow
      className={cn('min-h-0 font-mono text-sm leading-relaxed', props.className)}
    >
      {/* A click anywhere in the scrollback focuses the prompt, as in a terminal.
          It is a convenience for pointer users on top of the labeled input, which
          is what a keyboard or screen reader user reaches. */}
      <div
        ref={scroller}
        className="min-h-0 flex-1 cursor-text overflow-y-auto overscroll-contain px-3 pt-3 sm:px-4"
        onClick={() => {
          // Not while text is being selected for copying.
          if (window.getSelection()?.isCollapsed !== false) inputRef.current?.focus();
        }}
      >
        <div
          role="log"
          aria-live="polite"
          aria-relevant="additions"
          aria-label="Terminal output"
        >
          {entries.map((entry) => (
            <Line
              key={entry.id}
              entry={entry}
              home={home}
            />
          ))}
        </div>

        <div className="flex items-center pb-3">
          {running ? (
            <span
              role="status"
              className="text-fd-muted-foreground"
            >
              running… (Esc or Ctrl+C starts over, if it never returns)
            </span>
          ) : (
            <Prompt
              cwd={cwd}
              home={home}
            />
          )}
          <label
            htmlFor={inputId}
            className="sr-only"
          >
            Command
          </label>
          <span className="relative flex min-w-0 flex-1">
            <input
              ref={inputRef}
              id={inputId}
              type="text"
              name="command"
              value={value}
              readOnly={running}
              onChange={(event) => {
                edit(event.target.value);
                setAtEnd(event.target.selectionStart === event.target.value.length);
              }}
              onSelect={(event) => {
                const el = event.currentTarget;
                setAtEnd(
                  el.selectionStart === el.value.length && el.selectionEnd === el.value.length
                );
              }}
              onKeyDown={onKeyDown}
              aria-describedby={hintId}
              autoCapitalize="none"
              autoCorrect="off"
              autoComplete="off"
              spellCheck={false}
              enterKeyHint="send"
              // 16px on small screens: anything smaller makes iOS zoom the page on focus.
              className={cn(
                'min-w-0 flex-1 bg-transparent text-base caret-violet-600 outline-none [caret-shape:block] sm:text-sm dark:caret-violet-400',
                running && 'sr-only'
              )}
            />
            {/* The suggestion: the typed text, invisible, holds its place, and
              the rest is shown greyed. Screen readers hear it through the hint. */}
            {suggestion !== '' && (
              <span
                aria-hidden="true"
                className="pointer-events-none absolute inset-0 overflow-hidden text-base whitespace-pre sm:text-sm"
              >
                <span className="invisible">{value}</span>
                <span className="text-fd-muted-foreground opacity-70">{suggestion}</span>
              </span>
            )}
          </span>
        </div>
      </div>
      <footer className="flex flex-wrap items-center justify-between gap-x-4 gap-y-1 border-t px-3 py-1 font-sans text-xs text-fd-muted-foreground sm:px-4">
        <p id={hintId}>
          Type <kbd className="font-mono text-fd-foreground">/help</kbd> for every command. Tab
          completes, Right arrow takes the grey suggestion
          {suggestion !== '' && <span className="sr-only">: {value + suggestion}</span>}
        </p>
        {shareable !== '' && <ShareButton command={shareable} />}
      </footer>
    </TerminalWindow>
  );
}
