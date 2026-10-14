'use client';

import { useCallback, useEffect, useRef, useState } from 'react';
import { Keyboard as KeyboardIcon } from 'lucide-react';
import { HeroSession, type Runner } from '@/components/home/hero-session';
import { Button, Kbd, Keyboard, Monitor, TypingTerminal } from '@/components/ui';
import { heroSession } from '@/config/demos';
import { useGnux } from '@/hooks/use-gnux';
import { useInView } from '@/hooks/use-in-view';
import { type Keystroke } from '@/hooks/use-typewriter';

/** The longest line the visitor's typing grows to before it stops taking characters. */
const MAX_LINE = 160;

/** Whether a key event belongs to a field of the page (search, a form) rather than to the scene. */
function inField(target: EventTarget | null) {
  return (
    target instanceof HTMLElement &&
    (target.isContentEditable || ['INPUT', 'TEXTAREA', 'SELECT'].includes(target.tagName))
  );
}

/**
 * Lets the visitor take the keyboard from the animation. Ctrl+Enter, or the
 * button, takes it: the typing stops and
 * every key the visitor presses goes down under the hands, and onto the
 * prompt. Enter hands the line to `submit` and the up and down arrows bring
 * back earlier lines. Esc, or Ctrl+Enter again, gives the keyboard back. The
 * shortcut only answers while the scene is on screen, and keys typed into a
 * field of the page, or with a modifier, are left alone. `press` is the same
 * handling for a key clicked on the drawn keyboard, which only answers once
 * the keyboard is the visitor's.
 */
function useTakeover(scene: React.RefObject<HTMLElement | null>, submit: (line: string) => void) {
  const inView = useInView(scene);
  const [mine, setMine] = useState(false);
  const [line, setLine] = useState('');
  // Whether the keyboard has ever been taken: what the session waits for.
  const [taken, setTaken] = useState(false);
  const history = useRef<string[]>([]);
  // How far back in the history the prompt is showing; 0 is the line being typed.
  const back = useRef(0);

  const flip = useCallback(() => {
    setMine((m) => !m);
    setTaken(true);
    setLine('');
    back.current = 0;
  }, []);

  /**
   * Handles one key, by its `KeyboardEvent.key`, while the keyboard is the
   * visitor's. It reports whether the key was one of the scene's, whose
   * default (scrolling the page, for Space and the arrows) must not happen.
   */
  const handle = useCallback(
    (key: string): boolean => {
      if (key === 'Escape') {
        flip();
        return true;
      }
      if (key === 'Enter') {
        if (line.trim() !== '' && history.current[history.current.length - 1] !== line) {
          history.current.push(line);
        }
        back.current = 0;
        submit(line);
        setLine('');
        return true;
      }
      if (key === 'Backspace') {
        setLine((l) => l.slice(0, -1));
        return true;
      }
      if (key === 'ArrowUp' || key === 'ArrowDown') {
        const step = key === 'ArrowUp' ? 1 : -1;
        const next = Math.min(Math.max(back.current + step, 0), history.current.length);
        if (next !== back.current) {
          back.current = next;
          setLine(next === 0 ? '' : history.current[history.current.length - next]);
        }
        return true;
      }
      if (key.length === 1) {
        setLine((l) => (l.length < MAX_LINE ? l + key : l));
        return true;
      }
      return false;
    },
    [flip, line, submit]
  );

  useEffect(() => {
    if (!inView) return;
    const onKey = (event: KeyboardEvent) => {
      if (inField(event.target)) return;
      const ctrl = event.ctrlKey && !event.metaKey && !event.altKey;
      if (ctrl && event.key === 'Enter') {
        event.preventDefault();
        flip();
        return;
      }
      if (!mine) return;
      // Shortcuts of the browser and the page (Cmd+K search) keep working.
      if (event.ctrlKey || event.metaKey || event.altKey) return;
      if (handle(event.key)) event.preventDefault();
    };
    window.addEventListener('keydown', onKey);
    return () => window.removeEventListener('keydown', onKey);
  }, [inView, mine, flip, handle]);

  const press = useCallback(
    (key: string) => {
      if (mine) handle(key);
    },
    [mine, handle]
  );

  // Off screen nobody is typing at the scene, so the animation has it back.
  return { mine: mine && inView, taken, line, toggle: flip, press };
}

