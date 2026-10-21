// Package shredx wires the shredx command (GNU shred) to its engine.
package shredx

import (
	"errors"
	"fmt"
	"math"
	"strconv"
	"strings"

	"github.com/spf13/pflag"

	"github.com/braswelljr/gnux/core"
	"github.com/braswelljr/gnux/core/shred"
	"github.com/braswelljr/gnux/internal/docmeta"
	"github.com/braswelljr/gnux/internal/tool"
)

// Spec describes the shredx tool.
func Spec() tool.Spec {
	return tool.Spec{
		Meta:  Meta(),
		GNU:   "shred",
		Flags: Register,
		Run:   run,
	}
}

// Register installs the shredx flags on fs.
func Register(fs *pflag.FlagSet) {
	fs.BoolP("force", "f", false, "change permissions to allow writing if necessary")
	fs.StringP("iterations", "n", "", "overwrite `N` times instead of the default (3)")
	fs.Var(&sources{}, "random-source", "get random bytes from `FILE`")
	fs.StringP("size", "s", "", "shred this many bytes (suffixes like K, M, G accepted)")
	fs.StringP("remove", "u", "", "deallocate and remove file after overwriting; -u is --remove=wipesync, and `HOW` may be unlink, wipe or wipesync")
	fs.Lookup("remove").NoOptDefVal = "wipesync"
	fs.BoolP("verbose", "v", false, "show progress")
	fs.BoolP("exact", "x", false, "do not round file sizes up to the next full block; this is the default for non-regular files")
	fs.BoolP("zero", "z", false, "add a final overwrite with zeros to hide shredding")
}

// sources remembers every --random-source given: GNU refuses two different
// ones.
type sources struct{ vals []string }

func (s *sources) Set(v string) error { s.vals = append(s.vals, v); return nil }
func (s *sources) String() string {
	if len(s.vals) == 0 {
		return ""
	}
	return s.vals[len(s.vals)-1]
}
func (s *sources) Type() string { return "string" }

// fatal is a mistake GNU reports without the --help hint.
type fatal struct{ msg string }

func (e *fatal) Error() string { return e.msg }

var removeMethods = []struct {
	name string
	how  shred.Remove
}{{"unlink", shred.Unlink}, {"wipe", shred.Wipe}, {"wipesync", shred.WipeSync}}

// buildOptions folds the parsed flags into shred.Options.
func buildOptions(fs *pflag.FlagSet) (shred.Options, error) {
	var o shred.Options
	o.Force, _ = fs.GetBool("force")
	o.Verbose, _ = fs.GetBool("verbose")
	o.Exact, _ = fs.GetBool("exact")
	o.Zero, _ = fs.GetBool("zero")
	if fs.Changed("iterations") {
		v, _ := fs.GetString("iterations")
		n, err := strconv.ParseUint(strings.TrimLeft(v, " \t"), 10, 31)
		if err != nil {
			return o, &fatal{fmt.Sprintf("invalid number of passes: '%s'", v)}
		}
		o.Iterations = int(n)
		if n == 0 {
			o.Iterations = -1
		}
	}
	src := fs.Lookup("random-source").Value.(*sources)
	for _, v := range src.vals {
		if v != src.vals[0] {
			return o, &fatal{"multiple random sources specified"}
		}
	}
	o.RandomSource = src.String()
	if fs.Changed("size") {
		v, _ := fs.GetString("size")
		n, ok := parseSize(v)
		if !ok {
			return o, &fatal{fmt.Sprintf("invalid file size: '%s'", v)}
		}
		o.Size, o.SizeSet = n, true
	}
	if fs.Changed("remove") {
		v, _ := fs.GetString("remove")
		var found []shred.Remove
		for _, m := range removeMethods {
			if m.name == v {
				found = []shred.Remove{m.how}
				break
			}
			if v != "" && strings.HasPrefix(m.name, v) {
				found = append(found, m.how)
			}
		}
		if len(found) != 1 {
			what := "invalid"
			if len(found) > 1 {
				what = "ambiguous"
			}
			return o, core.Usagef("%s argument '%s' for '--remove'\nValid arguments are:\n  - 'unlink'\n  - 'wipe'\n  - 'wipesync'", what, v)
		}
		o.Remove = found[0]
	}
	return o, nil
}

