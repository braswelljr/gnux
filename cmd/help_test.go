package cmd

import (
	"strings"
	"testing"

	"github.com/spf13/pflag"

	"github.com/braswelljr/gnux/core"
)

func TestHelpOverviewListsEveryCommand(t *testing.T) {
	for _, argv := range [][]string{{"gnux", "--help"}, {"gnux", "help"}, {"gnux"}} {
		status, out, errb := run(t, argv...)
		if status != 0 || errb != "" {
			t.Fatalf("%v: status=%d stderr=%q", argv, status, errb)
		}
		for _, s := range Commands() {
			if !strings.Contains(out, s.Name()) || !strings.Contains(out, s.Meta.Short) {
				t.Errorf("%v: overview is missing %s", argv, s.Name())
			}
		}
		if !strings.Contains(out, "gnux help TOOL") {
			t.Errorf("%v: overview does not say how to read a manual", argv)
		}
	}
}

func TestToolHelpPointsToTheManual(t *testing.T) {
	for _, s := range Tools() {
		_, out, _ := run(t, s.Name(), "--help")
		for _, want := range []string{"Usage: " + s.Name(), "Exit status:", "gnux help " + s.Name(), "Online: "} {
			if !strings.Contains(out, want) {
				t.Errorf("%s --help: missing %q", s.Name(), want)
			}
		}
	}
}

// Every command, by every name, has a full manual with every section a reader
// relies on.
func TestHelpManualForEveryCommand(t *testing.T) {
	for _, s := range Commands() {
		names := []string{s.Name()}
		if s.GNU != "" {
			names = append(names, s.GNU)
		}
		for _, name := range names {
			status, out, errb := run(t, "gnux", "help", name)
			if status != 0 || errb != "" {
				t.Errorf("gnux help %s: status=%d stderr=%q", name, status, errb)
				continue
			}
			for _, section := range []string{"NAME", "SYNOPSIS", "DESCRIPTION", "OPTIONS", "EXAMPLES", "EXIT STATUS", "SEE ALSO"} {
				if !strings.Contains(out, "\n"+section+"\n") && !strings.HasPrefix(out, section+"\n") {
					t.Errorf("gnux help %s: no %s section", name, section)
				}
			}
			for _, e := range s.Meta.Examples {
				if e.Bash != "" && !strings.Contains(out, strings.Split(e.Bash, "\n")[0]) {
					t.Errorf("gnux help %s: example %q missing", name, e.Bash)
				}
			}
			if strings.Contains(out, "\x1b[") {
				t.Errorf("gnux help %s: color written to a non-terminal", name)
			}
		}
	}
}

// Every option of every tool can be looked up by long name and by shorthand.
func TestHelpForEveryOption(t *testing.T) {
	for _, s := range Commands() {
		s.Command(core.Discard()).Flags().VisitAll(func(f *pflag.Flag) {
			forms := []string{"--" + f.Name, f.Name}
			if f.Shorthand != "" {
				forms = append(forms, "-"+f.Shorthand)
			}
			for _, form := range forms {
				status, out, errb := run(t, "gnux", "help", s.Name(), form)
				if status != 0 || errb != "" || !strings.Contains(out, "--"+f.Name) {
					t.Errorf("gnux help %s %s: status=%d stderr=%q", s.Name(), form, status, errb)
				}
			}
		})
	}
}

func TestHelpTopicsAreShownInFull(t *testing.T) {
	_, out, _ := run(t, "gnux", "help", "rm", "-f")
	for _, want := range []string{"rmx force", "PERMISSIONS", "USE CASES", "EXAMPLES"} {
		if !strings.Contains(out, want) {
			t.Errorf("gnux help rm -f: missing %q", want)
		}
	}
}

func TestHelpErrors(t *testing.T) {
	tests := []struct {
		args []string
		want string
	}{
		{[]string{"nosuchtool"}, "gnux help: no help for 'nosuchtool'"},
		{[]string{"rmx", "--bogus"}, "gnux help: rmx has no option '--bogus'"},
		{[]string{"rmx", "-f", "extra"}, "gnux help: too many arguments"},
	}
	for _, tt := range tests {
		status, out, errb := run(t, append([]string{"gnux", "help"}, tt.args...)...)
		if status != 1 || out != "" || !strings.HasPrefix(errb, tt.want) ||
			!strings.Contains(errb, "Try 'gnux help --help' for more information.") {
			t.Errorf("gnux help %v: status=%d stdout=%q stderr=%q", tt.args, status, out, errb)
		}
	}
}

func TestHelpHelp(t *testing.T) {
	for _, args := range [][]string{{"help", "help"}, {"help", "--help"}} {
		status, out, _ := run(t, append([]string{"gnux"}, args...)...)
		if status != 0 || !strings.HasPrefix(out, "Usage: gnux help [TOOL [OPTION]]") {
			t.Errorf("gnux %v: status=%d stdout=%q", args, status, out)
		}
	}
}
