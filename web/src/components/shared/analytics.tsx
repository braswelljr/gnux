'use client';

import { Analytics as VercelAnalytics, type BeforeSendEvent } from '@vercel/analytics/next';

/**
 * Drops the query and fragment from every reported address: the playground
 * keeps a visitor's command in `?cmd=`, and that must never leave the page.
 */
function withoutQuery(event: BeforeSendEvent): BeforeSendEvent {
  return { ...event, url: event.url.split(/[?#]/, 1)[0] ?? event.url };
}

/** Vercel Web Analytics for the whole site. */
export function Analytics() {
  return <VercelAnalytics beforeSend={withoutQuery} />;
}
