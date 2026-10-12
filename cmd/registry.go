package cmd

import (
	"github.com/braswelljr/gnux/cmd/b2sumx"
	"github.com/braswelljr/gnux/cmd/base32x"
	"github.com/braswelljr/gnux/cmd/base64x"
	"github.com/braswelljr/gnux/cmd/basenamex"
	"github.com/braswelljr/gnux/cmd/catx"
	"github.com/braswelljr/gnux/cmd/chgrpx"
	"github.com/braswelljr/gnux/cmd/chmodx"
	"github.com/braswelljr/gnux/cmd/chownx"
	"github.com/braswelljr/gnux/cmd/cksumx"
	"github.com/braswelljr/gnux/cmd/commx"
	"github.com/braswelljr/gnux/cmd/cpx"
	"github.com/braswelljr/gnux/cmd/csplitx"
	"github.com/braswelljr/gnux/cmd/cutx"
	"github.com/braswelljr/gnux/cmd/datex"
	"github.com/braswelljr/gnux/cmd/ddx"
	"github.com/braswelljr/gnux/cmd/dfx"
	"github.com/braswelljr/gnux/cmd/dirnamex"
	"github.com/braswelljr/gnux/cmd/dux"
	"github.com/braswelljr/gnux/cmd/echox"
	"github.com/braswelljr/gnux/cmd/envx"
	"github.com/braswelljr/gnux/cmd/expandx"
	"github.com/braswelljr/gnux/cmd/exprx"
	"github.com/braswelljr/gnux/cmd/falsex"
	"github.com/braswelljr/gnux/cmd/fmtx"
	"github.com/braswelljr/gnux/cmd/foldx"
	"github.com/braswelljr/gnux/cmd/headx"
	"github.com/braswelljr/gnux/cmd/hostnamex"
	"github.com/braswelljr/gnux/cmd/idx"
	"github.com/braswelljr/gnux/cmd/installx"
	"github.com/braswelljr/gnux/cmd/joinx"
	"github.com/braswelljr/gnux/cmd/killx"
	"github.com/braswelljr/gnux/cmd/linkx"
	"github.com/braswelljr/gnux/cmd/lnx"
	"github.com/braswelljr/gnux/cmd/lsx"
	"github.com/braswelljr/gnux/cmd/md5sumx"
	"github.com/braswelljr/gnux/cmd/mkdirx"
	"github.com/braswelljr/gnux/cmd/mktempx"
	"github.com/braswelljr/gnux/cmd/mvx"
	"github.com/braswelljr/gnux/cmd/nicex"
	"github.com/braswelljr/gnux/cmd/nlx"
	"github.com/braswelljr/gnux/cmd/nohupx"
	"github.com/braswelljr/gnux/cmd/nprocx"
	"github.com/braswelljr/gnux/cmd/odx"
	"github.com/braswelljr/gnux/cmd/pastex"
	"github.com/braswelljr/gnux/cmd/printenvx"
	"github.com/braswelljr/gnux/cmd/printfx"
	"github.com/braswelljr/gnux/cmd/pwdx"
	"github.com/braswelljr/gnux/cmd/readlinkx"
	"github.com/braswelljr/gnux/cmd/realpathx"
	"github.com/braswelljr/gnux/cmd/rmdirx"
	"github.com/braswelljr/gnux/cmd/rmx"
	"github.com/braswelljr/gnux/cmd/seqx"
	"github.com/braswelljr/gnux/cmd/sha1sumx"
	"github.com/braswelljr/gnux/cmd/sha224sumx"
	"github.com/braswelljr/gnux/cmd/sha256sumx"
	"github.com/braswelljr/gnux/cmd/sha384sumx"
	"github.com/braswelljr/gnux/cmd/sha512sumx"
	"github.com/braswelljr/gnux/cmd/shufx"
	"github.com/braswelljr/gnux/cmd/sleepx"
	"github.com/braswelljr/gnux/cmd/sortx"
	"github.com/braswelljr/gnux/cmd/splitx"
	"github.com/braswelljr/gnux/cmd/statx"
	"github.com/braswelljr/gnux/cmd/tacx"
	"github.com/braswelljr/gnux/cmd/tailx"
	"github.com/braswelljr/gnux/cmd/teex"
	"github.com/braswelljr/gnux/cmd/testx"
	"github.com/braswelljr/gnux/cmd/timeoutx"
	"github.com/braswelljr/gnux/cmd/touchx"
	"github.com/braswelljr/gnux/cmd/truex"
	"github.com/braswelljr/gnux/cmd/truncatex"
	"github.com/braswelljr/gnux/cmd/trx"
	"github.com/braswelljr/gnux/cmd/ttyx"
	"github.com/braswelljr/gnux/cmd/unamex"
	"github.com/braswelljr/gnux/cmd/unexpandx"
	"github.com/braswelljr/gnux/cmd/uniqx"
	"github.com/braswelljr/gnux/cmd/unlinkx"
	"github.com/braswelljr/gnux/cmd/uptimex"
	"github.com/braswelljr/gnux/cmd/wcx"
	"github.com/braswelljr/gnux/cmd/whoamix"
	"github.com/braswelljr/gnux/cmd/whox"
	"github.com/braswelljr/gnux/cmd/yesx"
	"github.com/braswelljr/gnux/internal/tool"
)

