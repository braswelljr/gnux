import assert from 'node:assert/strict';
import { describe, test } from 'node:test';
import {
  completeSlash,
  helpText,
  parseSlash,
  SLASH_COMMANDS,
  suggest
} from '../lib/playground/commands.ts';

describe('parseSlash', () => {
  test('a shell line is not a playground command', () => {
    assert.equal(parseSlash('lsx -la'), null);
    assert.equal(parseSlash('catx /etc/hosts'), null);
  });

  test('reads the name and operands', () => {
    assert.deepEqual(parseSlash('  /docs  rmx '), { name: 'docs', word: 'docs', args: ['rmx'] });
    assert.deepEqual(parseSlash('/REFRESH'), { name: 'refresh', word: 'refresh', args: [] });
  });

  test('an unknown name is reported, not passed to the shell', () => {
    assert.deepEqual(parseSlash('/nope'), { name: null, word: 'nope', args: [] });
  });
});

describe('completeSlash', () => {
  test('one match completes with a trailing space', () => {
    assert.deepEqual(completeSlash('/ref', 4), { line: '/refresh ', cursor: 9, candidates: [] });
  });

  test('several matches grow to their common prefix and are listed', () => {
    const result = completeSlash('/h', 2);
    assert.deepEqual(result, { line: '/h', cursor: 2, candidates: ['/help', '/history'] });
  });

  test('leaves shell lines and operands to the shell', () => {
    assert.equal(completeSlash('lsx', 3), null);
    assert.equal(completeSlash('/docs rm', 8), null);
  });
});

describe('suggest', () => {
  test('prefers the newest matching history entry', () => {
    assert.equal(suggest('lsx', ['lsx -l', 'lsx -la']), ' -la');
  });

  test('falls back to a single completion', () => {
    assert.equal(suggest('/ref', [], { line: '/refresh ', cursor: 9, candidates: [] }), 'resh ');
  });

  test('offers nothing for an empty line or an ambiguous completion', () => {
    assert.equal(suggest('', ['lsx']), '');
    assert.equal(
      suggest('/h', [], { line: '/h', cursor: 2, candidates: ['/help', '/history'] }),
      ''
    );
  });
});

test('helpText lists every playground command and every tool', () => {
  const text = helpText(['lsx', 'rmx'], [{ name: 'rmx', description: 'Remove files.' }]);
  for (const c of SLASH_COMMANDS) assert.match(text, new RegExp(`/${c.name}\\b`));
  assert.match(text, /Tools \(2\)/);
  assert.match(text, /rmx\s+Remove files\n/);
  assert.match(text, /lsx/);
});
