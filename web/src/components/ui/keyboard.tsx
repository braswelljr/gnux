'use client';

import { memo, useEffect, useId, useState } from 'react';
import { cn } from 'lib/utils';
import { Keycap, type KeycapTone } from '@/components/ui/keycap';
import { type Keystroke } from '@/hooks/use-typewriter';

type KeyDef = {
  id: string;
  /** What is printed on the cap. */
  legend: string;
  /** The shifted character, printed above the legend on a symbol key. */
  upper?: string;
  /** Width in key units. */
  w?: number;
  tone?: KeycapTone;
  /** The characters the key types, plain and shifted. */
  chars?: [string, string];
  /** The `KeyboardEvent.code` values that press it. */
  codes: string[];
};

const CODE_OF: Record<string, string> = {
  '`': 'Backquote',
  '-': 'Minus',
  '=': 'Equal',
  '[': 'BracketLeft',
  ']': 'BracketRight',
  '\\': 'Backslash',
  ';': 'Semicolon',
  "'": 'Quote',
  ',': 'Comma',
  '.': 'Period',
  '/': 'Slash'
};

function codeOf(c: string) {
  if (/[a-z]/.test(c)) return `Key${c.toUpperCase()}`;
  if (/[0-9]/.test(c)) return `Digit${c}`;
  return CODE_OF[c];
}

/** A symbol key: both characters printed, shifted above plain. */
const sym = (plain: string, shifted: string): KeyDef => ({
  id: plain,
  legend: plain,
  upper: shifted,
  chars: [plain, shifted],
  codes: [codeOf(plain)]
});
const letters = (row: string): KeyDef[] =>
  [...row].map((c) => ({
    id: c,
    legend: c.toUpperCase(),
    chars: [c, c.toUpperCase()],
    codes: [codeOf(c)]
  }));
const mod = (id: string, legend: string, w: number, codes: string[], tone: KeycapTone = 'mod') => ({
  id,
  legend,
  w,
  tone,
  codes
});

/** A US ANSI layout with a Mac bottom row, row by row. Every row is 15 units wide. */
const ROWS: KeyDef[][] = [
  [
    sym('`', '~'),
    ...[...'1234567890'].map((c, i) => sym(c, '!@#$%^&*()'[i])),
    sym('-', '_'),
    sym('=', '+'),
    mod('backspace', 'delete', 2, ['Backspace'])
  ],
  [
    mod('tab', 'tab', 1.5, ['Tab']),
    ...letters('qwertyuiop'),
    sym('[', '{'),
    sym(']', '}'),
    { ...sym('\\', '|'), w: 1.5 }
  ],
  [
    mod('caps', 'caps lock', 1.75, ['CapsLock']),
    ...letters('asdfghjkl'),
    sym(';', ':'),
    sym("'", '"'),
    mod('enter', 'return', 2.25, ['Enter', 'NumpadEnter'], 'accent')
  ],
  [
    mod('lshift', 'shift', 2.25, ['ShiftLeft']),
    ...letters('zxcvbnm'),
    sym(',', '<'),
    sym('.', '>'),
    sym('/', '?'),
    mod('rshift', 'shift', 2.75, ['ShiftRight'])
  ],
  [
    mod('fn', 'fn', 1.25, ['Fn']),
    mod('ctrl', 'control', 1.25, ['ControlLeft']),
    mod('opt', 'option', 1.25, ['AltLeft']),
    mod('cmd', 'command', 1.5, ['MetaLeft']),
    mod('space', '', 5.75, ['Space'], 'alpha'),
    mod('rcmd', 'command', 1.5, ['MetaRight']),
    mod('ropt', 'option', 1.25, ['AltRight']),
    mod('rctrl', 'control', 1.25, ['ControlRight'])
  ]
];

const KEYS = ROWS.flat();

/** The key, and whether shift is held, that types each character. */
const BY_CHAR = new Map<string, { id: string; shift: boolean }>();
for (const k of KEYS) {
  if (!k.chars) continue;
  BY_CHAR.set(k.chars[0], { id: k.id, shift: false });
  BY_CHAR.set(k.chars[1], { id: k.id, shift: true });
}
BY_CHAR.set(' ', { id: 'space', shift: false });
BY_CHAR.set('\n', { id: 'enter', shift: false });

