package prx

import (
	"errors"
	"reflect"
	"testing"

	"github.com/braswelljr/gnux/core"
	"github.com/braswelljr/gnux/core/pr"
)

func TestParseArgs(t *testing.T) {
	tests := []struct {
		args  []string
		want  pr.Options
		files []string
	}{
		{nil, pr.Options{}, nil},
		{[]string{"-3", "-t", "f"}, pr.Options{Columns: 3, OmitHeader: true}, []string{"f"}},
		{[]string{"-12", "-a"}, pr.Options{Columns: 12, Across: true}, nil},
		{[]string{"-2", "-a", "-3"}, pr.Options{Columns: 3, Across: true}, nil},
		{[]string{"--columns=4", "-T"}, pr.Options{Columns: 4, OmitPagination: true}, nil},
		{[]string{"+3:5", "f", "+2"}, pr.Options{FirstPage: 3, LastPage: 5}, []string{"f", "+2"}},
		{[]string{"--pages=2", "+3"}, pr.Options{FirstPage: 2, LastPage: 1<<64 - 1}, []string{"+3"}},
		{[]string{"-n:3", "-N", "5"}, pr.Options{NumberLines: true, NumberSeparator: ':', NumberWidth: 3, FirstLineNumber: 5, FirstLineNumberSet: true}, nil},
		{[]string{"-e", "-i.4"}, pr.Options{ExpandTabs: true, OutputTabs: true, OutputTab: '.', OutputTabWidth: 4}, nil},
		{[]string{"-h", "Title", "-D", "%F", "-l66", "-o", "2"}, pr.Options{Header: "Title", HeaderSet: true, DateFormat: "%F", Length: 66, Indent: 2}, nil},
		{[]string{"-W", "60"}, pr.Options{Width: 60, Truncate: true}, nil},
		{[]string{"-w", "60"}, pr.Options{Width: 60, JoinLines: true}, nil},
		{[]string{"-w", "60", "-2"}, pr.Options{Width: 60, Columns: 2, Truncate: true}, nil},
		{[]string{"-s,", "-m"}, pr.Options{Separator: ",", Merge: true, JoinLines: true, UseSeparator: true}, nil},
		{[]string{"-S::", "-m"}, pr.Options{Separator: "::", UseSeparator: true, Merge: true}, nil},
		{[]string{"--", "-t"}, pr.Options{}, []string{"-t"}},
		{[]string{"-", "--acr"}, pr.Options{Across: true}, []string{"-"}},
	}
	for _, tt := range tests {
		got, files, err := parseArgs(tt.args)
		if err != nil || !reflect.DeepEqual(got, tt.want) || !reflect.DeepEqual(files, tt.files) {
			t.Errorf("parseArgs(%v) = %+v %v %v\nwant %+v %v", tt.args, got, files, err, tt.want, tt.files)
		}
	}
}

func TestParseMistakes(t *testing.T) {
	tests := []struct {
		args  []string
		msg   string
		usage bool
	}{
		{[]string{"-z"}, "invalid option -- 'z'", true},
		{[]string{"-l"}, "option requires an argument -- 'l'", true},
		{[]string{"--bogus"}, "unrecognized option '--bogus'", true},
		{[]string{"--across=1"}, "option '--across' doesn't allow an argument", true},
		{[]string{"-l", "x"}, "'-l PAGE_LENGTH' invalid number of lines: 'x'", false},
		{[]string{"-l", "0"}, "'-l PAGE_LENGTH' invalid number of lines: '0': Numerical result out of range", false},
		{[]string{"--pages=x"}, "invalid page range 'x'", false},
		{[]string{"-e0"}, "'-e' extra characters or invalid number in the argument: '0'", true},
		{[]string{"-2", "-m"}, "cannot specify number of columns when printing in parallel", false},
		{[]string{"-a", "-m"}, "cannot specify both printing across and printing in parallel", false},
	}
	for _, tt := range tests {
		_, _, err := parseArgs(tt.args)
		var usage *core.UsageError
		var f *fatal
		switch {
		case tt.usage && errors.As(err, &usage) && usage.Msg == tt.msg:
		case !tt.usage && errors.As(err, &f) && f.msg == tt.msg:
		default:
			t.Errorf("%v: err = %v, want %q", tt.args, err, tt.msg)
		}
	}
}
