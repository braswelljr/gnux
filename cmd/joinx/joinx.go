// Package joinx wires the joinx command (GNU join) to its engine.
package joinx

import (
	"errors"
	"fmt"
	"strconv"
	"strings"

	"github.com/spf13/pflag"

	"github.com/braswelljr/gnux/core"
	"github.com/braswelljr/gnux/core/join"
	"github.com/braswelljr/gnux/internal/docmeta"
	"github.com/braswelljr/gnux/internal/tool"
)

// Spec describes the joinx tool.
func Spec() tool.Spec {
	return tool.Spec{
		Meta:  Meta(),
		GNU:   "join",
		Flags: Register,
		Run:   run,
	}
}

// Meta returns the documentation metadata for joinx.
func Meta() docmeta.Meta {
	return docmeta.Meta{
		Name:    join.Name,
		SeeAlso: []string{"commx", "pastex", "sortx", "cutx", "uniqx"},
		Use:     join.Name + " [OPTION]... FILE1 FILE2",
		Short:   "Join lines of two files on a common field.",
		Long: "joinx prints, for each pair of input lines with identical join fields, one line " +
			"made of the join field followed by the remaining fields of both lines. By default " +
			"the join field is the first, fields are separated by blanks (leading blanks are " +
			"ignored) and output fields by a single space. Either FILE may be - for standard " +
			"input, but not both. Both files must be sorted on the join field, as sortx (or " +
			"LC_ALL=C sort) sorts them; lines are compared byte by byte, as in the C locale, " +
			"whatever LANG says. -1 FIELD and -2 FIELD pick the join field of each file and -j " +
			"FIELD both. -a FILENUM also prints the unpairable lines of that file, and -v " +
			"FILENUM prints only them. -o FORMAT lists the output fields as FILENUM.FIELD or 0 " +
			"(the join field), separated by commas or blanks; -o auto takes the number of " +
			"fields from the first line of each file. -e EMPTY fills missing fields. -t CHAR " +
			"separates fields on input and output with CHAR, and -t '' makes the whole line " +
			"one field. -i ignores case when comparing join fields. --header pairs the first " +
			"lines of the files as headers without checking their order. By default joinx " +
			"warns once per file when it finds a line out of order after an unpairable line " +
			"and then exits with status 1; --check-order fails at the first line out of " +
			"order, and --nocheck-order never checks. -z makes NUL the line delimiter. When " +
			"both --check-order and --nocheck-order are given, --check-order wins. GNU's " +
			"obsolescent -j1 FIELD and -j2 FIELD forms are not accepted; use -1 and -2.",
		Permissions: "Needs read permission on FILE1 and FILE2 and search permission on the " +
			"directories leading to them. Nothing is ever modified.",
		UseCases: []string{
			"Combining two tables that share a key column, like a relational join.",
			"Attaching names to IDs, or prices to product codes, from a lookup file.",
			"Finding the keys present in one list but missing from another with -v.",
			"Building a full outer join of two CSV files with -a 1 -a 2 and -e.",
			"Merging two reports on a column other than the first with -1 and -2.",
		},
		Examples: []docmeta.Example{
			{
				Description: "Combine a names list and an ages list that share an ID column:",
				Bash:        "joinx names.txt ages.txt",
				PowerShell:  "joinx names.txt ages.txt",
			},
			{
				Description: "Attach each order to its customer from two CSV exports:",
				Bash:        "joinx -t, users.csv orders.csv",
				PowerShell:  "joinx -t, users.csv orders.csv",
			},
			{
				Description: "Look up customer names when the ID is the second column of the orders file:",
				Bash:        "joinx -1 2 -2 1 orders.txt customers.txt",
				PowerShell:  "joinx -1 2 -2 1 orders.txt customers.txt",
			},
			{
				Description: "Keep everyone in the names list, even those with no age recorded:",
				Bash:        "joinx -a 1 names.txt ages.txt",
				PowerShell:  "joinx -a 1 names.txt ages.txt",
			},
			{
				Description: "Find the tasks in the full list that are not marked done yet:",
				Bash:        "joinx -v 1 all.txt done.txt",
				PowerShell:  "joinx -v 1 all.txt done.txt",
			},
			{
				Description: "Merge two inventories into one table, marking missing values as NULL:",
				Bash:        "joinx -a 1 -a 2 -e NULL -o auto a.txt b.txt",
				PowerShell:  "joinx -a 1 -a 2 -e NULL -o auto a.txt b.txt",
			},
			{
				Description: "Build a report with just the columns you need, in your order:",
				Bash:        "joinx -o 1.2,2.3,0 a.txt b.txt",
				PowerShell:  "joinx -o 1.2,2.3,0 a.txt b.txt",
			},
			{
				Description: "Join two CSV exports that each start with a header row:",
				Bash:        "joinx --header -t, users.csv orders.csv",
				PowerShell:  "joinx --header -t, users.csv orders.csv",
			},
			{
				Description: "Join two exports that came out of a database unsorted:",
				Bash:        "sortx a.txt > a.sorted && sortx b.txt > b.sorted && joinx a.sorted b.sorted",
				PowerShell:  "sortx a.txt > a.sorted; sortx b.txt > b.sorted; joinx a.sorted b.sorted",
			},
			{
				Description: "Match user names written with different capitalization in two systems:",
				Bash:        "joinx -i a.txt b.txt",
				PowerShell:  "joinx -i a.txt b.txt",
			},
			{
				Description: "Join the output of another command against a lookup file:",
				Bash:        "sortx ids.txt | joinx - names.txt",
				PowerShell:  "sortx ids.txt | joinx - names.txt",
			},
			{
				Description: "Make a script stop if an input was not sorted, rather than miss matches:",
				Bash:        "joinx --check-order a.txt b.txt || echox 'not sorted'",
				PowerShell:  "joinx --check-order a.txt b.txt; if ($LASTEXITCODE -ne 0) { 'not sorted' }",
			},
			{
				Description: "Count matched orders in the portable gnux shell, the same on every OS:",
				Bash:        "gnux shell -c 'joinx -t, users.csv orders.csv | wcx -l'",
				PowerShell:  "gnux shell -c 'joinx -t, users.csv orders.csv | wcx -l'",
			},
		},
	}
}