const BY_CODE = new Map<string, string>();
for (const k of KEYS) for (const code of k.codes) BY_CODE.set(code, k.id);

/** Keys on the left half: a shifted one of these is shifted with the right hand. */
const LEFT = new Set([...'`12345qwertasdfgzxcvb']);

/**
 * The keys the visitor is holding down on their own keyboard, by key id. It
 * only listens while `active`, and lets go of everything when the window
 * loses focus, since the matching keyup will never arrive.
 */
function useHeldKeys(active: boolean) {
  const [held, setHeld] = useState<ReadonlySet<string>>(() => new Set());

  useEffect(() => {
    if (!active) return;
    const change = (code: string, down: boolean) => {
      const id = BY_CODE.get(code);
      if (!id) return;
      setHeld((prev) => {
        if (prev.has(id) === down) return prev;
        const next = new Set(prev);
        if (down) next.add(id);
        else next.delete(id);
        return next;
      });
    };
    const clear = () => setHeld(new Set());
    const onDown = (e: KeyboardEvent) => change(e.code, true);
    const onUp = (e: KeyboardEvent) => {
      // macOS sends no keyup for a key released while Command is down, so
      // letting go of Command lets go of everything.
      if (e.key === 'Meta') clear();
      else change(e.code, false);
    };
    const onHide = () => {
      if (document.hidden) clear();
    };
    window.addEventListener('keydown', onDown);
    window.addEventListener('keyup', onUp);
    window.addEventListener('blur', clear);
    document.addEventListener('visibilitychange', onHide);
    return () => {
      window.removeEventListener('keydown', onDown);
      window.removeEventListener('keyup', onUp);
      window.removeEventListener('blur', clear);
      document.removeEventListener('visibilitychange', onHide);
      clear();
    };
  }, [active]);

  return held;
}

/**
 * The layout's geometry, in percent of the keyboard's width (the `cqw` the
 * keys are sized in): each row is `ROW` tall with `GAP` between keys and rows.
 * The hands are drawn in the same units, so they land on the keys they press.
 */
const GAP = 0.6;
const ROW = 6;
const UNIT = (100 + GAP) / 15;
const HEIGHT = 5 * ROW + 4 * GAP;

type Placed = { cx: number; cy: number; width: number };

const PLACED = new Map<string, Placed>();
ROWS.forEach((row, r) => {
  let units = 0;
  for (const k of row) {
    const w = k.w ?? 1;
    const width = w * UNIT - GAP;
    PLACED.set(k.id, { cx: units * UNIT + width / 2, cy: r * (ROW + GAP) + ROW / 2, width });
    units += w;
  }
});
const at = (id: string) => PLACED.get(id) as Placed;

type FingerName = 'L5' | 'L4' | 'L3' | 'L2' | 'L1' | 'R1' | 'R2' | 'R3' | 'R4' | 'R5';

/** Touch typing: the keys each finger is responsible for. */
const REACH: Record<FingerName, string[]> = {
  L5: [...'`1qaz', 'tab', 'caps', 'lshift', 'fn', 'ctrl'],
  L4: [...'2wsx', 'opt'],
  L3: [...'3edc'],
  L2: [...'45rtfgvb'],
  L1: ['cmd'],
  R1: ['space', 'rcmd'],
  R2: [...'67yuhjnm'],
  R3: [...'8ik,'],
  R4: [...'9ol.', 'ropt'],
  R5: [..."0-=p[]\\;'/", 'enter', 'rshift', 'backspace', 'rctrl']
};

const FINGER_OF = new Map<string, FingerName>();
for (const [finger, ids] of Object.entries(REACH)) {
  for (const id of ids) FINGER_OF.set(id, finger as FingerName);
}

type Finger = { name: FingerName; tip: [number, number]; base: [number, number]; width: number };

