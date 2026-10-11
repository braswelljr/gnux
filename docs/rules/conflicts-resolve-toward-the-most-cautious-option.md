# Conflicts resolve toward the most cautious option

When options conflict and gnux cannot resolve them the way GNU does, the most cautious option wins, whatever order they were given in. The precedence is fixed in the adapter's `buildOptions`, tested in both orders, and written into the tool's reference text.

## Why

GNU lets the last of conflicting options win (`rm -f -i` prompts, `rm -i -f` does not). cobra and pflag do not report the order in which flags appeared, so gnux cannot always reproduce that. When it has to guess, a guess that prompts or refuses when the user expected silence costs a keystroke; a guess that overwrites or deletes when the user expected a question can cost data. So the guess always goes toward caution.

Where the order matters for something other than safety and can be recovered, the tool records it instead: `headx` and `tailx` stamp each count flag as it is parsed, so the last of `-n`, `-c` and `-NUM` wins as in GNU.

## What it looks like

From [`cmd/mvx/mvx.go`](../../cmd/mvx/mvx.go):

```go
// buildOptions folds the parsed flags into mv.Options. GNU mv lets the last
// of -f, -i and -n win, which pflag cannot reproduce because it does not
// expose option order; the most cautious one wins instead (-n > -i > -f).
...
switch {
case o.NoClobber:
	o.Interactive, o.Force = false, false
case o.Interactive:
	o.Force = false
}
```

The same rule, per tool:

| Tool  | Precedence                                         | Where                                                       |
|-------|----------------------------------------------------|-------------------------------------------------------------|
| `rmx` | `-i` > `--interactive=WHEN` > `-I` > `-f`          | [`cmd/rmx/interactive/interactive.go`](../../cmd/rmx/interactive/interactive.go) |
| `mvx` | `-n` > `-i` > `-f`                                 | [`cmd/mvx/mvx.go`](../../cmd/mvx/mvx.go)                     |
| `cpx` | `-n` over `-i`; `-P`/`-d` over `-H` over `-L`     | [`cmd/cpx/cpx.go`](../../cmd/cpx/cpx.go)                     |
| `lnx` | `-i` over `-f`                                     | [`cmd/lnx/lnx.go`](../../cmd/lnx/lnx.go)                     |

In `rmx`, `force.Apply` runs first and sets the baseline `PromptNever`, then `interactive.Apply` raises it. `-f` still suppresses nonexistent-file errors whatever the prompting mode ends up being.

Do: decide the precedence in `buildOptions`, test both orders, and say it in `Meta().Long`.

Do not: let whichever flag happens to be checked last in the code win.

## How it is checked

`TestBuildOptions` in each adapter tests both orders. From [`cmd/mvx/mvx_test.go`](../../cmd/mvx/mvx_test.go):

```go
// Precedence: the most cautious option wins, whatever the order.
{"-i beats -f", []string{"-i", "-f"}, mv.Options{Interactive: true}},
{"-i beats -f reversed", []string{"-f", "-i"}, mv.Options{Interactive: true}},
{"-n beats both", []string{"-fin"}, mv.Options{NoClobber: true}},
```

That every conflicting pair has a case is review.

## Related

- BRULE-3 and constraint C-4 in [business requirements](../requirements/business-requirements.md); FR-9, FR-13 in [functional requirements](../requirements/functional-requirements.md); NFR-32 in [non-functional requirements](../requirements/non-functional-requirements.md).
- [GNU is the specification, and every difference is documented](gnu-is-the-specification-and-differences-are-documented.md)
- [The zero value is the safe default](the-zero-value-is-the-safe-default.md)