/**
 * The hero's centerpiece: a terminal typing real gnux commands, set above a
 * keyboard whose keys go down under two hands as they are typed. The terminal
 * drives the keyboard, so the two never drift apart, until the visitor takes
 * the keyboard over and the hands follow their keys instead. From then on the
 * terminal is the real one: gnux is downloaded, as the playground does it, and
 * each line the visitor enters is run by it. Nothing is downloaded before the
 * keyboard is taken. On a phone the keyboard is left out and the terminal
 * carries the scene alone.
 */
export function HeroStage() {
  const [stroke, setStroke] = useState<Keystroke | null>(null);
  const scene = useRef<HTMLDivElement>(null);
  const button = useRef<HTMLButtonElement>(null);
  const runner = useRef<Runner | null>(null);
  // Until gnux has loaded there is nothing to run a line with, and it is dropped.
  const submit = useCallback((entered: string) => runner.current?.(entered), []);
  const { mine, taken, line, toggle, press } = useTakeover(scene, submit);
  const { state } = useGnux(taken);

  // The button keeps the focus while the keyboard is the visitor's, so Space
  // and Enter would press it again: the focus moves off it, and comes back
  // when the keyboard is handed back.
  const wasMine = useRef(false);
  useEffect(() => {
    if (mine && document.activeElement === button.current) button.current?.blur();
    if (!mine && wasMine.current && document.activeElement === document.body) {
      button.current?.focus({ preventScroll: true });
    }
    wasMine.current = mine;
  }, [mine]);

  return (
    <div
      ref={scene}
      // From the breakpoint where the keyboard joins the monitor, the scene is
      // sized against the viewport's height too, so both are in view at once.
      className="relative mx-auto w-full max-w-4xl sm:max-w-[min(56rem,calc(100svh-12rem))]"
    >
      {/* The terminal on a monitor on the desk; the keyboard sits in front of its foot. */}
      <Monitor
        className="relative z-0 sm:w-[86%]"
        standClassName="hidden sm:flex"
        screenClassName="aspect-video p-0 sm:p-0"
      >
        <TypingTerminal
          scenes={heroSession}
          label="An animation of gnux in a terminal: listing a directory with lsx, counting lines with catx and wcx, removing a build directory with rmx, running a script through the gnux shell, and stopping an endless yesx with headx."
          title={mine ? 'you@gnux: ~/project (your keyboard)' : 'you@gnux: ~/project'}
          onStroke={setStroke}
          takenOver={mine ? line : undefined}
          scrollback={
            <HeroSession
              state={state}
              runnerRef={runner}
            />
          }
          className="relative mx-auto h-full"
          windowClassName="h-full rounded-none border-0 bg-fd-card shadow-none backdrop-blur-none"
          screenClassName="h-auto min-h-0 flex-1 sm:h-auto"
        />
      </Monitor>
      <div className="relative z-10 -mt-3 hidden sm:block">
        <div
          aria-hidden="true"
          className="[perspective:1800px]"
        >
          <div className="origin-top [transform:rotateX(22deg)]">
            <Keyboard
              stroke={mine ? null : stroke}
              live={mine}
              onPress={mine ? press : undefined}
              className="mx-auto"
            />
          </div>
        </div>
        <div className="relative mt-24 flex flex-col items-center gap-3 text-center">
          <Button
            ref={button}
            size="sm"
            variant={mine ? 'primary' : 'secondary'}
            aria-pressed={mine}
            onClick={toggle}
          >
            <KeyboardIcon aria-hidden="true" />
            {mine ? 'Give the keyboard back' : 'Take the keyboard'}
          </Button>
          <p
            role="status"
            className="font-mono text-xs text-fd-muted-foreground"
          >
            {mine ? (
              <>
                The hands follow your keys, or click the drawn ones; <Kbd>Enter</Kbd> runs the line.{' '}
                <Kbd>Esc</Kbd> gives the keyboard back.
              </>
            ) : (
              <>
                Press <Kbd>Ctrl</Kbd> + <Kbd>Enter</Kbd> to take the keyboard and type yourself.
              </>
            )}
          </p>
        </div>
      </div>
    </div>
  );
}
