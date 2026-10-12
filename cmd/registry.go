package cmd

import (
	"github.com/braswelljr/gnux/cmd/basenamex"
	"github.com/braswelljr/gnux/cmd/catx"
	"github.com/braswelljr/gnux/cmd/cpx"
	"github.com/braswelljr/gnux/cmd/cutx"
	"github.com/braswelljr/gnux/cmd/dirnamex"
	"github.com/braswelljr/gnux/cmd/echox"
	"github.com/braswelljr/gnux/cmd/envx"
	"github.com/braswelljr/gnux/cmd/falsex"
	"github.com/braswelljr/gnux/cmd/headx"
	"github.com/braswelljr/gnux/cmd/lnx"
	"github.com/braswelljr/gnux/cmd/lsx"
	"github.com/braswelljr/gnux/cmd/mkdirx"
	"github.com/braswelljr/gnux/cmd/mvx"
	"github.com/braswelljr/gnux/cmd/printfx"
	"github.com/braswelljr/gnux/cmd/pwdx"
	"github.com/braswelljr/gnux/cmd/realpathx"
	"github.com/braswelljr/gnux/cmd/rmdirx"
	"github.com/braswelljr/gnux/cmd/rmx"
	"github.com/braswelljr/gnux/cmd/seqx"
	"github.com/braswelljr/gnux/cmd/sleepx"
	"github.com/braswelljr/gnux/cmd/sortx"
	"github.com/braswelljr/gnux/cmd/tailx"
	"github.com/braswelljr/gnux/cmd/teex"
	"github.com/braswelljr/gnux/cmd/touchx"
	"github.com/braswelljr/gnux/cmd/truex"
	"github.com/braswelljr/gnux/cmd/trx"
	"github.com/braswelljr/gnux/cmd/uniqx"
	"github.com/braswelljr/gnux/cmd/wcx"
	"github.com/braswelljr/gnux/cmd/yesx"
	"github.com/braswelljr/gnux/internal/tool"
)

// Tools is the registry of every tool gnux ships, in the order they are listed
// in help and documentation. Add a new tool's Spec here to make it available as
// a subcommand, as an argv[0] name, and in the generated reference.
func Tools() []tool.Spec {
	return []tool.Spec{
		basenamex.Spec(),
		catx.Spec(),
		cpx.Spec(),
		cutx.Spec(),
		dirnamex.Spec(),
		echox.Spec(),
		envx.Spec(),
		falsex.Spec(),
		headx.Spec(),
		lnx.Spec(),
		lsx.Spec(),
		mkdirx.Spec(),
		mvx.Spec(),
		printfx.Spec(),
		pwdx.Spec(),
		realpathx.Spec(),
		rmdirx.Spec(),
		rmx.Spec(),
		seqx.Spec(),
		sleepx.Spec(),
		sortx.Spec(),
		tailx.Spec(),
		teex.Spec(),
		touchx.Spec(),
		truex.Spec(),
		trx.Spec(),
		uniqx.Spec(),
		wcx.Spec(),
		yesx.Spec(),
	}
}

// Commands is everything the root command carries: the tools, then the gnux
// commands that have no GNU counterpart.
func Commands() []tool.Spec {
	return append(Tools(), ShellSpec())
}

// Lookup finds a tool by its command name ("rmx") or the name of its GNU counterpart
// ("rm").
func Lookup(name string) (tool.Spec, bool) {
	for _, s := range Tools() {
		if name == s.Name() || (s.GNU != "" && name == s.GNU) {
			return s, true
		}
	}
	return tool.Spec{}, false
}