/** Builds one hand resting on the home row: four fingers, a thumb on the space bar, a palm. */
function hand(side: 'L' | 'R') {
  const home = side === 'L' ? ['a', 's', 'd', 'f'] : [';', 'l', 'k', 'j'];
  const widths = [3, 3.5, 3.65, 3.65];
  const tips = home.map((id) => [at(id).cx, at(id).cy] as [number, number]);
  const center = (tips[0][0] + tips[3][0]) / 2;
  const baseY = HEIGHT + 2.2;
  const fingers: Finger[] = home.map((_, i) => ({
    name: `${side}${5 - i}` as FingerName,
    tip: tips[i],
    base: [tips[i][0] + (center - tips[i][0]) * 0.3, baseY],
    width: widths[i]
  }));
  const space = at('space');
  const toward = side === 'L' ? 1 : -1;
  const thumbTip: [number, number] = [space.cx + space.width * toward * -0.22, space.cy];
  fingers.push({
    name: `${side}1` as FingerName,
    tip: thumbTip,
    base: [thumbTip[0] - toward * 7, baseY + 3.4],
    width: 4
  });
  const left = Math.min(...fingers.map((f) => f.base[0])) - 2.8;
  const right = Math.max(...fingers.map((f) => f.base[0])) + 2.8;
  const palm = { x: left, y: baseY - 1.8, width: right - left, height: 11 };
  return { fingers, palm };
}

const HANDS = { L: hand('L'), R: hand('R') };

/**
 * Where each finger is now: the one pressing a key reaches it, and its hand
 * follows a little. A wide key is pressed near its edge, as a real finger does.
 */
function reach(down: ReadonlySet<string>) {
  const offsets = new Map<FingerName, [number, number]>();
  const handShift: Record<'L' | 'R', [number, number]> = { L: [0, 0], R: [0, 0] };
  for (const id of down) {
    const finger = FINGER_OF.get(id);
    if (!finger) continue;
    const side = finger[0] as 'L' | 'R';
    const f = HANDS[side].fingers.find((x) => x.name === finger);
    if (!f) continue;
    const target = at(id);
    const tx =
      target.width > UNIT * 1.5
        ? f.tip[0] +
          (target.cx > f.tip[0] ? 1 : -1) *
            Math.min(Math.abs(target.cx - f.tip[0]), target.width / 2 - 2.1)
        : target.cx;
    const dx = tx - f.tip[0];
    const dy = target.cy - f.tip[1] + 0.3;
    offsets.set(finger, [dx, dy]);
    handShift[side] = [dx * 0.22, dy * 0.22];
  }
  return { offsets, handShift };
}

/** The two hands, drawn over the keys in the keyboard's own units. */
function Hands({ down }: { down: ReadonlySet<string> }) {
  const { offsets, handShift } = reach(down);
  const fade = `${useId().replace(/:/g, '')}-fade`;

  return (
    <svg
      viewBox={`0 0 100 ${HEIGHT}`}
      focusable="false"
      className="pointer-events-none absolute inset-x-0 top-0 h-auto w-full overflow-visible"
    >
      <defs>
        <linearGradient
          id={`${fade}-g`}
          x1="0"
          y1="0"
          x2="0"
          y2="1"
        >
          <stop
            offset="0.6"
            stopColor="white"
          />
          <stop
            offset="0.85"
            stopColor="white"
            stopOpacity="0"
          />
        </linearGradient>
        <mask
          id={fade}
          maskUnits="userSpaceOnUse"
          x="-10"
          y="-5"
          width="120"
          height={HEIGHT + 30}
        >
          <rect
            x="-10"
            y="-5"
            width="120"
            height={HEIGHT + 30}
            fill={`url(#${fade}-g)`}
          />
        </mask>
      </defs>
      <g
        mask={`url(#${fade})`}
        opacity="0.78"
      >
        {(['L', 'R'] as const).map((side) => {
          const { fingers, palm } = HANDS[side];
          const [hx, hy] = handShift[side];
          const move = (finger: FingerName, ms: number) => {
            const [x, y] = offsets.get(finger) ?? [hx, hy];
            return {
              transform: `translate(${x}px, ${y}px)`,
              transition: `transform ${ms}ms ease-out`
            };
          };
          const palmStyle = {
            transform: `translate(${hx}px, ${hy}px)`,
            transition: 'transform 120ms ease-out'
          };
          const capsule = (f: Finger, extra: number, paint: string) => (
            <line
              key={`${f.name}-${extra}`}
              x1={f.tip[0]}
              y1={f.tip[1]}
              x2={f.base[0]}
              y2={f.base[1]}
              stroke={paint}
              strokeWidth={f.width + extra}
              strokeLinecap="round"
              style={move(f.name, 90)}
            />
          );
          return (
            <g key={side}>
              {/* Outlines first, then fills over them, so the hand reads as one shape. */}
              <rect
                {...palm}
                rx="6.7"
                fill="none"
                stroke="var(--hand-stroke)"
                strokeWidth="0.5"
                style={palmStyle}
              />
              {fingers.map((f) => capsule(f, 0.5, 'var(--hand-stroke)'))}
              <rect
                {...palm}
                rx="6.7"
                fill="var(--hand-fill)"
                style={palmStyle}
              />
              {fingers.map((f) => capsule(f, 0, 'var(--hand-fill)'))}
              {fingers.map((f) =>
                offsets.has(f.name) ? (
                  <circle
                    key={`${f.name}-tip`}
                    cx={f.tip[0]}
                    cy={f.tip[1] - f.width * 0.15}
                    r={f.width * 0.24}
                    fill="var(--cap-accent-face)"
                    style={move(f.name, 90)}
                  />
                ) : null
              )}
            </g>
          );
        })}
      </g>
    </svg>
  );
}

