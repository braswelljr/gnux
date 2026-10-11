'use client';

import { useCallback, useEffect, useState } from 'react';
import { bootInBrowser } from 'lib/playground/browser.ts';
import { type GnuxSession } from 'lib/playground/runtime.ts';

export type GnuxState =
  | { status: 'loading' }
  | { status: 'ready'; session: GnuxSession }
  | { status: 'error'; message: string };

/**
 * Loads gnux.wasm in the browser and keeps one running instance.
 *
 * `reset` throws the instance away, filesystem included, and boots a new one.
 * `generation` changes with every boot, so it can key the UI that holds
 * per-session state.
 */
export function useGnux() {
  const [state, setState] = useState<GnuxState>({ status: 'loading' });
  const [generation, setGeneration] = useState(0);

  useEffect(() => {
    let cancelled = false;
    let current: GnuxSession | undefined;

    bootInBrowser().then(
      (session) => {
        // An effect that was cleaned up while the module was downloading
        // (a fast /refresh, or React's development double-mount) owns nothing.
        if (cancelled) {
          session.dispose();
          return;
        }
        current = session;
        setState({ status: 'ready', session });
      },
      (err: unknown) => {
        if (cancelled) return;
        console.error(err);
        setState({ status: 'error', message: err instanceof Error ? err.message : String(err) });
      }
    );

    return () => {
      cancelled = true;
      current?.dispose();
    };
  }, [generation]);

  const reset = useCallback(() => {
    setState({ status: 'loading' });
    setGeneration((n) => n + 1);
  }, []);

  return { state, generation, reset };
}
