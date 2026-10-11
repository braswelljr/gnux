/**
 * The playground's own commands. A line starting with `/` belongs to the page,
 * not to the gnux shell: it clears the screen, starts over, opens the docs,
 * and so on. Everything here is pure, so it is tested without a browser; the
 * page decides what each command does.
 */
import type { Completion } from './complete.ts';

export type SlashName =
  'help' | 'examples' | 'docs' | 'history' | 'clear' | 'fullscreen' | 'refresh';

export interface SlashCommand {
  name: SlashName;
  /** How its operand is written, if it takes one: `[TOOL]`. */
  args?: string;
  summary: string;
}

/** Every playground command, in the order `/help` lists them. */
export const SLASH_COMMANDS: SlashCommand[] = [
  { name: 'help', summary: 'this list: the playground commands and every tool' },
  { name: 'examples', args: '[N]', summary: 'commands to try; with a number, run that one' },
  { name: 'docs', args: '[TOOL]', summary: "open the reference, or one tool's page, in a new tab" },
  { name: 'history', summary: 'the commands run so far' },
  { name: 'clear', summary: 'clear the screen (also: clear, Ctrl+L)' },
  { name: 'fullscreen', summary: 'expand the terminal to fill the screen; again to leave' },
  { name: 'refresh', summary: 'start over: a fresh gnux and the original files' }
];

export interface SlashLine {
  /** The command, or null for a `/word` that is not one. */
  name: SlashName | null;
  /** What was typed after the slash, before any operands. */
  word: string;
  args: string[];
}

/** Reads a playground command line; null when the line is for the shell. */
export function parseSlash(line: string): SlashLine | null {
  const trimmed = line.trim();
  if (!trimmed.startsWith('/')) return null;
  const [head, ...args] = trimmed.slice(1).split(/\s+/);
  const word = head.toLowerCase();
  const known = SLASH_COMMANDS.find((c) => c.name === word);
  return { name: known ? known.name : null, word, args };
}

/**
 * Completes a playground command name: `/ref` becomes `/refresh `. It answers
 * only while the caret is in the first word of a `/` line, and returns null
 * otherwise so the shell's completion can take over.
 */
export function completeSlash(line: string, cursor: number): Completion | null {
  if (!line.startsWith('/')) return null;
  const head = line.slice(0, cursor);
  if (/\s/.test(head)) return null;
  const typed = head.slice(1).toLowerCase();
  const matches = SLASH_COMMANDS.map((c) => c.name).filter((name) => name.startsWith(typed));
  if (matches.length === 0) return { line, cursor, candidates: [] };
  if (matches.length === 1) {
    const done = `/${matches[0]} `;
    const rest = line.slice(cursor).replace(/^\S*/, '').replace(/^\s*/, '');
    return { line: done + rest, cursor: done.length, candidates: [] };
  }
  let prefix: string = matches[0];
  for (const name of matches) {
    while (!name.startsWith(prefix)) prefix = prefix.slice(0, -1);
  }
  const grown = `/${prefix}`;
  return {
    line: grown + line.slice(cursor),
    cursor: grown.length,
    candidates: matches.map((name) => `/${name}`)
  };
}

/**
 * The rest of the line to show greyed after the caret, for Right arrow or Tab
 * to accept: the newest command in history that starts with what is typed,
 * else what completion would add when it has one answer. Empty when there is
 * nothing to offer.
 */
export function suggest(line: string, history: string[], completion?: Completion): string {
  if (line.trim() === '') return '';
  for (let i = history.length - 1; i >= 0; i--) {
    if (history[i].startsWith(line) && history[i] !== line) return history[i].slice(line.length);
  }
  if (completion && completion.candidates.length === 0 && completion.line.startsWith(line)) {
    return completion.line.slice(line.length);
  }
  return '';
}

/** A tool as `/help` lists it. */
export interface ToolHelp {
  name: string;
  description: string;
}

function table(rows: [string, string][]): string {
  const width = Math.max(...rows.map(([left]) => left.length)) + 2;
  return rows.map(([left, right]) => `  ${left.padEnd(width)}${right}`).join('\n');
}

/**
 * The text `/help` prints: the playground commands, every tool in the running
 * build with what it does, and where the shell's own help is.
 */
export function helpText(tools: string[], described: ToolHelp[]): string {
  const about = new Map(described.map((t) => [t.name, t.description.replace(/\.$/, '')]));
  const commands = table(
    SLASH_COMMANDS.map((c) => [`/${c.name}${c.args ? ` ${c.args}` : ''}`, c.summary])
  );
  const toolRows = table(tools.map((name) => [name, about.get(name) ?? '']));
  return [
    'Playground commands',
    commands,
    '',
    `Tools (${tools.length})`,
    toolRows,
    '',
    'Shell: cd, exit and help are built in. Run "help" for the shell syntax,',
    'and "TOOL --help" for the options of one tool.',
    'Keys: Tab completes, Right arrow takes the grey suggestion, Up and Down',
    'recall, Ctrl+L clears, Esc or Ctrl+C refreshes a command that never returns.',
    ''
  ].join('\n');
}
