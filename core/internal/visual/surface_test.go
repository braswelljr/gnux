package visual

import "testing"

// TestSurfaces: every theme with a palette has its editor's background and
// foreground, and the variant of a family follows the background.
func TestSurfaces(t *testing.T) {
	for name := range schemes {
		if _, _, ok := Surface(name, func(string) string { return "" }); !ok {
			t.Errorf("%s has no surface", name)
		}
	}
	if len(surfaces) != len(schemes) {
		t.Errorf("%d surfaces for %d schemes", len(surfaces), len(schemes))
	}
	light := func(k string) string { return map[string]string{"GNUX_BACKGROUND": "light"}[k] }
	if bg, fg, ok := Surface("github", light); !ok || bg != "#ffffff" || fg != "#1f2328" {
		t.Errorf("github on light: %s %s %v", bg, fg, ok)
	}
	if _, _, ok := Surface("default", light); ok {
		t.Error("default has a surface")
	}
}