// Tools is the registry of every tool gnux ships, in the order they are listed
// in help and documentation. Add a new tool's Spec here to make it available as
// a subcommand, as an argv[0] name, and in the generated reference.
func Tools() []tool.Spec {
	return []tool.Spec{
		b2sumx.Spec(),
		base32x.Spec(),
		base64x.Spec(),
		basenamex.Spec(),
		catx.Spec(),
		chgrpx.Spec(),
		chmodx.Spec(),
		chownx.Spec(),
		cksumx.Spec(),
		commx.Spec(),
		cpx.Spec(),
		csplitx.Spec(),
		cutx.Spec(),
		datex.Spec(),
		ddx.Spec(),
		dfx.Spec(),
		dirnamex.Spec(),
		dux.Spec(),
		echox.Spec(),
		envx.Spec(),
		expandx.Spec(),
		exprx.Spec(),
		falsex.Spec(),
		fmtx.Spec(),
		foldx.Spec(),
		headx.Spec(),
		hostnamex.Spec(),
		idx.Spec(),
		installx.Spec(),
		joinx.Spec(),
		killx.Spec(),
		linkx.Spec(),
		lnx.Spec(),
		lsx.Spec(),
		md5sumx.Spec(),
		mkdirx.Spec(),
		mktempx.Spec(),
		mvx.Spec(),
		nicex.Spec(),
		nlx.Spec(),
		nohupx.Spec(),
		nprocx.Spec(),
		odx.Spec(),
		pastex.Spec(),
		printenvx.Spec(),
		printfx.Spec(),
		pwdx.Spec(),
		readlinkx.Spec(),
		realpathx.Spec(),
		rmdirx.Spec(),
		rmx.Spec(),
		seqx.Spec(),
		sha1sumx.Spec(),
		sha224sumx.Spec(),
		sha256sumx.Spec(),
		sha384sumx.Spec(),
		sha512sumx.Spec(),
		shufx.Spec(),
		sleepx.Spec(),
		sortx.Spec(),
		splitx.Spec(),
		statx.Spec(),
		tacx.Spec(),
		tailx.Spec(),
		teex.Spec(),
		testx.Spec(),
		timeoutx.Spec(),
		touchx.Spec(),
		truex.Spec(),
		truncatex.Spec(),
		trx.Spec(),
		ttyx.Spec(),
		unamex.Spec(),
		unexpandx.Spec(),
		uniqx.Spec(),
		unlinkx.Spec(),
		uptimex.Spec(),
		wcx.Spec(),
		whoamix.Spec(),
		whox.Spec(),
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
