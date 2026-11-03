package visual

import "golang.org/x/sys/windows/registry"

// systemDark reads Windows's app mode: AppsUseLightTheme is 0 in dark mode.
func systemDark(func(string) string) (dark, ok bool) {
	k, err := registry.OpenKey(registry.CURRENT_USER,
		`Software\Microsoft\Windows\CurrentVersion\Themes\Personalize`, registry.QUERY_VALUE)
	if err != nil {
		return false, false
	}
	defer k.Close()
	light, _, err := k.GetIntegerValue("AppsUseLightTheme")
	if err != nil {
		return false, false
	}
	return light == 0, true
}
