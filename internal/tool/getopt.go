package tool

import (
	"errors"
	"fmt"
	"strings"

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

// ParseFlags parses args into c's flags the way GNU getopt_long does where
// pflag differs: a long option may be abbreviated to any prefix that names
// one option only (rm --rec is rm --recursive), an ambiguous one lists its
// possibilities, an option that takes no argument refuses one (--force=yes),
// and a mistake is worded as glibc words it ("invalid option -- 'z'",
// "option '--lines' requires an argument"). A command built with RawArgs
// parses nothing.
func ParseFlags(c *cobra.Command, args []string) error {
	if c.DisableFlagParsing {
		return nil
	}
	return ParseFlagSet(c.Flags(), args, c.Annotations[noInterspersedKey] == "true")
}

// ParseFlagSet is ParseFlags for an adapter that parses its arguments itself
// (after rewriting the ones GNU treats specially): stopAtOperand says the
// options end at the first operand, as SetInterspersed(false) makes pflag do.
func ParseFlagSet(fs *pflag.FlagSet, args []string, stopAtOperand bool) error {
	args, err := expandLong(fs, args, stopAtOperand)
	if err != nil {
		return err
	}
	return getoptError(fs.Parse(args))
}

// ownNameKey annotates a long option name gnux gives an option GNU spells
// with a letter only (lsx --long for -l).
const ownNameKey = "gnux/own-name"

// OwnNames marks long options of fs as names of gnux's own: GNU spells those
// options with a letter only. Such a name is accepted when written in full,
// but takes no part in completing an abbreviation, so that an abbreviation
// means what it means to GNU (lsx --col is --color, as in GNU ls, although
// gnux also has --columns for -C). Every name must be registered.
func OwnNames(fs *pflag.FlagSet, names ...string) {
	for _, name := range names {
		if err := fs.SetAnnotation(name, ownNameKey, []string{"true"}); err != nil {
			panic(fmt.Sprintf("OwnNames: %v", err))
		}
	}
}

func isOwn(f *pflag.Flag) bool {
	_, ok := f.Annotations[ownNameKey]
	return ok
}

// resolveLong finds the option a long name stands for, as getopt_long does
// over GNU's own names: the name itself, else the one option it is a prefix
// of. A name of gnux's own (OwnNames) answers only when written in full and
// GNU has no option it abbreviates. It returns nil for no option at all.
func resolveLong(fs *pflag.FlagSet, name string) (*pflag.Flag, error) {
	exact := fs.Lookup(name)
	if exact != nil && !isOwn(exact) {
		return exact, nil
	}
	matches := prefixMatches(fs, name)
	switch {
	case len(matches) == 1:
		return matches[0], nil
	case len(matches) > 1:
		var list strings.Builder
		for _, m := range matches {
			fmt.Fprintf(&list, " '--%s'", m.Name)
		}
		return nil, fmt.Errorf("option '--%s' is ambiguous; possibilities:%s", name, list.String())
	}
	return exact, nil
}

// expandLong rewrites every abbreviated long option in args to its full name,
// leaving the arguments of options, operands and everything after "--" (or,
// with stopAtOperand, after the first operand) as they are. Hidden options
// complete only when written in full: they are second spellings of a visible
// option, or long names gnux gives an option GNU spells with a letter only.
func expandLong(fs *pflag.FlagSet, args []string, stopAtOperand bool) ([]string, error) {
	out := make([]string, 0, len(args))
	for i := 0; i < len(args); i++ {
		a := args[i]
		switch {
		case a == "--":
			return append(out, args[i:]...), nil

		case strings.HasPrefix(a, "--"):
			name, value, hasValue := strings.Cut(a[2:], "=")
			f, err := resolveLong(fs, name)
			if err != nil {
				return nil, err
			}
			if f == nil {
				// glibc quotes the whole argument, value and all.
				return nil, fmt.Errorf("unrecognized option '%s'", a)
			}
			if hasValue && isSwitch(f) && isOwn(f) {
				// GNU has no such option, so it is no switch to GNU.
				return nil, fmt.Errorf("unrecognized option '%s'", a)
			}
			if hasValue && isSwitch(f) {
				return nil, fmt.Errorf("option '--%s' doesn't allow an argument", f.Name)
			}
			full := "--" + f.Name
			if hasValue {
				full += "=" + value
			}
			out = append(out, full)
			if !hasValue && f.NoOptDefVal == "" && i+1 < len(args) {
				i++
				out = append(out, args[i])
			}

		case strings.HasPrefix(a, "-") && len(a) > 1:
			out = append(out, a)
			// In a cluster (-rfv), the first option that takes an argument
			// takes the rest of the cluster, or the next argument.
			for j := 1; j < len(a); j++ {
				f := fs.ShorthandLookup(a[j : j+1])
				if f == nil {
					break
				}
				if f.NoOptDefVal == "" {
					if j == len(a)-1 && i+1 < len(args) {
						i++
						out = append(out, args[i])
					}
					break
				}
			}

		default:
			out = append(out, a)
			if stopAtOperand {
				return append(out, args[i+1:]...), nil
			}
		}
	}
	return out, nil
}

// prefixMatches lists the visible options of GNU's whose long name starts
// with prefix, in name order.
func prefixMatches(fs *pflag.FlagSet, prefix string) []*pflag.Flag {
	var matches []*pflag.Flag
	if prefix == "" {
		return nil
	}
	fs.VisitAll(func(f *pflag.Flag) {
		if !f.Hidden && !isOwn(f) && strings.HasPrefix(f.Name, prefix) {
			matches = append(matches, f)
		}
	})
	return matches
}

// isSwitch reports an option that takes no argument at all: pflag's
// booleans and counters, and any value that says it is a boolean the way
// pflag's own do. A custom value typed "bool" for its help screen may still
// take an optional argument (sedx --in-place[=SUFFIX]), so the type name
// alone does not decide.
func isSwitch(f *pflag.Flag) bool {
	if b, ok := f.Value.(interface{ IsBoolFlag() bool }); ok && b.IsBoolFlag() {
		return true
	}
	return f.Value.Type() == "count"
}

// getoptError rewords pflag's parse errors as glibc's getopt words them.
func getoptError(err error) error {
	var (
		notExist *pflag.NotExistError
		required *pflag.ValueRequiredError
		invalid  *pflag.InvalidValueError
	)
	switch {
	case err == nil:
		return nil
	case errors.As(err, &notExist):
		if notExist.GetSpecifiedShortnames() != "" {
			return fmt.Errorf("invalid option -- '%s'", notExist.GetSpecifiedName())
		}
		return fmt.Errorf("unrecognized option '--%s'", notExist.GetSpecifiedName())
	case errors.As(err, &required):
		if required.GetSpecifiedShortnames() != "" {
			return fmt.Errorf("option requires an argument -- '%s'", required.GetSpecifiedName())
		}
		return fmt.Errorf("option '--%s' requires an argument", required.GetSpecifiedName())
	case errors.As(err, &invalid) && isSwitch(invalid.GetFlag()):
		return fmt.Errorf("option '--%s' doesn't allow an argument", invalid.GetFlag().Name)
	}
	return err
}
