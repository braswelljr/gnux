# GNU is the specification, and every difference is documented

Each gnux tool mirrors its GNU counterpart's flags, semantics, diagnostics and exit statuses. GNU's behavior is not a source of inspiration but the specification: when gnux behaves differently, the difference is deliberate, explained in the tool's own reference text, and never left for a user to discover.

## Why

gnux is GNU-compatible: it brings the GNU tools' interface to every platform. A script written for `rm`, `cp` or `head` must keep working under `rmx`, `cpx` or `headx`, and a person who already knows GNU should not need a second manual. A silent difference breaks both promises. Some differences are forced on gnux (the command-line framework, the platform), so the rule is not "never differ" but "never differ silently".

## What it looks like

When the framework cannot reproduce GNU, the tool says so in `Meta().Long`, which becomes the reference page for every format. From [`cmd/mvx/mvx.go`](../../cmd/mvx/mvx.go):

```go
"directory. GNU mv lets the last of -f, -i and -n win; mvx cannot see option order, " +
"so it prefers the most cautious instead: -n over -i over -f.",
```

When a platform cannot do what GNU does, the `Permissions` note says what happens instead. From [`cmd/touchx/touchx.go`](../../cmd/touchx/touchx.go):

```go
"Changing the times of a symbolic link itself (-h) is only possible on Linux; " +
"elsewhere touchx reports the link rather than silently touching its target.",
```

Where GNU's behavior can be reproduced with more work, the work is done rather than documented away. GNU `head` lets the last of `-n`, `-c` and `-NUM` win, so [`cmd/headx/headx.go`](../../cmd/headx/headx.go) stamps each count flag as it is parsed:

```go
// order stamps each count-setting flag as it is parsed. GNU head lets the last
// of -n, -c and -NUM win, and pflag does not otherwise record flag order.
type order struct{ n int }
```

Do: match GNU, or write the difference into the tool's `Long` or `Permissions` text.

Do not: pick a different behavior because it seemed nicer, and leave the reference silent about it.

## How it is checked

- `TestRegistryIsConsistent` in [`cmd/main_test.go`](../../cmd/main_test.go) requires every tool to have a non-empty `Short`, `Long` and at least one example, so there is always somewhere to write the difference.
- The engine suites (`core/<gnu>/<gnu>_test.go`) assert on GNU's exact output and diagnostics.
- Whether a given difference is documented is review.

## Related

- BRULE-2 in [business requirements](../requirements/business-requirements.md); FR-2, FR-10 in [functional requirements](../requirements/functional-requirements.md).
- [Conflicts resolve toward the most cautious option](conflicts-resolve-toward-the-most-cautious-option.md)
- [Diagnostics are written in GNU's words](diagnostics-are-written-in-gnus-words.md)
- [The reference documents itself, and it is checked](the-reference-documents-itself-and-it-is-checked.md)
