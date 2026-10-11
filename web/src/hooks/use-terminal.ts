'use client';

import { useCallback, useEffect, useRef, useState } from 'react';
import { type GnuxSession } from 'lib/playground/runtime.ts';

export type EntryKind = 'command' | 'stdout' | 'stderr' | 'info';

export type Entry = {
  id: number;
  kind: EntryKind;
  text: string;
  /** For a command: the directory it was entered in. */
  cwd?: string;
};

/** Scrollback limits: a runaway command must not take the page down with it. */
const MAX_ENTRIES = 500;
const MAX_ENTRY_CHARS = 200_000;
const TRIMMED = '[earlier output trimmed]\n';

/** Adds entries to the scrollback, joining consecutive output of one stream. */
function append(entries: Entry[], added: Entry[]): Entry[] {
  const next = entries.slice();
  for (const entry of added) {
    const last = next[next.length - 1];
    const joins =
      last && last.kind === entry.kind && (entry.kind === 'stdout' || entry.kind === 'stderr');
    let text = joins ? last.text + entry.text : entry.text;
    if (text.length > MAX_ENTRY_CHARS) {
      // Cut at a line boundary so the first visible line is a whole one.
      const cut = text.indexOf('\n', text.length - MAX_ENTRY_CHARS);
      text = TRIMMED + text.slice(cut === -1 ? text.length - MAX_ENTRY_CHARS : cut + 1);
    }
    if (joins) next[next.length - 1] = { ...last, text };
    else next.push({ ...entry, text });
  }
  return next.length > MAX_ENTRIES ? next.slice(next.length - MAX_ENTRIES) : next;
}

/**
 * The state of one terminal session: its scrollback, command history and
 * working directory, and `run`, which hands a line to gnux.
 */
export function useTerminal(session: GnuxSession, banner: string) {
  const [entries, setEntries] = useState<Entry[]>(() => [{ id: 0, kind: 'info', text: banner }]);
  const [history, setHistory] = useState<string[]>([]);
  const [running, setRunning] = useState(false);
  const [cwd, setCwd] = useState(() => session.cwd());

  const nextId = useRef(1);
  const busy = useRef(false);
  const pending = useRef<Entry[]>([]);
  const frame = useRef(0);

  /** Moves buffered output into the scrollback. */
  const flush = useCallback(() => {
    cancelAnimationFrame(frame.current);
    frame.current = 0;
    if (pending.current.length === 0) return;
    const added = pending.current;
    pending.current = [];
    setEntries((prev) => append(prev, added));
  }, []);

  const print = useCallback(
    (kind: EntryKind, text: string, at?: string) => {
      pending.current.push({ id: nextId.current++, kind, text, cwd: at });
      // gnux can produce thousands of chunks before it yields; one render per
      // frame keeps up with it.
      if (frame.current === 0) frame.current = requestAnimationFrame(flush);
    },
    [flush]
  );

  useEffect(() => {
    // One streaming decoder per stream, so a multi-byte character split
    // across two chunks still decodes.
    const decoders = { 1: new TextDecoder(), 2: new TextDecoder() };
    session.onOutput((fd, chunk) => {
      const text = decoders[fd].decode(chunk, { stream: true });
      if (text !== '') print(fd === 2 ? 'stderr' : 'stdout', text);
    });
    return () => {
      session.onOutput(undefined);
      cancelAnimationFrame(frame.current);
      frame.current = 0;
    };
  }, [session, print]);

  const clear = useCallback(() => {
    pending.current = [];
    setEntries([]);
  }, []);

  const remember = useCallback((line: string) => {
    const command = line.trim();
    if (command === '') return;
    setHistory((prev) => (prev[prev.length - 1] === command ? prev : [...prev, command]));
  }, []);

  /** Shows a line the page handles itself (a playground command) as if it had run. */
  const echo = useCallback(
    (line: string) => {
      remember(line);
      print('command', line, session.cwd());
      flush();
    },
    [remember, print, flush, session]
  );

  const run = useCallback(
    async (line: string) => {
      if (busy.current) return;
      const command = line.trim();
      remember(line);
      // `clear` belongs to the terminal, not to the shell.
      if (command === 'clear') {
        clear();
        return;
      }
      print('command', line, session.cwd());
      flush();
      if (command === '') return;

      busy.current = true;
      setRunning(true);
      try {
        const status = await session.run(line);
        if (status !== 0) print('info', `[exit status ${status}]\n`);
      } catch (err) {
        // The Go program itself failed, which no command should manage.
        console.error(err);
        print(
          'stderr',
          `gnux stopped unexpectedly: ${String(err)}\nRun /refresh to start again.\n`
        );
      }
      flush();
      setCwd(session.cwd());
      busy.current = false;
      setRunning(false);
    },
    [session, print, flush, clear, remember]
  );

  return { entries, history, running, cwd, run, clear, print, echo };
}