// Register installs the joinx flags on fs.
func Register(fs *pflag.FlagSet) {
	fs.StringArrayP("unpaired", "a", nil, "also print unpairable lines from file `FILENUM`, where FILENUM is 1 or 2")
	fs.StringArrayP("empty", "e", nil, "replace missing input fields with `EMPTY`")
	fs.BoolP("ignore-case", "i", false, "ignore differences in case when comparing fields")
	fs.StringArrayP("join-field", "j", nil, "equivalent to '-1 FIELD -2 FIELD'")
	fs.StringArrayP("output-format", "o", nil, "obey `FORMAT` while constructing output line")
	fs.StringArrayP("separator", "t", nil, "use `CHAR` as input and output field separator")
	fs.StringArrayP("only-unpaired", "v", nil, "like -a FILENUM, but suppress joined output lines")
	fs.StringArrayP("field1", "1", nil, "join on this `FIELD` of file 1")
	fs.StringArrayP("field2", "2", nil, "join on this `FIELD` of file 2")
	fs.Bool("check-order", false, "check that the input is correctly sorted, even if all input lines are pairable")
	fs.Bool("nocheck-order", false, "do not check that the input is correctly sorted")
	fs.Bool("header", false, "treat the first line in each file as field headers, print them without trying to pair them")
	fs.BoolP("zero-terminated", "z", false, "line delimiter is NUL, not newline")
}

// fatalError is a mistake GNU join reports without the --help hint.
type fatalError struct{ msg string }

func (e *fatalError) Error() string { return e.msg }

func fatalf(format string, a ...any) error {
	return &fatalError{msg: fmt.Sprintf(format, a...)}
}

// values returns every value given for a string-array flag, empty ones
// included.
func values(fs *pflag.FlagSet, name string) []string {
	return fs.Lookup(name).Value.(pflag.SliceValue).GetSlice()
}

