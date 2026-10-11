'use client';

import { useEffect, useRef, type ReactNode } from 'react';
import { cn } from 'lib/utils';
import { Caret } from '@/components/ui/caret';
import { TerminalWindow } from '@/components/ui/terminal-window';
import { useInView } from '@/hooks/use-in-view';
import { useReducedMotion } from '@/hooks/use-reduced-motion';
import { useTypewriter, type Keystroke, type TypingScene } from '@/hooks/use-typewriter';

type TypingTerminalProps = {
  scenes: TypingScene[];
  /** Says what the animation shows, for readers who cannot see it. */
  label: string;
  /** The prompt before each command. */
  prompt?: ReactNode;
  /** The title bar's text. */
  title?: ReactNode;
  /** Characters per millisecond, roughly: lower is faster. */
  speed?: number;
  loop?: boolean;
  glow?: boolean;
  /** Called on every key down and key up, to drive a keyboard drawn beside it. */
  onStroke?: (stroke: Keystroke | null) => void;
  /**
   * Someone else has the keyboard: the animation holds where it is, and this
   * line, what they are typing, sits at a fresh prompt below it.
   */
  takenOver?: string;
  className?: string;
  /** Sets the screen's height; older lines scroll off the top. */
  screenClassName?: string;
};

/**
 * A terminal that types real gnux commands and prints what they print. It is a
 * picture of a session, not a session: the window is hidden from assistive
 * technology and the same commands and output are given as text beside it.
 * It only runs while on screen, and under reduced motion it shows the finished
 * session and stays still.
 */
export function TypingTerminal({
  scenes,
  label,
  prompt = '$',
  title,
  speed,
  loop = true,
  glow,
  onStroke,
  takenOver,
  className,
  screenClassName
}: TypingTerminalProps) {
  const ref = useRef<HTMLElement>(null);
  const inView = useInView(ref, '0px 0px -15% 0px');
  const reduced = useReducedMotion();
  const { scenes: shown, stroke } = useTypewriter(scenes, {
    active: inView && takenOver === undefined,
    reduced,
    speed,
    loop
  });
  const typing = takenOver === undefined && shown.some((scene) => scene.typing);

  useEffect(() => onStroke?.(stroke), [stroke, onStroke]);

  return (
    <figure
      ref={ref}
      className={cn('min-w-0 text-left', className)}
    >
      <figcaption className="sr-only">{label}</figcaption>
      <TerminalWindow
        aria-hidden="true"
        title={title}
        glow={glow}
        data-paused={!inView}
      >
        <div
          className={cn(
            'flex h-56 flex-col justify-end overflow-hidden p-3 font-mono text-xs leading-relaxed sm:p-4 sm:text-sm',
            screenClassName
          )}
        >
          {shown.map((scene, i) => (
            <div key={i}>
              <p className="break-words whitespace-pre-wrap">
                <span className="text-violet-700 select-none dark:text-violet-300">{prompt} </span>
                {scene.command}
                {scene.typing && takenOver === undefined && <Caret blinking={false} />}
              </p>
              {scene.output !== undefined && (
                <p className="break-words whitespace-pre-wrap text-fd-muted-foreground">
                  {scene.output}
                </p>
              )}
            </div>
          ))}
          {!typing && (
            <p className="break-words whitespace-pre-wrap">
              <span className="text-violet-700 select-none dark:text-violet-300">{prompt} </span>
              {takenOver}
              <Caret blinking={!takenOver} />
            </p>
          )}
        </div>
      </TerminalWindow>
      <ol className="sr-only">
        {scenes.map((scene) => (
          <li key={scene.command}>
            <code>{scene.command}</code>
            {scene.output ? (
              <>
                {' '}
                prints <samp>{scene.output}</samp>
              </>
            ) : (
              ' prints nothing'
            )}
          </li>
        ))}
      </ol>
    </figure>
  );
}
