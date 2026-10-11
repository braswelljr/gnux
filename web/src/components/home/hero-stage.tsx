'use client';

import { useEffect, useRef, useState } from 'react';
import { Keyboard as KeyboardIcon } from 'lucide-react';
import { Button, Kbd, Keyboard, TypingTerminal } from '@/components/ui';
import { heroSession } from '@/config/demos';
import { useInView } from '@/hooks/use-in-view';
import { type Keystroke } from '@/hooks/use-typewriter';

/** The longest line the visitor's typing grows to before it stops taking characters. */
const MAX_LINE = 64;

/** Whether a key event belongs to a field of the page (search, a form) rather than to the scene. */
function inField(target: EventTarget | null) {
  return (
    target instanceof HTMLElement &&
    (target.isContentEditable || ['INPUT', 'TEXTAREA', 'SELECT'].includes(target.tagName))
  );
}

/**
 * Lets the visitor take the keyboard from the animation. Ctrl+Enter, or the
 * button, takes it: the typing stops and every key the visitor presses goes
 * down under the hands, and onto the prompt. Esc, or Ctrl+Enter again, gives
 * it back. The shortcut only answers while the scene is on screen, and keys
 * typed into a field of the page, or with a modifier, are left alone.
 */
function useTakeover(scene: React.RefObject<HTMLElement | null>) {
  const inView = useInView(scene);
  const [mine, setMine] = useState(false);
  const [line, setLine] = useState('');

  useEffect(() => {
    if (!inView) return;
    const onKey = (event: KeyboardEvent) => {
      if (inField(event.target)) return;
      const ctrl = event.ctrlKey && !event.metaKey && !event.altKey;
      if (ctrl && event.key === 'Enter') {
        event.preventDefault();
        setMine((m) => !m);
        setLine('');
        return;
      }
      if (!mine) return;
      if (event.key === 'Escape') {
        setMine(false);
        setLine('');
        return;
      }
      // Shortcuts of the browser and the page (Cmd+K search) keep working.
      if (event.ctrlKey || event.metaKey || event.altKey) return;
      if (event.key === 'Enter') {
        event.preventDefault();
        setLine('');
      } else if (event.key === 'Backspace') {
        event.preventDefault();
        setLine((l) => l.slice(0, -1));
      } else if (event.key.length === 1) {
        // Space would otherwise scroll the page out from under the keyboard.
        event.preventDefault();
        setLine((l) => (l.length < MAX_LINE ? l + event.key : l));
      }
    };
    window.addEventListener('keydown', onKey);
    return () => window.removeEventListener('keydown', onKey);
  }, [inView, mine]);

  const toggle = () => {
    setMine((m) => !m);
    setLine('');
  };

  // Off screen nobody is typing at the scene, so the animation has it back.
  return { mine: mine && inView, line, toggle };
}

/**
 * The hero's centerpiece: a terminal typing real gnux commands, set above a
 * keyboard whose keys go down under two hands as they are typed. The terminal
 * drives the keyboard, so the two never drift apart, until the visitor takes
 * the keyboard over and the hands follow their keys instead. On a phone the
 * keyboard is left out and the terminal carries the scene alone.
 */
export function HeroStage() {
  const [stroke, setStroke] = useState<Keystroke | null>(null);
  const scene = useRef<HTMLDivElement>(null);
  const button = useRef<HTMLButtonElement>(null);
  const { mine, line, toggle } = useTakeover(scene);

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
      className="relative mx-auto w-full max-w-4xl"
    >
      <TypingTerminal
        scenes={heroSession}
        label="An animation of gnux in a terminal: listing a directory with lsx, counting lines with catx and wcx, removing a build directory with rmx, running a script through the gnux shell, and stopping an endless yesx with headx."
        title={mine ? 'you@gnux: ~/project (your keyboard)' : 'you@gnux: ~/project'}
        glow
        onStroke={setStroke}
        takenOver={mine ? line : undefined}
        className="relative z-10 mx-auto max-w-3xl"
        screenClassName="h-64 sm:h-72"
      />
      <div className="relative z-0 -mt-2 hidden sm:block">
        <div
          aria-hidden="true"
          className="[perspective:1800px]"
        >
          <div className="origin-top [transform:rotateX(22deg)]">
            <Keyboard
              stroke={mine ? null : stroke}
              live={mine}
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
                The hands follow your keys now. <Kbd>Esc</Kbd> gives the keyboard back.
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