// buildOptions folds the parsed flags into join.Options.
func buildOptions(fs *pflag.FlagSet) (join.Options, error) {
	on := func(name string) bool {
		b, _ := fs.GetBool(name)
		return b
	}
	o := join.Options{
		IgnoreCase:     on("ignore-case"),
		Header:         on("header"),
		ZeroTerminated: on("zero-terminated"),
	}
	switch {
	case on("check-order"):
		o.Check = join.CheckOrder
	case on("nocheck-order"):
		o.Check = join.NoCheckOrder
	}

	fileNum := func(s string) (int, error) {
		switch s {
		case "1":
			return 1, nil
		case "2":
			return 2, nil
		}
		return 0, fatalf("invalid file number: '%s'", s)
	}
	for _, v := range values(fs, "unpaired") {
		n, err := fileNum(v)
		if err != nil {
			return join.Options{}, err
		}
		o.Unpaired1, o.Unpaired2 = o.Unpaired1 || n == 1, o.Unpaired2 || n == 2
	}
	for _, v := range values(fs, "only-unpaired") {
		n, err := fileNum(v)
		if err != nil {
			return join.Options{}, err
		}
		o.OnlyUnpaired = true
		o.Unpaired1, o.Unpaired2 = o.Unpaired1 || n == 1, o.Unpaired2 || n == 2
	}

	for _, v := range values(fs, "empty") {
		if o.EmptySet && o.Empty != v {
			return join.Options{}, fatalf("conflicting empty-field replacement strings")
		}
		o.Empty, o.EmptySet = v, true
	}

	// -j sets both fields; -1 and -2 one each. A field given twice with
	// different values is an error, as in GNU.
	set := func(dst *int, v string) error {
		n, err := fieldNumber(v)
		if err != nil {
			return err
		}
		if *dst != 0 && *dst != n {
			return fatalf("incompatible join fields %d, %d", *dst, n)
		}
		*dst = n
		return nil
	}
	for _, v := range values(fs, "join-field") {
		if err := set(&o.Field1, v); err != nil {
			return join.Options{}, err
		}
		if err := set(&o.Field2, v); err != nil {
			return join.Options{}, err
		}
	}
	for _, v := range values(fs, "field1") {
		if err := set(&o.Field1, v); err != nil {
			return join.Options{}, err
		}
	}
	for _, v := range values(fs, "field2") {
		if err := set(&o.Field2, v); err != nil {
			return join.Options{}, err
		}
	}

	for _, v := range values(fs, "separator") {
		c, err := separator(v)
		if err != nil {
			return join.Options{}, err
		}
		if o.SeparatorSet && o.Separator != c {
			return join.Options{}, fatalf("incompatible tabs")
		}
		o.Separator, o.SeparatorSet = c, true
	}

	for _, v := range values(fs, "output-format") {
		if v == "auto" {
			o.AutoFormat = true
			continue
		}
		fields, err := parseFormat(v)
		if err != nil {
			return join.Options{}, err
		}
		o.Format = append(o.Format, fields...)
	}
	return o, nil
}

// fieldNumber parses a 1-based field number.
func fieldNumber(s string) (int, error) {
	n, err := strconv.ParseUint(s, 10, 0)
	if err != nil {
		if errors.Is(err, strconv.ErrRange) {
			return 0, fatalf("invalid field number: '%s': Value too large for defined data type", s)
		}
		return 0, fatalf("invalid field number: '%s'", s)
	}
	if n == 0 || n > uint64(^uint(0)>>1) {
		return 0, fatalf("invalid field number: '%s'", s)
	}
	return int(n), nil
}

// separator parses -t: one byte, '\0' for NUL, or empty for "the whole line
// is one field".
func separator(s string) (byte, error) {
	switch {
	case s == "":
		return '\n', nil
	case s == `\0`:
		return 0, nil
	case len(s) > 1:
		return 0, fatalf("multi-character tab '%s'", s)
	}
	return s[0], nil
}

// parseFormat parses an -o list: entries separated by commas or blanks, each
// 0 or FILENUM.FIELD.
func parseFormat(s string) ([]join.Field, error) {
	var out []join.Field
	for _, spec := range strings.FieldsFunc(s, func(r rune) bool {
		return r == ',' || r == ' ' || r == '\t'
	}) {
		if spec == "0" {
			out = append(out, join.Field{})
			continue
		}
		file, field, ok := strings.Cut(spec, ".")
		if !ok {
			return nil, fatalf("invalid field specifier: '%s'", spec)
		}
		if file != "1" && file != "2" {
			return nil, fatalf("invalid file number in field spec: '%s'", spec)
		}
		n, err := fieldNumber(field)
		if err != nil {
			return nil, err
		}
		out = append(out, join.Field{File: int(file[0] - '0'), Field: n})
	}
	return out, nil
}

func run(fs *pflag.FlagSet, args []string, s core.Streams) error {
	opts, err := buildOptions(fs)
	var fatal *fatalError
	if errors.As(err, &fatal) {
		d := core.NewDiag(join.Name, s)
		d.Errorf("%s", fatal.msg)
		return d.Result()
	}
	return join.New(opts, s).Run(args)
}