type KeyboardProps = {
  /** The key an animation is pressing now, if any. */
  stroke?: Keystroke | null;
  /** Also press the keys the visitor presses on their own keyboard. */
  live?: boolean;
  /** Draw the two hands that type. */
  hands?: boolean;
  className?: string;
};

/**
 * A mechanical keyboard built from keycaps, with two hands typing on it. A
 * key goes down and lights when an animation types it (`stroke`) or, with
 * `live`, when the visitor presses it, and the finger touch typing assigns to
 * that key reaches for it; a shifted character holds the shift key under the
 * other hand. The hands are translucent so every legend stays readable. It never
 * takes focus or stops a keystroke, and it is decorative, so it is hidden from
 * assistive technology.
 */
export const Keyboard = memo(function Keyboard({
  stroke,
  live = false,
  hands = true,
  className
}: KeyboardProps) {
  const held = useHeldKeys(live);
  const down = new Set(held);
  const typed = stroke ? BY_CHAR.get(stroke.key) : undefined;
  if (typed) {
    down.add(typed.id);
    if (typed.shift) down.add(LEFT.has(typed.id) ? 'rshift' : 'lshift');
  }

  return (
    <div
      aria-hidden="true"
      className={cn('rounded-2xl plate p-2 select-none sm:p-3', className)}
    >
      <div className="@container">
        <div className="relative flex flex-col gap-[0.6cqw]">
          {ROWS.map((row, r) => (
            <div
              key={r}
              className="grid h-[6cqw] grid-cols-[repeat(60,minmax(0,1fr))] gap-[0.6cqw]"
            >
              {row.map((k) => {
                const on = down.has(k.id);
                const wide = (k.w ?? 1) > 1;
                return (
                  <Keycap
                    key={k.id}
                    tone={k.tone}
                    pressed={on}
                    style={{ gridColumn: `span ${(k.w ?? 1) * 4}` }}
                    className="flex overflow-hidden rounded-[0.9cqw] font-mono [--cap-depth:0.5cqw]"
                  >
                    {/* The lit face: on at once, fading out after the key comes back up. */}
                    <span
                      className={cn(
                        'absolute inset-0 bg-lit-face transition-opacity motion-reduce:transition-none',
                        on ? 'opacity-90 duration-0' : 'opacity-0 duration-700'
                      )}
                    />
                    <span
                      className={cn(
                        'relative flex size-full p-[0.7cqw] text-xs leading-none transition-colors motion-reduce:transition-none',
                        wide ? 'items-end justify-start' : 'items-center justify-center',
                        k.upper && 'flex-col gap-[0.4cqw]',
                        on ? 'text-white duration-0' : 'duration-700'
                      )}
                    >
                      {k.upper && <span className="opacity-60">{k.upper}</span>}
                      {k.legend}
                    </span>
                  </Keycap>
                );
              })}
            </div>
          ))}
          {hands && <Hands down={down} />}
        </div>
      </div>
    </div>
  );
});
