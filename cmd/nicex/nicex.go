// Package nicex wires the nicex command (GNU nice) to its engine.
package nicex

import (
	"errors"
	"strconv"
	"strings"

	"github.com/spf13/pflag"

	"github.com/braswelljr/gnux/core"
	"github.com/braswelljr/gnux/core/nice"
	"github.com/braswelljr/gnux/internal/docmeta"
	"github.com/braswelljr/gnux/internal/tool"
)

// Spec describes the nicex tool. GNU nice accepts the adjustment as -ADJ,
// --ADJ for a negative one, which a flag parser would reject, and stops
// reading options at the command, so the arguments are parsed here.
func Spec() tool.Spec {
	return tool.Spec{
		Meta:      Meta(),
		GNU:       "nice",
		Flags:     Register,
		Run:       run,
		RawArgs:   true,
		UsageExit: nice.ExitCanceled,
	}
}

// Meta returns the documentation metadata for nicex.
func Meta() docmeta.Meta {
	return docmeta.Meta{
		Name:    nice.Name,
		SeeAlso: []string{"timeoutx", "nohupx", "killx", "envx"},
		Use:     nice.Name + " [OPTION] [COMMAND [ARG]...]",
		Short:   "Run a program with modified scheduling priority.",
		Long: "nicex runs COMMAND with its niceness adjusted by 10, or by -n ADJUSTMENT, so " +
			"that it gets less (or, with a negative adjustment, more) CPU time than other " +
			"processes. Niceness ranges from -20 (most favorable) to 19 (least); the result " +
			"is clamped to that range. With no COMMAND it prints the current niceness. The " +
			"obsolete forms -ADJUSTMENT and --ADJUSTMENT (negative) are accepted as in GNU: " +
			"'nicex -5 make' and 'nicex --5 make'. Options must come before COMMAND. Lowering " +
			"the niceness needs privilege; without it nicex warns 'cannot set niceness' and " +
			"runs the command anyway, as GNU nice does. The exit status is the command's, or " +
			"125 when nicex itself failed, 126 when COMMAND was found but could not be run " +
			"and 127 when it was not found. nicex starts COMMAND as a child and sets its " +
			"niceness as it starts, where GNU nice changes its own and then becomes COMMAND. " +
			"On Windows, which has no niceness, the adjustment picks the nearest priority " +
			"class instead (idle, below normal, normal, above normal or high; never " +
			"realtime), so 'nicex make' really does lower the build's priority there too, " +
			"and nicex with no COMMAND prints the niceness its own class corresponds to.",
		Permissions: "Raising the niceness (lowering the priority) needs no privilege. " +
			"Lowering it below the current value needs root (or CAP_SYS_NICE); on Windows " +
			"the high priority class may need an administrator. Execute permission is needed " +
			"on COMMAND.",
		UseCases: []string{
			"Running a long build or compression job without slowing an interactive desktop.",
			"Running backups and batch jobs at low priority.",
			"Giving a latency-sensitive job more CPU, as root, with a negative adjustment.",
			"Checking the current niceness of a shell.",
			"Lowering a job's priority the same way on Linux, macOS and Windows.",
		},
		Examples: []docmeta.Example{
			{
				Description: "Run a build at lower priority:",
				Bash:        "nicex make -j8",
				PowerShell:  "nicex make -j8",
			},
			{
				Description: "Run at the lowest priority:",
				Bash:        "nicex -n 19 tar -czf backup.tgz data/",
				PowerShell:  "nicex -n 19 tar -czf backup.tgz data",
			},
			{
				Description: "Use the obsolete adjustment form:",
				Bash:        "nicex -5 make",
				PowerShell:  "nicex -5 make",
			},
			{
				Description: "Raise a job's priority (needs privilege):",
				Bash:        "sudo nicex -n -5 ./server",
				PowerShell:  "nicex -n -5 .\\server.exe",
			},
			{
				Description: "Raise it with the obsolete form:",
				Bash:        "sudo nicex --5 ./server",
				PowerShell:  "nicex --5 .\\server.exe",
			},
			{
				Description: "Print the current niceness:",
				Bash:        "nicex",
				PowerShell:  "nicex",
			},
			{
				Description: "Print the niceness a child would get:",
				Bash:        "nicex nicex",
				PowerShell:  "nicex nicex",
			},
			{
				Description: "Combine with a time limit:",
				Bash:        "nicex timeoutx 1h ./batch.sh",
				PowerShell:  "nicex timeoutx 1h .\\batch.ps1",
			},
			{
				Description: "Pass the adjustment as a long option:",
				Bash:        "nicex --adjustment=15 gzip -9 big.log",
				PowerShell:  "nicex --adjustment=15 gzip -9 big.log",
			},
			{
				Description: "Keep the command's exit status:",
				Bash:        "nicex ./tests.sh; echox \"exit: $?\"",
				PowerShell:  "nicex .\\tests.ps1; \"exit: $LASTEXITCODE\"",
			},
			{
				Description: "Run a low-priority job from the portable gnux shell:",
				Bash:        "gnux shell -c 'nicex -n 10 envx make'",
				PowerShell:  "gnux shell -c 'nicex -n 10 envx make'",
			},
		},
	}
}

