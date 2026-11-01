// Package visual is gnux's terminal presentation layer: the theme of field
// colors, the size and age gradients, file icons, tree branches and bars a
// tool can opt into so that its output reads well on a terminal. Nothing
// here changes what a tool prints unless the tool asks for it: GNU's output
// stays every tool's default, and a tool turns these on with options of
// gnux's own (lsx --pretty, --icons, --tree, ...). How a tool adopts them is
// described in docs/workflows/terminal-visuals.md.
package visual

import (
	"maps"
	"strings"
	"time"
)

// The theme keys. They follow eza's EZA_COLORS names where eza has one, so
// a theme written for eza carries over; the age keys are gnux's own.
const (
	UserRead    = "ur" // r in the owner's permission triplet
	UserWrite   = "uw"
	UserExec    = "ux" // x on a regular file
	UserExecDir = "ue" // x on any other type (search permission)
	GroupRead   = "gr"
	GroupWrite  = "gw"
	GroupExec   = "gx"
	OtherRead   = "tr"
	OtherWrite  = "tw"
	OtherExec   = "tx"
	Special     = "su" // setuid, setgid and sticky letters
	Punctuation = "xx" // the "-" of an unset bit, separators, tree branches

	TypeDir     = "di" // the type letter of a mode string
	TypeLink    = "ln"
	TypePipe    = "pi"
	TypeSocket  = "so"
	TypeDevice  = "bd"
	TypeUnknown = "fi"

	Size     = "sn" // a size, when no gradient is asked for
	SizeByte = "nb" // the size gradient: under 1 KiB, KiB, MiB, GiB, TiB and up
	SizeKilo = "nk"
	SizeMega = "nm"
	SizeGiga = "ng"
	SizeTera = "nt"

	Date      = "da" // a timestamp, when no gradient is asked for
	AgeHour   = "dh" // the age gradient: under an hour, a day, a week, a month, a year, older
	AgeDay    = "dd"
	AgeWeek   = "dw"
	AgeMonth  = "dm"
	AgeYear   = "dy"
	AgeOlder  = "do"
	You       = "uu" // a user or group that is the caller's
	Others    = "un" // anyone else's
	Links     = "lc" // a link count
	Inode     = "in"
	Blocks    = "bl"
	Octal     = "oc"
	Header    = "hd" // a column header row
	Branch    = "tb" // tree branches (falls back to Punctuation)
	BarFull   = "bf" // the filled part of a bar under the warning level
	BarWarn   = "bw" // ... at or above it
	BarAlert  = "ba" // ... at or above the alert level
	BarEmpty  = "be" // the unfilled part
	GitNew    = "ga"
	GitMod    = "gm"
	GitDel    = "gd"
	GitRename = "gv"
	GitType   = "gt"
	GitIgnore = "gi"
	GitClash  = "gc"
	GitClean  = "g-"
)

// defaults is the built-in theme, close to eza's own.
var defaults = map[string]string{
	UserRead: "1;33", UserWrite: "1;31", UserExec: "1;4;32", UserExecDir: "1;32",
	GroupRead: "33", GroupWrite: "31", GroupExec: "32",
	OtherRead: "33", OtherWrite: "31", OtherExec: "32",
	Special: "35", Punctuation: "90",
	TypeDir: "1;34", TypeLink: "36", TypePipe: "33", TypeSocket: "1;35", TypeDevice: "1;33", TypeUnknown: "",
	Size: "1;32", SizeByte: "32", SizeKilo: "1;32", SizeMega: "1;33", SizeGiga: "1;31", SizeTera: "1;35",
	Date: "34", AgeHour: "1;96", AgeDay: "96", AgeWeek: "1;34", AgeMonth: "34", AgeYear: "2;34", AgeOlder: "90",
	You: "1;33", Others: "", Links: "1;31", Inode: "35", Blocks: "36", Octal: "35",
	Header: "4", Branch: "90",
	BarFull: "32", BarWarn: "33", BarAlert: "1;31", BarEmpty: "90",
	GitNew: "32", GitMod: "34", GitDel: "31", GitRename: "33", GitType: "35", GitIgnore: "90", GitClash: "1;31", GitClean: "90",
}

// Theme maps theme keys to SGR parameter strings ("1;34"). The zero Theme is
// not usable; build one with NewTheme.
type Theme struct {
	codes map[string]string
}

// NewTheme returns the built-in theme overlaid with each spec in turn. A
// spec is a colon-separated list of KEY=CODE pairs, as EZA_COLORS is
// written ("ur=33:da=2;34"); pairs whose key is not a theme key (eza's file
// globs, "reset") are ignored, and a CODE of "0" or "00" switches a key off.
func NewTheme(specs ...string) *Theme {
	t := &Theme{codes: maps.Clone(defaults)}
	for _, spec := range specs {
		for item := range strings.SplitSeq(spec, ":") {
			key, code, ok := strings.Cut(item, "=")
			if _, known := defaults[key]; !ok || !known {
				continue
			}
			if code == "0" || code == "00" {
				code = ""
			}
			t.codes[key] = code
		}
	}
	return t
}

// Code returns the SGR parameters for key, or "" when it is off. A nil
// Theme has every key off, so a caller can hold nil for "no color".
func (t *Theme) Code(key string) string {
	if t == nil {
		return ""
	}
	return t.codes[key]
}

// Paint returns s in the color of key, or s unchanged when the key is off
// or t is nil.
func (t *Theme) Paint(key, s string) string {
	return Paint(t.Code(key), s)
}

// Paint wraps s in the SGR sequence code, resetting afterwards. An empty
// code or s leaves s unchanged.
func Paint(code, s string) string {
	if code == "" || s == "" {
		return s
	}
	return "\x1b[" + code + "m" + s + "\x1b[0m"
}

// SizeKey picks the gradient key for a size in bytes.
func SizeKey(bytes int64) string {
	switch {
	case bytes < 1<<10:
		return SizeByte
	case bytes < 1<<20:
		return SizeKilo
	case bytes < 1<<30:
		return SizeMega
	case bytes < 1<<40:
		return SizeGiga
	}
	return SizeTera
}

// AgeKey picks the gradient key for a timestamp seen at now. A timestamp in
// the future counts as new.
func AgeKey(t, now time.Time) string {
	switch age := now.Sub(t); {
	case age < time.Hour:
		return AgeHour
	case age < 24*time.Hour:
		return AgeDay
	case age < 7*24*time.Hour:
		return AgeWeek
	case age < 30*24*time.Hour:
		return AgeMonth
	case age < 365*24*time.Hour:
		return AgeYear
	}
	return AgeOlder
}

// UTF8 reports whether the environment's locale can show the box-drawing
// and block characters the tree branches and bars use: true unless the
// first of LC_ALL, LC_CTYPE and LANG that is set names another character
// set (C, POSIX, ISO-8859-1, ...). An environment with none of them set
// (most Windows terminals) gets the Unicode forms.
func UTF8(getenv func(string) string) bool {
	for _, name := range []string{"LC_ALL", "LC_CTYPE", "LANG"} {
		if v := getenv(name); v != "" {
			v = strings.ToLower(v)
			return strings.Contains(v, "utf-8") || strings.Contains(v, "utf8")
		}
	}
	return true
}
