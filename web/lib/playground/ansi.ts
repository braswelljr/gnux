/**
 * A small reader for the ANSI escape sequences a command-line tool emits.
 * Colors and bold (SGR) are kept as styled spans; every other escape sequence
 * (cursor movement, titles, ...) is dropped, since the playground's scrollback
 * is not a real terminal screen.
 */

export interface AnsiSpan {
  text: string;
  /** Foreground color index: 0-7 normal, 8-15 bright. */
  fg?: number;
  bold?: boolean;
  dim?: boolean;
  underline?: boolean;
}

// CSI sequences (ESC [ ... final byte), OSC sequences (ESC ] ... BEL or ESC \),
// and two-character escapes.
// eslint-disable-next-line no-control-regex -- matching the ESC and BEL control characters is the point
const ESCAPE = /\x1b\[([0-9;:?]*)([ -/]*[@-~])|\x1b\][^\x07\x1b]*(?:\x07|\x1b\\)|\x1b[@-Z\\^_]/g;

/** Removes every escape sequence, leaving the text a terminal would show. */
export function stripAnsi(input: string): string {
  return input.replace(ESCAPE, '');
}

/** Splits text into runs of uniform style. */
export function parseAnsi(input: string): AnsiSpan[] {
  const spans: AnsiSpan[] = [];
  let style: Omit<AnsiSpan, 'text'> = {};
  let last = 0;

  const push = (text: string) => {
    if (text !== '') spans.push({ text, ...style });
  };

  for (const match of input.matchAll(ESCAPE)) {
    push(input.slice(last, match.index));
    last = match.index + match[0].length;
    // Only "select graphic rendition" (final byte m) changes the style.
    if (match[2] === 'm') style = applySgr(style, match[1]);
  }
  push(input.slice(last));
  return spans;
}

function applySgr(style: Omit<AnsiSpan, 'text'>, params: string): Omit<AnsiSpan, 'text'> {
  const next = { ...style };
  const codes = params === '' ? [0] : params.split(';').map((p) => Number(p) || 0);

  for (let i = 0; i < codes.length; i++) {
    const code = codes[i];
    if (code === 0) {
      delete next.fg;
      delete next.bold;
      delete next.dim;
      delete next.underline;
    } else if (code === 1) next.bold = true;
    else if (code === 2) next.dim = true;
    else if (code === 4) next.underline = true;
    else if (code === 22) {
      delete next.bold;
      delete next.dim;
    } else if (code === 24) delete next.underline;
    else if (code >= 30 && code <= 37) next.fg = code - 30;
    else if (code >= 90 && code <= 97) next.fg = code - 90 + 8;
    else if (code === 39) delete next.fg;
    else if (code === 38 || code === 48) {
      // Extended colors carry arguments (5;n or 2;r;g;b). They are not
      // rendered, but their arguments must not be read as codes of their own.
      i += codes[i + 1] === 5 ? 2 : codes[i + 1] === 2 ? 4 : 0;
    }
  }
  return next;
}