// parseSize reads a -s size as gnulib's xstrtoumax does with the suffixes
// shred allows: c is 1, b 512, B 1024; K, M, G, ... are powers of 1024,
// also spelled KiB, MiB, ..., and KB, MB, ... are powers of 1000.
func parseSize(s string) (int64, bool) {
	digits := len(s) - len(strings.TrimLeft(s, "0123456789"))
	if digits == 0 {
		return 0, false
	}
	n, err := strconv.ParseUint(s[:digits], 10, 63)
	if err != nil {
		return 0, false
	}
	mult := uint64(1)
	switch suffix := s[digits:]; suffix {
	case "", "c":
	case "b":
		mult = 512
	case "B":
		mult = 1024
	default:
		power := strings.IndexByte("KMGTPEZYRQ", suffix[0]) + 1
		if suffix[0] == 'k' {
			power = 1
		}
		base := uint64(0)
		switch suffix[1:] {
		case "", "iB":
			base = 1024
		case "B", "D":
			base = 1000
		}
		if power == 0 || base == 0 {
			return 0, false
		}
		for range power {
			if mult > math.MaxInt64/base {
				return 0, false
			}
			mult *= base
		}
	}
	if n != 0 && mult > math.MaxInt64/n {
		return 0, false
	}
	return int64(n * mult), true
}

func run(fs *pflag.FlagSet, args []string, s core.Streams) error {
	opts, err := buildOptions(fs)
	var f *fatal
	if errors.As(err, &f) {
		d := core.NewDiag(shred.Name, s)
		d.Errorf("%s", f.msg)
		return d.Result()
	}
	if err != nil {
		return err
	}
	return shred.New(opts, s).Run(args)
}

// Meta returns the documentation metadata for shredx.
func Meta() docmeta.Meta {
	n := shred.Name
	return docmeta.Meta{
		Name:    n,
		SeeAlso: []string{"rmx", "truncatex", "syncx", "ddx"},
		Use:     n + " [OPTION]... FILE...",
		Short:   "Overwrite a file to hide its contents, and optionally delete it.",
		Long: n + " overwrites each FILE several times (3 unless -n says otherwise), so that " +
			"its old contents are harder to recover, flushing each pass to disk. The passes " +
			"mix random data with fixed bit patterns, as GNU shred chooses them; -z adds a " +
			"last pass of zeros so that the shredding is less obvious. A regular file is " +
			"overwritten up to the end of its last block (-x keeps to its exact size), and -s " +
			"limits the overwriting to that many bytes. With -u (or --remove=wipesync) the " +
			"file is then truncated, renamed to ever shorter names of zeros to hide its name " +
			"too, and removed; --remove=wipe skips flushing the directory after each rename, " +
			"and --remove=unlink removes the file without renaming it. -f makes a read-only " +
			"file writable first. A FILE of - shreds standard output, which must be a file " +
			"opened for reading and writing, not for appending. Random bytes come from the " +
			"system's generator, or from --random-source; the order of the passes differs " +
			"from GNU's run for run, as GNU's own does. CAUTION: on journaled, " +
			"copy-on-write and log-structured file systems, on SSDs, and on anything with " +
			"backups or snapshots, the old data may survive elsewhere; shredx cannot reach it.",
		Permissions: "Needs write permission on each FILE (or ownership of it, with -f), " +
			"and write permission on its directory to remove it. Overwrites the data in place: " +
			"what it shreds cannot be recovered.",
		UseCases: []string{
			"Destroying a private key or a password file before giving a machine away.",
			"Wiping a disk image or a USB stick device before reusing it.",
			"Removing a file with sensitive contents more thoroughly than rmx does.",
		},
		Examples: []docmeta.Example{
			{Description: "Overwrite a secret file and delete it, showing each pass:", Bash: n + " -v -u secrets.txt", PowerShell: n + " -v -u secrets.txt",
				Output: "shredx: secrets.txt: pass 1/3 (random)...\nshredx: secrets.txt: pass 2/3 (random)...\nshredx: secrets.txt: pass 3/3 (random)...\nshredx: secrets.txt: removing\nshredx: secrets.txt: renamed to 00000000000\n...\nshredx: secrets.txt: removed"},
			{Description: "One random pass and then zeros, quicker for a large file:", Bash: n + " -n 1 -z backup.img", PowerShell: n + " -n 1 -z backup.img"},
			{Description: "Wipe a whole USB stick before reusing it (Linux):", Bash: "sudo " + n + " -v -n 1 /dev/sdX"},
			{Description: "Shred a read-only key file:", Bash: n + " -f -u id_ed25519", PowerShell: n + " -f -u id_ed25519"},
			{Description: "Shred every log file in a directory:", Bash: "findx logs -type f -name '*.log' -exec " + n + " -u {} +",
				PowerShell: "findx logs -type f -name '*.log' -exec " + n + " -u {} +"},
		},
	}
}
