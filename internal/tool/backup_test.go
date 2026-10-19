package tool

import (
	"errors"
	"testing"

	"github.com/spf13/pflag"

	"github.com/braswelljr/gnux/core"
)

func TestBackupOptions(t *testing.T) {
	tests := []struct {
		args []string
		env  map[string]string
		want core.Backup
		err  string
	}{
		{nil, nil, core.Backup{}, ""},
		{[]string{"-b"}, nil, core.Backup{Control: core.ExistingBackup}, ""},
		{[]string{"--backup"}, nil, core.Backup{Control: core.ExistingBackup}, ""},
		{[]string{"--backup="}, nil, core.Backup{Control: core.ExistingBackup}, ""},
		{[]string{"--backup=t"}, nil, core.Backup{Control: core.NumberedBackup}, ""},
		{[]string{"--backup=none"}, nil, core.Backup{}, ""},
		{[]string{"-b"}, map[string]string{"VERSION_CONTROL": "simple"}, core.Backup{Control: core.SimpleBackup}, ""},
		{[]string{"--backup=numbered"}, map[string]string{"VERSION_CONTROL": "bogus"}, core.Backup{Control: core.NumberedBackup}, ""},
		{nil, map[string]string{"VERSION_CONTROL": "bogus"}, core.Backup{}, ""},
		{[]string{"-S", ".bak"}, nil, core.Backup{Control: core.ExistingBackup, Suffix: ".bak"}, ""},
		{[]string{"-b"}, map[string]string{"SIMPLE_BACKUP_SUFFIX": ".orig"}, core.Backup{Control: core.ExistingBackup, Suffix: ".orig"}, ""},
		{[]string{"-b", "-S", ""}, map[string]string{"SIMPLE_BACKUP_SUFFIX": ".orig"}, core.Backup{Control: core.ExistingBackup}, ""},
		{[]string{"-b"}, map[string]string{"VERSION_CONTROL": "bogus"}, core.Backup{}, "invalid argument 'bogus' for '$VERSION_CONTROL'"},
		{[]string{"--backup=x"}, nil, core.Backup{}, "invalid argument 'x' for 'backup type'"},
	}
	for _, tt := range tests {
		fs := pflag.NewFlagSet("t", pflag.ContinueOnError)
		RegisterBackup(fs)
		if err := fs.Parse(tt.args); err != nil {
			t.Fatalf("%v: %v", tt.args, err)
		}
		got, err := BackupOptions(fs, func(k string) string { return tt.env[k] })
		var usage *core.UsageError
		switch {
		case tt.err != "":
			if !errors.As(err, &usage) || len(usage.Msg) < len(tt.err) || usage.Msg[:len(tt.err)] != tt.err {
				t.Errorf("%v %v: err = %v, want %q", tt.args, tt.env, err, tt.err)
			}
		case err != nil || got != tt.want:
			t.Errorf("%v %v: got %+v, %v; want %+v", tt.args, tt.env, got, err, tt.want)
		}
	}
}
