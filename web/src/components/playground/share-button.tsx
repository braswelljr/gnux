'use client';

import { Check, Link2 } from 'lucide-react';
import { Button } from '@/components/ui';
import { useCopyToClipboard } from '@/hooks/use-copy-to-clipboard';

/**
 * Copies a link that opens the playground with `command` typed at the prompt.
 * The command is in the URL (`?cmd=`), so the link is the whole of the state.
 */
export function ShareButton({ command }: { command: string }) {
  const { copyToClipboard, isCopied, error } = useCopyToClipboard();

  function copyLink() {
    const url = new URL(window.location.href);
    url.search = '';
    url.searchParams.set('cmd', command);
    void copyToClipboard(url.toString());
  }

  return (
    <span className="inline-flex items-center gap-2">
      {error && (
        <span
          role="status"
          className="text-xs"
        >
          Could not copy. Copy the address bar instead.
        </span>
      )}
      <Button
        variant="ghost"
        size="sm"
        onClick={copyLink}
        title="Copy a link that opens the playground with this command"
      >
        {isCopied ? <Check aria-hidden="true" /> : <Link2 aria-hidden="true" />}
        {isCopied ? 'Link copied' : 'Copy link'}
      </Button>
    </span>
  );
}
