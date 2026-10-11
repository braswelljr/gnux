.. _interactive:

interactive
-----------

Prompt before removals: -i (every file), -I (once), --interactive[=WHEN].

Synopsis
~~~~~~~~


rmx offers three levels of confirmation. -i prompts before every single removal. -I prompts just once before removing more than three files or before any recursive removal - less intrusive than -i while still guarding against the worst mistakes. --interactive[=WHEN] selects the level explicitly, where WHEN is never, once, or always (bare --interactive means always). When several are combined, rmx picks the most cautious: -i > --interactive=WHEN > -I > -f.

PERMISSIONS. On a terminal, a write-protected file normally prompts 'remove write-protected ...?'. -i replaces that with an ordinary prompt for every file regardless of mode, while --interactive=never (like -f) suppresses the write-protected prompt entirely. The prompts never change what you are permitted to delete - only whether you are asked.

USE CASES.
  - Reviewing each deletion when clearing a directory of mixed important and junk files (-i).
  - A single safety confirmation before a large or recursive delete (-I).
  - Scripting a specific prompt policy independent of -f (--interactive=WHEN).
  - Overriding a -f baked into an alias, since -i is more cautious and wins.
  - Feeding prepared answers from a pipeline (yesx) in tests.

::

  interactive [FILE]...

Examples
~~~~~~~~

::

    # Ask before removing a single file
    rmx -i notes.txt

    # Ask once, then recursively remove a project directory
    rmx -I -r project/

    # Delete without any prompt, equivalent to -f for prompting
    rmx --interactive=never old.tmp

    # Ask once before removing more than three files
    rmx --interactive=once a.log b.log c.log d.log

    # -i wins over -f, so every removal is still confirmed
    rmx -fi draft.md

    # Answer yes to every prompt from a pipeline
    yesx | rmx -iv a.tmp b.tmp

Options
~~~~~~~

::

  -h, --help                            help for interactive
      --interactive string[="always"]   prompt according to WHEN: never, once, or always
  -i, --prompt-each                     prompt before every removal
  -I, --prompt-once                     prompt once before removing more than three files, or when removing recursively