// Register installs the nicex flags on fs; it is used for the help text, the
// arguments themselves being parsed by parseArgs.
func Register(fs *pflag.FlagSet) {
	fs.StringP("adjustment", "n", "", "add integer `N` to the niceness (default 10)")
}

// parsed is a command line split into its parts.
type parsed struct {
	opts    nice.Options
	command []string
	help    bool
	version bool
}

// fatalError is a mistake GNU nice reports without the --help hint.
type fatalError struct{ msg string }

func (e *fatalError) Error() string { return e.msg }

// obsoleteAdjustment reads the obsolete -ADJ form: '-' then a number with an
// optional sign, so -5, -+5 and --5 (negative).
func obsoleteAdjustment(a string) (string, bool) {
	adj := a[1:]
	digits := strings.TrimPrefix(strings.TrimPrefix(adj, "-"), "+")
	if len(digits) < len(adj)-1 || digits == "" || strings.Trim(digits, "0123456789") != "" {
		return "", false
	}
	return adj, true
}

func isObsolete(a string) bool {
	_, ok := obsoleteAdjustment(a)
	return ok
}

// parseArgs reads GNU nice's command line: options until the first operand,
// which starts the command.
func parseArgs(args []string) (parsed, error) {
	var p parsed
	adj := ""
	i := 0
loop:
	for ; i < len(args); i++ {
		a := args[i]
		switch {
		case a == "--":
			i++
			break loop
		case len(a) < 2 || a[0] != '-':
			break loop
		case isObsolete(a):
			adj, _ = obsoleteAdjustment(a)
		case a == "--help":
			p.help = true
		case a == "--version":
			p.version = true
		case strings.HasPrefix(a, "--adjustment="):
			adj = strings.TrimPrefix(a, "--adjustment=")
		case a == "--adjustment" || a == "-n":
			if i+1 >= len(args) {
				if a == "-n" {
					return p, core.Usagef("option requires an argument -- 'n'")
				}
				return p, core.Usagef("option '--adjustment' requires an argument")
			}
			i++
			adj = args[i]
		case strings.HasPrefix(a, "-n"):
			adj = a[2:]
		case strings.HasPrefix(a, "--"):
			return p, core.Usagef("unrecognized option '%s'", a)
		default:
			return p, core.Usagef("invalid option -- '%c'", a[1])
		}
	}
	p.command = args[i:]
	if adj != "" || (i > 0 && hasEmptyAdjustment(args[:i])) {
		n, err := parseAdjustment(adj)
		if err != nil {
			return p, err
		}
		p.opts = nice.Options{Adjustment: n, Adjust: true}
	}
	return p, nil
}

// hasEmptyAdjustment reports an explicitly empty --adjustment=.
func hasEmptyAdjustment(opts []string) bool {
	for _, a := range opts {
		if a == "--adjustment=" {
			return true
		}
	}
	return false
}

// parseAdjustment reads a signed integer; one too large for an int is
// clamped, since the engine clamps the niceness anyway.
func parseAdjustment(s string) (int, error) {
	n, err := strconv.ParseInt(s, 10, 32)
	if err != nil {
		var num *strconv.NumError
		if errors.As(err, &num) && errors.Is(num.Err, strconv.ErrRange) {
			if strings.HasPrefix(s, "-") {
				return nice.MinNiceness * 2, nil
			}
			return nice.MaxNiceness*2 + 1, nil
		}
		return 0, &fatalError{"invalid adjustment '" + s + "'"}
	}
	return int(n), nil
}

func run(fs *pflag.FlagSet, args []string, s core.Streams) error {
	p, err := parseArgs(args)
	var fatal *fatalError
	switch {
	case errors.As(err, &fatal):
		core.NewDiag(nice.Name, s).Errorf("%s", fatal.msg)
		return &core.ExitError{Code: nice.ExitCanceled}
	case err != nil:
		return err
	case p.help:
		Spec().WriteHelp(s.Out, fs, tool.StyleFor(s.Out))
		return nil
	case p.version:
		tool.PrintVersion(s.Out, nice.Name)
		return nil
	}
	return nice.New(p.opts, s).Run(p.command)
}
