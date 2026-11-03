package visual

import "strings"

// EnvEntry paints one NAME=value entry of the environment: the name in the
// EnvName color and the '=' in the punctuation color. In a variable whose
// name ends in PATH (PATH, MANPATH, GOPATH), the list separators listSep
// are dimmed too, so each directory stands apart. An entry without '=' is
// left as it is.
func (t *Theme) EnvEntry(kv string, listSep byte) string {
	name, value, ok := strings.Cut(kv, "=")
	if !ok || t == nil {
		return kv
	}
	if strings.HasSuffix(name, "PATH") && strings.IndexByte(value, listSep) >= 0 {
		sep := t.Paint(Punctuation, string(listSep))
		value = strings.ReplaceAll(value, string(listSep), sep)
	}
	return t.Paint(EnvName, name) + t.Paint(Punctuation, "=") + value
}
